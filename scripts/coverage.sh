#!/usr/bin/env bash
# Fails if any package's statement coverage is below the threshold (default 95).
# Exempt: packages under examples/, test-support packages (internal/schematest)
# and packages without statements.
set -euo pipefail
cd "$(dirname "$0")/.."
threshold="${COVERAGE_THRESHOLD:-95}"
out="$(go test -race -cover ./... 2>&1)" || { echo "$out"; exit 1; }
echo "$out"
fail=0
while IFS= read -r line; do
  # Lines look like "ok  <pkg>  1.2s  coverage: 97.0% of statements" or, for a
  # package without tests, "<tab><pkg><tab><tab>coverage: 0.0% of statements".
  pkg="$(awk '{ if ($1=="ok" || $1=="FAIL" || $1=="?") print $2; else print $1 }' <<<"$line")"
  case "$pkg" in */examples/*|*/internal/schematest) continue ;; esac
  pct="$(sed -n 's/.*coverage: \([0-9.]*\)% of statements.*/\1/p' <<<"$line")"
  [[ -z "$pct" ]] && continue
  if awk -v p="$pct" -v t="$threshold" 'BEGIN{exit !(p<t)}'; then
    echo "FAIL: $pkg coverage ${pct}% < ${threshold}%"
    fail=1
  fi
done < <(grep 'coverage:' <<<"$out")
exit $fail
