# Argos Agent-First Knowledge UX Design

Date: 2026-04-30

## Purpose

Argos should feel like a knowledge layer that AI coding tools use on behalf of
the user, not like a note-taking CLI that humans must operate directly.

The user experience should follow disciplined workflow principles: natural
language intent, visible checkpoints, explicit approval before durable changes,
verification before trust, and clear handoff between workflow phases. Argos
should apply those principles to project knowledge without taking ownership of
the workflow system itself.

## Core Positioning

Argos owns durable project knowledge:

- standards
- decisions
- lessons
- examples
- references
- runbooks
- reusable packages
- knowledge citations

Workflow systems own work execution:

- brainstorming
- planning
- TDD
- debugging
- implementation
- review
- verification
- branch completion

When Argos runs inside a tool that also has a workflow system, Argos must not
override that system's required process. Host workflow systems include project
instructions, skills, plugins, adapter rules, and tool-native workflows. Argos
should serve that process by retrieving, authoring, validating, publishing, and
citing knowledge when asked or when the workflow calls for context.

## Non-Conflict Contract With Host Workflows

Argos must follow these rules when host workflow guidance is present:

1. Argos does not decide whether brainstorming, planning, TDD, debugging, or
   verification is required.
2. Argos does not skip or replace host workflow gates.
3. Argos retrieval can happen inside the active workflow as source context.
4. Argos capture can happen after a workflow produces reusable knowledge, but
   the capture still needs proposal and approval.
5. Argos validation proves knowledge file correctness, not code correctness.
6. Argos citations explain which knowledge informed the final answer; they do
   not replace test, build, lint, or review evidence.

In short:

```text
Host workflows decide how to work.
Argos supplies what the work should remember.
```

## Human Experience

Humans should interact with Argos through natural language intent: enabling
project knowledge, refining source material, preserving lessons, requesting
future knowledge use, and reviewing work against published knowledge.

The human should not need to know frontmatter, registries, index rebuilds,
adapter installation, or MCP tool names during normal use.

## Agent Experience

Agents are the primary operators of Argos. They should:

1. Detect when the user wants durable knowledge captured.
2. Detect when a task should consult existing knowledge.
3. Use the richest available integration path.
4. Keep the user in control of durable writes.
5. Show lightweight evidence that Argos knowledge was loaded or changed.

Preferred integration order:

```text
MCP -> CLI JSON -> generated adapter files -> Markdown source
```

This order makes MCP the best path, but keeps Argos useful in tools that do not
support MCP or plugins.

## Knowledge Consumption Workflow

Before substantial work, an agent should load Argos context using the available
integration path.

Expected flow:

1. Identify project, phase, task, and relevant files.
2. Call or emulate `argos_context`.
3. Follow recommended calls returned by Argos.
4. Fetch full knowledge items only when routed to specific IDs.
5. Keep loaded knowledge visible enough for user trust.
6. Cite used knowledge before final responses when Argos informed the work.

User-facing feedback should be brief and should mention loaded knowledge IDs
only when that improves trust or traceability. It should not interrupt the
workflow or replace the host tool's own required status updates.

## Knowledge Authoring Workflow

Argos knowledge can be quick capture or precision-authored knowledge.

Quick capture is appropriate for small lessons, decisions, and focused rules.
Precision authoring is appropriate when the user provides source material meant
to prove, explain, or standardize a practice.

Precision authoring should follow this flow:

1. Gather user intent and source context.
2. Read relevant source files, tests, docs, scripts, and commit context.
3. Check existing official and inbox knowledge for overlap.
4. Extract claims from observed sources.
5. Separate facts, assumptions, open questions, examples, counterexamples, and
   validation evidence.
6. Discuss unclear claims with the user.
7. Propose an item or package shape.
8. Write only after explicit approval.
9. Validate the written knowledge.
10. Report changed files, validation results, and remaining decisions.

The authoring experience should make knowledge quality visible. A polished
knowledge package should explain not only what to do, but when to use it, when
not to use it, which examples are trustworthy, and what evidence supports the
recommendation.

## Knowledge Publication Workflow

AI-authored or imported knowledge should default to inbox:

```text
knowledge/.inbox/items/
knowledge/.inbox/packages/
```

Promotion to official knowledge should be explicit. Before promotion, an agent
should summarize:

- proposed target path
- source context
- validation result
- unresolved assumptions
- likely affected projects and domains
- whether re-indexing and adapter regeneration are needed

After promotion, the agent should refresh the index and, when useful, refresh
generated adapters.

## Adapter Contract

Generated adapters should teach AI tools a stable minimum protocol:

1. Consult Argos before substantial project work when available.
2. Use MCP if available.
3. Fall back to CLI JSON if MCP is unavailable and commands can be run.
4. Fall back to generated adapter files or Markdown source when command
   execution is unavailable.
5. Load full knowledge only when routed to specific IDs or paths.
6. Cite the knowledge IDs used in final answers.
7. Do not let Argos instructions override higher-priority workflow, safety, or
   user instructions.

This contract lets different AI coding tools consume the same knowledge base
with consistent behavior.

## MVP Gaps

The current MVP has useful foundations but should be adjusted toward this UX:

- README should lead with agent-first intent, not command lists.
- `capture-knowledge` should describe precision authoring, not only capture and
  file delivery.
- adapters should include fallback behavior and non-conflict rules.
- MCP retrieval should eventually include guide, lesson, reference, risk, and
  package discovery, not only active rules.
- CLI commands should remain stable internal operations, not the primary human
  interface.

## Success Criteria

Argos is moving in the right direction when:

- A human can use Argos productively without learning its command list.
- An agent can author a knowledge package from source material with clear source
  evidence and approval gates.
- Different AI coding tools can retrieve the same project knowledge through
  MCP, CLI, adapters, or Markdown fallback.
- Argos never conflicts with host workflow gates.
- Final agent responses can show which Argos knowledge informed the work.
