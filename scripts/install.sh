#!/bin/bash -p
# MarsDocSync:
# docs:
# - docs/design-docs/release-versioning.md
# - docs/features/F-009-release-update-lifecycle.md
# - docs/features/F-018-goreleaser-distribution.md
if [[ "$-" == *p* ]]; then
    builtin unset mars_inherited_gh_token mars_inherited_github_token
    mars_inherited_gh_token="${GH_TOKEN:-}"
    mars_inherited_github_token="${GITHUB_TOKEN:-}"
    GH_TOKEN= GITHUB_TOKEN= /usr/bin/env -i \
        "PATH=${PATH:-}" \
        "HOME=${HOME:-}" \
        "TMPDIR=${TMPDIR:-}" \
        /bin/bash -p -s -- "$@" \
        3<<<"$mars_inherited_gh_token" 4<<<"$mars_inherited_github_token" <<'MARS_INSTALL_BODY'
IFS= builtin read -r mars_gh_token <&3 || builtin true
IFS= builtin read -r mars_github_token <&4 || builtin true
exec 3<&- 4<&-
if [[ -e /dev/fd/3 || -e /dev/fd/4 ]]; then
    builtin printf '%s\n' 'Error: bootstrap credential transport descriptors did not close; stop and retry from a clean terminal.' >&2
    builtin exit 1
fi
builtin set -euo pipefail

builtin readonly MARS_COMMAND="github.com/greaveselliott/mars/cmd/mars"
builtin readonly MARS_MODULE="github.com/greaveselliott/mars"
builtin readonly MIN_GO_MAJOR=1
builtin readonly MIN_GO_MINOR=25
builtin readonly MIN_GO_PATCH=12
builtin readonly PUBLIC_GO_PROXY="https://proxy.golang.org"
builtin readonly PUBLIC_GO_SUMDB="sum.golang.org"

staging_dir=""
staging_prefix=""
go_path=""

log() {
    builtin printf '%s\n' "$*"
}

fail() {
    builtin printf 'Error: %s\n' "$*" >&2
    builtin exit 1
}

usage() {
    builtin printf '%s\n' \
        'Usage: install.sh vMAJOR.MINOR.PATCH ABSOLUTE_INSTALL_DIR' \
        'Example: install.sh v0.69.1 "$HOME/.local/bin"' >&2
}

cleanup() {
    builtin local dir="${staging_dir:-}" suffix remove_status
    suffix="${dir#"${staging_prefix:-}"}"
    if [[ -z "$dir" || -z "${staging_prefix:-}" || "$dir" != "${staging_prefix}"* || ! "$suffix" =~ ^[A-Za-z0-9]{8}$ ]]; then
        [[ -z "$dir" ]] && builtin return 0
        builtin return 1
    fi
    if [[ ! -e "$dir" && ! -L "$dir" ]]; then
        staging_dir=""
        builtin return 0
    fi
    if [[ ! -d "$dir" || -L "$dir" || ! -O "$dir" ]]; then
        builtin return 1
    fi
    builtin set +e
    builtin command /bin/rm -rf -- "$dir" >/dev/null 2>&1
    remove_status="$?"
    builtin set -e
    if (( remove_status != 0 )); then
        builtin return 1
    fi
    if [[ -e "$dir" || -L "$dir" ]]; then
        builtin return 1
    fi
    staging_dir=""
    builtin return 0
}

cleanup_on_exit() {
    builtin local status="$?"
    if ! cleanup; then
        builtin printf '%s\n' 'Warning: bootstrap staging cleanup incomplete; inspect the owner-controlled mars-bootstrap directory under the original TMPDIR before retrying.' >&2
    fi
    builtin exit "$status"
}

require_exact_version() {
    builtin local version="$1"
    if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
        fail 'version must be one exact stable semantic tag such as v0.69.1; latest, branches, pseudo-versions, prereleases, and build metadata are not accepted.'
    fi
}

require_install_dir() {
    builtin local install_dir="$1"
    if [[ "$install_dir" != /* || "$install_dir" == / || ! -d "$install_dir" || -L "$install_dir" || ! -O "$install_dir" || ! -w "$install_dir" ]]; then
        fail 'the install directory must be an existing, writable, owner-controlled absolute directory; create one with mode 0700 and retry.'
    fi
}

resolve_go_path() {
    builtin local found go_dir
    found="$(builtin type -P go 2>/dev/null)" \
        || fail 'Go 1.25.12 or newer is required for first bootstrap; install Go from go.dev and retry.'
    if [[ "$found" != /* ]]; then
        fail 'the Go command must resolve to an absolute executable path; repair PATH and retry.'
    fi
    go_dir="$(builtin cd -P -- "${found%/*}" 2>/dev/null && builtin pwd -P)" \
        || fail 'the Go command path could not be resolved; repair the Go installation and retry.'
    go_path="${go_dir%/}/${found##*/}"
    if [[ ! -f "$go_path" || ! -x "$go_path" ]]; then
        fail 'the resolved Go command is not an executable regular file; repair the Go installation and retry.'
    fi
}

sanitize_build_environment() {
    PATH=/usr/bin:/bin
    builtin export PATH
    builtin unset -f builtin command stat mktemp chmod mkdir rm cd pwd printf read type set readonly local export unset trap umask return exit true break 2>/dev/null || builtin true
    builtin unset BASH_ENV ENV CDPATH GOROOT GOCACHEPROG GOEXPERIMENT GOROOT_FINAL
    builtin unset GCCGO CC CXX AR FC PKG_CONFIG GO_EXTLINK_ENABLED
    builtin unset CGO_CFLAGS CGO_CPPFLAGS CGO_CXXFLAGS CGO_FFLAGS CGO_LDFLAGS
    builtin unset GOOS GOARCH GO386 GOAMD64 GOARM GOARM64 GOMIPS GOMIPS64 GOPPC64 GORISCV64 GOWASM
    builtin export GOAUTH=off
    builtin export CGO_ENABLED=0
}

run_go_version() {
    GOENV=off \
    GOTOOLCHAIN=local \
    GOWORK=off \
    GOFLAGS= \
    GOAUTH=off \
    CGO_ENABLED=0 \
    "$go_path" env GOVERSION
}

require_go_version() {
    builtin local go_version major minor patch
    go_version="$(run_go_version 2>/dev/null)" \
        || fail 'Go version detection failed; repair the Go installation and retry.'
    if [[ ! "$go_version" =~ ^go([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        fail 'Go reported an unsupported version; install stable Go 1.25.12 or newer and retry.'
    fi
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    patch="${BASH_REMATCH[3]}"
    if (( major < MIN_GO_MAJOR ||
          (major == MIN_GO_MAJOR && minor < MIN_GO_MINOR) ||
          (major == MIN_GO_MAJOR && minor == MIN_GO_MINOR && patch < MIN_GO_PATCH) )); then
        fail 'Go 1.25.12 or newer is required for first bootstrap; upgrade Go and retry.'
    fi
}

run_trusted_go() {
    TMPDIR="$staging_dir/tmp" \
    GOTMPDIR="$staging_dir/tmp" \
    GOENV=off \
    GOTOOLCHAIN=local \
    GOWORK=off \
    GOFLAGS=-modcacherw \
    GOAUTH=off \
    CGO_ENABLED=0 \
    GOPROXY="$PUBLIC_GO_PROXY" \
    GONOPROXY= \
    GOPRIVATE= \
    GONOSUMDB= \
    GOINSECURE= \
    GOSUMDB="$PUBLIC_GO_SUMDB" \
    GOBIN="$staging_dir/bin" \
    GOMODCACHE="$staging_dir/modcache" \
    GOCACHE="$staging_dir/buildcache" \
    "$go_path" "$@"
}

directory_owner_and_mode() {
    builtin local path="$1"
    if builtin command /usr/bin/stat -f '%u %Lp' "$path" >/dev/null 2>&1; then
        builtin command /usr/bin/stat -f '%u %Lp' "$path"
        builtin return
    fi
    builtin command /usr/bin/stat -c '%u %a' "$path" 2>/dev/null \
        || fail 'temporary-directory permissions could not be inspected; repair TMPDIR and retry.'
}

require_safe_staging_root() {
    builtin local requested_root="${TMPDIR:-/tmp}"
    builtin local resolved_root current identity owner mode_text mode
    if [[ "$requested_root" != /* || ! -d "$requested_root" || ! -w "$requested_root" ]]; then
        fail 'the temporary-directory root must be an existing writable absolute directory; repair TMPDIR and retry.'
    fi
    resolved_root="$(builtin cd -P -- "$requested_root" 2>/dev/null && builtin pwd -P)" \
        || fail 'the temporary-directory root could not be resolved; repair TMPDIR and retry.'
    current="$resolved_root"
    while builtin true; do
        if [[ ! -d "$current" || -L "$current" ]]; then
            fail 'the resolved temporary-directory ancestry must contain only real directories; repair TMPDIR and retry.'
        fi
        identity="$(directory_owner_and_mode "$current")"
        builtin read -r owner mode_text <<< "$identity"
        if [[ ! "$owner" =~ ^[0-9]+$ || ! "$mode_text" =~ ^[0-7]{3,6}$ ]]; then
            fail 'temporary-directory ancestry permissions are malformed; repair TMPDIR and retry.'
        fi
        mode=$((8#$mode_text))
        if (( owner == EUID && !(mode & 0022) )); then
            builtin true
        elif (( owner == 0 && !(mode & 0022) )); then
            builtin true
        elif (( owner == 0 && (mode & 01000) && (mode & 0002) )); then
            builtin true
        elif (( owner == EUID )); then
            fail 'user-owned temporary-directory ancestry must not be group- or world-writable; use a private TMPDIR and retry.'
        elif (( owner != 0 )); then
            fail 'temporary-directory ancestry must be owned only by the current user or root; repair TMPDIR and retry.'
        else
            fail 'writable root-owned temporary-directory ancestry must be world-writable and sticky; repair TMPDIR and retry.'
        fi
        [[ "$current" == / ]] && builtin break
        current="${current%/*}"
        [[ -n "$current" ]] || current=/
    done
    staging_prefix="${resolved_root%/}/mars-bootstrap."
}

require_private_staging_dir() {
    builtin local identity owner mode_text mode suffix
    suffix="${staging_dir#"$staging_prefix"}"
    if [[ "$staging_dir" != "${staging_prefix}"* || ! "$suffix" =~ ^[A-Za-z0-9]{8}$ || ! -d "$staging_dir" || -L "$staging_dir" || ! -O "$staging_dir" ]]; then
        fail 'the bootstrap staging directory identity is invalid; no release asset was installed.'
    fi
    identity="$(directory_owner_and_mode "$staging_dir")"
    builtin read -r owner mode_text <<< "$identity"
    if [[ ! "$owner" =~ ^[0-9]+$ || ! "$mode_text" =~ ^[0-7]{3,6}$ ]]; then
        fail 'the bootstrap staging directory permissions are malformed; no release asset was installed.'
    fi
    mode=$((8#$mode_text))
    if (( owner != EUID || mode != 0700 )); then
        fail 'the bootstrap staging directory must be newly created with owner-only mode 0700; no release asset was installed.'
    fi
}

verify_staged_identity() {
    builtin local binary="$1"
    builtin local version="$2"
    builtin local metadata line record_kind record_path record_version record_sum extra
    builtin local path_records=0
    builtin local module_records=0

    metadata="$(run_trusted_go version -m "$binary" 2>/dev/null)" \
        || fail 'the staged bootstrap has no readable Go build identity; repair Go and retry.'

    while IFS= builtin read -r line; do
        case "$line" in
            $'\tpath\t'*)
                IFS=$'\t' builtin read -r record_kind record_path record_version extra <<< "${line#$'\t'}"
                if [[ "$record_kind" != path || "$record_path" != "$MARS_COMMAND" || -n "${record_version:-}" || -n "${extra:-}" ]]; then
                    fail 'the staged bootstrap command identity is not canonical; no release asset was installed.'
                fi
                ((path_records += 1))
                ;;
            $'\tmod\t'*)
                IFS=$'\t' builtin read -r record_kind record_path record_version record_sum extra <<< "${line#$'\t'}"
                if [[ "$record_kind" != mod || "$record_path" != "$MARS_MODULE" || "$record_version" != "$version" || ! "$record_sum" =~ ^h1:[A-Za-z0-9+/]{43}=$ || -n "${extra:-}" ]]; then
                    fail 'the staged bootstrap module identity or version is not canonical; no release asset was installed.'
                fi
                ((module_records += 1))
                ;;
            $'\t=>\t'*)
                fail 'the staged bootstrap contains a module replacement; no release asset was installed.'
                ;;
        esac
    done <<< "$metadata"

    if (( path_records != 1 || module_records != 1 )); then
        fail 'the staged bootstrap identity is incomplete or ambiguous; no release asset was installed.'
    fi
}

main() {
    if (( $# != 2 )); then
        usage
        fail 'provide exactly one stable release tag and one final install directory.'
    fi

    builtin local version="$1"
    builtin local install_dir="$2"
    require_exact_version "$version"
    require_install_dir "$install_dir"
    resolve_go_path
    sanitize_build_environment
    require_go_version

    require_safe_staging_root
    builtin umask 077
    staging_dir="$(builtin command /usr/bin/mktemp -d "${staging_prefix}XXXXXXXX" 2>/dev/null)" \
        || fail 'could not create an owner-controlled bootstrap staging directory; repair temporary-directory permissions and retry.'
    builtin trap cleanup_on_exit EXIT
    builtin trap 'builtin exit 1' HUP INT TERM
    require_private_staging_dir
    builtin command /bin/mkdir -m 0700 "$staging_dir/bin" "$staging_dir/modcache" "$staging_dir/buildcache" "$staging_dir/tmp" \
        || fail 'could not prepare owner-controlled bootstrap staging; no release asset was installed.'

    TMPDIR="$staging_dir/tmp"
    GOTMPDIR="$staging_dir/tmp"
    builtin export TMPDIR GOTMPDIR

    log 'Building the exact-version MARS bootstrap through the public Go proxy and checksum database...'
    if ! run_trusted_go install "${MARS_COMMAND}@${version}" >/dev/null 2>&1; then
        fail 'the exact-version Go/SumDB bootstrap failed; verify network access to proxy.golang.org and sum.golang.org, then retry.'
    fi

    builtin local staged_binary="$staging_dir/bin/mars"
    if [[ ! -f "$staged_binary" || -L "$staged_binary" || ! -O "$staged_binary" || ! -x "$staged_binary" ]]; then
        fail 'Go did not produce one owner-controlled staged MARS command; no release asset was installed.'
    fi
    verify_staged_identity "$staged_binary" "$version"

    log 'Staged identity verified; delegating archive and signature verification to MARS...'
    if ! GH_TOKEN="$mars_gh_token" GITHUB_TOKEN="$mars_github_token" \
        "$staged_binary" update tool --version "$version" --install-dir "$install_dir" --bootstrap-exact-module --skip-shell-path; then
        fail 'signed release installation failed; follow the updater recovery guidance, preserve .mars-update.transaction if instructed, and retry the same exact tag only after recovery.'
    fi
    if ! cleanup; then
        builtin trap - EXIT HUP INT TERM
        fail 'binary installed but bootstrap staging cleanup is incomplete; inspect the owner-controlled mars-bootstrap directory under the original TMPDIR before continuing.'
    fi
    builtin trap - EXIT HUP INT TERM
    log 'MARS signed release installation completed.'
}

main "$@"
MARS_INSTALL_BODY
else
    /usr/bin/printf '%s\n' 'Error: execute ./scripts/install.sh directly so #!/bin/bash -p establishes startup isolation.' >&2
    [[ 1 == 0 ]]
fi
