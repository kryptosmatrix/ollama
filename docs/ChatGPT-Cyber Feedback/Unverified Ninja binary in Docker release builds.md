Unverified Ninja binary in Docker release builds
Link: https://chatgpt.com/codex/cloud/security/findings/ec5a04193c048191aea6cf561bcb8614?repo=https%3A%2F%2Fgithub.com%2Fkryptosmatrix%2Follama&sev=high
Criticality: high (attack path: high)
Status: new

# Metadata
Repo: kryptosmatrix/ollama
Commit: 288077c
Author: dhiltgen@users.noreply.github.com
Created: 9/8/2026, 4:33:06 pm
Assignee: Unassigned
Signals: Security, Validated, Patch generated, Attack-path

# Summary
Introduced a supply-chain integrity gap for a new build executable. The repository already had some other unverified build-tool downloads, but this commit specifically adds Ninja as a mandatory unverified executable used by the Docker CMake builds.
The Dockerfile now fetches Ninja binaries directly from github.com/ninja-build/ninja release assets, unzips them into /usr/local/bin, and sets CMAKE_GENERATOR=Ninja. The downloaded executable is then used by CMake build stages that create the Ollama binary/native runner artifacts later copied into Linux release bundles and Docker images. Unlike the previous amd64 path that installed ninja-build through the signed distribution package manager, the new ZIP downloads are not pinned by digest and are not verified with a checksum, signature, or provenance check. If the GitHub release asset/CDN/TLS trust path/build cache were compromised, the malicious Ninja binary would execute during the release Docker build and could backdoor shipped artifacts or leak build-time data. This is a supply-chain issue rather than a runtime API vulnerability; exploitation requires compromising the build dependency delivery path.

# Validation
## Rubric
- [x] Confirm the commit actually introduced Ninja as a new build dependency/download path in Dockerfile.
- [x] Confirm the Ninja executable is fetched from a remote release asset and installed without checksum, signature, pinned digest, or provenance verification.
- [x] Confirm the downloaded executable is mandatory/used by CMake via `CMAKE_GENERATOR=Ninja` and CMake build commands.
- [x] Confirm the build outputs produced through this path are copied into Linux bundles/final Docker images used by release workflows.
- [x] Demonstrate, with a minimal bounded PoC, that a substituted `ninja` executable runs during the actual Ollama CMake build path.
## Report
Validated the finding as a supply-chain build-integrity issue, not a runtime crash/memory-safety bug. Crash/valgrind/debugger validation was not applicable: the suspected issue is an unverified Docker build-tool download, Docker is unavailable in the container, and `command -v valgrind`/`command -v gdb` returned nothing. Targeted code/dynamic build-path validation confirmed the issue. Commit 288077c3 introduces `ARG NINJAVERSION=1.12.1` and downloads Ninja release ZIPs directly from GitHub with `curl -fsSL -o /tmp/ninja.zip ...` followed by `unzip /tmp/ninja.zip -d /usr/local/bin` in Dockerfile:28-31, Dockerfile:93-97, and Dockerfile:109-113. The same stages set `ENV CMAKE_GENERATOR=Ninja` at Dockerfile:32, :98, and :114. Grepping the touched files showed the Ninja downloads but no checksum/signature/provenance verification near them (`sha256`, `gpg`, `cosign`, etc. absent for those downloads). The CMake build stages immediately use this generator: CPU stage runs `cmake --preset 'CPU'` and `cmake --build --preset 'CPU' -- -l $(nproc)` at Dockerfile:40-43; JetPack stages do the same at Dockerfile:101-104 and :117-120. `CMakePresets.json` does not define a generator, so `CMAKE_GENERATOR=Ninja` controls generator selection. I demonstrated executable control on the actual Ollama CMake path by placing a wrapper named `ninja` first in PATH and running `CMAKE_GENERATOR=Ninja cmake --preset CPU` plus a dry-run `cmake --build --preset CPU -- -n`. The resulting `build/CMakeCache.txt` contained `CMAKE_MAKE_PROGRAM:FILEPATH=/tmp/ollama-ninja-poc/ninja`, and the wrapper log showed multiple executions such as `FAKE_NINJA_EXECUTED argv=--version pwd=/workspace/ollama`, `FAKE_NINJA_EXECUTED argv=-v cmTC_70f6f ...`, and `FAKE_NINJA_EXECUTED argv=-n ggml-cpu pwd=/workspace/ollama/build`. This proves that a substituted Ninja executable runs during the Ollama native build. Release impact is supported by `.github/workflows/release.yaml`: linux release bundles are built with `docker/build-push-action@v6` using this Dockerfile and output to `dist/...` at lines 343-375, and Docker images are built/pushed with the same action at lines 446-452. Dockerfile:199-210 copies build outputs into the archive/final image. The prior Dockerfile installed `ninja-build` through `dnf` on the amd64 path; this commit newly adds GitHub ZIP-based Ninja downloads without digest/signature verification.

# Evidence
Dockerfile (L106 to 114)
  Note: The JetPack 6 stage also downloads and installs an unverified Ninja executable used for builds.
```
FROM --platform=linux/arm64 nvcr.io/nvidia/l4t-jetpack:${JETPACK6VERSION} AS jetpack-6
ARG CMAKEVERSION
ARG NINJAVERSION
RUN apt-get update && apt-get install -y curl ccache unzip \
    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1 \
    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux-aarch64.zip \
    && unzip /tmp/ninja.zip -d /usr/local/bin \
    && rm /tmp/ninja.zip
ENV CMAKE_GENERATOR=Ninja
```

Dockerfile (L24 to 32)
  Note: The base Docker stage downloads a prebuilt Ninja ZIP from GitHub, unzips it into /usr/local/bin, and sets CMAKE_GENERATOR=Ninja without verifying a checksum or signature.
```
FROM base-${TARGETARCH} AS base
ARG CMAKEVERSION
ARG NINJAVERSION
RUN curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1
RUN dnf install -y unzip \
    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux$([ "$(uname -m)" = "aarch64" ] && echo "-aarch64").zip \
    && unzip /tmp/ninja.zip -d /usr/local/bin \
    && rm /tmp/ninja.zip
ENV CMAKE_GENERATOR=Ninja
```

Dockerfile (L40 to 43)
  Note: A CMake build stage then invokes the configured Ninja generator to build artifacts that are later packaged.
```
RUN --mount=type=cache,target=/root/.ccache \
    cmake --preset 'CPU' \
        && cmake --build --preset 'CPU' -- -l $(nproc) \
        && cmake --install build --component CPU --strip
```

Dockerfile (L90 to 98)
  Note: The JetPack 5 stage repeats the unverified Ninja binary download and enables Ninja as the generator.
```
FROM --platform=linux/arm64 nvcr.io/nvidia/l4t-jetpack:${JETPACK5VERSION} AS jetpack-5
ARG CMAKEVERSION
ARG NINJAVERSION
RUN apt-get update && apt-get install -y curl ccache unzip \
    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1 \
    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux-aarch64.zip \
    && unzip /tmp/ninja.zip -d /usr/local/bin \
    && rm /tmp/ninja.zip
ENV CMAKE_GENERATOR=Ninja
```

github/workflows/release.yaml (L364 to 373)
  Note: The release workflow uses this Dockerfile to build Linux release outputs, so a compromised build tool can affect shipped artifacts.

github/workflows/release.yaml (L446 to 452)
  Note: The same Dockerfile is used to build and push release Docker images, extending impact to published container artifacts.

Proposed patch:
diff --git a/Dockerfile b/Dockerfile
index bc9cfc3cab88d7aa65ccba7d801770fca0b6fd90..52dbfdff7f42a875c5664b95d8da8b914c9249a7 100644
--- a/Dockerfile
+++ b/Dockerfile
@@ -1,61 +1,56 @@
 # vim: filetype=dockerfile
 
 ARG FLAVOR=${TARGETARCH}
 
 ARG ROCMVERSION=7.2.1
 ARG JETPACK5VERSION=r35.4.1
 ARG JETPACK6VERSION=r36.4.0
 ARG CMAKEVERSION=3.31.2
-ARG NINJAVERSION=1.12.1
 ARG VULKANVERSION=1.4.321.1
 
 # Default empty stages for local MLX source overrides.
 # Override with: docker build --build-context local-mlx=../mlx --build-context local-mlx-c=../mlx-c
 FROM scratch AS local-mlx
 FROM scratch AS local-mlx-c
 
 FROM --platform=linux/amd64 rocm/dev-almalinux-8:${ROCMVERSION}-complete AS base-amd64
 RUN dnf install -y yum-utils ccache gcc-toolset-13-gcc gcc-toolset-13-gcc-c++ gcc-toolset-13-binutils \
     && yum-config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel8/x86_64/cuda-rhel8.repo
 ENV PATH=/opt/rh/gcc-toolset-13/root/usr/bin:$PATH
 
 FROM --platform=linux/arm64 almalinux:8 AS base-arm64
 # install epel-release for ccache
 RUN yum install -y yum-utils epel-release \
     && dnf install -y clang ccache git \
     && yum-config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel8/sbsa/cuda-rhel8.repo
 ENV CC=clang CXX=clang++
 
 FROM base-${TARGETARCH} AS base
 ARG CMAKEVERSION
-ARG NINJAVERSION
 RUN curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1
-RUN dnf install -y unzip \
-    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux$([ "$(uname -m)" = "aarch64" ] && echo "-aarch64").zip \
-    && unzip /tmp/ninja.zip -d /usr/local/bin \
-    && rm /tmp/ninja.zip
+RUN dnf install -y ninja-build
 ENV CMAKE_GENERATOR=Ninja
 ENV LDFLAGS=-s
 
 #
 # GPU toolchain stages — provide compilers for llama-server GPU builds
 #
 
 FROM base AS cpu-deps
 RUN dnf install -y gcc-toolset-13-gcc gcc-toolset-13-gcc-c++
 ENV PATH=/opt/rh/gcc-toolset-13/root/usr/bin:$PATH
 
 FROM base AS cuda-12-deps
 ARG CUDA12VERSION=12.8
 RUN dnf install -y cuda-toolkit-${CUDA12VERSION//./-}
 ENV PATH=/usr/local/cuda-12/bin:$PATH
 
 FROM base AS cuda-13-deps
 ARG CUDA13VERSION=13.0
 RUN dnf install -y cuda-toolkit-${CUDA13VERSION//./-}
 ENV PATH=/usr/local/cuda-13/bin:$PATH
 
 FROM base AS rocm-7-deps
 ENV PATH=/opt/rocm/llvm/bin:/opt/rocm/hcc/bin:/opt/rocm/hip/bin:/opt/rocm/bin:$PATH
 
 FROM base AS vulkan-deps
@@ -133,76 +128,68 @@ RUN --mount=type=cache,target=/root/.ccache \
         && cmake --build build/llama-server-rocm_v7_2 -- -l $(nproc) \
         && cmake --install build/llama-server-rocm_v7_2 --component llama-server --strip
 RUN rm -f dist/lib/ollama/rocm_v7_2/rocblas/library/*gfx90[06]*
 
 FROM scratch AS publish-llama-server-rocm_v7_2
 COPY --from=llama-server-rocm_v7_2 dist/lib/ollama /lib/ollama/
 
 FROM vulkan-deps AS llama-server-vulkan
 COPY LLAMA_CPP_VERSION .
 COPY llama/server llama/server
 COPY llama/compat llama/compat
 RUN --mount=type=cache,target=/root/.ccache \
     cmake -S llama/server --preset vulkan \
         && cmake --build build/llama-server-vulkan -- -l $(nproc) \
         && cmake --install build/llama-server-vulkan --component llama-server --strip
 
 FROM scratch AS publish-llama-server-vulkan
 COPY --from=llama-server-vulkan dist/lib/ollama /lib/ollama/
 
 #
 # JetPack stages — self-contained with their own base images
 #
 
 FROM --platform=linux/arm64 nvcr.io/nvidia/l4t-jetpack:${JETPACK5VERSION} AS jetpack-5
 ARG CMAKEVERSION
-ARG NINJAVERSION
-RUN apt-get update && apt-get install -y curl ccache git unzip \
-    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1 \
-    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux-aarch64.zip \
-    && unzip /tmp/ninja.zip -d /usr/local/bin \
-    && rm /tmp/ninja.zip
+RUN apt-get update && apt-get install -y curl ccache git ninja-build \
+    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1
 ENV CMAKE_GENERATOR=Ninja
 COPY LLAMA_CPP_VERSION .
 COPY llama/server llama/server
 COPY llama/compat llama/compat
 RUN --mount=type=cache,target=/root/.ccache \
     cmake -S llama/server --preset llama_cuda_jetpack5 \
         && cmake --build build/llama-server-cuda_jetpack5 -- -l $(nproc) \
         && cmake --install build/llama-server-cuda_jetpack5 --component llama-server --strip
 
 FROM scratch AS publish-llama-server-cuda_jetpack5
 COPY --from=jetpack-5 dist/lib/ollama /lib/ollama/
 
 FROM --platform=linux/arm64 nvcr.io/nvidia/l4t-jetpack:${JETPACK6VERSION} AS jetpack-6
 ARG CMAKEVERSION
-ARG NINJAVERSION
-RUN apt-get update && apt-get install -y curl ccache git unzip \
-    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1 \
-    && curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux-aarch64.zip \
-    && unzip /tmp/ninja.zip -d /usr/local/bin \
-    && rm /tmp/ninja.zip
+RUN apt-get update && apt-get install -y curl ccache git ninja-build \
+    && curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-$(uname -m).tar.gz | tar xz -C /usr/local --strip-components 1
 ENV CMAKE_GENERATOR=Ninja
 COPY LLAMA_CPP_VERSION .
 COPY llama/server llama/server
 COPY llama/compat llama/compat
 RUN --mount=type=cache,target=/root/.ccache \
     cmake -S llama/server --preset llama_cuda_jetpack6 \
         && cmake --build build/llama-server-cuda_jetpack6 -- -l $(nproc) \
         && cmake --install build/llama-server-cuda_jetpack6 --component llama-server --strip
 
 FROM scratch AS publish-llama-server-cuda_jetpack6
 COPY --from=jetpack-6 dist/lib/ollama /lib/ollama/
 
 #
 # MLX stage
 #
 
 FROM base AS mlx
 ARG CUDA13VERSION=13.0
 ARG OLLAMA_MLX_BUILD_JOBS=
 ARG OLLAMA_MLX_NVCC_THREADS=2
 ARG MLX_CUDA_RAM_MB=
 RUN dnf install -y cuda-toolkit-${CUDA13VERSION//./-} \
     && dnf install -y openblas-devel lapack-devel \
     && dnf install -y libcudnn9-cuda-13 libcudnn9-devel-cuda-13 \
     && dnf install -y libnccl libnccl-devel


diff --git a/scripts/env.sh b/scripts/env.sh
index 5b55da84bd6fd3770d83faafe95ddcf865a46018..f94a73c8a94df63744a117f1f8e13caeddbc642c 100644
--- a/scripts/env.sh
+++ b/scripts/env.sh
@@ -1,46 +1,45 @@
 # Common environment setup across build*.sh scripts
 
 export VERSION=${VERSION:-$(git describe --tags --first-parent --abbrev=7 --long --dirty --always | sed -e "s/^v//g")}
 export GOFLAGS="'-ldflags=-w -s \"-X=github.com/ollama/ollama/version.Version=$VERSION\" \"-X=github.com/ollama/ollama/server.mode=release\"'"
 # TODO - consider `docker buildx ls --format=json` to autodiscover platform capability
 PLATFORM=${PLATFORM:-"linux/arm64,linux/amd64"}
 DOCKER_ORG=${DOCKER_ORG:-"ollama"}
 FINAL_IMAGE_REPO=${FINAL_IMAGE_REPO:-"${DOCKER_ORG}/ollama"}
 OLLAMA_COMMON_BUILD_ARGS="--build-arg=GOFLAGS"
 
 add_build_arg() {
     eval "_value=\"\${$1:-}\""
     if [ -n "$_value" ]; then
         OLLAMA_COMMON_BUILD_ARGS="$OLLAMA_COMMON_BUILD_ARGS --build-arg=$1"
     fi
 }
 
 for arg in \
     CGO_CFLAGS \
     CGO_CXXFLAGS \
     CMAKEVERSION \
-    NINJAVERSION \
     ROCMVERSION \
     JETPACK5VERSION \
     JETPACK6VERSION \
     CUDA12VERSION \
     CUDA13VERSION \
     VULKANVERSION \
     MLX_CUDA_RAM_MB \
     APT_MIRROR \
     OLLAMA_MLX_BUILD_JOBS \
     OLLAMA_MLX_NVCC_THREADS
 do
     add_build_arg "$arg"
 done
 
 # Forward local MLX source overrides as Docker build contexts
 if [ -n "${OLLAMA_MLX_SOURCE:-}" ]; then
     OLLAMA_COMMON_BUILD_ARGS="$OLLAMA_COMMON_BUILD_ARGS --build-context local-mlx=$(cd "$OLLAMA_MLX_SOURCE" && pwd)"
 fi
 if [ -n "${OLLAMA_MLX_C_SOURCE:-}" ]; then
     OLLAMA_COMMON_BUILD_ARGS="$OLLAMA_COMMON_BUILD_ARGS --build-context local-mlx-c=$(cd "$OLLAMA_MLX_C_SOURCE" && pwd)"
 fi
 echo "Building Ollama"
 echo "VERSION=$VERSION"
 echo "PLATFORM=$PLATFORM"

# Attack-path analysis
Final: high | Decider: model_decided | Matrix severity: high | Policy adjusted: high
## Rationale
Severity remains high, not critical. Repository evidence confirms the core statement: the Dockerfile downloads executable Ninja ZIPs from GitHub at lines 29, 95, and 111, immediately installs them, sets CMAKE_GENERATOR=Ninja, and uses CMake build stages to produce artifacts. The release workflow then builds Linux bundles and Docker images from this Dockerfile and publishes artifacts/images. The impact is high because a substituted build tool can backdoor shipped Ollama binaries/libraries/images and affect downstream users. However, likelihood is low because exploitation is not available to ordinary remote users of the daemon and requires compromise of a third-party artifact delivery path or build cache; no direct theft of release signing keys or registry credentials from inside the Docker build was proven. This supports high supply-chain severity but not critical.
## Likelihood
high - The path is not directly reachable through Ollama's runtime API. Exploitation requires compromising the upstream Ninja release asset, GitHub/CDN/TLS delivery, or a trusted build cache, then waiting for an affected release build. These are significant preconditions, but plausible for a supply-chain adversary. | Remote network vector
## Impact
high - A malicious Ninja binary executes during native build stages and can modify artifacts later shipped as Linux archives and Docker images. That can lead to downstream arbitrary code execution when users run trusted Ollama releases.
## Assumptions
- The checked-in .github/workflows/release.yaml is used for real Linux release bundles and Docker image publishing.
- Static repository evidence is sufficient to evaluate the build path; no cloud APIs or live CI settings were inspected.
- An attacker cannot directly control the Ninja download in normal operation unless they compromise or poison the upstream GitHub release asset, CDN/TLS trust path, or a trusted build cache/source.
- Compromise or substitution of the ninja-build GitHub release ZIP, GitHub/CDN/TLS delivery path, or trusted Docker build cache layer
- A release or Docker image build runs the affected Dockerfile stages
- The malicious Ninja binary remains compatible enough for CMake/build steps to produce releasable artifacts
## Path
Remote Ninja ZIP -> curl/unzip in Dockerfile -> CMAKE_GENERATOR=Ninja executes it -> Ollama artifacts -> release archives/Docker images -> downstream users
## Path evidence
- `Dockerfile:9` - Ninja is pinned only by version argument, not by immutable digest.
- `Dockerfile:24-32` - Base Docker stage downloads ninja-linux*.zip from github.com, unzips it into /usr/local/bin, and sets CMAKE_GENERATOR=Ninja without visible integrity verification.
- `Dockerfile:40-43` - CPU CMake build stage runs after CMAKE_GENERATOR=Ninja is set, causing the installed Ninja executable to be used for artifact generation.
- `Dockerfile:90-98` - JetPack 5 arm64 stage repeats the unverified Ninja ZIP download and generator selection.
- `Dockerfile:106-114` - JetPack 6 arm64 stage repeats the unverified Ninja ZIP download and generator selection.
- `Dockerfile:199-210` - Build outputs from Docker stages are copied into archive/final image paths.
- `Dockerfile:214-215` - Final image exposes the Ollama runtime service on 0.0.0.0:11434; this is product deployment context, though the vulnerability itself is build-time.
- `.github/workflows/release.yaml:343-375` - Linux release bundles are built using docker/build-push-action with the repository Dockerfile and local dist outputs.
- `.github/workflows/release.yaml:400-407` - Linux build outputs are packaged and uploaded as release artifacts.
- `.github/workflows/release.yaml:442-452` - Docker image release job logs in to a registry and pushes images built from the same Docker context.
- `.github/workflows/release.yaml:500-545` - Release job has contents write permission and uploads artifacts to GitHub releases.
## Narrative
The finding is real and in scope as a build supply-chain issue. The Dockerfile pins only NINJAVERSION=1.12.1, downloads Ninja ZIPs from GitHub with curl, unzips them into /usr/local/bin, and sets CMAKE_GENERATOR=Ninja. No checksum, signature, cosign, or provenance verification is present for those ZIPs. The subsequent CMake build stages invoke the generator and install native artifacts. The release workflow uses docker/build-push-action with this Dockerfile for Linux bundles and Docker images, then uploads release artifacts and pushes images. The earlier validation PoC also showed a substituted ninja executable being executed by the actual CMake path. This is not a runtime API vulnerability and requires compromise of the dependency delivery path, so likelihood is low, but impact is high because successful exploitation can backdoor shipped binaries/images.
## Controls
- HTTPS download via curl -fsSL
- NINJAVERSION pins a semantic version but not a digest
- GitHub Actions release environment is used for release jobs
- Release checksums are generated after build, which helps consumers detect transfer corruption but does not verify build inputs
- No checksum/signature/provenance verification was found for the Ninja ZIP downloads
## Blindspots
- Static-only review did not inspect live GitHub environment protections, branch protections, cache permissions, or release approval gates.
- No live Docker build or cloud/CI execution was performed in this attack-path stage.
- The analysis did not prove practical compromise of github.com, ninja-build release assets, TLS, CDN, or build cache; it evaluates the consequence if substitution occurs.
- Other unverified build-tool downloads exist, but this assessment focuses on the commit-introduced Ninja path.