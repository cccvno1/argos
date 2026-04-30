---
name: capture-knowledge
description: Use when the user asks to remember, capture, save, document, preserve, or turn reusable project knowledge, standards, API contracts, templates, examples, practices, decisions, or lessons into Argos knowledge; also use before suggesting durable capture of reusable context.
---

# Capture Knowledge

## Overview

Use this skill to turn natural user intent into deliberate Argos knowledge
capture. The user should not need to know Argos paths or commands. Your job is
to gather context, check for overlap, propose the capture, ask for the user's
decisions, then use existing Argos files and CLI commands after approval.

Argos is an agent-operated knowledge layer. Humans express knowledge intent in
natural language; the agent operates Argos in the background.

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

Before writing files, you must:

1. Check existing Argos knowledge.
2. Present a concrete proposal.
3. Ask the user to choose create, update, or stop when overlap exists.
4. Ask the user to choose a delivery path.
5. Get explicit approval for the proposal and delivery path.

Never run `argos promote` automatically. Never execute scripts from a knowledge
package unless the validation plan names them and the user confirms execution.
Never set `priority: must` unless the user explicitly authorizes it.
Never claim captured knowledge is official while it is still in inbox.

## Language

Match the user's language for human-facing writing: proposals, questions,
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

### 1. Confirm Capture Intent

Use the skill immediately when the user explicitly asks to capture, preserve,
standardize, or make project knowledge reusable. This applies across languages;
the trigger is durable intent, not a specific phrase.

When you notice reusable knowledge but the user did not ask to capture it, ask
for permission in the user's language before starting this workflow.

If the user declines, stop the capture workflow.

### 2. Gather Context

Gather only the context needed to make a proposal:

- user-provided facts and intent from the conversation
- relevant repository files, docs, tests, or examples
- existing Argos knowledge that might overlap
- current git state when a write may happen

Keep facts and assumptions separate. Do not turn an inference into a fact.

If the user provides source material intended to establish a practice, treat the
capture as precision authoring rather than a quick note.
Read the material closely enough to explain what it proves and what it does not
prove.

For precision authoring, classify source material as:

- `user-provided`: stated directly by the user
- `observed`: visible in repository files, tests, examples, docs, or command
  output
- `inferred`: a reasonable conclusion that still depends on interpretation
- `unverified`: plausible but not yet proven by the available material

Do not present inferred or unverified material as fact.

### 3. Check Existing Knowledge

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
- related inbox candidates found
- related index results found
- check could not be completed, with the reason

If overlap exists, ask whether the user wants to create new knowledge, update
existing knowledge, or stop the capture.

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

For the first version of this skill, fully execute package capture. For item
capture, propose the item shape and use the repository's existing manual Argos
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

### 6. Present The Proposal

For package capture, present a proposal with these sections:

```text
Title
Why This Is Durable Knowledge
Why This Should Be A Package
Scope
Entrypoint
Proposed Structure
Omitted Structure
Existing Knowledge Check
Source Context
Evidence And Trust
Applicability Boundaries
Counterexamples Or Trade-Offs
Delivery Path
Validation Plan
Open Questions
```

The `Entrypoint` section must include:

```text
title
proposed ID
proposed path
status
priority
tags
projects
tech domains
business domains
```

Default new package metadata:

```yaml
status: draft
priority: should
```

The `Source Context` section must group claims as:

```text
User-Provided
Observed From Repository
Inferred
Existing Knowledge
Assumptions
```

Any assumption that affects correctness or trust must also appear in `Open
Questions`.

The `Evidence And Trust` section must name the files, examples, tests, command
outputs, or user statements that support the proposed knowledge. If evidence is
missing, say so.

The `Applicability Boundaries` section must explain when the knowledge should be
used and when it should not be used.

The `Counterexamples Or Trade-Offs` section may be brief, but it must not be
omitted for precision-authored packages.

The `Omitted Structure` section must name optional directories that will not be
created and explain why.

### 7. Require A Delivery Path

Present the user with the two supported delivery paths and wait for a choice.

Inbox candidate:

```text
Write under knowledge/.inbox/packages/, validate the candidate, and leave it for
later review or promotion.
```

PR-style change:

```text
Write under knowledge/packages/ on a review branch or the user's chosen current
branch, validate the changed path, and commit only if the user confirms.
```

Do not infer the delivery path from context. The user owns that choice.

### 8. Write Only After Approval

After proposal approval and delivery-path selection, write files only inside the
chosen boundary:

```text
knowledge/.inbox/packages/
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

Every file under `examples/` must have a declared trust level in the proposal
and in the package's `Load On Demand` guidance:

- `illustrative`: explains the idea and should not be copied into production as
  is.
- `tested`: has been compiled, linted, run, or otherwise verified as described.
- `template`: intended to be copied or adapted by future work.

### 9. Validate

After writing package files, run:

```bash
argos validate --path TARGET_PATH
```

If the local binary is not installed but the repo can run it, use:

```bash
go run ./cmd/argos validate --path TARGET_PATH
```

If validation fails because of protocol issues, fix the written files and run
validation again. If validation fails because of an unresolved product or
knowledge decision, stop and ask the user.

### 10. Publish Only After Review

If the user asks to publish or submit a candidate to official knowledge, first
summarize:

- candidate path
- proposed official target path
- source context
- validation result
- unresolved assumptions
- affected projects and domains
- whether `argos index` and `argos install-adapters` should run afterward

Ask for explicit approval before running:

```bash
argos promote --path TARGET_PATH
```

After promotion, run `argos index` unless the user asks not to. Run
`argos install-adapters` when project adapter instructions should be refreshed.

### 11. Report The Result

End with:

- changed files
- validation commands and outcomes
- whether the package is inbox candidate or PR-style official change
- loaded or cited Argos knowledge IDs, when relevant
- any remaining decisions

Do not claim the knowledge is official when it is still in inbox. Do not promote
or commit unless the user explicitly asked for that step.
