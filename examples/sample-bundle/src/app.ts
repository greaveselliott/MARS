/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/role-customization.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-019-typescript-monorepo-docsync.md
*/
function add(a: number, b: number): number {
  return a + b;
}

const result: string = add(1, 2); // Type error: number assigned to string
console.log(result);
