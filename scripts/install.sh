#!/bin/bash
set -euo pipefail

REPO="greaveselliott/mars-harness"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"
BINARY_NAME="mars-harness"

log()  { printf '  %s\n' "$*"; }
info() { printf '\033[0;34m==> %s\033[0m\n' "$*"; }
err()  { printf '\033[0;31mError: %s\033[0m\n' "$*" >&2; exit 1; }

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "${os}" in
        linux)  echo "linux" ;;
        darwin) echo "darwin" ;;
        *)      err "Unsupported operating system: ${os}. Mars Harness supports linux and darwin." ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              err "Unsupported architecture: ${arch}. Mars Harness supports amd64 and arm64." ;;
    esac
}

resolve_version() {
    if [ "${VERSION}" = "latest" ]; then
        info "Resolving latest version..."
        VERSION="$(curl -sS -f "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' \
            | head -1 \
            | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')" \
            || err "Failed to resolve latest version. Check network connectivity and try VERSION=v1.0.0 $0"
        [ -n "${VERSION}" ] || err "Could not determine latest version from GitHub API."
    fi
    log "Version: ${VERSION}"
}

download_binary() {
    local os="$1" arch="$2"
    local artifact="${BINARY_NAME}-${os}-${arch}"
    local base_url="https://github.com/${REPO}/releases/download/${VERSION}"
    local binary_url="${base_url}/${artifact}"
    local checksums_url="${base_url}/checksums.txt"

    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "${tmpdir}"' EXIT

    info "Downloading ${artifact}..."
    curl -sS -fL -o "${tmpdir}/${artifact}" "${binary_url}" \
        || err "Download failed for ${artifact}. Verify that release ${VERSION} exists and includes binary assets at https://github.com/${REPO}/releases"

    info "Downloading checksums..."
    curl -sS -fL -o "${tmpdir}/checksums.txt" "${checksums_url}" \
        || err "Checksum download failed. Cannot verify binary integrity — aborting."

    info "Verifying checksum..."
    local expected_checksum
    expected_checksum="$(awk -v artifact="${artifact}" '$2 == artifact {print $1}' "${tmpdir}/checksums.txt")" \
        || err "Binary ${artifact} not found in checksums.txt. Release may be incomplete."
    [ -n "${expected_checksum}" ] \
        || err "Empty checksum for ${artifact}. Release may be corrupt."

    local actual_checksum
    if command -v sha256sum &>/dev/null; then
        actual_checksum="$(sha256sum "${tmpdir}/${artifact}" | awk '{print $1}')"
    elif command -v shasum &>/dev/null; then
        actual_checksum="$(shasum -a 256 "${tmpdir}/${artifact}" | awk '{print $1}')"
    else
        err "Neither sha256sum nor shasum found. Cannot verify binary integrity — aborting."
    fi

    if [ "${expected_checksum}" != "${actual_checksum}" ]; then
        rm -f "${tmpdir}/${artifact}"
        err "Checksum mismatch! Expected ${expected_checksum}, got ${actual_checksum}. Binary deleted — aborting."
    fi
    log "Checksum verified: ${actual_checksum}"

    info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
    if [ -w "${INSTALL_DIR}" ]; then
        mv "${tmpdir}/${artifact}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "${tmpdir}/${artifact}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
}

configure_path() {
    if "${INSTALL_DIR}/${BINARY_NAME}" path setup --install-dir "${INSTALL_DIR}"; then
        return 0
    fi
    log "Shell PATH setup did not complete automatically."
    log "Add it manually with: export PATH=\"${INSTALL_DIR}:\$PATH\""
}

main() {
    info "Mars Harness Installer"
    echo ""

    local os arch
    os="$(detect_os)"
    arch="$(detect_arch)"
    log "Platform: ${os}/${arch}"

    resolve_version
    download_binary "${os}" "${arch}"
    configure_path

    echo ""
    info "Installation complete!"
    log "Run 'mars-harness version' to verify."
    log "Run 'mars-harness setup' to get started."

    if ! command -v mars-harness &>/dev/null; then
        echo ""
        log "Note: ${INSTALL_DIR} may not be in your PATH."
        log "Add it with: export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

main
