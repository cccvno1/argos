# Argos Discovery Layer Design

Date: 2026-04-30

## Purpose

Argos Discovery is the agent-facing knowledge navigation layer for Argos.
It is not a plain search box, a bare vector database interface, or a
replacement for host workflow systems.

Discovery should help an AI coding agent answer four questions:

1. What knowledge exists in this project knowledge base?
2. Which knowledge corresponds to the current task, phase, and files?
3. How should the agent retrieve the selected knowledge without overfull reading
   context?
4. When the knowledge base has no strong match, how should the agent avoid
   overclaiming, oversearching, or inventing Argos-backed guidance?

The design follows the adapter contract already established for generated
instruction files: host workflows remain in control, Argos provides governed
knowledge retrieval, full Markdown bodies are read progressively, and final
answers cite the Argos knowledge IDs that informed them.

## Design Principles

### Lightweight Core

Discovery must work on ordinary developer machines without model downloads,
GPU requirements, background services, or a required embedding provider.

The default discovery path uses local SQLite data:

- structured metadata
- project, type, status, priority, domain, and tag filters
- file-scope matching
- SQLite FTS5 full-text search
- package entrypoint indexing
- deterministic ranking
- explicit match explanations

Semantic search is an optional enhancement. If semantic search is unavailable,
misconfigured, or stale, `argos_find_knowledge` and `argos_list_knowledge` still return useful
results from the core SQLite path.

### Progressive Reading

Discovery returns routes, summaries, explanations, and next steps. It does not
return full Markdown bodies.

Full knowledge bodies are read only through explicit second-stage tools such
as `argos_read_knowledge`. Final citations are produced through `argos_cite_knowledge`.

The normal agent flow is:

```text
argos_context
  -> argos_list_knowledge
  -> argos_find_knowledge
  -> argos_read_knowledge
  -> argos_cite_knowledge
```

Agents should read only the IDs recommended by discovery or explicitly needed
for the user's task.

### Governed Retrieval

Argos owns the knowledge boundary. Agents should not query vector tables or
SQLite internals directly as their main workflow.

Argos is responsible for:

- excluding deprecated knowledge unless requested
- respecting project and file scope
- grouping package section matches back to package entrypoints
- reporting weak or missing support
- recommending only implemented tools
- returning citation-ready knowledge IDs

### Absence Is A First-Class Result

No match is a valid discovery result, not a failure.

Discovery must distinguish:

- `strong`: relevant knowledge exists and should be read before work
- `partial`: some useful knowledge exists, but there are clear gaps
- `weak`: only broad or low-confidence knowledge exists
- `none`: no useful Argos knowledge was found for this request

Weak or empty results must not be inflated into Argos-backed guidance. Agents
must not cite or claim Argos-backed guidance unless Argos returned knowledge IDs
that were actually used.

## Harness Model

Discovery is a harness around agent knowledge use:

```text
Inventory -> Routing -> Read Planning -> Controlled Reading -> Citation
```

### `argos_context`: Workflow Gate

`argos_context` remains the first workflow-aware entrypoint. It does not search
the knowledge base directly. It recommends the next Argos calls for the current
project, phase, task, and files.

For broad or unfamiliar project work, it should recommend `argos_list_knowledge` or
`argos_find_knowledge`. For implementation and review, it should continue to
recommend standards-oriented retrieval.

### `argos_list_knowledge`: Inventory Gate

`argos_list_knowledge` answers what the knowledge base contains.

It returns a lightweight inventory by project, domain, type, and package:

- counts by knowledge type
- known domains and tags
- active packages and their entrypoints
- grouped rule, decision, lesson, runbook, reference, and package summaries
- deprecated and inbox visibility only when requested

It does not return full Markdown bodies.

This prevents agents from blindly guessing what to search for.

### `argos_find_knowledge`: Routing Gate

`argos_find_knowledge` maps the current task to relevant knowledge.

It accepts project, phase, task, files, query text, and optional filters. It
returns ranked knowledge routes with:

- `id`
- `type`
- `title`
- `summary`
- `status`
- `priority`
- `path`
- `score`
- `score_components`
- `why_matched`
- `matched_sections`
- `read_status`
- `recommended_step`

It also returns `support` and `next_steps` so agents know whether to read,
skim, inspect the list, proceed without Argos-specific knowledge, or cite later.

### `argos_read_knowledge`: Reading Gate

`argos_read_knowledge` remains the explicit full-body reading tool.

Agents should call it only for selected IDs returned by `argos_find_knowledge`,
`argos_list_knowledge`, or a direct user need. It returns the full Markdown body and
metadata for one indexed knowledge item.

### `argos_cite_knowledge`: Accountability Gate

`argos_cite_knowledge` remains the final citation tool.

Agents should call it for knowledge IDs actually used in the final answer.
Discovery results alone do not justify citation if the agent did not apply that
knowledge.

## Tool Surface

Discovery adds two agent-facing tools:

```text
argos_find_knowledge
argos_list_knowledge
```

Existing tools remain:

```text
argos_context
argos_standards
argos_read_knowledge
argos_cite_knowledge
```

`argos_standards` stays as a rule-only shortcut. The broader discovery entrypoint
is `argos_find_knowledge`.

All `next_steps` returned by any discovery response must reference implemented
tools only.

## `argos_find_knowledge`

### Request

```json
{
  "project": "mall-api",
  "phase": "implementation",
  "task": "add refresh token endpoint",
  "query": "refresh token session renewal",
  "files": ["internal/auth/session.go"],
  "types": ["rule", "decision", "lesson", "runbook", "reference", "package"],
  "tags": ["auth"],
  "domains": ["backend", "security"],
  "status": ["active"],
  "include_deprecated": false,
  "limit": 8
}
```

`project` should be required for normal project work. At least one of `task` or
`query` should be present. `phase`, `files`, and filters refine ranking and
support.

Default behavior:

- exclude deprecated knowledge
- exclude inbox candidates
- search all supported knowledge types
- return at most 8 items
- cap limit at 20
- avoid full bodies

### Response

```json
{
  "project": "mall-api",
  "phase": "implementation",
  "query": "refresh token session renewal",
  "capabilities": {
    "metadata": "enabled",
    "fts": "enabled",
    "semantic": "disabled",
    "semantic_reason": "semantic provider is not configured"
  },
  "support": {
    "level": "strong",
    "confidence": 0.86,
    "reason": "Found active project knowledge matching auth files and refresh token terms.",
    "recommendation": "Read high-priority matched knowledge before implementation."
  },
  "items": [
    {
      "id": "rule:backend.auth.v1",
      "type": "rule",
      "title": "Auth middleware rule",
      "summary": "Require explicit auth middleware for account endpoints.",
      "status": "active",
      "priority": "must",
      "path": "knowledge/items/backend/auth.md",
      "score": 0.91,
      "score_components": {
        "project": 1.0,
        "file_scope": 1.0,
        "type_phase": 0.8,
        "priority": 1.0,
        "tag_domain": 0.6,
        "lexical": 0.7,
        "semantic": 0.0
      },
      "why_matched": [
        "project mall-api matched",
        "file internal/auth/session.go matched applies_to.files internal/auth/**",
        "implementation phase prefers rules and packages",
        "query matched auth and token terms"
      ],
      "matched_sections": [],
      "read_status": {
        "level": "summary",
        "full_body_available": true,
        "read_tool": "argos_read_knowledge"
      },
      "recommended_step": "read_full_before_implementation"
    }
  ],
  "next_steps": [
    {
      "tool": "argos_read_knowledge",
      "ids": ["rule:backend.auth.v1"],
      "reason": "Read high-priority matched knowledge before implementation."
    },
    {
      "tool": "argos_cite_knowledge",
      "reason": "Cite Argos knowledge IDs actually used in the final response."
    }
  ]
}
```

### Support Semantics

`support.level` guides agent behavior:

- `strong`: read recommended high-priority IDs before work
- `partial`: read only high-confidence IDs and mention gaps when relevant
- `weak`: skim summaries or inspect the list; do not treat results as Argos-backed guidance
- `none`: proceed without Argos-specific claims and do not cite Argos knowledge

Weak or empty discovery should include missing knowledge hints when useful:

```json
{
  "support": {
    "level": "partial",
    "confidence": 0.48,
    "reason": "Found general backend rules, but no refresh-token-specific knowledge.",
    "missing_knowledge_hints": [
      "refresh token implementation standard",
      "session renewal decision",
      "auth token rotation lesson"
    ],
    "recommendation": "Use matched general rules cautiously and do not claim refresh-token-specific Argos guidance."
  }
}
```

## `argos_list_knowledge`

### Request

```json
{
  "project": "mall-api",
  "domain": "backend",
  "types": ["rule", "decision", "lesson", "runbook", "reference", "package"],
  "include_deprecated": false
}
```

### Response

```json
{
  "project": "mall-api",
  "inventory": {
    "types": {
      "rule": 12,
      "decision": 5,
      "lesson": 8,
      "runbook": 3,
      "reference": 6,
      "package": 4
    },
    "domains": ["backend", "security", "database"],
    "tags": ["auth", "session", "migration"],
    "packages": [
      {
        "id": "package:backend.auth-refresh-token.v1",
        "title": "Refresh token implementation",
        "summary": "Implementation guidance for refresh token flows.",
        "entrypoint": "knowledge/packages/backend/auth-refresh-token/KNOWLEDGE.md",
        "read_status": {
          "level": "summary",
          "full_body_available": true,
          "read_tool": "argos_read_knowledge"
        }
      }
    ]
  },
  "groups": [
    {
      "key": "backend/auth",
      "title": "Backend Auth",
      "items": [
        {
          "id": "rule:backend.auth.v1",
          "type": "rule",
          "title": "Auth middleware rule",
          "summary": "Require explicit auth middleware for account endpoints.",
          "status": "active",
          "priority": "must",
          "path": "knowledge/items/backend/auth.md",
          "read_status": {
            "level": "summary",
            "full_body_available": true,
            "read_tool": "argos_read_knowledge"
          }
        }
      ]
    }
  ]
}
```

`argos_list_knowledge` is for inventory and orientation. It should not try to replace
`argos_find_knowledge` for task-specific ranking.

## CLI Surface

CLI JSON fallbacks mirror the MCP tools:

```bash
argos knowledge find --json --project mall-api --phase implementation --task "add refresh token endpoint" --files internal/auth/session.go
argos knowledge list --json --project mall-api --domain backend
```

Non-JSON output may be added later for humans, but JSON is the first-class
agent contract.

## Index Model

Discovery extends `argos/index.db`; it does not require a separate database or
service.

### `knowledge_items`

The existing item index remains the source of item-level metadata and full
bodies. It should include tags so discovery can filter and rank by tags:

```text
id
path
title
type
tech_domains
business_domains
projects
status
priority
scope
tags
updated_at
summary
body
```

### `knowledge_fts`

SQLite FTS5 powers the default full-text discovery path:

```text
item_id
title
summary
body
tags
domains
```

FTS must support title, summary, body, tag, and domain matches. These matches
should contribute to `lexical` score components and `why_matched` reasons.

### `knowledge_chunks`

Chunks provide finer search without changing progressive reading:

```text
chunk_id
item_id
path
section
heading_path
ordinal
text
token_estimate
```

Chunks are internal search units. Discovery results group chunk hits back to
their knowledge item IDs.

### `knowledge_vectors`

Vectors are optional:

```text
chunk_id
provider
model
dimensions
embedding
```

This table is populated only when semantic discovery is enabled and a provider
is configured.

### `index_metadata`

Index metadata tracks schema and semantic state:

```text
schema_version
built_at
semantic_enabled
embedding_provider
embedding_model
embedding_dimensions
content_hash
```

Discovery responses use this table to explain whether semantic search is
enabled, disabled, stale, or degraded.

## Optional Semantic Layer

Semantic search is opt-in. It enhances search but is never required for
successful discovery.

The provider contract is:

```text
Embed(texts []string) -> vectors
```

The first implementation should prefer a generic command provider:

```yaml
discovery:
  semantic:
    enabled: true
    provider: command
    command: ["argos-embed", "--json"]
```

Future providers may include local model runtimes such as Ollama, but Argos
must not require Ollama, auto-download models, start a background service, or
require GPU hardware for Discovery v1.

Semantic failure rules:

- provider not configured: semantic disabled, core discovery continues
- provider timeout: semantic disabled for the request, core discovery continues
- vector dimension mismatch: semantic disabled with rebuild guidance
- stale semantic metadata: semantic disabled or degraded with rebuild guidance
- vector table missing: semantic disabled, core discovery continues

## Search And Ranking

Discovery uses a deterministic hybrid pipeline:

```text
1. Normalize request
2. Apply hard filters
3. Run search signals
4. Fuse scores
5. Group chunk and package matches
6. Assign support and recommended steps
7. Return explanations and next steps
```

### Hard Filters

Hard filters remove results that should not be considered:

- deprecated knowledge unless `include_deprecated` is true
- non-matching explicit `types`
- non-matching explicit `tags`
- non-matching explicit `domains`
- non-matching explicit `status`
- knowledge outside the requested project unless it is global or cross-project

Inbox discovery is out of scope for Discovery v1 because the local discovery
index contains official knowledge only; inbox candidates remain path-based
validation and promotion inputs rather than queryable status-based results.

### Search Signals

Default signals:

- project match
- file scope match
- type and phase fit
- priority
- status
- domain match
- tag match
- FTS lexical match
- package section match

Optional signal:

- semantic vector match

### Score Components

The first ranking implementation should use simple, explainable weighted
components:

```text
project_match      * 0.18
file_scope_match   * 0.18
type_phase_fit     * 0.14
priority_weight    * 0.12
status_weight      * 0.08
tag_domain_match   * 0.12
lexical_score      * 0.12
semantic_score     * 0.06
```

When semantic search is disabled, the remaining weights should be normalized or
reported clearly so semantic absence does not unfairly penalize all results.

Tie-break order:

```text
score desc
priority rank
type phase rank
id asc
```

The same request against the same index should produce stable ordering.

## Phase-Aware Routing

`phase` informs type preference and recommended steps:

- `planning`: decision, guide, package, reference
- `implementation`: rule, package, runbook, decision
- `review`: rule, decision, lesson
- `debugging`: lesson, runbook, decision, risk-like tags
- `operations` or `deployment`: runbook, decision, rule

This should affect ranking and recommendations, not hard-exclude other types
unless the user explicitly filters by type.

## Package-Aware Discovery

Packages are first-class knowledge items.

Only `KNOWLEDGE.md` is indexed as the package entrypoint. Optional package
directories such as `references/`, `examples/`, `checklists/`, `scripts/`, and
`assets/` are not exposed as independent knowledge items in Discovery v1.

The chunker may index package entrypoint sections:

- `Purpose`
- `When To Use`
- `Start Here`
- `Load On Demand`

If one or more package chunks match, discovery returns one package item with
`matched_sections`; it does not return chunk full text.

Example:

```json
{
  "id": "package:backend.redis.best-practices.v1",
  "type": "package",
  "matched_sections": ["When To Use", "Start Here"],
  "read_status": {
    "level": "summary",
    "full_body_available": true,
    "read_tool": "argos_read_knowledge"
  }
}
```

## Recommended Steps

Discovery results should use a small step vocabulary:

- `read_full_before_planning`
- `read_full_before_implementation`
- `read_full_before_review`
- `read_full_before_debugging`
- `skim_summary_only`
- `inspect_list`
- `cite_after_read_and_used`
- `ignore_unless_task_expands`
- `proceed_without_argos_specific_knowledge`

Rules:

- high-confidence `must` rules in implementation or review should recommend
  full reading
- high-confidence packages should recommend full reading before implementation
- lessons in debugging should recommend full reading when confidence is high
- low-confidence general references should recommend summary-only handling
- weak matches should not recommend full reading
- no match should recommend proceeding without Argos-specific claims

## Fallback And Error Handling

Discovery should report capabilities and degraded states in the response.

Index missing:

- MCP tools return a tool error with guidance to run `argos index`
- CLI exits non-zero with the same guidance

FTS unavailable:

- fall back to metadata, title, and summary scanning where possible
- mark `capabilities.fts` as degraded

Semantic unavailable:

- do not fail discovery
- set `capabilities.semantic` to `disabled` or `degraded`
- include a clear reason

Unknown filters or malformed arguments:

- return argument errors
- do not silently ignore unknown fields in MCP arguments

No match:

- return an empty item list and `support.level = "none"`
- include a recommendation not to cite Argos knowledge for the task

## Adapter Contract Impact

Generated adapters should be updated after discovery exists.

They should recommend:

- `argos_context` before substantial work
- `argos_find_knowledge` for task-specific knowledge routing
- `argos_list_knowledge` for broad orientation or unfamiliar project areas
- `argos_read_knowledge` only for selected IDs
- `argos_cite_knowledge` for knowledge used in final answers

Adapters must continue to state that Argos does not replace host workflow,
safety, user instructions, tests, builds, linting, or review.

## Testing Strategy

### Index Tests

- tags are stored in the index
- FTS matches title, summary, body, tags, and domains
- packages are indexed through `KNOWLEDGE.md`
- package entrypoints are chunked by section
- package assets and non-entrypoint files are not indexed as standalone items

### Query Tests

- project matches rank above unrelated knowledge
- file scope matches rank above general project knowledge
- phase preferences influence type ranking
- deprecated knowledge is excluded by default
- inbox knowledge is excluded by default
- explicit filters are honored
- limit and stable ordering work

### Support Tests

- strong match returns read recommendations
- partial match reports gaps
- weak match does not recommend full reading
- no match returns no items and no citation recommendation

### Progressive Reading Tests

- `argos_find_knowledge` does not return full body text
- `argos_list_knowledge` does not return full body text
- each result contains read_status metadata
- high-priority selected IDs appear in `next_steps`

### Package Tests

- package section matches return package entrypoint results
- `matched_sections` is populated
- chunk text is not exposed in discovery responses

### MCP And CLI Tests

- `tools/list` includes `argos_find_knowledge` and `argos_list_knowledge` schemas
- MCP argument decoding is strict
- MCP discovery requires index availability
- CLI JSON shape matches query service responses
- index-missing errors are clear

### Semantic Fallback Tests

- semantic disabled still returns core results
- provider errors degrade semantic only
- vector dimension mismatch disables semantic with rebuild guidance

## Out Of Scope

Discovery v1 does not include:

- required embeddings
- cloud embedding providers
- automatic model downloads
- direct agent access to vector tables
- package asset vector indexing
- graph database storage
- web UI
- background daemon
- automatic script execution
- remote package installation

## Open Implementation Notes

The design is intentionally complete at the harness level while allowing the
implementation plan to stage work safely.

A practical implementation sequence is:

1. Add index support for tags and FTS.
2. Add chunk indexing for items and package entrypoints.
3. Implement `query.FindKnowledge` and `query.ListKnowledge` with SQLite core ranking.
4. Expose `argos knowledge find --json` and `argos knowledge list --json`.
5. Expose MCP `argos_find_knowledge` and `argos_list_knowledge`.
6. Update adapters and README.
7. Add the optional semantic provider contract and degraded-state reporting.

The implementation plan may decide whether the command semantic provider lands
in the first code phase or a follow-up phase, but Discovery v1 must keep the
response schema and fallback contract ready for semantic enhancement.
