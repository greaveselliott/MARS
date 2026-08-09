---
name: github-private-release-auth
scope: all
---

# GitHub Release Access Skill

Use this before update, release verification, install repair, version-drift
remediation, or any workflow that needs MARS GitHub Release access.

## Workflow

1. Run `mars auth github check` or the `github_auth_check` tool. The check makes
   one exact no-redirect anonymous request to the official release metadata
   endpoint and reports `anonymous`, `authenticated`, or `unavailable`.
2. Continue without credentials when access is `anonymous`. Only an exact 401,
   403, or 404 from that official endpoint may resolve an optional credential
   and retry once at the exact same origin and path.
3. For a private fork or access that requires authentication, run
   `gh auth login`, then `mars auth github setup`. Headless installs can use
   `GH_TOKEN`, `GITHUB_TOKEN`, or
   `mars auth github setup --token <token>` with repository contents
   read access.
4. Use `mars auth github clear-local` to remove only the `github_token` stored
   in MARS config. It does not change environment credentials, GitHub CLI auth,
   GitHub App credentials, repositories, remotes, or remote state.

## Security Rules

- Never paste token values into chat, docs, commits, traces, tickets, logs, or
  tool output.
- Prefer anonymous public access when it is available.
- Store local fallback tokens only through `mars auth github setup`;
  they belong under `~/.mars/`, never in a target repository.

## Stop Conditions

- Stop and return a blocker when release access is `unavailable` and the
  required workflow cannot proceed.
- Stop and record a release blocker when asset verification depends on missing
  credentials.
