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
  only to retrieve, author, validate, or cite knowledge.

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
JSON or Markdown source, read full knowledge only when selected by Argos guidance, and cite Argos
knowledge IDs that informed final responses.

Before substantial work, an agent should read relevant Argos context and
standards. Before final answers that relied on Argos knowledge, it should cite
the knowledge IDs it used.

## Discovery

Discovery is the agent-facing knowledge navigation layer. It helps agents
inventory available knowledge, find relevant knowledge for current work, and
avoid Argos-backed claims when no strong match exists.

The default discovery path is local and lightweight: SQLite metadata, file
scope matching, tags, package entrypoints, and FTS5 full-text search. Embedding
and vector search are optional future enhancements; `argos knowledge find`
works without Ollama, model downloads, GPU hardware, or a background service.

Use:

```bash
argos knowledge find --json --project <project> --phase <phase> --task "<task>" --query "<query>"
argos knowledge list --json --project <project> --domain <domain>
argos knowledge read --json <id>
argos knowledge cite --json <id>...
```

`argos knowledge list` returns inventory and orientation.
`argos knowledge find` returns ranked knowledge results, `why_matched`, `support`,
`usage`, `search_status`, `missing_needs`, and `next_steps`. Find and list do
not return full Markdown bodies. Read selected full items with
`argos knowledge read`; cite used IDs with `argos knowledge cite`.

Support states:

- `strong`: read recommended high-priority IDs before work.
- `partial`: useful knowledge exists, but gaps remain.
- `weak`: skim summaries or inspect the list; do not treat results as
  Argos-backed guidance.
- `none`: proceed without Argos-specific claims and do not cite Argos knowledge.

### Discovery Validation

Discovery has a golden validation harness under `testdata/discovery-golden/`.

Run automated validation with:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp ./internal/adapters
```

The golden corpus and `cases.json` verify inventory, strong/partial/weak/none
support, progressive reading, citation guardrails, and entrypoint
consistency.

AI dogfood validation uses productized harness commands plus the review
checklist and report template:

```bash
argos dogfood cases --json
argos dogfood packet --case <case-handle> --workspace <fixture> --argos-binary <argos>
argos dogfood evaluate --case <case-handle> --report <report.md> --json
```

- `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- `docs/superpowers/templates/argos-discovery-dogfood-report.md`

Dogfood runners use fresh minimal context per case. Generate runner packets
instead of hand-copying inputs, and do not give runner agents expected IDs,
expected support, prior transcripts, or design history. Evaluate saved reports
separately against `testdata/discovery-golden/cases.json`.

## Agent Knowledge Authoring

Argos write-side UX is agent-operated. Humans describe the engineering
knowledge they want future agents to have, then review a Knowledge Design
Proposal and verification packet. Agents run the commands.

Single knowledge items live under `knowledge/items/`.

Structured knowledge packages live under `knowledge/packages/` and use
`KNOWLEDGE.md` as their indexed entrypoint. Optional package directories include
`references/`, `examples/`, `checklists/`, `scripts/`, and `assets/`.

Authoring inboxes keep reviewed-but-unpromoted work separate:

- `knowledge/.inbox/proposals/` stores Knowledge Design Proposals.
- `knowledge/.inbox/items/` stores item candidates.
- `knowledge/.inbox/packages/` stores package candidates.

The durable flow is:

1. The human describes reusable engineering knowledge, source material, a
   decision, or a lesson.
2. The agent inspects existing knowledge and authoring policy:

   ```bash
   argos author inspect --json \
     --project mall-api \
     --goal "create product-list cache engineering knowledge" \
     --mode synthesized \
     --future-task "implement product list cache" \
     --phase implementation \
     --files internal/catalog/products.go
   ```

3. The agent turns that inspection output into a Knowledge Design Proposal under
   `knowledge/.inbox/proposals/`.
4. The human reviews the proposal and decides whether the agent should write an
   inbox candidate.
5. After approval, the agent writes candidate knowledge under
   `knowledge/.inbox/items/` or `knowledge/.inbox/packages/`.
6. The agent verifies the candidate against the approved proposal:

   ```bash
   argos author verify --json \
     --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json \
     --path knowledge/.inbox/packages/backend/product-list-cache
   ```

7. The human reviews the verification packet. Verification checks the proposal
   contract, candidate validation, policy gates, and discoverability through the
   normal query path. It does not promote candidates or mutate the official
   index.

Official knowledge should not be mutated silently. Inbox candidates are the
default path for AI-authored or imported knowledge. Promotion remains explicit:

```bash
argos promote --path knowledge/.inbox/packages/backend/product-list-cache
argos index
```

## Agent Knowledge Authoring Skill

Argos keeps the installable skill source at `skills/capture-knowledge/` with
frontmatter name `capture-knowledge` for compatibility with existing skill
triggers.

Use this skill when a user asks an agent to author durable reusable project
knowledge. The skill is proposal-first and agent-facing. It guides the agent to:

- understand authoring intent and gather enough context for accurate knowledge
- run or emulate `argos author inspect --json`
- check existing `knowledge/items/`, `knowledge/packages/`, and
  `knowledge/.inbox/` content
- distinguish facts, assumptions, examples, counterexamples, and validation
  evidence
- produce a Knowledge Design Proposal before writing files
- ask whether overlap means create new, update existing, or stop
- ask for an inbox candidate or PR-style delivery path
- write inbox candidates only after approval
- run `argos author verify --json --proposal <proposal.json> --path <candidate>`
- present a review packet without promoting automatically

Human-facing documentation written through the skill should match the user's
language. Argos protocol fields stay stable: frontmatter keys, IDs, paths,
required section headings, commands, filenames, and technical identifiers are
not translated.

## MCP

Run the local MCP server over stdio:

```bash
argos mcp
```

The server supports tool discovery with `tools/list`. The shared-knowledge
`tools/call` entries are:

- `argos_find_knowledge`: returns ranked knowledge results, support, explanations,
  and next steps without full bodies. Arguments: `project`, `phase`, `task`,
  `query`, `files`, `types`, `tags`, `domains`, `status`,
  `include_deprecated`, `limit`.
- `argos_list_knowledge`: returns project/domain knowledge inventory without full bodies.
  Arguments: `project`, `domain`, `types`, `include_deprecated`.
- `argos_read_knowledge`: fetches one indexed knowledge item including its full
  body. Arguments: `id`.
- `argos_cite_knowledge`: returns citation metadata for indexed knowledge items and
  reports missing ids. Arguments: `ids`.

Additional workflow shortcuts include `argos_context` and `argos_standards`.

Run `argos index` before calling index-backed tools:
`argos_standards`, `argos_find_knowledge`, `argos_list_knowledge`,
`argos_read_knowledge`, and `argos_cite_knowledge`.

## Agent/Internal Commands

These commands are stable operations for agents and automation. Humans can run
them directly when debugging Argos, but they are not the primary user
experience.

```bash
argos init
argos validate
argos validate --inbox
argos validate --path <path>
argos author inspect --json --project <project> --goal <goal>
argos author verify --json --proposal <proposal.json> --path <candidate>
argos promote --path <candidate>
argos index
argos install-adapters
argos context --json --project <project>
argos knowledge find --json --project <project> --task <task>
argos knowledge list --json --project <project>
argos knowledge read --json <id>
argos knowledge cite --json <id>...
argos mcp
```
