#!/usr/bin/env bash
# MarsDocSync:
# docs:
# - docs/design-docs/release-versioning.md
# - docs/features/F-002-zero-config-shell-path.md
# - docs/features/F-009-release-update-lifecycle.md
set -euo pipefail

BINARY_NAME="${BINARY_NAME:-mars}"
REMOTE_NAME="${REMOTE_NAME:-origin}"
REMOTE_BRANCH="${REMOTE_BRANCH:-main}"
GO_CMD="${GO:-go}"

log() { printf '%s\n' "$*"; }
err() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

require_command() {
    command -v "$1" >/dev/null 2>&1 || err "$1 is required. Install $1 and retry."
}

require_command git
require_command "$GO_CMD"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" \
    || err "make update-tool must run inside a mars git checkout. Run make install for a local-only build."
cd "$repo_root"

if ! git remote get-url "$REMOTE_NAME" >/dev/null 2>&1; then
    err "No $REMOTE_NAME remote found. Run make install to install this checkout as-is, or add a remote before make update-tool."
fi

if [ -n "$(git status --porcelain)" ]; then
    err "The worktree has uncommitted changes. Commit or stash them, or run make install to install this checkout as-is."
fi

log "Fetching $REMOTE_NAME $REMOTE_BRANCH..."
git fetch "$REMOTE_NAME" "$REMOTE_BRANCH"
remote_ref="$REMOTE_NAME/$REMOTE_BRANCH"

if ! git rev-parse --verify --quiet "$remote_ref" >/dev/null; then
    err "Could not resolve $remote_ref after fetch. Check the remote branch name and retry."
fi

if ! git merge-base --is-ancestor HEAD "$remote_ref"; then
    err "This checkout cannot fast-forward to $remote_ref. Resolve divergence manually, then rerun make update-tool."
fi

if ! git merge-base --is-ancestor "$remote_ref" HEAD; then
    log "Fast-forwarding to $remote_ref..."
    git merge --ff-only "$remote_ref"
else
    log "Checkout is already up to date with $remote_ref."
fi

gobin="$("$GO_CMD" env GOBIN)"
gopath="$("$GO_CMD" env GOPATH)"
install_bin="$gobin"
if [ -z "$install_bin" ]; then
    install_bin="$gopath/bin"
fi
if [ -z "$install_bin" ]; then
    err "Could not resolve Go install directory from GOBIN or GOPATH."
fi

log "Installing $BINARY_NAME to $install_bin/$BINARY_NAME..."
CGO_ENABLED=0 "$GO_CMD" install ./cmd/mars

"$install_bin/$BINARY_NAME" path setup --install-dir "$install_bin"
log "Installed $BINARY_NAME to $install_bin/$BINARY_NAME"
"$install_bin/$BINARY_NAME" version
