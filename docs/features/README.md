# BDD Feature Contracts

BDD feature contracts define feature completeness. Walking skeleton is the
implementation strategy used to make scenarios pass through the thinnest real
end-to-end path.

V1 uses Markdown Given/When/Then. Do not introduce a custom Gherkin parser until
there is evidence that Markdown plus Go integration/E2E tests is not enough.

## Required Fields

- `Feature ID`
- `Goals`
- `Status`: draft, active, partially-passing, passing, superseded
- `Owner`
- `Scenario Schedule`
- `Out of Scope`
- `Descoped Scenarios`
- `Evidence`

## Rules

- BDD defines the full feature before implementation.
- The schedule is the ordered list of failing BDD scenarios.
- Tickets implement only the current failing scenario or scenario group.
- No feature ships until in-scope scenarios pass or are explicitly descoped.
- Every feature has at least one integration/E2E evidence link mapped to
  scenario IDs.
- Enabler work can complete without feature evidence, but it must not claim
  shipped feature value.
