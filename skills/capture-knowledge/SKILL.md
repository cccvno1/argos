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

## Workflow

### 1. Confirm Capture Intent

Use the skill immediately when the user explicitly asks to capture durable
knowledge, such as:

- "记下来"
- "把这个沉淀成知识库"
- "创建一个知识包"
- "capture this"
- "turn this into a standard"
- "make this reusable for future agents"

When you notice reusable knowledge but the user did not ask to capture it, ask:

```text
This looks reusable. Would you like me to run the capture-knowledge workflow and
propose an Argos knowledge entry for it?
```

If the user declines, stop the capture workflow.

### 2. Gather Context

Gather only the context needed to make a proposal:

- user-provided facts and intent from the conversation
- relevant repository files, docs, tests, or examples
- existing Argos knowledge that might overlap
- current git state when a write may happen

Keep facts and assumptions separate. Do not turn an inference into a fact.

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

If overlap exists, ask the user:

```text
I found related knowledge. Do you want to create a new entry, update an existing
entry, or stop this capture?
```

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

### 5. Present The Proposal

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
Existing Knowledge
Assumptions
```

Any assumption that affects correctness or trust must also appear in `Open
Questions`.

The `Omitted Structure` section must name optional directories that will not be
created and explain why.

### 6. Require A Delivery Path

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

### 7. Write Only After Approval

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

### 8. Validate

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

### 9. Report The Result

End with:

- changed files
- validation commands and outcomes
- whether the package is inbox candidate or PR-style official change
- any remaining decisions

Do not claim the knowledge is official when it is still in inbox. Do not promote
or commit unless the user explicitly asked for that step.
