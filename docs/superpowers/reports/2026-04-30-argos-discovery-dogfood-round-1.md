# Argos Discovery Dogfood Round 1

Date: 2026-04-30

## Scope

Targeted rerun after tightening Round 0 guardrails.

Cases:

- `map_inventory_empty`
- `map_inventory_normal`
- `progressive_disclosure_and_citation_guard`

Isolation:

- Each case ran in a fresh runner session.
- Runners received only the case input, workspace path, binary path, allowed
  flow, and reporting instructions.
- Runners did not receive golden expected values.
- `map_inventory_empty` used a true empty workspace with registry files only.

## Result

Overall: `pass`

- `map_inventory_empty`: pass
- `map_inventory_normal`: pass
- `progressive_disclosure_and_citation_guard`: pass

## Findings

### Empty Workspace

The empty workspace indexed `0` knowledge items. `argos_map` for
`unknown-api` returned empty inventory and no groups. The runner did not load
or cite anything.

Status: fixed.

### Inventory-Only Citation Guard

The normal map case returned summary-level inventory for `mall-api`, including
the auth-refresh package and expected backend/security/account-related
inventory. The runner did not call `get_knowledge_item` or `cite_knowledge`.

Status: fixed.

### Progressive Disclosure And Citation

The workflow case returned `strong` coverage and discovered the expected refresh
token rule. The runner loaded only selected discovery IDs through MCP and cited
only IDs it loaded and applied.

Observation: the runner loaded and cited both the refresh-token rule and the
session-renewal decision. This is valid under the current guardrail because both
IDs came from discovery, were loaded, and were used. If Argos should optimize
for minimal citations, add an explicit evaluator rule such as "cite only the
smallest sufficient loaded set."

## Verification

Commands run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/adapters -count=1
```

Result: passed.

## Decision

The two Round 0 issues are addressed well enough to continue. The next useful
step is to expand discovery behavior rather than reworking the harness again:

- add missing-knowledge / weak-coverage dogfood cases to prove Argos does not
  over-search or over-cite when knowledge is absent;
- then decide whether semantic/vector retrieval is needed as an optional
  enhancement based on those misses.
