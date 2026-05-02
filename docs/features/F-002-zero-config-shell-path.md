# F-002: Zero-Config Shell PATH

- Feature ID: F-002
- Goals: G-004, G-002
- Status: passing
- Owner: Release Manager

## Scenario Schedule

1. F-002-S001 - Source `make install` configures shell PATH after installing the binary.
2. F-002-S002 - `mars-harness setup` configures shell PATH during first-run setup.
3. F-002-S003 - `mars-harness update tool` configures shell PATH after reinstalling/upgrading the binary.
4. F-002-S004 - PATH setup is idempotent and does not duplicate profile snippets.
5. F-002-S005 - Unsupported shells get clear manual remediation instead of a misleading success claim.

## Scenarios

### F-002-S001: Source Install Configures PATH

Given a developer runs `make install`
When Go installs `mars-harness` into the resolved Go binary directory
Then the installed binary runs `mars-harness path setup --install-dir <dir>` so future shells can resolve `mars-harness`

### F-002-S002: Setup Configures PATH

Given a user can invoke `mars-harness setup`
When setup runs
Then it detects the install directory and writes an idempotent profile snippet for the user's shell

### F-002-S003: Update Tool Configures PATH

Given a user runs `mars-harness update tool`
When the binary is reinstalled or upgraded
Then the target install directory is also configured in the user's shell profile

### F-002-S004: Idempotent Profile Updates

Given the shell profile already contains the Mars Harness managed PATH entry
When setup, update, or path setup runs again
Then no duplicate PATH block is written

### F-002-S005: Unsupported Shell Guidance

Given the user's shell is not one of Fish, Zsh, Bash, POSIX sh, Ksh, Csh, or Tcsh
When PATH setup runs
Then the command reports the unsupported shell and the install directory the user must add manually

## Out of Scope

- Rewriting system-wide `/etc/paths` or administrator-owned shell profiles.
- Modifying the current parent shell process after the command exits.
- Release-asset installation without Go; that remains tracked by `MH-031`.

## Descoped Scenarios

None.

## Evidence

- F-002-S001: `make install` invokes the installed binary's `path setup` command.
- F-002-S002: `go test ./internal/setup -run TestRun_testMode`
- F-002-S003: `go test ./internal/selfupdate`
- F-002-S004: `go test ./internal/shellpath -run TestEnsureZshIsIdempotent`
- F-002-S005: `go test ./internal/shellpath -run TestEvaluateUnsupportedShellDoesNotWrite`
