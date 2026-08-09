Claude auto-install executes remote shell script
Link: https://chatgpt.com/codex/cloud/security/findings/e0b5b447db208191af8d9f0d62a2ed14?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: 9c02d8e
Author: 63033505+hoyyeva@users.noreply.github.com
Created: 9/8/2026, 1:50:10 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced: Claude Code was changed from only locating an already-installed binary to automatically installing it by executing a remotely fetched installer script.
When Claude Code is missing, `ollama launch claude` now offers to install it and, after confirmation or when confirmation is auto-approved via launch policy, runs a platform-specific shell command that pipes a remote HTTPS installer directly into `bash` or PowerShell `iex`. There is no checksum, signature verification, pinned release artifact, or post-download validation before executing the retrieved code. An attacker able to compromise the installer endpoint, its delivery infrastructure, or a trusted TLS/network interception path could execute arbitrary code as the local user running Ollama. The command string itself is not attacker-interpolated, and this is a CLI/operator-mediated flow, but the commit introduces a new remote-code execution supply-chain trust path for the Claude integration.

# Validation
## Rubric
- [x] Confirm the commit introduced a new path from `Claude.Run()` to auto-install logic instead of only locating an existing binary.
- [x] Confirm the missing-Claude path executes an installer command after confirmation and that confirmation can be auto-approved by launch policy/`--yes`.
- [x] Confirm the installer command executes remote HTTPS content directly through `bash` or PowerShell `iex` without checksum/signature/pinning/post-download validation.
- [x] Confirm Claude is registered as auto-installable, making the path reachable from integration install/launch flows.
- [x] Dynamically reproduce the attacker model by simulating compromised installer content and showing that the production `ensureClaudeInstalled()` path executes that content; collect debugger evidence of the call chain.
## Report
Validated the finding at commit 9c02d8e69dc49f80edfd910d39478e8622c578e0. Code review showed `Claude.Run()` now calls `ensureClaudeInstalled()` before executing Claude (`cmd/launch/claude.go:52-58`). If Claude is missing, `ensureClaudeInstalled()` prompts, obtains `claudeInstallerCommand(runtime.GOOS)`, and runs it via `exec.Command` with user stdio attached (`cmd/launch/claude.go:76-104`). The platform commands are direct remote-script execution: Windows uses `irm https://claude.ai/install.ps1 | iex`, while Linux/macOS use `bash -c 'curl -fsSL https://claude.ai/install.sh | bash'` (`cmd/launch/claude.go:137-151`). Claude is also registered as auto-installable through `EnsureInstalled` (`cmd/launch/registry.go:43-51`). Confirmation can be auto-approved by launch policy: `ConfirmPromptWithOptions` returns true when `currentLaunchConfirmPolicy.yes` is set (`cmd/launch/selector_hooks.go:95-98`), and `--yes` maps to auto-approve in `defaultLaunchPolicy` (`cmd/launch/launch.go:84-106`).

I created a targeted package-level Go PoC test that simulates a compromised installer endpoint by placing a fake `curl` first in PATH. The fake `curl` outputs attacker-controlled shell code; the real `ensureClaudeInstalled()` then runs the production command `curl -fsSL https://claude.ai/install.sh | bash`, causing the payload to execute, write a marker file, and install a fake Claude binary. Normal run output: `=== RUN TestClaudeInstallerExecutesFetchedScriptPayload`, `Installing Claude Code...`, `Claude Code installed successfully`, `--- PASS`. I also built a debug test binary with `-gcflags=all='-N -l'` and used LLDB non-interactively. LLDB breakpoints confirmed the chain: `TestClaudeInstallerExecutesFetchedScriptPayload -> ensureClaudeInstalled at claude.go:76`, then `ensureClaudeInstalled at claude.go:93 -> claudeInstallerCommand at claude.go:137`, after which the installer completed successfully. Valgrind was unavailable; ASan-instrumented `go test -asan` also reproduced the same payload-execution path, though this is not a memory-safety issue. Direct crash was attempted/considered via the debug test binary; no crash is expected because the bug is intentional execution of unverified remote installer content, not memory corruption.

Conclusion: the suspected supply-chain RCE path is valid. The command string is static and user/CLI mediated, but the implementation executes network-fetched script content without checksum, signature, pinned artifact, or independent validation before execution.

# Evidence
cmd/launch/claude.go (L137 to 151)
  Note: The installer commands execute remote network content via PowerShell `iex` or `curl ... | bash` without independent integrity verification.
```
func claudeInstallerCommand(goos string) (string, []string, error) {
	switch goos {
	case "windows":
		return "powershell", []string{
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			"irm https://claude.ai/install.ps1 | iex",
		}, nil
	case "darwin", "linux":
		return "bash", []string{
			"-c",
			"curl -fsSL https://claude.ai/install.sh | bash",
		}, nil
```

cmd/launch/claude.go (L52 to 58)
  Note: Claude launch now calls the auto-install routine before executing the Claude binary.
```
func (c *Claude) Run(model string, _ []LaunchModel, args []string) error {
	claudePath, err := ensureClaudeInstalled()
	if err != nil {
		return err
	}

	cmd := exec.Command(claudePath, c.args(model, args)...)
```

cmd/launch/claude.go (L85 to 104)
  Note: After confirmation, the code runs the installer command directly and attaches the user's terminal to it.
```
	ok, err := ConfirmPrompt("Claude Code is not installed. Install now?")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("claude installation cancelled")
	}

	bin, args, err := claudeInstallerCommand(runtime.GOOS)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "\nInstalling Claude Code...\n")
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install claude: %w", err)
```

cmd/launch/registry.go (L43 to 51)
  Note: The Claude integration is registered as auto-installable, making this new installer path reachable from integration install flows.
```
		Install: IntegrationInstallSpec{
			CheckInstalled: func() bool {
				_, err := (&Claude{}).findPath()
				return err == nil
			},
			EnsureInstalled: func() error {
				_, err := ensureClaudeInstalled()
				return err
			},
```

Proposed patch:
diff --git a/cmd/launch/claude.go b/cmd/launch/claude.go
index 394feefb58b1674a30510488b58ccb62bfff283e..8872ced1bf09127879f70a68582631b184620c01 100644
--- a/cmd/launch/claude.go
+++ b/cmd/launch/claude.go
@@ -1,179 +1,97 @@
 package launch
 
 import (
 	"fmt"
 	"os"
 	"os/exec"
 	"path/filepath"
 	"runtime"
 	"strconv"
-	"strings"
 
 	"github.com/ollama/ollama/envconfig"
 )
 
 // Claude implements Runner for Claude Code integration.
 type Claude struct{}
 
 func (c *Claude) String() string { return "Claude Code" }
 
 func (c *Claude) args(model string, extra []string) []string {
 	var args []string
 	if model != "" {
 		args = append(args, "--model", model)
 	}
 	args = append(args, extra...)
 	return args
 }
 
 func (c *Claude) findPath() (string, error) {
 	if p, err := exec.LookPath("claude"); err == nil {
 		return p, nil
 	}
 	home, err := os.UserHomeDir()
 	if err != nil {
 		return "", err
 	}
 	name := "claude"
 	if runtime.GOOS == "windows" {
 		name = "claude.exe"
 	}
 	for _, fallback := range []string{
 		filepath.Join(home, ".local", "bin", name),
 		filepath.Join(home, ".claude", "local", name),
 	} {
 		if _, err := os.Stat(fallback); err == nil {
 			return fallback, nil
 		}
 	}
 	return "", fmt.Errorf("claude binary not found")
 }
 
 func (c *Claude) Run(model string, _ []LaunchModel, args []string) error {
-	claudePath, err := ensureClaudeInstalled()
+	claudePath, err := c.findPath()
 	if err != nil {
 		return err
 	}
 
 	cmd := exec.Command(claudePath, c.args(model, args)...)
 	cmd.Stdin = os.Stdin
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 
 	cmd.Env = append(os.Environ(), c.envVars(model)...)
 	return cmd.Run()
 }
 
 func (c *Claude) envVars(model string) []string {
 	env := []string{
 		"ANTHROPIC_BASE_URL=" + envconfig.Host().String(),
 		"ANTHROPIC_API_KEY=",
 		"ANTHROPIC_AUTH_TOKEN=ollama",
 		"CLAUDE_CODE_ATTRIBUTION_HEADER=0",
 		"DISABLE_ERROR_REPORTING=1",
 		"DISABLE_FEEDBACK_COMMAND=1",
 		"CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1",
 	}
 
 	env = append(env, c.modelEnvVars(model)...)
 	return env
 }
 
-func ensureClaudeInstalled() (string, error) {
-	if path, err := (&Claude{}).findPath(); err == nil {
-		return path, nil
-	}
-
-	if err := checkClaudeInstallerDependencies(); err != nil {
-		return "", err
-	}
-
-	ok, err := ConfirmPrompt("Claude Code is not installed. Install now?")
-	if err != nil {
-		return "", err
-	}
-	if !ok {
-		return "", fmt.Errorf("claude installation cancelled")
-	}
-
-	bin, args, err := claudeInstallerCommand(runtime.GOOS)
-	if err != nil {
-		return "", err
-	}
-
-	fmt.Fprintf(os.Stderr, "\nInstalling Claude Code...\n")
-	cmd := exec.Command(bin, args...)
-	cmd.Stdin = os.Stdin
-	cmd.Stdout = os.Stdout
-	cmd.Stderr = os.Stderr
-	if err := cmd.Run(); err != nil {
-		return "", fmt.Errorf("failed to install claude: %w", err)
-	}
-
-	path, err := (&Claude{}).findPath()
-	if err != nil {
-		return "", fmt.Errorf("claude was installed but the binary was not found on PATH\n\nYou may need to restart your shell")
-	}
-
-	fmt.Fprintf(os.Stderr, "%sClaude Code installed successfully%s\n\n", ansiGreen, ansiReset)
-	return path, nil
-}
-
-func checkClaudeInstallerDependencies() error {
-	switch runtime.GOOS {
-	case "windows":
-		if _, err := exec.LookPath("powershell"); err != nil {
-			return fmt.Errorf("claude is not installed and required dependencies are missing\n\nInstall the following first:\n  PowerShell: https://learn.microsoft.com/powershell/\n\nThen re-run:\n  ollama launch claude")
-		}
-	default:
-		var missing []string
-		if _, err := exec.LookPath("curl"); err != nil {
-			missing = append(missing, "curl: https://curl.se/")
-		}
-		if _, err := exec.LookPath("bash"); err != nil {
-			missing = append(missing, "bash: https://www.gnu.org/software/bash/")
-		}
-		if len(missing) > 0 {
-			return fmt.Errorf("claude is not installed and required dependencies are missing\n\nInstall the following first:\n  %s\n\nThen re-run:\n  ollama launch claude", strings.Join(missing, "\n  "))
-		}
-	}
-	return nil
-}
-
-func claudeInstallerCommand(goos string) (string, []string, error) {
-	switch goos {
-	case "windows":
-		return "powershell", []string{
-			"-NoProfile",
-			"-ExecutionPolicy",
-			"Bypass",
-			"-Command",
-			"irm https://claude.ai/install.ps1 | iex",
-		}, nil
-	case "darwin", "linux":
-		return "bash", []string{
-			"-c",
-			"curl -fsSL https://claude.ai/install.sh | bash",
-		}, nil
-	default:
-		return "", nil, fmt.Errorf("unsupported platform for claude install: %s", goos)
-	}
-}
-
 // modelEnvVars returns Claude Code env vars that route all model tiers through Ollama.
 func (c *Claude) modelEnvVars(model string) []string {
 	env := []string{
 		"ANTHROPIC_DEFAULT_OPUS_MODEL=" + model,
 		"ANTHROPIC_DEFAULT_SONNET_MODEL=" + model,
 		"ANTHROPIC_DEFAULT_HAIKU_MODEL=" + model,
 		"CLAUDE_CODE_SUBAGENT_MODEL=" + model,
 	}
 
 	if isCloudModelName(model) {
 		if l, ok := lookupCloudModelLimit(model); ok {
 			env = append(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW="+strconv.Itoa(l.Context))
 		}
 	}
 
 	return env
 }


diff --git a/cmd/launch/claude_test.go b/cmd/launch/claude_test.go
index e27d8097b2e91b105f6a154ceb008faea992b94e..bd8686dd214ede87c0466bfa3b059ee81aee5257 100644
--- a/cmd/launch/claude_test.go
+++ b/cmd/launch/claude_test.go
@@ -1,29 +1,28 @@
 package launch
 
 import (
-	"fmt"
 	"os"
 	"path/filepath"
 	"runtime"
 	"slices"
 	"strings"
 	"testing"
 
 	"github.com/ollama/ollama/envconfig"
 )
 
 func TestClaudeIntegration(t *testing.T) {
 	c := &Claude{}
 
 	t.Run("String", func(t *testing.T) {
 		if got := c.String(); got != "Claude Code" {
 			t.Errorf("String() = %q, want %q", got, "Claude Code")
 		}
 	})
 
 	t.Run("implements Runner", func(t *testing.T) {
 		var _ Runner = c
 	})
 }
 
 func TestClaudeFindPath(t *testing.T) {
@@ -82,254 +81,50 @@ func TestClaudeFindPath(t *testing.T) {
 		fallback := filepath.Join(tmpDir, ".local", "bin", name)
 		os.MkdirAll(filepath.Dir(fallback), 0o755)
 		os.WriteFile(fallback, []byte("#!/bin/sh\n"), 0o755)
 
 		got, err := c.findPath()
 		if err != nil {
 			t.Fatalf("unexpected error: %v", err)
 		}
 		if got != fallback {
 			t.Errorf("findPath() = %q, want %q", got, fallback)
 		}
 	})
 
 	t.Run("returns error when neither PATH nor fallback exists", func(t *testing.T) {
 		tmpDir := t.TempDir()
 		setTestHome(t, tmpDir)
 		t.Setenv("PATH", t.TempDir()) // empty dir, no claude binary
 
 		_, err := c.findPath()
 		if err == nil {
 			t.Fatal("expected error, got nil")
 		}
 	})
 }
 
-func TestEnsureClaudeInstalled(t *testing.T) {
-	withConfirm := func(t *testing.T, fn func(prompt string) (bool, error)) {
-		t.Helper()
-		oldConfirm := DefaultConfirmPrompt
-		DefaultConfirmPrompt = func(prompt string, options ConfirmOptions) (bool, error) {
-			return fn(prompt)
-		}
-		t.Cleanup(func() { DefaultConfirmPrompt = oldConfirm })
-	}
-
-	t.Run("already installed", func(t *testing.T) {
-		setTestHome(t, t.TempDir())
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		writeFakeBinary(t, tmpDir, "claude")
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			t.Fatalf("did not expect prompt, got %q", prompt)
-			return false, nil
-		})
-
-		bin, err := ensureClaudeInstalled()
-		if err != nil {
-			t.Fatalf("ensureClaudeInstalled() error = %v", err)
-		}
-		if filepath.Base(bin) != "claude" && filepath.Base(bin) != "claude.cmd" {
-			t.Fatalf("bin = %q, want claude binary", bin)
-		}
-	})
-
-	t.Run("missing dependencies", func(t *testing.T) {
-		setTestHome(t, t.TempDir())
-		t.Setenv("PATH", t.TempDir())
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			t.Fatalf("did not expect prompt, got %q", prompt)
-			return false, nil
-		})
-
-		_, err := ensureClaudeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "required dependencies are missing") {
-			t.Fatalf("expected missing dependency error, got %v", err)
-		}
-	})
-
-	t.Run("missing and user declines install", func(t *testing.T) {
-		setTestHome(t, t.TempDir())
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		writeClaudeInstallerDeps(t, tmpDir)
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			if prompt != "Claude Code is not installed. Install now?" {
-				t.Fatalf("unexpected prompt: %q", prompt)
-			}
-			return false, nil
-		})
-
-		_, err := ensureClaudeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "installation cancelled") {
-			t.Fatalf("expected cancellation error, got %v", err)
-		}
-	})
-
-	t.Run("missing and user confirms install succeeds", func(t *testing.T) {
-		if runtime.GOOS == "windows" {
-			t.Skip("uses POSIX shell fake binaries")
-		}
-
-		homeDir := t.TempDir()
-		setTestHome(t, homeDir)
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-
-		writeFakeBinary(t, tmpDir, "curl")
-
-		installLog := filepath.Join(tmpDir, "bash.log")
-		installedClaude := filepath.Join(homeDir, ".local", "bin", "claude")
-		bashScript := fmt.Sprintf(`#!/bin/sh
-echo "$@" >> %q
-if [ "$1" = "-c" ]; then
-  /bin/mkdir -p %q
-  /bin/cat > %q <<'EOS'
-#!/bin/sh
-exit 0
-EOS
-  /bin/chmod +x %q
-fi
-exit 0
-`, installLog, filepath.Dir(installedClaude), installedClaude, installedClaude)
-		if err := os.WriteFile(filepath.Join(tmpDir, "bash"), []byte(bashScript), 0o755); err != nil {
-			t.Fatalf("failed to write fake bash: %v", err)
-		}
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			return true, nil
-		})
-
-		bin, err := ensureClaudeInstalled()
-		if err != nil {
-			t.Fatalf("ensureClaudeInstalled() error = %v", err)
-		}
-		if bin != installedClaude {
-			t.Fatalf("bin = %q, want %q", bin, installedClaude)
-		}
-
-		logData, err := os.ReadFile(installLog)
-		if err != nil {
-			t.Fatalf("failed to read install log: %v", err)
-		}
-		if !strings.Contains(string(logData), "https://claude.ai/install.sh") {
-			t.Fatalf("expected install.sh command in log, got:\n%s", string(logData))
-		}
-	})
-
-	t.Run("install command fails", func(t *testing.T) {
-		if runtime.GOOS == "windows" {
-			t.Skip("uses POSIX shell fake binaries")
-		}
-
-		setTestHome(t, t.TempDir())
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		writeFakeBinary(t, tmpDir, "curl")
-		if err := os.WriteFile(filepath.Join(tmpDir, "bash"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
-			t.Fatalf("failed to write fake bash: %v", err)
-		}
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			return true, nil
-		})
-
-		_, err := ensureClaudeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "failed to install claude") {
-			t.Fatalf("expected install failure error, got %v", err)
-		}
-	})
-}
-
-func writeClaudeInstallerDeps(t *testing.T, dir string) {
-	t.Helper()
-	if runtime.GOOS == "windows" {
-		writeFakeBinary(t, dir, "powershell")
-		return
-	}
-	writeFakeBinary(t, dir, "curl")
-	writeFakeBinary(t, dir, "bash")
-}
-
-func TestClaudeInstallerCommand(t *testing.T) {
-	tests := []struct {
-		name    string
-		goos    string
-		wantBin string
-		want    string
-		wantErr string
-	}{
-		{
-			name:    "unix",
-			goos:    "linux",
-			wantBin: "bash",
-			want:    "curl -fsSL https://claude.ai/install.sh | bash",
-		},
-		{
-			name:    "macos",
-			goos:    "darwin",
-			wantBin: "bash",
-			want:    "curl -fsSL https://claude.ai/install.sh | bash",
-		},
-		{
-			name:    "windows",
-			goos:    "windows",
-			wantBin: "powershell",
-			want:    "irm https://claude.ai/install.ps1 | iex",
-		},
-		{
-			name:    "unsupported",
-			goos:    "plan9",
-			wantErr: "unsupported platform",
-		},
-	}
-
-	for _, tt := range tests {
-		t.Run(tt.name, func(t *testing.T) {
-			bin, args, err := claudeInstallerCommand(tt.goos)
-			if tt.wantErr != "" {
-				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
-					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
-				}
-				return
-			}
-			if err != nil {
-				t.Fatalf("claudeInstallerCommand() error = %v", err)
-			}
-			if bin != tt.wantBin {
-				t.Fatalf("bin = %q, want %q", bin, tt.wantBin)
-			}
-			if !slices.Contains(args, tt.want) {
-				t.Fatalf("args = %v, want command containing %q", args, tt.want)
-			}
-		})
-	}
-}
-
 func TestClaudeArgs(t *testing.T) {
 	c := &Claude{}
 
 	tests := []struct {
 		name  string
 		model string
 		args  []string
 		want  []string
 	}{
 		{"with model", "llama3.2", nil, []string{"--model", "llama3.2"}},
 		{"empty model", "", nil, nil},
 		{"with model and verbose", "llama3.2", []string{"--verbose"}, []string{"--model", "llama3.2", "--verbose"}},
 		{"empty model with help", "", []string{"--help"}, []string{"--help"}},
 		{"with allowed tools", "llama3.2", []string{"--allowedTools", "Read,Write,Bash"}, []string{"--model", "llama3.2", "--allowedTools", "Read,Write,Bash"}},
 		{"with channels", "llama3.2", []string{"--channels", "plugin:telegram@claude-plugins-official"}, []string{"--model", "llama3.2", "--channels", "plugin:telegram@claude-plugins-official"}},
 	}
 
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			got := c.args(tt.model, tt.args)
 			if !slices.Equal(got, tt.want) {
 				t.Errorf("args(%q, %v) = %v, want %v", tt.model, tt.args, got, tt.want)
 			}
 		})
 	}


diff --git a/cmd/launch/integrations_test.go b/cmd/launch/integrations_test.go
index 8a0c03b52aba52f28d1eee1969a7176d4985d3e8..d362a3a87191019084094efaca066a7f9881aded 100644
--- a/cmd/launch/integrations_test.go
+++ b/cmd/launch/integrations_test.go
@@ -2083,51 +2083,51 @@ func TestIntegration_Editor(t *testing.T) {
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			got := false
 			integration, err := integrationFor(tt.name)
 			if err == nil {
 				got = integration.editor
 			}
 			if got != tt.want {
 				t.Errorf("integrationFor(%q).editor = %v, want %v", tt.name, got, tt.want)
 			}
 		})
 	}
 }
 
 func TestIntegration_AutoInstallable(t *testing.T) {
 	tests := []struct {
 		name string
 		want bool
 	}{
 		{"openclaw", true},
 		{"pi", true},
 		{"hermes", true},
 		{"hermes-desktop", true},
 		{"cline", true},
 		{"qwen", true},
-		{"claude", true},
+		{"claude", false},
 		{"claude-desktop", false},
 		{"codex", false},
 		{"opencode", true},
 		{"omp", false},
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			got := false
 			integration, err := integrationFor(tt.name)
 			if err == nil {
 				got = integration.autoInstallable
 			}
 			if got != tt.want {
 				t.Errorf("integrationFor(%q).autoInstallable = %v, want %v", tt.name, got, tt.want)
 			}
 		})
 	}
 }
 
 func TestEnsureIntegrationInstalled_PoolsideUnsupportedOnWindows(t *testing.T) {
 	prev := poolsideGOOS
 	poolsideGOOS = "windows"
 	t.Cleanup(func() { poolsideGOOS = prev })
 
 	err := EnsureIntegrationInstalled("pool", &Poolside{})


diff --git a/cmd/launch/registry.go b/cmd/launch/registry.go
index 52c93a24dde99b45ef4ff4e6dfdd18397cdc6fab..10b392f5a5346addc8345ebaee317f8a252404a3 100644
--- a/cmd/launch/registry.go
+++ b/cmd/launch/registry.go
@@ -23,54 +23,50 @@ type IntegrationSpec struct {
 	Aliases     []string
 	Hidden      bool
 	Description string
 	Install     IntegrationInstallSpec
 }
 
 // IntegrationInfo contains display information about a registered integration.
 type IntegrationInfo struct {
 	Name        string
 	DisplayName string
 	Description string
 }
 
 var launcherIntegrationOrder = []string{"claude", "chatgpt", "hermes", "openclaw", "opencode", "hermes-desktop", "codex", "copilot", "omp", "cline", "droid", "pi", "pool", "qwen"}
 
 var integrationSpecs = []*IntegrationSpec{
 	{
 		Name:        "claude",
 		Runner:      &Claude{},
 		Description: "Anthropic's coding tool with subagents",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := (&Claude{}).findPath()
 				return err == nil
 			},
-			EnsureInstalled: func() error {
-				_, err := ensureClaudeInstalled()
-				return err
-			},
 			URL: "https://code.claude.com/docs/en/quickstart",
 		},
 	},
 	{
 		Name:        "claude-desktop",
 		Runner:      &ClaudeDesktop{},
 		Aliases:     []string{"claude-app"},
 		Description: "Claude Desktop with Ollama Cloud",
 		Hidden:      true,
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				return claudeDesktopInstalled()
 			},
 			URL: "https://claude.com/download",
 		},
 	},
 	{
 		Name:        "cline",
 		Runner:      &Cline{},
 		Description: "Autonomous coding agent with parallel execution",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := exec.LookPath("cline")
 				return err == nil
 			},

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Severity is kept at high. The vulnerable code is in-scope main CLI runtime code and is reachable through `ollama launch claude`. The repository evidence shows an executable sink that runs network-fetched installer bytes through `bash` or PowerShell `iex` with no independent integrity control. The security impact is arbitrary code execution as the invoking OS user, and validation evidence included an executable PoC simulating compromised installer content. This does not rise to critical because it is not default unauthenticated inbound RCE against the Ollama daemon and requires user/operator interaction plus a compromised installer delivery path; however, that is still a credible supply-chain attack path with major impact.
## Likelihood
high - The path is easy for a user to trigger during normal Claude integration launch and can be auto-approved with `--yes`, but an arbitrary internet attacker cannot directly invoke it against the daemon. Exploitation requires compromise/control of the vendor installer endpoint or delivery path and a user/operator install event. | Remote network vector
## Impact
high - Successful exploitation executes attacker-supplied shell or PowerShell code as the local user running Ollama. That can compromise user files, local credentials, SSH keys, Ollama configuration, models, and any tokens available to the user context.
## Assumptions
- The affected `cmd/launch` package is shipped as part of the Ollama CLI product.
- The attacker model is compromise or malicious control of the Claude installer content delivered from `https://claude.ai/install.sh` or `https://claude.ai/install.ps1`, or a trusted TLS/network interception path.
- The local user either confirms the prompt or invokes launch with a policy that auto-approves confirmations, such as `--yes`.
- Claude Code is not already installed, so `ensureClaudeInstalled()` takes the installer path.
- Claude binary absent from PATH or fallback locations
- User runs `ollama launch claude` or selects the Claude integration from the launcher
- User confirms install, or launch policy auto-approves confirmation
- Attacker controls the remote installer content or its delivery path
## Path
User `ollama launch claude`
  -> Claude missing
  -> confirm install / --yes auto-approve
  -> exec.Command("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash") or PowerShell `irm ... | iex`
  -> compromised installer bytes become shell code
  -> arbitrary code execution as invoking local user
## Path evidence
- `cmd/cmd.go:2616` - The root Ollama CLI registers the launch command, placing this flow in the shipped product CLI.
- `cmd/launch/launch.go:280-316` - The CLI exposes `launch [INTEGRATION]` and documents `ollama launch claude` as a supported example.
- `cmd/launch/launch.go:83-105` - `--yes` causes `LaunchConfirmAutoApprove`, which maps to a confirmation policy with `yes: true`.
- `cmd/launch/selector_hooks.go:95-100` - The shared confirmation function returns true immediately when the launch confirm policy has `yes` set.
- `cmd/launch/launch.go:477-523` - `LaunchIntegration` calls `EnsureIntegrationInstalled` before launching non-configure integrations, making the install path reachable.
- `cmd/launch/registry.go:38-52` - The Claude integration is registered with `EnsureInstalled` calling `ensureClaudeInstalled()`.
- `cmd/launch/claude.go:52-58` - `Claude.Run()` calls `ensureClaudeInstalled()` before executing Claude.
- `cmd/launch/claude.go:76-104` - When Claude is absent, the code prompts, obtains an installer command, and executes it with inherited stdin/stdout/stderr.
- `cmd/launch/claude.go:137-151` - The installer commands directly execute remote HTTPS content through PowerShell `iex` or `curl ... | bash` without independent integrity verification.
## Narrative
The finding is real and reachable in the shipped CLI. `Claude.Run()` now calls `ensureClaudeInstalled()` instead of only locating an existing binary. If Claude is absent, the code checks for shell dependencies, asks for confirmation, and then runs `exec.Command` with user stdio attached. The command executes `curl -fsSL https://claude.ai/install.sh | bash` on Linux/macOS or `irm https://claude.ai/install.ps1 | iex` under PowerShell on Windows. Claude is registered as auto-installable, and the launch command exposes `ollama launch claude`; `--yes` maps to an auto-approve confirmation policy. There is no checksum, signature verification, pinned artifact, or independent post-download validation in the repository evidence. Impact is arbitrary code execution as the invoking local user if the installer content or delivery path is compromised. Severity remains high because this is a credible supply-chain code-execution path in a main product CLI, but it is not critical because exploitation is not unauthenticated inbound RCE: it requires a missing integration, user/operator launch action or auto-approval, and compromise of a remote installer trust path.
## Controls
- Local CLI/operator-mediated entry point rather than inbound server route
- Interactive confirmation prompt in ordinary interactive use
- `--yes` can auto-approve the confirmation
- Uses HTTPS URLs for the installer fetch
- Requires local dependencies `curl` and `bash` or PowerShell
- No repository evidence of checksum verification, signature verification, artifact pinning, or sandboxing before execution
## Blindspots
- Static analysis did not contact or inspect the live `claude.ai` installer endpoint.
- Repository-only review cannot determine whether external Claude installer infrastructure has independent signing or transparency controls outside Ollama.
- No deployment manifests indicate fleet size or how commonly users run `ollama launch claude` with `--yes`.
- Static review did not inspect built release packaging to confirm whether this exact commit is included in all distributed binaries.