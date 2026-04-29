# Argos Capture Knowledge Skill Design

Date: 2026-04-29

## Purpose

Argos needs an agent-facing skill that turns natural user intent into deliberate
knowledge capture. A user should be able to say things like "remember this",
"turn this into a reusable standard", or "capture this API convention" while
they are already working. The agent then uses Argos conventions and CLI commands
under the hood, without asking the user to memorize Argos file layouts or
commands.

The skill is named `capture-knowledge`.

This skill does not replace Argos validation, indexing, promotion, or review. It
teaches an agent when and how to propose new or updated knowledge, how to surface
existing overlap, and how to create files only after the user has made the
necessary decisions.

## Scope

This design covers the behavior of the `capture-knowledge` skill:

- trigger conditions
- item-versus-package judgment
- existing-knowledge checks
- proposal format
- delivery-path selection
- source-context handling
- package file planning
- validation behavior

This design does not implement the skill yet. It also does not add new Argos CLI
commands. The skill should use the existing Argos file protocol and CLI
operations where possible.

## Design Principles

Argos should give users a strong solution protocol without deciding their
knowledge model for them.

Strict constraints:

- The agent must not silently mutate official trusted knowledge.
- The agent must check for existing related knowledge before proposing a write.
- The agent must present a proposal before creating or changing files.
- The agent must get an explicit delivery-path choice before writing files.
- The agent must separate facts from assumptions.
- The agent must validate written knowledge with Argos.
- The agent must not promote candidates automatically.

Flexible choices:

- Users choose whether overlapping knowledge means create new, update existing,
  or stop.
- Users choose whether to use an inbox candidate or a PR-style change.
- Users choose package paths and tags.
- Users can keep packages small or add supporting files when the knowledge needs
  progressive disclosure.
- Users can model project information, API contracts, standards, templates,
  best practices, operational notes, and other knowledge without Argos forcing a
  fixed category tree.

In short:

```text
Skill: guides the capture workflow
Proposal: makes decisions visible
Argos files: store the durable knowledge
Argos CLI: validates and moves knowledge safely
User: owns the final choices
```

## Trigger Model

The skill should trigger when the user explicitly wants durable knowledge
captured or when the agent sees a clear chance to preserve reusable knowledge and
asks for permission.

Explicit user triggers include:

- "记下来"
- "把这个沉淀成知识库"
- "创建一个知识包"
- "capture this"
- "turn this into a standard"
- "make this reusable for future agents"

Agent-suggested triggers must be phrased as a question, not an action:

```text
This looks reusable. Would you like me to run the capture-knowledge workflow and
propose an Argos knowledge entry for it?
```

The skill should not trigger for ordinary explanations, one-off debugging
updates, temporary notes, or speculative ideas unless the user asks to preserve
them.

## Item Versus Package

The skill should decide whether to propose a knowledge item or a knowledge
package based on the shape of the knowledge.

Prefer a knowledge item when the content is:

- a single rule
- a small decision
- a short lesson
- a compact runbook
- a focused reference note

Prefer a knowledge package when the content needs progressive disclosure:

- project information with multiple entry points
- API contracts or examples
- database, Redis, queue, deployment, or testing practices
- coding standards with rationale and examples
- reusable templates
- review checklists
- scripts or generated artifacts
- any knowledge that benefits from `references/`, `examples/`, `checklists/`,
  `scripts/`, or `assets/`

If the choice is unclear, the agent should ask one question about intended reuse
or structure. It should not ask the user to choose "item versus package" unless
that is the simplest way to resolve the ambiguity.

The first implementation should focus on package capture because knowledge
packages are the richer workflow and the place where proposal discipline matters
most. Item capture remains part of the skill's decision model, but full item
file creation can be implemented as a follow-up after package capture works.

## Existing Knowledge Check

Before proposing any write, the skill must check for related knowledge. The
check should search:

- `knowledge/items/`
- `knowledge/packages/`
- `knowledge/.inbox/`
- the Argos index or MCP tools when available

The check is advisory. It surfaces possible overlap but does not decide what the
user should do.

The proposal must include an `Existing Knowledge Check` section with one of
these outcomes:

- no related knowledge found
- related official knowledge found
- related inbox candidates found
- related index results found
- check could not be completed, with the reason

When overlap exists, the agent must ask the user to choose the next action before
writing:

```text
I found related knowledge. Do you want to create a new entry, update an existing
entry, or stop this capture?
```

The agent must not silently merge, replace, or update existing knowledge.

## Proposal Protocol

The proposal is the central control point. It should be specific enough for the
user to understand what will happen, but not so ceremonial that simple captures
become painful.

Every package proposal must include:

- `Title`
- `Why This Is Durable Knowledge`
- `Why This Should Be A Package`
- `Scope`
- `Entrypoint`
- `Proposed Structure`
- `Omitted Structure`
- `Existing Knowledge Check`
- `Source Context`
- `Delivery Path`
- `Validation Plan`
- `Open Questions`

The `Entrypoint` section should include:

- title
- proposed ID
- proposed path
- status
- priority
- tags
- projects
- tech domains
- business domains

The default metadata for new captured package knowledge is:

```yaml
status: draft
priority: should
```

The agent must not default to `priority: must`. That priority requires explicit
user authorization because it changes how future agents should treat the
knowledge.

The `Omitted Structure` section is required for packages. It should name optional
directories that are intentionally not created, such as `scripts/` or `assets/`,
and briefly explain why they are unnecessary for this package.

## Delivery Paths

The skill supports two delivery paths. The agent must present them and wait for
the user's explicit choice before writing files.

### Inbox Candidate

Use this when the user wants a local candidate that can be reviewed and promoted
later.

Flow:

```text
proposal approved
-> write package under knowledge/.inbox/packages/
-> run argos validate --path <candidate>
-> fix protocol issues if needed
-> leave candidate in inbox
```

This path is useful for new knowledge, experiments, imports, and agent-generated
material. It keeps candidate knowledge separate from official knowledge.

### PR-Style Change

Use this when the user wants review through normal source-control workflow.

Flow:

```text
proposal approved
-> create or use a review branch if needed
-> write or update official knowledge files
-> run argos validate --path <changed-path>
-> commit the change if the user confirms a commit
-> user opens or continues a PR
```

This path is the preferred way to modify existing official knowledge because the
branch and PR provide the review container.

The skill must not create branches, commits, or PRs without user confirmation.
If the user is already on a suitable branch and asks to use it, the skill may
write there after confirming the proposal and delivery path.

## Source Context Protocol

The proposal and written knowledge must distinguish the source of claims.

The proposal's `Source Context` section should use these groups:

- `User-Provided`: facts or intent explicitly stated by the user.
- `Observed From Repository`: facts the agent inspected in files, tests, docs,
  or code.
- `Existing Knowledge`: related Argos knowledge found during the check.
- `Assumptions`: inferences the agent is making.

Assumptions must not be written as facts. Any assumption that affects correctness
or trust must also appear in `Open Questions`.

Examples of correctness-affecting assumptions:

- a code example is production-ready
- an API contract is authoritative
- a package applies to all projects
- a standard should have `priority: must`
- a script is safe to execute

## Package File Planning

The package entrypoint is always `KNOWLEDGE.md`. Optional directories are created
only when useful:

```text
references/
examples/
checklists/
scripts/
assets/
```

For each planned file, the proposal should state why it exists and when an agent
should load it.

Every file under `examples/` must have a declared trust level in the proposal and
in the package's `Load On Demand` guidance:

- `illustrative`: explains the idea and should not be copied into production as
  is.
- `tested`: has been compiled, linted, run, or otherwise verified as described.
- `template`: intended to be copied or adapted by future work.

Scripts may be created as knowledge assets, but they are not executed
automatically. A script can be executed only when the validation plan says so and
the user confirms that execution is allowed.

## Write And Validation Behavior

After the user approves the proposal and selects a delivery path, the agent may
write files.

For inbox candidates, the target path must be under:

```text
knowledge/.inbox/packages/
```

For PR-style package captures, the target path must be under:

```text
knowledge/packages/
```

For future PR-style item captures, the target path must be under
`knowledge/items/`.

After writing, the agent must run:

```text
argos validate --path <target-path>
```

If validation fails because of protocol issues, the agent should fix the written
files and validate again. If validation fails because of an unresolved product or
knowledge decision, the agent should stop and ask the user.

The agent should not run `argos promote` automatically. Promotion is a separate
user decision.

## Error Handling

If Argos is unavailable, the skill should still produce a proposal and explain
that validation is blocked. It may write files only if the user explicitly
accepts that validation cannot be run.

If the repository has unrelated local changes, the skill should avoid touching
them. If related files have concurrent changes, the agent should inspect them and
work with the current content rather than reverting anything.

If the requested target path already exists, the agent must stop before writing
and ask whether the user wants to update that knowledge, choose another path, or
stop.

## Out Of Scope

This skill does not:

- install remote knowledge libraries
- define source priority between personal, organization, company, and public
  knowledge
- automatically resolve conflicts
- automatically decide whether overlap means update or create new
- automatically promote inbox candidates
- automatically open pull requests
- execute scripts without confirmation
- require users to adopt a fixed domain taxonomy beyond Argos metadata

Those decisions can be designed later as separate workflows.

## Example Interaction

User:

```text
把我们刚才讨论的 Redis key 命名规范沉淀成知识库
```

Agent:

```text
I will run the capture-knowledge workflow first. I will check existing Argos
knowledge, then propose what to create before writing files.
```

Agent checks existing knowledge, then proposes:

```text
Title: Redis Key Naming Standard
Why This Is Durable Knowledge:
The naming rules affect future implementation and review across Redis-backed
features.

Why This Should Be A Package:
The standard needs a short entrypoint, rationale, examples, and a review
checklist.

Entrypoint:
- ID: package:backend.redis.key-naming.v1
- Path: backend/redis/key-naming
- Status: draft
- Priority: should
- Tags: redis, naming, cache

Proposed Structure:
- KNOWLEDGE.md
- references/rationale.md
- examples/go/cache_keys.go (trust: illustrative)
- checklists/review.md

Omitted Structure:
- scripts/ because no automated verification is defined
- assets/ because the package has no binary or media assets

Existing Knowledge Check:
Related official package found: package:backend.redis.best-practices.v1

Delivery Path:
Please choose inbox candidate or PR-style change before I write files.
```

The agent waits. If the user chooses inbox candidate, the agent writes under
`knowledge/.inbox/packages/backend/redis/key-naming/` and validates that path.

## Open Questions For Implementation Planning

- Should the first implementation create a complete local `capture-knowledge`
  skill folder, or should it first add repo documentation and examples for the
  workflow?
- Should package capture be implemented before item capture, with item capture
  documented as a smaller follow-up?
- Should the skill include helper scripts for proposal generation, or should it
  remain pure instructions in `SKILL.md` for the first version?
