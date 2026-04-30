# Argos Discovery Action Policy Design

Date: 2026-04-30

## Purpose

Argos Discovery already returns `coverage`, ranked summary items, and
`next_calls`. Dogfood rounds 1 and 2 show that agents can usually infer the
right behavior from those fields, but that still leaves too much judgment in the
agent prompt.

Add an explicit, machine-readable `action_policy` to discovery and map
responses so agents can decide whether to load, cite, mention gaps, or avoid
Argos-backed claims without interpreting prose.

This is a contract hardening step. It does not add vector search, new storage,
or new retrieval algorithms.

## Goals

- Make strong, partial, weak, none, and inventory-only behavior explicit.
- Prevent citation from map-only or weak/none discovery results.
- Keep progressive disclosure intact: summaries first, full bodies only when
  policy permits and the agent chooses relevant IDs.
- Preserve existing `coverage` and `next_calls` fields for compatibility.
- Give future semantic retrieval a fixed safety boundary it cannot bypass.

## Non-Goals

- Do not add a mandatory local vector database.
- Do not change ranking or scoring thresholds in this step.
- Do not add new MCP tools or CLI commands.
- Do not make `cite_knowledge` track whether an agent really used an item; that
  remains a caller contract enforced by policy, adapter text, and dogfood.

## Response Shape

Add this field to `DiscoveryResponse` and `MapResponse`:

```json
"action_policy": {
  "authority": "strong|partial|weak|none|inventory",
  "load": "recommended|allowed|forbidden",
  "cite": "after_loaded_and_used|forbidden",
  "claim": "allowed|must_mention_gap|forbidden",
  "reason": "short stable explanation"
}
```

Field semantics:

- `authority`: the policy-level authority state. For discovery, this mirrors
  `coverage.status`. For map, it is always `inventory`.
- `load`: whether full body loading should happen.
- `cite`: whether citations may be created.
- `claim`: whether the agent may make Argos-backed claims from the result.
- `reason`: compact text for adapter, CLI, MCP, and dogfood reports.

## Policy Matrix

| Context | authority | load | cite | claim |
| --- | --- | --- | --- | --- |
| Strong discovery | `strong` | `recommended` | `after_loaded_and_used` | `allowed` |
| Partial discovery | `partial` | `allowed` | `after_loaded_and_used` | `must_mention_gap` |
| Weak discovery | `weak` | `forbidden` | `forbidden` | `forbidden` |
| None discovery | `none` | `forbidden` | `forbidden` | `forbidden` |
| Map inventory | `inventory` | `forbidden` | `forbidden` | `forbidden` |

Strong and partial still require progressive loading before citation. Summary
results alone are never enough to cite.

Partial allows loading because a related item may be useful, but final answers
must mention that Argos coverage is incomplete when relying on that knowledge.

Weak allows summary inspection only. It may explain that related low-confidence
items exist, but it must not become authoritative project guidance.

None means the agent should proceed without Argos-specific claims and use
missing knowledge hints as gap indicators only.

Map inventory is orientation. It can tell the agent what exists, but it does not
authorize citation or Argos-backed claims for the current task.

## Compatibility

Existing clients can ignore `action_policy`.

`coverage` remains the human-readable discovery assessment.

`next_calls` remains a convenience list for common follow-up calls. The policy
is the stronger contract:

- if `load` is `forbidden`, `next_calls` must not include
  `get_knowledge_item`;
- if `cite` is `forbidden`, `next_calls` must not include `cite_knowledge`;
- if `cite` is `after_loaded_and_used`, citation is still only valid for IDs
  loaded through `get_knowledge_item` and actually applied.

## Interfaces

CLI JSON output should include `action_policy` automatically because it prints
the query response structs.

MCP tool responses should include `action_policy` automatically because they
serialize the same responses.

Generated adapters should mention the policy briefly:

- follow `action_policy.load` before loading full items;
- follow `action_policy.cite` before calling `cite_knowledge`;
- do not cite map inventory or weak/none discovery results;
- mention gaps when `action_policy.claim` is `must_mention_gap`.

## Testing

Add focused query tests for the policy matrix:

- strong discovery recommends load, permits citation after loaded-and-used, and
  allows claims;
- partial discovery allows load and citation after loaded-and-used, but requires
  gap mention;
- weak discovery forbids load, citation, and Argos-backed claims;
- none discovery forbids load, citation, and Argos-backed claims;
- map inventory forbids load, citation, and Argos-backed claims.

Extend golden discovery tests so existing cases assert policy fields alongside
coverage and next calls.

Add CLI/MCP smoke assertions that serialized responses include
`action_policy`.

Add adapter contract assertions for the new policy wording.

## Dogfood

Run Round 3 after implementation with fresh runner contexts:

- one strong workflow case;
- one partial workflow case;
- one weak case;
- one none case;
- one map-only inventory case.

Evaluator requirements:

- runner follows `action_policy` without needing expected values;
- weak/none do not load or cite;
- partial mentions the gap when applying loaded knowledge;
- map-only does not cite;
- strong still performs progressive loading before citation.

## Open Design Decision

No unresolved design decisions remain for this implementation slice. Semantic
retrieval should be designed later as a provider behind the same
`action_policy` contract.
