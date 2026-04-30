# Argos Shared Knowledge Discovery Semantics Design

Date: 2026-04-30

## Purpose

Argos is intended to be a shared knowledge platform: people upload reusable
knowledge, and agents use Discovery to understand which uploaded knowledge can
support the current task. Discovery should not imply that the system owns an
official project truth or that uncovered needs should automatically become new
knowledge.

The previous gap-candidate model was directionally useful as a safety guard,
but its naming and fields over-weighted a capture workflow. This design
replaces that model with shared-knowledge coverage semantics.

## Goals

- Fully migrate Discovery away from the previous gap-candidate response model
  and capture-oriented fields.
- Introduce `coverage_gaps` as uncovered task needs, not upload proposals.
- Make the agent separate Argos-backed knowledge from general reasoning.
- Keep progressive disclosure, citation accountability, and `action_policy`.
- Keep semantic recall optional and lightweight.
- Treat this as a development-stage interface break; no compatibility field is
  required.
- Broaden test scenarios so no single example dominates the design.

## Non-Goals

- Do not build a capture/upload workflow in this change.
- Do not automatically recommend that users upload new knowledge.
- Do not add a vector database or embedding provider.
- Do not solve global naming across the whole product in this change.
- Do not change knowledge storage format unless required by the Discovery
  response rename.

## Design Principles

### Shared Knowledge, Not Official Truth

Argos should describe whether shared uploaded knowledge supports a task. It
should not imply that uploaded knowledge is universally authoritative or that
all missing support is a defect in the knowledge base.

### Coverage Boundaries Over Capture Prompts

An uncovered need means the current shared knowledge pool does not provide
enough support for that part of the task. It is a boundary marker for the
agent, not a command to create knowledge.

### Attribution Is The Core Agent Duty

The agent may continue working when coverage is partial, weak, or none. The
important rule is attribution: do not present general reasoning as Argos-backed
knowledge, and do not cite shared knowledge that was not loaded and applied.

### Interface Can Break During Development

Because Argos is still in active development, this migration should remove the
misleading interface rather than preserve it. Tests, adapters, docs, and
dogfood should move to the new names in one slice.

## Response Model

Use:

```json
{
  "coverage_gaps": [
    {
      "need": "payment webhook signature verification",
      "reason": "No sufficiently relevant shared knowledge matched this task need.",
      "source": "unmatched_intent",
      "severity": "important",
      "argos_backed": false
    }
  ]
}
```

### `CoverageGap`

```go
type CoverageGap struct {
    Need        string `json:"need"`
    Reason      string `json:"reason"`
    Source      string `json:"source"`
    Severity    string `json:"severity"`
    ArgosBacked bool   `json:"argos_backed"`
}
```

Field meanings:

- `need`: the task need that is not supported by shared knowledge.
- `reason`: why Discovery considers it uncovered.
- `source`: why the gap was created.
- `severity`: how important the uncovered need is for agent attribution.
- `argos_backed`: always `false` for coverage gaps.

Valid `source` values:

- `unmatched_intent`: no sufficiently relevant shared knowledge matched.
- `weak_match`: some knowledge matched weakly but cannot support a claim.
- `partial_match`: some knowledge applies, but important needs remain uncovered.
- `filter_excluded`: explicit filters excluded otherwise relevant knowledge.
- `conflicting_shared_knowledge`: multiple uploaded items disagree enough that
  the agent must not present a single Argos-backed conclusion.
- `low_confidence`: matched knowledge is stale, draft-like, deprecated, or too
  low quality to support confident attribution.
- `cross_domain_mismatch`: lexical similarity exists, but domain/scope does not
  match the task.

Valid `severity` values:

- `blocking`: the uncovered need is central to the task and must be stated
  before applying any Argos-backed knowledge.
- `important`: the uncovered need affects design or implementation decisions.
- `informational`: useful boundary information, but not central to the task.

## Coverage And Action Policy

Coverage statuses remain:

- `strong`: shared knowledge directly supports the task.
- `partial`: some shared knowledge applies, but meaningful needs are uncovered.
- `weak`: only broad or low-confidence matches exist.
- `none`: no relevant shared knowledge supports the task.

Action policy should shift wording from authority/capture toward attribution:

```json
{
  "action_policy": {
    "authority": "partial",
    "load": "allowed",
    "cite": "after_loaded_and_used",
    "claim": "must_separate_argos_backed_and_general_reasoning",
    "reason": "Some shared knowledge applies, but uncovered needs must be separated from general reasoning."
  }
}
```

First implementation may keep the `authority` field name because it is already
part of the response, but adapters and docs should describe it as the current
support level. A later naming audit may rename it to `support_level` or
`evidence_level`.

Claim behavior:

- `allowed`: the agent may make Argos-backed claims after loading and applying
  relevant items.
- `must_separate_argos_backed_and_general_reasoning`: the agent may use loaded
  Argos knowledge, but must explicitly separate uncovered parts.
- `forbidden`: the agent must not make Argos-backed claims.

## Agent Behavior

### Strong Coverage

The agent should load recommended items before applying them, use the loaded
knowledge, and cite only loaded IDs that actually influenced the answer.
`coverage_gaps` should be empty.

### Partial Coverage

The agent may load relevant IDs. It must state which parts are backed by shared
knowledge and which parts are general reasoning or uncovered by Argos. It must
not imply that `coverage_gaps` are knowledge items.

### Weak Coverage

The agent may inspect summaries but must not load, cite, or make Argos-backed
claims unless a later query produces stronger coverage. `coverage_gaps` explain
why the weak match is insufficient.

### None Coverage

The agent may continue using general reasoning, normal coding ability, and user
instructions. It must not cite Argos or claim that Argos has supporting shared
knowledge for the task. `coverage_gaps` explain the unsupported needs.

### Upload Or Capture

Discovery should not start upload, capture, inbox creation, or proposal writing.
If the user explicitly asks to share knowledge, a separate upload/capture flow
can use the task context and coverage gaps as input. Discovery itself only
reports coverage boundaries.

## Scenario Matrix

The test and dogfood suite should cover multiple task families. No single
scenario should carry the design.

### Complete Coverage

The shared knowledge pool contains directly relevant implementation, business,
or operational knowledge. Discovery returns `strong`, no `coverage_gaps`, and
load/cite recommendations for concrete IDs.

### Partial Engineering Coverage

The pool contains reusable engineering knowledge, such as language, database,
testing, or deployment practices, but lacks the task-specific design knowledge.
Discovery returns `partial`, routes the reusable knowledge, and marks uncovered
business or architecture needs.

### Partial Business Coverage

The pool contains domain or workflow knowledge, but lacks implementation or
operational details. Discovery returns `partial`, routes domain knowledge, and
marks technical needs as uncovered.

### Weak Generic Term Match

The pool contains a broad term that overlaps with the task but does not support
the requested work. Discovery returns `weak`, forbids load/cite, and reports a
`weak_match` coverage gap.

### No Coverage

No relevant shared knowledge matches. Discovery returns `none`, no items or
next calls, and `coverage_gaps` from `unmatched_intent`.

### Filter-Excluded Coverage

Relevant knowledge exists, but explicit filters exclude it. Discovery returns
`none` or `partial` depending on remaining matches and reports
`filter_excluded` gaps rather than bypassing the user's filter.

### Conflicting Shared Knowledge

Multiple uploaded items match but provide incompatible recommendations.
Discovery should not collapse them into one authoritative answer. It should
route the conflicting items as appropriate and report
`conflicting_shared_knowledge` gaps until conflict handling is designed.

### Low-Confidence Or Outdated Coverage

Matched knowledge is deprecated, stale, draft-quality, or otherwise low trust.
Discovery should either exclude it or lower confidence and report
`low_confidence` gaps.

### Cross-Domain Mismatch

Text matches but project, domain, tags, files, or scope show the knowledge does
not apply. Discovery should avoid over-ranking it and report
`cross_domain_mismatch` when the mismatch is meaningful for the task.

## Data Flow

1. Adapter or agent calls `argos_context` for broad routing.
2. Agent calls `argos_discover` with task/query/files/filters.
3. Discovery gathers candidates from metadata, file scope, and FTS.
4. Optional semantic recall may add score evidence if configured.
5. Discovery ranks shared knowledge items.
6. Discovery computes coverage status and action policy.
7. Discovery derives `coverage_gaps` from uncovered task needs, weak evidence,
   explicit filters, conflicts, or low-confidence matches.
8. Agent follows `action_policy`.
9. Agent loads only routed IDs when allowed.
10. Agent cites only loaded-and-applied IDs.
11. Agent states uncovered needs as non-Argos-backed when relevant.

## Migration Scope

This migration should update:

- `internal/query/query.go`: replace the previous gap-candidate type with
  `CoverageGap`, remove capture-oriented fields, populate `coverage_gaps`.
- `internal/query/query_test.go`: assert strong has no gaps; partial/weak/none
  have coverage gaps with `argos_backed=false`.
- `internal/discoverytest/golden.go` and `testdata/discovery-golden/cases.json`:
  rename expected gap kinds to expected coverage gap sources or needs.
- `internal/query/discovery_golden_test.go`: assert `coverage_gaps`.
- `internal/cli/discovery_golden_test.go`: assert CLI JSON exposes
  `coverage_gaps`.
- `internal/mcp/server_test.go`: assert MCP JSON exposes `coverage_gaps`.
- `internal/adapters/adapters.go`: replace capture/gap candidate wording with
  shared knowledge attribution wording.
- `internal/adapters/adapters_test.go`: lock the new wording.
- Dogfood template and reports: replace gap candidate guards with coverage
  boundary and attribution guards.
- Previous design docs can remain as historical records, but the new spec is
  the active direction.

## Testing Strategy

### Unit Tests

- `Discover` returns no `coverage_gaps` for strong coverage.
- `Discover` returns `coverage_gaps` for partial, weak, and none coverage.
- Every coverage gap has `argos_backed=false`, non-empty `need`, non-empty
  `reason`, valid `source`, and valid `severity`.
- Weak/none still forbid load and citation.
- Partial uses
  `must_separate_argos_backed_and_general_reasoning`.

### Golden Tests

Golden cases should cover at least:

- complete coverage;
- partial engineering coverage;
- partial business coverage;
- weak generic term match;
- no coverage;
- filter-excluded coverage;
- cross-domain mismatch.

Conflict and low-confidence cases can be design-locked now and implemented as
focused tests when the underlying ranking metadata exists.

### Interface Tests

CLI and MCP should expose `coverage_gaps` and should not expose the retired
gap-candidate JSON field.

### Adapter Tests

Adapters should instruct agents to:

- treat Argos as shared uploaded knowledge;
- distinguish Argos-backed claims from general reasoning;
- never cite `coverage_gaps`;
- not start upload/capture from Discovery alone;
- follow `action_policy` as the behavior contract.

### Dogfood

Run a fresh-context dogfood round with varied scenarios. The runner should not
receive expected outputs or prior reports. It should verify that agents:

- use routed shared knowledge when allowed;
- state uncovered needs without treating them as knowledge;
- continue normal work when coverage is none;
- avoid capture/upload unless the user explicitly requests sharing knowledge.

## Naming Audit Follow-Up

After this migration, run a separate naming audit for product language. The
likely review set includes:

- `official knowledge` -> possibly `shared knowledge`;
- `project knowledge` -> possibly `workspace knowledge` or `shared knowledge`;
- `authority` -> possibly `support_level` or `evidence_level`;
- `claim` -> possibly `attribution`;
- `missing_knowledge_hints` -> remove or rename;
- `rule`, `standard`, `decision`, `lesson` -> verify fit for personal upload
  and sharing;
- adapter text that implies project-owned or official truth.

This audit should be a separate spec because it may affect storage taxonomy,
adapter wording, docs, and user-facing concepts beyond Discovery.

## Success Criteria

- No production response exposes the retired gap-candidate JSON field.
- No adapter instructs agents to treat Discovery gaps as capture prompts.
- Partial/weak/none cases expose coverage boundaries rather than upload
  proposals.
- Agents can explain what is Argos-backed and what is general reasoning.
- Tests and dogfood cover multiple task families instead of one dominant
  example.
