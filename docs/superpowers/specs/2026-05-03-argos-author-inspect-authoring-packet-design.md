# Argos Author Inspect Authoring Packet Design

Date: 2026-05-03

## Purpose

Improve the write-side authoring experience by making `argos author inspect
--json` return an agent-ready authoring packet.

The current `inspect` output already contains objective facts, policy, overlap
data, and a `proposal_scaffold`. Fresh runners can use it, but they still have
to infer too much of the workflow from scattered fields. This creates avoidable
variation: some agents write invalid proposal shapes, miss human-review
booleans, choose weak candidate paths, or fail to stop on review-only cases.

The target experience should match the query side: Argos returns not only raw
facts, but also clear usage guidance and next steps. The packet does not author
the knowledge for the agent. It tells the agent how to proceed safely.

## Decision

Add an `authoring_packet` field to the existing `InspectResponse`.

Do not add a new command in this slice. `author inspect` is already the durable
entrypoint for pre-write inspection, and keeping one command reduces agent
branching. A later slice can split `argos author plan` out if the packet grows
beyond inspect's responsibility.

## Goals

- Make `author inspect` directly usable by an AI agent as the next-step control
  packet for writing knowledge.
- Keep Argos workflow-neutral. The packet must not require brainstorming, plan
  mode, subagents, or any named host workflow.
- Reduce human command knowledge. Humans should review knowledge design, not
  remember file paths and policy flags.
- Make stop conditions explicit: overlap, missing content, unapproved candidate
  writing, unapproved official mutation, unapproved promotion, and unapproved
  `priority: must`.
- Preserve proposal-first behavior. Inspect never writes files and never marks
  candidate writing as approved.
- Keep the existing `proposal_scaffold` as the canonical JSON shape.
- Keep `recommended_next_steps` for compatibility, but make `authoring_packet`
  the richer source of action guidance.

## Non-Goals

- Do not generate final authoritative knowledge from vague intent.
- Do not write proposal or candidate files from `inspect`.
- Do not automate human approval decisions.
- Do not mutate official knowledge, promote candidates, or refresh the index.
- Do not encode or enumerate host-agent thinking workflows.
- Do not replace `author verify`.
- Do not add semantic search or ranking changes.

## Current Shape

`InspectResponse` currently returns:

- identity: `project`, `goal`, `mode`
- capability facts: filesystem and index status
- registry facts
- overlap matches from official, inbox, and optional index
- path risk
- policy
- proposal requirements
- `proposal_scaffold`
- simple `recommended_next_steps`

This is sufficient for deterministic inspection, but not yet sufficient for a
stable write-side user experience. The missing layer is the equivalent of query
side `support`, `usage`, `missing_needs`, and `next_steps`: a concise statement
of what the agent may do next and what blocks progress.

## Proposed Response Addition

Add:

```json
{
  "authoring_packet": {
    "state": "ready_for_proposal",
    "recommended_action": "write_proposal",
    "action_reason": "No blocking overlap was detected; write a proposal for human review before any candidate files.",
    "proposal_path": "knowledge/.inbox/proposals/mall-api/product-list-cache/proposal.json",
    "candidate_path": "knowledge/.inbox/packages/mall-api/product-list-cache",
    "candidate_allowed": false,
    "review_only": false,
    "stop_conditions": [
      "Do not write candidate files until human_review.candidate_write_approved is true.",
      "Do not mutate official knowledge unless official mutation is explicitly authorized.",
      "Do not promote candidates unless promotion is explicitly authorized.",
      "Do not use priority must unless priority_must_authorized is true."
    ],
    "proposal_focus": [
      "Clarify the future agent task this knowledge supports.",
      "Separate user-stated intent, observed facts, imported facts, synthesized guidance, assumptions, and open questions.",
      "Record applicability boundaries and negative triggers.",
      "Record overlap decision before candidate writing."
    ],
    "source_work": {
      "user_confirmed": ["Preserve the user's request as user-stated intent."],
      "observed": ["Add observed repo facts only after inspecting files or commands."],
      "synthesized": ["Mark agent recommendations as synthesized and review-needed."],
      "assumptions": ["Keep missing details visible as assumptions or open questions."]
    },
    "human_review_questions": [
      "Is the knowledge goal correct and specific enough?",
      "Is candidate writing approved after reviewing the proposal?",
      "Should overlap create new knowledge, update existing knowledge, merge, or stop?"
    ],
    "commands": {
      "write_proposal": "write JSON to knowledge/.inbox/proposals/mall-api/product-list-cache/proposal.json",
      "verify_candidate": "argos author verify --json --proposal knowledge/.inbox/proposals/mall-api/product-list-cache/proposal.json --path knowledge/.inbox/packages/mall-api/product-list-cache"
    }
  }
}
```

The exact field names are intentionally operational. Agents should not need to
derive safety policy from prose.

## States

### `ready_for_proposal`

Use when inspect found no blocking overlap and no path issue that prevents a
proposal. This is the common path.

Expected guidance:

- write proposal JSON only;
- candidate writing remains blocked until human review approves it;
- provide proposal and candidate path suggestions;
- provide verify command for later use after candidate writing is approved.

### `review_only`

Use when a proposal artifact is useful, but candidate writing must not happen
yet.

Triggers:

- official, inbox, or index overlap requires a human choice;
- content is missing enough that a candidate would become empty reminder
  knowledge;
- requested delivery crosses an authorization boundary;
- candidate path is unsafe or outside allowed inbox paths.

Expected guidance:

- write proposal-only artifact;
- set proposal shape to review-only;
- leave candidate files empty;
- set verification validate path empty;
- record the exact review blocker.

### `blocked`

Use when even proposal writing is not safe or useful without more input.

Triggers:

- missing required `project` or `goal` is already handled by CLI validation, but
  internal callers may still need a state;
- registry cannot load;
- requested path cannot be represented safely;
- available information is too ambiguous to shape a proposal path or goal.

Expected guidance:

- ask the human for the missing input;
- do not write proposal or candidate files.

## Recommended Actions

`recommended_action` should be one of:

- `write_proposal`
- `write_review_only_proposal`
- `resolve_overlap`
- `collect_content`
- `ask_human`
- `stop`

The action is a short machine-friendly label. `action_reason` explains the
decision in human-readable language.

## Candidate Permission Model

`candidate_allowed` must be `false` in inspect output. Inspect happens before
human proposal review, so it cannot authorize candidate writes.

The packet may include a suggested `candidate_path`, because path design is part
of proposal review. The packet must pair that path with stop conditions that say
candidate files cannot be written until the proposal's
`human_review.candidate_write_approved` is true.

This prevents a common failure mode: agents treating a scaffold path as write
permission.

## Review-Only Packet Rules

When `review_only` is true, the packet should make the review-only shape
explicit:

- `proposed_shape.kind`: `review`
- `proposed_shape.type`: `review`
- `proposed_shape.artifact_state`: `review_only`
- `proposed_shape.entrypoint_load`: `reference_only`
- `candidate_files`: empty
- `verification_plan.validate_path`: empty string
- `human_review.candidate_write_approved`: false

The existing `proposal_scaffold` should already reflect this. The packet repeats
the decision so agents understand the operational consequence.

## Source Work

The packet should guide claim separation without inventing content:

- `user_confirmed`: preserve the user's stated goal and explicit decisions.
- `observed`: add only facts from inspected files, command output, tests, or
  docs.
- `imported`: use when user supplies external material.
- `synthesized`: mark agent recommendations and design synthesis.
- `templates` and `examples`: use for reusable scaffolds, code examples, or
  conventions.
- `assumptions`: make gaps explicit.
- `open_questions`: ask what is needed before candidate knowledge is usable.

This is guidance, not content generation. The agent still has to inspect source
material and design the knowledge.

## Commands

`commands` should contain command strings or command descriptions for agent use:

- `write_proposal`: path-oriented instruction, because file writing happens via
  the host environment rather than an Argos write command.
- `verify_candidate`: full `argos author verify --json ...` command when a
  candidate path exists.

For review-only packets, `verify_candidate` may be empty because no candidate
exists. The packet should say verification is not run until candidate writing is
approved.

## Relationship To Existing Fields

`proposal_scaffold` remains canonical for JSON structure.

`proposal_requirements` remains a compatibility list and terse checklist.

`recommended_next_steps` remains for existing consumers. In this slice it should
align with `authoring_packet.recommended_action`, but clients should prefer the
packet for detailed guidance.

`policy` remains raw policy facts. `authoring_packet` translates those facts into
the next operational action.

## CLI Behavior

`argos author inspect --json` keeps the same required flags and output format.
Existing consumers that ignore unknown JSON fields continue to work.

No non-JSON inspect output is added. The authoring path is AI-facing and already
requires `--json`.

## Testing Strategy

Unit tests should cover:

- normal no-overlap request returns `state=ready_for_proposal`,
  `recommended_action=write_proposal`, candidate path suggestions, and
  `candidate_allowed=false`;
- overlap request returns `state=review_only`,
  `recommended_action=resolve_overlap` or `write_review_only_proposal`, and
  review-only stop conditions;
- missing-content personal convention request returns `state=review_only`
  because preserving intent and review blockers is useful even when candidate
  content is absent;
- consumer API request includes consumer-audience packet guidance;
- CLI JSON includes `authoring_packet` without breaking existing inspect tests;
- dogfood `case-001`, `case-002`, `case-003`, and `case-008` continue to pass or
  produce the intended review-needed result.

## Rollout

Implement this as a backward-compatible response addition:

1. Add packet structs under `internal/author`.
2. Build the packet from the existing `InspectResponse` and `InspectRequest`
   after `proposal_scaffold` is built.
3. Keep packet generation deterministic and side-effect-free.
4. Update README and the capture-knowledge skill guidance after tests pass.
5. Rerun targeted authoring dogfood to confirm runner behavior improves without
   adding new command surface.

## Acceptance Criteria

- `author inspect --json` includes `authoring_packet`.
- The packet gives a clear next action and stop conditions for both candidate and
  review-only paths.
- Inspect never reports candidate writing as approved.
- Existing inspect response fields remain present.
- Fresh-runner authoring cases do not require hidden context to understand the
  next write-side action.
- Full Go tests pass.
