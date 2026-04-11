# Pipeline Learnings

Standing tracker for recurring failure patterns and fix recipes. Populated during harness operation. Consumed by the Reviewer meta-role and Pipeline Fixer role.

## Schema

Each entry follows this format:

```markdown
### [Date] — [Failure class]

**Run ID:** [job ID]
**Job:** [role name]
**Error excerpt:** [key error message]
**Root cause:** [one-line diagnosis]
**Fix applied:** [what was done]
**Prevention:** [concrete rule: "When [doing X], [always do Y]"]
```

## Entries

(None yet — populated during operation.)
