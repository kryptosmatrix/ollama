//go:build darwin

// Controls for the OQ-3 instrument itself (Codex options check, Q7): the scorer and the runner must
// detect what they exist to detect. The expected requests here are written out by hand from the
// blueprint's words (§5 and §6.3 A1), not produced by oq3Transform, so the oracle is checked against
// an independent statement of the rule.

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func oq3MustNorm(t *testing.T, s string) any {
	t.Helper()
	v, err := oq3Normalise([]byte(s), "")
	if err != nil {
		t.Fatalf("fixture does not parse: %v\n%s", err, s)
	}
	return v
}

func oq3JSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestOQ3ScorerControls(t *testing.T) {
	A, B := oq3Texts["A"], oq3Texts["B"]
	ms1 := oq3ModelSystem
	// Header text written out literally from blueprint §5, independently of oq3Header.
	hA11 := "User-configured instructions; revision 1; selection 1:\n"
	hB23 := "User-configured instructions; revision 2; selection 3:\n"

	desktopGolden := `{"model":"oq3-model","messages":[{"role":"user","content":"Hello."}],"stream":true,"options":null}`
	desktopExpected := `{"model":"oq3-model","messages":[{"role":"system","content":` + oq3JSONString(ms1+"\n\n"+hA11+A) + `},{"role":"user","content":"Hello."}],"stream":true,"options":null,"admission":{"version":1}}`
	terminalGolden := `{"model":"oq3-model","messages":[{"role":"system","content":"BASE PROMPT"},{"role":"user","content":"Hi."}],"options":{}}`
	terminalExpected := `{"model":"oq3-model","messages":[{"role":"system","content":` + oq3JSONString("BASE PROMPT\n\n"+hB23+B) + `},{"role":"user","content":"Hi."}],"options":{},"admission":{"version":1}}`
	terminalOffGolden := `{"model":"oq3-model","messages":[{"role":"user","content":"Hi."}],"options":{}}`
	terminalOffExpected := `{"model":"oq3-model","messages":[{"role":"system","content":` + oq3JSONString(hA11+A) + `},{"role":"user","content":"Hi."}],"options":{},"admission":{"version":1}}`

	ms := func(model string) (string, bool) {
		if model == "oq3-model" {
			return ms1, true
		}
		return "", false
	}
	eD := cD(1, 1, "A")
	eT := cT(2, 3, "B")
	eTOff := cT(1, 1, "A")

	// 1. The transform matches the hand-written expectation in all three shapes.
	for _, c := range []struct {
		name, golden, expected string
		e                      oq3Expect
	}{{"desktop", desktopGolden, desktopExpected, eD}, {"terminal", terminalGolden, terminalExpected, eT}, {"terminal-system-off", terminalOffGolden, terminalOffExpected, eTOff}} {
		got, err := oq3Transform(oq3MustNorm(t, c.golden), c.e, ms)
		if err != nil {
			t.Fatalf("%s transform: %v", c.name, err)
		}
		if !reflect.DeepEqual(got, oq3MustNorm(t, c.expected)) {
			gb, _ := json.Marshal(got)
			t.Fatalf("%s transform differs from the hand-written expectation:\n got %s\nwant %s", c.name, gb, c.expected)
		}
		// 2. A perfect delivery scores correct.
		if v := oq3Score(oq3MustNorm(t, c.expected), got, c.e); !v.Correct {
			t.Fatalf("%s: a perfect delivery scored %+v", c.name, v)
		}
	}
	// A desktop golden that already leads with a system message (MCP) must not be guessed.
	if _, err := oq3Transform(oq3MustNorm(t, `{"model":"oq3-model","messages":[{"role":"system","content":"MCP"},{"role":"user","content":"x"}]}`), eD, ms); err == nil {
		t.Fatal("the oracle guessed the unspecified model-system/MCP join")
	}

	exp := oq3MustNorm(t, desktopExpected)
	sys := func(content string) string { return `{"role":"system","content":` + oq3JSONString(content) + `}` }
	user := `{"role":"user","content":"Hello."}`
	adm := `,"admission":{"version":1}`
	tail := `,"stream":true,"options":null`
	cases := []struct {
		name     string
		observed string
		e        oq3Expect
		expected any
		want     []string // every one must be present
	}{
		{"missing carrier", desktopGolden, eD, exp, []string{"missing", "admission-missing"}},
		{"wrong revision", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\nUser-configured instructions; revision 2; selection 1:\n"+A) + `,` + user + `]` + tail + adm + `}`, eD, exp, []string{"wrong-revision"}},
		{"wrong generation", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\nUser-configured instructions; revision 1; selection 2:\n"+A) + `,` + user + `]` + tail + adm + `}`, eD, exp, []string{"wrong-revision"}},
		{"duplicated", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+A) + `,` + sys(hA11+A) + `,` + user + `]` + tail + adm + `}`, eD, exp, []string{"duplicated"}},
		{"altered text", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+strings.Replace(A, "SENTINEL-A-MID-9e41 ", "", 1)) + `,` + user + `]` + tail + adm + `}`, eD, exp, []string{"altered"}},
		{"base lost", `{"model":"oq3-model","messages":[` + sys(hA11+A) + `,` + user + `]` + tail + adm + `}`, eD, exp, []string{"base-lost"}},
		{"misplaced in the user message", `{"model":"oq3-model","messages":[{"role":"user","content":` + oq3JSONString("Hello.\n\n"+hA11+A) + `}]` + tail + adm + `}`, eD, exp, []string{"misplaced"}},
		{"admission missing only", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+A) + `,` + user + `]` + tail + `}`, eD, exp, []string{"admission-missing"}},
		{"history changed", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+A) + `,{"role":"user","content":"Hello!"}]` + tail + adm + `}`, eD, exp, []string{"request-changed"}},
		{"unexpected carrier", `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+A) + `,` + user + `]` + tail + adm + `}`, nD(), oq3MustNorm(t, desktopGolden), []string{"unexpected-carrier", "unexpected-admission", "leaked"}},
		{"multi-defect", `{"model":"oq3-model","messages":[` + sys("User-configured instructions; revision 3; selection 1:\n"+A) + `,` + user + `]` + tail + `}`, eD, exp, []string{"wrong-revision", "base-lost", "admission-missing"}},
		{"extra field on the carrier message", `{"model":"oq3-model","messages":[{"role":"system","content":` + oq3JSONString(ms1+"\n\n"+hA11+A) + `,"images":[]},` + user + `]` + tail + adm + `}`, eD, exp, []string{"other"}},
	}
	for _, c := range cases {
		v := oq3Score(oq3MustNorm(t, c.observed), c.expected, c.e)
		if v.Correct {
			t.Fatalf("%s: a faulty delivery scored correct", c.name)
		}
		for _, w := range c.want {
			found := false
			for _, got := range v.Categories {
				if got == w {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s: category %q not reported; got %v (%s)", c.name, w, v.Categories, v.Detail)
			}
		}
		t.Logf("control %-36s -> %v", c.name, v.Categories)
	}

	// 2b. Admission objects (review finding 6): a reserve within A2's bounds is correct; a wrong
	// version, an out-of-range reserve or an unknown member is its own category.
	withAdm := func(adm string) string {
		return `{"model":"oq3-model","messages":[` + sys(ms1+"\n\n"+hA11+A) + `,` + user + `]` + tail + `,"admission":` + adm + `}`
	}
	if v := oq3Score(oq3MustNorm(t, withAdm(`{"version":1,"reserve":2048}`)), exp, eD); !v.Correct {
		t.Fatalf("a permitted reserve scored %+v", v)
	}
	for _, bad := range []string{`{"version":2}`, `{"version":1,"reserve":20000}`, `{"version":1,"reserve":-1}`, `{"version":1,"extra":true}`} {
		v := oq3Score(oq3MustNorm(t, withAdm(bad)), exp, eD)
		if v.Correct || !strings.Contains(strings.Join(v.Categories, ","), "admission-invalid") {
			t.Fatalf("admission %s scored %+v", bad, v)
		}
	}
	// 2c. Normalisation removes both spellings of the sandbox path (review finding 7).
	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	n, err := oq3Normalise([]byte(`{"a":`+oq3JSONString(resolved+"/work")+`,"b":`+oq3JSONString(root+"/work")+`}`), root)
	if err != nil {
		t.Fatal(err)
	}
	if nm, _ := n.(map[string]any); nm["a"] != "<SANDBOX>/work" || nm["b"] != "<SANDBOX>/work" {
		t.Fatalf("normalisation left a machine-dependent path: %v (root %s, resolved %s)", nm, root, resolved)
	}
	if resolved == root {
		t.Log("note: the temporary root has no symlinked spelling on this machine; the resolved-path arm is not exercised")
	}

	// 3. Summariser expectations.
	clean := `{"model":"oq3-model","messages":[{"role":"system","content":"Summarize the archived part of an Ollama agent conversation. Keep goals."},{"role":"user","content":"archive"}]}`
	if v := oq3CheckSummariser(oq3MustNorm(t, clean)); !v.Correct {
		t.Fatalf("clean summariser scored %+v", v)
	}
	for name, s := range map[string]string{
		"summariser-carries-header":  `{"messages":[{"role":"system","content":"Summarize the archived part of an Ollama agent conversation."},{"role":"user","content":` + oq3JSONString("archive "+hA11) + `}]}`,
		"summariser-carries-profile": `{"messages":[{"role":"system","content":"Summarize the archived part of an Ollama agent conversation."},{"role":"user","content":"archive SENTINEL-B-END-83d9"}]}`,
		"summariser-prompt-replaced": `{"messages":[{"role":"system","content":"something else"},{"role":"user","content":"archive"}]}`,
	} {
		v := oq3CheckSummariser(oq3MustNorm(t, s))
		if v.Correct || !strings.Contains(strings.Join(v.Categories, ","), name) {
			t.Fatalf("summariser control %s: %+v", name, v)
		}
	}

	// 4. Arithmetic: counts reconcile, a multi-defect request counts once, and the ratio is exact.
	sc := oq3Scenario{ID: "SYN", Trials: []oq3Trial{{"S-a", famSave, "desktop", ""}, {"S-b", famEdit, "desktop", ""}, {"S-c", famDisable, "terminal", ""}}}
	res := oq3ScenarioResult{Scenario: sc, Deliveries: []oq3Delivery{
		{Trial: "S-a", Status: "correct", Verdict: oq3DeliveryVerdict{Correct: true}},
		{Trial: "S-a", Status: "bad", Verdict: oq3DeliveryVerdict{Categories: []string{"missing", "admission-missing"}}},
		{Trial: "S-b", Status: "missing", Verdict: oq3DeliveryVerdict{Categories: []string{"not-dispatched"}}},
		{Trial: "S-c", Status: "unexpected", Verdict: oq3DeliveryVerdict{Categories: []string{"unexpected-request"}}},
		{Trial: "S-c", Status: "bad", Verdict: oq3DeliveryVerdict{Categories: []string{"wrong-revision", "base-lost", "admission-missing"}}},
	}}
	s := oq3Summarise([]oq3ScenarioResult{res}, "control", "synthetic", "none")
	if s.Trials != 3 || s.ObservedDeliveries != 4 || s.BadObserved != 3 || s.MissingRequired != 1 || s.Unexpected != 1 || s.FailureNumerator != 4 || s.FailureNumber != "4/3 = 1.3333" {
		t.Fatalf("arithmetic control: %+v", s)
	}
	if s.CategoryCounts["admission-missing"] != 2 || s.CategoryCounts["not-dispatched"] != 1 {
		t.Fatalf("category control: %v", s.CategoryCounts)
	}
	if oq3Ratio(0, 0) == "0/0 = 0.0000" {
		t.Fatal("a zero denominator printed as a number")
	}

	// 5. T8 (review round 2, finding 1). An adapter that wrongly commits the reload refused during the
	// held turn, while the held turn's continuation keeps the selection it copied, must fail the ledger.
	// Its selections are modelled here from that description, step by step, not from the ledger's
	// expectations. Under the round-2 ledger, which had no turn between the await and the idle reload,
	// it scored zero; a correct adapter scores zero under both.
	var t8 oq3Scenario
	for _, sc := range oq3AllScenarios() {
		if sc.ID == "T8" {
			t8 = sc
		}
	}
	var round2 []oq3Step
	inserted := 0
	for _, st := range t8.Steps {
		if st.Op == "terminal-turn" && strings.Contains(st.Prompt, "at idle before any reload") {
			inserted++
			continue
		}
		round2 = append(round2, st)
	}
	if t8.ID == "" || inserted != 1 || len(round2) != 10 {
		t.Fatalf("T8: scenario %q, %d inserted turns (want 1), %d round-2 steps (want 10)", t8.ID, inserted, len(round2))
	}
	simulate := func(steps []oq3Step, commitsWhileBusy bool) (bad, total int) {
		type sel struct {
			rev, gen int
			text     string
		}
		var saved, cur, copied sel
		held := false
		golden := oq3MustNorm(t, terminalGolden)
		for _, st := range steps {
			switch {
			case st.Op == "cli-set":
				saved = sel{rev: saved.rev + 1, text: st.Args["text"]}
			case st.Op == "terminal-start":
				cur = sel{rev: saved.rev, gen: 1, text: saved.text}
			case st.Op == "terminal-turn":
				if st.Args["hold"] == "1" {
					held, copied = true, cur
				}
				use := cur
				if held {
					use = copied
				}
				for _, e := range st.Expect {
					obs, err := oq3Transform(golden, cT(use.rev, use.gen, use.text), ms)
					if err != nil {
						t.Fatal(err)
					}
					exp, err := oq3Transform(golden, e, ms)
					if err != nil {
						t.Fatal(err)
					}
					if !oq3Score(obs, exp, e).Correct {
						bad++
					}
					total++
				}
			case st.Op == "terminal-await":
				held = false
			case st.Op == "terminal-slash" && st.Args["cmd"] == "/instructions reload":
				busy := st.Class == "required-refusal"
				if (!busy || commitsWhileBusy) && cur.rev != saved.rev {
					cur = sel{rev: saved.rev, gen: cur.gen + 1, text: saved.text}
				}
			}
		}
		return bad, total
	}
	wrongNow, nNow := simulate(t8.Steps, true)
	wrongThen, nThen := simulate(round2, true)
	rightNow, _ := simulate(t8.Steps, false)
	rightThen, _ := simulate(round2, false)
	if wrongNow == 0 || wrongThen != 0 || rightNow != 0 || rightThen != 0 {
		t.Fatalf("T8 control: busy-committing adapter %d/%d bad now, %d/%d under the round-2 ledger (want >0 and 0); correct adapter %d now, %d then (want 0, 0)",
			wrongNow, nNow, wrongThen, nThen, rightNow, rightThen)
	}
	t.Logf("control T8: busy-committing adapter %d of %d deliveries bad (round-2 ledger: %d of %d); correct adapter 0", wrongNow, nNow, wrongThen, nThen)
}

func TestOQ3RunnerControls(t *testing.T) {
	all := oq3AllScenarios()
	// Empty and unknown selections are refused.
	if _, err := oq3Select(all, ""); err == nil {
		t.Fatal("empty selection accepted")
	}
	if _, err := oq3Select(all, "D1,NOPE"); err == nil {
		t.Fatal("unknown scenario accepted")
	}
	// Frozen trial identities are unique across the ledger.
	seen := map[string]bool{}
	ids := []string{}
	for _, sc := range all {
		for _, tr := range sc.Trials {
			if seen[tr.ID] {
				t.Fatalf("duplicate trial %s", tr.ID)
			}
			seen[tr.ID] = true
			ids = append(ids, tr.ID)
		}
		for i, st := range sc.Steps {
			if st.Class == "required-dispatch" && (st.Trial == "" || !seen[st.Trial] && !oq3TrialIn(sc, st.Trial)) {
				t.Fatalf("%s step %d dispatches for an undeclared trial %q", sc.ID, i, st.Trial)
			}
		}
	}
	sort.Strings(ids)
	t.Logf("ledger: %d scenarios, %d trials", len(all), len(ids))

	// A missing trial is caught by the completeness check.
	sel, _ := oq3Select(all, "D1")
	if p := oq3Completeness(sel, []oq3ScenarioResult{{Scenario: sel[0]}}); len(p) == 0 {
		t.Fatal("a trial with no delivery record passed completeness")
	}
	if p := oq3Completeness(sel, nil); len(p) == 0 {
		t.Fatal("a scenario with no result passed completeness")
	}

	goldens := t.TempDir()
	// Auxiliary faults (review findings 2-4; review round 2, findings 3 and 4): each must be reported
	// as an auxiliary failure, which gates acceptance; an unknown inference endpoint must make the run
	// invalid; a clean metadata request must report nothing. The escaped faults carry no marker in their
	// raw bytes, which the round-2 scan read; only decoding finds them.
	send := func(d *oq3Daemon, method, path string, header map[string]string, body string) int {
		req, err := http.NewRequest(method, d.srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	escaped := map[string]string{
		"leak-escaped-sentinel":    `{"model":"oq3-model","system":"\u0053ENTINEL-A-MID-9e41"}`,
		"leak-escaped-header-text": `{"model":"oq3-model","system":"\u0055ser-configured instructions; revision 1; selection 1:\n"}`,
		"leak-nested-json":         `{"model":"oq3-model","meta":"{\"note\":\"\\u0053ENTINEL-B-MID-4c18\"}"}`,
	}
	for fault, body := range escaped {
		if len(oq3Leaks(body)) != 0 {
			t.Fatalf("control %s is not escaped: its raw bytes already carry a marker", fault)
		}
	}
	for _, fault := range []string{"undeclared-summariser", "leak-in-metadata", "unknown-endpoint",
		"leak-escaped-sentinel", "leak-escaped-header-text", "leak-nested-json", "leak-in-key", "leak-in-header", "leak-in-query",
		"metadata-wrong-method", "clean-metadata"} {
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, "AUX-"+fault)
		sc := oq3Scenario{ID: "AUX", Trials: []oq3Trial{{"AUX-X", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dTurn("AUX-X", "X", "OQ3 auxiliary control turn.", nD()), h("desktop-stop")}}
		run := oq3NewRun(t, sc, d, s)
		run.execute()
		d.setLabel(oq3Label("AUX", 1, "desktop-turn"))
		switch fault {
		case "undeclared-summariser":
			oq3PostRaw(t, d, []byte(`{"model":"oq3-model","messages":[{"role":"system","content":"`+oq3CompactionSystemPrefix+` extra"},{"role":"user","content":"archive"}]}`))
		case "leak-in-metadata":
			send(d, http.MethodPost, "/api/show", nil, `{"model":"oq3-model","system":"SENTINEL-A-MID-9e41"}`)
		case "unknown-endpoint":
			send(d, http.MethodPost, "/api/embed", nil, `{"model":"oq3-model","input":"x"}`)
		case "leak-escaped-sentinel", "leak-escaped-header-text", "leak-nested-json":
			send(d, http.MethodPost, "/api/show", nil, escaped[fault])
		case "leak-in-key":
			send(d, http.MethodPost, "/api/show", nil, `{"model":"oq3-model","SENTINEL-B-START-2a6e":true}`)
		case "leak-in-header":
			send(d, http.MethodGet, "/api/tags", map[string]string{"X-OQ3-Note": "SENTINEL-A-END-b7f0"}, "")
		case "leak-in-query":
			send(d, http.MethodGet, "/api/tags?note=User-configured%20instructions", nil, "")
		case "metadata-wrong-method":
			if st := send(d, http.MethodGet, "/api/show", nil, ""); st != http.StatusMethodNotAllowed {
				t.Fatalf("GET /api/show answered %d, want 405 as the real daemon answers", st)
			}
		case "clean-metadata":
			send(d, http.MethodPost, "/api/show", nil, `{"model":"oq3-model"}`)
		}
		res := run.score(goldens, true, "auxiliary control")
		sum := oq3Summarise([]oq3ScenarioResult{res}, "control", "auxiliary control", "none")
		switch fault {
		case "unknown-endpoint":
			if len(res.Invalid) == 0 {
				t.Fatalf("an unknown inference endpoint did not make the run invalid")
			}
			t.Logf("auxiliary fault %-24s -> invalid: %v", fault, res.Invalid)
		case "clean-metadata":
			if sum.AuxiliaryBad != 0 || len(res.Invalid) != 0 {
				t.Fatalf("a clean metadata request was reported: %v %v", sum.AuxiliaryBadList, res.Invalid)
			}
			t.Logf("auxiliary control %-22s -> nothing reported", fault)
		default:
			if sum.AuxiliaryBad == 0 {
				t.Fatalf("auxiliary fault %s was not reported", fault)
			}
			t.Logf("auxiliary fault %-24s -> %v", fault, sum.AuxiliaryBadList)
		}
	}

	// A delivery sent with the wrong method (review round 2, finding 3): the request a desktop turn
	// sends, replayed as PUT under the same step, is refused by the fake as the real daemon refuses it
	// and scores as an incorrect delivery; the same body replayed as POST scores correct.
	{
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, "RC-method")
		sc := oq3Scenario{ID: "RM", Trials: []oq3Trial{{"RM-X", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dTurn("RM-X", "X", "OQ3 method control turn.", nD()), h("desktop-stop")}}
		run := oq3NewRun(t, sc, d, s)
		run.execute()
		res := run.score(goldens, true, "runner control")
		if len(res.Invalid) != 0 || len(res.Deliveries) != 1 || res.Deliveries[0].Status != "correct" {
			t.Fatalf("method control baseline: invalid %v, deliveries %+v", res.Invalid, res.Deliveries)
		}
		var body string
		for _, c := range res.Captures {
			if c.Path == "/api/chat" && c.Method == http.MethodPost {
				body = string(c.Body)
			}
		}
		for _, method := range []string{http.MethodPut, http.MethodPost} {
			d2 := oq3StartDaemon(t)
			d2.setLabel(oq3Label("RM", 1, "desktop-turn"))
			status := send(d2, method, "/api/chat", nil, body)
			run2 := oq3NewRun(t, sc, d2, s)
			run2.events = run.events
			res2 := run2.score(goldens, false, "runner control")
			if len(res2.Deliveries) != 1 {
				t.Fatalf("%s replay: %d deliveries", method, len(res2.Deliveries))
			}
			got := res2.Deliveries[0]
			switch method {
			case http.MethodPut:
				if status != http.StatusMethodNotAllowed || got.Status != "bad" || !strings.Contains(strings.Join(got.Verdict.Categories, ","), "wrong-method") {
					t.Fatalf("PUT replay: status %d, delivery %+v", status, got)
				}
			case http.MethodPost:
				if status != http.StatusOK || got.Status != "correct" {
					t.Fatalf("POST replay: status %d, delivery %+v", status, got)
				}
			}
			t.Logf("method control %-4s -> HTTP %d, delivery %s %v", method, status, got.Status, got.Verdict.Categories)
		}
	}

	// The idle proof (review round 2, finding 5), on the real terminal. The probe must read an idle chat
	// as idle, and must not read as idle an open approval prompt after the daemon has fully answered
	// every request it was sent (where the round-2 rule, daemon completion and a quiet screen, holds),
	// nor a turn the daemon is holding. A command the chat keeps as input while a turn runs must not
	// reach the next turn's text.
	{
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, "RC-idle")
		d.setLabel("RCI/00/terminal-start")
		term := oq3StartTerminal(t, s, d, "RCI")
		defer term.kill()
		if !oq3WaitFor(30*time.Second, func() bool { return d.count("POST", "/api/generate") >= 1 }) {
			t.Fatalf("idle control: the session never preloaded; screen %q", tail(term.screen(), 1500))
		}
		if !term.idle(30 * time.Second) {
			t.Fatalf("idle control: the probe did not read an idle chat as idle; screen %q", tail(term.screen(), 1500))
		}
		label := "RCI/01/terminal-turn"
		d.setLabel(label)
		d.setToolRounds(label, 1)
		mark := term.rawLen()
		term.typeLine("OQ3 idle control " + oq3ToolTrigger)
		if !oq3WaitFor(30*time.Second, func() bool { return term.screenContainsAfter(mark, "Approve once") }) {
			t.Fatalf("idle control: no approval prompt; screen %q", tail(term.screen(), 1500))
		}
		if !oq3WaitFor(10*time.Second, func() bool {
			n := d.countLabel("POST", "/api/chat", label)
			return n == 1 && d.completedLabel(label) >= n
		}) || !term.quiet(600*time.Millisecond, 10*time.Second) {
			t.Fatalf("idle control: the round-2 completion rule did not hold at the approval prompt")
		}
		if term.idle(4 * time.Second) {
			t.Fatal("idle control: the probe read an open approval prompt as idle")
		}
		term.key("1")
		if !oq3WaitFor(30*time.Second, func() bool {
			n := d.countLabel("POST", "/api/chat", label)
			return n == 2 && d.completedLabel(label) >= n
		}) || !term.idle(30*time.Second) {
			t.Fatalf("idle control: the approved turn never became idle; screen %q", tail(term.screen(), 1500))
		}
		label = "RCI/02/terminal-turn"
		d.setLabel(label)
		d.hold(label)
		term.typeLine("OQ3 idle control, held turn.")
		if !oq3WaitFor(30*time.Second, func() bool { return d.countLabel("POST", "/api/chat", label) >= 1 }) {
			t.Fatalf("idle control: the held turn never reached the daemon")
		}
		if term.idle(4 * time.Second) {
			t.Fatal("idle control: the probe read a held turn as idle")
		}
		// Typed while the turn runs: the unchanged candidate does not know the command and keeps it as
		// input (cmd/tui/chat/input.go:100).
		term.typeLine("/instructions reload")
		d.release(label)
		if !oq3WaitFor(30*time.Second, func() bool { return d.completedLabel(label) >= 1 }) || !term.idle(30*time.Second) {
			t.Fatalf("idle control: the held turn never became idle; screen %q", tail(term.screen(), 1500))
		}
		label = "RCI/03/terminal-turn"
		d.setLabel(label)
		const text = "OQ3 idle control, the turn after a command typed while busy."
		term.typeLine(text)
		if !oq3WaitFor(30*time.Second, func() bool { return d.countLabel("POST", "/api/chat", label) >= 1 }) {
			t.Fatalf("idle control: the turn after the busy command was not sent; screen %q", tail(term.screen(), 1500))
		}
		var last string
		for _, c := range d.snapshot() {
			if c.Label == label && c.Path == "/api/chat" {
				if v, err := oq3Normalise(c.Body, ""); err == nil {
					if msgs := oq3Messages(v); len(msgs) > 0 {
						last = oq3Str(msgs[len(msgs)-1], "content")
					}
				}
			}
		}
		if last != text {
			t.Fatalf("idle control: the turn after the busy command sent %q, want %q", last, text)
		}
		if !oq3WaitFor(30*time.Second, func() bool { return d.completedLabel(label) >= 1 }) || !term.idle(30*time.Second) {
			t.Fatalf("idle control: the last turn never became idle")
		}
		term.typeLine("/bye")
		if err, ok := term.waitExit(20 * time.Second); !ok || err != nil {
			t.Fatalf("idle control: the session did not exit on /bye: ok=%v err=%v", ok, err)
		}
		t.Log("idle control: idle at rest; not idle at an approval prompt after the daemon finished, nor during a held turn; a busy command kept as input did not reach the next turn")
	}

	// The runner uses that proof (review round 2, finding 5): a turn whose daemon has finished while the
	// chat still waits (an approval nobody gives) must make the run invalid, where the round-2 rule
	// (every response written, then a quiet screen) would have advanced as if the turn had ended.
	{
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, "RC-busy")
		sc := oq3Scenario{ID: "RCB", Trials: []oq3Trial{{"RCB-X", famSave, "terminal", "runner idle control"}},
			Steps: []oq3Step{h("terminal-start"), tTurn("RCB-X", "OQ3 runner idle control "+oq3ToolTrigger, nT()), h("terminal-stop")}}
		d.setToolRounds(oq3Label("RCB", 1, "terminal-turn"), 1) // a tool call the step does not declare, so nothing approves it
		run := oq3NewRun(t, sc, d, s)
		run.execute()
		res := run.score(goldens, true, "runner control")
		turnVerdict := ""
		for _, e := range res.Events {
			if e.Step == 1 {
				turnVerdict = e.Verdict
			}
		}
		// The turn itself must fail; a runner that trusted the quiet screen records it as done and
		// only fails later, when /bye lands on the approval prompt.
		if turnVerdict != "harness-failed" || !strings.Contains(strings.Join(res.Invalid, " | "), "never proved idle") {
			t.Fatalf("a turn left waiting after the daemon finished was treated as ended: turn verdict %q, invalid %v", turnVerdict, res.Invalid)
		}
		t.Logf("runner idle control -> turn %s; invalid: %v", turnVerdict, res.Invalid)
	}

	// Refusal at start-up against a daemon without chat.admission.v1 (review round 2, finding 6): a
	// session that reports context_unavailable naming the daemon's version and exits is a held refusal,
	// and its no-dispatch turn is scored as refused as required; a session that exits without that
	// report is a harness failure, never a refusal.
	for _, mode := range []string{"terminal-refuse-at-start", "terminal-exit-silently"} {
		d := oq3StartDaemon(t)
		d.noCapability = true
		s := oq3NewSandbox(t, "RC-"+mode)
		sc := oq3Scenario{ID: "RCR", NoCapability: true, Trials: []oq3Trial{{"RCR-X", famSave, "terminal", "refusal control"}},
			Steps: []oq3Step{h("terminal-start"),
				{Op: "terminal-turn", Class: "required-no-dispatch", Trial: "RCR-X", Prompt: "OQ3 refusal control turn.", Args: map[string]string{}},
				h("terminal-stop")}}
		run := oq3NewRun(t, sc, d, s)
		run.terminalHelper = mode
		run.execute()
		res := run.score(goldens, true, "runner control")
		switch mode {
		case "terminal-refuse-at-start":
			refused := false
			for _, e := range res.Events {
				refused = refused || e.Verdict == "refused-at-start"
			}
			if len(res.Invalid) != 0 || !refused || len(res.Deliveries) != 1 || res.Deliveries[0].Status != "refused-as-required" {
				t.Fatalf("a permitted refusal at start-up was not held: invalid %v, events %+v, deliveries %+v", res.Invalid, res.Events, res.Deliveries)
			}
		case "terminal-exit-silently":
			if len(res.Invalid) == 0 {
				t.Fatalf("a session that died at start-up without a report passed as a refusal: %+v", res.Deliveries)
			}
		}
		t.Logf("refusal control %-26s -> invalid %v; deliveries %d", mode, res.Invalid, len(res.Deliveries))
	}

	// Harness faults on a real desktop scenario: each must make the run invalid, not change the number.
	for _, fault := range []string{"malformed-capture", "extra-request", "unknown-label", "none"} {
		d := oq3StartDaemon(t)
		s := oq3NewSandbox(t, "RC-"+fault)
		sc := oq3Scenario{ID: "RC", Trials: []oq3Trial{{"RC-X", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dTurn("RC-X", "X", "OQ3 runner control turn.", nD()), h("desktop-stop")}}
		run := oq3NewRun(t, sc, d, s)
		run.execute()
		label := oq3Label("RC", 1, "desktop-turn")
		switch fault {
		case "malformed-capture":
			d.setLabel(label)
			oq3PostRaw(t, d, []byte(`{"model":"oq3-model","messages":[`))
		case "extra-request":
			d.setLabel(label)
			oq3PostRaw(t, d, []byte(`{"model":"oq3-model","messages":[{"role":"user","content":"extra"}]}`))
		case "unknown-label":
			d.setLabel("RC/99/nowhere")
			oq3PostRaw(t, d, []byte(`{"model":"oq3-model","messages":[{"role":"user","content":"stray"}]}`))
		}
		res := run.score(goldens, true, "runner control")
		if fault == "none" {
			if len(res.Invalid) != 0 {
				t.Fatalf("clean control run invalid: %v", res.Invalid)
			}
			continue
		}
		if len(res.Invalid) == 0 {
			t.Fatalf("harness fault %s did not make the run invalid", fault)
		}
		t.Logf("harness fault %-18s -> invalid: %v", fault, res.Invalid)
	}

	// A failing child: a terminal session for a model the daemon does not have exits at once; the
	// step is a harness failure and the run is invalid, never a missing delivery.
	d := oq3StartDaemon(t)
	s := oq3NewSandbox(t, "RC-child")
	sc := oq3Scenario{ID: "RCC", Trials: []oq3Trial{{"RCC-X", famSave, "terminal", ""}},
		Steps: []oq3Step{with(h("terminal-start"), "model", "no-such-model"), tTurn("RCC-X", "OQ3 turn that must not be scored.", nT()), h("terminal-stop")}}
	run := oq3NewRun(t, sc, d, s)
	run.execute()
	res := run.score(goldens, true, "runner control")
	if len(res.Invalid) == 0 {
		t.Fatal("a failing child process did not make the run invalid")
	}
	// Review finding 12: steps not run after a harness failure are not measured, never missing.
	if sum := oq3Summarise([]oq3ScenarioResult{res}, "control", "failing child", "none"); sum.FailureNumerator != 0 || len(sum.NotMeasured) == 0 {
		t.Fatalf("a failing child changed the number: numerator %d, not measured %v", sum.FailureNumerator, sum.NotMeasured)
	}
	t.Logf("failing child -> invalid: %v; not measured: %v", res.Invalid, res.NotMeasured)
}

func oq3TrialIn(sc oq3Scenario, id string) bool {
	for _, tr := range sc.Trials {
		if tr.ID == id {
			return true
		}
	}
	return false
}

func oq3PostRaw(t *testing.T, d *oq3Daemon, body []byte) {
	t.Helper()
	resp, err := http.Post(d.srv.URL+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
