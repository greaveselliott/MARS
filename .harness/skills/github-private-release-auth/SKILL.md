---
name: github-private-release-auth
scope: all
---

# GitHub Private Release Auth Skill

Use this before update, release verification, install repair, version-drift
remediation, or any workflow that needs private MARS GitHub Release
assets.

## Workflow

1. Run `mars auth github check` or the `github_auth_check` tool.
2. If auth is missing, ask the operator to run `gh auth login`, then
   `mars auth github setup`. Setup verifies access and stores a
   GitHub CLI token as the owner-only local fallback so later update runs do not
   depend on keychain access.
3. For headless installs, use `GH_TOKEN`, `GITHUB_TOKEN`, or
   `mars auth github setup --token <token>` with repository contents
   read access.
4. Retry the blocked update or release command only after the auth check is
   `ok`.

## Security Rules

- Never paste token values into chat, docs, commits, traces, tickets, logs, or
  tool output.
- Store local fallback tokens only through `mars auth github setup`;
  they belong under `~/.mars/`, never in a target repository.

## Stop Conditions

- Stop and return a blocker when the token is rejected, SSO authorization is
  required, or the authenticated account cannot see the private release repo.
- Stop and record a release blocker when asset verification depends on missing
  credentials.
