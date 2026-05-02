# Argos Authoring V2 Contract And Harness Design

Date: 2026-05-02

## Purpose

Argos authoring should turn a human's fuzzy engineering knowledge intent into
future-agent-readable, verifiable, reviewable knowledge.

The authoring product is not a memo system. It exists so future agents can
discover, load, apply, and cite project knowledge while doing development work.

The first authoring protocol established the right safety gates:

```text
intent -> inspect -> proposal -> approved inbox candidate -> verify -> review
```

This v2 design tightens the contract around real authoring scenarios. It also
defines where the authoring harness belongs and what it should prove.

## Core Product Function

The production path should be:

```text
natural-language knowledge intent
-> agent helps design operational knowledge
-> Argos inspects existing knowledge, scope, overlap, and policy
-> agent writes a Knowledge Design Proposal
-> human reviews the proposal
-> agent writes inbox candidate after approval
-> Argos verifies proposal, candidate, policy, and future retrieval behavior
-> human decides keep, edit, promote, or discard
```

Humans should think about knowledge quality:

- what future agents should know;
- whether the proposed wording is correct;
- where the knowledge applies;
- where it must not apply;
- whether assumptions are acceptable;
- whether a rule is strong enough to publish.

Agents should operate Argos commands. Argos should give deterministic evidence.

## Why V2 Is Needed

The current `authoring.proposal.v1` contract is useful, but still too close to
Argos internals:

- `authoring_mode` is a single enum, while real knowledge usually combines
  user intent, observed evidence, imported references, templates, and
  synthesized judgment.
- Harness cases use `goal` and `mode`, which makes the runner solve an Argos
  form instead of a normal user request.
- Business-domain examples can accidentally narrow the product toward company
  workflow instead of personal, small-team, project, library, API, and
  organization knowledge.
- The harness risks checking whether a runner filled known fields rather than
  whether it designed useful future-agent knowledge.

V2 keeps the same safety principle but changes the center of gravity:

```text
from: classify this authoring mode
to: design knowledge for future agent use, with source/trust and retrieval
```

## Product Principles

### Agent-First

Authoring commands are primarily AI-facing. A human should be able to say:

> I designed a Go service template. Help me turn it into reusable knowledge for
> future agents.

The human should not need to know proposal paths, inbox layout, frontmatter
keys, or verification flags.

### Workflow-Neutral

Argos must not try to control the agent's thinking workflow.

An agent may use brainstorming, plan mode, parallel agents, custom project
instructions, or user-created workflows. Argos should not enumerate those flows
or depend on them. The stable guarantee is the artifact contract:

```text
Any host workflow is acceptable if it produces reviewable artifacts that pass
Argos authoring gates.
```

This is the right answer to the "can Argos force mature thinking tools?"
question. Argos can encourage and document good agent behavior, but it should
not make correctness depend on controlling a host tool's private workflow.

### Future-Agent Utility

Knowledge is designed for future work, not for human memory. Each authored
artifact should say:

- which future tasks should retrieve it;
- which agent role or workflow moment should use it;
- what should be read immediately;
- what should be loaded only on demand;
- what would be a harmful or noisy match.

### Source And Trust Before Authority

The product must distinguish:

- facts observed in repository material;
- user-defined standards and preferences;
- imported external references;
- agent-synthesized recommendations;
- examples and templates;
- assumptions and open questions.

Authority comes from source and review, not from confident prose.

### Inbox By Default

AI-authored, imported, or synthesized knowledge defaults to inbox:

```text
knowledge/.inbox/proposals/
knowledge/.inbox/items/
knowledge/.inbox/packages/
```

Official mutation, `priority: must`, and promotion remain explicit decisions.

## Real Scenario Model

V2 should support these scenarios without making any one of them the product's
identity.

### Go Template Standard

User request:

> I designed a Go template. I want future agents to write Go services in this
> style.

Likely knowledge shape:

- a package for Go service style;
- a start-here entrypoint;
- examples or templates;
- a rule or checklist for new service generation;
- negative triggers for one-off scripts, legacy services, or projects outside
  the declared scope.

Source profile:

- user-confirmed standards;
- observed template files if available;
- illustrative examples;
- synthesized rationale marked as such.

### Redis Best Practices

User request:

> We use Redis heavily. I am not an expert. Help me and the agent design a set
> of practices that future agents can use safely.

Likely knowledge shape:

- a draft best-practice package;
- rules only for decisions the user explicitly approves;
- references to imported docs if supplied;
- tradeoffs and anti-patterns;
- open questions for production-sensitive topics.

Source profile:

- synthesized recommendations;
- imported references if provided;
- user decisions;
- assumptions and unresolved risks.

This must not pretend to be company truth merely because the user said "we".

### API Or Business Capability Knowledge

User request:

> This project is called by many people. I want to describe the business points
> and interfaces so future agents can help other developers use it.

Likely knowledge shape:

- a reference or package for consumer-facing integration knowledge;
- API surface, concepts, examples, constraints, and caller responsibilities;
- audience marked as consumer agent, implementer agent, or reviewer agent;
- negative triggers for internal-only implementation tasks.

Source profile:

- user-provided domain knowledge;
- observed API/interface files;
- imported docs if supplied;
- assumptions separated from contract facts.

### Generalization

The same model must also work for:

- personal project conventions;
- small-team coding standards;
- reusable library usage notes;
- one-off migration lessons;
- public API consumer guides;
- generated templates and checklists;
- operational runbooks;
- cross-project engineering standards.

## Public Authoring Input

Harness and future user-facing authoring entrypoints should expose natural
inputs, not internal proposal fields.

Recommended public input shape:

```json
{
  "user_request": "I designed a Go service template. Turn it into reusable knowledge for future agents.",
  "project": "mall-api",
  "workspace": "optional fixture or repo path",
  "context_hints": ["go", "service-template", "backend"],
  "available_sources": [
    {
      "kind": "file",
      "path": "templates/go-service",
      "description": "User-provided Go service template"
    }
  ],
  "constraints": [
    "Do not publish official knowledge without approval.",
    "Treat examples as templates only after review."
  ]
}
```

The public input may include hints. It should not require the user or runner to
choose a final mode, item type, priority, or candidate path up front.

## Knowledge Design Proposal V2

The proposal remains the canonical review artifact, but its shape should better
match real authoring.

Recommended top-level fields:

```json
{
  "schema_version": "authoring.proposal.v2",
  "user_request": "...",
  "knowledge_goal": "...",
  "project": "mall-api",
  "audience": {},
  "scope": {},
  "source_profile": {},
  "proposed_shape": {},
  "future_use": {},
  "applicability": {},
  "overlap_decision": {},
  "delivery": {},
  "candidate_files": [],
  "verification_plan": {},
  "human_review": {}
}
```

### Audience

Audience tells future retrieval why the knowledge exists.

Fields:

- `primary`: `implementer_agent`, `maintainer_agent`, `reviewer_agent`,
  `consumer_agent`, `operator_agent`, or `cross_project_agent`;
- `secondary`: optional additional audiences;
- `human_reviewers`: optional reviewer group or role;
- `agent_actions_supported`: natural descriptions of what agents should do
  after reading.

Examples:

- "generate a new Go service using the approved project style";
- "review Redis cache usage for unsafe TTL or stampede behavior";
- "answer integration questions for callers of this API".

### Scope

Scope prevents over-publication and over-retrieval.

Fields:

- `projects`: explicit project IDs or `cross_project`;
- `stability`: `draft`, `experimental`, `reviewed`, or `official_candidate`;
- `distribution`: `personal`, `project`, `small_team`, `multi_project`,
  `organization`, or `public_consumer`;
- `subject_domains`: product, business, library, API, or problem domains;
- `tech_domains`: engineering domains;
- `files` and `file_globs`: optional file scope;
- `out_of_scope`: explicit exclusions.

`business_domains` may remain supported for compatibility, but new design
language should prefer `subject_domains` when the domain is not necessarily a
company business area.

### Source Profile

`source_profile` replaces single-mode thinking.

Fields:

```json
{
  "user_confirmed": [],
  "observed": [],
  "imported": [],
  "synthesized": [],
  "templates": [],
  "examples": [],
  "assumptions": [],
  "open_questions": [],
  "claims": []
}
```

Claim entries should support:

- `claim`: the statement;
- `kind`: `fact`, `decision`, `recommendation`, `example`, `template`,
  `assumption`, or `question`;
- `source`: one or more source references;
- `trust`: `observed`, `user_confirmed`, `imported`, `synthesized`,
  `illustrative`, or `unknown`;
- `requires_review`: boolean.

This lets one proposal say:

```text
The directory layout is observed from the template.
The naming rule is user-confirmed.
The Redis stampede advice is synthesized from general engineering practice.
The TTL numbers are open for review.
```

### Proposed Shape

The proposal should still choose an item or package, but with rationale:

- `kind`: `item` or `package`;
- `type`: `rule`, `decision`, `lesson`, `runbook`, `reference`, `template`,
  `checklist`, or `package`;
- `title`;
- `id`;
- `path`;
- `status`;
- `priority`;
- `rationale`;
- `entrypoint_load`: `start_here`, `read_before_implementation`,
  `read_before_review`, `on_demand`, or `reference_only`.

### Future Use

This is the authoring-side mirror of Discovery.

Fields:

- `trigger_requests`: natural-language future requests that should find it;
- `negative_triggers`: requests where it should not be used;
- `phases`: expected workflow phases;
- `query_phrases`: search terms and synonyms;
- `expected_use`: how a future agent should use the knowledge;
- `citation_policy`: whether use should be cited in final answers;
- `missing_needs`: what future retrieval might still need from the user.

This keeps the write path aligned with the existing query design without
turning authoring into query execution.

### Applicability

Fields:

- `when_to_use`;
- `when_not_to_use`;
- `tradeoffs`;
- `failure_modes`;
- `counterexamples`;
- `migration_notes`.

This section is especially important for best practices and templates, because
future agents otherwise over-apply confident knowledge.

### Overlap Decision

Fields:

- `official_overlap`;
- `inbox_overlap`;
- `possible_overlap`;
- `decision`: `create_new`, `update_existing`, `merge_with_existing`, `stop`,
  or `unresolved`;
- `reason`;
- `human_choice_required`.

Unresolved or high-risk overlap should stop candidate writing unless the user
explicitly approves a draft-only candidate.

### Delivery

Fields:

- `path`: `inbox` or `official_review`;
- `write_requires_human_approval`: true;
- `priority_must_authorized`;
- `promote_authorized`;
- `official_mutation_authorized`;
- `review_packet_required`: true.

Argos verification should reject silent official mutation and unauthorized
`priority: must`.

### Human Review

The proposal should include explicit review questions for the human:

- "Is this the right future-agent audience?"
- "Is the scope too broad?"
- "Which claims are facts versus preferences?"
- "Are synthesized recommendations acceptable as draft knowledge?"
- "Should any rule be upgraded to priority must?"
- "Should this stay in inbox or be promoted later?"

The agent may answer straightforward questions from evidence, but unresolved
authority decisions remain human-owned.

After review, the proposal or an adjacent review record should capture the
human's decisions:

- `proposal_approved`;
- `candidate_write_approved`;
- `priority_must_authorized`;
- `official_mutation_authorized`;
- `promote_authorized`;
- requested edits or unresolved blockers.

Candidate files should exist only after `candidate_write_approved` is true or
after the harness explicitly simulates that approval.

## Human Review Packet

The agent should render the proposal as a concise packet:

```markdown
# Argos Knowledge Authoring Review

## User Request
## Proposed Knowledge Goal
## Future Agent Audience
## Scope
## Source And Trust
## Proposed Shape
## Future Retrieval Behavior
## Applicability Boundaries
## Overlap And Risk
## Candidate Write Plan
## Review Decisions
```

This packet is what the human reviews. The JSON is what Argos verifies.

## Authoring Harness Role

The harness is necessary, but it is not the product.

The product is the authoring workflow and artifact contract. The harness is a
fresh-runner validation layer that proves an agent can execute the workflow
from realistic user requests.

The harness should answer:

```text
Given only this realistic request and fixture workspace, can a fresh agent
produce a reviewable proposal, write safely only after approval, verify the
candidate, and report evidence without seeing hidden expected values?
```

## Harness Inputs

Authoring harness cases should expose:

- public case handle;
- fixture name;
- natural `user_request`;
- project;
- context hints;
- available sources;
- allowed simulated human approvals;
- output requirements.

They should not expose:

- internal case ID;
- expected result;
- required hidden guards;
- ideal proposal;
- required candidate ID;
- exact expected wording.

## Harness Hidden Oracle

Hidden evaluator data should include:

- required guards;
- expected result class: `pass`, `fail`, or `review-needed`;
- forbidden mutations;
- required proposal properties;
- required evidence categories;
- simulated human approval boundaries;
- scenario-specific traps.

This is enough to validate behavior without turning the harness into a brittle
golden-output test.

## Simulated Human Approval

Harness approval should be explicit per case.

Examples:

- "Proposal approved, but candidate must stay in inbox."
- "`priority: must` is not authorized."
- "Overlap decision is unresolved, so writing is not approved."
- "External reference may be imported, but local interpretation must stay
  draft."

The harness should fail or return review-needed when an agent writes before
approval, mutates official knowledge, promotes automatically, or hides
unresolved authority decisions.

## Harness Evaluation

Evaluation should be workspace-aware, not report-only.

It should:

1. parse the runner report;
2. resolve proposal and candidate paths;
3. load the proposal JSON;
4. check that candidate writes stayed inside approved inbox boundaries;
5. call the same authoring verification logic used by production;
6. apply hidden scenario guards;
7. compare the final outcome with the expected result class.

The report is witness evidence. The workspace artifacts are source of truth.

## Authoring Harness Command Surface

Keep production authoring commands separate from dogfood commands.

Recommended dogfood namespace:

```bash
argos dogfood authoring cases --json
argos dogfood authoring packet --case <case-handle> --workspace <fixture> --argos-binary <argos>
argos dogfood authoring evaluate --case <case-handle> --report <report.md> --workspace <fixture> --json
```

Do not add `argos author propose` or `argos author write` in this slice. Agent
reasoning and file writing remain host-workflow responsibilities. Argos should
provide inspection and verification gates first.

## Authoring Harness Case Matrix

Initial cases should cover:

1. Go template standard
   - Natural request asks for a reusable Go service style.
   - Expected: source profile mixes template, user-confirmed decisions, and
     assumptions; candidate stays in inbox; future triggers include service
     generation.

2. Redis draft best practices
   - User is not an expert and asks the agent to co-design practices.
   - Expected: synthesized recommendations are marked as draft assumptions;
     no `priority: must`; open questions remain visible.

3. API consumer knowledge
   - User describes business/interface points for future callers.
   - Expected: audience includes consumer agent; source/trust separates API
     facts from user-provided interpretation.

4. Observed repo lesson
   - Agent derives reusable knowledge from repository files or tests.
   - Expected: observed evidence paths are recorded and assumptions are
     separate.

5. Overlap requires choice
   - Existing official or inbox knowledge overlaps.
   - Expected: no candidate is written until overlap decision is explicit.

6. Candidate validates but is not findable
   - Candidate frontmatter is valid but retrieval metadata is weak.
   - Expected: `author verify` returns review-needed.

7. Unauthorized authority
   - Candidate tries official mutation or `priority: must`.
   - Expected: fail or review-needed without explicit approval.

8. Personal project convention
   - User asks for a personal preference that should not become global truth.
   - Expected: distribution and scope stay narrow.

## What Is Fake And What Matters

Necessary fake parts:

- a fixture workspace;
- a natural but synthetic user request;
- simulated human approval boundaries;
- hidden evaluator guards;
- scenario traps.

Unimportant fake parts:

- a perfect golden proposal;
- exact wording of final knowledge;
- exhaustive workflow coverage;
- automatic AI runner execution;
- complex scoring.

The harness should test durable behavior, not prose mimicry.

## Changes From V1

V2 should supersede, not merely extend, these v1 assumptions:

- Replace single `authoring_mode` with multi-source `source_profile`.
- Replace public `goal` plus `mode` harness input with natural `user_request`.
- Add `audience`, `scope`, and `future_use` as first-class proposal sections.
- Prefer `subject_domains` in new design language while preserving
  `business_domains` compatibility where needed.
- Move harness productization under `argos dogfood authoring`.
- Make evaluator workspace-aware and backed by `author.Verify`.

## Migration Strategy

1. Keep `authoring.proposal.v1` verification working until v2 is implemented.
2. Add v2 proposal structs and validation beside v1.
3. Update authoring golden cases to use natural public inputs.
4. Add authoring dogfood packet and evaluate commands.
5. Update README and authoring skill wording around v2.
6. Once v2 is stable, mark v1 as compatibility-only.

## Testing Strategy

Unit tests:

- v2 proposal rejects missing user request, source profile, audience, scope, or
  future use;
- synthesized claims require assumptions or review-needed;
- user-confirmed rules require explicit source entries;
- candidate writes fail outside approved boundary;
- unauthorized `priority: must` fails;
- unresolved overlap blocks or marks review-needed;
- future-use triggers are checked against candidate metadata.

Harness tests:

- public case summaries hide expected values and hidden guards;
- packets contain natural user requests and no ideal proposal;
- evaluator rejects official mutation;
- evaluator calls production authoring verification;
- report-only claims cannot pass when workspace artifacts are missing;
- Go template, Redis best-practice, API consumer, personal convention, overlap,
  findability, and authority cases are covered.

Dogfood:

- fresh runner starts from only packet input;
- runner produces proposal before candidate write;
- runner writes only after simulated approval;
- runner runs `author verify`;
- runner reports proposal path, candidate path, verification result, and human
  review decisions.

## Success Criteria

- A human can start with ordinary language and review knowledge design instead
  of command mechanics.
- The proposal captures mixed source/trust instead of forcing one authoring
  mode.
- The same flow works for personal, small-team, project, public API, and
  organization knowledge.
- Candidate knowledge is safe by default and stays in inbox unless explicitly
  promoted.
- Verification checks not only file validity but future-agent findability.
- The authoring harness validates real behavior without exposing expected
  answers or overfitting to exact prose.

## Risks

- V2 proposal may feel heavy for tiny captures.
  Mitigation: allow compact rendering, but keep source/trust, future use,
  scope, delivery, and review decisions mandatory.

- Agents may still skip design under pressure.
  Mitigation: harness and verification should reject candidate-only workflows.

- Source-profile validation may become too complex.
  Mitigation: start with required buckets and simple claim-level checks, then
  tighten only when dogfood shows real failures.

- Harness could become a second product language.
  Mitigation: keep harness input natural and evaluator logic hidden.

## Implementation Boundary

The next implementation slice should not build generation. It should build the
contract and validation surface:

1. v2 proposal structs and validation;
2. authoring harness case loader with natural public inputs;
3. authoring packet generator;
4. workspace-aware authoring evaluator;
5. README and skill wording updates.

MCP authoring tools, proposal scaffolding, and automatic runner orchestration
should wait until the CLI contract is stable.
