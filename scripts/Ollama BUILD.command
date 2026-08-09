#!/bin/bash
#
# Ollama BUILD.command — double-click to package this working tree into
# Ollama.app and Ollama.dmg at the repository root.
#
# This wraps scripts/build_darwin.sh rather than reimplementing it, so there is
# one build in this repository and not two that can drift apart. Two things it
# does that the underlying script does not:
#
#   1. It creates the .dmg even without Apple credentials. In build_darwin.sh
#      the DMG step lives inside the code-signing branch, so an unsigned build
#      produces an .app and a zip and no disk image at all.
#
#   2. It points GOCACHE at a temporary directory for the duration, because
#      build_darwin.sh runs `go clean -cache`. Left alone that wipes the Go
#      build cache of whoever ran it, and the next ordinary `go test` in this
#      repo then rebuilds the world.
#
# The result is NOT signed or notarised. macOS will refuse to open it by
# double-click the first time; the script says how at the end.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

BOLD=$'\033[1m'; PLAIN=$'\033[0m'
say() { printf "%s==>%s %s\n" "$BOLD" "$PLAIN" "$*"; }
die() { printf "\n%sBUILD FAILED:%s %s\n\n" "$BOLD" "$PLAIN" "$*" >&2; exit 1; }

# A double-clicked .command opens a Terminal window that closes on exit, taking
# any error message with it. Hold it open so a failure can be read.
trap 'printf "\n%sPress return to close this window.%s\n" "$BOLD" "$PLAIN"; read -r _' EXIT

say "Repository: $REPO"
[ -f scripts/build_darwin.sh ] || die "scripts/build_darwin.sh is missing; is this the ollama repository?"
[ -f scripts/create-dmg.sh ] || die "scripts/create-dmg.sh is missing."

for tool in cmake npm node go lipo hdiutil plutil ditto; do
    command -v "$tool" >/dev/null 2>&1 || die "$tool is not installed."
done

VERSION="$(git describe --tags --first-parent --abbrev=7 --long --dirty --always 2>/dev/null | sed -e 's/^v//g')"
BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
say "Version: ${VERSION:-unknown}   Branch: $BRANCH"

cat <<BANNER

  This is a full universal build: llama.cpp and MLX are compiled from source
  for both arm64 and x86_64. build_darwin.sh cannot build a single
  architecture — it lipos both unconditionally when it assembles the runtime —
  so there is no faster path here without changing that script.

  Expect tens of minutes on a cold cache, and several gigabytes under dist/.

BANNER

# Scope the cache build_darwin.sh clears, so the caller's own survives.
GOCACHE_KEEP="$(go env GOCACHE)"
SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/ollama-build-XXXXXX")"
export GOCACHE="$SCRATCH/gocache"
mkdir -p "$GOCACHE"
say "Using a temporary Go cache so yours is untouched: $GOCACHE_KEEP"
cleanup_scratch() { rm -rf "$SCRATCH"; }

STARTED=$SECONDS

say "Building the native runtime and the app bundle"
# build   — compiles the native payload for both architectures
# sign    — with no APPLE_IDENTITY this only assembles the universal runtime
# app     — builds the frontend, the Go app binary, and dist/Ollama.app
./scripts/build_darwin.sh build sign app || { cleanup_scratch; die "build_darwin.sh failed; the output above says where."; }

[ -d dist/Ollama.app ] || { cleanup_scratch; die "dist/Ollama.app was not produced."; }

say "Creating the disk image"
# A volume of the same name still mounted makes hdiutil fail with "Operation
# not permitted", which is why the name is made unique per run.
VOL_NAME="Ollama ${VERSION:-build} $$"
rm -f dist/Ollama.dmg dist/rw*.dmg
(
    cd dist
    ../scripts/create-dmg.sh \
        --volname "$VOL_NAME" \
        --volicon ../app/darwin/Ollama.app/Contents/Resources/icon.icns \
        --background ../app/assets/background.png \
        --window-pos 200 120 \
        --window-size 800 400 \
        --icon-size 128 \
        --icon "Ollama.app" 200 190 \
        --hide-extension "Ollama.app" \
        --app-drop-link 600 190 \
        --text-size 12 \
        "Ollama.dmg" \
        "Ollama.app"
) || { cleanup_scratch; die "create-dmg.sh failed."; }
rm -f dist/rw*.dmg

say "Placing the results at the repository root"
rm -rf "$REPO/Ollama.app" "$REPO/Ollama.dmg"
ditto dist/Ollama.app "$REPO/Ollama.app"
cp dist/Ollama.dmg "$REPO/Ollama.dmg"

# ---------------------------------------------------------------------------
# Verification. A build script that reports success without checking has told
# you nothing; every claim below is tested rather than assumed.
# ---------------------------------------------------------------------------
say "Verifying"
FAILURES=0
check() {
    if eval "$2" >/dev/null 2>&1; then
        printf "    ok    %s\n" "$1"
    else
        printf "    FAIL  %s\n" "$1"
        FAILURES=$((FAILURES + 1))
    fi
}

APP="$REPO/Ollama.app"
check "the app bundle exists"                 "[ -d '$APP' ]"
check "its executable is present and runnable" "[ -x '$APP/Contents/MacOS/Ollama' ]"
check "Info.plist parses"                      "plutil -lint '$APP/Contents/Info.plist'"
check "the app is universal"                   "lipo '$APP/Contents/MacOS/Ollama' -verify_arch x86_64 arm64"
check "the bundled ollama binary is universal" "lipo '$APP/Contents/Resources/ollama' -verify_arch x86_64 arm64"
check "the bundled ollama binary runs"         "'$APP/Contents/Resources/ollama' --version"
check "the version was stamped"                "plutil -extract CFBundleShortVersionString raw '$APP/Contents/Info.plist' | grep -q ."
check "the embedded frontend is present"       "[ -s app/ui/app/dist/index.html ]"
check "the disk image exists"                  "[ -f '$REPO/Ollama.dmg' ]"
check "the disk image is intact"               "hdiutil verify '$REPO/Ollama.dmg'"

# Mounting proves the image actually carries the app, which "the file exists"
# does not.
MOUNT="$(mktemp -d "${TMPDIR:-/tmp}/ollama-dmg-XXXXXX")"
if hdiutil attach "$REPO/Ollama.dmg" -mountpoint "$MOUNT" -nobrowse -quiet >/dev/null 2>&1; then
    check "the disk image contains Ollama.app" "[ -d '$MOUNT/Ollama.app' ]"
    hdiutil detach "$MOUNT" -quiet >/dev/null 2>&1 || hdiutil detach "$MOUNT" -force -quiet >/dev/null 2>&1 || true
else
    printf "    FAIL  the disk image could not be mounted\n"
    FAILURES=$((FAILURES + 1))
fi
rmdir "$MOUNT" 2>/dev/null || true

cleanup_scratch

ELAPSED=$((SECONDS - STARTED))
printf "\n"
if [ "$FAILURES" -ne 0 ]; then
    die "$FAILURES verification check(s) failed after ${ELAPSED}s. The outputs above are not trustworthy."
fi

say "Built in $((ELAPSED / 60))m $((ELAPSED % 60))s"
printf "\n    %s\n    %s\n\n" "$REPO/Ollama.app" "$REPO/Ollama.dmg"

cat <<'NOTES'
  This build is NOT signed or notarised, so macOS Gatekeeper will refuse it on
  first open. To run it anyway, either right-click the app and choose Open, or:

      xattr -dr com.apple.quarantine /Users/krypto/GitHub/ollama/Ollama.app

  Because it is unsigned it is also unsuitable for testing the in-app updater,
  which is what build_darwin.sh warns about when signing is skipped.

NOTES
