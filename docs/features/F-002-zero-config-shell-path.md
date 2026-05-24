# F-002: Zero-Config Shell PATH

- Feature ID: F-002
- Goals: G-004, G-002
- Status: passing
- Owner: Release Manager

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-002-S001 - Source `make install` configures shell PATH after installing the binary.
2. F-002-S002 - `mars-harness setup` configures shell PATH during first-run setup.
3. F-002-S003 - `mars-harness update tool` configures shell PATH after reinstalling/upgrading the binary.
4. F-002-S004 - PATH setup is idempotent and does not duplicate profile snippets.
5. F-002-S005 - Unsupported shells get clear manual remediation instead of a misleading success claim.
6. F-002-S006 - Source `make update-tool` refreshes a clean clone and configures shell PATH after installing the updated binary.

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

### F-002-S006: Source Clone Update Configures PATH

Given a developer has a clean Mars Harness source checkout with an `origin/main` remote
When the developer runs `make update-tool`
Then the checkout fast-forwards only when it can do so safely
And Go installs `mars-harness` into the resolved Go binary directory
And the installed binary runs `mars-harness path setup --install-dir <dir>`
And the command prints the installed `mars-harness version`

Given the source checkout has uncommitted changes, no `origin` remote, or divergent history
When the developer runs `make update-tool`
Then the command refuses to install and tells the developer to resolve the checkout or use `make install` for a local-only install

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
- F-002-S006: `go test ./internal/selfupdate -run TestUpdateToolScript`
