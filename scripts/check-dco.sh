#!/usr/bin/env bash
# MarsDocSync:
# docs:
# - CONTRIBUTING.md
# - docs/features/F-017-open-source-publication.md
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "DCO check: expected BASE_COMMIT and HEAD_COMMIT; run scripts/check-dco.sh <base> <head>" >&2
  exit 2
fi

base=$1
head=$2

git cat-file -e "${base}^{commit}" 2>/dev/null || {
  echo "DCO check: base commit $base is unavailable; fetch the complete pull-request history and retry" >&2
  exit 2
}
git cat-file -e "${head}^{commit}" 2>/dev/null || {
  echo "DCO check: head commit $head is unavailable; fetch the complete pull-request history and retry" >&2
  exit 2
}
git merge-base --is-ancestor "$base" "$head" || {
  echo "DCO check: base $base is not an ancestor of head $head; refresh the pull request and retry" >&2
  exit 2
}

checked=0
while IFS= read -r commit; do
  [[ -n "$commit" ]] || continue
  checked=$((checked + 1))
  author_email=$(git show -s --format=%ae "$commit")
  if ! git show -s --format=%B "$commit" | git interpret-trailers --parse | awk -v author_email="$author_email" '
    BEGIN { author_email = tolower(author_email); found = 0 }
    {
      line = tolower($0)
      if (line ~ /^signed-off-by:[[:space:]]+.+[[:space:]]+<[^<>]+>$/ && index(line, "<" author_email ">") > 0) {
        found = 1
      }
    }
    END { exit found ? 0 : 1 }
  '; then
    echo "DCO check: commit $commit lacks a Signed-off-by trailer matching author email $author_email" >&2
    echo "DCO check: amend it with 'git commit --amend --signoff' (or sign off the original commit) and retry" >&2
    exit 1
  fi
done < <(git rev-list --reverse --no-merges "${base}..${head}")

if [[ $checked -eq 0 ]]; then
  echo "DCO check: no non-merge commits exist between $base and $head; refresh the pull request and retry" >&2
  exit 2
fi

echo "DCO check: $checked non-merge commit(s) have author-matching sign-offs"

