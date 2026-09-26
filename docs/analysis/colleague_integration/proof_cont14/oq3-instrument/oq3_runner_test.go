//go:build darwin

// OQ-3 runner: executes the ledger, records every event, scores every delivery, writes the evidence.

package cmd

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type oq3Event struct {
	Scenario string `json:"scenario"`
	Step     int    `json:"step"`
	Op       string `json:"op"`
	Class    string `json:"class"`
	Label    string `json:"label"`
	Trial    string `json:"trial,omitempty"`
	Verdict  string `json:"verdict"` // pass | fail | absent | done | harness-failed
	Outcome  any    `json:"outcome,omitempty"`
	At       string `json:"at"`
}

type oq3Run struct {
	t        *testing.T
	sc       oq3Scenario
	d        *oq3Daemon
	s        *oq3Sandbox
	desk     *oq3Desktop
	deskLife int
	term     *oq3Terminal
	termN    int
	convIDs  map[string]string
	async    map[string]chan asyncResult
	marks    map[string]int
	// marksLabel remembers which label a held turn was sent under, per conversation handle.
	marksLabel map[string]string
	// builtBinary, when set, is the built `ollama` used by the entry arm for CLI steps.
	builtBinary []string
	events      []oq3Event
	invalid     []string
}

func oq3NewRun(t *testing.T, sc oq3Scenario, d *oq3Daemon, s *oq3Sandbox) *oq3Run {
	return &oq3Run{t: t, sc: sc, d: d, s: s, convIDs: map[string]string{}, async: map[string]chan asyncResult{},
		marks: map[string]int{}, marksLabel: map[string]string{}}
}

type asyncResult struct {
	id  string
	res oq3HTTPResult
}

func oq3Label(sc string, i int, op string) string { return fmt.Sprintf("%s/%02d/%s", sc, i, op) }

func (r *oq3Run) event(i int, st oq3Step, verdict string, outcome any) {
	r.events = append(r.events, oq3Event{Scenario: r.sc.ID, Step: i, Op: st.Op, Class: st.Class, Label: oq3Label(r.sc.ID, i, st.Op),
		Trial: st.Trial, Verdict: verdict, Outcome: outcome, At: time.Now().Format(time.RFC3339Nano)})
	if verdict == "harness-failed" {
		r.invalid = append(r.invalid, fmt.Sprintf("%s step %d %s: %v", r.sc.ID, i, st.Op, outcome))
	}
}

// execute runs every step. A harness failure stops the scenario (later steps are recorded as not
// run) and makes the run invalid; it is never scored as a delivery defect.
func (r *oq3Run) execute() {
	for i, st := range r.sc.Steps {
		label := oq3Label(r.sc.ID, i, st.Op)
		switch st.Op {
		case "terminal-await":
			// Requests caused while a held turn completes (its tool continuations) belong to that
			// turn, so its label stays active.
			r.d.setLabel(r.heldLabel("terminal"))
		case "desktop-await":
			r.d.setLabel(r.heldLabel(st.Args["conv"]))
		default:
			r.d.setLabel(label)
		}
		if len(r.invalid) > 0 {
			r.event(i, st, "not-run", "an earlier harness failure stopped this scenario")
			continue
		}
		func() {
			defer func() {
				if p := recover(); p != nil {
					r.event(i, st, "harness-failed", fmt.Sprint(p))
				}
			}()
			r.step(i, st, label)
		}()
	}
	if r.term != nil {
		r.term.kill()
	}
	if r.desk != nil {
		_ = r.desk.cmd.Process.Kill()
	}
}

func (r *oq3Run) harnessFail(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}

func (r *oq3Run) step(i int, st oq3Step, label string) {
	switch st.Op {
	case "desktop-start":
		r.deskLife++
		r.desk = oq3StartDesktopNoFatal(r, r.deskLife)
		r.event(i, st, "done", map[string]any{"life": r.deskLife})
	case "desktop-stop":
		if err := r.desk.stopNoFatal(); err != nil {
			r.harnessFail("desktop stop: %v", err)
		}
		r.event(i, st, "done", map[string]any{"life": r.deskLife, "exited": true})
		r.desk = nil
	case "auth-control":
		unauth := r.desk.doAuth(http.MethodGet, "/api/v1/settings", nil, false)
		auth := r.desk.doAuth(http.MethodGet, "/api/v1/settings", nil, true)
		ok := unauth.Status == http.StatusForbidden && auth.Status == http.StatusOK && strings.Contains(auth.Body, `"settings"`)
		if !ok {
			r.harnessFail("auth control: unauthenticated %d, authenticated %d", unauth.Status, auth.Status)
		}
		r.event(i, st, "done", map[string]any{"unauthenticated": unauth.Status, "authenticated": auth.Status})
	case "legacy-fixture":
		id := "0192f0a0-3c3d-7000-8000-0000000000aa"
		c := oq3HelperCmd(r.s, r.d, "legacy", "OQ3_CHAT_ID="+id)
		out, err := c.CombinedOutput()
		if err != nil {
			r.harnessFail("legacy fixture: %v: %s", err, out)
		}
		r.convIDs[st.Args["conv"]] = id
		r.event(i, st, "done", map[string]any{"chat_id": id})
	case "desktop-save":
		body := map[string]any{"expected_revision": st.Args["expected_revision"],
			"edit": map[string]any{"enabled": st.Args["enabled"] == "true", "text": oq3TextOrEmpty(st.Args["text"]), "binding": nil}}
		res := r.desk.do(http.MethodPut, "/api/v1/instructions", body)
		verdict := oq3ProfileVerdict(res, st.Want)
		if verdict == "pass" {
			verdict = r.storeCommitted(st.Want)
		}
		r.event(i, st, verdict, res)
	case "desktop-draft":
		res := r.desk.do(http.MethodPost, "/api/v1/create-chat", map[string]any{})
		var v struct {
			ID string `json:"id"`
		}
		if res.Status != 200 || json.Unmarshal([]byte(res.Body), &v) != nil || v.ID == "" {
			r.event(i, st, "fail", res)
			r.harnessFail("create-chat did not return an id: %d %s", res.Status, res.Body)
		}
		r.convIDs[st.Conv] = v.ID
		r.event(i, st, "pass", map[string]any{"status": res.Status, "chat_id": v.ID})
	case "desktop-turn":
		r.desktopTurn(i, st, label)
	case "desktop-await":
		ch := r.async[st.Args["conv"]]
		r.d.release(r.heldLabel(st.Args["conv"]))
		select {
		case ar := <-ch:
			r.convIDs[st.Args["conv"]] = ar.id
			r.event(i, st, "done", map[string]any{"status": ar.res.Status})
		case <-time.After(60 * time.Second):
			r.harnessFail("held desktop turn never completed")
		}
	case "desktop-reload":
		id := r.convIDs[st.Conv]
		if id == "" {
			r.harnessFail("reload of unknown conversation %q", st.Conv)
		}
		body := map[string]any{"expected_generation": st.Args["expected_generation"], "expected_target_revision": st.Args["expected_target_revision"]}
		res := r.desk.do(http.MethodPost, "/api/v1/chat/"+id+"/instructions/reload", body)
		r.event(i, st, oq3ReloadVerdict(res, st.Want), res)
	case "cli-set":
		path := filepath.Join(r.s.tmp, fmt.Sprintf("instructions-%s-%02d.txt", r.sc.ID, i))
		if err := os.WriteFile(path, []byte(oq3TextOrEmpty(st.Args["text"])), 0o600); err != nil {
			r.harnessFail("write instructions file: %v", err)
		}
		res := oq3RunCLINoFatal(r, "instructions", "set", "--file", path, "--expected-revision", st.Args["expected_revision"])
		verdict := "fail"
		switch {
		case strings.Contains(res.Output, "unknown command"):
			verdict = "absent"
		case res.ExitCode == 0:
			// Blueprint §7 as resolved in continuation 14: `set` saves the file's text as an
			// enabled profile. Exit 0 counts only when the store shows it committed.
			verdict = r.storeCommitted(map[string]string{"revision": st.Want["revision"], "enabled": "true", "text": st.Args["text"]})
		}
		r.event(i, st, verdict, res)
	case "terminal-start":
		r.termN++
		r.term = oq3StartTerminalNoFatal(r, fmt.Sprintf("%s-%d", r.sc.ID, r.termN), st.Args["model"])
		if !oq3WaitFor(30*time.Second, func() bool {
			return r.d.countLabel("POST", "/api/generate", label) >= 1 || r.term.hasExited()
		}) || r.term.hasExited() {
			r.harnessFail("terminal session never preloaded; screen tail %q", tail(r.term.screen(), 1500))
		}
		r.mustQuiet("after the session started")
		r.event(i, st, "done", map[string]any{"session": r.termN})
	case "terminal-stop":
		r.term.typeLine("/bye")
		if err, ok := r.term.waitExit(20 * time.Second); !ok || err != nil {
			r.harnessFail("terminal did not exit cleanly on /bye: ok=%v err=%v", ok, err)
		}
		r.event(i, st, "done", map[string]any{"session": r.termN, "exited": true})
		r.term.kill()
		r.term = nil
	case "entry-start":
		// The entry arm: the built `ollama` binary with no arguments, through the launcher menu
		// (cmd/cmd.go:2161-2260) to the agent chat, exactly as the operator starts it.
		if len(r.builtBinary) == 0 {
			r.harnessFail("entry arm needs the built binary (OQ3_BINARY)")
		}
		r.termN++
		r.term = oq3StartTerminalWith(oq3Panic, r.s, r.d, fmt.Sprintf("%s-%d", r.sc.ID, r.termN), "", r.builtBinary)
		mark := 0
		if !oq3WaitFor(30*time.Second, func() bool { return r.term.screenContainsAfter(mark, "Chat, Code") || r.term.hasExited() }) || r.term.hasExited() {
			r.harnessFail("launcher menu never appeared; screen tail %q", tail(r.term.screen(), 1500))
		}
		r.term.quiet(500*time.Millisecond, 10*time.Second)
		mark = r.term.rawLen()
		r.term.key("\r")
		if !oq3WaitFor(30*time.Second, func() bool {
			return r.term.screenContainsAfter(mark, "Select model") || r.d.countLabel("POST", "/api/generate", label) >= 1 || r.term.hasExited()
		}) || r.term.hasExited() {
			r.harnessFail("the run-model action reached neither a model picker nor the chat; screen tail %q", tail(r.term.screen(), 1500))
		}
		picked := false
		if r.d.countLabel("POST", "/api/generate", label) == 0 {
			r.term.quiet(400*time.Millisecond, 10*time.Second)
			r.term.key(oq3Model)
			r.term.quiet(400*time.Millisecond, 10*time.Second)
			r.term.key("\r")
			picked = true
		}
		if !oq3WaitFor(30*time.Second, func() bool { return r.d.countLabel("POST", "/api/generate", label) >= 1 || r.term.hasExited() }) || r.term.hasExited() {
			r.harnessFail("the agent chat never preloaded after the launcher; screen tail %q", tail(r.term.screen(), 1500))
		}
		r.term.quiet(700*time.Millisecond, 15*time.Second)
		r.event(i, st, "done", map[string]any{"session": r.termN, "model_picker_used": picked, "binary": r.builtBinary[0]})
	case "entry-stop":
		mark := r.term.rawLen()
		r.term.typeLine("/bye")
		if !oq3WaitFor(20*time.Second, func() bool { return r.term.screenContainsAfter(mark, "Chat, Code") || r.term.hasExited() }) {
			r.harnessFail("the launcher menu did not return after /bye; screen tail %q", tail(r.term.screen(), 1500))
		}
		if !r.term.hasExited() {
			r.term.quiet(400*time.Millisecond, 10*time.Second)
			r.term.key("q")
		}
		if err, ok := r.term.waitExit(20 * time.Second); !ok || err != nil {
			r.harnessFail("the launcher did not exit cleanly: ok=%v err=%v", ok, err)
		}
		r.event(i, st, "done", map[string]any{"session": r.termN, "exited": true})
		r.term.kill()
		r.term = nil
	case "terminal-turn":
		r.terminalTurn(i, st, label)
	case "terminal-await":
		lbl := r.heldLabel("terminal")
		mark := r.marks["terminal"]
		r.d.release(lbl)
		r.finishTerminalTurn(lbl, r.marks["terminal-rounds"], 0, mark)
		r.event(i, st, "done", nil)
	case "terminal-slash":
		r.terminalSlash(i, st, label)
	case "store-identity":
		r.event(i, st, r.storeIdentity(st.Want), nil)
	case "store-events":
		r.event(i, st, r.storeEvents(), nil)
	default:
		r.harnessFail("unknown op %q", st.Op)
	}
}

func (r *oq3Run) heldLabel(conv string) string { return r.marksLabel[conv] }

func oq3TextOrEmpty(key string) string {
	if key == "" {
		return ""
	}
	return oq3Texts[key]
}

// ---------------------------------------------------------------------------------------------
// Desktop turns.

func (r *oq3Run) desktopTurn(i int, st oq3Step, label string) {
	id := r.convIDs[st.Conv]
	if id == "" {
		id = "new"
	}
	body := map[string]any{"model": oq3Model, "prompt": st.Prompt}
	if m := st.Args["model"]; m != "" {
		body["model"] = m
	}
	if st.Args["think"] == "true" {
		body["think"] = true
	}
	if st.Args["web_search"] == "true" {
		body["web_search"] = true
	}
	if name := st.Args["attachment"]; name != "" {
		body["attachments"] = []any{map[string]any{"filename": name, "data": base64.StdEncoding.EncodeToString([]byte("OQ3 attachment line one.\nOQ3 attachment line two.\n"))}}
	}
	if s := st.Args["fail"]; s != "" {
		n, _ := strconv.Atoi(s)
		r.d.failNext(label, n)
	}
	if st.Args["hold"] == "1" {
		r.d.hold(label)
		r.marksLabel[st.Conv] = label
		ch := make(chan asyncResult, 1)
		r.async[st.Conv] = ch
		desk := r.desk
		go func() {
			newID, res := desk.chatBody(id, body)
			ch <- asyncResult{id: newID, res: res}
		}()
		if !oq3WaitFor(30*time.Second, func() bool { return r.d.countLabel("POST", "/api/chat", label) >= 1 }) {
			r.harnessFail("held desktop turn never reached the daemon")
		}
		r.event(i, st, "dispatched-held", map[string]any{"conversation": st.Conv})
		return
	}
	newID, res := r.desk.chatBody(id, body)
	if newID != "" && newID != "new" {
		r.convIDs[st.Conv] = newID
	}
	r.event(i, st, "done", map[string]any{"status": res.Status, "chat_id": newID, "stream_tail": tail(res.Body, 400)})
}

// ---------------------------------------------------------------------------------------------
// Terminal turns and commands.

func (r *oq3Run) terminalTurn(i int, st oq3Step, label string) {
	rounds, _ := strconv.Atoi(st.Args["tool_rounds"])
	if rounds > 0 {
		r.d.setToolRounds(label, rounds)
	}
	if st.Args["auto_compact"] == "1" {
		r.d.reportPromptEval(label, 30000)
	}
	mark := r.term.rawLen()
	if st.Args["hold"] == "1" {
		r.d.hold(label)
		r.marksLabel["terminal"] = label
		r.marks["terminal"] = mark
	}
	r.term.typeLine(st.Prompt)
	if !oq3WaitFor(30*time.Second, func() bool { return r.d.countLabel("POST", "/api/chat", label) >= 1 }) {
		// No request reached the daemon: recorded, and scored by the step's class (a missing
		// required delivery, or a refusal held as required).
		r.mustQuiet("after a turn that sent no request")
		r.event(i, st, "no-request", map[string]any{"screen_tail": tail(r.term.screen(), 1200)})
		return
	}
	if st.Args["hold"] == "1" {
		r.marks["terminal-rounds"] = rounds
		r.event(i, st, "dispatched-held", nil)
		return
	}
	r.finishTerminalTurn(label, rounds, st.Summarisers, mark)
	r.event(i, st, "done", nil)
}

// finishTerminalTurn approves each declared tool round, then waits until every response under the
// label has been fully written by the daemon and the screen has gone quiet (review finding 8).
func (r *oq3Run) finishTerminalTurn(label string, rounds, summarisers, mark int) {
	for k := 0; k < rounds; k++ {
		approvalMark := mark
		if !oq3WaitFor(30*time.Second, func() bool { return r.term.screenContainsAfter(approvalMark, "Approve once") }) {
			r.harnessFail("tool round %d: approval prompt never appeared; screen tail %q", k+1, tail(r.term.screen(), 1500))
		}
		want := r.d.countLabel("POST", "/api/chat", label) + 1
		mark = r.term.rawLen()
		r.term.key("1")
		if !oq3WaitFor(30*time.Second, func() bool { return r.d.countLabel("POST", "/api/chat", label) >= want }) {
			r.harnessFail("tool round %d: continuation never reached the daemon", k+1)
		}
	}
	if !oq3WaitFor(30*time.Second, func() bool {
		n := r.d.countLabel("POST", "/api/chat", label)
		return r.d.countSummaries(label) >= summarisers && r.d.completedLabel(label) >= n
	}) {
		r.harnessFail("responses under %s never completed (%d requests, %d completed, %d summaries)", label,
			r.d.countLabel("POST", "/api/chat", label), r.d.completedLabel(label), r.d.countSummaries(label))
	}
	r.mustQuiet("after the turn completed")
}

// mustQuiet waits for the terminal to stop producing output; a timeout is a harness failure, never
// a reason to advance (review finding 8).
func (r *oq3Run) mustQuiet(when string) {
	if !r.term.quiet(600*time.Millisecond, 20*time.Second) {
		r.harnessFail("terminal did not go quiet %s; screen tail %q", when, tail(r.term.screen(), 800))
	}
}

func (r *oq3Run) terminalSlash(i int, st oq3Step, label string) {
	cmd := st.Args["cmd"]
	mark := r.term.rawLen()
	if pick := st.Args["pick"]; pick != "" {
		r.term.typeLine(cmd)
		if !oq3WaitFor(15*time.Second, func() bool { return r.term.screenContainsAfter(mark, pick) }) {
			r.harnessFail("model picker never listed %s; screen tail %q", pick, tail(r.term.screen(), 1500))
		}
		r.term.quiet(400*time.Millisecond, 10*time.Second)
		r.term.key(pick)
		r.term.quiet(400*time.Millisecond, 10*time.Second)
		r.term.key("\r")
		r.term.quiet(700*time.Millisecond, 15*time.Second)
		r.event(i, st, "done", map[string]any{"picked": pick, "screen_tail": tail(r.term.screen(), 600)})
		return
	}
	r.term.typeLine(cmd)
	if st.Class == "required-refusal" {
		// Sent while a held turn runs: the screen only animates, so nothing on it can show the
		// refusal. Its effect is judged by the held turn's own continuation, which must keep the
		// old revision (review finding 1). No wait: the turn is still running.
		r.event(i, st, "sent-while-running", nil)
		return
	}
	if st.Summarisers > 0 {
		if !oq3WaitFor(30*time.Second, func() bool {
			return r.d.countSummaries(label) >= st.Summarisers && r.d.completedLabel(label) >= r.d.countLabel("POST", "/api/chat", label)
		}) {
			r.mustQuiet("after a compaction that did not complete")
			r.event(i, st, "fail", map[string]any{"summarisers_seen": r.d.countSummaries(label), "screen_tail": tail(r.term.screen(), 1200)})
			return
		}
	}
	r.mustQuiet("after " + cmd)
	unknown := r.term.screenContainsAfter(mark, "Unknown command") || r.term.screenContainsAfter(mark, "usage: ")
	verdict := "done"
	switch {
	case unknown && strings.HasPrefix(cmd, "/instructions"):
		verdict = "absent"
	case unknown:
		// An existing command that printed an error or its usage did not do what the step declares.
		r.harnessFail("%s was rejected by the terminal; screen tail %q", cmd, tail(oq3ANSI.ReplaceAllString(r.term.rawSince(mark), ""), 600))
	case strings.HasPrefix(cmd, "/instructions"):
		// No error was printed. The blueprint (§7) names no success output, so the screen cannot
		// show success; the delivery after this step decides (review finding 1).
		verdict = "no-error-seen"
	}
	r.event(i, st, verdict, map[string]any{"screen_tail": tail(oq3ANSI.ReplaceAllString(r.term.rawSince(mark), ""), 600)})
}

// ---------------------------------------------------------------------------------------------
// Operation verdicts. "absent" means the designated surface does not exist at this commit (the
// application's HTML fallback or an unknown command); it is a failed operation, not a harness fault.

func oq3IsHTML(res oq3HTTPResult) bool {
	return strings.Contains(strings.ToLower(res.ContentType), "text/html") || strings.HasPrefix(strings.TrimSpace(strings.ToLower(res.Body)), "<!doctype html")
}

func oq3ProfileVerdict(res oq3HTTPResult, want map[string]string) string {
	if oq3IsHTML(res) || res.Status == http.StatusNotFound {
		return "absent"
	}
	var p struct {
		Revision string `json:"revision"`
		Enabled  bool   `json:"enabled"`
		Text     string `json:"text"`
	}
	if res.Status != 200 || json.Unmarshal([]byte(res.Body), &p) != nil {
		return "fail"
	}
	if p.Revision != want["revision"] || fmt.Sprint(p.Enabled) != want["enabled"] || p.Text != oq3TextOrEmpty(want["text"]) {
		return "fail"
	}
	return "pass"
}

func oq3ReloadVerdict(res oq3HTTPResult, want map[string]string) string {
	if oq3IsHTML(res) || res.Status == http.StatusNotFound {
		return "absent"
	}
	if fmt.Sprint(res.Status) != want["status"] {
		return "fail"
	}
	if want["code"] != "" {
		var e struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(res.Body), &e) != nil || e.Error.Code != want["code"] {
			return "fail"
		}
		return "pass"
	}
	var sel struct {
		Revision   string `json:"revision"`
		Generation string `json:"generation"`
	}
	if json.Unmarshal([]byte(res.Body), &sel) != nil || sel.Revision != want["revision"] || sel.Generation != want["generation"] {
		return "fail"
	}
	return "pass"
}

func (r *oq3Run) storePath() string {
	return filepath.Join(r.s.home, ".ollama", "instructions", "instructions.sqlite")
}

// storeCommitted reads the designated store's current profile (blueprint §3.1 schema) read-only:
// "absent" before G1, "pass" when it holds the expected revision, enabled flag and text (review
// finding 10: an exit status or a response body alone is not a committed save).
func (r *oq3Run) storeCommitted(want map[string]string) string {
	if _, err := os.Stat(r.storePath()); err != nil {
		return "absent"
	}
	db, err := sql.Open("sqlite3", "file:"+r.storePath()+"?mode=ro")
	if err != nil {
		return "fail"
	}
	defer db.Close()
	var rev int64
	var enabled int
	var text string
	if err := db.QueryRow(`SELECT m.current_revision, r.enabled, r.text FROM metadata m JOIN revisions r ON r.revision = m.current_revision WHERE m.id = 1`).Scan(&rev, &enabled, &text); err != nil {
		return "fail"
	}
	if fmt.Sprint(rev) == want["revision"] && fmt.Sprint(enabled == 1) == want["enabled"] && text == oq3TextOrEmpty(want["text"]) {
		return "pass"
	}
	return "fail"
}

// storeEvents checks the terminal event sequence of blueprint §3.1: two conversations opened under
// distinct identities, and one reload recorded under the SECOND identity, from revision 2 to 3 at
// generation 2 (review finding 11: a new binding the runtime never uses would leave the reload on
// the first identity).
func (r *oq3Run) storeEvents() string {
	if _, err := os.Stat(r.storePath()); err != nil {
		return "absent"
	}
	db, err := sql.Open("sqlite3", "file:"+r.storePath()+"?mode=ro")
	if err != nil {
		return "fail"
	}
	defer db.Close()
	rows, err := db.Query(`SELECT conversation_id, generation, kind, COALESCE(from_revision, -1), to_revision FROM events WHERE namespace = 'terminal' ORDER BY rowid`)
	if err != nil {
		return "fail"
	}
	defer rows.Close()
	type ev struct {
		id         string
		gen        int
		kind       string
		from, to   int
	}
	var evs []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.id, &e.gen, &e.kind, &e.from, &e.to); err != nil {
			return "fail"
		}
		evs = append(evs, e)
	}
	var opened []ev
	var reloaded []ev
	for _, e := range evs {
		switch e.kind {
		case "opened":
			opened = append(opened, e)
		case "reloaded":
			reloaded = append(reloaded, e)
		}
	}
	if len(opened) != 2 || len(reloaded) != 1 || opened[0].id == opened[1].id {
		return "fail"
	}
	rl := reloaded[0]
	if rl.id != opened[1].id || rl.gen != 2 || rl.from != 2 || rl.to != 3 || opened[0].to != 1 || opened[1].to != 2 {
		return "fail"
	}
	return "pass"
}

// storeIdentity reads the designated instruction store (blueprint §3, §3.1) for distinct terminal
// conversation identities. Before G1 the store does not exist, which is reported as absent.
func (r *oq3Run) storeIdentity(want map[string]string) string {
	path := filepath.Join(r.s.home, ".ollama", "instructions", "instructions.sqlite")
	if _, err := os.Stat(path); err != nil {
		return "absent"
	}
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		return "fail"
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT conversation_id) FROM conversations WHERE namespace = 'terminal'`).Scan(&n); err != nil {
		return "fail"
	}
	min, _ := strconv.Atoi(want["min_terminal"])
	if n >= min {
		return "pass"
	}
	return "fail"
}

// ---------------------------------------------------------------------------------------------
// Scoring.

type oq3Delivery struct {
	Scenario string             `json:"scenario"`
	Step     int                `json:"step"`
	Label    string             `json:"label"`
	Trial    string             `json:"trial"`
	Index    int                `json:"index"`
	Expect   *oq3Expect         `json:"expect,omitempty"`
	Seq      int                `json:"capture_seq,omitempty"`
	Kind     string             `json:"kind,omitempty"`   // delivery (default) | summariser | preload | leak-scan
	Status   string             `json:"status"` // correct | bad | missing | unexpected | refused-as-required
	Verdict  oq3DeliveryVerdict `json:"verdict"`
	Golden   string             `json:"golden,omitempty"`
}

type oq3ScenarioResult struct {
	Scenario   oq3Scenario   `json:"scenario"`
	Events     []oq3Event    `json:"events"`
	Deliveries []oq3Delivery `json:"deliveries"`
	Auxiliary  []oq3Delivery `json:"auxiliary"`
	Captures   []oq3Capture  `json:"captures"`
	Invalid    []string      `json:"invalid,omitempty"`
	NotMeasured []string     `json:"not_measured,omitempty"`
	Sandbox    string        `json:"sandbox"`
}

func oq3GoldenPath(dir, sc string, step, idx int) string {
	return filepath.Join(dir, sc, fmt.Sprintf("%02d-%d.json", step, idx))
}

type oq3GoldenFile struct {
	Scenario string    `json:"scenario"`
	Step     int       `json:"step"`
	Index    int       `json:"index"`
	Label    string    `json:"label"`
	Trial    string    `json:"trial"`
	Expect   oq3Expect `json:"expect"`
	Kind     string    `json:"kind,omitempty"`
	Request  any       `json:"request"`
	Source   string    `json:"source"`
}

// oq3KnownPaths are the daemon endpoints the entry points are known to call. Anything else is an
// unclassified request and makes the run invalid (review finding 4).
var oq3KnownPaths = map[string]bool{"/": true, "/api/version": true, "/api/show": true, "/api/tags": true, "/api/ps": true,
	"/api/generate": true, "/api/chat": true, "/api/status": true, "/api/me": true, "/api/pull": true,
	"/api/experimental/model-recommendations": true}

// score attributes every captured request to the step that caused it and scores it.
// freeze: when a golden is absent, the observed pre-G1 request becomes the golden (baseline only).
func (r *oq3Run) score(goldenDir string, freeze bool, source string) oq3ScenarioResult {
	res := oq3ScenarioResult{Scenario: r.sc, Events: r.events, Invalid: append([]string(nil), r.invalid...), Sandbox: r.s.root}
	caps := r.d.snapshot()
	res.Captures = caps
	notRun := map[int]bool{}
	lastHeld := -1
	for _, e := range r.events {
		if e.Verdict == "dispatched-held" {
			lastHeld = e.Step
		}
		if e.Verdict == "not-run" || e.Verdict == "harness-failed" {
			notRun[e.Step] = true
			// A held turn whose completion failed was not measured either.
			if (e.Op == "terminal-await" || e.Op == "desktop-await") && lastHeld >= 0 {
				notRun[lastHeld] = true
			}
		}
	}
	type key struct {
		label, path string
	}
	byKey := map[key][]oq3Capture{}
	for _, c := range caps {
		if !oq3KnownPaths[c.Path] {
			res.Invalid = append(res.Invalid, fmt.Sprintf("unclassified request %s %s under %q", c.Method, c.Path, c.Label))
		}
		byKey[key{c.Label, c.Path}] = append(byKey[key{c.Label, c.Path}], c)
	}
	known := map[string]bool{}
	scoredSeq := map[int]bool{} // /api/chat captures scored as conversation deliveries
	loadOrFreeze := func(gp string, gf oq3GoldenFile, obs any) (any, error) {
		golden, err := oq3LoadGolden(gp)
		if err == nil {
			return golden, nil
		}
		if !freeze {
			return nil, err
		}
		gf.Request = obs
		gf.Source = source
		if werr := oq3WriteJSON(gp, gf); werr != nil {
			r.t.Fatalf("write golden: %v", werr)
		}
		return obs, nil
	}
	aux := func(d oq3Delivery) { res.Auxiliary = append(res.Auxiliary, d) }
	for i, st := range r.sc.Steps {
		label := oq3Label(r.sc.ID, i, st.Op)
		known[label] = true
		if notRun[i] {
			res.NotMeasured = append(res.NotMeasured, fmt.Sprintf("%s (%s)", label, st.Trial))
			continue
		}
		var turns, sums []oq3Capture
		for _, c := range byKey[key{label, "/api/chat"}] {
			if c.Response == "summary" {
				sums = append(sums, c)
			} else {
				turns = append(turns, c)
			}
		}
		// Summariser requests: allowed only where the step declares them, exactly that many, each
		// equal to its frozen pre-G1 golden (review findings 2 and 3).
		if len(sums) != st.Summarisers {
			if st.Summarisers > 0 && len(sums) < st.Summarisers {
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: compaction did not happen (%d summariser requests, %d declared)", label, len(sums), st.Summarisers))
			}
			if len(sums) > st.Summarisers {
				aux(oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Status: "bad",
					Verdict: oq3DeliveryVerdict{Categories: []string{"undeclared-summariser"}, Detail: fmt.Sprintf("%d summariser requests, %d declared", len(sums), st.Summarisers)}})
			}
		}
		for k, c := range sums {
			d := oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Index: k, Seq: c.Seq, Kind: "summariser"}
			obs, err := oq3Normalise(c.Body, r.s.root)
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"malformed-capture"}}
				aux(d)
				continue
			}
			gp := filepath.Join(goldenDir, r.sc.ID, fmt.Sprintf("%02d-s%d.json", i, k))
			d.Golden = gp
			golden, gerr := loadOrFreeze(gp, oq3GoldenFile{Scenario: r.sc.ID, Step: i, Index: k, Label: label, Trial: st.Trial, Kind: "summariser"}, obs)
			if gerr != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"no-golden"}}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: no summariser golden %s", label, gp))
				aux(d)
				continue
			}
			d.Verdict = oq3Unchanged(obs, golden)
			if v := oq3CheckSummariser(obs); !v.Correct {
				d.Verdict.Correct = false
				d.Verdict.Categories = append(d.Verdict.Categories, v.Categories...)
			}
			d.Status = map[bool]string{true: "correct", false: "bad"}[d.Verdict.Correct]
			aux(d)
		}
		// Preloads: each equal to its frozen pre-G1 golden (review finding 4).
		for k, c := range byKey[key{label, "/api/generate"}] {
			d := oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Index: k, Seq: c.Seq, Kind: "preload"}
			obs, err := oq3Normalise(c.Body, r.s.root)
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"malformed-capture"}}
				aux(d)
				continue
			}
			gp := filepath.Join(goldenDir, r.sc.ID, fmt.Sprintf("%02d-g%d.json", i, k))
			d.Golden = gp
			golden, gerr := loadOrFreeze(gp, oq3GoldenFile{Scenario: r.sc.ID, Step: i, Index: k, Label: label, Trial: st.Trial, Kind: "preload"}, obs)
			if gerr != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"no-golden"}}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: no preload golden %s", label, gp))
				aux(d)
				continue
			}
			d.Verdict = oq3Unchanged(obs, golden)
			d.Status = map[bool]string{true: "correct", false: "bad"}[d.Verdict.Correct]
			aux(d)
		}
		switch st.Class {
		case "required-no-dispatch":
			// A refusal is required: any model request under this step is an incorrect delivery.
			if len(turns) == 0 {
				res.Deliveries = append(res.Deliveries, oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial,
					Status: "refused-as-required", Verdict: oq3DeliveryVerdict{Correct: true}})
			}
			for k, c := range turns {
				scoredSeq[c.Seq] = true
				res.Deliveries = append(res.Deliveries, oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Index: k, Seq: c.Seq,
					Status: "bad", Verdict: oq3DeliveryVerdict{Categories: []string{"dispatched-despite-required-refusal"}}})
			}
			continue
		case "required-dispatch":
		default:
			if len(turns) > 0 {
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: %d unclassified model requests under a non-dispatch step", label, len(turns)))
			}
			continue
		}
		for k := 0; k < len(st.Expect) || k < len(turns); k++ {
			d := oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Index: k}
			if k < len(st.Expect) {
				e := st.Expect[k]
				d.Expect = &e
			}
			if k >= len(turns) {
				d.Status = "missing"
				d.Verdict = oq3DeliveryVerdict{Categories: []string{"not-dispatched"}}
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			c := turns[k]
			d.Seq = c.Seq
			scoredSeq[c.Seq] = true
			obs, err := oq3Normalise(c.Body, r.s.root)
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"malformed-capture"}, Detail: err.Error()}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: malformed capture %d", label, c.Seq))
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			if d.Expect == nil {
				d.Status = "unexpected"
				d.Verdict = oq3DeliveryVerdict{Categories: []string{"unexpected-request"}}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: more model requests than declared (%d > %d)", label, len(turns), len(st.Expect)))
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			gp := oq3GoldenPath(goldenDir, r.sc.ID, i, k)
			d.Golden = gp
			golden, gerr := loadOrFreeze(gp, oq3GoldenFile{Scenario: r.sc.ID, Step: i, Index: k, Label: label, Trial: st.Trial, Expect: *d.Expect, Kind: "delivery"}, obs)
			if gerr != nil {
				d.Status = "bad"
				d.Verdict = oq3DeliveryVerdict{Categories: []string{"no-golden"}, Detail: gerr.Error()}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: no golden %s", label, gp))
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			if k == 0 {
				// A state-changing step must have reached its declared state in the pre-G1 request
				// (review finding 5); checked on the golden, which is what every later run is held to.
				if problem := oq3AssertState(golden, st.Args); problem != "" {
					res.Invalid = append(res.Invalid, fmt.Sprintf("%s: declared state not reached: %s", label, problem))
				}
			}
			exp, terr := oq3Transform(golden, *d.Expect, func(model string) (string, bool) {
				s, ok := r.d.modelSystems[oq3ModelBase(model)]
				return s, ok
			})
			if terr != nil {
				d.Status = "bad"
				d.Verdict = oq3DeliveryVerdict{Categories: []string{"oracle-unresolved"}, Detail: terr.Error()}
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: %v", label, terr))
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			d.Verdict = oq3Score(obs, exp, *d.Expect)
			d.Status = map[bool]string{true: "correct", false: "bad"}[d.Verdict.Correct]
			res.Deliveries = append(res.Deliveries, d)
		}
	}
	// Unknown labels, and a leak scan over every other capture on every path (review finding 4).
	for _, c := range caps {
		if !known[c.Label] {
			res.Invalid = append(res.Invalid, fmt.Sprintf("request %d (%s %s) under unknown label %q", c.Seq, c.Method, c.Path, c.Label))
			continue
		}
		if scoredSeq[c.Seq] || (c.Path == "/api/chat" && c.Response == "summary") || c.Path == "/api/generate" {
			continue
		}
		raw := string(c.Body) + c.RawBody
		leaked := strings.Contains(raw, "User-configured instructions")
		for _, s := range oq3AllSentinels {
			leaked = leaked || strings.Contains(raw, s)
		}
		if leaked {
			aux(oq3Delivery{Scenario: r.sc.ID, Label: c.Label, Seq: c.Seq, Kind: "leak-scan", Status: "bad",
				Verdict: oq3DeliveryVerdict{Categories: []string{"carries-instructions"}, Detail: c.Method + " " + c.Path}})
		}
	}
	// Compaction must be demonstrable at the request boundary: the first request after a compaction
	// step carries the summariser's text in its history.
	for i, st := range r.sc.Steps {
		if st.Summarisers == 0 || st.Op != "terminal-slash" || notRun[i] {
			continue
		}
		for j := i + 1; j < len(r.sc.Steps); j++ {
			if r.sc.Steps[j].Class != "required-dispatch" {
				continue
			}
			nl := oq3Label(r.sc.ID, j, r.sc.Steps[j].Op)
			cs := byKey[key{nl, "/api/chat"}]
			if len(cs) == 0 || !strings.Contains(string(cs[0].Body), "OQ3-SUMMARY") {
				res.Invalid = append(res.Invalid, fmt.Sprintf("%s: the next request (%s) does not carry the compacted history", oq3Label(r.sc.ID, i, st.Op), nl))
			}
			break
		}
	}
	if len(res.NotMeasured) > 0 {
		res.Invalid = append(res.Invalid, fmt.Sprintf("%d steps not measured after a harness failure", len(res.NotMeasured)))
	}
	return res
}

// oq3AssertState checks a golden (a pre-G1 request) against the state its step declares:
// assert_tools ("0" or "some"), assert_model, assert_system ("absent" or "present").
func oq3AssertState(golden any, args map[string]string) string {
	m, _ := golden.(map[string]any)
	if want := args["assert_tools"]; want != "" {
		tools, _ := m["tools"].([]any)
		if (want == "0") != (len(tools) == 0) {
			return fmt.Sprintf("tools: want %s, request has %d", want, len(tools))
		}
	}
	if want := args["assert_model"]; want != "" {
		if got, _ := m["model"].(string); oq3ModelBase(got) != want {
			return fmt.Sprintf("model: want %s, request has %s", want, got)
		}
	}
	if want := args["assert_system"]; want != "" {
		msgs := oq3Messages(golden)
		has := len(msgs) > 0 && oq3Str(msgs[0], "role") == "system"
		if (want == "present") != has {
			return fmt.Sprintf("system message: want %s", want)
		}
	}
	return ""
}

// scoreDeliveriesOnly scores the requests captured under each dispatch step against that step's
// expectations and the frozen goldens, without freezing and without the omission, summariser and
// compaction checks. The reference arm uses it: it only re-sends requests that have a golden.
func (r *oq3Run) scoreDeliveriesOnly(goldenDir string) oq3ScenarioResult {
	res := oq3ScenarioResult{Scenario: r.sc, Sandbox: r.s.root}
	caps := r.d.snapshot()
	res.Captures = caps
	for i, st := range r.sc.Steps {
		if st.Class != "required-dispatch" {
			continue
		}
		label := oq3Label(r.sc.ID, i, st.Op)
		k := 0
		for _, c := range caps {
			if c.Path != "/api/chat" || c.Label != label {
				continue
			}
			d := oq3Delivery{Scenario: r.sc.ID, Step: i, Label: label, Trial: st.Trial, Index: k, Seq: c.Seq}
			if k >= len(st.Expect) {
				d.Status, d.Verdict = "unexpected", oq3DeliveryVerdict{Categories: []string{"unexpected-request"}}
				res.Deliveries = append(res.Deliveries, d)
				k++
				continue
			}
			e := st.Expect[k]
			d.Expect = &e
			gp := oq3GoldenPath(goldenDir, r.sc.ID, i, k)
			d.Golden = gp
			k++
			obs, err := oq3Normalise(c.Body, r.s.root)
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"malformed-capture"}}
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			golden, err := oq3LoadGolden(gp)
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"no-golden"}}
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			exp, err := oq3Transform(golden, e, func(model string) (string, bool) {
				s, ok := r.d.modelSystems[oq3ModelBase(model)]
				return s, ok
			})
			if err != nil {
				d.Status, d.Verdict = "bad", oq3DeliveryVerdict{Categories: []string{"oracle-unresolved"}}
				res.Deliveries = append(res.Deliveries, d)
				continue
			}
			d.Verdict = oq3Score(obs, exp, e)
			d.Status = map[bool]string{true: "correct", false: "bad"}[d.Verdict.Correct]
			res.Deliveries = append(res.Deliveries, d)
		}
	}
	return res
}

func oq3MarshalCanonical(v any) ([]byte, error) {
	return json.Marshal(v)
}

func oq3LoadGolden(path string) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var gf struct {
		Request json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(b, &gf); err != nil {
		return nil, err
	}
	return oq3Normalise(gf.Request, "")
}

func oq3WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// ---------------------------------------------------------------------------------------------
// Summary and the number.

type oq3Summary struct {
	Mode                  string         `json:"mode"`
	Source                string         `json:"source"`
	Trials                int            `json:"controlled_conversation_trials"`
	ObservedDeliveries    int            `json:"observed_deliveries"`
	BadObserved           int            `json:"bad_observed_deliveries"`
	MissingRequired       int            `json:"missing_required_deliveries"`
	Unexpected            int            `json:"unexpected_requests"`
	FailureNumber         string         `json:"failure_number"`
	FailureNumerator      int            `json:"failure_numerator"`
	CategoryCounts        map[string]int `json:"category_counts_overlapping"`
	PerFamily             map[string]any `json:"per_family"`
	PerTrial              map[string]any `json:"per_trial"`
	OperationVerdicts     map[string]int `json:"operation_verdicts"`
	RefusalVerdicts       map[string]int `json:"required_refusal_verdicts"`
	AuxiliaryChecks       map[string]int `json:"auxiliary_checks"`
	AuxiliaryBad          int            `json:"auxiliary_bad"`
	AuxiliaryBadList      []string       `json:"auxiliary_bad_list,omitempty"`
	RefusalsHeld          int            `json:"required_refusals_held"`
	NotMeasured           []string       `json:"not_measured,omitempty"`
	Valid                 bool           `json:"valid"`
	Invalid               []string       `json:"invalid,omitempty"`
	ScenariosRun          []string       `json:"scenarios_run"`
	LedgerSHA256          string         `json:"ledger_sha256"`
}

func oq3Summarise(results []oq3ScenarioResult, mode, source, ledgerSHA string) oq3Summary {
	s := oq3Summary{Mode: mode, Source: source, CategoryCounts: map[string]int{}, PerFamily: map[string]any{}, PerTrial: map[string]any{},
		OperationVerdicts: map[string]int{}, RefusalVerdicts: map[string]int{}, AuxiliaryChecks: map[string]int{}, LedgerSHA256: ledgerSHA, Valid: true}
	family := map[string]string{}
	type tally struct{ Observed, Bad, Missing, Correct int }
	trialT := map[string]*tally{}
	for _, r := range results {
		s.ScenariosRun = append(s.ScenariosRun, r.Scenario.ID)
		for _, tr := range r.Scenario.Trials {
			family[tr.ID] = tr.Family
			trialT[tr.ID] = &tally{}
			s.Trials++
		}
		for _, d := range r.Deliveries {
			t := trialT[d.Trial]
			switch d.Status {
			case "correct":
				s.ObservedDeliveries++
				t.Observed++
				t.Correct++
			case "bad":
				s.ObservedDeliveries++
				s.BadObserved++
				t.Observed++
				t.Bad++
			case "missing":
				s.MissingRequired++
				t.Missing++
			case "unexpected":
				s.ObservedDeliveries++
				s.Unexpected++
				s.BadObserved++
				t.Observed++
				t.Bad++
			case "refused-as-required":
				s.RefusalsHeld++
			}
			for _, c := range d.Verdict.Categories {
				s.CategoryCounts[c]++
			}
		}
		for _, a := range r.Auxiliary {
			s.AuxiliaryChecks[a.Kind+":"+a.Status]++
			if a.Status != "correct" {
				s.AuxiliaryBad++
				s.AuxiliaryBadList = append(s.AuxiliaryBadList, fmt.Sprintf("%s %s %v", a.Label, a.Kind, a.Verdict.Categories))
			}
		}
		s.NotMeasured = append(s.NotMeasured, r.NotMeasured...)
		for _, e := range r.Events {
			switch e.Class {
			case "operation":
				s.OperationVerdicts[e.Op+":"+e.Verdict]++
			case "required-refusal":
				s.RefusalVerdicts[e.Op+":"+e.Verdict]++
			}
		}
		if len(r.Invalid) > 0 {
			s.Valid = false
			s.Invalid = append(s.Invalid, r.Invalid...)
		}
	}
	fam := map[string]*tally{}
	for id, t := range trialT {
		f := family[id]
		if fam[f] == nil {
			fam[f] = &tally{}
		}
		fam[f].Observed += t.Observed
		fam[f].Bad += t.Bad
		fam[f].Missing += t.Missing
		fam[f].Correct += t.Correct
		s.PerTrial[id] = map[string]any{"family": f, "observed": t.Observed, "correct": t.Correct, "bad": t.Bad, "missing": t.Missing}
	}
	famTrials := map[string]int{}
	for _, f := range family {
		famTrials[f]++
	}
	for f, t := range fam {
		s.PerFamily[f] = map[string]any{"trials": famTrials[f], "observed": t.Observed, "correct": t.Correct, "bad": t.Bad, "missing": t.Missing,
			"failure_number": oq3Ratio(t.Bad+t.Missing, famTrials[f])}
	}
	s.FailureNumerator = s.BadObserved + s.MissingRequired
	s.FailureNumber = oq3Ratio(s.FailureNumerator, s.Trials)
	return s
}

// oq3Ratio prints an exact fraction and a decimal; a zero denominator is refused, never printed as zero.
func oq3Ratio(num, den int) string {
	if den == 0 {
		return "undefined (no trials)"
	}
	return fmt.Sprintf("%d/%d = %.4f", num, den, float64(num)/float64(den))
}

func oq3LedgerSHA(scs []oq3Scenario) (string, []byte) {
	b, _ := json.MarshalIndent(scs, "", "  ")
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), b
}

func oq3SortedKeys(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
