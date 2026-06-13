# Validation Profiles

Reusable project briefs for ephemeral foundation replays (AD-293). Each file is
YAML frontmatter plus the `spec.md` body written into a fresh validation run.

| Profile | Archetype | Use |
| --- | --- | --- |
| `static-browser-todo` | static-browser | Bootstrap, planning, static smoke |
| `inventory-api` | api-service | Go HTTP JSON API lifecycle |
| `depot-supplies-api` | api-service | Second API/service shape for batched replays |
| `package-managed-frontend` | package-managed-frontend | Vite/React build and dev scripts |

Create a run:

```bash
node scripts/validation-target.mjs create --profile inventory-api --label wsd-closure
mars-harness start --repo ../demo/validation-runs/run-<timestamp>-inventory-api-wsd-closure
```

List and discard:

```bash
node scripts/validation-target.mjs list
node scripts/validation-target.mjs discard run-<timestamp>-inventory-api-wsd-closure
node scripts/validation-target.mjs cleanup --keep 3
```

See [foundation-operating-model.md](../../design-docs/foundation-operating-model.md)
(AD-293) and [validation/README.md](../README.md).
