# Argos Discovery Dogfood Round 3

Date: 2026-04-30

## Scope

Action policy dogfood validation.

Cases:

- `progressive_disclosure_and_citation_guard`
- `partial_lesson_without_rule`
- `weak_single_generic_term`
- `none_payment_webhook`
- `map_inventory_normal`

Isolation:

- Each case ran in a fresh runner session.
- Runners received only the case input, workspace path, binary path, allowed
  flow, and reporting instructions.
- Runners did not receive golden expected values.
- The shared fixture workspace indexed 9 official knowledge items.

## Result

Overall: `pass`

- Strong workflow: pass
- Partial workflow: pass after rerun with explicit MCP line protocol guidance
- Weak discovery: pass
- None discovery: pass
- Map inventory: pass

## Findings

### Strong

The strong workflow returned:

- `action_policy.authority: strong`
- `action_policy.load: recommended`
- `action_policy.cite: after_loaded_and_used`
- `action_policy.claim: allowed`

The runner loaded `rule:backend.auth-refresh.v1` through MCP and cited only
that loaded-and-used ID.

Status: pass.

### Partial

The partial workflow returned:

- `action_policy.authority: partial`
- `action_policy.load: allowed`
- `action_policy.cite: after_loaded_and_used`
- `action_policy.claim: must_mention_gap`

The first runner understood the policy but did not know how to complete MCP
JSON-RPC loading from the prompt alone, so it reported review-needed without
bypassing progressive disclosure. A rerun with explicit newline-delimited MCP
JSON-RPC instructions loaded `lesson:backend.auth-debug.v1`, cited only that
loaded-and-used ID, and explicitly mentioned the coverage gap.

Status: pass with harness note.

Harness note: future dogfood prompts should include a minimal MCP JSON-RPC
example when a runner is expected to call `get_knowledge_item` or
`cite_knowledge`.

### Weak

The weak case returned:

- `action_policy.authority: weak`
- `action_policy.load: forbidden`
- `action_policy.cite: forbidden`
- `action_policy.claim: forbidden`

The runner inspected summaries/map output only, did not load, did not cite, and
did not treat weak auth/session matches as authoritative for the barcode scanner
task.

Status: pass.

### None

The none case returned:

- `action_policy.authority: none`
- `action_policy.load: forbidden`
- `action_policy.cite: forbidden`
- `action_policy.claim: forbidden`

The runner treated missing knowledge hints as gap indicators only and did not
use unrelated backend/auth inventory as task guidance.

Status: pass.

### Map Inventory

The map case returned:

- `action_policy.authority: inventory`
- `action_policy.load: forbidden`
- `action_policy.cite: forbidden`
- `action_policy.claim: forbidden`

The runner treated map output as orientation only. It did not load full items,
call citation, or make task-level claims from inventory.

Status: pass.

## Decision

`action_policy` is clear enough to become the stable safety boundary for
optional semantic retrieval. Future semantic providers may improve recall, but
they must not bypass policy:

- weak and none still forbid load, cite, and Argos-backed claims;
- map inventory remains orientation only;
- partial requires gap disclosure when applied;
- citation remains valid only after loading and using full items.
