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
note() { printf "    %s\n" "$*"; }
die() { printf "\n%sBUILD FAILED:%s %s\n\n" "$BOLD" "$PLAIN" "$*" >&2; exit 1; }

SCRATCH=""

# Detach any disk image this script mounted, and remove the half-built images
# it leaves behind when a DMG attempt fails part-way. Only images backed by a
# file under this repository's dist/ are touched — never anything else the
# machine has mounted.
our_attached_devices() {
    local line image
    image=""
    while IFS= read -r line; do
        case "$line" in
            "================"*) image="" ;;
            image-path*) image="${line#*: }" ;;
            /dev/disk*)
                case "$image" in
                    "$REPO"/dist/rw*.dmg|"$REPO"/dist/Ollama.dmg)
                        printf "%s\n" "${line%%[[:space:]]*}"
                        ;;
                esac
                ;;
        esac
    done < <(hdiutil info 2>/dev/null || true)
}

release_disk_images() {
    local waited=0 device any
    # A killed attempt can leave hdiutil mid-write, so detaching once is not
    # enough: keep detaching until none of our images are attached. Without this
    # the next attempt races a volume that is still going away, and fails with
    # "Input/output error" on a file it cannot reach.
    while :; do
        any=0
        while IFS= read -r device; do
            [ -n "$device" ] || continue
            any=1
            hdiutil detach "$device" -quiet >/dev/null 2>&1 \
                || hdiutil detach "$device" -force -quiet >/dev/null 2>&1 \
                || true
        done < <(our_attached_devices)
        if [ "$any" -eq 0 ] || [ "$waited" -ge 40 ]; then
            break
        fi
        sleep 2
        waited=$((waited + 2))
    done
    # create-dmg's Finder script can outlive a killed attempt. The pattern is
    # its own temp file name, so nothing else on the machine matches.
    pkill -f createdmg.tmp >/dev/null 2>&1 || true
    rm -f "$REPO"/dist/rw*.dmg
}

cleanup() {
    release_disk_images
    [ -n "$SCRATCH" ] && rm -rf "$SCRATCH"
    return 0
}

# A double-clicked .command opens a Terminal window that closes on exit, taking
# any error message with it. Hold it open so a failure can be read.
trap 'cleanup; printf "\n%sPress return to close this window.%s\n" "$BOLD" "$PLAIN"; read -r _' EXIT

# Signal a whole job, not just the shell leading it. create-dmg spawns hdiutil
# and osascript; killing only the parent leaves those running, and an orphaned
# hdiutil still writing to a volume will wreck whatever is attempted next.
#
# The guard matters: if job control did not put the child in its own process
# group, its group is ours, and signalling it would kill this build.
kill_job() {
    local pid="$1" signal="$2" theirs mine
    theirs="$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ')"
    mine="$(ps -o pgid= -p $$ 2>/dev/null | tr -d ' ')"
    if [ -n "$theirs" ] && [ -n "$mine" ] && [ "$theirs" != "$mine" ]; then
        kill "-$signal" "-$theirs" >/dev/null 2>&1 || true
    else
        kill "-$signal" "$pid" >/dev/null 2>&1 || true
    fi
}

# Run a command with a deadline. Returns 124 if it had to be killed. macOS has
# no timeout(1), and the styled DMG step can block indefinitely — see make_dmg.
run_with_deadline() {
    local seconds="$1"; shift
    local pid waited status
    set -m          # job control, so the child leads its own process group
    "$@" &
    pid=$!
    set +m
    waited=0
    while kill -0 "$pid" 2>/dev/null; do
        if [ "$waited" -ge "$seconds" ]; then
            # Say so before the shell prints its own "Terminated" line, which
            # otherwise reads like a crash.
            note "Deadline reached — stopping that step and everything it started."
            kill_job "$pid" TERM
            sleep 3
            kill_job "$pid" KILL
            wait "$pid" >/dev/null 2>&1 || true
            return 124
        fi
        sleep 1
        waited=$((waited + 1))
    done
    status=0
    wait "$pid" || status=$?
    return "$status"
}

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
say "Go cache"
note "using  $GOCACHE"
note "so that $GOCACHE_KEEP is left alone — build_darwin.sh clears the one it uses"

STARTED=$SECONDS

say "Building the native runtime and the app bundle"
# build   — compiles the native payload for both architectures
# sign    — with no APPLE_IDENTITY this only assembles the universal runtime
# app     — builds the frontend, the Go app binary, and dist/Ollama.app
./scripts/build_darwin.sh build sign app || die "build_darwin.sh failed; the output above says where."

[ -d dist/Ollama.app ] || die "dist/Ollama.app was not produced."

# ---------------------------------------------------------------------------
# The disk image.
#
# create-dmg lays the window out by driving Finder over Apple Events, and its
# template ends in an unbounded `repeat` that waits for Finder to write a
# .DS_Store to the mounted volume. That needs a GUI session that is awake,
# unlocked, and willing to answer Apple Events. A long unattended build does not
# have one — the screen locks part-way through and the whole step dies with
# "AppleEvent timed out (-1712)", losing a build that had otherwise succeeded.
#
# So the cosmetics are not allowed to decide whether the build succeeds. The
# styled image is attempted under a deadline; if it cannot be had, a plain image
# is built instead. Both contain the app and the Applications link, which is
# what the image is actually for.
# ---------------------------------------------------------------------------
STYLED_DEADLINE=420

make_dmg() {
    local mode="$1"
    local extra=()
    if [ "$mode" = plain ]; then
        extra=(--skip-jenkins)
    fi
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
            ${extra[@]+"${extra[@]}"} \
            "Ollama.dmg" \
            "Ollama.app"
    )
}

say "Creating the disk image"
# A volume of the same name still mounted makes hdiutil fail with "Operation
# not permitted", which is why the name is made unique per run.
VOL_NAME="Ollama ${VERSION:-build} $$"
DMG_STYLE=""
DMG_FALLBACK_REASON=""

if ioreg -n Root -d1 -a 2>/dev/null | grep -q CGSSessionScreenIsLocked; then
    DMG_FALLBACK_REASON="the screen is locked, and laying the window out needs Finder"
    note "Screen locked — building a plain image; unlock and re-run for the styled one."
else
    note "Laying the window out with Finder (up to ${STYLED_DEADLINE}s)"
    set +e
    run_with_deadline "$STYLED_DEADLINE" make_dmg styled
    styled_status=$?
    set -e
    if [ "$styled_status" -eq 0 ]; then
        DMG_STYLE="styled"
    elif [ "$styled_status" -eq 124 ]; then
        DMG_FALLBACK_REASON="Finder did not finish within ${STYLED_DEADLINE}s, which is what happens when the screen locks during a build"
    else
        DMG_FALLBACK_REASON="create-dmg's Finder step failed (exit $styled_status); the output above says how"
    fi
fi

if [ -z "$DMG_STYLE" ]; then
    say "Falling back to a plain disk image"
    note "$DMG_FALLBACK_REASON"
    note "The image still carries Ollama.app and the Applications link; it has no"
    note "background picture and its icons are unpositioned."
    release_disk_images
    make_dmg plain || die "create-dmg.sh failed even without the Finder step."
    DMG_STYLE="plain"
fi

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
    # Without this link the image cannot be used the way every macOS user
    # expects to use one, whether or not Finder laid the window out.
    check "it offers the Applications folder"  "[ -L '$MOUNT/Applications' ]"
    hdiutil detach "$MOUNT" -quiet >/dev/null 2>&1 || hdiutil detach "$MOUNT" -force -quiet >/dev/null 2>&1 || true
else
    printf "    FAIL  the disk image could not be mounted\n"
    FAILURES=$((FAILURES + 1))
fi
rmdir "$MOUNT" 2>/dev/null || true

ELAPSED=$((SECONDS - STARTED))
printf "\n"
if [ "$FAILURES" -ne 0 ]; then
    die "$FAILURES verification check(s) failed after ${ELAPSED}s. The outputs above are not trustworthy."
fi

say "Built in $((ELAPSED / 60))m $((ELAPSED % 60))s"
printf "\n    %s\n    %s (%s)\n\n" "$REPO/Ollama.app" "$REPO/Ollama.dmg" "$DMG_STYLE"

if [ "$DMG_STYLE" = plain ]; then
    cat <<NOTES
  The disk image has no background picture because $DMG_FALLBACK_REASON.
  It installs and runs exactly the same. For the styled one, re-run this with
  the screen unlocked and the machine set not to sleep.

NOTES
fi

cat <<'NOTES'
  This build is NOT signed or notarised, so macOS Gatekeeper will refuse it on
  first open. To run it anyway, either right-click the app and choose Open, or:

      xattr -dr com.apple.quarantine /Users/krypto/GitHub/ollama/Ollama.app

  Because it is unsigned it is also unsuitable for testing the in-app updater,
  which is what build_darwin.sh warns about when signing is skipped.

NOTES
