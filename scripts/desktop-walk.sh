#!/bin/sh
# Walks every GUI flow through the real binary, driven by the keyboard, so a
# development machine or a CI runner produces screen evidence with no device.
#
# Each flow gets a throwaway fixture root and its own evidence directory. Every
# flow must reach the screens it claims; the walk.tsv record beside the frames
# names the screen each key reached, the selected package, the action, and any
# error, so a run can be checked without reading pixels.
#
# usage: desktop-walk.sh ROOT OUTDIR [--install]
#   ROOT     parent directory for the per-flow scratch roots
#   OUTDIR   parent directory for the per-flow frames and records
#   --install  also walk the flows that download a package (needs network)
#
# Environment: WALK_BIN, WALK_INDEX, WALK_DEVICE, WALK_ARCH, WALK_RESOLUTION,
# WALK_TIMEOUT, and WALK_PACKAGE_PATH.
set -eu

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "usage: $0 ROOT OUTDIR [--install]" >&2
  exit 2
fi

ROOT=$1
OUTDIR=$2
INSTALL_FLOWS=no
case "${3:-}" in
  "") ;;
  --install) INSTALL_FLOWS=yes ;;
  *) echo "unknown option: $3" >&2; exit 2 ;;
esac

case "$ROOT" in
  /*) ;;
  *) echo "root must be an absolute path: $ROOT" >&2; exit 2 ;;
esac
case "$OUTDIR" in
  /*) ;;
  *) echo "output directory must be an absolute path: $OUTDIR" >&2; exit 2 ;;
esac
if [ "$ROOT" = "/" ] || [ "$OUTDIR" = "/" ]; then
  echo "root and output directory must not be /" >&2
  exit 2
fi
if [ "$ROOT" = "$OUTDIR" ]; then
  echo "root and output directory must be different" >&2
  exit 2
fi

BIN=${WALK_BIN:-build/knulli-app-ui}
INDEX=${WALK_INDEX:-build/catalog-index.json}
DEVICE=${WALK_DEVICE:-trimui-smart-pro}
ARCH=${WALK_ARCH:-aarch64}
RESOLUTION=${WALK_RESOLUTION:-1280x720}
TIMEOUT=${WALK_TIMEOUT:-60s}
# Where the walkthrough looks for an installed package when it tampers with one.
PACKAGE_PATH=${WALK_PACKAGE_PATH:-/userdata/roms/tools/Grout}

if [ ! -x "$BIN" ]; then
  echo "missing $BIN: run 'make build-ui' first" >&2
  exit 2
fi
if [ ! -f "$INDEX" ]; then
  echo "missing $INDEX: run 'make catalogue' first" >&2
  exit 2
fi

mkdir -p "$ROOT" "$OUTDIR"
FAILURES=0
SUMMARY=$OUTDIR/summary.tsv
printf 'flow\tkeys\tframes\tscreens\tresult\n' >"$SUMMARY"

# walk NAME INPUT KEYS EXPECTED_SCREENS [EXPECTED_MESSAGE] [MODE] [ROOT_NAME]
#
# EXPECTED_SCREENS is a comma-separated list that must all appear in the record.
# EXPECTED_MESSAGE is a substring the record must contain, for a flow whose
# point is a message rather than a screen. MODE is "fresh" (the default), which
# starts from a new fixture root and a first run, or "reuse", which continues an
# existing root so a flow can walk the state an earlier flow left behind.
walk() {
  name=$1
  input=$2
  keys=$3
  expected=$4
  message=${5:-}
  mode=${6:-fresh}
  root="$ROOT/${7:-$name}"
  out="$OUTDIR/$name"
  rm -rf "$out"
  if [ "$mode" != reuse ]; then
    rm -rf "$root"
    scripts/desktop-fixture.sh "$DEVICE" "$root" >/dev/null
  fi

  result=ok
  if ! "$BIN" \
    -input "$input" \
    -root "$root" \
    -arch "$ARCH" \
    -resolution "$RESOLUTION" \
    -catalog "$INDEX" \
    -keys "$keys" \
    -shot-dir "$out" \
    -walk-timeout "$TIMEOUT" >"$out.log" 2>&1; then
    result="exited non-zero"
    FAILURES=$((FAILURES + 1))
  fi

  if [ ! -f "$out/walk.tsv" ]; then
    echo "FAIL $name: no walk.tsv record" >&2
    sed 's/^/    /' "$out.log" >&2
    printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$keys" 0 "none" "no record" >>"$SUMMARY"
    FAILURES=$((FAILURES + 1))
    return 0
  fi

  frames=$(($(wc -l <"$out/walk.tsv") - 1))
  screens=$(cut -f2 "$out/walk.tsv" | tail -n +2 | awk '!seen[$0]++' | tr '\n' ',' | sed 's/,$//')

  for screen in $(echo "$expected" | tr ',' ' '); do
    if ! cut -f2 "$out/walk.tsv" | tail -n +2 | grep -qx "$screen"; then
      echo "FAIL $name: never reached screen '$screen' (reached: $screens)" >&2
      result="missing $screen"
      FAILURES=$((FAILURES + 1))
    fi
  done

  if [ -n "$message" ] && ! grep -qF "$message" "$out/walk.tsv"; then
    echo "FAIL $name: record does not show '$message'" >&2
    result="missing message"
    FAILURES=$((FAILURES + 1))
  fi

  if [ "$frames" -eq 0 ]; then
    echo "FAIL $name: the walkthrough captured no frame" >&2
    result="no frames"
    FAILURES=$((FAILURES + 1))
  fi

  if [ "$result" != ok ]; then
    echo "  record:" >&2
    sed 's/^/    /' "$out/walk.tsv" >&2
  fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$keys" "$frames" "$screens" "$result" >>"$SUMMARY"
  return 0
}

# Controller setup, driven entirely by the keyboard.
walk first-run-use-detected keyboard \
  "enter" \
  "setup,catalogue"

walk first-run-test-detected keyboard \
  "down,enter,up,down,left,right,enter,esc,y,q" \
  "setup,preview,catalogue" \
  "Controller mapping tested and saved"

walk first-run-customize keyboard \
  "down,down,enter,up,enter,down,enter,left,enter,right,enter,enter,enter,esc,enter,y,enter,q,enter,up,down,left,right,enter,esc,y,q" \
  "setup,calibration,assignment-review,preview,catalogue" \
  "Controller mapping tested and saved"

walk first-run-safe-exit keyboard \
  "down,down,down,enter" \
  "setup"

# Catalogue, package details, the action list, and the confirmation dialog.
walk catalogue-walk keyboard \
  "enter,down,up,enter,enter,esc,esc,q" \
  "setup,catalogue,actions,confirm"

# A read-only package offers no action and says so. The catalogue is sorted by
# package name, so a candidate sits below the two actionable packages.
walk catalogue-read-only keyboard \
  "enter,down,down,enter" \
  "setup,catalogue" \
  "No safe action is available"

# Settings: export diagnostics writes a bundle below the scratch root.
walk settings-export-diagnostics keyboard \
  "enter,y,down,enter,esc" \
  "setup,catalogue,settings" \
  "Diagnostics saved to"

# No controller at all: the blocked screen still exports and still leaves.
walk blocked-export-diagnostics auto \
  "enter,esc" \
  "blocked" \
  "Diagnostics saved to"

# tamper appends to one managed file of an installed package, so the next walk
# starts from an issue the health check has to report.
tamper() {
  name=$1
  destination="$ROOT/$name$PACKAGE_PATH"
  file=$(find "$destination" -type f | head -1)
  if [ -z "$file" ]; then
    echo "FAIL: nothing installed at $destination to tamper with" >&2
    exit 1
  fi
  printf 'tampered by the walkthrough\n' >>"$file"
}

if [ "$INSTALL_FLOWS" = yes ]; then
  # Downloads, verifies, and applies a package, then walks the lifecycle the
  # installed package exposes: the health check, repair, and uninstall.
  walk install-package keyboard \
    "enter,enter,enter,enter" \
    "setup,catalogue,actions,confirm,progress" \
    "Install completed"

  tamper install-package
  walk install-package-health keyboard \
    "enter,enter,down,enter,enter" \
    "health,actions,confirm,progress" \
    "Repair completed" \
    reuse install-package

  walk installed-uninstall keyboard \
    "enter,enter,enter,enter" \
    "catalogue,actions,confirm,progress" \
    "Uninstall completed" \
    reuse install-package
fi

echo
echo "walkthrough evidence: $OUTDIR"
column -t -s "$(printf '\t')" "$SUMMARY" 2>/dev/null || cat "$SUMMARY"
if [ "$FAILURES" -ne 0 ]; then
  echo
  echo "$FAILURES flow check(s) failed" >&2
  exit 1
fi
echo
echo "all flows reached the screens they claim"
