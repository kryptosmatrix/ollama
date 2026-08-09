PR-controlled version file can inject CI outputs
Link: https://chatgpt.com/codex/cloud/security/findings/aaa1600fcc8c8191887a87330186cb19?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: 6308826
Author: dhiltgen@users.noreply.github.com
Created: 9/8/2026, 2:20:01 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced a CI command-injection vulnerability in the newly added pull_request workflow. The vulnerable sink is the unescaped write of PR-controlled LLAMA_CPP_VERSION contents to GITHUB_OUTPUT; the injected output is then trusted as GOFLAGS in subsequent build jobs.
The added test-llamacpp-update workflow is triggered by pull requests that modify LLAMA_CPP_VERSION. In setup-environment, the workflow appends `vendorsha=$(cat LLAMA_CPP_VERSION)-...` directly to GITHUB_OUTPUT using the single-line output format. Because LLAMA_CPP_VERSION is PR-controlled and is not validated or encoded with the multiline-safe GITHUB_OUTPUT syntax, embedded newlines can create additional output records. An attacker can include a line such as `GOFLAGS='-toolexec=/bin/sh -c "..."'` in LLAMA_CPP_VERSION, followed by another valid `KEY=value` line to keep the output file parseable. This can override the earlier GOFLAGS output. The overridden GOFLAGS is then propagated into later build jobs and Docker build arguments, where Go will honor flags such as `-toolexec`, resulting in arbitrary command execution on the workflow runners. The workflow only grants contents:read, but several jobs run on `linux`, `linux-arm64`, and `windows` labels that may correspond to reusable/self-hosted release infrastructure; compromise of those runners can be security significant even if the artifacts are explicitly unsigned.

# Validation
## Rubric
- [x] Confirm the vulnerable workflow is present/introduced and is reachable from pull requests changing LLAMA_CPP_VERSION.
- [x] Confirm PR-controlled LLAMA_CPP_VERSION is written to GITHUB_OUTPUT using unsafe single-line output syntax without validation or multiline encoding.
- [x] Demonstrate that embedded newlines can inject a later GOFLAGS output while keeping the output file parseable.
- [x] Confirm downstream build jobs trust the setup job's GOFLAGS output and pass it to Go builds/Docker build args.
- [x] Demonstrate that the injected GOFLAGS value can cause command execution through Go -toolexec during the actual repository build path.
## Report
Validated the finding with targeted workflow inspection plus a dynamic PoC. Memory-crash/valgrind/debugger validation is not applicable because the vulnerable component is GitHub Actions YAML/output parsing rather than native memory handling.

Evidence:
1. The workflow was newly added in this commit: `git show --name-status HEAD` shows `A .github/workflows/test-llamacpp-update.yaml`.
2. `.github/workflows/test-llamacpp-update.yaml:7-10` triggers on `pull_request` changes to `LLAMA_CPP_VERSION`, so a PR can cause this workflow to run when that file is changed.
3. `.github/workflows/test-llamacpp-update.yaml:31-38` writes outputs using single-line `KEY=value` syntax, including `echo "vendorsha=$(cat LLAMA_CPP_VERSION)-$(cat MLX_VERSION)-$(cat MLX_C_VERSION)"`. A newline in `LLAMA_CPP_VERSION` therefore creates additional output records.
4. `.github/workflows/test-llamacpp-update.yaml:43-60`, `:137-155`, and `:449-453` trust `needs.setup-environment.outputs.GOFLAGS` in later jobs. `Dockerfile:248-255` accepts `ARG GOFLAGS` and then runs `go build`, where Go honors the `GOFLAGS` environment/build arg.
5. The PoC used a malicious `LLAMA_CPP_VERSION` payload containing:
   `attacker-controlled-version\nGOFLAGS=-toolexec=./.ci-poc-toolexec.sh\nTAIL=absorbs-suffix`.
   Running the exact vulnerable output block produced:
   `vendorsha=attacker-controlled-version`, then a later injected `GOFLAGS=-toolexec=./.ci-poc-toolexec.sh`, then `TAIL=absorbs-suffix-...`, keeping the output file parseable while overriding the earlier legitimate `GOFLAGS`.
6. The PoC then invoked the repository's real Go build path with the parsed injected `GOFLAGS`. Output showed command execution via Go `-toolexec`:
   `toolexec executed: tool=/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-amd64/pkg/tool/linux_amd64/compile pwd=/workspace/ollama`
   followed by `go: error obtaining buildID for go tool compile: exit status 42`, where exit 42 was deliberately returned by the attacker-controlled wrapper.

Conclusion: the suspected CI command-injection vulnerability is valid. A PR-controlled multiline `LLAMA_CPP_VERSION` can inject/override the `GOFLAGS` workflow output, and the injected `GOFLAGS` can execute attacker-controlled code during later Go builds.

# Evidence
github/workflows/test-llamacpp-update.yaml (L137 to 155)
  Note: The Linux Go build passes the injected GOFLAGS output into the Docker build, where Go will honor dangerous flags such as -toolexec during `go build`.

github/workflows/test-llamacpp-update.yaml (L31 to 38)
  Note: The setup step writes `cat LLAMA_CPP_VERSION` directly into GITHUB_OUTPUT using the unsafe single-line format. Embedded newlines in LLAMA_CPP_VERSION can inject additional outputs, including overriding GOFLAGS.

github/workflows/test-llamacpp-update.yaml (L43 to 60)
  Note: The downstream Darwin build trusts the setup job's GOFLAGS output as an environment variable before invoking the build script.

github/workflows/test-llamacpp-update.yaml (L449 to 453)
  Note: The Windows build also trusts the attacker-injectable GOFLAGS output on a reusable `windows` runner label.

github/workflows/test-llamacpp-update.yaml (L7 to 10)
  Note: The workflow is triggered by pull requests that modify LLAMA_CPP_VERSION, making that file attacker-controlled in this workflow context.

Proposed patch:
diff --git a/.github/workflows/test-llamacpp-update.yaml b/.github/workflows/test-llamacpp-update.yaml
index 004791c0a0909f41e1890eafed2bc8113aea1046..c87e75431dbb5dd0c2eb482aab898afc3ff8ad18 100644
--- a/.github/workflows/test-llamacpp-update.yaml
+++ b/.github/workflows/test-llamacpp-update.yaml
@@ -9,54 +9,59 @@ on:
     paths:
       - 'LLAMA_CPP_VERSION'
 
 permissions:
   contents: read
 
 env:
   CGO_CFLAGS: '-O3'
   CGO_CXXFLAGS: '-O3'
 
 jobs:
   setup-environment:
     runs-on: ubuntu-latest
     outputs:
       GOFLAGS: ${{ steps.goflags.outputs.GOFLAGS }}
       VERSION: ${{ steps.goflags.outputs.VERSION }}
       vendorsha: ${{ steps.goflags.outputs.vendorsha }}
     steps:
       - uses: actions/checkout@v4
       - name: Set environment
         id: goflags
         shell: bash
         run: |
           set -euo pipefail
           VERSION="0.0.0-llamacpp-${GITHUB_SHA::7}"
+          llama_cpp_version=$(<LLAMA_CPP_VERSION)
+          if [[ ! "${llama_cpp_version}" =~ ^b[0-9]+$ ]]; then
+            echo "LLAMA_CPP_VERSION must contain a single llama.cpp build tag" >&2
+            exit 1
+          fi
           {
             echo "GOFLAGS='-ldflags=-w -s \"-X=github.com/ollama/ollama/version.Version=${VERSION}\" \"-X=github.com/ollama/ollama/server.mode=release\"'"
             echo "VERSION=${VERSION}"
-            echo "vendorsha=$(cat LLAMA_CPP_VERSION)-$(cat MLX_VERSION)-$(cat MLX_C_VERSION)"
+            echo "vendorsha=${llama_cpp_version}-$(cat MLX_VERSION)-$(cat MLX_C_VERSION)"
           } >>"${GITHUB_OUTPUT}"
 
   darwin-build:
     runs-on: macos-26-xlarge
     needs: setup-environment
     env:
       GOFLAGS: ${{ needs.setup-environment.outputs.GOFLAGS }}
       VERSION: ${{ needs.setup-environment.outputs.VERSION }}
       CGO_CFLAGS: '-mmacosx-version-min=14.0 -O3'
       CGO_CXXFLAGS: '-mmacosx-version-min=14.0 -O3'
       CGO_LDFLAGS: '-mmacosx-version-min=14.0 -O3'
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-go@v5
         with:
           go-version-file: go.mod
           cache-dependency-path: |
             go.sum
             LLAMA_CPP_VERSION
             MLX_VERSION
             MLX_C_VERSION
       - name: Build unsigned Darwin runtime
         run: ./scripts/build_darwin.sh build package
       - name: Log build results
         run: ls -l dist/

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Severity remains high, but not critical. The exploit path is concrete and validated: a PR-controlled version file is written unsafely to GITHUB_OUTPUT and can override GOFLAGS, which later reaches Go build executable sinks. This is remote PR-reachable CI command execution. High impact is justified by the use of multiple non-standard/custom runner labels that plausibly represent reusable build infrastructure. It is not raised to critical because the workflow grants only contents: read, the artifacts are explicitly unsigned and not release artifacts, and static repository evidence cannot prove access to signing keys, repository write credentials, or persistent runner compromise. If all runners are ephemeral isolated hosted runners, this would likely drop to medium.
## Likelihood
high - The input is directly PR-controlled and the vulnerable output write is straightforward to reach. Exploitation may require workflow approval for forked PRs and depends on downstream jobs running, so likelihood is medium rather than high. | Remote network vector
## Impact
high - Successful exploitation gives arbitrary command execution during CI builds. On reusable/self-hosted or release-adjacent runner labels this can expose runner state, caches, workspace contents, network access, and any local credentials. Direct repository write or signed release compromise is not evidenced because GITHUB_TOKEN is read-only and artifacts are unsigned PR-validation outputs.
## Assumptions
- An attacker can submit a pull request that changes LLAMA_CPP_VERSION and the workflow is allowed to run after any required GitHub/repository approval.
- GitHub Actions output parsing accepts the injected newline-delimited records in GITHUB_OUTPUT; the validation evidence demonstrated this behavior.
- The non-standard runner labels such as linux, linux-arm64, windows, and macos-26-xlarge represent reusable/custom build infrastructure or otherwise security-relevant CI runners; if all runners are fully ephemeral and isolated GitHub-hosted runners with no sensitive network/cache/state, impact is lower.
- Ability to open a pull request modifying LLAMA_CPP_VERSION
- Workflow execution for the pull request
- Downstream build job reaches a Go build path using the injected GOFLAGS output
## Path
Attacker PR
  -> LLAMA_CPP_VERSION with newline output injection
  -> setup-environment writes unsafe GITHUB_OUTPUT
  -> GOFLAGS output override
  -> darwin/linux/windows build jobs
  -> go build honors -toolexec
  -> command execution on CI runner
## Path evidence
- `.github/workflows/test-llamacpp-update.yaml:7-13` - Workflow is remotely reachable through pull_request events that change LLAMA_CPP_VERSION; token permissions are limited to contents: read.
- `.github/workflows/test-llamacpp-update.yaml:31-38` - setup-environment writes attacker-controlled LLAMA_CPP_VERSION content into GITHUB_OUTPUT using unsafe single-line output syntax.
- `.github/workflows/test-llamacpp-update.yaml:40-60` - Darwin build job trusts the setup job GOFLAGS output as an environment variable before invoking the build script.
- `.github/workflows/test-llamacpp-update.yaml:137-155` - Linux Go build passes the injected GOFLAGS output into Docker build arguments.
- `Dockerfile:248-255` - Docker build accepts ARG GOFLAGS and then runs go build, where dangerous Go flags such as -toolexec are honored.
- `.github/workflows/test-llamacpp-update.yaml:449-453` - Windows build job runs on a non-standard windows runner label and trusts the attacker-injectable GOFLAGS output.
- `.github/workflows/test-llamacpp-update.yaml:3-5` - Workflow comments state PR validation artifacts are unsigned, not notarized, and not intended to be published as release artifacts, limiting direct release impact.
## Narrative
The finding is real. The new workflow triggers on pull_request changes to LLAMA_CPP_VERSION, then writes vendorsha=$(cat LLAMA_CPP_VERSION)-... directly to GITHUB_OUTPUT. Because that file is PR-controlled and not constrained to a single line, embedded newlines can create additional output records, including a later GOFLAGS output. Multiple downstream jobs trust needs.setup-environment.outputs.GOFLAGS as an environment variable or Docker build argument, and Dockerfile publish-go runs go build with that argument in scope. The prior executable PoC demonstrated Go -toolexec command execution. The main limitation is impact: the workflow grants only contents: read and states artifacts are unsigned/not release artifacts, so direct repository or user release compromise is not shown. Severity remains high primarily because this is remote PR-reachable command execution on several non-standard/custom CI runner labels that may be reusable build infrastructure.
## Controls
- GitHub Actions permissions set to contents: read
- PR validation artifacts explicitly marked unsigned and not notarized
- No Docker registry push credentials evident in the workflow
- Path filter limits automatic trigger to PRs changing LLAMA_CPP_VERSION, but does not validate file contents
- Linux build execution occurs through Docker build; Darwin and Windows jobs execute build scripts directly on runners
## Blindspots
- Static repository review cannot verify GitHub Actions fork-PR approval settings.
- Static repository review cannot determine whether linux, linux-arm64, windows, and macos-26-xlarge labels are self-hosted, larger hosted, ephemeral, or reused across release workloads.
- No cloud/API calls were made, so runner inventory, attached secrets, network reachability, and cache persistence were not verified.
- Repository artifacts do not prove whether PR validation artifacts can ever be promoted into release artifacts.