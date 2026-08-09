OpenCode auto-install runs unverified remote code
Link: https://chatgpt.com/codex/cloud/security/findings/20fdcb5b6ad88191bb106407389cfb5a?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: e434a93
Author: 63033505+hoyyeva@users.noreply.github.com
Created: 9/8/2026, 1:49:49 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced: OpenCode is now marked auto-installable and the install implementation executes an unpinned remote shell installer or latest npm package without integrity verification.
This commit makes the OpenCode integration auto-installable. On Linux and macOS it runs `curl -fsSL https://opencode.ai/install | bash` through `bash -c`; on Windows it runs `npm install -g opencode-ai@latest`. In both cases, Ollama executes code fetched at install time without verifying a pinned digest, signature, or immutable version. If the installer endpoint, DNS/TLS trust chain, npm package, or distribution account is compromised, a user who launches OpenCode and accepts the prompt, or uses `--yes`, will execute attacker-controlled code with the user's privileges. This is a CWE-494 style supply-chain/code-download issue introduced for the OpenCode integration.

# Validation
## Rubric
- [x] Confirm the commit introduced OpenCode auto-install instead of only returning a manual install error.
- [x] Confirm Linux/macOS installer construction uses unpinned `curl -fsSL https://opencode.ai/install | bash` and Windows uses mutable `opencode-ai@latest`.
- [x] Confirm the generic launch registry makes OpenCode auto-installable/reachable via `EnsureInstalled`.
- [x] Confirm acceptance/`--yes` can reach `exec.Command` with user privileges and without digest/signature/immutable-version verification.
- [x] Reproduce the risky behavior dynamically with a bounded PoC and debugger trace; direct crash was not applicable, valgrind was attempted but unavailable.
## Report
Validated the finding. Targeted code review of the changed files shows that commit e434a938 introduced `const openCodeInstallScript = "curl -fsSL https://opencode.ai/install | bash"` at `cmd/launch/opencode.go:17`; `ensureOpenCodeInstalled()` prompts and then runs `exec.Command(bin, args...)` with inherited stdin/stdout/stderr at `cmd/launch/opencode.go:77-95`; `openCodeInstallerCommand()` returns `npm install -g opencode-ai@latest` on Windows and `bash -c "set -o pipefail; curl -fsSL https://opencode.ai/install | bash"` on Linux/macOS at `cmd/launch/opencode.go:129-134`; and OpenCode is registered as auto-installable via `EnsureInstalled` at `cmd/launch/registry.go:155-162`. `git diff HEAD^..HEAD` confirms the commit changed `OpenCode.Run` from returning an install-from-URL error to calling `ensureOpenCodeInstalled()`, and added the registry `EnsureInstalled` hook.

Dynamic PoC: I added `cmd/launch/opencode_poc_test.go`, which sets a temporary HOME/PATH, places a real `/bin/bash` and a fake `curl` in PATH, auto-accepts the prompt, and calls the real `ensureOpenCodeInstalled()`. The fake `curl` returns a shell payload that writes a marker file and creates `~/.opencode/bin/opencode`. The test then asserts both that the marker exists and that `curl` was invoked with `https://opencode.ai/install` and `-fsSL` (`cmd/launch/opencode_poc_test.go:47-85`). Command output: `go test ./cmd/launch -run TestOpenCodeCurlPipeInstallerExecutesFetchedShell -count=1 -v` produced `Installing OpenCode...`, `OpenCode installed successfully`, and `--- PASS: TestOpenCodeCurlPipeInstallerExecutesFetchedShell`, proving attacker-controlled installer response bytes are executed as shell code by the real install path.

Crash/valgrind/debugger attempts: direct debug test-binary execution passed without a crash, which is expected because this is a CWE-494 supply-chain/code-download issue rather than memory corruption. Valgrind was attempted and was unavailable (`valgrind not installed`). I then built a debug Go test binary with `go test -c -gcflags=all='-N -l' -o /tmp/launch.test ./cmd/launch` and traced it in LLDB. LLDB hit `ensureOpenCodeInstalled` at `opencode.go:81` immediately after confirmation and then at `opencode.go:91` on `cmd := exec.Command(bin, args...)`; the backtrace shows the call originated from `TestOpenCodeCurlPipeInstallerExecutesFetchedShell at opencode_poc_test.go:63`, after which the test completed successfully. This confirms the vulnerable execution chain reaches the process-spawn site.

# Evidence
cmd/launch/opencode.go (L129 to 135)
  Note: Constructs unpinned installer commands: `npm install -g opencode-ai@latest` on Windows and curl-piped-to-bash on Linux/macOS.
```
func openCodeInstallerCommand(goos string) (string, []string, error) {
	switch goos {
	case "windows":
		return "npm", []string{"install", "-g", "opencode-ai@latest"}, nil
	case "darwin", "linux":
		return "bash", []string{"-c", "set -o pipefail; " + openCodeInstallScript}, nil
	default:
```

cmd/launch/opencode.go (L17 to 17)
  Note: Defines the Linux/macOS installer as a curl-to-bash command with no integrity verification.
```
const openCodeInstallScript = "curl -fsSL https://opencode.ai/install | bash"
```

cmd/launch/opencode.go (L68 to 95)
  Note: The new auto-install flow prompts, then executes the installer command directly with user privileges.
```
func ensureOpenCodeInstalled() (string, error) {
	if opencodePath, ok := findOpenCode(); ok {
		return opencodePath, nil
	}

	if err := checkOpenCodeInstallerDependencies(); err != nil {
		return "", err
	}

	ok, err := ConfirmPrompt("OpenCode is not installed. Install now?")
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("opencode installation cancelled")
	}

	bin, args, err := openCodeInstallerCommand(openCodeGOOS)
	if err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "\nInstalling OpenCode...\n")
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
```

cmd/launch/registry.go (L160 to 162)
  Note: Marks OpenCode as auto-installable, making the unverified installer reachable from the generic launch installation flow.
```
			EnsureInstalled: func() error {
				_, err := ensureOpenCodeInstalled()
				return err
```

Proposed patch:
diff --git a/cmd/launch/integrations_test.go b/cmd/launch/integrations_test.go
index 8a0c03b52aba52f28d1eee1969a7176d4985d3e8..2c0387aceff79e9255642c62d48f2b42fab3e9f9 100644
--- a/cmd/launch/integrations_test.go
+++ b/cmd/launch/integrations_test.go
@@ -2086,51 +2086,51 @@ func TestIntegration_Editor(t *testing.T) {
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
 		{"claude", true},
 		{"claude-desktop", false},
 		{"codex", false},
-		{"opencode", true},
+		{"opencode", false},
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
 	if err == nil {
 		t.Fatal("expected Windows unsupported error")
 	}


diff --git a/cmd/launch/opencode.go b/cmd/launch/opencode.go
index 1f38666a6bae55812bbdfbbb516af532fedcbf32..9003306c0c227d664b8b438725fba8be2d45740f 100644
--- a/cmd/launch/opencode.go
+++ b/cmd/launch/opencode.go
@@ -1,164 +1,90 @@
 package launch
 
 import (
 	"encoding/json"
 	"fmt"
 	"os"
 	"os/exec"
 	"path/filepath"
 	"runtime"
 	"slices"
 	"strings"
 
 	"github.com/ollama/ollama/cmd/internal/fileutil"
 	"github.com/ollama/ollama/envconfig"
 )
 
-const openCodeInstallScript = "curl -fsSL https://opencode.ai/install | bash"
-
 var openCodeGOOS = runtime.GOOS
 
 // OpenCode implements Runner and Editor for OpenCode integration.
 // Config is passed via OPENCODE_CONFIG_CONTENT env var at launch time
 // instead of writing to opencode's config files.
 type OpenCode struct {
 	configContent string // JSON config built by Edit, passed to Run via env var
 }
 
 func (o *OpenCode) String() string { return "OpenCode" }
 
 // findOpenCode returns the opencode binary path, checking PATH first then the
 // curl installer location (~/.opencode/bin) which may not be on PATH yet.
 func findOpenCode() (string, bool) {
 	if p, err := exec.LookPath("opencode"); err == nil {
 		return p, true
 	}
 	home, err := os.UserHomeDir()
 	if err != nil {
 		return "", false
 	}
 	name := "opencode"
 	if openCodeGOOS == "windows" {
 		name = "opencode.exe"
 	}
 	fallback := filepath.Join(home, ".opencode", "bin", name)
 	if _, err := os.Stat(fallback); err == nil {
 		return fallback, true
 	}
 	return "", false
 }
 
 func (o *OpenCode) Run(model string, models []LaunchModel, args []string) error {
-	opencodePath, err := ensureOpenCodeInstalled()
-	if err != nil {
-		return err
+	opencodePath, ok := findOpenCode()
+	if !ok {
+		return fmt.Errorf("opencode is not installed, install from https://opencode.ai")
 	}
 
 	cmd := exec.Command(opencodePath, args...)
 	cmd.Stdin = os.Stdin
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 	cmd.Env = os.Environ()
 	if content := o.resolveContent(model, models); content != "" {
 		cmd.Env = append(cmd.Env, "OPENCODE_CONFIG_CONTENT="+content)
 	}
 	return cmd.Run()
 }
 
-func ensureOpenCodeInstalled() (string, error) {
-	if opencodePath, ok := findOpenCode(); ok {
-		return opencodePath, nil
-	}
-
-	if err := checkOpenCodeInstallerDependencies(); err != nil {
-		return "", err
-	}
-
-	ok, err := ConfirmPrompt("OpenCode is not installed. Install now?")
-	if err != nil {
-		return "", err
-	}
-	if !ok {
-		return "", fmt.Errorf("opencode installation cancelled")
-	}
-
-	bin, args, err := openCodeInstallerCommand(openCodeGOOS)
-	if err != nil {
-		return "", err
-	}
-
-	fmt.Fprintf(os.Stderr, "\nInstalling OpenCode...\n")
-	cmd := exec.Command(bin, args...)
-	cmd.Stdin = os.Stdin
-	cmd.Stdout = os.Stdout
-	cmd.Stderr = os.Stderr
-	if err := cmd.Run(); err != nil {
-		return "", fmt.Errorf("failed to install opencode: %w", err)
-	}
-
-	opencodePath, ok := findOpenCode()
-	if !ok {
-		return "", fmt.Errorf("opencode was installed but the binary was not found on PATH\n\nYou may need to restart your shell")
-	}
-
-	fmt.Fprintf(os.Stderr, "%sOpenCode installed successfully%s\n\n", ansiGreen, ansiReset)
-	return opencodePath, nil
-}
-
-func checkOpenCodeInstallerDependencies() error {
-	switch openCodeGOOS {
-	case "windows":
-		if _, err := exec.LookPath("npm"); err != nil {
-			return fmt.Errorf("opencode is not installed and required dependencies are missing\n\nInstall the following first:\n  npm (Node.js): https://nodejs.org/\n\nThen re-run:\n  ollama launch opencode")
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
-			return fmt.Errorf("opencode is not installed and required dependencies are missing\n\nInstall the following first:\n  %s\n\nThen re-run:\n  ollama launch opencode", strings.Join(missing, "\n  "))
-		}
-	}
-	return nil
-}
-
-func openCodeInstallerCommand(goos string) (string, []string, error) {
-	switch goos {
-	case "windows":
-		return "npm", []string{"install", "-g", "opencode-ai@latest"}, nil
-	case "darwin", "linux":
-		return "bash", []string{"-c", "set -o pipefail; " + openCodeInstallScript}, nil
-	default:
-		return "", nil, fmt.Errorf("unsupported platform for opencode install: %s", goos)
-	}
-}
-
 // resolveContent returns the inline config to send via OPENCODE_CONFIG_CONTENT.
 // Returns content built by Edit if available, otherwise builds from model.json
 // with the requested model as primary (e.g. re-launch with saved config).
 func (o *OpenCode) resolveContent(model string, models []LaunchModel) string {
 	if o.configContent != "" {
 		return o.configContent
 	}
 	resolvedModels := resolveOpenCodeRunModels(model, models, readModelJSONModels())
 	if len(resolvedModels) == 0 {
 		return ""
 	}
 	content, err := buildInlineConfig(resolvedModels[0], resolvedModels)
 	if err != nil {
 		return ""
 	}
 	return content
 }
 
 func resolveOpenCodeRunModels(primary string, models []LaunchModel, stateModels []string) []LaunchModel {
 	if primary == "" {
 		return nil
 	}
 
 	resolved := make([]LaunchModel, 0, 1+len(models)+len(stateModels))
 	appendModel := func(name string) {


diff --git a/cmd/launch/opencode_test.go b/cmd/launch/opencode_test.go
index 14c9af1789380981d2a6ff9c495bf75b99dd6b4c..be26b897b97f4b661e2799a4c46610d6545f6244 100644
--- a/cmd/launch/opencode_test.go
+++ b/cmd/launch/opencode_test.go
@@ -354,300 +354,57 @@ func TestFindOpenCode(t *testing.T) {
 		if _, ok := findOpenCode(); ok {
 			t.Fatal("findOpenCode should fail when binary is not on PATH or in fallback location")
 		}
 
 		// Create a fake binary at the curl install fallback location
 		binDir := filepath.Join(tmpDir, ".opencode", "bin")
 		os.MkdirAll(binDir, 0o755)
 		name := "opencode"
 		if runtime.GOOS == "windows" {
 			name = "opencode.exe"
 		}
 		fakeBin := filepath.Join(binDir, name)
 		os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0o755)
 
 		// Now findOpenCode should succeed via fallback
 		path, ok := findOpenCode()
 		if !ok {
 			t.Fatal("findOpenCode should succeed with fallback binary")
 		}
 		if path != fakeBin {
 			t.Errorf("findOpenCode = %q, want %q", path, fakeBin)
 		}
 	})
 }
 
-func TestEnsureOpenCodeInstalled(t *testing.T) {
-	oldGOOS := openCodeGOOS
-	t.Cleanup(func() { openCodeGOOS = oldGOOS })
-
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
-		openCodeGOOS = runtime.GOOS
-		writeFakeBinary(t, tmpDir, "opencode")
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			t.Fatalf("did not expect prompt, got %q", prompt)
-			return false, nil
-		})
-
-		bin, err := ensureOpenCodeInstalled()
-		if err != nil {
-			t.Fatalf("ensureOpenCodeInstalled() error = %v", err)
-		}
-		if filepath.Base(bin) == "" {
-			t.Fatalf("expected opencode binary path, got %q", bin)
-		}
-	})
-
-	t.Run("missing dependencies", func(t *testing.T) {
-		setTestHome(t, t.TempDir())
-		t.Setenv("PATH", t.TempDir())
-		openCodeGOOS = "linux"
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			t.Fatalf("did not expect prompt, got %q", prompt)
-			return false, nil
-		})
-
-		_, err := ensureOpenCodeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "required dependencies are missing") {
-			t.Fatalf("expected missing dependency error, got %v", err)
-		}
-	})
+func TestOpenCodeRunRequiresManualInstall(t *testing.T) {
+	setTestHome(t, t.TempDir())
+	t.Setenv("PATH", t.TempDir())
 
-	t.Run("missing and user declines install", func(t *testing.T) {
-		setTestHome(t, t.TempDir())
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		openCodeGOOS = "linux"
-		writeFakeBinary(t, tmpDir, "curl")
-		writeFakeBinary(t, tmpDir, "bash")
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			if !strings.Contains(prompt, "OpenCode is not installed.") {
-				t.Fatalf("unexpected prompt: %q", prompt)
-			}
-			return false, nil
-		})
-
-		_, err := ensureOpenCodeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "installation cancelled") {
-			t.Fatalf("expected cancellation error, got %v", err)
-		}
-	})
-
-	t.Run("missing and user confirms unix install succeeds", func(t *testing.T) {
-		if runtime.GOOS == "windows" {
-			t.Skip("uses POSIX shell fake binaries")
-		}
-
-		homeDir := t.TempDir()
-		setTestHome(t, homeDir)
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		openCodeGOOS = "linux"
-		writeFakeBinary(t, tmpDir, "curl")
-
-		installLog := filepath.Join(tmpDir, "bash.log")
-		opencodePath := filepath.Join(homeDir, ".opencode", "bin", "opencode")
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
-`, installLog, filepath.Dir(opencodePath), opencodePath, opencodePath)
-		if err := os.WriteFile(filepath.Join(tmpDir, "bash"), []byte(bashScript), 0o755); err != nil {
-			t.Fatalf("failed to write fake bash: %v", err)
-		}
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			return true, nil
-		})
-
-		bin, err := ensureOpenCodeInstalled()
-		if err != nil {
-			t.Fatalf("ensureOpenCodeInstalled() error = %v", err)
-		}
-		if bin != opencodePath {
-			t.Fatalf("bin = %q, want %q", bin, opencodePath)
-		}
-
-		logData, err := os.ReadFile(installLog)
-		if err != nil {
-			t.Fatalf("failed to read install log: %v", err)
-		}
-		if !strings.Contains(string(logData), openCodeInstallScript) {
-			t.Fatalf("expected opencode install script in log, got:\n%s", string(logData))
-		}
-	})
-
-	t.Run("missing and user confirms windows install succeeds", func(t *testing.T) {
-		if runtime.GOOS == "windows" {
-			t.Skip("uses POSIX shell fake binaries")
-		}
-
-		homeDir := t.TempDir()
-		setTestHome(t, homeDir)
-		tmpDir := t.TempDir()
-		t.Setenv("PATH", tmpDir)
-		openCodeGOOS = "windows"
-
-		installLog := filepath.Join(tmpDir, "npm.log")
-		opencodePath := filepath.Join(homeDir, ".opencode", "bin", "opencode.exe")
-		npmScript := fmt.Sprintf(`#!/bin/sh
-echo "$@" >> %q
-/bin/mkdir -p %q
-/bin/cat > %q <<'EOS'
-@echo off
-exit /b 0
-EOS
-/bin/chmod +x %q
-exit 0
-`, installLog, filepath.Dir(opencodePath), opencodePath, opencodePath)
-		if err := os.WriteFile(filepath.Join(tmpDir, "npm"), []byte(npmScript), 0o755); err != nil {
-			t.Fatalf("failed to write fake npm: %v", err)
-		}
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			return true, nil
-		})
-
-		bin, err := ensureOpenCodeInstalled()
-		if err != nil {
-			t.Fatalf("ensureOpenCodeInstalled() error = %v", err)
-		}
-		if bin != opencodePath {
-			t.Fatalf("bin = %q, want %q", bin, opencodePath)
-		}
-
-		logData, err := os.ReadFile(installLog)
-		if err != nil {
-			t.Fatalf("failed to read install log: %v", err)
-		}
-		if !strings.Contains(string(logData), "install -g opencode-ai@latest") {
-			t.Fatalf("expected npm install command in log, got:\n%s", string(logData))
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
-		openCodeGOOS = "linux"
-		writeFakeBinary(t, tmpDir, "curl")
-		if err := os.WriteFile(filepath.Join(tmpDir, "bash"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
-			t.Fatalf("failed to write fake bash: %v", err)
-		}
-
-		withConfirm(t, func(prompt string) (bool, error) {
-			return true, nil
-		})
-
-		_, err := ensureOpenCodeInstalled()
-		if err == nil || !strings.Contains(err.Error(), "failed to install opencode") {
-			t.Fatalf("expected install failure error, got %v", err)
-		}
-	})
-}
-
-func TestOpenCodeInstallerCommand(t *testing.T) {
-	tests := []struct {
-		name      string
-		goos      string
-		wantBin   string
-		wantParts []string
-		wantErr   bool
-	}{
-		{
-			name:      "linux",
-			goos:      "linux",
-			wantBin:   "bash",
-			wantParts: []string{"-c", "set -o pipefail", "https://opencode.ai/install"},
-		},
-		{
-			name:      "darwin",
-			goos:      "darwin",
-			wantBin:   "bash",
-			wantParts: []string{"-c", "set -o pipefail", "https://opencode.ai/install"},
-		},
-		{
-			name:      "windows",
-			goos:      "windows",
-			wantBin:   "npm",
-			wantParts: []string{"install", "-g", "opencode-ai@latest"},
-		},
-		{
-			name:    "unsupported",
-			goos:    "plan9",
-			wantErr: true,
-		},
-	}
-
-	for _, tt := range tests {
-		t.Run(tt.name, func(t *testing.T) {
-			bin, args, err := openCodeInstallerCommand(tt.goos)
-			if tt.wantErr {
-				if err == nil {
-					t.Fatal("expected error")
-				}
-				return
-			}
-			if err != nil {
-				t.Fatalf("openCodeInstallerCommand() error = %v", err)
-			}
-			if bin != tt.wantBin {
-				t.Fatalf("bin = %q, want %q", bin, tt.wantBin)
-			}
-			joined := strings.Join(args, " ")
-			for _, want := range tt.wantParts {
-				if !strings.Contains(joined, want) {
-					t.Fatalf("args %q missing %q", joined, want)
-				}
-			}
-		})
+	err := (&OpenCode{}).Run("llama3.2", nil, nil)
+	if err == nil || !strings.Contains(err.Error(), "install from https://opencode.ai") {
+		t.Fatalf("expected manual installation error, got %v", err)
 	}
 }
 
 // Verify that the BackfillsCloudModelLimitOnExistingEntry test from the old
 // file-based approach is covered by the new inline config approach.
 func TestOpenCodeEdit_CloudModelLimitStructure(t *testing.T) {
 	o := &OpenCode{}
 	tmpDir := t.TempDir()
 	setTestHome(t, tmpDir)
 
 	expected := cloudModelLimits["glm-4.7"]
 
 	if err := o.Edit(testLaunchModels("glm-4.7:cloud")); err != nil {
 		t.Fatal(err)
 	}
 
 	var cfg map[string]any
 	json.Unmarshal([]byte(o.configContent), &cfg)
 	provider, _ := cfg["provider"].(map[string]any)
 	ollama, _ := provider["ollama"].(map[string]any)
 	models, _ := ollama["models"].(map[string]any)
 	entry, _ := models["glm-4.7:cloud"].(map[string]any)
 
 	limit, ok := entry["limit"].(map[string]any)
 	if !ok {


diff --git a/cmd/launch/registry.go b/cmd/launch/registry.go
index 52c93a24dde99b45ef4ff4e6dfdd18397cdc6fab..ea0f9b75c7cf703a655c6b64f36faf3c2fc7d0a8 100644
--- a/cmd/launch/registry.go
+++ b/cmd/launch/registry.go
@@ -135,54 +135,50 @@ var integrationSpecs = []*IntegrationSpec{
 			},
 			URL: "https://github.com/features/copilot/cli/",
 		},
 	},
 	{
 		Name:        "droid",
 		Runner:      &Droid{},
 		Description: "Factory's coding agent across terminal and IDEs",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := exec.LookPath("droid")
 				return err == nil
 			},
 			URL: "https://docs.factory.ai/cli/getting-started/quickstart",
 		},
 	},
 	{
 		Name:        "opencode",
 		Runner:      &OpenCode{},
 		Description: "Anomaly's open-source coding agent",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, ok := findOpenCode()
 				return ok
 			},
-			EnsureInstalled: func() error {
-				_, err := ensureOpenCodeInstalled()
-				return err
-			},
 			URL: "https://opencode.ai",
 		},
 	},
 	{
 		Name:        "omp",
 		Runner:      &OMP{},
 		Description: "AI coding agent with IDE integration",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := (&OMP{}).findPath()
 				return err == nil
 			},
 			URL: "https://omp.sh",
 		},
 	},
 	{
 		Name:        "openclaw",
 		Runner:      &Openclaw{},
 		Aliases:     []string{"clawdbot", "moltbot"},
 		Description: "Personal AI with 100+ skills",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				if _, err := exec.LookPath("openclaw"); err == nil {
 					return true
 				}

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Kept as high. The vulnerable code is in-scope product CLI/runtime code, OpenCode is a supported launch integration, and the normal launch path can reach an executable sink that runs mutable remote code without pinning or integrity verification. The impact is arbitrary code execution as the invoking OS user. The rating is not critical because exploitation is not zero-click or daemon-network-reachable and depends on user confirmation/--yes plus compromise or control of an external delivery path.
## Likelihood
high - The path is product-supported and easy for users to reach, and `--yes` can remove the interactive prompt. However, exploitation generally requires user action plus compromise/control of a third-party installer, package, DNS/TLS, or distribution account, so it is not a broad unauthenticated remote exploit. | Remote network vector
## Impact
high - If the installer or npm package delivery path is malicious, the code executes with the privileges of the local user running Ollama. That permits local data access, command execution, persistence within user-writable locations, and tampering with the user's Ollama/OpenCode environment.
## Assumptions
- Analysis is limited to repository artifacts in /workspace/ollama and the supplied validation evidence.
- The OpenCode installer endpoint and npm package are outside this repository, so their current contents or signing behavior were not fetched or verified.
- The vulnerable path is evaluated as an end-user CLI/runtime feature, not as a network-exposed Ollama daemon endpoint.
- OpenCode is not already installed or discoverable by findOpenCode
- Victim runs the in-scope Ollama launch integration flow for opencode
- Victim accepts the install prompt, or runs with --yes in a context that reaches launch
- Linux/macOS host has curl and bash, or Windows host has npm
- Attacker controls or compromises the installer delivery path, DNS/TLS trust chain, opencode.ai install script, npm package, npm account, or equivalent distribution component
## Path
Remote supply-chain control
  -> victim: ollama launch opencode
  -> registry EnsureInstalled
  -> prompt accepted / --yes
  -> bash -c 'curl -fsSL https://opencode.ai/install | bash' OR npm install -g opencode-ai@latest
  -> code execution as invoking user
## Path evidence
- `cmd/launch/opencode.go:17` - Defines the Linux/macOS install script as `curl -fsSL https://opencode.ai/install | bash`, which executes a mutable remote response without repository-visible integrity verification.
- `cmd/launch/opencode.go:68-95` - When OpenCode is missing, the installer flow checks dependencies, asks for confirmation, builds the installer command, and executes it with `exec.Command` using inherited terminal streams.
- `cmd/launch/opencode.go:129-135` - Constructs unpinned installer commands: `opencode-ai@latest` via npm on Windows and curl-piped-to-bash on Linux/macOS.
- `cmd/launch/registry.go:151-164` - Registers OpenCode as a supported integration and wires `EnsureInstalled` to `ensureOpenCodeInstalled`, making the installer reachable from the generic launch flow.
- `cmd/launch/launch.go:477-528` - The product launch flow calls `EnsureIntegrationInstalled` before launching non-config-only integrations, including editor integrations such as OpenCode.
- `cmd/launch/registry.go:491-512` - Generic installation helper invokes an integration's auto-install hook when the integration is missing.
- `cmd/launch/selector_hooks.go:85-100` - Confirmation is the primary user-interaction control; it can also be auto-approved when the launch policy has `yes` set.
- `cmd/launch/launch.go:274-316` - Documents `ollama launch [INTEGRATION]` and lists `opencode` as a supported product CLI integration.
## Narrative
The finding is real and reachable. OpenCode is a supported `ollama launch` integration, and the registry marks it auto-installable. If the binary is absent, `ensureOpenCodeInstalled` prompts and then runs an installer command through `exec.Command`. On Linux/macOS the command is `bash -c 'set -o pipefail; curl -fsSL https://opencode.ai/install | bash'`; on Windows it is `npm install -g opencode-ai@latest`. The repository evidence shows no digest, signature, or immutable-version verification before execution. Exploitation is not a direct unauthenticated network attack against the Ollama daemon: it requires victim launch/confirmation and compromise or control of a third-party delivery path. But successful exploitation gives arbitrary code execution as the invoking user, so the original high severity is supportable for a supply-chain code-execution issue, with medium likelihood due to the preconditions.
## Controls
- User confirmation prompt before installing when interactive
- `--yes` flag explicitly auto-approves confirmation prompts
- Headless --yes launch requires a model override for non-autodiscovery integrations
- Dependency checks require curl/bash on Unix-like platforms or npm on Windows
- HTTPS URL is used for the curl installer
- No repository-visible checksum, signature, or immutable version pin is enforced before installer execution
## Blindspots
- Static-only repository review did not fetch or inspect the live opencode.ai installer or npm package.
- No cloud, DNS, TLS, or npm registry APIs were called, so real-world compromise likelihood of the external delivery path was not measured.
- Prevalence of users invoking `ollama launch opencode` with OpenCode absent is unknown.
- The current repository evidence does not show whether the third-party installer performs its own downstream verification after execution begins; this would not remove the initial curl-to-bash execution risk.