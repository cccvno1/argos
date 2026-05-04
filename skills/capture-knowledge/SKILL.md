---
name: capture-knowledge
description: Use when the user asks to remember, capture, save, document, preserve, or turn reusable project knowledge, standards, API contracts, templates, examples, practices, decisions, or lessons into Argos knowledge; also use before suggesting durable capture of reusable context.
---

# Agent Knowledge Authoring

## Overview

Use this skill to turn natural user intent into deliberate Argos engineering
knowledge for future agents. This is not a memo workflow. The user should not
need to know Argos paths or commands. Your job is to help design the knowledge,
inspect existing Argos facts, produce a knowledge design, ask for the user's
decisions, write inbox drafts only after approval, check the draft, and present
review questions.

Argos is an agent-operated knowledge layer. Humans express knowledge intent in
natural language and review knowledge design; the agent operates Argos in the
background.

## Relationship To Workflow Systems

This skill must not conflict with the host tool's workflow system, including
project instructions, skills, plugins, adapter rules, or other workflow
guidance.

If a workflow system requires brainstorming, planning, TDD, systematic
debugging, review, verification, or branch-completion steps, follow that system.
Use Argos only to retrieve, author, validate, publish, or cite durable knowledge
inside the active workflow.

Argos validation proves that knowledge files satisfy the Argos protocol. It does
not replace code tests, linting, builds, review, or host workflow verification
gates.

## Hard Gates

Do not silently mutate official trusted knowledge.

Before writing draft knowledge files, you must:

1. Check existing Argos knowledge.
2. Present a concrete knowledge design.
3. Ask the user to choose create, update, or stop when overlap exists.
4. Ask the user to choose a write boundary.
5. Record explicit approval for the design and write boundary in provenance.

Never run publish automatically. Never execute scripts from a knowledge
package unless the check plan names them and the user confirms execution.
Never set `priority: must` unless the user explicitly authorizes it.
Never claim authored knowledge is official while it is still in inbox.

Treat `sources` as multi-source. Do not force the request into one mode
when it combines user-confirmed standards, observed files, imported references,
templates, examples, synthesized recommendations, assumptions, and open
questions.

Before presenting the design, run or emulate:

```bash
argos knowledge design --json --project mall-api --intent "create product-list cache engineering knowledge"
```

Before designing project-scoped knowledge, run:

```bash
argos project list --json
```

If the target project is missing and the user has provided the needed project
ID, name, path, and domains, register it with:

```bash
argos project add --id <project> --name <name> --path <path> --tech-domain <domain> --business-domain <domain>
```

Do not hand-edit `knowledge/projects.yaml` unless the CLI is unavailable.

Read `write_guidance` before writing files. Treat it as the operational
control guidance for this write attempt:

- follow `next_action` for the next safe step;
- use `design_path` for the design JSON;
- use `draft_path` only as a planned path, not as write approval;
- obey every `stop_conditions` entry;
- use `design_focus` and `source_work` to shape the design summary;
- ask the `review_questions` before draft writing when correctness or
  authorization depends on the answer;
- record design and draft-write approval with `commands.start_provenance`,
  `commands.record_design_decision`, and
  `commands.record_draft_write_decision`;
- run `commands.record_check` after draft files exist;
- record publish approval with `commands.record_publish_decision`, then run
  `commands.verify_provenance` before publish.

After approved draft files are written, run:

```bash
argos provenance record-check --json --provenance <id>
argos provenance record-decision --json --provenance <id> --stage publish --decision approved --decided-by <actor> --role knowledge_owner --source conversation --reason <reason> --recorded-by <agent>
argos provenance verify --json --provenance <id>
```

Design JSON describes intent, scope, sources, draft output, and check plan. It
does not prove approval. Approval and publish evidence must be recorded in Argos
provenance.

## Language

Match the user's language for human-facing writing: designs, questions,
status updates, package explanations, references, checklists, and example
commentary should use the same language the user is using.

Keep Argos protocol tokens unchanged:

- frontmatter keys and registry values
- IDs, paths, filenames, directory names, tags, and command names
- required `KNOWLEDGE.md` section headings such as `Purpose`, `When To Use`,
  `Start Here`, and `Load On Demand`
- code, API names, schema names, and other technical identifiers

If the user mixes languages, use the user's dominant language for prose and keep
technical terms in their original form. Ask only if the desired documentation
language is genuinely unclear.

## Workflow

### 1. Confirm Authoring Intent

Use the skill immediately when the user explicitly asks to capture, preserve,
standardize, or make project knowledge reusable. This applies across languages;
the trigger is durable intent, not a specific phrase.

When you notice reusable knowledge but the user did not ask to capture it, ask
for permission in the user's language before starting this authoring workflow.

If the user declines, stop the authoring workflow.

### 2. Gather Context

Gather only the context needed to make a design:

- user-provided facts and intent from the conversation
- relevant repository files, docs, tests, or examples
- existing Argos knowledge that might overlap
- current git state when a write may happen

Keep facts and assumptions separate. Do not turn an inference into a fact.

If the user provides source material intended to establish a practice, treat the
request as precision authoring rather than a quick note.
Read the material closely enough to explain what it proves and what it does not
prove.

For precision authoring, classify source material as:

- `user-provided`: stated directly by the user
- `observed`: visible in repository files, tests, examples, docs, or command
  output
- `inferred`: a reasonable conclusion that still depends on interpretation
- `unverified`: plausible but not yet proven by the available material

Do not present inferred or unverified material as fact.

### 3. Inspect Existing Knowledge

Before proposing any write, search these locations when they exist:

```text
knowledge/items/
knowledge/packages/
knowledge/.inbox/
```

Use `rg` for text search. Prefer search terms from the user's language, likely
package titles, technology names, project names, API names, and proposed tags.
If an Argos index or MCP tools are available in the current environment, use
them as an additional check.

Classify the result as one of:

- no related knowledge found
- related official knowledge found
- related inbox drafts found
- related index results found
- check could not be completed, with the reason

If overlap exists, ask whether the user wants to create new knowledge, update
existing knowledge, or stop authoring.

Do not merge, replace, or update existing knowledge until the user chooses.

### 4. Choose Item Or Package Shape

Prefer a knowledge item when the durable knowledge is:

- a single rule
- a small decision
- a short lesson
- a compact runbook
- a focused reference note

Prefer a knowledge package when the durable knowledge needs progressive
disclosure:

- project information with multiple entry points
- API contracts or examples
- database, Redis, queue, deployment, or testing practices
- coding standards with rationale and examples
- reusable templates
- review checklists
- scripts or generated artifacts
- supporting files under `references/`, `examples/`, `checklists/`, `scripts/`,
  or `assets/`

For the first version of this skill, fully execute package authoring. For item
authoring, propose the item shape and use the repository's existing manual Argos
workflow unless the user asks for a package instead.

If the shape is unclear, ask one question about intended reuse or structure.

### 5. Refine Precision Knowledge

Use this step when the user asks to turn source material or a prior discussion
into durable knowledge.

Before proposing files, refine the knowledge with the user:

1. State the practice, rule, or pattern the source material appears to teach.
2. Identify the evidence: files, tests, examples, docs, command output, or user
   statements.
3. Name the intended audience and future reuse context.
4. Name applicability boundaries: when to use it and when not to use it.
5. Look for counterexamples, failure modes, or trade-offs.
6. Separate required rules from optional guidance.
7. Identify what should be loaded immediately and what should be load-on-demand.
8. Ask focused questions only where correctness or trust depends on the answer.

For examples, assign a trust level:

- `illustrative`: explains the idea and should not be copied into production as
  is.
- `tested`: has been compiled, linted, run, or otherwise verified as described.
- `template`: intended to be copied or adapted by future work.

If the knowledge is not ready to write, stop with a concise list of open
questions or missing evidence instead of creating weak knowledge.

### 6. Present The Knowledge Design

Present a knowledge design summary with these sections:

```text
User Request
Knowledge Goal
Future Agent Audience
Scope
Sources
Draft Output
Future Use
Applicability Boundaries
Existing Knowledge
Write Boundary
Draft Files
Check Plan
Review Decisions
```

The review text is not the persisted design. When the user approves the design,
write a knowledge design JSON to
`knowledge/.inbox/designs/.../design.json` or the chosen design path before
writing draft knowledge. The JSON describes intent, scope, sources, draft
output, and check plan. It does not prove approval. Approval and publish
evidence must be recorded in Argos provenance. The JSON must use
`schema_version: knowledge.design.v1` and the snake_case fields that
`argos knowledge check` validates:

```text
user_request
knowledge_goal
audience
scope
sources
draft_output
future_use
applicability
existing_knowledge
write_boundary
draft_files
check_plan
review
```

Set `write_boundary.write_requires_review_approval` and
`write_boundary.review_packet_required` to true. Keep the JSON design and the
human-facing design summary synchronized; do not write a Markdown-only design.

The `Sources` section must group claim sources and trust levels as:

```text
User Input
Observed
Imported
AI Suggested
Templates
Examples
Assumptions
Open Questions
Claim-Level Trust
```

Do not collapse sources into one mode. A Redis best-practice draft may combine
AI-suggested recommendations, assumptions, and open questions. A Go template
package may combine user input, observed files, and template evidence.

The `Draft Output` section must include:

```text
kind
type
title
draft ID
draft path
status
priority
rationale
entrypoint load
```

Default new draft metadata:

```yaml
status: draft
priority: should
```

Storage contract:

- inbox drafts under `knowledge/.inbox/items/` and
  `knowledge/.inbox/packages/` must use `status: draft`;
- official knowledge under `knowledge/items/` and `knowledge/packages/` must
  not use `status: draft`;
- use `argos knowledge publish` for the standard transition from inbox draft to
  official knowledge.

The `Future Use` section must name:

```text
trigger requests
negative triggers
phases
query phrases
expected use
citation policy
```

The `Applicability Boundaries` section must explain when the knowledge should be
used, when it should not be used, and the trade-offs.

The `Existing Knowledge` section must name related official or inbox knowledge
and say whether to create new, update existing, merge, stop, or ask for review
choice.

The `Write Boundary` section must state whether the draft stays in inbox or
needs an explicit review path for official writing. `priority: must`, official
writing, and publishing require explicit authorization.

The `Draft Files` section must list each planned file, purpose, and load
behavior. For packages, `KNOWLEDGE.md` is the indexed entrypoint; optional
directories are created only when useful.

The `Review Decisions` section must list design approval, draft write approval,
priority escalation, official writing, publishing, requested edits, and
unresolved blockers.

### 7. Require A Write Boundary

Present the user with the supported write boundaries and wait for a choice.

Inbox draft:

```text
Write under knowledge/.inbox/items/ or knowledge/.inbox/packages/, check the
draft, and leave it for later review or publishing.
```

PR-style change:

```text
Write under knowledge/items/ or knowledge/packages/ on a review branch or the
user's chosen current branch, check the changed path, and commit only if the
user confirms.
```

Do not infer the write boundary from context. The user owns that choice.

### 8. Write Only After Approval

After the design is written, start provenance and record the design and
draft-write decisions:

```bash
argos provenance start --json --design DESIGN_PATH --draft DRAFT_PATH
argos provenance record-decision --json --provenance PROVENANCE_ID --stage design --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
argos provenance record-decision --json --provenance PROVENANCE_ID --stage draft_write --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
```

Only after those decisions are recorded, write files inside the chosen boundary:

```text
knowledge/.inbox/designs/
knowledge/.inbox/items/
knowledge/.inbox/packages/
knowledge/items/
knowledge/packages/
```

If the target path already exists, stop and ask whether to update that knowledge,
choose another path, or stop.

The package entrypoint must be:

```text
KNOWLEDGE.md
```

Optional directories are created only when useful:

```text
references/
examples/
checklists/
scripts/
assets/
```

Every file under `examples/` must have a declared trust level in the design
and in the package's `Load On Demand` guidance:

- `illustrative`: explains the idea and should not be copied into production as
  is.
- `tested`: has been compiled, linted, run, or otherwise verified as described.
- `template`: intended to be copied or adapted by future work.

### 9. Check

After writing draft files, run:

```bash
argos knowledge check --json --design DESIGN_PATH --draft TARGET_PATH
```

If the local binary is not installed but the repo can run it, use:

```bash
go run ./cmd/argos knowledge check --json --design DESIGN_PATH --draft TARGET_PATH
```

If the check fails because of protocol issues, fix the written files and run
the check again. If the check fails because of an unresolved product or
knowledge decision, stop and ask the user.

### 10. Publish Only After Provenance Approval

If the user asks to publish or submit a draft to official knowledge, first
summarize:

- draft path
- proposed official target path
- source context
- check result
- unresolved assumptions
- affected projects and domains
- whether `argos index` and `argos install-adapters` should run afterward

After checking the draft, use this provenance sequence before publishing:

```bash
argos provenance record-check --json --provenance PROVENANCE_ID
argos provenance record-decision --json --provenance PROVENANCE_ID --stage publish --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
argos provenance verify --json --provenance PROVENANCE_ID
```

Before publishing, run `argos provenance status --json --provenance PROVENANCE_ID`.
If the user asks for a broader review queue, run `argos knowledge audit --json`
and summarize categories, findings, and actions. Treat audit/status as evidence
organization only; the user or PR reviewer decides whether to publish or merge.

Publish only after provenance contains the required decision records and verify
passes:

```bash
argos knowledge publish --provenance PROVENANCE_ID
```

After publishing, run `argos index` unless the user asks not to. Run
`argos install-adapters` when project adapter instructions should be refreshed.
Inbox drafts must use `status: draft`; `argos knowledge publish` converts
published official knowledge to `status: active`.

### 11. Report The Result

End with:

- changed files
- check commands and outcomes
- whether the package is an inbox draft or PR-style official change
- loaded or cited Argos knowledge IDs, when relevant
- any remaining decisions

Do not claim the knowledge is official when it is still in inbox. Do not publish
or commit unless the user explicitly asked for that step.
