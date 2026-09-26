//go:build darwin

// OQ-3 oracle: what a correct delivery looks like, derived independently of the code under test.
//
// The expected request for every delivery is built from two inputs only: the request the unchanged
// (pre-G1) candidate sent for the same step, frozen as a golden, and the composition rule of the G1
// blueprint §5 (docs/_design/G1_PERSISTED_INSTRUCTIONS.md:200-206) implemented here from its text.
// No production composition code is called. The admission field is §6.3 A1 (version 1).

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// oq3Unchanged is the expectation for auxiliary model traffic that G1 must leave exactly as it was
// (a compaction summariser request, a preload): equal to its frozen pre-G1 golden. When it is not,
// it names whether instructions leaked into it.
func oq3Unchanged(observed, golden any) oq3DeliveryVerdict {
	if reflect.DeepEqual(observed, golden) {
		return oq3DeliveryVerdict{Correct: true}
	}
	var cats []string
	b, _ := json.Marshal(observed)
	if bytes.Contains(b, []byte("User-configured instructions")) {
		cats = append(cats, "carries-header")
	}
	for _, s := range oq3AllSentinels {
		if bytes.Contains(b, []byte(s)) {
			cats = append(cats, "carries-profile")
			break
		}
	}
	cats = append(cats, "request-changed")
	return oq3DeliveryVerdict{Categories: cats, Detail: oq3FirstDifference(observed, golden)}
}

// oq3Texts are the contrasting instruction profiles. Each carries sentinels at its start, middle and
// end, a line break and a non-ASCII character, so truncation, re-encoding or partial loss is visible.
var oq3Texts = map[string]string{
	"A": "SENTINEL-A-START-5d2c OQ3 instruction profile A: answer in Australian English and keep replies short.\n" +
		"SENTINEL-A-MID-9e41 Name the café you would recommend — this line checks bytes beyond ASCII.\n" +
		"Close every reply with the word ALPHA. SENTINEL-A-END-b7f0",
	"B": "SENTINEL-B-START-2a6e OQ3 instruction profile B: answer as a checklist of numbered steps.\n" +
		"SENTINEL-B-MID-4c18 Prefer metric units — naïve estimates are acceptable here.\n" +
		"Close every reply with the word BRAVO. SENTINEL-B-END-83d9",
}

// oq3AllSentinels lists every profile sentinel; none may appear anywhere a carrier is not expected.
var oq3AllSentinels = []string{"SENTINEL-A-START-5d2c", "SENTINEL-A-MID-9e41", "SENTINEL-A-END-b7f0",
	"SENTINEL-B-START-2a6e", "SENTINEL-B-MID-4c18", "SENTINEL-B-END-83d9"}

const oq3HeaderPrefix = "User-configured instructions; revision "

func oq3Header(rev, gen int) string {
	return fmt.Sprintf("User-configured instructions; revision %d; selection %d:\n", rev, gen)
}

var oq3HeaderRE = regexp.MustCompile(`User-configured instructions; revision ([0-9]+); selection ([0-9]+):\n`)

// oq3Expect is one required delivery's expectation.
type oq3Expect struct {
	Carrier bool   `json:"carrier"`
	Rev     int    `json:"revision,omitempty"`
	Gen     int    `json:"generation,omitempty"`
	Text    string `json:"text,omitempty"` // key into oq3Texts
	// Adapter decides where the base comes from: "desktop" (model-system text of the request's model,
	// §5) or "terminal" (the system message the pre-G1 candidate sent, which is the built-in base).
	Adapter string `json:"adapter"`
}

// ---------------------------------------------------------------------------------------------
// Normalisation: only declared variable inputs are parameterised (Method 04 T-07).

var (
	oq3DateRE = regexp.MustCompile(`Current date: [A-Z][a-z]+, [A-Z][a-z]+ [0-9]{1,2}, [0-9]{4}\.`)
)

// oq3Normalise decodes a request body and replaces declared variable text: the terminal prompt's
// date line and the sandbox root path. Numbers stay exact (json.Number).
func oq3Normalise(body []byte, sandboxRoot string) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("more than one JSON value")
	}
	// The sandbox is spelt two ways on macOS (/var/... and its resolved /private/var/...); the
	// resolved spelling is replaced first so no machine-dependent prefix survives (review finding 7).
	var roots []string
	if sandboxRoot != "" {
		if resolved, err := filepath.EvalSymlinks(sandboxRoot); err == nil && resolved != sandboxRoot {
			roots = append(roots, resolved)
		}
		roots = append(roots, sandboxRoot)
	}
	return oq3Walk(v, func(s string) string {
		s = oq3DateRE.ReplaceAllString(s, "Current date: <DATE>.")
		for _, r := range roots {
			s = strings.ReplaceAll(s, r, "<SANDBOX>")
		}
		return s
	}), nil
}

// oq3AdmitReserve accepts an admission object that §6.3 A1-A2 permit for a local request: version
// 1 and, optionally, a reserve that is non-negative and at most half the effective context (the
// fixture's context is 32768). It returns false for anything else (review finding 6).
func oq3AdmitReserve(adm any) bool {
	m, ok := adm.(map[string]any)
	if !ok {
		return false
	}
	for k := range m {
		if k != "version" && k != "reserve" {
			return false
		}
	}
	if v, ok := m["version"].(json.Number); !ok || v.String() != "1" {
		return false
	}
	if r, ok := m["reserve"]; ok {
		n, isNum := r.(json.Number)
		if !isNum {
			return false
		}
		iv, err := n.Int64()
		if err != nil || iv < 0 || iv > 16384 {
			return false
		}
	}
	return true
}

func oq3Walk(v any, f func(string) string) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = oq3Walk(val, f)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = oq3Walk(val, f)
		}
		return out
	case string:
		return f(x)
	default:
		return v
	}
}

func oq3DeepCopy(v any) any {
	return oq3Walk(v, func(s string) string { return s })
}

func oq3Messages(req any) []map[string]any {
	m, _ := req.(map[string]any)
	raw, _ := m["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		if mm, ok := r.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

func oq3Str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

// ---------------------------------------------------------------------------------------------
// The expected request.

// oq3Transform returns the request a correct G1 implementation sends for this delivery, given the
// normalised pre-G1 golden for the same step and the expectation. modelSystem is the model-system
// text the fake daemon serves for the request's model (desktop base, §5).
func oq3Transform(golden any, e oq3Expect, modelSystem func(model string) (string, bool)) (any, error) {
	exp := oq3DeepCopy(golden)
	m, ok := exp.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("golden is not an object")
	}
	if !e.Carrier {
		delete(m, "admission")
		return m, nil
	}
	text, ok := oq3Texts[e.Text]
	if !ok {
		return nil, fmt.Errorf("unknown text %q", e.Text)
	}
	carrier := oq3Header(e.Rev, e.Gen) + text
	msgs, _ := m["messages"].([]any)
	switch e.Adapter {
	case "desktop":
		if len(msgs) > 0 {
			if first, ok := msgs[0].(map[string]any); ok && oq3Str(first, "role") == "system" {
				// A pre-G1 desktop request only leads with a system message when MCP servers supplied
				// instructions; the blueprint does not state how model-system and MCP text are joined
				// (G1_PERSISTED_INSTRUCTIONS.md:204), so this oracle refuses to guess.
				return nil, fmt.Errorf("oracle unresolved: desktop golden already leads with a system message (MCP join unspecified)")
			}
		}
		model, _ := m["model"].(string)
		base, ok := modelSystem(model)
		if !ok {
			return nil, fmt.Errorf("no model-system text for %q", model)
		}
		content := carrier
		if base != "" {
			content = base + "\n\n" + carrier
		}
		m["messages"] = append([]any{map[string]any{"role": "system", "content": content}}, msgs...)
	case "terminal":
		if len(msgs) > 0 {
			if first, ok := msgs[0].(map[string]any); ok && oq3Str(first, "role") == "system" {
				base := oq3Str(first, "content")
				nf := oq3DeepCopy(first).(map[string]any)
				if base != "" {
					nf["content"] = base + "\n\n" + carrier
				} else {
					nf["content"] = carrier
				}
				msgs[0] = nf
				m["messages"] = msgs
				break
			}
		}
		// No built-in base (for example after /system off): the carrier alone leads (§5).
		m["messages"] = append([]any{map[string]any{"role": "system", "content": carrier}}, msgs...)
	default:
		return nil, fmt.Errorf("unknown adapter %q", e.Adapter)
	}
	m["admission"] = map[string]any{"version": json.Number("1")}
	return m, nil
}

// ---------------------------------------------------------------------------------------------
// Scoring one observed delivery.

type oq3DeliveryVerdict struct {
	Correct    bool     `json:"correct"`
	Categories []string `json:"categories,omitempty"`
	Detail     string   `json:"detail,omitempty"`
}

// oq3Score compares one observed request (normalised) with the expected request, and when they
// differ, names every category of difference it can identify. An observed request that differs is
// one bad delivery, however many categories it has.
func oq3Score(observed, expected any, e oq3Expect) oq3DeliveryVerdict {
	cats := map[string]bool{}
	// A permitted reserve in the observed admission object is not a difference (finding 6); an
	// admission object outside A1-A2 is its own category.
	if om, ok := observed.(map[string]any); ok {
		if em, ok := expected.(map[string]any); ok {
			if oa, has := om["admission"]; has {
				if _, want := em["admission"]; want {
					if oq3AdmitReserve(oa) {
						em = oq3DeepCopy(em).(map[string]any)
						em["admission"] = oq3DeepCopy(oa)
						expected = em
					} else {
						cats["admission-invalid"] = true
					}
				}
			}
		}
	}
	if len(cats) == 0 && reflect.DeepEqual(observed, expected) {
		return oq3DeliveryVerdict{Correct: true}
	}
	var notes []string
	obsMsgs := oq3Messages(observed)
	expMsgs := oq3Messages(expected)

	// Where do carrier headers occur in the observed request?
	type hit struct {
		msg, rev, gen int
		role          string
	}
	var hits []hit
	for i, msg := range obsMsgs {
		for _, mm := range oq3HeaderRE.FindAllStringSubmatch(oq3Str(msg, "content"), -1) {
			var r, g int
			fmt.Sscan(mm[1], &r)
			fmt.Sscan(mm[2], &g)
			hits = append(hits, hit{msg: i, rev: r, gen: g, role: oq3Str(msg, "role")})
		}
	}
	obsJSON, _ := json.Marshal(observed)
	om, _ := observed.(map[string]any)
	_, hasAdmission := om["admission"]

	if e.Carrier {
		text := oq3Texts[e.Text]
		switch {
		case len(hits) == 0:
			cats["missing"] = true
		case len(hits) > 1:
			cats["duplicated"] = true
		default:
			h := hits[0]
			if h.rev != e.Rev || h.gen != e.Gen {
				cats["wrong-revision"] = true
				notes = append(notes, fmt.Sprintf("header revision %d selection %d, expected %d/%d", h.rev, h.gen, e.Rev, e.Gen))
			}
			if h.msg != 0 || h.role != "system" {
				cats["misplaced"] = true
			}
			content := oq3Str(obsMsgs[h.msg], "content")
			idx := strings.Index(content, oq3HeaderPrefix)
			if idx < 0 {
				cats["other"] = true
				break
			}
			after := content[idx:]
			if loc := oq3HeaderRE.FindStringIndex(after); loc != nil {
				after = after[loc[1]:]
			}
			if after != text {
				cats["altered"] = true
			}
			if len(expMsgs) > 0 {
				expContent := oq3Str(expMsgs[0], "content")
				if ei := strings.Index(expContent, oq3HeaderPrefix); ei >= 0 && content[:idx] != expContent[:ei] {
					cats["base-lost"] = true
				}
			}
		}
		if !hasAdmission {
			cats["admission-missing"] = true
		}
	} else {
		if len(hits) > 0 {
			cats["unexpected-carrier"] = true
		}
		if hasAdmission {
			cats["unexpected-admission"] = true
		}
	}
	// Profile text anywhere the expectation does not put it.
	expJSON, _ := json.Marshal(expected)
	for _, s := range oq3AllSentinels {
		if strings.Count(string(obsJSON), s) > strings.Count(string(expJSON), s) {
			cats["leaked"] = true
			notes = append(notes, "sentinel "+s)
		}
	}
	// Everything outside the leading message must equal the expectation (history, tools, options).
	obsRest, expRest := oq3WithoutLeadingSystem(observed), oq3WithoutLeadingSystem(expected)
	if !reflect.DeepEqual(obsRest, expRest) {
		cats["request-changed"] = true
		notes = append(notes, oq3FirstDifference(obsRest, expRest))
	}
	if len(cats) == 0 {
		// Differs, but in a way none of the named checks isolates (for example the leading
		// message's role or an extra field on it).
		cats["other"] = true
	}
	out := make([]string, 0, len(cats))
	for c := range cats {
		out = append(out, c)
	}
	sort.Strings(out)
	return oq3DeliveryVerdict{Correct: false, Categories: out, Detail: strings.Join(notes, "; ")}
}

func oq3WithoutLeadingSystem(req any) any {
	c := oq3DeepCopy(req)
	m, ok := c.(map[string]any)
	if !ok {
		return c
	}
	delete(m, "admission")
	msgs, _ := m["messages"].([]any)
	if len(msgs) > 0 {
		if first, ok := msgs[0].(map[string]any); ok && oq3Str(first, "role") == "system" {
			msgs = msgs[1:]
		}
	}
	m["messages"] = msgs
	return m
}

// oq3FirstDifference names the first top-level field (or message index) that differs.
func oq3FirstDifference(a, b any) string {
	am, _ := a.(map[string]any)
	bm, _ := b.(map[string]any)
	keys := map[string]bool{}
	for k := range am {
		keys[k] = true
	}
	for k := range bm {
		keys[k] = true
	}
	var ks []string
	for k := range keys {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		if !reflect.DeepEqual(am[k], bm[k]) {
			if k == "messages" {
				amsg, _ := am[k].([]any)
				bmsg, _ := bm[k].([]any)
				if len(amsg) != len(bmsg) {
					return fmt.Sprintf("messages: %d observed, %d expected", len(amsg), len(bmsg))
				}
				for i := range amsg {
					if !reflect.DeepEqual(amsg[i], bmsg[i]) {
						return fmt.Sprintf("messages[%d] differs", i)
					}
				}
			}
			return "field " + k + " differs"
		}
	}
	return "no top-level difference found"
}

// oq3CheckSummariser is the separate expectation for a compaction summariser request (Q2): it keeps
// the compactor's own system prompt, and carries no carrier header and no profile text, so the
// carrier was neither injected into it nor stored in the history it summarises.
func oq3CheckSummariser(observed any) oq3DeliveryVerdict {
	msgs := oq3Messages(observed)
	var cats []string
	if len(msgs) == 0 || oq3Str(msgs[0], "role") != "system" || !strings.HasPrefix(oq3Str(msgs[0], "content"), oq3CompactionSystemPrefix) {
		cats = append(cats, "summariser-prompt-replaced")
	}
	b, _ := json.Marshal(observed)
	if oq3HeaderRE.Match(b) || bytes.Contains(b, []byte("User-configured instructions")) {
		cats = append(cats, "summariser-carries-header")
	}
	for _, s := range oq3AllSentinels {
		if bytes.Contains(b, []byte(s)) {
			cats = append(cats, "summariser-carries-profile")
			break
		}
	}
	return oq3DeliveryVerdict{Correct: len(cats) == 0, Categories: cats}
}
