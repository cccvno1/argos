# Argos Agent Knowledge Authoring Protocol Design

Date: 2026-05-02

## Purpose

Argos should help agents create engineering knowledge that future agents can
find, load, apply, and cite during development work.

This is not a memo or note-taking workflow. Authoring is the production side of
the same loop that Discovery consumes:

```text
author engineering knowledge
-> verify protocol and future findability
-> review and promote
-> future agent uses context/find/read/cite
```

The user should not need to learn Argos commands or file layouts. Humans should
focus on whether the proposed knowledge is correct, useful, well-scoped, and
safe to publish. Agents operate the Argos commands as structured, AI-facing
tools.

## Background

Argos already has a mature consumption path:

- `argos_context` carries project, phase, task, and files into follow-up calls.
- `argos_find_knowledge` and `argos_list_knowledge` discover knowledge without
  full bodies.
- `argos_read_knowledge` loads selected full bodies.
- `argos_cite_knowledge` supports citation accountability.
- Discovery reports support, usage, missing needs, search status, and next
  steps.
- Dogfood validates fresh-runner behavior without hidden expected values.

The write side is less productized. The repository has:

- a `capture-knowledge` skill with strong process guidance;
- `argos validate --path` and `argos validate --inbox`;
- `argos promote --path`;
- inbox directories for item and package candidates.

But the authoring path still depends too much on agent discipline. It lacks a
workflow-neutral artifact contract, deterministic inspection, future retrieval
verification, and a review packet that keeps humans focused on knowledge design
rather than command mechanics.

## Design Position

Argos must remain workflow-neutral.

Argos should not prescribe whether an agent uses brainstorming, plan mode,
parallel agents, custom user workflows, or any specific host-tool process.
Those workflows may help the agent think, but Argos should not enumerate or
control them.

Instead, Argos defines the artifact gates that any workflow must satisfy before
knowledge becomes trusted:

```text
Host workflow: free-form and user-owned
Argos authoring protocol: stable artifact contract
Argos verification: deterministic evidence
Human review: publish decision
```

Argos verifies the knowledge artifact and its intended retrieval behavior, not
the agent's thinking process.

## Goals

- Reframe capture as agent knowledge authoring.
- Make the write path serve future agent development tasks, not human memory.
- Support purpose-built engineering knowledge, not only lessons derived from
  past development.
- Keep commands AI-facing and minimize human command knowledge.
- Define a `Knowledge Design Proposal` artifact that can be reviewed by humans
  and checked by Argos.
- Add deterministic authoring gates for inspection and verification.
- Validate that candidate knowledge is protocol-correct and findable for its
  intended future scenarios.
- Preserve explicit human approval before durable writes, official mutation,
  `priority: must`, and promotion.

## Non-Goals

- Do not build a general workflow engine.
- Do not require or name specific host workflow tools.
- Do not automate AI runner execution.
- Do not make Argos generate authoritative knowledge from vague intent alone.
- Do not automatically mutate official knowledge.
- Do not automatically promote inbox candidates.
- Do not make Discovery results automatically trigger authoring.
- Do not require humans to operate authoring CLI commands directly.
- Do not solve semantic search or ranking enhancements in this slice.

## Roles

### Human

The human owns knowledge intent and review decisions:

- whether a topic deserves durable knowledge;
- whether the proposed scope and wording are correct;
- whether assumptions are acceptable;
- whether `priority: must` is authorized;
- whether an inbox candidate should be edited, kept, promoted, or discarded.

### Agent

The agent owns authoring work inside the host workflow:

- gather context;
- ask clarifying questions when the knowledge goal is underspecified;
- use Argos inspection facts;
- produce the `Knowledge Design Proposal`;
- write candidate files only after approval;
- run Argos verification;
- present a human review packet.

### Argos

Argos owns deterministic protocol and evidence:

- registry facts;
- existing official and inbox knowledge checks;
- index-backed overlap when available;
- proposal schema validation;
- candidate path and metadata validation;
- knowledge protocol validation;
- future retrieval simulation;
- verification output.

## Authoring Modes

Every proposal declares an authoring mode. The mode defines trust posture and
review expectations.

### `observed`

Knowledge derived from repository files, tests, docs, configs, command output,
or reviewed implementation work.

Observed knowledge must name evidence paths or commands. The agent may infer
lessons from observed evidence, but inferences must be marked separately from
facts.

### `user_specified`

Knowledge defined by the user as a standard, preference, architectural
constraint, or operating rule.

User-specified knowledge may become a `rule` or `decision`, but broad scope and
`priority: must` still require explicit user authorization in the proposal.

### `synthesized`

Knowledge proposed by the agent from general engineering expertise plus project
context.

Synthesized knowledge defaults to `status: draft`, `priority: should`, and must
record assumptions. It must not claim to be observed project truth unless
evidence supports it.

### `imported`

Knowledge based on external material supplied by the user, such as API docs,
team standards, ADRs, design docs, migration notes, or vendor references.

Imported knowledge must preserve provenance and distinguish source statements
from local project interpretation.

### `template`

Reusable scaffolds, checklists, examples, prompts, scripts, or review guides
intended to help future agents perform recurring development tasks.

Examples must declare trust level: `illustrative`, `tested`, or `template`.
Scripts are never executed by Argos verification unless a separate user-approved
workflow does so.

## Knowledge Lifecycle

### 1. Author Intent

The user may start from an incomplete goal:

- "Generate engineering knowledge for product-list cache."
- "Create an auth review checklist for future agents."
- "Turn this external API guide into project knowledge."
- "Make a package that helps agents work on payments webhooks."

The agent must not assume the first wording is the final knowledge design.

### 2. Knowledge Design

The agent helps the user turn intent into an operational knowledge design. This
is workflow-neutral: the agent may use any host workflow, but the output must be
the same artifact contract.

The design must answer:

- What future agent work should this knowledge support?
- What type or package shape fits that work?
- What evidence or authority supports the claims?
- What should future agents load immediately versus on demand?
- Where can the knowledge be misapplied?
- What existing official or inbox knowledge overlaps?
- What review decisions remain open?

No files are written before proposal approval.

### 3. Inspect

Argos provides objective facts for the design:

```bash
argos author inspect --json \
  --project mall-api \
  --goal "create product-list cache engineering knowledge" \
  --mode synthesized \
  --future-task "implement product list cache" \
  --phase implementation \
  --files internal/catalog/products.go
```

`inspect` does not write files. It should work without an index, but include
index-backed overlap when an index exists.

### 4. Proposal

The agent writes a `Knowledge Design Proposal` artifact. This is the main
human-reviewed design product.

The proposal is not merely prose. It has a machine-checkable JSON contract and
a human-readable review packet. The JSON contract is canonical for
verification; the Markdown review packet is optimized for human review.

The documented repository-local proposal location is a slugged subdirectory of
`knowledge/.inbox/proposals/`, for example
`knowledge/.inbox/proposals/product-list-cache/proposal.json`. Commands accept
an explicit proposal path, but docs, tests, and dogfood should use this inbox
proposal location. Proposal files are authoring artifacts; they are not official
knowledge and are not promoted by `argos promote`.

### 5. Candidate Draft

After proposal approval, the agent writes candidate knowledge under the
approved boundary:

```text
knowledge/.inbox/items/
knowledge/.inbox/packages/
```

Official knowledge may be modified only in an explicit review path approved by
the user. The default path for AI-authored knowledge is inbox.

### 6. Verify

Argos verifies the proposal and candidate:

```bash
argos author verify --json \
  --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json \
  --path knowledge/.inbox/packages/backend/product-list-cache
```

Verification combines schema checks, knowledge validation, policy checks, and
future retrieval simulation.

### 7. Human Review And Promotion

The human reviews the packet and chooses:

- request edits;
- keep candidate in inbox;
- authorize `priority: must`;
- promote to official knowledge;
- stop or discard.

Promotion remains explicit and separate:

```bash
argos promote --path knowledge/.inbox/packages/backend/product-list-cache
argos index
```

## Knowledge Design Proposal

The proposal JSON must contain:

```json
{
  "schema_version": "authoring.proposal.v1",
  "knowledge_goal": "...",
  "authoring_mode": "observed | user_specified | synthesized | imported | template",
  "project": "mall-api",
  "proposed_shape": {
    "kind": "item | package",
    "type": "rule | decision | lesson | runbook | reference | package",
    "title": "...",
    "id": "...",
    "path": "knowledge/.inbox/packages/...",
    "status": "draft",
    "priority": "should"
  },
  "future_retrieval_contract": {
    "tasks": ["implement product list cache"],
    "phases": ["implementation", "review"],
    "files": ["internal/catalog/products.go"],
    "file_globs": ["internal/catalog/**"],
    "query_phrases": ["product list cache", "cache ttl product list"],
    "projects": ["mall-api"],
    "tech_domains": ["backend", "database"],
    "business_domains": ["catalog"],
    "tags": ["cache", "redis"],
    "expected_use": "read_before_implementation",
    "cite_after_use": true
  },
  "source_and_trust": {
    "user_provided": [],
    "observed": [],
    "imported": [],
    "synthesized": [],
    "assumptions": [],
    "open_questions": []
  },
  "applicability": {
    "when_to_use": [],
    "when_not_to_use": [],
    "tradeoffs": []
  },
  "overlap_decision": {
    "official_overlap": [],
    "inbox_overlap": [],
    "decision": "create_new | update_existing | stop | unresolved",
    "reason": "..."
  },
  "delivery": {
    "path": "inbox | official_review",
    "requires_human_approval": true,
    "priority_must_authorized": false,
    "promote_authorized": false
  },
  "candidate_files": [
    {
      "path": "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md",
      "purpose": "entrypoint",
      "load": "start_here"
    }
  ],
  "verification_plan": {
    "validate_path": "knowledge/.inbox/packages/backend/product-list-cache",
    "findability_scenarios": [
      {
        "project": "mall-api",
        "phase": "implementation",
        "task": "implement product list cache",
        "query": "cache ttl product list",
        "files": ["internal/catalog/products.go"]
      }
    ]
  }
}
```

The human review packet renders the same information under readable headings:

- Knowledge Goal
- Proposed Shape
- Future Retrieval Contract
- Source And Trust
- Applicability Boundaries
- Overlap Check And Decision
- Candidate Files
- Verification Plan
- Human Review Decisions

## Deterministic Gates

### `argos author inspect`

`inspect` gathers facts and policy context. It does not design the knowledge by
itself.

Inputs:

- `project` required;
- `goal` required;
- `mode` optional but recommended;
- `future-task`, `phase`, `query`, `files`, `domains`, `tags` optional;
- `candidate-path` optional for path risk checks.

Response:

```json
{
  "project": "mall-api",
  "goal": "create product-list cache engineering knowledge",
  "mode": "synthesized",
  "capabilities": {
    "filesystem": "enabled",
    "index": "enabled"
  },
  "registry": {
    "project_known": true,
    "tech_domains": ["backend", "database"],
    "business_domains": ["catalog"]
  },
  "overlap": {
    "official": [],
    "inbox": [],
    "index": []
  },
  "path_risk": {
    "candidate_path": "",
    "status": "not_checked"
  },
  "policy": {
    "write": "after_proposal_approval",
    "official_mutation": "requires_explicit_review_path",
    "promote": "requires_explicit_approval",
    "priority_must": "requires_explicit_authorization",
    "synthesized_claims": "must_mark_assumptions"
  },
  "proposal_requirements": [
    "future_retrieval_contract",
    "source_and_trust",
    "applicability",
    "overlap_decision",
    "verification_plan"
  ],
  "recommended_next_steps": [
    {
      "step": "write_knowledge_design_proposal",
      "reason": "Human review is required before durable writes."
    }
  ]
}
```

Overlap search should include:

- official items and packages;
- inbox items and packages;
- index-backed find results when available;
- simple filesystem text matches when index is unavailable.

`inspect` must label heuristic matches as possible overlap, not as final
decisions.

### `Knowledge Design Proposal`

The proposal is created by the agent and reviewed by the human. Argos may
provide schema and validation, but it should not pretend to have solved the
knowledge design alone.

### `argos author verify`

`verify` checks proposal and candidate artifacts.

Inputs:

- `proposal` path required;
- `path` candidate path required;
- optional `--json`;
- optional `--strict` in a later slice.

Verification checks:

1. Proposal schema is valid and complete.
2. Candidate path matches the proposal.
3. Candidate path stays in the approved delivery boundary.
4. `argos validate --path` equivalent passes.
5. Candidate metadata aligns with the future retrieval contract:
   - project scope;
   - domains and tags;
   - file globs or `applies_to`;
   - type and phase use.
6. Policy checks pass:
   - inbox candidates use `status: draft`;
   - `priority: must` requires authorization;
   - synthesized claims are represented as assumptions or review-needed;
   - official mutation is not implied by an inbox proposal.
7. Findability simulation passes or returns review-needed.

Response:

```json
{
  "result": "pass | fail | review-needed",
  "proposal": {
    "path": "proposal.json",
    "status": "valid"
  },
  "candidate": {
    "path": "knowledge/.inbox/packages/backend/product-list-cache",
    "validation": "pass"
  },
  "policy": {
    "result": "pass",
    "findings": []
  },
  "findability": {
    "result": "pass",
    "scenarios": [
      {
        "task": "implement product list cache",
        "matched_candidate": true,
        "support": "strong"
      }
    ]
  },
  "findings": []
}
```

`fail` means the artifact violates protocol or policy. `review-needed` means
the artifact may be acceptable but needs human judgment: weak findability,
missing evidence, broad assumptions, unresolved overlap, or incomplete review
decisions.

## Future Retrieval Simulation

Authoring verification should answer:

> If a future agent starts with the intended task, will this knowledge be
> discoverable and useful through the normal Discovery path?

The first implementation can build a temporary index containing official
knowledge plus the candidate. It should not promote the candidate or mutate the
workspace index.

Simulation uses the proposal's `findability_scenarios` and checks:

- the candidate appears in results for intended tasks;
- obvious unrelated official knowledge does not outrank it in a way that hides
  the candidate;
- support is not `none`;
- the candidate's summary/type/metadata explain why it should be read;
- full body remains gated behind read.

This is not a ranking guarantee. It is an authoring quality signal.

## Human Review Packet

Humans review design and evidence, not command details.

The agent should present a concise packet:

```markdown
# Argos Knowledge Authoring Review

## Knowledge Goal

## Proposed Shape

## Future Retrieval Contract

## Source And Trust

## Applicability Boundaries

## Overlap Check And Decision

## Candidate Files

## Verification Evidence

## Human Review Decisions
```

The packet should make these decisions explicit:

- approve proposal before writing;
- request edits;
- approve candidate to remain in inbox;
- authorize `priority: must`;
- approve promotion to official knowledge;
- stop.

## Command Surface

The first product slice should add:

```bash
argos author inspect --json ...
argos author verify --json \
  --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json \
  --path knowledge/.inbox/packages/backend/product-list-cache
```

Do not add `author propose` as a CLI that generates knowledge design prose in
the first slice. Proposal creation requires reasoning, context reading, and
human collaboration. That belongs to the agent and host workflow, not a
deterministic Go command.

Do not add automatic `author write` or `author promote` in the first slice.
Agents can write files after approval using normal file operations. Argos then
verifies the artifacts.

MCP tools can mirror `inspect` and `verify` after the CLI contracts stabilize.
They should remain deterministic and should not expose direct official mutation
tools.

## Skill And Documentation Changes

The existing `capture-knowledge` skill should be reframed as agent knowledge
authoring.

Recommended change:

- either rename it to `author-knowledge`;
- or keep `capture-knowledge` as a compatibility alias while the skill body uses
  authoring terminology.

The skill should:

- follow the user's host workflow without naming specific workflow systems;
- call or emulate `argos author inspect`;
- help the human design future-agent operational knowledge;
- produce the `Knowledge Design Proposal`;
- write inbox candidates only after approval;
- call `argos author verify`;
- present the Human Review Packet;
- never auto-promote.

README should present authoring as an AI-operated workflow. Humans express
goals and review packets; agents run commands.

## Authoring Harness

Authoring needs its own dogfood harness rather than being forced into the
Discovery dogfood suite.

Suggested fixtures:

```text
testdata/authoring-golden/
```

Case types:

1. `purpose_built_rule_user_specified`
   - User asks for an engineering rule that did not arise from recent work.
   - Expected: proposal marks user authority, future retrieval contract, and no
     unsupported `priority: must`.

2. `observed_lesson_from_repo`
   - Agent derives a lesson from repository files or tests.
   - Expected: evidence paths are recorded and assumptions are separate.

3. `synthesized_package_requires_assumptions`
   - Agent creates a suggested package from general engineering expertise plus
     repo context.
   - Expected: assumptions are explicit and candidate remains draft.

4. `overlap_requires_user_choice`
   - Existing official or inbox knowledge overlaps.
   - Expected: no candidate is written until overlap decision is present.

5. `candidate_validates_but_not_findable`
   - Candidate passes file validation but fails future retrieval simulation.
   - Expected: `author verify` returns review-needed.

6. `priority_must_requires_authorization`
   - Candidate asks for `priority: must`.
   - Expected: verification fails or returns review-needed without explicit
     authorization in the proposal.

Authoring runner packets should not contain hidden expected IDs or ideal
proposals. They should test whether a fresh agent can produce a complete
proposal, write only inside allowed boundaries, and generate verification
evidence.

## Testing Strategy

Unit tests:

- `inspect` reports registry facts and policy.
- `inspect` finds official overlap.
- `inspect` finds inbox overlap.
- `inspect` degrades gracefully without an index.
- proposal schema validation rejects missing future retrieval contract.
- proposal schema validation rejects missing source/trust information.
- `verify` rejects candidate path outside approved boundary.
- `verify` rejects `priority: must` without authorization.
- `verify` returns review-needed for synthesized claims without assumptions.
- `verify` runs existing knowledge validation.
- `verify` simulates findability without mutating the workspace index.

CLI tests:

- `author inspect --json` requires JSON, project, and goal.
- `author inspect --json` returns overlap and policy fields.
- `author verify --json` requires proposal and path.
- `author verify --json` returns pass/fail/review-needed.

Dogfood:

- fresh runner starts from a vague purpose-built engineering knowledge goal;
- runner does not write before proposal approval;
- runner writes inbox only;
- runner runs verify;
- runner presents a human review packet.

## Migration Plan

1. Add this design as the active authoring direction.
2. Keep older capture specs as historical references.
3. Add `internal/author` for inspect and verify.
4. Add `argos author inspect` and `argos author verify`.
5. Reframe `skills/capture-knowledge` toward authoring protocol.
6. Add authoring harness fixtures and dogfood evaluation.
7. Update README command examples and human experience wording.

## Success Criteria

- Humans can ask for purpose-built engineering knowledge without learning Argos
  commands.
- Agents can produce a complete knowledge design before writing files.
- Argos can verify proposal completeness, candidate validity, policy
  compliance, and future findability.
- AI-authored knowledge defaults to inbox and cannot silently become official.
- Discovery remains separate from authoring; missing needs do not automatically
  become write prompts.
- The authoring harness catches malformed proposals, unsafe write boundaries,
  unsupported `must`, and non-findable candidates.

## Risks

- The proposal schema may feel heavy for small lessons.
  Mitigation: allow compact proposals, but never omit future retrieval,
  source/trust, overlap, and review decisions.

- Findability simulation may overfit to supplied scenarios.
  Mitigation: treat simulation as evidence, not proof. Keep human review in the
  loop.

- Agents may still skip design under pressure.
  Mitigation: verification rejects missing proposal artifacts and dogfood tests
  the full workflow.

- Naming transition from capture to authoring may confuse existing docs.
  Mitigation: keep `capture-knowledge` as a compatibility trigger while
  reframing the workflow around authoring.

## Implementation Decisions

- Proposal artifacts use explicit paths, with docs and harness fixtures under
  `knowledge/.inbox/proposals/`.
- `author verify` reads the canonical JSON proposal in the first slice. Markdown
  review packets are generated for humans but are not parsed by Argos.
- MCP authoring tools wait until CLI behavior is stable. The first slice is CLI
  only.
- Candidate skeleton generation is not included in the first slice. Agents write
  approved candidate files directly; a later `author scaffold` command can add
  templates if repeated authoring friction appears.
