# T-079 Private Contribution Controls

**Date:** 2026-08-25  
**Ticket:** T-079  
**Repository:** `greaveselliott/MARS`  
**Source checkpoint:** `b807afa8de29756ccf0b6612135aad749e16ee07`  
**Visibility:** private  
**Ruleset:** `21491158`  
**Publication status:** not authorized or performed

## Primary Outcome Contract

**Primary Outcome:** The private repository now has conventional contribution files, fork-safe
pull-request automation, Dependabot configuration, private-safe community
settings, and one active default-branch ruleset. The transaction did not
change visibility, Pages, Apps, tags, Releases, release assets, immutable
Releases, publication authority, or public-only security controls.

**Primary Pass Gate:** The exact source and hosted settings pass local and
hosted verification while the repository remains private and every excluded
release, public-only, and account-wide surface remains unchanged.

**Primary Status:** `primary_blocked`

T-079 is awaiting the final evidence-source hosted run; the overall launch
remains blocked on the separately approved remaining gates.

**Current Primary Blocker:** the final T-079 evidence-source hosted run and
the separately approval-gated T-078 cleanup remain incomplete.

**Next Primary Action:** push and verify this exact evidence source, close
T-079, then return to the separately approval-gated T-078 hosted transaction.

**Supporting Evidence:** source commit `b807afa8de29756ccf0b6612135aad749e16ee07`,
hosted run `32901437495`, live ruleset `21491158`, and the exact pre/post
receipts recorded below.

Public CodeQL/code scanning, secret scanning, push protection, private
vulnerability reporting, Pages, and a genuine hostile-fork smoke remain
assigned to T-080 after a separately approved visibility change.

## Source Contract

Commit `b807afa8de29756ccf0b6612135aad749e16ee07` added:

- `CODEOWNERS`, a pull-request template, issue forms, DCO enforcement, and
  community/conduct/governance/security/support documents;
- weekly Dependabot updates for the root Go module, the notices tool module,
  and full-SHA GitHub Actions;
- a `pull_request`-only contribution workflow with `contents: read`, no
  secrets, no OIDC, no write permission, and only a GitHub-hosted runner; and
- an active default-branch ruleset contract with an explicit repository-admin
  bypass for the documented maintainer trunk workflow.

Every checked-in action is GitHub-owned and pinned to a full commit SHA.
Hosted source-compatibility run
[`32901437495`](https://github.com/greaveselliott/MARS/actions/runs/32901437495)
completed successfully for that exact commit.

## Pre-Mutation Revalidation

Immediately before mutation, local `HEAD` and `origin/main` were clean and
equal at `b807afa8de29756ccf0b6612135aad749e16ee07`. GitHub reported:

- private visibility, `main` as default, Issues enabled, Discussions and Pages
  disabled, and Wiki enabled;
- no rulesets or branch protection;
- Actions enabled with `allowed_actions=all`, SHA pinning disabled, a read-only
  default workflow token, and workflow approval authority disabled;
- private-fork workflows disabled, with no write-token or secret sharing;
- zero Actions secrets, zero Actions variables, and zero Dependabot secrets;
- vulnerability alerts and automated security fixes disabled; and
- only `greaveselliott` with repository-admin authority.

Any drift from those facts was a stop condition.

## Exact Hosted Transaction

The owner authorized Item 9. The transaction performed only these reversible
repository-scoped changes:

1. Actions changed to `selected` with full-length SHA pinning required.
2. The selected-actions policy allows GitHub-owned actions only; verified
   Marketplace actions and custom patterns remain disallowed.
3. Private-fork pull-request workflows were enabled with approval required,
   read-only tokens, and no secrets or variables.
4. Discussions were enabled while visibility remained private.
5. Dependabot vulnerability alerts and automated security updates were
   enabled.
6. The exact `dependencies` label was created.
7. Ruleset `21491158`, `Protect main and require contributor review`, was
   created for the default branch.

The ruleset rejects deletion and non-fast-forward updates for contributors,
requires one approving review, CODEOWNERS review, stale-review dismissal,
last-push approval, unattributed-change approval, resolved review threads,
strict required checks, and the exact contexts `DCO sign-off`,
`below-minimum`, `dependency-notices`, `supported-source (1.25.13)`, and
`supported-source (1.27.0)`. Repository administrators have the explicit
always-bypass required by MARS's documented maintainer trunk workflow.

## Postconditions

Read-only GitHub receipts proved:

- visibility is still private; Issues and Discussions are enabled; Pages is
  disabled; Wiki is unchanged;
- Actions are `selected`, SHA pinning is required, only GitHub-owned actions
  are allowed, default workflow authority remains read-only, and workflows
  cannot approve pull requests;
- private-fork workflows require approval and receive neither write tokens nor
  secrets/variables;
- Actions secrets, Actions variables, and Dependabot secrets remain zero;
- vulnerability alerts return the documented enabled response and automated
  security updates report `enabled=true`, `paused=false`;
- exactly one ruleset exists and its live rules match the checked-in contract;
- immutable Releases remain `enabled=false`, `enforced_by_owner=false`;
- 56 historical Releases and 301 tags remain; and
- no App, Page, Release, tag, asset, visibility, publication, or announcement
  mutation occurred.

## Validation

Before the hosted transaction, the source checkpoint passed full normal and
vet gates, focused race gates, DCO positive/missing/mismatched-signatory
fixtures, documentation consistency, DocSync with zero findings, formatting,
diff checks, and 86.2% coverage for `internal/release`. The exact hosted run is
linked above.

The explicit API-default field and this durable evidence are undergoing the
final local and hosted source gates before T-079 moves to done. A genuine
public fork test is deliberately not claimed by this private checkpoint.
