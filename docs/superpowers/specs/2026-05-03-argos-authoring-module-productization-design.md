# Argos Authoring Module Productization Design

Date: 2026-05-03

## Purpose

Finish the write-side authoring module to a first-release standard comparable
to the discovery module.

Comparable does not mean mirroring discovery. Discovery is a consumption
protocol for existing knowledge. Authoring is a production protocol for turning
human intent, observed project facts, imported material, and AI-designed
guidance into reusable knowledge that future agents can find and apply.

The module is complete for first release when an AI agent can:

1. recognize that the user wants durable knowledge;
2. ask Argos how to design that knowledge;
3. write a reviewable knowledge design;
4. write draft knowledge only after the review boundary allows it;
5. check that draft knowledge is valid and findable;
6. publish only after explicit authorization;
7. confirm the published knowledge is discoverable through the normal query
   path.

Humans should mainly think about the knowledge goal and review the proposed
knowledge design. They should not need to remember Argos paths, hidden review
rules, or command choreography.

## Current State

The write side already has the core pieces:

- `argos author inspect --json`
- `argos author verify --json`
- `authoring.proposal.v2`
- `authoring_packet`
- authoring dogfood cases, packets, report template, evaluator, and hidden
  oracle checks
- promotion and reindexing primitives
- discovery-side patterns for MCP tools, adapters, usage guidance, and dogfood
  reports

The remaining gap is product shape. Public names still expose too much internal
review vocabulary, the AI-facing entrypoints are CLI-oriented, and the harness
does not yet enforce write guidance as the public behavior contract.

## Decision

Make the public authoring module use a simple pipeline:

```text
intent -> knowledge_design -> draft_knowledge -> check -> publish
```

The public AI-facing vocabulary must use this pipeline consistently in CLI JSON,
MCP tools, adapter guidance, dogfood packets, report templates, and docs.

Replace old public authoring names directly:

- `authoring_packet`
- `proposal_scaffold`
- `proposal`
- `candidate`
- `source_profile`
- `proposed_shape`
- `overlap_decision`
- `delivery`
- `verification_plan`
- `human_review`
- `artifact_state`

Because the project is not in production, there is no migration or compatibility
requirement for the old public vocabulary. The first-release public surface
should expose only the new vocabulary. Internal Go package names may change in
the same slice or in small mechanical follow-ups, but no public JSON, MCP
schema, adapter, dogfood asset, or help text should keep the old names.

## Goals

- Make the write-side module agent-first and human-review-friendly.
- Replace public authoring jargon with task vocabulary that communicates intent.
- Provide write-side MCP tools comparable in quality to discovery MCP tools.
- Teach generated adapters when and how to start the write-side flow.
- Make dogfood harnesses verify that agents followed the write guidance.
- Validate realistic authoring scenarios, including design-only knowledge that
  did not originate in code.
- Keep Argos workflow-neutral. Argos must not depend on Plan mode,
  brainstorming, subagents, or any host workflow.
- Keep publishing explicitly authorized. Argos may guide and check drafts, but
  should not silently promote or reindex official knowledge.

## Non-Goals

- Do not build a UI.
- Do not add model calls or agent orchestration inside Argos.
- Do not implement semantic search or ranking changes.
- Do not implement team permissions, accounts, or collaboration workflows.
- Do not automate human approval decisions.
- Do not infer official organization policy from AI-suggested guidance.
- Do not require a specific host workflow for knowledge design.

## Public Vocabulary

Use these names in all public authoring surfaces.

| Old public/internal word | New public word |
| --- | --- |
| authoring packet | write guidance |
| proposal scaffold | knowledge design template |
| proposal | knowledge design |
| candidate | draft knowledge |
| source profile | sources |
| user confirmed, user stated | user input |
| synthesized | ai suggested |
| proposed shape | draft output |
| overlap decision | existing knowledge |
| delivery | write boundary |
| verification plan | check plan |
| human review | review |
| artifact state | draft state |

Internal Go type names may be renamed when that keeps the implementation clear.
If a private name temporarily survives to keep one slice reviewable, it must not
appear in public JSON, MCP schemas, adapter text, dogfood templates, help text,
or docs.

## Public CLI Surface

Replace the public write-side CLI surface with knowledge-oriented commands:

```bash
argos knowledge design --json --project <project> --intent <intent>
argos knowledge check --json --design <design.json> --draft <draft-path>
argos knowledge publish --path <draft-path>
```

Relationship to existing commands:

- `argos knowledge design` replaces `argos author inspect`.
- `argos knowledge check` replaces `argos author verify`.
- `argos knowledge publish` is the public publishing command backed by the
  current promote implementation.
- `argos author inspect` and `argos author verify` should be removed from the
  public first-release CLI surface and help text.

`design` remains read-only with respect to knowledge files. It returns guidance
and a design template, but does not write the design or the draft.

`check` validates a written design and draft. It must not publish.

`publish` requires an explicit user action. It should remain outside automatic
MCP write guidance for first release.

## Public MCP Surface

Add two agent-facing tools:

```text
argos_design_knowledge
argos_check_knowledge
```

Do not add an automatic publish MCP tool in this slice. Publishing should remain
an explicit CLI or human-mediated action for first release.

### `argos_design_knowledge`

Purpose: turn a user's durable-knowledge intent into write guidance and a
knowledge design template.

Request:

```json
{
  "project": "mall-api",
  "intent": "Create Redis cache best practices for future backend agents.",
  "future_task": "Implement or review Redis caching behavior.",
  "phase": "planning",
  "query": "redis cache best practices",
  "files": ["internal/cache/client.go"],
  "domains": ["backend", "redis"],
  "tags": ["cache", "redis"],
  "draft_path": "knowledge/.inbox/packages/mall-api/redis-cache-best-practices"
}
```

Required:

- `project`
- `intent`

Response:

```json
{
  "project": "mall-api",
  "intent": "Create Redis cache best practices for future backend agents.",
  "write_guidance": {
    "state": "ready_to_design",
    "next_action": "write_knowledge_design",
    "reason": "No blocking existing knowledge was found. Write the design first; draft knowledge remains blocked until review approves it.",
    "design_path": "knowledge/.inbox/designs/mall-api/redis-cache-best-practices/design.json",
    "draft_path": "knowledge/.inbox/packages/mall-api/redis-cache-best-practices",
    "draft_allowed": false,
    "design_only": false,
    "stop_conditions": [
      "Do not write draft knowledge until review.draft_write_approved is true.",
      "Do not publish unless review.publish_approved is true.",
      "Do not write official knowledge unless review.official_write_approved is true.",
      "Do not use priority must unless review.priority_must_approved is true."
    ],
    "design_focus": [
      "Clarify what future agents should do differently after reading this knowledge.",
      "Separate user input, observed facts, imported material, AI suggestions, assumptions, and open questions.",
      "Record when to use and when not to use the knowledge.",
      "Record existing knowledge decisions before draft writing."
    ],
    "review_questions": [
      "Is this the right knowledge goal for future agents?",
      "Is draft writing approved after this design is reviewed?",
      "Should this remain personal/project draft knowledge or become official later?"
    ],
    "commands": {
      "write_design": "write JSON to knowledge/.inbox/designs/mall-api/redis-cache-best-practices/design.json",
      "check_draft": "argos knowledge check --json --design knowledge/.inbox/designs/mall-api/redis-cache-best-practices/design.json --draft knowledge/.inbox/packages/mall-api/redis-cache-best-practices"
    }
  },
  "knowledge_design_template": {
    "schema_version": "knowledge.design.v1"
  }
}
```

### `argos_check_knowledge`

Purpose: validate that a written knowledge design and draft knowledge are safe,
consistent, and findable.

Request:

```json
{
  "design": "knowledge/.inbox/designs/mall-api/redis-cache-best-practices/design.json",
  "draft": "knowledge/.inbox/packages/mall-api/redis-cache-best-practices"
}
```

Response:

```json
{
  "result": "pass",
  "design": {
    "path": "knowledge/.inbox/designs/mall-api/redis-cache-best-practices/design.json",
    "validation": "pass"
  },
  "draft": {
    "path": "knowledge/.inbox/packages/mall-api/redis-cache-best-practices",
    "validation": "pass",
    "ids": ["package:mall-api.redis-cache-best-practices.v1"]
  },
  "policy": {
    "result": "pass"
  },
  "findability": {
    "result": "pass",
    "checks": []
  }
}
```

`result` values stay aligned with existing Argos status vocabulary:

- `pass`
- `review-needed`
- `fail`

## Knowledge Design Schema

Introduce `knowledge.design.v1` as the canonical public design schema. The
existing `authoring.proposal.v2` implementation can inform the code changes, but
the first-release public schema should not expose `authoring.proposal.v2` or its
old field names.

Top-level fields:

```json
{
  "schema_version": "knowledge.design.v1",
  "user_request": "",
  "knowledge_goal": "",
  "project": "",
  "audience": {},
  "scope": {},
  "sources": {},
  "draft_output": {},
  "future_use": {},
  "applicability": {},
  "existing_knowledge": {},
  "write_boundary": {},
  "draft_files": [],
  "check_plan": {},
  "review": {}
}
```

Keep these names because they are already direct enough:

- `user_request`
- `knowledge_goal`
- `project`
- `audience`
- `scope`
- `future_use`
- `applicability`

Rename the less direct fields:

- `source_profile` to `sources`
- `proposed_shape` to `draft_output`
- `overlap_decision` to `existing_knowledge`
- `delivery` to `write_boundary`
- `candidate_files` to `draft_files`
- `verification_plan` to `check_plan`
- `human_review` to `review`

### Sources

`sources` should make knowledge trust visible:

```json
{
  "user_input": [],
  "observed": [],
  "imported": [],
  "ai_suggested": [],
  "templates": [],
  "examples": [],
  "assumptions": [],
  "open_questions": [],
  "claims": []
}
```

Rules:

- `user_input` is what the user stated or approved.
- `observed` is from files, commands, tests, docs, or generated artifacts the
  agent inspected.
- `imported` is material supplied by the user from outside the workspace.
- `ai_suggested` is designed or recommended by the AI and requires review unless
  a human explicitly approves it.
- `assumptions` and `open_questions` must stay visible when content is not
  ready for draft writing.

This supports personal, small-team, project, organization, and public-consumer
knowledge without forcing a single source workflow.

### Write Guidance States

Use simple states:

- `ready_to_design`: write a knowledge design; draft remains blocked.
- `design_only`: write a knowledge design only because draft writing needs a
  human decision, more content, or existing-knowledge resolution.
- `blocked`: do not write design or draft until required input is provided.

`draft_allowed` must be `false` in design responses. A design response happens
before review, so it cannot authorize draft writing.

### Design Review Boundary

The design schema should include:

```json
{
  "review": {
    "questions": [],
    "design_approved": false,
    "draft_write_approved": false,
    "priority_must_approved": false,
    "official_write_approved": false,
    "publish_approved": false,
    "requested_edits": [],
    "unresolved_blockers": []
  }
}
```

This replaces public `human_review` wording with a shorter task word.

## Data Flow

### Design Flow

1. Agent detects durable knowledge intent from the user.
2. Agent calls `argos_design_knowledge` or `argos knowledge design`.
3. Argos loads registry, official knowledge, inbox knowledge, and optional
   index overlap data.
4. Argos returns `write_guidance` and `knowledge_design_template`.
5. Agent writes only the knowledge design file.
6. Human reviews the design and updates review booleans or requested edits.
7. Agent writes draft knowledge only if `review.draft_write_approved` is true.

### Check Flow

1. Agent calls `argos_check_knowledge` or `argos knowledge check`.
2. Argos loads the design file and draft path.
3. Argos validates schema, source separation, write boundary, draft metadata,
   future-use contract, and findability checks.
4. Argos returns `pass`, `review-needed`, or `fail`.
5. Agent reports the result and does not publish automatically.

### Publish Flow

1. Human explicitly authorizes publishing.
2. Agent runs `argos knowledge publish --path <draft-path>`.
3. Agent runs `argos index`.
4. Agent confirms the knowledge is discoverable by calling
   `argos_find_knowledge` or equivalent CLI JSON.

The publish flow belongs in end-to-end acceptance, but it should not become an
automatic MCP action in this slice.

## Adapter Protocol

Generated adapter files should add a write-side section:

1. Use write-side Argos only when the user explicitly asks to create, capture,
   improve, or publish durable knowledge.
2. Do not start from `argos_find_knowledge` results alone. Query results may
   reveal missing needs, but writing still starts with `argos_design_knowledge`.
3. Write a knowledge design before any draft knowledge.
4. Separate user input, observed facts, imported material, AI suggestions,
   assumptions, and open questions.
5. Write draft knowledge only after the design review says draft writing is
   approved.
6. Run `argos_check_knowledge` after writing draft knowledge.
7. Publish only after explicit user authorization.
8. After publishing, rebuild the index and confirm the knowledge is findable.

This keeps Argos focused on the knowledge boundary while host workflows remain
responsible for thinking, planning, parallelism, and context management.

## Dogfood Harness Contract

Authoring dogfood must be upgraded from instruction-following to
contract-following.

Packet generation should:

- seed the fixture workspace before building the public packet;
- call the same design logic used by `argos_design_knowledge`;
- embed `write_guidance` and the public design template in the runner packet;
- stop using old public names in runner instructions.

Report template should require:

- write guidance state;
- next action;
- design path;
- draft path or `none`;
- draft allowed;
- design-only status;
- check result or `not-run`;
- review booleans;
- guard statuses.

Evaluator should verify:

- reported state and next action match the generated write guidance;
- draft writing did not happen when `draft_allowed=false` or review blocked it;
- design-only cases did not write draft knowledge;
- draft path matches write guidance and design review;
- check ran when draft knowledge exists;
- official write, priority must, publish, and promotion boundaries were not
  crossed without explicit approval;
- hidden oracle expectations remain private.

## Realistic Acceptance Cases

The authoring module is not complete until these scenarios are covered by
dogfood or end-to-end tests:

1. Go template standard: user asks to turn a generated Go template into future
   implementation guidance.
2. Redis best practices: user and AI jointly design recommended behavior with
   assumptions and review-needed AI suggestions.
3. Business interface knowledge: user wants business/API points organized for
   other agents or consumers.
4. Missing content: user gives a goal but not enough actionable content, so the
   result is design-only.
5. Existing knowledge overlap: Argos detects related official or inbox
   knowledge and requires a human decision before draft writing.
6. Draft check and query loop: after an approved draft is checked and published,
   `argos_find_knowledge` can find it for the future task.

## Error Handling

Use errors only for invalid requests or failed system operations:

- missing required request fields;
- unsafe absolute or parent-directory paths;
- registry, filesystem, index, or JSON parse failures;
- invalid MCP arguments.

Use `write_guidance.state` and `result=review-needed` for ordinary authoring
uncertainty:

- vague intent;
- missing source content;
- existing knowledge overlap;
- unapproved official write;
- unapproved publish;
- unapproved priority must;
- draft not findable enough for the future task.

This keeps expected review work out of the failure path.

## Testing Strategy

Unit tests:

- design response emits `write_guidance` and `knowledge_design_template` using
  new public names;
- old public names are absent from new MCP responses and generated dogfood
  packets;
- design-only states block draft writing;
- missing content becomes design-only instead of an invalid draft;
- existing knowledge overlap becomes design-only or requires a decision;
- check validates design, draft path, write boundary, source separation, and
  findability.

CLI tests:

- `argos knowledge design --json` works and requires `project` and `intent`;
- `argos knowledge check --json` works and requires `design` and `draft`;
- public help text does not list `argos author inspect` or
  `argos author verify`;
- invoking removed old write-side public commands returns the same style of
  unknown-command error as other absent commands;
- help text uses design/draft/check/publish vocabulary.

MCP tests:

- `tools/list` includes `argos_design_knowledge` and `argos_check_knowledge`;
- schemas are strict and reject unknown fields;
- missing required fields return tool errors;
- tool responses use public names.

Adapter tests:

- generated adapter guidance explains when to design knowledge;
- it forbids direct writes from query results alone;
- it requires check before publish;
- it requires explicit publish authorization.

Dogfood tests:

- packet, report template, parser, and evaluator agree on required fields;
- evaluator detects runner violations of write guidance;
- public dogfood assets do not leak hidden oracle fields;
- fresh-runner reports can be evaluated without hidden context.

End-to-end tests:

- build a draft from a realistic design;
- run check;
- publish with explicit authorization;
- rebuild index;
- confirm query discovery finds the published knowledge.

## Rollout Slices

This is one module-level closure, but implementation should be split into
controlled slices.

### Slice 1: Public Vocabulary And Schema

- Introduce `knowledge.design.v1` public schema.
- Return `write_guidance` and `knowledge_design_template`.
- Add canonical CLI commands under `argos knowledge`.
- Remove old public response field names from canonical JSON responses.
- Remove old write-side public CLI verbs and help text.

### Slice 2: AI Entry And Harness Contract

- Add MCP tools.
- Update generated adapters.
- Make dogfood packet generation consume real write guidance.
- Update report parser and evaluator to enforce write guidance.

### Slice 3: End-To-End Release Validation

- Run realistic authoring cases.
- Check draft knowledge.
- Publish only in explicitly approved test fixtures.
- Reindex.
- Confirm discovery can find the new knowledge.
- Record the round report and remaining non-blocking follow-up items.

## Acceptance Criteria

The write-side module is first-release complete when:

- public authoring vocabulary is consistent across CLI JSON, MCP, adapters,
  dogfood packet, report template, and docs;
- AI-facing entrypoints exist for design and check;
- generated adapters tell agents when to start the write flow and when not to;
- dogfood evaluator enforces write guidance instead of only checking final
  artifacts;
- draft writing, official writing, priority must, publish, and promotion all
  require explicit review authorization;
- realistic design, draft, check, publish, reindex, and query-findback flows
  pass;
- `go test ./... -count=1` passes.

After this point, remaining work should be treated as later module design or
enhancement, not as unfinished core authoring.
