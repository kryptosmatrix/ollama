Docs recommend exposing unauthenticated Ollama API
Link: https://chatgpt.com/codex/cloud/security/findings/e2cc3364eaac81918fbb599c93bd437a?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: 22c2bdb
Author: brucewmacdonald@gmail.com
Created: 9/8/2026, 3:56:15 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced: unsafe documentation that encourages binding Ollama's unauthenticated API to all interfaces for NemoClaw/container usage. The server behavior that makes this dangerous already existed, but the newly added documented recommendation creates a new user-facing insecure configuration path.
Ollama's server is intentionally unauthenticated and local-by-default. This commit adds a public integration page that tells users running inside WSL2 or containers to make Ollama reachable using `OLLAMA_HOST=0.0.0.0`. In this codebase, `OLLAMA_HOST` directly controls the address passed to `net.Listen`; when it is set to `0.0.0.0`, the daemon listens on all interfaces. The Host-header protection is only applied for loopback listeners and is skipped when the server is bound to a non-loopback/unspecified address. As a result, users following the new documentation can expose model management and inference endpoints such as pull, push, delete, create, generate, and chat to any network that can reach the host unless they independently add firewall or reverse-proxy authentication. The underlying unauthenticated behavior is pre-existing and by design, but this commit newly documents an unsafe exposure pattern without a security warning or safer container-specific alternative.

# Validation
## Rubric
- [x] The commit newly adds/publishes NemoClaw documentation that recommends `OLLAMA_HOST=0.0.0.0` for WSL2/container reachability.
- [x] `OLLAMA_HOST` is parsed by the real envconfig code and the documented value is preserved as the server bind host rather than converted to loopback.
- [x] The server bind path passes `envconfig.Host().Host` directly to `net.Listen`, and dynamic validation shows wildcard bind is reachable via a non-loopback container IP while loopback is not.
- [x] The Host-header protection is skipped for non-loopback/unspecified listener addresses, as shown by server/routes.go and a minimal branch reproduction.
- [x] Sensitive model-management and inference endpoints are registered without a global authentication middleware, so network reachability implies unauthenticated remote access to those routes.
## Report
Validated the finding as a documentation-induced insecure exposure path. Commit 22c2bdbd adds only docs/docs.json and docs/integrations/nemoclaw.mdx. docs/integrations/nemoclaw.mdx:49 tells WSL2/container users to make Ollama reachable with `OLLAMA_HOST=0.0.0.0`, and docs/docs.json:160-166 publishes the page under Assistant Sandboxing. Code review of the relevant server path confirms `envconfig.Host()` defaults to loopback but preserves `OLLAMA_HOST` values: envconfig/config.go:20-22,25-57. The daemon binds directly to that host with `net.Listen("tcp", envconfig.Host().Host)` in cmd/cmd.go:1821-1827. I built and ran a small PoC importing the real `github.com/ollama/ollama/envconfig` package and using the same `net.Listen` call. Output showed: `OLLAMA_HOST="0.0.0.0:0" envconfig.Host().Host="0.0.0.0:0" listener.Addr="[::]:33727" ... DIAL_NON_LOOPBACK=success`, while the loopback control `OLLAMA_HOST=127.0.0.1:0` produced `DIAL_NON_LOOPBACK=failed: connect: connection refused`. This demonstrates the documented value makes the service reachable on the container's non-loopback interface. The Host-header protection is skipped for non-loopback/unspecified listener addresses at server/routes.go:1593-1596; a minimal reproduction of that exact branch produced `bind=0.0.0.0:0 listener.Addr=[::]:41395 ... first_branch_cNext=true` and `bind=127.0.0.1:0 ... first_branch_cNext=false`. Route setup uses only CORS and `allowedHostsMiddleware` globally at server/routes.go:1652-1657, then registers management/inference routes such as `/api/pull`, `/api/delete`, `/api/create`, `/api/generate`, `/api/chat`, and OpenAI-compatible endpoints at server/routes.go:1666-1703 without a general authentication middleware. I attempted a debug build of the full target first, but it could not complete because the container cannot fetch `github.com/dlclark/regexp2` from the Go proxy. Valgrind is not installed. I then used LLDB non-interactively on the debug-symbol PoC; LLDB output includes the PoC's wildcard bind/non-loopback dial success and a backtrace stopped at `main.main` after listener creation.

# Evidence
cmd/cmd.go (L1821 to 1827)
  Note: The server binds directly to envconfig.Host().Host, so OLLAMA_HOST=0.0.0.0 causes the daemon to listen on all interfaces.
```
func RunServer(_ *cobra.Command, _ []string) error {
	if err := initializeKeypair(); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", envconfig.Host().Host)
	if err != nil {
```

docs/docs.json (L160 to 166)
  Note: The commit publishes the new NemoClaw integration page in the documentation navigation, making the unsafe guidance discoverable.
```
            "group": "More information",
            "pages": [
              "/cli",
              {
                "group": "Assistant Sandboxing",
                "pages": [
                  "/integrations/nemoclaw"
```

docs/integrations/nemoclaw.mdx (L47 to 49)
  Note: The new documentation recommends using OLLAMA_HOST=0.0.0.0 to make Ollama reachable from WSL2/container sandboxes, without warning that this exposes the unauthenticated API on all interfaces.
```
CMD and PowerShell are not supported on Windows — WSL2 is required.

<Note>Ollama must be installed and running before the installer runs. When running inside WSL2 or a container, ensure Ollama is reachable from the sandbox (e.g. `OLLAMA_HOST=0.0.0.0`).</Note>
```

envconfig/config.go (L20 to 22)
  Note: OLLAMA_HOST is the environment variable controlling the server host; the default is loopback, but the documented value overrides that local-only default.
```
// Host returns the scheme and host. Host can be configured via the OLLAMA_HOST environment variable.
// Default is scheme "http" and host "127.0.0.1:11434"
func Host() *url.URL {
```

server/routes.go (L1593 to 1596)
  Note: The Host-header protection is bypassed for non-loopback listeners, so binding to all interfaces removes the loopback DNS-rebinding safeguard.
```
		if addr, err := netip.ParseAddrPort(addr.String()); err == nil && !addr.Addr().IsLoopback() {
			c.Next()
			return
		}
```

server/routes.go (L1666 to 1703)
  Note: These model-management and inference routes are registered without general server-side authentication, so a network-reachable daemon permits unauthenticated remote actions.
```
	// Local model cache management (new implementation is at end of function)
	r.POST("/api/pull", s.PullHandler)
	r.POST("/api/push", s.PushHandler)
	r.HEAD("/api/tags", s.ListHandler)
	r.GET("/api/tags", s.ListHandler)
	r.POST("/api/show", s.ShowHandler)
	r.DELETE("/api/delete", s.DeleteHandler)

	r.POST("/api/me", s.WhoamiHandler)

	r.POST("/api/signout", s.SignoutHandler)
	// deprecated
	r.DELETE("/api/user/keys/:encodedKey", s.SignoutHandler)

	// Create
	r.POST("/api/create", s.CreateHandler)
	r.POST("/api/blobs/:digest", s.CreateBlobHandler)
	r.HEAD("/api/blobs/:digest", s.HeadBlobHandler)
	r.POST("/api/copy", s.CopyHandler)
	r.POST("/api/experimental/web_search", s.WebSearchExperimentalHandler)
	r.POST("/api/experimental/web_fetch", s.WebFetchExperimentalHandler)

	// Inference
	r.GET("/api/ps", s.PsHandler)
	r.POST("/api/generate", s.withInferenceRequestLogging("/api/generate", s.GenerateHandler)...)
	r.POST("/api/chat", s.withInferenceRequestLogging("/api/chat", s.ChatHandler)...)
	r.POST("/api/embed", s.EmbedHandler)
	r.POST("/api/embeddings", s.EmbeddingsHandler)

	// Inference (OpenAI compatibility)
	// TODO(cloud-stage-a): apply Modelfile overlay deltas for local models with cloud
	// parents on v1 request families while preserving this explicit :cloud passthrough.
	r.POST("/v1/chat/completions", s.withInferenceRequestLogging("/v1/chat/completions", cloudPassthroughMiddleware(cloudErrRemoteInferenceUnavailable), middleware.ChatMiddleware(), s.ChatHandler)...)
	r.POST("/v1/completions", s.withInferenceRequestLogging("/v1/completions", cloudPassthroughMiddleware(cloudErrRemoteInferenceUnavailable), middleware.CompletionsMiddleware(), s.GenerateHandler)...)
	r.POST("/v1/embeddings", cloudPassthroughMiddleware(cloudErrRemoteInferenceUnavailable), middleware.EmbeddingsMiddleware(), s.EmbedHandler)
	r.GET("/v1/models", middleware.ListMiddleware(), s.ListHandler)
	r.GET("/v1/models/:model", cloudModelPathPassthroughMiddleware(cloudErrRemoteModelDetailsUnavailable), middleware.RetrieveMiddleware(), s.ShowHandler)
	r.POST("/v1/responses", s.withInferenceRequestLogging("/v1/responses", cloudPassthroughMiddleware(cloudErrRemoteInferenceUnavailable), middleware.ResponsesMiddleware(), s.ChatHandler)...)
```

Proposed patch:
diff --git a/docs/integrations/nemoclaw.mdx b/docs/integrations/nemoclaw.mdx
index 0aee01a04000113c184c09629ffb54bddb4ed0a5..af43d7f0dd863824c7e2648ee245f47dac763479 100644
--- a/docs/integrations/nemoclaw.mdx
+++ b/docs/integrations/nemoclaw.mdx
@@ -24,44 +24,46 @@ curl -fsSL https://www.nvidia.com/nemoclaw.sh | \
 
 Connect to your sandbox:
 
 ```bash
 nemoclaw my-assistant connect
 ```
 
 Open the TUI:
 
 ```bash
 openclaw tui
 ```
 
 <Note>Ollama support in NemoClaw is still experimental.</Note>
 
 ## Platform support
 
 | Platform | Runtime | Status |
 |----------|---------|--------|
 | Linux (Ubuntu 22.04+) | Docker | Primary |
 | macOS (Apple Silicon) | Colima or Docker Desktop | Supported |
 | Windows | WSL2 with Docker Desktop | Supported |
 
 CMD and PowerShell are not supported on Windows — WSL2 is required.
 
-<Note>Ollama must be installed and running before the installer runs. When running inside WSL2 or a container, ensure Ollama is reachable from the sandbox (e.g. `OLLAMA_HOST=0.0.0.0`).</Note>
+<Note>Ollama must be installed and running before the installer runs. When running inside WSL2 or a container, ensure Ollama is reachable from the sandbox.</Note>
+
+<Warning>Ollama's local API does not require authentication. Keep the default loopback binding and use host networking when possible. If you must set `OLLAMA_HOST=0.0.0.0`, restrict port `11434` to trusted clients with a firewall or an authenticated reverse proxy; otherwise, anyone who can reach the host can use the API.</Warning>
 
 ## System requirements
 
 - CPU: 4 vCPU minimum
 - RAM: 8 GB minimum (16 GB recommended)
 - Disk: 20 GB free (40 GB recommended for local models)
 - Node.js 20+ and npm 10+
 - Container runtime (Docker preferred)
 
 ## Recommended models
 
 - `nemotron-3-super:cloud` — Strong reasoning and coding
 - `qwen3.5:cloud` — 397B; reasoning and code generation
 - `nemotron-3-nano:30b` — Recommended local model; fits in 24 GB VRAM
 - `qwen3.5:27b` — Fast local reasoning (~18 GB VRAM)
 - `glm-4.7-flash` — Reasoning and code generation (~25 GB VRAM)
 
 More models at [ollama.com/search](https://ollama.com/search).

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Retained high severity because the attack path is evidenced in main product code and official documentation: the new docs recommend OLLAMA_HOST=0.0.0.0, envconfig preserves that value, RunServer binds it with net.Listen, Host-header loopback protection is bypassed for non-loopback listeners, and sensitive model-management/inference routes are registered without general authentication. The issue is not critical because it requires operator action and network reachability and does not prove code execution, cross-tenant compromise, or direct crown-jewel secret theft. The presence of earlier FAQ examples using 0.0.0.0 means this commit broadens an existing unsafe documentation pattern rather than introducing the underlying unauthenticated all-interface behavior, but the new integration page still creates a realistic insecure configuration path for WSL2/container users.
## Likelihood
high - The unsafe configuration is explicitly documented and technically straightforward, and no credentials are needed after exposure. However, exploitation is not default: it requires the operator to apply the documented setting and for the listener to be reachable through host/container/WSL networking or firewall rules. | Remote network vector
## Impact
high - A reachable unauthenticated Ollama daemon allows remote state-changing model-cache operations and resource-intensive inference/pull actions. This can delete or replace local models, fill disk or consume bandwidth, and monopolize CPU/GPU/memory. Confidentiality impact is lower, mainly model inventory/details and unauthorized inference access, with no direct RCE or secret exfiltration proven.
## Assumptions
- An operator follows the newly added NemoClaw documentation and configures Ollama with OLLAMA_HOST=0.0.0.0 or equivalent.
- The host, WSL2 environment, container bridge, published container port, firewall, VPN, or LAN routing allows another machine or untrusted sandbox to reach TCP port 11434.
- The Ollama API remains intentionally unauthenticated unless the operator adds an external reverse proxy, firewall, or authentication layer.
- User/operator applies documented OLLAMA_HOST=0.0.0.0 configuration
- Network path exists to the Ollama listener
- No external firewall or authenticated reverse proxy blocks access
## Path
Docs recommend OLLAMA_HOST=0.0.0.0
  -> operator applies config
  -> envconfig.Host() feeds net.Listen()
  -> Ollama listens on all interfaces:11434
  -> Host check skipped for non-loopback bind
  -> unauthenticated /api/delete,/api/pull,/api/create,/api/generate,/api/chat
  -> remote model tampering and resource exhaustion
## Path evidence
- `docs/integrations/nemoclaw.mdx:49` - New public integration note recommends OLLAMA_HOST=0.0.0.0 for WSL2/container reachability without an authentication or firewall warning.
- `docs/docs.json:160-167` - The commit publishes the NemoClaw page in the documentation navigation under Assistant Sandboxing.
- `envconfig/config.go:20-58` - OLLAMA_HOST controls the host value, defaults to 127.0.0.1, and preserves operator-supplied IP/host values such as 0.0.0.0.
- `cmd/cmd.go:1821-1827` - The daemon binds directly to envconfig.Host().Host via net.Listen.
- `server/routes.go:1593-1596` - The Host-header protection short-circuits and allows requests when the listener address is non-loopback.
- `server/routes.go:1652-1657` - Global middleware shown is CORS plus allowedHostsMiddleware; no general authentication middleware is applied here.
- `server/routes.go:1666-1703` - Model-management and inference endpoints are registered, including pull, push, delete, create, generate, chat, embeddings, and OpenAI-compatible endpoints.
- `docs/faq.mdx:75-101` - Existing documentation already showed OLLAMA_HOST=0.0.0.0:11434 examples, indicating the commit extends an existing unsafe documentation pattern rather than changing the daemon's underlying design.
## Narrative
The finding is real and reachable under a plausible documented configuration. The new NemoClaw integration page tells WSL2/container users to make Ollama reachable with OLLAMA_HOST=0.0.0.0. The envconfig code preserves that host value and RunServer passes it directly to net.Listen. The server's loopback Host-header protection is skipped for non-loopback listener addresses, and the global router setup shows only CORS and allowedHostsMiddleware before registering unauthenticated model-management and inference routes. This is not a default internet exposure, and it requires user/operator action plus network reachability, but following the official guidance can expose a powerful unauthenticated local model service. Existing FAQ material also already documents 0.0.0.0 for network exposure, so this commit broadens and republishes an existing unsafe pattern rather than introducing the underlying server behavior.
## Controls
- Default bind address is 127.0.0.1:11434 when OLLAMA_HOST is not changed
- CORS middleware is configured, but it does not protect non-browser clients
- allowedHostsMiddleware provides loopback DNS-rebinding protection only while bound to loopback; it is skipped for non-loopback listener addresses
- No general server-side authentication or authorization is evident on the listed model-management and inference routes
- External firewall, VPN, or authenticated reverse proxy is required to safely expose the service
## Blindspots
- Static repository review cannot determine whether any given user host has a public IP, permissive firewall, published container port, or LAN/VPN exposure.
- No cloud APIs were called and no live deployment was inspected.
- The commit is documentation-only; the vulnerable server behavior pre-existed.
- Full runtime build was not repeated in this stage; prior validation evidence included an executable PoC for the bind behavior and middleware branch.
- Actual damage from endpoints depends on installed models, available disk/GPU resources, and whether registry/cloud credentials are configured.