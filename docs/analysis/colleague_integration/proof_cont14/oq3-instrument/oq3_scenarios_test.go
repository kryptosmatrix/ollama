//go:build darwin

// OQ-3 scenario ledger. Every step is declared before execution with its class:
//   required-dispatch  a model request must reach the daemon, with the stated expectation per request;
//   required-refusal   a production operation must refuse (for example a reload during a held turn);
//   operation          a production operation with a stated expected outcome (save, reload, CLI);
//   harness            process control and fixtures; a failure makes the run invalid, never a defect.
// Trial identities are frozen here; a trial that fails to start is still counted.

package cmd

import (
	"fmt"
)

type oq3Step struct {
	Op     string            `json:"op"`
	Class  string            `json:"class"`
	Trial  string            `json:"trial,omitempty"`
	Conv   string            `json:"conv,omitempty"`
	Prompt string            `json:"prompt,omitempty"`
	Expect []oq3Expect       `json:"expect,omitempty"`
	Want   map[string]string `json:"want,omitempty"`
	Args   map[string]string `json:"args,omitempty"`
	// Summarisers is the number of compaction summariser requests this step must cause.
	Summarisers int `json:"summarisers,omitempty"`
}

type oq3Trial struct {
	ID      string `json:"id"`
	Family  string `json:"family"`
	Adapter string `json:"adapter"`
	Note    string `json:"note,omitempty"`
}

type oq3Scenario struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Trials []oq3Trial `json:"trials"`
	Steps  []oq3Step  `json:"steps"`
}

// The six families named by the failure number (blueprint line 18).
const (
	famSave    = "save/restart/new"
	famEdit    = "edit/continue"
	famReload  = "explicit reload"
	famDisable = "disable"
	famReset   = "terminal reset"
	famCompact = "compaction"
)

func cD(rev, gen int, text string) oq3Expect {
	return oq3Expect{Carrier: true, Rev: rev, Gen: gen, Text: text, Adapter: "desktop"}
}
func nD() oq3Expect { return oq3Expect{Adapter: "desktop"} }
func cT(rev, gen int, text string) oq3Expect {
	return oq3Expect{Carrier: true, Rev: rev, Gen: gen, Text: text, Adapter: "terminal"}
}
func nT() oq3Expect { return oq3Expect{Adapter: "terminal"} }

func h(op string, args ...string) oq3Step {
	s := oq3Step{Op: op, Class: "harness", Args: map[string]string{}}
	for i := 0; i+1 < len(args); i += 2 {
		s.Args[args[i]] = args[i+1]
	}
	return s
}

// dSave saves through the designated desktop editor API (PUT /api/v1/instructions, §7).
func dSave(expected int, enabled bool, text string, wantRev int) oq3Step {
	return oq3Step{Op: "desktop-save", Class: "operation",
		Args: map[string]string{"expected_revision": fmt.Sprint(expected), "enabled": fmt.Sprint(enabled), "text": text},
		Want: map[string]string{"revision": fmt.Sprint(wantRev), "enabled": fmt.Sprint(enabled), "text": text}}
}

// cliSet saves through the designated terminal CLI (`ollama instructions set`, §7).
func cliSet(expected int, text string) oq3Step {
	return oq3Step{Op: "cli-set", Class: "operation", Args: map[string]string{"expected_revision": fmt.Sprint(expected), "text": text},
		Want: map[string]string{"exit": "0"}}
}

func dTurn(trial, conv, prompt string, ex ...oq3Expect) oq3Step {
	return oq3Step{Op: "desktop-turn", Class: "required-dispatch", Trial: trial, Conv: conv, Prompt: prompt, Expect: ex, Args: map[string]string{}}
}

func dReload(conv string, gen, target int, wantGen int) oq3Step {
	return oq3Step{Op: "desktop-reload", Class: "operation", Conv: conv,
		Args: map[string]string{"expected_generation": fmt.Sprint(gen), "expected_target_revision": fmt.Sprint(target)},
		Want: map[string]string{"status": "200", "revision": fmt.Sprint(target), "generation": fmt.Sprint(wantGen)}}
}

func tTurn(trial, prompt string, ex ...oq3Expect) oq3Step {
	return oq3Step{Op: "terminal-turn", Class: "required-dispatch", Trial: trial, Prompt: prompt, Expect: ex, Args: map[string]string{}}
}

func tSlash(cmd string) oq3Step {
	return oq3Step{Op: "terminal-slash", Class: "operation", Args: map[string]string{"cmd": cmd}}
}

func with(s oq3Step, kv ...string) oq3Step {
	if s.Args == nil {
		s.Args = map[string]string{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		s.Args[kv[i]] = kv[i+1]
	}
	return s
}

func oq3AllScenarios() []oq3Scenario {
	return []oq3Scenario{
		// ------------------------------------------------------------------ desktop, HTTP arm
		{ID: "D0", Title: "desktop unconfigured compatibility control (tools, thinking, attachment)",
			Trials: []oq3Trial{{"D0-C", famDisable, "desktop", "unconfigured control"}},
			Steps: []oq3Step{h("desktop-start"), h("auth-control"),
				dTurn("D0-C", "C", "OQ3 D0 turn 1: a plain question.", nD()),
				with(dTurn("D0-C", "C", "OQ3 D0 turn 2: with thinking requested.", nD()), "think", "true"),
				with(dTurn("D0-C", "C", "OQ3 D0 turn 3: with a text attachment.", nD()), "attachment", "notes.txt"),
				with(dTurn("D0-C", "C", "OQ3 D0 turn 4: with web search enabled.", nD()), "web_search", "true"),
				h("desktop-stop")}},
		{ID: "D1", Title: "desktop save, restart, new conversation",
			Trials: []oq3Trial{{"D1-X", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1), h("desktop-stop"), h("desktop-start"),
				dTurn("D1-X", "X", "OQ3 D1 turn 1.", cD(1, 1, "A")),
				dTurn("D1-X", "X", "OQ3 D1 turn 2.", cD(1, 1, "A")),
				h("desktop-stop")}},
		{ID: "D2", Title: "desktop edit while a conversation continues",
			Trials: []oq3Trial{{"D2-X", famEdit, "desktop", ""}, {"D2-Y", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D2-X", "X", "OQ3 D2 X turn 1.", cD(1, 1, "A")),
				dSave(1, true, "B", 2),
				dTurn("D2-X", "X", "OQ3 D2 X turn 2.", cD(1, 1, "A")),
				dTurn("D2-X", "X", "OQ3 D2 X turn 3.", cD(1, 1, "A")),
				dTurn("D2-Y", "Y", "OQ3 D2 Y turn 1.", cD(2, 1, "B")),
				h("desktop-stop")}},
		{ID: "D3", Title: "desktop explicit reload, then an idempotent reload",
			Trials: []oq3Trial{{"D3-X", famReload, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D3-X", "X", "OQ3 D3 turn 1.", cD(1, 1, "A")),
				dSave(1, true, "B", 2),
				dReload("X", 1, 2, 2),
				dTurn("D3-X", "X", "OQ3 D3 turn 2.", cD(2, 2, "B")),
				dReload("X", 2, 2, 2),
				dTurn("D3-X", "X", "OQ3 D3 turn 3.", cD(2, 2, "B")),
				h("desktop-stop")}},
		{ID: "D4", Title: "desktop disable, restart while disabled, restore, reset",
			Trials: []oq3Trial{{"D4-X", famDisable, "desktop", "active conversation keeps A after a global disable, then adopts it"},
				{"D4-Z", famDisable, "desktop", ""}, {"D4-W", famDisable, "desktop", "restart while disabled"},
				{"D4-V", famSave, "desktop", "restored text"}, {"D4-U", famDisable, "desktop", "reset"}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D4-X", "X", "OQ3 D4 X turn 1.", cD(1, 1, "A")),
				dSave(1, false, "A", 2),
				dTurn("D4-X", "X", "OQ3 D4 X turn 2.", cD(1, 1, "A")),
				dTurn("D4-Z", "Z", "OQ3 D4 Z turn 1.", nD()),
				dReload("X", 1, 2, 2),
				dTurn("D4-X", "X", "OQ3 D4 X turn 3.", nD()),
				h("desktop-stop"), h("desktop-start"),
				dTurn("D4-W", "W", "OQ3 D4 W turn 1.", nD()),
				dSave(2, true, "A", 3),
				dTurn("D4-V", "V", "OQ3 D4 V turn 1.", cD(3, 1, "A")),
				dSave(3, false, "", 4),
				dTurn("D4-U", "U", "OQ3 D4 U turn 1.", nD()),
				h("desktop-stop")}},
		{ID: "D5", Title: "desktop legacy transcript: revision zero, then explicit adoption",
			Trials: []oq3Trial{{"D5-L", famReload, "desktop", "legacy transcript written before G1"}},
			Steps: []oq3Step{with(h("legacy-fixture"), "conv", "L"), h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D5-L", "L", "OQ3 D5 legacy turn 2.", nD()),
				dReload("L", 1, 1, 2),
				dTurn("D5-L", "L", "OQ3 D5 legacy turn 3.", cD(1, 2, "A")),
				h("desktop-stop")}},
		{ID: "D6", Title: "desktop restart and reopen a pinned conversation",
			Trials: []oq3Trial{{"D6-X", famEdit, "desktop", "pinned across restart"}, {"D6-Y", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D6-X", "X", "OQ3 D6 X turn 1.", cD(1, 1, "A")),
				dSave(1, true, "B", 2),
				h("desktop-stop"), h("desktop-start"),
				dTurn("D6-X", "X", "OQ3 D6 X turn 2.", cD(1, 1, "A")),
				dTurn("D6-Y", "Y", "OQ3 D6 Y turn 1.", cD(2, 1, "B")),
				h("desktop-stop")}},
		{ID: "D7", Title: "desktop binding timing: an ID allocated before a save binds at its first turn",
			Trials: []oq3Trial{{"D7-Q", famSave, "desktop", "draft allocated at A, first turn after B"}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				{Op: "desktop-draft", Class: "operation", Conv: "Q", Want: map[string]string{"status": "200"}},
				dSave(1, true, "B", 2),
				dTurn("D7-Q", "Q", "OQ3 D7 first turn.", cD(2, 1, "B")),
				h("desktop-stop")}},
		{ID: "D8", Title: "desktop failed first turn, retry under the same ID, then a new ID",
			Trials: []oq3Trial{{"D8-F", famSave, "desktop", "reserved revision survives a failed first turn"}, {"D8-G", famSave, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				with(dTurn("D8-F", "F", "OQ3 D8 first turn, answered with an injected failure.", cD(1, 1, "A")), "fail", "500"),
				dSave(1, true, "B", 2),
				dTurn("D8-F", "F", "OQ3 D8 retry under the same ID.", cD(1, 1, "A")),
				dTurn("D8-G", "G", "OQ3 D8 new conversation.", cD(2, 1, "B")),
				h("desktop-stop")}},
		{ID: "D9", Title: "desktop held turn: a save from another process completes, a reload is refused, then reload at idle",
			Trials: []oq3Trial{{"D9-X", famReload, "desktop", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D9-X", "X", "OQ3 D9 turn 1.", cD(1, 1, "A")),
				with(dTurn("D9-X", "X", "OQ3 D9 turn 2, held by the daemon.", cD(1, 1, "A")), "hold", "1"),
				cliSet(1, "B"),
				{Op: "desktop-reload", Class: "required-refusal", Conv: "X",
					Args: map[string]string{"expected_generation": "1", "expected_target_revision": "2"},
					Want: map[string]string{"status": "409", "code": "conversation_busy"}},
				with(h("desktop-await"), "conv", "X"),
				dReload("X", 1, 2, 2),
				dTurn("D9-X", "X", "OQ3 D9 turn 3.", cD(2, 2, "B")),
				h("desktop-stop")}},
		{ID: "D10", Title: "desktop tool and model change keep the pinned revision",
			Trials: []oq3Trial{{"D10-X", famEdit, "desktop", "web search on, then another model"}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1),
				dTurn("D10-X", "X", "OQ3 D10 turn 1.", cD(1, 1, "A")),
				dSave(1, true, "B", 2),
				with(dTurn("D10-X", "X", "OQ3 D10 turn 2 with web search.", cD(1, 1, "A")), "web_search", "true"),
				with(dTurn("D10-X", "X", "OQ3 D10 turn 3 on another model.", cD(1, 1, "A")), "model", oq3Model2),
				h("desktop-stop")}},

		// ------------------------------------------------------------------ terminal, launch-function arm
		{ID: "T0", Title: "terminal unconfigured compatibility control (/system off and on, /tools off)",
			Trials: []oq3Trial{{"T0-X", famDisable, "terminal", "unconfigured control"}},
			Steps: []oq3Step{h("terminal-start"),
				tTurn("T0-X", "OQ3 T0 turn 1.", nT()),
				tSlash("/system off"),
				tTurn("T0-X", "OQ3 T0 turn 2, built-in prompt off.", nT()),
				tSlash("/system on"),
				tTurn("T0-X", "OQ3 T0 turn 3, built-in prompt on.", nT()),
				tSlash("/tools off"),
				tTurn("T0-X", "OQ3 T0 turn 4, tools off.", nT()),
				h("terminal-stop")}},
		{ID: "T1", Title: "terminal save, new process, two tool rounds",
			Trials: []oq3Trial{{"T1-X", famSave, "terminal", "every tool continuation carries the carrier"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T1-X", "OQ3 T1 turn 1.", cT(1, 1, "A")),
				with(tTurn("T1-X", "OQ3 T1 turn 2 "+oq3ToolTrigger+" read the fixture twice.", cT(1, 1, "A"), cT(1, 1, "A"), cT(1, 1, "A")), "tool_rounds", "2"),
				h("terminal-stop")}},
		{ID: "T2", Title: "terminal edit while continuing, then /new",
			Trials: []oq3Trial{{"T2-X", famEdit, "terminal", ""}, {"T2-Y", famReset, "terminal", "/new"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T2-X", "OQ3 T2 X turn 1.", cT(1, 1, "A")),
				cliSet(1, "B"),
				tTurn("T2-X", "OQ3 T2 X turn 2.", cT(1, 1, "A")),
				tSlash("/new"),
				tTurn("T2-Y", "OQ3 T2 Y turn 1.", cT(2, 1, "B")),
				h("terminal-stop"),
				{Op: "store-identity", Class: "operation", Want: map[string]string{"min_terminal": "2"}}}},
		{ID: "T3", Title: "terminal explicit reload",
			Trials: []oq3Trial{{"T3-X", famReload, "terminal", ""}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T3-X", "OQ3 T3 turn 1.", cT(1, 1, "A")),
				cliSet(1, "B"),
				tSlash("/instructions reload"),
				tTurn("T3-X", "OQ3 T3 turn 2.", cT(2, 2, "B")),
				h("terminal-stop")}},
		{ID: "T4", Title: "terminal disable, then a new process while disabled",
			Trials: []oq3Trial{{"T4-X", famDisable, "terminal", ""}, {"T4-Y", famDisable, "terminal", "new process after disable"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T4-X", "OQ3 T4 X turn 1.", cT(1, 1, "A")),
				tSlash("/instructions off"),
				tTurn("T4-X", "OQ3 T4 X turn 2.", nT()),
				h("terminal-stop"), h("terminal-start"),
				tTurn("T4-Y", "OQ3 T4 Y turn 1.", nT()),
				h("terminal-stop"),
				{Op: "store-identity", Class: "operation", Want: map[string]string{"min_terminal": "2"}}}},
		{ID: "T5", Title: "terminal binding timing: a session idle at A binds A before its first prompt",
			Trials: []oq3Trial{{"T5-X", famSave, "terminal", "B saved while idle before the first prompt"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"), cliSet(1, "B"),
				tTurn("T5-X", "OQ3 T5 first prompt.", cT(1, 1, "A")),
				h("terminal-stop")}},
		{ID: "T6", Title: "terminal compaction: manual, repeated, after reload, automatic",
			Trials: []oq3Trial{{"T6-X", famCompact, "terminal", ""}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T6-X", "OQ3 T6 turn 1.", cT(1, 1, "A")),
				tTurn("T6-X", "OQ3 T6 turn 2.", cT(1, 1, "A")),
				tTurn("T6-X", "OQ3 T6 turn 3.", cT(1, 1, "A")),
				tTurn("T6-X", "OQ3 T6 turn 4.", cT(1, 1, "A")),
				cliSet(1, "B"),
				withSummarisers(tSlash("/compact"), 1),
				tTurn("T6-X", "OQ3 T6 turn 5, after compaction.", cT(1, 1, "A")),
				withSummarisers(tSlash("/compact"), 1),
				tTurn("T6-X", "OQ3 T6 turn 6, after a second compaction.", cT(1, 1, "A")),
				tSlash("/instructions reload"),
				tTurn("T6-X", "OQ3 T6 turn 7, after reload.", cT(2, 2, "B")),
				withSummarisers(tSlash("/compact"), 1),
				tTurn("T6-X", "OQ3 T6 turn 8, after compacting the reloaded conversation.", cT(2, 2, "B")),
				withSummarisers(with(tTurn("T6-X", "OQ3 T6 turn 9, near the context limit.", cT(2, 2, "B")), "auto_compact", "1"), 1),
				tTurn("T6-X", "OQ3 T6 turn 10, after automatic compaction.", cT(2, 2, "B")),
				h("terminal-stop")}},
		{ID: "T7", Title: "terminal built-in prompt off keeps the custom instructions",
			Trials: []oq3Trial{{"T7-X", famEdit, "terminal", "/system off then on"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tSlash("/system off"),
				tTurn("T7-X", "OQ3 T7 turn 1, built-in prompt off.", cT(1, 1, "A")),
				tSlash("/system on"),
				tTurn("T7-X", "OQ3 T7 turn 2, built-in prompt on.", cT(1, 1, "A")),
				h("terminal-stop")}},
		{ID: "T8", Title: "terminal held turn: a save from another process completes, a reload is refused, then reload at idle",
			Trials: []oq3Trial{{"T8-X", famReload, "terminal", ""}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T8-X", "OQ3 T8 turn 1.", cT(1, 1, "A")),
				with(tTurn("T8-X", "OQ3 T8 turn 2, held by the daemon.", cT(1, 1, "A")), "hold", "1"),
				cliSet(1, "B"),
				{Op: "terminal-slash", Class: "required-refusal", Args: map[string]string{"cmd": "/instructions reload"}, Want: map[string]string{"refused": "true"}},
				h("terminal-await"),
				tSlash("/instructions reload"),
				tTurn("T8-X", "OQ3 T8 turn 3.", cT(2, 2, "B")),
				h("terminal-stop")}},
		{ID: "T9", Title: "terminal tools and model switches keep the pinned revision",
			Trials: []oq3Trial{{"T9-X", famEdit, "terminal", "/tools off, /tools on, /model"}},
			Steps: []oq3Step{cliSet(0, "A"), h("terminal-start"),
				tTurn("T9-X", "OQ3 T9 turn 1.", cT(1, 1, "A")),
				cliSet(1, "B"),
				tSlash("/tools off"),
				tTurn("T9-X", "OQ3 T9 turn 2, tools off.", cT(1, 1, "A")),
				tSlash("/tools on"),
				with(tSlash("/model"), "pick", oq3Model2),
				tTurn("T9-X", "OQ3 T9 turn 3, on another model.", cT(1, 1, "A")),
				h("terminal-stop")}},

		// ------------------------------------------------------------------ entry arm (built binary)
		{ID: "E1", Title: "entry: the built binary's CLI saves; its launcher reaches a new terminal conversation",
			Trials: []oq3Trial{{"E1-X", famSave, "terminal", "built binary, launcher menu, model picker"}},
			Steps: []oq3Step{cliSet(0, "A"), h("entry-start"),
				tTurn("E1-X", "OQ3 E1 turn 1.", cT(1, 1, "A")),
				h("entry-stop")}},

		// ------------------------------------------------------------------ cross-adapter
		{ID: "X1", Title: "cross-adapter: saved in the desktop, used by a new terminal",
			Trials: []oq3Trial{{"X1-T", famSave, "terminal", ""}},
			Steps: []oq3Step{h("desktop-start"), dSave(0, true, "A", 1), h("desktop-stop"), h("terminal-start"),
				tTurn("X1-T", "OQ3 X1 terminal turn 1.", cT(1, 1, "A")),
				h("terminal-stop")}},
		{ID: "X2", Title: "cross-adapter: saved by the CLI, used by a new desktop conversation",
			Trials: []oq3Trial{{"X2-D", famSave, "desktop", ""}},
			Steps: []oq3Step{cliSet(0, "A"), h("desktop-start"),
				dTurn("X2-D", "Y", "OQ3 X2 desktop turn 1.", cD(1, 1, "A")),
				h("desktop-stop")}},
		{ID: "X3", Title: "cross-adapter: disabled in the terminal; new desktop conversations follow, a bound one does not",
			Trials: []oq3Trial{{"X3-DC", famDisable, "desktop", "bound before the disable"}, {"X3-TX", famDisable, "terminal", ""}, {"X3-DE", famDisable, "desktop", "new after the disable"}},
			Steps: []oq3Step{cliSet(0, "A"), h("desktop-start"),
				dTurn("X3-DC", "C", "OQ3 X3 desktop C turn 1.", cD(1, 1, "A")),
				h("terminal-start"),
				tTurn("X3-TX", "OQ3 X3 terminal turn 1.", cT(1, 1, "A")),
				tSlash("/instructions off"),
				tTurn("X3-TX", "OQ3 X3 terminal turn 2.", nT()),
				dTurn("X3-DE", "E", "OQ3 X3 desktop E turn 1.", nD()),
				dTurn("X3-DC", "C", "OQ3 X3 desktop C turn 2.", cD(1, 1, "A")),
				h("terminal-stop"), h("desktop-stop")}},
	}
}

func withSummarisers(s oq3Step, n int) oq3Step {
	s.Summarisers = n
	return s
}
