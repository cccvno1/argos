# Argos Discovery Recall And Gap Workflow Design

Date: 2026-04-30

## Purpose

Argos Discovery now has a stable safety boundary:

- progressive disclosure
- citation accountability
- `coverage`
- `action_policy`
- dogfood-verified weak/none behavior

The next Discovery step is not to make a vector database mandatory. It is to
separate two cases that look similar to an agent:

1. Relevant knowledge exists, but lexical/metadata discovery did not recall it.
2. Relevant knowledge does not exist yet, so the right next step is a safe gap
   or capture workflow.

This design adds a recall-and-gap layer that can support optional semantic
retrieval while preserving `action_policy` as the final authority boundary.

## Goals

- Add an optional semantic recall boundary without requiring Ollama, local
  models, GPUs, background services, or network access.
- Keep metadata, file-scope, and SQLite FTS as the default reliable path.
- Expose whether semantic recall is disabled, unavailable, stale, or used.
- Upgrade missing knowledge hints from strings into structured gap candidates.
- Let agents propose safe capture next steps without treating candidate
  knowledge as official Argos knowledge.
- Preserve all existing weak/none no-overclaim behavior.

## Non-Goals

- Do not add mandatory embeddings.
- Do not require a specific provider such as Ollama.
- Do not let semantic-only matches bypass `action_policy`.
- Do not automatically write inbox candidates from discovery.
- Do not automatically promote captured candidates.
- Do not replace the `capture-knowledge` skill proposal and approval workflow.

## Design Principles

### Safety Before Recall

Semantic retrieval can add candidate evidence, but `action_policy` remains the
agent contract. If the final policy says load, cite, or claim is forbidden, the
agent must obey that policy regardless of semantic similarity.

### Default-Light, Provider-Optional

Ordinary machines should keep working with no extra setup. The default semantic
state is disabled, and Discovery remains useful through metadata, file-scope,
and FTS.

### Gap Is Not Knowledge

A missing knowledge hint is not a knowledge item. A gap candidate can suggest
what durable knowledge might be useful, but it cannot be cited, loaded as
official knowledge, or used as project authority.

### Capture Is User-Gated

Discovery may recommend the `capture-knowledge` workflow when a durable gap is
clear. The capture workflow remains proposal-first and user-approved. Inbox
candidates remain candidates until reviewed and promoted.

## Response Additions

Extend discovery responses with two new fields:

```json
{
  "recall": {
    "semantic": {
      "status": "disabled|unavailable|stale|used",
      "provider": "",
      "model": "",
      "reason": "semantic provider is not configured"
    }
  },
  "gap_candidates": [
    {
      "kind": "standard|decision|lesson|runbook|reference|package",
      "suggested_title": "Payment webhook signature verification standard",
      "reason": "No active Argos knowledge matched payment webhook signature verification.",
      "source": "missing_knowledge_hint",
      "next_action": "capture_candidate",
      "capture_mode": "proposal_required",
      "authority": "candidate_only"
    }
  ]
}
```

`recall.semantic.status` meanings:

- `disabled`: no semantic provider is configured.
- `unavailable`: a provider is configured but cannot be used.
- `stale`: semantic index exists but is out of date or incompatible.
- `used`: semantic evidence contributed to ranking.

`gap_candidates` replace or wrap the current string-only
`missing_knowledge_hints`. Existing clients may continue reading the old field
during the transition.

## Semantic Provider Boundary

Discovery should define a provider interface, not a provider requirement.

Conceptual provider contract:

```go
type SemanticProvider interface {
	Name() string
	Model() string
	Dimensions() int
	EmbedTexts(ctx context.Context, texts []string) ([]Embedding, error)
	EmbedQuery(ctx context.Context, query string) (Embedding, error)
}
```

Provider configuration is explicit. If no provider is configured, Discovery
sets semantic status to `disabled` and continues with metadata and FTS.

Provider failures are non-fatal. If a provider times out, returns incompatible
dimensions, or cannot be reached, Discovery sets semantic status to
`unavailable` and continues with deterministic recall.

Provider staleness is non-fatal. If stored vectors do not match current content,
provider, model, or dimensions, Discovery sets semantic status to `stale` and
continues without semantic score.

The first implementation should not include a concrete embedding provider. It
should add the boundary and capability reporting so future provider work has a
stable target.

## Hybrid Ranking

Semantic evidence becomes one score component:

```json
"score_components": {
  "project": 0.2,
  "file_scope": 0,
  "type_phase": 0.1,
  "priority": 0.1,
  "status": 0.05,
  "tag_domain": 0.1,
  "lexical": 0.3,
  "semantic": 0.25
}
```

Rules:

- Metadata, project scope, status, deprecation, explicit filters, and file scope
  still apply before semantic ranking.
- Semantic candidates must point back to official indexed knowledge IDs.
- Semantic matches can improve ranking, but they do not automatically make
  coverage strong.
- Coverage should require enough combined evidence to justify its status.
- `action_policy` is derived after hybrid scoring and remains the final behavior
  contract.

This prevents a high semantic score from turning a weak or out-of-scope result
into authoritative guidance by itself.

## Gap Candidate Boundary

Structured gap candidates should be generated when coverage is `partial`,
`weak`, or `none`.

Gap candidates should include:

- `kind`: expected knowledge type.
- `suggested_title`: human-readable candidate title.
- `reason`: why this gap exists.
- `source`: `missing_knowledge_hint`, `coverage_gap`, or `filter_excluded`.
- `next_action`: normally `capture_candidate`.
- `capture_mode`: always `proposal_required` for now.
- `authority`: always `candidate_only`.

Gap candidates must not appear as discovery items. They are not routable
knowledge IDs and must not be passed to `get_knowledge_item` or
`cite_knowledge`.

## Gap Workflow

When a response includes gap candidates, the agent should follow this sequence:

```text
read action_policy
-> if claim/load/cite are forbidden, do not apply Argos-backed claims
-> inspect gap_candidates
-> explain the missing durable knowledge in plain language
-> ask whether to start capture-knowledge
-> if approved, run capture-knowledge proposal workflow
-> write only inbox/PR-style candidates after approval
-> validate candidates
-> never promote automatically
```

Discovery may recommend capture as a next action only when the user intent is
durable enough. For normal coding tasks, it should surface gap candidates but
not force capture.

Recommended wording for agents:

```text
Argos does not currently have official knowledge for this part of the task.
I can proceed without Argos-backed claims, or I can start the capture-knowledge
workflow to propose a reusable candidate for review.
```

## Interaction With `capture-knowledge`

Discovery should not duplicate the capture skill. It should hand off to it.

The handoff can include:

- task/query text
- project
- phase
- files
- gap candidates
- related weak/partial discovery items
- action policy
- recall state

The capture skill remains responsible for:

- checking existing official and inbox knowledge
- presenting a proposal
- asking create/update/stop when overlap exists
- asking for inbox candidate or PR-style delivery path
- validating written knowledge
- never promoting automatically

## CLI And MCP Surface

No new command is required for the first implementation.

Existing `argos discover --json` and MCP `argos_discover` should include
`recall` and `gap_candidates`.

Future commands may be added later, but they are not required for this design:

- `argos semantic index`
- `argos semantic status`
- `argos gap propose`

Those commands should wait until the provider boundary and gap response model
prove useful through tests and dogfood.

## Adapter Guidance

Generated adapters should add short guidance:

- Treat `gap_candidates` as candidate prompts, not official knowledge.
- Start `capture-knowledge` only with user approval.
- Do not cite gap candidates.
- Semantic recall never overrides `action_policy`.

This keeps agent behavior stable across MCP, CLI, and generated file fallback.

## Testing

Add tests in stages:

1. `recall.semantic.status` is `disabled` by default and existing discovery
   behavior is unchanged.
2. Provider unavailable/stale states degrade to metadata/FTS without failing
   discovery.
3. Semantic score is represented in `score_components.semantic` only when used.
4. Explicit filters and project scope still exclude semantic candidates.
5. `action_policy` remains consistent after semantic evidence is added.
6. Partial, weak, and none responses include structured gap candidates.
7. Gap candidates are absent or empty for strong coverage.
8. CLI and MCP JSON include recall and gap candidates.
9. Adapters warn that gap candidates are not official knowledge.
10. Dogfood verifies that agents do not cite gap candidates and only start
    capture after asking.

## Dogfood

Run a new dogfood round after implementation:

- `semantic_disabled_strong_still_works`
- `semantic_disabled_none_has_gap_candidates`
- `weak_gap_does_not_load_or_cite`
- `partial_gap_mentions_gap_before_capture`
- `filter_excluded_match_produces_filter_gap`
- `agent_requests_capture_permission`

Runner isolation rules remain the same as previous dogfood rounds.

Evaluator rules:

- Gap candidates are not loaded or cited.
- Weak/none still obey `action_policy`.
- Agents do not invent official knowledge from gaps.
- Agents ask before invoking capture workflow.
- Semantic disabled/unavailable state is treated as a capability fact, not a
  failure.

## Rollout

Recommended implementation order:

1. Add structured `RecallState` and `GapCandidate` response types with semantic
   disabled default.
2. Convert string missing hints into structured gap candidates while preserving
   the old field for compatibility.
3. Add adapter and dogfood guidance.
4. Dogfood the gap workflow with semantic disabled.
5. Add the provider interface and stale/unavailable capability states.
6. Only after that, evaluate concrete semantic providers.

This keeps the product useful before any embedding provider exists.

## Open Design Decisions

No decision is needed about a concrete semantic provider in this design.

The only product choice for implementation planning is whether the first slice
should include the provider interface immediately or stop after structured gap
candidates. The recommended first slice includes structured recall state and gap
candidates, but not a concrete provider.
