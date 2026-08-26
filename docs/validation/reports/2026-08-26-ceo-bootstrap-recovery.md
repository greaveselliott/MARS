# CEO Bootstrap Recovery — Clean Target Validation

**Date:** 2026-08-26
**Classification:** foundation-owned runtime and generated-doctrine recovery
**Supporting ticket:** MH-062
**Source ref:** local build from `8e12721` plus this uncommitted repair
**Model:** local `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf`

## Primary Outcome Contract

**Primary Outcome:** A fresh CEO bootstrap completes its strategy work and
hands the required planning artifact to COO without expanding CEO authority.

**Primary Pass Gate:** The local-model noughts-and-crosses replay records a
completed CEO `exec_plan` disposition for COO and the queue claims the COO job.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** none

**Next Primary Action:** Commit and push the validated foundation repair.

**Supporting Evidence:** Trace `tr-1787709582203156000`, COO job
`d0380da8-4f28-4e33-bbc4-aba47ff112a5`, and the focused source checks below.

## Scope

The preceding clean noughts-and-crosses factory run (`tr-1787707973149210000`)
showed a CEO correctly blocked from COO-owned feature and active-plan writes,
but it failed to record a disposition. This run validates the foundation fix:
CEO retains the boundary and must complete its handoff to COO.

## Target and Command

- Target: `/private/tmp/mars-noughts-and-crosses-validation`
- Baseline commits: `538f903` harness init and `f1e7d21` product brief
- Command:

  ```text
  mars start --repo /private/tmp/mars-noughts-and-crosses-validation --new-lifecycle \
    --concurrency 1 --debug \
    --log-file /private/tmp/mars-noughts-and-crosses-validation-factory.log \
    --execution-profile host --acknowledge-host-execution --yes
  ```

## Result: Pass

At 03:00:24 BST the CEO finished successfully after nine model calls and
recorded disposition trace `tr-1787709582203156000`:

- `status=completed`
- `next_need=exec_plan`
- `suggested_role=coo`
- expected output: active execution plan and canonical feature contract

The orchestrator immediately enqueued COO job
`d0380da8-4f28-4e33-bbc4-aba47ff112a5` and the single worker claimed it. The
run was then stopped intentionally at that handoff checkpoint; no target code
was claimed or needed for this foundation recovery validation. The stopped COO
job is therefore not a product failure.

No CEO planning-write authority was added. Focused executor regression covers
the blocked-write path itself: after the first rejected CEO planning write,
only `git_status`, `git_commit`, and `job_disposition_record` are permitted,
and the policy response names the completed `exec_plan` handoff to COO.

## Additional Checks

- `make check` — passed (race-enabled repository suite)
- `go test -count=1 ./internal/tools ./internal/personas ./internal/scanner ./internal/orchestration`
- `go test ./internal/docsconsistency/...`
- `mars docsync audit --repo .` — `docsync: checked 366 files, findings 0`

## Cleanup

Sent `SIGINT` to the factory after COO was claimed. A process check confirmed
that neither the factory nor its local llama server remained running. The
disposable target, log, and isolated SQLite database remain available for audit.
