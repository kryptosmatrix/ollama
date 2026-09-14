Hermes auto-install executes unverified remote script
Link: https://chatgpt.com/codex/cloud/security/findings/1ed60befb32c8191ba0481f6e46e23cf?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: 43f90de
Author: parth.sareen@ollama.com
Created: 9/8/2026, 3:09:31 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced: the commit adds the Hermes installer and wires it into the integration auto-install flow, creating a new unverified remote-code execution path.
The commit adds an auto-install path for Hermes that runs `curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash ...`. Because the script is fetched from the mutable `main` branch and no checksum, signature, release tag, or pinned commit is verified before execution, compromise of the upstream repository, GitHub account, or delivery path would result in arbitrary code execution as the local user when they launch/install Hermes. This is also executed from the Windows WSL install path. The prompt only asks whether to install Hermes; it does not provide a verifiable artifact identity, and `--yes` launch policy would auto-approve the confirmation.

# Validation
## Rubric
- [x] Confirm Hermes installer is fetched from a mutable upstream branch rather than a pinned immutable artifact.
- [x] Confirm no checksum, signature, pinned commit, release tag, or verification step exists in the Hermes install path.
- [x] Confirm Unix auto-install executes the fetched script after only a generic install confirmation.
- [x] Confirm Windows/WSL install path executes the same unverified script.
- [x] Confirm `--yes`/auto-confirm policy can approve the install without calling the interactive prompt, and a PoC payload returned by curl is executed.
## Report
Validated the finding with targeted code review plus a dynamic Go test PoC. Relevant code evidence: cmd/launch/hermes.go:29 defines hermesInstallScript as `curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash -s -- --skip-setup`, using the mutable `main` branch and no checksum/signature/pinned commit. cmd/launch/hermes.go:221-230 prompts only `Hermes is not installed. Install now?` and then executes `hermesAttachedCommand("bash", "-lc", hermesInstallScript).Run()`. cmd/launch/hermes.go:255-267 uses the same script through WSL on Windows. cmd/launch/launch.go:83-86 maps `--yes` to `LaunchConfirmAutoApprove`, and cmd/launch/selector_hooks.go:79-82 returns true without prompting when that policy is active. I added a temporary PoC test that places a fake `curl` first in PATH. The fake curl emits a shell payload that writes `ARBITRARY_PAYLOAD_EXECUTED args=--skip-setup` and creates a fake Hermes binary so install completion succeeds. Running `go test ./cmd/launch -run TestHermesCurlToBashAutoYesExecutesRemoteScriptPoC -count=1 -v` passed, proving the install path executes whatever shell code is returned by that URL and that the auto-yes policy bypasses the prompt. I also built a debug test binary with `go test -c ./cmd/launch -gcflags='all=-N -l'` and ran the PoC successfully. Crash validation is not applicable because this is an intended command-execution/supply-chain flaw rather than memory corruption; valgrind/gdb/dlv were attempted but unavailable in the container (`command not found`).

# Evidence
cmd/launch/hermes.go (L221 to 231)
  Note: After a generic install confirmation, the Unix install path executes the unverified remote script through bash.
```
	ok, err := ConfirmPrompt("Hermes is not installed. Install now?")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("hermes installation cancelled")
	}

	fmt.Fprintf(os.Stderr, "\nInstalling Hermes...\n")
	if err := hermesAttachedCommand("bash", "-lc", hermesInstallScript).Run(); err != nil {
		return fmt.Errorf("failed to install hermes: %w", err)
```

cmd/launch/hermes.go (L255 to 268)
  Note: The Windows/WSL install path also executes the same unverified remote script.
```
	ok, err := ConfirmPromptWithOptions("Hermes runs through WSL2 on Windows. Install it in WSL now?", ConfirmOptions{
		YesLabel: "Use WSL",
		NoLabel:  "Show manual steps",
	})
	if err != nil {
		return err
	}
	if !ok {
		return hermesWindowsHint(fmt.Errorf("hermes is not installed"))
	}

	fmt.Fprintf(os.Stderr, "\nInstalling Hermes in WSL...\n")
	if err := h.runWSL("bash", "-lc", hermesInstallScript); err != nil {
		return hermesWindowsHint(fmt.Errorf("failed to install hermes in WSL: %w", err))
```

cmd/launch/hermes.go (L28 to 30)
  Note: Defines the installer as a curl-to-bash pipeline from the mutable upstream main branch, with no pinned commit, checksum, or signature verification.
```
const (
	hermesInstallScript     = "curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash -s -- --skip-setup"
	hermesProviderName      = "Ollama"
```

Proposed patch:
diff --git a/cmd/launch/hermes.go b/cmd/launch/hermes.go
index 9c8ffe071efecc75fe9937745bcf64aa867a6c92..bcf0730cde4194e6a275e94ca29c728fa8ee70a9 100644
--- a/cmd/launch/hermes.go
+++ b/cmd/launch/hermes.go
@@ -4,53 +4,50 @@ import (
 	"bufio"
 	"bytes"
 	"context"
 	"fmt"
 	"net/http"
 	"os"
 	"os/exec"
 	"path/filepath"
 	"runtime"
 	"slices"
 	"strconv"
 	"strings"
 
 	"golang.org/x/mod/semver"
 	"gopkg.in/yaml.v3"
 
 	"github.com/ollama/ollama/api"
 	"github.com/ollama/ollama/cmd/config"
 	"github.com/ollama/ollama/cmd/internal/fileutil"
 	"github.com/ollama/ollama/envconfig"
 )
 
 const (
 	// https://github.com/NousResearch/hermes-agent/releases/tag/v2026.6.5
 	hermesDesktopMinVersion = "v0.16.0"
-	hermesInstallScript     = "curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash -s -- --skip-setup"
-	hermesWindowsInstallURL = "https://hermes-agent.nousresearch.com/install.ps1"
-	hermesWindowsInstallCmd = "& ([scriptblock]::Create((irm " + hermesWindowsInstallURL + "))) -SkipSetup"
 	hermesProviderName      = "Ollama"
 	hermesProviderKey       = "ollama-launch"
 	hermesLegacyKey         = "ollama"
 	hermesPlaceholderKey    = "ollama"
 	hermesGatewaySetupHint  = "hermes gateway setup"
 	hermesGatewaySetupTitle = "Connect a messaging app now?"
 )
 
 var (
 	hermesGOOS      = runtime.GOOS
 	hermesLookPath  = exec.LookPath
 	hermesCommand   = exec.Command
 	hermesUserHome  = os.UserHomeDir
 	hermesOllamaURL = envconfig.ConnectableHost
 )
 
 var hermesMessagingEnvGroups = [][]string{
 	{"TELEGRAM_BOT_TOKEN"},
 	{"DISCORD_BOT_TOKEN"},
 	{"SLACK_BOT_TOKEN"},
 	{"SIGNAL_ACCOUNT"},
 	{"EMAIL_ADDRESS"},
 	{"TWILIO_ACCOUNT_SID"},
 	{"MATRIX_ACCESS_TOKEN", "MATRIX_PASSWORD"},
 	{"MATTERMOST_TOKEN"},
@@ -336,99 +333,50 @@ func (h *Hermes) RequiresInteractiveOnboarding() bool {
 	return false
 }
 
 func (h *Hermes) RefreshRuntimeAfterConfigure() error {
 	running, err := h.gatewayRunning()
 	if err != nil {
 		return fmt.Errorf("check Hermes gateway status: %w", err)
 	}
 	if !running {
 		return nil
 	}
 
 	fmt.Fprintf(os.Stderr, "%sRefreshing Hermes messaging gateway...%s\n", ansiGray, ansiReset)
 	if err := h.restartGateway(); err != nil {
 		return fmt.Errorf("restart Hermes gateway: %w", err)
 	}
 	fmt.Fprintln(os.Stderr)
 	return nil
 }
 
 func (h *Hermes) installed() bool {
 	_, err := h.binary()
 	return err == nil
 }
 
-func (h *Hermes) ensureInstalled() error {
-	return h.ensureInstalledFor("hermes")
-}
-
-func (h *Hermes) ensureInstalledFor(command string) error {
-	if h.installed() {
-		return nil
-	}
-
-	var missing []string
-	if hermesGOOS != "windows" {
-		for _, dep := range []string{"bash", "curl", "git"} {
-			if _, err := hermesLookPath(dep); err != nil {
-				missing = append(missing, dep)
-			}
-		}
-	}
-	if len(missing) > 0 {
-		return fmt.Errorf("Hermes is not installed and required dependencies are missing\n\nInstall the following first:\n  %s\n\nThen re-run:\n  ollama launch %s", strings.Join(missing, "\n  "), command)
-	}
-
-	ok, err := ConfirmPrompt("Hermes is not installed. Install now?")
-	if err != nil {
-		return err
-	}
-	if !ok {
-		return fmt.Errorf("hermes installation cancelled")
-	}
-
-	fmt.Fprintf(os.Stderr, "\nInstalling Hermes...\n")
-	if err := h.runInstallScript(); err != nil {
-		return fmt.Errorf("failed to install hermes: %w", err)
-	}
-
-	if !h.installed() {
-		return fmt.Errorf("hermes was installed but the binary was not found on PATH\n\nYou may need to restart your shell")
-	}
-
-	fmt.Fprintf(os.Stderr, "%sHermes installed successfully%s\n\n", ansiGreen, ansiReset)
-	return nil
-}
-
-func (h *Hermes) runInstallScript() error {
-	if hermesGOOS == "windows" {
-		return hermesAttachedCommand("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", hermesWindowsInstallCmd).Run()
-	}
-	return hermesAttachedCommand("bash", "-lc", hermesInstallScript).Run()
-}
-
 func (h *Hermes) listModels(defaultModel string) []string {
 	client := hermesOllamaClient()
 	resp, err := client.List(context.Background())
 	if err != nil {
 		return []string{defaultModel}
 	}
 
 	models := make([]string, 0, len(resp.Models)+1)
 	seen := make(map[string]struct{}, len(resp.Models)+1)
 	add := func(name string) {
 		name = strings.TrimSpace(name)
 		if name == "" {
 			return
 		}
 		if _, ok := seen[name]; ok {
 			return
 		}
 		seen[name] = struct{}{}
 		models = append(models, name)
 	}
 
 	add(defaultModel)
 	for _, entry := range resp.Models {
 		add(entry.Name)
 	}


diff --git a/cmd/launch/hermes_test.go b/cmd/launch/hermes_test.go
index 186c925829a44f6c26a340cbf1cc2d97c444a3de..99870a042e680447b0cd705a5a3d4c491736699c 100644
--- a/cmd/launch/hermes_test.go
+++ b/cmd/launch/hermes_test.go
@@ -1308,191 +1308,50 @@ func TestHermesMessagingConfiguredRecognizesSupportedGatewayVars(t *testing.T) {
 		{name: "matrix token", env: "MATRIX_ACCESS_TOKEN=token\n", want: true},
 		{name: "matrix password", env: "MATRIX_PASSWORD=secret\n", want: true},
 		{name: "mattermost", env: "MATTERMOST_TOKEN=token\n", want: true},
 		{name: "whatsapp", env: "WHATSAPP_PHONE_NUMBER_ID=phone\n", want: true},
 		{name: "dingtalk", env: "DINGTALK_CLIENT_ID=client\n", want: true},
 		{name: "feishu", env: "FEISHU_APP_ID=app\n", want: true},
 		{name: "wecom", env: "WECOM_BOT_ID=bot\n", want: true},
 		{name: "weixin", env: "WEIXIN_ACCOUNT_ID=account\n", want: true},
 		{name: "bluebubbles", env: "BLUEBUBBLES_SERVER_URL=https://example.invalid\n", want: true},
 		{name: "webhooks", env: "WEBHOOK_ENABLED=true\n", want: true},
 	}
 
 	h := &Hermes{}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			if err := os.WriteFile(envPath, []byte(tt.env), 0o644); err != nil {
 				t.Fatal(err)
 			}
 			if got := h.messagingConfigured(); got != tt.want {
 				t.Fatalf("messagingConfigured() = %v, want %v", got, tt.want)
 			}
 		})
 	}
 }
 
-func TestHermesEnsureInstalledWindowsRunsPowerShellInstaller(t *testing.T) {
-	if runtime.GOOS == "windows" {
-		t.Skip("uses a POSIX shell test binary")
-	}
-
-	tmpDir := t.TempDir()
-	setTestHome(t, tmpDir)
-	withLauncherHooks(t)
-	withHermesPlatform(t, "windows")
-	t.Setenv("PATH", tmpDir)
-	t.Setenv("LOCALAPPDATA", filepath.Join(tmpDir, "AppData", "Local"))
-
-	powershell := filepath.Join(tmpDir, "powershell.exe")
-	script := fmt.Sprintf(`#!/bin/sh
-printf '%%s\n' "$*" >> %q
-/bin/mkdir -p %q
-/bin/cat > %q <<'EOS'
-#!/bin/sh
-exit 0
-EOS
-/bin/chmod +x %q
-exit 0
-`,
-		filepath.Join(tmpDir, "powershell.log"),
-		filepath.Dir(filepath.Join(tmpDir, "AppData", "Local", "hermes", "hermes-agent", "venv", "Scripts", "hermes.exe")),
-		filepath.Join(tmpDir, "AppData", "Local", "hermes", "hermes-agent", "venv", "Scripts", "hermes.exe"),
-		filepath.Join(tmpDir, "AppData", "Local", "hermes", "hermes-agent", "venv", "Scripts", "hermes.exe"),
-	)
-	if err := os.WriteFile(powershell, []byte(script), 0o755); err != nil {
-		t.Fatal(err)
-	}
-
-	DefaultConfirmPrompt = func(prompt string, options ConfirmOptions) (bool, error) {
-		if prompt != "Hermes is not installed. Install now?" {
-			t.Fatalf("unexpected install prompt %q", prompt)
-		}
-		return true, nil
-	}
-
-	h := &Hermes{}
-	if err := h.ensureInstalled(); err != nil {
-		t.Fatalf("ensureInstalled returned error: %v", err)
-	}
-
-	data, err := os.ReadFile(filepath.Join(tmpDir, "powershell.log"))
-	if err != nil {
-		t.Fatal(err)
-	}
-	logs := string(data)
-	for _, want := range []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", hermesWindowsInstallURL, "-SkipSetup"} {
-		if !strings.Contains(logs, want) {
-			t.Fatalf("expected PowerShell installer args to contain %q, got logs:\n%s", want, logs)
-		}
-	}
-}
-
-func TestHermesEnsureInstalledUnixPromptsBeforeInstall(t *testing.T) {
-	if runtime.GOOS == "windows" {
-		t.Skip("uses POSIX shell test binaries")
-	}
-
-	tmpDir := t.TempDir()
-	setTestHome(t, tmpDir)
-	withHermesPlatform(t, "darwin")
-	withLauncherHooks(t)
-	t.Setenv("PATH", tmpDir)
-
-	writeScript := func(name, content string) {
-		t.Helper()
-		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o755); err != nil {
-			t.Fatal(err)
-		}
-	}
-
-	writeScript("curl", "#!/bin/sh\nexit 0\n")
-	writeScript("git", "#!/bin/sh\nexit 0\n")
-	writeScript("bash", fmt.Sprintf(`#!/bin/sh
-printf '%%s\n' "$*" >> %q
-/bin/cat > %q <<'EOS'
-#!/bin/sh
-exit 0
-EOS
-/bin/chmod +x %q
-exit 0
-`, filepath.Join(tmpDir, "bash.log"), filepath.Join(tmpDir, "hermes"), filepath.Join(tmpDir, "hermes")))
-
-	DefaultConfirmPrompt = func(prompt string, options ConfirmOptions) (bool, error) {
-		if prompt != "Hermes is not installed. Install now?" {
-			t.Fatalf("unexpected install prompt %q", prompt)
-		}
-		return true, nil
-	}
-
-	h := &Hermes{}
-	if err := h.ensureInstalled(); err != nil {
-		t.Fatalf("ensureInstalled returned error: %v", err)
-	}
-
-	data, err := os.ReadFile(filepath.Join(tmpDir, "bash.log"))
-	if err != nil {
-		t.Fatal(err)
-	}
-	if !strings.Contains(string(data), "--skip-setup") {
-		t.Fatalf("expected install script to skip upstream setup, got logs:\n%s", data)
-	}
-	if !strings.Contains(string(data), "-lc "+hermesInstallScript) {
-		t.Fatalf("expected official install script invocation, got logs:\n%s", data)
-	}
-}
-
-func TestHermesEnsureInstalledUnixCanBeDeclined(t *testing.T) {
-	if runtime.GOOS == "windows" {
-		t.Skip("uses POSIX shell test binaries")
-	}
-
-	tmpDir := t.TempDir()
-	setTestHome(t, tmpDir)
-	withHermesPlatform(t, "darwin")
-	withLauncherHooks(t)
-	t.Setenv("PATH", tmpDir)
-
-	for _, name := range []string{"bash", "curl", "git"} {
-		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
-			t.Fatal(err)
-		}
-	}
-
-	DefaultConfirmPrompt = func(prompt string, options ConfirmOptions) (bool, error) {
-		if prompt != "Hermes is not installed. Install now?" {
-			t.Fatalf("unexpected install prompt %q", prompt)
-		}
-		return false, nil
-	}
-
-	h := &Hermes{}
-	err := h.ensureInstalled()
-	if err == nil || !strings.Contains(err.Error(), "hermes installation cancelled") {
-		t.Fatalf("expected install cancellation error, got %v", err)
-	}
-}
-
 func TestHermesOnboardSkipsWhenLaunchConfigAlreadyMarked(t *testing.T) {
 	tmpDir := t.TempDir()
 	setTestHome(t, tmpDir)
 	withHermesPlatform(t, runtime.GOOS)
 
 	if err := config.MarkIntegrationOnboarded("hermes"); err != nil {
 		t.Fatalf("failed to mark Hermes onboarded: %v", err)
 	}
 
 	h := &Hermes{}
 	if err := h.Onboard(); err != nil {
 		t.Fatalf("expected Onboard to no-op when already marked, got %v", err)
 	}
 }
 
 func TestHermesOnboardMarksLaunchConfig(t *testing.T) {
 	tmpDir := t.TempDir()
 	setTestHome(t, tmpDir)
 	withHermesPlatform(t, runtime.GOOS)
 
 	h := &Hermes{}
 	if err := h.Onboard(); err != nil {
 		t.Fatalf("Onboard returned error: %v", err)
 	}
 


diff --git a/cmd/launch/integrations_test.go b/cmd/launch/integrations_test.go
index 8a0c03b52aba52f28d1eee1969a7176d4985d3e8..56c900955a38f2a95c21a43113ac314259528f21 100644
--- a/cmd/launch/integrations_test.go
+++ b/cmd/launch/integrations_test.go
@@ -2079,52 +2079,52 @@ func TestIntegration_Editor(t *testing.T) {
 		{"codex", false},
 		{"omp", false},
 		{"nonexistent", false},
 	}
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
-		{"hermes", true},
-		{"hermes-desktop", true},
+		{"hermes", false},
+		{"hermes-desktop", false},
 		{"cline", true},
 		{"qwen", true},
 		{"claude", true},
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


diff --git a/cmd/launch/registry.go b/cmd/launch/registry.go
index 52c93a24dde99b45ef4ff4e6dfdd18397cdc6fab..8ccaa750cd91b9950c065c9e94ceb86ce51373ee 100644
--- a/cmd/launch/registry.go
+++ b/cmd/launch/registry.go
@@ -212,67 +212,61 @@ var integrationSpecs = []*IntegrationSpec{
 				return err
 			},
 			Command: []string{"npm", "install", "-g", "@earendil-works/pi-coding-agent@latest"},
 		},
 	},
 	{
 		Name:        "pool",
 		Runner:      &Poolside{},
 		Description: "Poolside's software agent for enterprise development",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := exec.LookPath("pool")
 				return err == nil
 			},
 			URL: "https://github.com/poolsideai/pool",
 		},
 	},
 	{
 		Name:        "hermes",
 		Runner:      &Hermes{},
 		Description: "Self-improving AI agent built by Nous Research",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				return (&Hermes{}).installed()
 			},
-			EnsureInstalled: func() error {
-				return (&Hermes{}).ensureInstalled()
-			},
 			URL: "https://hermes-agent.nousresearch.com/docs/getting-started/installation/",
 		},
 	},
 	{
 		Name:        "hermes-desktop",
 		Runner:      &HermesDesktop{},
 		Description: "Desktop app for Hermes Agent by Nous Research",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				return (&Hermes{}).installed()
 			},
-			EnsureInstalled: func() error {
-				return (&Hermes{}).ensureInstalledFor("hermes-desktop")
-			},
 			URL: "https://hermes-agent.nousresearch.com/docs/getting-started/installation/",
 		},
 	},
 	{
 		Name:        "vscode",
 		Runner:      &VSCode{},
 		Aliases:     []string{"code"},
 		Description: "Microsoft's open-source AI code editor",
 		Hidden:      true,
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				return (&VSCode{}).findBinary() != ""
 			},
 			URL: "https://code.visualstudio.com",
 		},
 	},
 	{
 		Name:        "qwen",
 		Runner:      &Qwen{},
 		Description: "Qwen's AI coding agent with tool use",
 		Install: IntegrationInstallSpec{
 			CheckInstalled: func() bool {
 				_, err := (&Qwen{}).findPath()
 				return err == nil
 			},


diff --git a/docs/integrations/hermes-desktop.mdx b/docs/integrations/hermes-desktop.mdx
index 66ba6e9540b67b2778df3c3f6dddd4469786eae4..0c994427fc0583b621a58befe69c2f1622a9171a 100644
--- a/docs/integrations/hermes-desktop.mdx
+++ b/docs/integrations/hermes-desktop.mdx
@@ -1,38 +1,38 @@
 ---
 title: Hermes Desktop
 ---
 
 Hermes Desktop is a native AI assistant app by Nous Research. It provides a desktop chat interface for Hermes Agent, an AI agent that can work with models, run tools, manage projects, use memory and skills, and connect to messaging gateways.
 
 ![Hermes Desktop with Ollama](http://files.ollama.com/hermes-agent.png)
 
 ## Quick start
 
 ```bash
 ollama launch hermes-desktop
 ```
 
-Ollama handles the setup flow automatically:
+Install Hermes using the [official installation instructions](https://hermes-agent.nousresearch.com/docs/getting-started/installation/), then run the command above. Ollama handles the remaining setup flow automatically:
 
-1. **Install** - If Hermes isn't installed, Ollama prompts to install the Hermes command-line agent. On first desktop launch, Hermes builds its packaged desktop app.
+1. **Build** - On first desktop launch, Hermes builds its packaged desktop app.
 2. **Model** - Pick a model from the selector
 3. **Configure** - Ollama configures Hermes Desktop to use your selected Ollama model
 4. **Launch** - Ollama opens Hermes Desktop
 
 ## Run directly with a model
 
 ```bash
 ollama launch hermes-desktop --model <model>
 ```
 
 Run `ollama launch hermes-desktop` again to switch models later.
 
 ## Install Hermes Desktop directly
 
 On macOS and Windows, the Hermes Desktop installer is the recommended upstream installation path. It installs the desktop app and Hermes Agent together. If you prefer the command line, `ollama launch hermes-desktop` remains the explicit Ollama-managed path and uses the same Hermes configuration, sessions, skills, and memory as the CLI.
 
 To force Hermes to rebuild its packaged desktop app:
 
 ```bash
 ollama launch hermes-desktop -- --force-build
 ```


diff --git a/docs/integrations/hermes.mdx b/docs/integrations/hermes.mdx
index d4b77a207262d8bf1e720961132420d443c3fbe2..839d4862cf659fc3769ad075e8d4779e6c667c20 100644
--- a/docs/integrations/hermes.mdx
+++ b/docs/integrations/hermes.mdx
@@ -1,85 +1,80 @@
 ---
 title: Hermes Agent
 ---
 
 Hermes Agent is a self-improving AI agent built by Nous Research. It features automatic skill creation, cross-session memory, and 70+ skills that it ships with by default. 
 
 ![Hermes Agent with Ollama](/images/hermes.png)
 
 ## Quick start
 
 ```bash
 ollama launch hermes
 ```
 
-Ollama handles everything automatically:
+Install Hermes using the [official installation instructions](https://hermes-agent.nousresearch.com/docs/getting-started/installation/), then run the command above. Ollama handles the remaining setup automatically:
 
-1. **Install** — If Hermes isn't installed, Ollama prompts to install the Hermes command-line agent
-2. **Model** — Pick a model from the selector (local or cloud)
-3. **Onboarding** — Ollama configures the Ollama provider, points Hermes at `http://127.0.0.1:11434/v1`, and sets your model as the primary
-4. **Gateway** — Optionally connects a messaging platform (Telegram, Discord, Slack, WhatsApp, Signal, Email) and launches the Hermes chat
+1. **Model** — Pick a model from the selector (local or cloud)
+2. **Onboarding** — Ollama configures the Ollama provider, points Hermes at `http://127.0.0.1:11434/v1`, and sets your model as the primary
+3. **Gateway** — Optionally connects a messaging platform (Telegram, Discord, Slack, WhatsApp, Signal, Email) and launches the Hermes chat
 
 ## Recommended models
 
 **Cloud models**:
 
 - `kimi-k2.5:cloud` — Multimodal reasoning with subagents
 - `glm-5.1:cloud` — Reasoning and code generation
 - `qwen3.5:cloud` — Reasoning, coding, and agentic tool use with vision
 - `minimax-m2.7:cloud` — Fast, efficient coding and real-world productivity
 
 **Local models:**
 
 - `gemma4` — Reasoning and code generation locally (~16 GB VRAM)
 - `qwen3.6` — Reasoning, coding, and visual understanding locally (~24 GB VRAM)
 
 More models at [ollama.com/search](https://ollama.com/search?c=cloud).
 
 ## Connect messaging apps
 
 Link Telegram, Discord, Slack, WhatsApp, Signal, or Email to chat with your models from anywhere:
 
 ```bash
 hermes gateway setup
 ```
 
 ## Reconfigure
 
 Use Hermes's model picker to change providers or models later:
 
 ```bash
 hermes model
 ```
 
 ## Manual setup
 
-If you'd rather drive Hermes's own wizard instead of `ollama launch hermes`, install it directly:
-
-```bash
-curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
-```
+If you'd rather drive Hermes's own wizard instead of `ollama launch hermes`, follow the [official installation instructions](https://hermes-agent.nousresearch.com/docs/getting-started/installation/) and launch Hermes directly.
 
 Hermes launches the setup wizard automatically. Choose **Quick setup**:
 
 ```
 How would you like to set up Hermes?
 
  →  Quick setup — provider, model & messaging (recommended)
     Full setup — configure everything
 ```
 
 ### Connect to Ollama
 
 1. Select **More providers...**
 2. Select **Custom endpoint (enter URL manually)**
 3. Set the API base URL to the Ollama OpenAI-compatible endpoint:
 
    ```
    API base URL [e.g. https://api.example.com/v1]: http://127.0.0.1:11434/v1
    ```
 
 4. Leave the API key blank (not required for local Ollama):
 
    ```
    API key [optional]:
    ```

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Keeping the original high severity. The vulnerable code is in-scope shipped CLI functionality, not tests or docs. Evidence shows an executable sink: a mutable third-party `main` branch installer is piped to `bash` without any checksum, signature, release tag, or pinned commit verification, and both Unix and Windows/WSL install paths execute it. The launch command is registered in the main CLI and Hermes has an auto-install hook, so the issue is reachable through normal `ollama launch hermes` use. Impact is high because a compromised upstream script yields arbitrary code execution as the local/WSL user. It is not critical because exploitation is not unauthenticated against a listening service, is not default 0-click, and requires either user install interaction or `--yes` plus compromise/control of the upstream script or delivery path.
## Likelihood
high - The path is reachable during normal use of the shipped CLI and can be auto-approved with `--yes`, but an attacker generally must compromise/control the upstream GitHub repository/account or trusted delivery path, and the user must launch/install Hermes. HTTPS prevents ordinary unauthenticated network injection. | Remote network vector
## Impact
high - Successful exploitation executes arbitrary shell code returned by the remote installer as the user running Ollama, with access to that user's files, environment, model cache, credentials, and WSL environment on Windows.
## Assumptions
- The shipped Ollama CLI includes the new `ollama launch hermes` integration path.
- The user has not already installed Hermes, so the auto-install path is reached.
- Required local dependencies such as bash, curl, and git are present.
- A realistic attacker would need to compromise or control the upstream NousResearch/hermes-agent script source, a relevant GitHub account/repository path, or the trusted delivery path; ordinary network attackers are constrained by HTTPS.
- Execution occurs with the privileges of the local Ollama/CLI user or the WSL user on Windows, not as a remote Ollama server process by default.
- Victim runs the Ollama launch command for the Hermes integration or selects Hermes through the launcher flow.
- Hermes is missing so installation is attempted.
- Victim approves the generic install prompt, or runs with `--yes` so confirmation is auto-approved.
- The remote install script at the mutable GitHub `main` branch is malicious or compromised.
## Path
Attacker-controlled upstream script
  -> Ollama hermesInstallScript uses raw.githubusercontent.com/.../main/.../install.sh
  -> `ollama launch hermes` missing-install flow prompts or `--yes` auto-approves
  -> `bash -lc "curl ... | bash -s -- --skip-setup"`
  -> arbitrary commands run as local/WSL user
## Path evidence
- `cmd/launch/hermes.go:29` - Defines the installer as `curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash -s -- --skip-setup`, using a mutable `main` branch and no visible integrity verification.
- `cmd/launch/hermes.go:221-230` - Unix missing-install path asks only a generic install confirmation and then executes the installer with `bash -lc`.
- `cmd/launch/hermes.go:255-267` - Windows/WSL missing-install path uses the same unverified remote installer through `h.runWSL("bash", "-lc", hermesInstallScript)`.
- `cmd/launch/launch.go:83-86` - `--yes` launch policy maps confirmations to `LaunchConfirmAutoApprove`.
- `cmd/launch/selector_hooks.go:79-81` - The shared confirmation hook returns true without prompting when the current confirmation policy has `yes` set.
- `cmd/launch/registry.go:140-149` - Hermes is a registered launch integration with an auto-install `EnsureInstalled` hook.
- `cmd/launch/launch.go:337-363` - The launch flow calls `EnsureIntegrationInstalled` for integrations, making the auto-install path reachable from normal `ollama launch` usage.
- `cmd/cmd.go:2328-2345` - The launch command is registered in the main Ollama root CLI command, showing this is shipped product code.
## Narrative
The finding is real and reachable. `cmd/launch/hermes.go` defines the Hermes installer as a curl-to-bash pipeline from `raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh`, which is a mutable branch path and is executed without checksum, signature, pinned commit, or release verification. The Unix missing-install path runs it via `bash -lc`; the Windows path runs the same string inside WSL. The main CLI registers `launch.LaunchCmd`, and Hermes is listed as a supported integration. The `--yes` flag changes the launch policy to auto-approve confirmations, and the confirmation hook returns true when that policy is active. Impact is arbitrary code execution as the local/WSL user if the upstream script source or delivery path is compromised. Likelihood is not critical-level because exploitation requires user launch/install interaction and compromise/control of the upstream script source rather than direct unauthenticated reachability to the Ollama daemon.
## Controls
- Generic user confirmation prompt before install in interactive mode.
- Non-interactive launches require `--yes` and, for headless `--yes`, a model override.
- HTTPS is used for the GitHub raw URL.
- No checksum verification found for the Hermes installer.
- No signature verification found for the Hermes installer.
- No immutable release tag or commit pin found for the Hermes installer.
- No sandboxing around the shell execution sink.
## Blindspots
- Static review did not inspect the live upstream Hermes repository or its release/security controls.
- No cloud or GitHub APIs were called, so upstream branch mutability and current script content were inferred from the repository URL pattern and local code only.
- The exact installed-user privilege level depends on how the victim runs the Ollama CLI and their WSL configuration.
- Repository search was targeted to the Hermes launch path and did not exhaustively review all integration installers.