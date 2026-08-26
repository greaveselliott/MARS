#!/usr/bin/env bash

# MarsDocSync:
# docs:
# - docs/design-docs/release-versioning.md
# - docs/features/F-018-goreleaser-distribution.md

set -euo pipefail

if [[ $# -ne 5 ]]; then
  echo "usage: release-produce.sh <repo-root> <dist-dir> <version> <full-commit> <commit-time>" >&2
  exit 2
fi

repo_root=$1
dist_dir=$2
version=$3
full_commit=$4
commit_time=$5

if [[ ! $version =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "release producer: version must be MAJOR.MINOR.PATCH" >&2
  exit 1
fi
if [[ ! $full_commit =~ ^[0-9a-f]{40}$ ]]; then
  echo "release producer: commit must be 40 lowercase hexadecimal characters" >&2
  exit 1
fi
if [[ -e $dist_dir ]]; then
  echo "release producer: dist path already exists; use a new empty output path" >&2
  exit 1
fi
if [[ ! -d $repo_root/.git ]]; then
  echo "release producer: repo root is not a Git checkout" >&2
  exit 1
fi

repo_root=$(cd "$repo_root" && pwd -P)
dist_parent=$(dirname "$dist_dir")
mkdir -p "$dist_parent"
dist_parent=$(cd "$dist_parent" && pwd -P)
dist_dir="$dist_parent/$(basename "$dist_dir")"

go_bin=$(command -v go)
syft_bin=${MARS_RELEASE_SYFT:-}
if [[ -z $syft_bin || ! -x $syft_bin ]]; then
  echo "release producer: set MARS_RELEASE_SYFT to the executable Syft v1.51.0 binary" >&2
  exit 1
fi
if [[ $("$go_bin" env GOVERSION) != go1.27.0 ]]; then
  echo "release producer: exact Go 1.27.0 is required" >&2
  exit 1
fi
if ! tar --version 2>/dev/null | grep -F 'GNU tar' >/dev/null; then
  echo "release producer: GNU tar is required; run on the supported Ubuntu producer" >&2
  exit 1
fi
if ! commit_time=$(date -u -d "$commit_time" '+%Y-%m-%dT%H:%M:%SZ'); then
  echo "release producer: commit time must be valid RFC3339" >&2
  exit 1
fi
if [[ $(git -C "$repo_root" rev-parse HEAD) != "$full_commit" ]]; then
  echo "release producer: checkout HEAD does not match the requested commit" >&2
  exit 1
fi
if [[ -n $(git -C "$repo_root" status --porcelain=v1 --untracked-files=all) ]]; then
  echo "release producer: checkout must be clean" >&2
  exit 1
fi
if [[ -n ${ACTIONS_ID_TOKEN_REQUEST_URL:-}${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}${COSIGN_PASSWORD:-}${COSIGN_PRIVATE_KEY:-}${SSH_AUTH_SOCK:-} ]]; then
  echo "release producer: signing or SSH authority is present in the build job" >&2
  exit 1
fi
if ! "$go_bin" version -m "$syft_bin" | grep -F $'\tmod\tgithub.com/anchore/syft\tv1.51.0\t' >/dev/null; then
  echo "release producer: Syft module identity is not v1.51.0" >&2
  exit 1
fi
if ! "$go_bin" version -m "$syft_bin" | grep -F ': go1.27.0' >/dev/null; then
  echo "release producer: Syft was not built with Go 1.27.0" >&2
  exit 1
fi

work_root=$(mktemp -d "${TMPDIR:-/tmp}/mars-release-produce.XXXXXX")
cleanup() {
  status=$?
  chmod -R u+w -- "$work_root" 2>/dev/null || true
  rm -rf -- "$work_root" || status=1
  if [[ -e $work_root ]]; then status=1; fi
  exit "$status"
}
trap cleanup EXIT
umask 077
mkdir -m 700 "$dist_dir"

for target in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64; do
  goos=${target%-*}
  goarch=${target#*-}
  stage="$work_root/$target"
  mkdir -m 700 "$stage"
  install -m 0644 "$repo_root/LICENSE" "$stage/LICENSE"
  install -m 0644 "$repo_root/NOTICE" "$stage/NOTICE"
  install -m 0644 "$repo_root/THIRD_PARTY_NOTICES" "$stage/THIRD_PARTY_NOTICES"

  ldflags="-s -w -X main.version=$version -X main.commit=$full_commit -X main.date=$commit_time"
  (
    cd "$repo_root"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 GOTOOLCHAIN=local GOENV=off \
      "$go_bin" build -trimpath -buildvcs=true -ldflags "$ldflags" -o "$stage/mars" ./cmd/mars
  )
  chmod 0755 "$stage/mars"

  archive="$dist_dir/mars_${version}_${goos}_${goarch}.tar.gz"
  tar --create --blocking-factor=1 --format=ustar --owner=root --group=root --mtime="$commit_time" \
    --file=- --directory="$stage" LICENSE NOTICE THIRD_PARTY_NOTICES mars | gzip -n -9 > "$archive"
  "$syft_bin" "file:$archive" --config "$repo_root/.syft.yaml" --quiet \
    --output "spdx-json=$archive.sbom.json"
done

(
  cd "$dist_dir"
  test "$(find . -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.sbom.json' \) | wc -l | tr -d ' ')" = 8
  sha256sum ./*.tar.gz ./*.sbom.json | sed 's#  \./#  #' | LC_ALL=C sort -k2,2 > checksums.txt
  test "$(find . -maxdepth 1 -type f | wc -l | tr -d ' ')" = 9
)
