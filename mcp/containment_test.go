package mcp

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// sdkModule is the protocol library this package is built on.
const sdkModule = "github.com/modelcontextprotocol/go-sdk"

// ownPackage is the only package in this module permitted to import it.
const ownPackage = "github.com/ollama/ollama/mcp"

// TestSDKIsContainedToThisPackage is the guard that keeps the choice of
// protocol library reversible.
//
// The operator ruled that Ollama uses the official Go SDK and that this work is
// aimed at an upstream pull request. Those pull in opposite directions: a
// substantial new dependency is the likeliest reason such a request is refused.
// The agreement was that the risk would be contained rather than argued about —
// no SDK type appears outside this package, so every caller sees only
// mcp.ServerSpec, mcp.Tool, mcp.Manager and api.ToolFunction.
//
// The value of that containment is that swapping the SDK for a hand-written or
// vendored implementation later is a change to one package with no call-site
// churn. The moment another package imports the SDK directly, that stops being
// true, and it stops being true quietly. Hence this test.
func TestSDKIsContainedToThisPackage(t *testing.T) {
	list := exec.Command("go", "list", "-json", "github.com/ollama/ollama/...")
	output, err := list.Output()
	if err != nil {
		var stderr string
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = string(exit.Stderr)
		}
		t.Fatalf("go list: %v\n%s", err, stderr)
	}

	type pkg struct {
		ImportPath   string
		Imports      []string
		TestImports  []string
		XTestImports []string
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	var offenders []string
	var inspected int

	for decoder.More() {
		var p pkg
		if err := decoder.Decode(&p); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		inspected++
		if p.ImportPath == ownPackage {
			continue
		}
		for _, group := range [][]string{p.Imports, p.TestImports, p.XTestImports} {
			for _, imported := range group {
				if strings.HasPrefix(imported, sdkModule) {
					offenders = append(offenders, p.ImportPath+" imports "+imported)
				}
			}
		}
	}

	if inspected < 50 {
		t.Fatalf("go list reported only %d packages; the check did not cover the module", inspected)
	}
	if len(offenders) > 0 {
		t.Errorf("the MCP protocol library must not be imported outside %s, so that swapping it stays a one-package change. Found:\n  %s",
			ownPackage, strings.Join(offenders, "\n  "))
	}
}
