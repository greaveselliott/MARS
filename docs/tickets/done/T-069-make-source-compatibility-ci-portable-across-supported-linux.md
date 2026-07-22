---
id: T-069
title: Make source compatibility CI portable across supported Linux lanes
priority: high
complexity: small
work_type: enabler
bdd_scenarios: ["F-018-S003"]
end_to_end_evidence: not_applicable
evidence_links: ["https://github.com/greaveselliott/MARS/actions/runs/29898672813"]
verified_by: "QA, Security, and foundation-maintainer"
owner: "engineer"
last_attempt: "2026-07-22: exact-head source-compatibility run 29898672813 passed both supported Go lanes and the below-minimum rejection lane"
blocker: "none"
blocked_by: []
trace_id: "github-actions:29898672813"
next_action: "Complete; resume T-068 checkpoint C state reconciliation and final sign-offs without publication."
dedupe_key: "ci:source-compatibility-linux-portability"
metadata:
  classification: "foundation-owned"
  publication_authority: "denied"
  supports: "F-018-S003"
source: current-operating-plan.md — T-068 checkpoint C failed-closed CI reconciliation
created: 2026-07-22
depends_on: [T-067]
---

# T-069: Make source compatibility CI portable across supported Linux lanes

## Context

T-068 checkpoint C found that the last twelve source-compatibility runs were red. Exact-head run 29896978096 failed identically in both supported Go lanes because Linux fixtures used a world-writable temporary ancestor, a bare Git fixture retained the platform default branch, serve tests depended on host model eligibility, and release-tag policy lowercased explicit HEAD. Production safety correctly rejected the unsafe path; one release-policy failure is a real case-sensitive-filesystem bug.

## Requirements

1. Give signed-replacement tests an owner-controlled secure temporary root without weakening production admission.
2. Make the update-script bare repository explicitly use main.
3. Make serve tests deterministic on low-memory CI through a test-only eligible-hardware seam while preserving real routing/preflight tests.
4. Preserve release tag and target operand case locally; explicit HEAD must resolve on case-sensitive filesystems.
5. Change no release version, tag, Release, signature, upload, visibility, model threshold, or runtime safety boundary.

## Acceptance criteria

- [x] The exact failed tests pass under Go 1.25.12 and Go 1.26.5.
- [x] Source compatibility passes at one exact pushed head in both supported lanes and the below-minimum lane still rejects Go 1.25.11.
- [x] Signed replacement still rejects genuinely unsafe ancestors and the release parser retains explicit operand case.
- [x] QA and Security approve the bounded diff; T-068 resumes checkpoint C without publication authority.

## Evidence

- Commit `03008f74477ba1b5fa8ea85c3af55e448ea037ac` passed exact-head source-compatibility run `29898672813`: Go `1.25.12` job `88854289553`, Go `1.26.5` job `88854289567`, and the expected Go `1.25.11` rejection job `88854289555` all completed successfully.
- Exact local package gates passed under Go `1.25.12` and `1.26.5`; affected-package vet and source-wide DocSync (`352` files, `0` findings) passed. The explicit `0777` ancestor fixture remains fail-closed before lock or transaction creation, and case-insensitive Git executable recognition cannot bypass case-preserved `HEAD` policy.
- QA and Security are GO. No version, tag, Release, signature, upload, visibility, threshold, or publication authority changed.
