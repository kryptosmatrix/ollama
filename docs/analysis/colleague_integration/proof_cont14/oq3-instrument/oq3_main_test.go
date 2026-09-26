//go:build darwin

// TestOQ3FailureNumber runs the frozen ledger and publishes the failure number with its components
// and a separate validity verdict. Environment:
//   OQ3_EVIDENCE_DIR  where the ledger, per-scenario results, captures and summary are written (required)
//   OQ3_GOLDENS       the frozen pre-G1 goldens directory (required)
//   OQ3_FREEZE=1      write a golden when absent (only for the pre-G1 baseline run that creates them)
//   OQ3_MODE          baseline (valid run required; the number is recorded) or accept (the number must be 0)
//   OQ3_SCENARIOS     comma-separated scenario ids, or "all"; empty is refused
//   OQ3_SOURCE        a description of the candidate under test, recorded in goldens and the summary

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func oq3Select(all []oq3Scenario, sel string) ([]oq3Scenario, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return nil, errors.New("empty scenario selection refused")
	}
	if sel == "all" {
		return all, nil
	}
	byID := map[string]oq3Scenario{}
	for _, s := range all {
		byID[s.ID] = s
	}
	var out []oq3Scenario
	for _, id := range strings.Split(sel, ",") {
		s, ok := byID[strings.TrimSpace(id)]
		if !ok {
			return nil, fmt.Errorf("unknown scenario %q", id)
		}
		out = append(out, s)
	}
	return out, nil
}

// oq3Completeness checks that every frozen trial of every selected scenario produced delivery
// records (observed or missing) and that every selected scenario produced a result.
func oq3Completeness(selected []oq3Scenario, results []oq3ScenarioResult) []string {
	var problems []string
	got := map[string]oq3ScenarioResult{}
	for _, r := range results {
		got[r.Scenario.ID] = r
	}
	for _, sc := range selected {
		r, ok := got[sc.ID]
		if !ok {
			problems = append(problems, "scenario "+sc.ID+" produced no result")
			continue
		}
		seen := map[string]int{}
		for _, d := range r.Deliveries {
			seen[d.Trial]++
		}
		for _, tr := range sc.Trials {
			if seen[tr.ID] == 0 {
				problems = append(problems, "trial "+tr.ID+" has no delivery record")
			}
		}
	}
	return problems
}

func TestOQ3FailureNumber(t *testing.T) {
	evidence := os.Getenv("OQ3_EVIDENCE_DIR")
	goldens := os.Getenv("OQ3_GOLDENS")
	mode := os.Getenv("OQ3_MODE")
	if evidence == "" || goldens == "" || (mode != "baseline" && mode != "accept") {
		t.Fatalf("OQ3_EVIDENCE_DIR, OQ3_GOLDENS and OQ3_MODE (baseline|accept) are required")
	}
	freeze := os.Getenv("OQ3_FREEZE") == "1"
	if freeze && mode != "baseline" {
		t.Fatalf("goldens may only be frozen by a baseline run")
	}
	source := os.Getenv("OQ3_SOURCE")
	all := oq3AllScenarios()
	selected, err := oq3Select(all, os.Getenv("OQ3_SCENARIOS"))
	if err != nil {
		t.Fatal(err)
	}
	ledgerSHA, ledgerJSON := oq3LedgerSHA(all)
	if err := os.MkdirAll(evidence, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "ledger.json"), ledgerJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("OQ3 ledger sha256 %s (%d scenarios), selected %d", ledgerSHA, len(all), len(selected))

	var results []oq3ScenarioResult
	var refResults []oq3ScenarioResult
	for _, sc := range selected {
		start := time.Now()
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, sc.ID)
		run := oq3NewRun(t, sc, d, s)
		if strings.HasPrefix(sc.ID, "E") {
			if bin := os.Getenv("OQ3_BINARY"); bin != "" {
				run.builtBinary = []string{bin}
			}
		}
		run.execute()
		res := run.score(goldens, freeze, source)
		results = append(results, res)
		if err := oq3WriteJSON(filepath.Join(evidence, "scenarios", sc.ID+".json"), res); err != nil {
			t.Fatal(err)
		}
		// Reference arm (Q7 b): the expected requests, sent through real HTTP to a fresh fake daemon
		// under the same labels, must all score correct. Kept outside the candidate's numbers.
		ref := oq3ReferenceArm(t, run, res, goldens)
		refResults = append(refResults, ref)
		if err := oq3WriteJSON(filepath.Join(evidence, "reference", sc.ID+".json"), ref); err != nil {
			t.Fatal(err)
		}
		t.Logf("OQ3 scenario %s done in %s: %d deliveries recorded, %d invalid", sc.ID, time.Since(start).Round(time.Millisecond), len(res.Deliveries), len(res.Invalid))
	}

	sum := oq3Summarise(results, mode, source, ledgerSHA)
	if problems := oq3Completeness(selected, results); len(problems) > 0 {
		sum.Valid = false
		sum.Invalid = append(sum.Invalid, problems...)
	}
	refSum := oq3Summarise(refResults, "reference", "expected requests sent by the reference arm", ledgerSHA)
	if refSum.FailureNumerator != 0 || refSum.ObservedDeliveries == 0 {
		sum.Valid = false
		sum.Invalid = append(sum.Invalid, fmt.Sprintf("reference arm did not read correct deliveries as correct: %d bad or missing of %d", refSum.FailureNumerator, refSum.ObservedDeliveries))
	}
	if err := oq3WriteJSON(filepath.Join(evidence, "summary.json"), sum); err != nil {
		t.Fatal(err)
	}
	if err := oq3WriteJSON(filepath.Join(evidence, "reference-summary.json"), refSum); err != nil {
		t.Fatal(err)
	}
	t.Logf("OQ3_FAILURE_NUMBER=%s valid=%v observed=%d bad=%d missing=%d trials=%d", sum.FailureNumber, sum.Valid, sum.ObservedDeliveries, sum.BadObserved, sum.MissingRequired, sum.Trials)
	t.Logf("OQ3_REFERENCE_ARM=%s observed=%d", refSum.FailureNumber, refSum.ObservedDeliveries)
	if !sum.Valid {
		t.Fatalf("OQ3 run is not valid: %s", strings.Join(sum.Invalid, " | "))
	}
	if mode == "accept" {
		if sum.FailureNumerator != 0 {
			t.Fatalf("OQ3 acceptance: failure number %s", sum.FailureNumber)
		}
		// Operations must pass on their own evidence; "done" is accepted only for the existing
		// terminal commands (/system, /tools, /new, /compact, /model), whose effect is checked by the
		// deliveries that follow them. "unobserved" and "absent" fail.
		for k, n := range sum.OperationVerdicts {
			if !strings.HasSuffix(k, ":pass") && k != "terminal-slash:done" {
				t.Fatalf("OQ3 acceptance: operation verdict %s x%d", k, n)
			}
		}
		for k, n := range sum.RefusalVerdicts {
			if !strings.HasSuffix(k, ":pass") {
				t.Fatalf("OQ3 acceptance: required refusal %s x%d", k, n)
			}
		}
	}
}

// oq3ReferenceArm sends, for every delivery the candidate was scored on, the request a correct G1
// implementation would send (the oracle's expectation), through real HTTP to a fresh fake daemon,
// and scores the captures with the same scorer. A correct delivery must read as correct.
func oq3ReferenceArm(t *testing.T, run *oq3Run, res oq3ScenarioResult, goldens string) oq3ScenarioResult {
	rd := oq3StartDaemon(t)
	for _, dl := range res.Deliveries {
		if dl.Expect == nil || dl.Golden == "" {
			continue
		}
		golden, err := oq3LoadGolden(dl.Golden)
		if err != nil {
			continue
		}
		exp, err := oq3Transform(golden, *dl.Expect, func(model string) (string, bool) {
			s, ok := rd.modelSystems[oq3ModelBase(model)]
			return s, ok
		})
		if err != nil {
			continue
		}
		body, err := oq3MarshalCanonical(exp)
		if err != nil {
			t.Fatal(err)
		}
		rd.setLabel(dl.Label)
		resp, err := http.Post(rd.srv.URL+"/api/chat", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("reference arm post: %v", err)
		}
		resp.Body.Close()
	}
	refRun := oq3NewRun(t, run.sc, rd, run.s)
	// The reference arm reuses the scenario's steps for attribution; its captures are the expected
	// requests. Summariser requests are not re-sent, so compaction-demonstration checks do not apply.
	scored := refRun.scoreDeliveriesOnly(goldens)
	return scored
}
