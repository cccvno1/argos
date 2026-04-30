# Argos

Argos is a local-first knowledge substrate for AI coding workflows.

Argos is not meant to be a human-facing note app or a CLI that every user has
to learn. The preferred experience is conversational: a person works with an AI
agent to author, refine, validate, publish, retrieve, and cite durable project
knowledge. Argos provides the file protocol, validation, index, adapters, and
MCP tools that make that experience stable across AI coding tools.

## Human Experience

Humans should be able to express knowledge intent in natural language: enable
Argos for a project, refine source material into durable knowledge, preserve a
lesson, ask future agents to consult existing knowledge, or check work against
published knowledge.

The agent performs the Argos operations under the hood: reading source context,
checking existing knowledge, proposing a knowledge shape, writing inbox
candidates after approval, validating files, promoting reviewed knowledge, and
refreshing the local index.

## Relationship To Workflow Systems

Argos complements host workflow systems: project instructions, skills, plugins,
adapter rules, and tool-native workflows. It must not replace or override their
control flow.

- Host workflow systems decide how work proceeds: brainstorming, planning, TDD,
  debugging, review, verification, and branch completion.
- Argos provides the project knowledge those workflows should remember:
  standards, decisions, lessons, examples, runbooks, references, and packages.
- When both are available, workflow instructions stay in charge and call Argos
  only to retrieve, capture, validate, or cite knowledge.

## Agent Experience

Argos supports several integration paths so different AI coding tools can use
the same knowledge base:

- MCP tools for dynamic retrieval.
- Static adapters for tools that read project instruction files.
- CLI JSON and validation commands for agents that can run local commands.
- Markdown source files as the durable system of record.

Agents should prefer the richest available path:

```text
MCP -> CLI JSON -> generated adapter files -> Markdown source
```

Generated adapters define a minimum contract for tools that read project
instruction files: preserve host workflow control, prefer MCP, fall back to CLI
JSON or Markdown source, load full knowledge only when routed, and cite Argos
knowledge IDs that informed final responses.

Before substantial work, an agent should load relevant Argos context and
standards. Before final answers that relied on Argos knowledge, it should cite
the knowledge IDs it used.

## Discovery

Discovery is the agent-facing knowledge navigation layer. It helps agents
inventory available knowledge, route current work to relevant knowledge, and
avoid Argos-backed claims when no strong match exists.

The default discovery path is local and lightweight: SQLite metadata, file
scope matching, tags, package entrypoints, and FTS5 full-text search. Embedding
and vector search are optional future enhancements; `argos discover` works
without Ollama, model downloads, GPU hardware, or a background service.

Use:

```bash
argos discover --json --project <project> --phase <phase> --task "<task>" --query "<query>"
argos map --json --project <project> --domain <domain>
```

`argos map` returns inventory and orientation. `argos discover` returns ranked
routes, `why_matched`, `coverage`, and `next_calls`. Neither command returns
full Markdown bodies. Load selected full items with `get_knowledge_item` and
cite used IDs with `cite_knowledge`.

Coverage states:

- `strong`: load recommended high-priority IDs before work.
- `partial`: useful knowledge exists, but gaps remain.
- `weak`: skim summaries or inspect the map; do not treat results as authority.
- `none`: proceed without Argos-specific claims and do not cite Argos knowledge.

## Knowledge Authoring

Single knowledge items live under `knowledge/items/`.

Structured knowledge packages live under `knowledge/packages/` and use
`KNOWLEDGE.md` as their indexed entrypoint. Optional package directories include
`references/`, `examples/`, `checklists/`, `scripts/`, and `assets/`.

Package candidates are created under `knowledge/.inbox/packages/` and reviewed
before promotion.

High-quality knowledge is expected to be refined. A typical authoring workflow
is:

1. The user gives the agent source material, a decision, or a lesson.
2. The agent reads the relevant source context and checks existing Argos
   knowledge.
3. The agent separates observed facts, user-provided intent, assumptions,
   open questions, examples, counterexamples, and verification evidence.
4. The agent proposes an item or package shape and asks for approval.
5. The agent writes an inbox candidate only after approval.
6. The agent runs Argos validation and reports the result.
7. The user reviews and decides whether to promote the candidate.

Official knowledge should not be mutated silently. Inbox candidates are the
default path for AI-authored or imported knowledge.

## Capture Knowledge Skill

Argos includes an installable `capture-knowledge` skill source at
`skills/capture-knowledge/`.

Use this skill when a user asks an agent to remember, preserve, document, or
turn reusable project knowledge into Argos knowledge. The skill is
proposal-first and agent-facing. It guides the agent to:

- gather enough context to author accurate knowledge
- check existing `knowledge/items/`, `knowledge/packages/`, and
  `knowledge/.inbox/` content
- distinguish facts, assumptions, examples, counterexamples, and validation
  evidence
- propose the knowledge shape before writing files
- ask whether overlap means create new, update existing, or stop
- ask for an inbox candidate or PR-style delivery path
- write package files only after approval
- run `argos validate --path TARGET_PATH`

Human-facing documentation written through the skill should match the user's
language. Argos protocol fields stay stable: frontmatter keys, IDs, paths,
required section headings, commands, filenames, and technical identifiers are
not translated.

## MCP

Run the local MCP server over stdio:

```bash
argos mcp
```

The server supports tool discovery with `tools/list` and implements these
`tools/call` entries:

- `argos_context`: returns workflow context and recommended next calls.
  Arguments: `project`, `phase`, `task`, `files`.
- `argos_standards`: returns active standards for project work from the local
  index. Arguments: `project`, `task_type`, `files`, `limit`.
- `argos_discover`: returns ranked knowledge routes, coverage, explanations,
  and next calls without full bodies. Arguments: `project`, `phase`, `task`,
  `query`, `files`, `types`, `tags`, `domains`, `status`, `include_inbox`,
  `include_deprecated`, `limit`.
- `argos_map`: returns project/domain knowledge inventory without full bodies.
  Arguments: `project`, `domain`, `types`, `include_inbox`,
  `include_deprecated`.
- `get_knowledge_item`: fetches one indexed knowledge item including its full
  body. Arguments: `id`.
- `cite_knowledge`: returns citation metadata for indexed knowledge items and
  reports missing ids. Arguments: `ids`.

Run `argos index` before calling index-backed tools:
`argos_standards`, `argos_discover`, `argos_map`, `get_knowledge_item`, and
`cite_knowledge`.

## Agent/Internal Commands

These commands are stable operations for agents and automation. Humans can run
them directly when debugging Argos, but they are not the primary user
experience.

```bash
argos init
argos validate
argos validate --inbox
argos validate --path <path>
argos promote --path <candidate>
argos index
argos install-adapters
argos context --json --project <project>
argos discover --json --project <project> --task <task>
argos map --json --project <project>
argos mcp
```
