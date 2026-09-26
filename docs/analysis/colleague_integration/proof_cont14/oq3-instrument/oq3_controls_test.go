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
	"reflect"
	"sort"
	"strings"
	"testing"
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
	t.Logf("failing child -> invalid: %v", res.Invalid)
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
