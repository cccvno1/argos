# Argos Naming Audit Design

Date: 2026-04-30

## Purpose

Argos should feel easy to understand for normal developers, including
non-native English speakers, without looking unprofessional when open sourced.
The product is a shared knowledge platform: people upload knowledge, and agents
use that knowledge while keeping clear boundaries between shared knowledge and
general reasoning.

This naming audit defines a simpler, more durable language for the agent-facing
tool surface, JSON responses, internal Go names, documentation, tests, and
adapters.

## Goals

- Prefer common developer words over abstract system-design language.
- Make the agent workflow obvious from tool names.
- Keep Argos professional enough for open source users and contributors.
- Remove terms that imply official truth, mandatory capture, or hidden
  platform authority.
- Rename development-stage interfaces in one coherent slice.
- Keep historical reports and old plans readable, but mark superseded language
  where needed.

## Non-Goals

- Do not change the storage model.
- Do not add semantic search, vectors, upload, or capture behavior.
- Do not redesign ranking behavior unless a rename exposes a real mismatch.
- Do not preserve compatibility aliases for development-stage names unless a
  test or adapter needs a short transition.

## Naming Principles

### Plain But Professional

Use short words that developers already know: `list`, `find`, `read`, `cite`,
`support`, `items`, `next_steps`, and `status`.

Avoid academic or overly architectural words in user-visible surfaces:
`authority`, `coverage`, `recall`, `disclosure`, `governed`,
`accountability`, and `provenance`.

These words may still appear in design history or comments when they are the
right technical explanation, but they should not be the primary public API.

### Action Names Should Match Agent Actions

Tool names should describe what the agent does:

```text
list -> find -> read -> cite
```

This is easier to learn than:

```text
map -> discover -> disclose -> account
```

### Shared Knowledge, Not Official Truth

Argos should say whether uploaded shared knowledge supports a task. It should
not imply that Argos is the final authority for the project.

Use `support` instead of `authority`.
Use `missing_needs` instead of `coverage_gaps`.

### Progressive Reading, Not Hidden Disclosure

The constraint remains the same: the agent should not load every full Markdown
body at once. The name should be simpler.

Use `read` for the public action and `read_status` for item-level guidance.
Avoid `disclosure` in public JSON.

### Internal Names Can Be Slightly More Technical

External API names should be simplest. Internal Go names may stay more
engineering-oriented when that improves clarity:

- external: `support`
- internal: `Support`
- external: `missing_needs`
- internal: `MissingNeed`
- external: `search_status`
- internal: `SearchStatus`

The internal names should still avoid terms that conflict with product meaning,
such as `Authority`.

## Recommended Tool Names

### Public MCP Tools

Use:

```text
argos_list_knowledge
argos_find_knowledge
argos_read_knowledge
argos_cite_knowledge
```

Meaning:

- `argos_list_knowledge`: show what shared knowledge exists for a project.
- `argos_find_knowledge`: find knowledge related to the current task.
- `argos_read_knowledge`: read the full body for one selected knowledge item.
- `argos_cite_knowledge`: create citations for knowledge actually used.

### Existing Tool Mapping

| Current name | New name | Reason |
| --- | --- | --- |
| `argos_map` | `argos_list_knowledge` | `list` is more common than `map`; the tool returns inventory. |
| `argos_discover` | `argos_find_knowledge` | `find` is direct and action-oriented. |
| `get_knowledge_item` | `argos_read_knowledge` | Adds Argos namespace and describes full-body reading. |
| `cite_knowledge` | `argos_cite_knowledge` | Adds Argos namespace and keeps the clear verb. |

`argos_context` and `argos_standards` can remain for now because they are
workflow shortcuts, not the new shared-knowledge search path. A later audit can
decide whether `argos_standards` should become `argos_find_standards`.

## Recommended Response Names

### `argos_find_knowledge` Response

Use:

```json
{
  "project": "mall-api",
  "phase": "implementation",
  "query": "refresh token session renewal",
  "search_status": {
    "semantic": {
      "status": "disabled",
      "reason": "Semantic search is optional and not configured."
    }
  },
  "support": {
    "level": "partial",
    "confidence": 0.72,
    "reason": "Some shared knowledge matches this task, but not every need is supported.",
    "recommendation": "Read the matched items, then separate shared knowledge from general reasoning."
  },
  "usage": {
    "read": "allowed",
    "cite": "after_read_and_used",
    "claim": "separate_shared_knowledge_from_general_reasoning",
    "reason": "Only part of the task is supported by shared knowledge."
  },
  "missing_needs": [
    {
      "need": "payment webhook signature verification",
      "reason": "No shared knowledge matched this need.",
      "source": "not_found",
      "severity": "important"
    }
  ],
  "items": [],
  "next_steps": []
}
```

### Field Mapping

| Current field | New field | Reason |
| --- | --- | --- |
| `coverage` | `support` | Describes whether shared knowledge supports the task. |
| `coverage.status` | `support.level` | `level` is easier than `status` for strong/partial/weak/none. |
| `coverage_gaps` | `missing_needs` | Plainly says which task needs were not supported. |
| `action_policy` | `usage` | Describes how the agent may use the result. |
| `authority` | remove or fold into `support.level` | Avoids official-truth language. |
| `recall` | `search_status` | Explains search capability without retrieval jargon. |
| `disclosure` | `read_status` | Explains whether full reading is available. |
| `next_calls` | `next_steps` | More natural for agents and humans. |
| `recommended_action` | `recommended_step` | Aligns with `next_steps`. |

### Support Levels

Keep the current level values:

- `strong`: shared knowledge directly supports the task.
- `partial`: some shared knowledge applies, but some needs are missing.
- `weak`: only broad or low-confidence matches exist.
- `none`: no useful shared knowledge matched.

These values are simple, stable, and already understandable.

### Missing Need Sources

Use simpler source values:

| Current source | New source | Reason |
| --- | --- | --- |
| `unmatched_intent` | `not_found` | Direct and easy to understand. |
| `weak_match` | `weak_match` | Clear enough to keep. |
| `partial_match` | `partial_match` | Clear enough to keep. |
| `filter_excluded` | `filtered_out` | More natural phrasing. |
| `conflicting_shared_knowledge` | `conflict` | Shorter; details can live in `reason`. |
| `low_confidence` | `low_confidence` | Clear enough to keep. |
| `cross_domain_mismatch` | `wrong_scope` | Easier than domain mismatch and covers project, domain, file, and scope. |

## Go Type Names

Use clear internal names that mirror the public API without becoming too casual:

| Current type | New type |
| --- | --- |
| `DiscoveryResponse` | `FindKnowledgeResponse` |
| `MapResponse` | `ListKnowledgeResponse` |
| `Coverage` | `Support` |
| `CoverageGap` | `MissingNeed` |
| `ActionPolicy` | `UsageGuidance` |
| `RecallState` | `SearchStatus` |
| `SemanticRecallState` | `SemanticSearchStatus` |
| `DiscoveryItem` | `KnowledgeSummary` |
| `Disclosure` | `ReadStatus` |
| `RecommendedCall` | `NextStep` |

The package name `query` can remain until there is a stronger reason to split
or rename it. The product language can improve without a large package
restructure.

## CLI Names

The CLI should follow the same verbs where possible:

```text
argos knowledge list
argos knowledge find
argos knowledge read <id>
argos knowledge cite <id>...
```

Existing commands can remain temporarily if changing CLI names would distract
from the MCP/adapter migration, but generated adapter instructions should teach
the new names first.

## Adapter Language

Adapters should describe the workflow in ordinary language:

```text
1. List available shared knowledge when you need orientation.
2. Find knowledge for the current task.
3. Read only the items you plan to use.
4. Cite only items you actually used.
5. If support is weak or none, continue with normal reasoning but do not claim
   Argos-backed guidance.
```

Avoid:

- "Argos authority"
- "coverage gap candidate"
- "recall boundary"
- "progressive disclosure gate"

Use:

- "shared knowledge support"
- "missing needs"
- "search status"
- "read full knowledge"

## Migration Strategy

Because Argos is still in development, the implementation can migrate public
names directly instead of carrying long-lived compatibility aliases.

Recommended implementation order:

1. Add golden tests for the new MCP tool names and JSON fields.
2. Rename internal Go response types and fields.
3. Rename MCP tool schemas, call handlers, and error messages.
4. Rename CLI JSON fields and CLI commands if included in this slice.
5. Update adapters and generated instruction text.
6. Update active specs, templates, and checklists.
7. Add negative scans for retired user-visible names in active surfaces.
8. Keep old terms only in historical plans/reports with supersession notes.

## Active Surface Scan

After migration, these old terms should not appear in active surfaces such as
`internal`, `testdata`, `docs/superpowers/templates`, active specs, and active
checklists:

```text
argos_map
argos_discover
get_knowledge_item
coverage_gaps
CoverageGap
action_policy
ActionPolicy
authority
recall
RecallState
disclosure
Disclosure
```

`cite_knowledge` should be rejected only when it appears as the old standalone
tool name. The new tool name `argos_cite_knowledge` is valid.

Historical implementation plans and reports may keep old terms when they are
part of the record, but any active guidance should point to the new vocabulary.

## Testing Requirements

The migration should keep the current behavior while changing names.

Tests should prove:

- MCP `tools/list` exposes the new tool names.
- MCP rejects old tool names after the migration.
- JSON responses use `support`, `missing_needs`, `usage`,
  `search_status`, and `next_steps`.
- Strong support recommends reading and later citation.
- Weak and none support do not recommend reading or citation.
- Partial support separates shared-knowledge-backed work from general
  reasoning.
- Missing needs are not treated as knowledge IDs.
- Adapter output teaches list/find/read/cite in that order.
- Negative scans reject retired user-visible names in active surfaces.

## Open Questions

### Should CLI Rename In The Same Slice?

Recommendation: yes if the current CLI is still development-only. Keeping MCP
and CLI aligned avoids two vocabularies.

If the CLI has external users, keep aliases for one release. Current project
state suggests a direct migration is acceptable.

### Should `argos_context` Change?

Recommendation: no for this naming slice. `argos_context` is a workflow helper,
not a shared-knowledge search action. It can recommend the new list/find/read
tools without being renamed now.

### Should `argos_standards` Change?

Recommendation: no for this naming slice. It is a narrow shortcut that predates
the shared-knowledge discovery path. Revisit after the list/find/read/cite
surface is stable.

## Success Criteria

The naming migration is successful when a new contributor can understand the
main workflow from tool names alone:

```text
list knowledge -> find knowledge -> read knowledge -> cite knowledge
```

The JSON should read like ordinary product language:

```text
support, missing_needs, usage, search_status, items, next_steps
```

The behavior should remain unchanged: Argos helps agents use shared uploaded
knowledge without overloading context, overclaiming support, or citing knowledge
that was not read and used.
