//go:build darwin

package cmd

import (
	"strings"
	"testing"
	"time"
)

// TestOQ3HarnessSmoke proves the plumbing only: one real terminal session and one real desktop
// process each reach the fake daemon with an actual /api/chat request. It scores nothing.
func TestOQ3HarnessSmoke(t *testing.T) {
	d := oq3StartDaemon(t)

	s := oq3NewSandbox(t, "smoke-terminal")
	d.setLabel("smoke/terminal/start")
	term := oq3StartTerminal(t, s, d, "smoke")
	defer term.kill()
	if !oq3WaitFor(30*time.Second, func() bool { return d.count("POST", "/api/generate") >= 1 }) {
		t.Fatalf("terminal never preloaded; screen=%q", tail(term.screen(), 2000))
	}
	term.quiet(700*time.Millisecond, 15*time.Second)
	d.setLabel("smoke/terminal/turn1")
	before := d.count("POST", "/api/chat")
	mark := term.rawLen()
	term.typeLine("hello from the oq3 smoke test")
	if !oq3WaitFor(20*time.Second, func() bool { return d.count("POST", "/api/chat") > before }) {
		t.Fatalf("terminal turn never reached the daemon; screen=%q", tail(term.screen(), 3000))
	}
	if !oq3WaitFor(20*time.Second, func() bool { return term.screenContainsAfter(mark, "OQ3-REPLY-") }) {
		t.Fatalf("terminal never rendered the reply; screen=%q", tail(term.screen(), 3000))
	}
	term.quiet(500*time.Millisecond, 10*time.Second)
	term.typeLine("/bye")
	if err, ok := term.waitExit(20 * time.Second); !ok {
		t.Fatalf("terminal did not exit on /bye; screen=%q", tail(term.screen(), 2000))
	} else if err != nil {
		t.Fatalf("terminal exit: %v; screen=%q", err, tail(term.screen(), 2000))
	}

	ds := oq3NewSandbox(t, "smoke-desktop")
	d.setLabel("smoke/desktop/turn1")
	desk := oq3StartDesktop(t, ds, d, 1)
	before = d.count("POST", "/api/chat")
	id, res := desk.chat("new", "hello from the oq3 desktop smoke test")
	if res.Status != 200 || id == "new" || id == "" {
		t.Fatalf("desktop chat status=%d id=%q body=%s", res.Status, id, tail(res.Body, 2000))
	}
	if d.count("POST", "/api/chat") != before+1 {
		t.Fatalf("desktop turn did not reach the daemon exactly once")
	}
	desk.stop()

	for _, c := range d.snapshot() {
		t.Logf("capture %d %s %s label=%s kind=%s body=%s", c.Seq, c.Method, c.Path, c.Label, c.Response, tail(string(c.Body), 600))
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

var _ = strings.TrimSpace
