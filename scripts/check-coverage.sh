#!/usr/bin/env bash
# MarsDocSync:
# docs:
# - docs/design-docs/source-quality-gates.md
# - docs/design-docs/code-documentation-map.md
#
# Per-package coverage ratchet gate (T-024, AD-280).
#
# Compares `go test -cover` per-package results against the ratchet floors in
# scripts/coverage-floors.txt and fails on any regression below floor, any
# package missing a floor entry, and any stale floor entry whose package no
# longer exists. Floors only go up: raise a floor in the same commit that
# durably improves a package's coverage.
#
# Usage:
#   scripts/check-coverage.sh                 # runs go test -count=1 -cover ./...
#   scripts/check-coverage.sh --input FILE    # parses pre-captured go test -cover output
#   scripts/check-coverage.sh --floors FILE   # overrides the floors file (tests)
set -euo pipefail

input_file=""
floors_file="$(dirname "$0")/coverage-floors.txt"

while [ $# -gt 0 ]; do
  case "$1" in
    --input)
      input_file="$2"
      shift 2
      ;;
    --floors)
      floors_file="$2"
      shift 2
      ;;
    *)
      echo "check-coverage: unknown argument '$1'." >&2
      echo "Usage: scripts/check-coverage.sh [--input FILE] [--floors FILE]" >&2
      exit 2
      ;;
  esac
done

if [ ! -f "$floors_file" ]; then
  echo "check-coverage: floors file not found at $floors_file." >&2
  echo "Fix: restore scripts/coverage-floors.txt or pass --floors <file>." >&2
  exit 2
fi

if [ -n "$input_file" ]; then
  if [ ! -f "$input_file" ]; then
    echo "check-coverage: input file not found at $input_file." >&2
    echo "Fix: capture output first, e.g. go test -count=1 -cover ./... | tee coverage-report.txt" >&2
    exit 2
  fi
  coverage_output="$(cat "$input_file")"
else
  echo "check-coverage: running go test -count=1 -cover ./... (use --input FILE to reuse a captured run)"
  coverage_output="$(go test -count=1 -cover ./... 2>&1)" || {
    status=$?
    echo "$coverage_output"
    echo "check-coverage: go test failed (exit $status); fix the failing tests before checking coverage." >&2
    exit "$status"
  }
fi

# Parse `go test -cover` output into "<pkg> <percent|notest|nostmt>" lines.
# Handles:
#   ok   <pkg>  1.2s  coverage: 51.8% of statements
#   ok   <pkg>  (cached)  coverage: [no statements]
#   ?    <pkg>  [no test files]
#   <pkg>  coverage: 0.0% of statements   (builds with -cover but no test run)
parsed="$(echo "$coverage_output" | awk '
  /\[no test files\]/ { print $2, "notest"; next }
  /coverage: \[no statements\]/ {
    pkg = ($1 == "ok") ? $2 : $1
    print pkg, "nostmt"; next
  }
  /coverage: [0-9.]+% of statements/ {
    pkg = ($1 == "ok") ? $2 : $1
    for (i = 1; i <= NF; i++) {
      if ($i == "coverage:") {
        pct = $(i + 1)
        sub(/%$/, "", pct)
        print pkg, pct
        next
      }
    }
  }
')"

if [ -z "$parsed" ]; then
  echo "check-coverage: no coverage lines found in the input." >&2
  echo "Fix: ensure the input is the output of go test -cover ./... (per-package mode)." >&2
  exit 2
fi

failures=0

report_failure() {
  failures=$((failures + 1))
  echo "FAIL: $1" >&2
  echo "  Fix: $2" >&2
}

# Gate 1: every measured package meets its floor.
while read -r pkg actual; do
  floor="$(awk -v p="$pkg" '$1 == p { print $2 }' "$floors_file")"
  if [ -z "$floor" ]; then
    report_failure "package $pkg has no floor entry in $floors_file" \
      "add '$pkg <floor>' seeded from its current coverage (round down; >=70 gets 70; notest/nostmt for untestable packages)."
    continue
  fi
  case "$actual" in
    notest)
      if [ "$floor" != "notest" ]; then
        report_failure "package $pkg reports [no test files] but its floor is $floor" \
          "restore the package's tests; deleting all tests is a coverage regression."
      fi
      ;;
    nostmt)
      if [ "$floor" != "nostmt" ]; then
        report_failure "package $pkg reports [no statements] but its floor is $floor" \
          "verify the package still contains statements; update the floor entry only if the package legitimately became test-only."
      fi
      ;;
    *)
      if [ "$floor" = "notest" ] || [ "$floor" = "nostmt" ]; then
        echo "NOTE: package $pkg now reports ${actual}% coverage but its floor is '$floor'; ratchet it to a numeric floor."
        continue
      fi
      meets="$(awk -v a="$actual" -v f="$floor" 'BEGIN { print (a + 0 >= f + 0) ? "yes" : "no" }')"
      if [ "$meets" != "yes" ]; then
        report_failure "package $pkg coverage ${actual}% is below its ratchet floor ${floor}%" \
          "add or restore tests for $pkg until coverage is at least ${floor}%; floors are ratchet-only and must not be lowered."
      fi
      ;;
  esac
done <<<"$parsed"

# Gate 2: no stale floor entries for removed packages.
while read -r pkg floor; do
  case "$pkg" in
    ''|'#'*) continue ;;
  esac
  found="$(echo "$parsed" | awk -v p="$pkg" '$1 == p { print "yes"; exit }')"
  if [ -z "$found" ]; then
    report_failure "floor entry for $pkg matches no package in the coverage run" \
      "remove the stale entry from $floors_file (or fix the package path) in the same commit that removed the package."
  fi
done <"$floors_file"

if [ "$failures" -gt 0 ]; then
  echo "check-coverage: $failures coverage gate failure(s)." >&2
  exit 1
fi

echo "check-coverage: all packages meet their ratchet floors."
