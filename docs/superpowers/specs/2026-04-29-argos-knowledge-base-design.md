# Argos Knowledge Base Design

Date: 2026-04-29

## Purpose

Argos is a local-first knowledge substrate for AI development workflows. It helps AI coding tools, agent frameworks, skills collections, and enterprise workflows reuse project knowledge, domain knowledge, best practices, operational assets, and hard-won lessons during design, implementation, debugging, review, and operations.

Argos is not tied to one AI tool. It exposes knowledge through standard files, CLI commands, MCP tools/resources/prompts, and workflow-level contracts so different tools can consume the same knowledge progressively.

## Product Positioning

Argos serves two categories of consumers:

- AI coding tools: Codex, Claude Code, Cursor, Gemini CLI, Aider, Windsurf, and similar tools.
- Agent workflows: superpowers, skills collections, open-source workflow packs, enterprise agent systems, and custom development processes.

The core product must work without any one skill system. Skills or workflow integrations are optional accelerators that teach a specific agent framework when and how to call Argos.

## Core Principles

- Markdown is the source of truth.
- YAML frontmatter is the only supported metadata write format.
- Local indexes are derived caches and can always be rebuilt.
- Static agent files provide a small knowledge map, not full knowledge content.
- MCP provides dynamic, progressive knowledge access.
- Workflow-level contracts let higher-level agent processes use Argos without understanding its internal schema.
- AI-generated or imported knowledge must enter an inbox and be reviewed before becoming official knowledge.

## Knowledge Model

The smallest unit is a knowledge item. Each item is a Markdown file with YAML frontmatter.

Example:

```yaml
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api, admin-console]
status: active
priority: must
applies_to:
  languages: [typescript]
  frameworks: [nestjs]
  files: ["src/auth/**"]
updated_at: 2026-04-29
```

Supported knowledge types:

- `rule`: mandatory or strongly preferred constraints that AI should follow.
- `guide`: step-by-step development guidance for common tasks.
- `runbook`: operational procedures for deployment, rollback, debugging, and on-call work.
- `decision`: architecture or product decisions and their rationale.
- `lesson`: pitfalls, incidents, debugging notes, and experience summaries.
- `reference`: background information, glossary entries, domain explanations, or stable context.

Domains are split into two controlled dimensions:

- `tech_domains`: frontend, backend, database, DevOps, testing, security, AI engineering, and similar technical areas.
- `business_domains`: account, order, payment, risk, content, analytics, and similar business areas.

Projects are an applicability dimension, not the root hierarchy. One item can apply to multiple projects, technical domains, and business domains.

## Repository Layout

Recommended first-version layout:

```text
knowledge/
  domains.yaml
  projects.yaml
  types.yaml
  .inbox/
  items/
    backend/
      auth/
        jwt-refresh-token.md
    devops/
      deployment/
        rollback-runbook.md

argos/
  index.db
  generated/
    AGENTS.md
    CLAUDE.md
    GEMINI.md
    cursor-rules/
```

`knowledge/**/*.md` files are the official knowledge source. `domains.yaml`, `projects.yaml`, and `types.yaml` are controlled registries used to prevent taxonomy drift.

`argos/index.db` is generated locally and should not be committed. It stores derived metadata, summaries, relationships, full-text search data, and optional embeddings. The index can be deleted and rebuilt at any time.

Generated adapter files are produced by Argos. Users should not manually edit them as knowledge sources.

## Write Flow

Human-authored knowledge:

```text
argos new
-> choose type, domains, projects, scope
-> generate Markdown template
-> edit content
-> argos validate
-> argos index
```

AI-generated or imported knowledge:

```text
candidate content
-> knowledge/.inbox/*.md
-> argos validate
-> human review
-> move into knowledge/items
-> argos index
```

Knowledge must not move directly from chat transcripts, imports, or AI summaries into `knowledge/items` without review.

## CLI

The MVP CLI includes:

```text
argos init
argos new
argos validate
argos index
argos install-adapters
argos mcp
```

Responsibilities:

- `init`: create the initial folder structure and registry files.
- `new`: create a knowledge item template.
- `validate`: check metadata schema, registry references, unique IDs, status, priority, and scope.
- `index`: rebuild the local index from Markdown sources.
- `install-adapters`: generate AI tool adapter files for registered projects.
- `mcp`: start the MCP server.

## Static AI Adapters

Argos generates tool-native entry files:

```text
AGENTS.md
CLAUDE.md
GEMINI.md
.cursor/rules/argos.mdc
```

These files should contain:

- project ID
- technical domains
- business domains
- brief project knowledge map
- instructions for using Argos progressively
- available MCP server name and high-level tools

They should not contain full knowledge bodies. Their job is to tell the AI tool where to start and how to ask for more.

Example instruction:

```md
Before substantial design, implementation, debugging, review, or operations work:
1. Call `argos_context` with the project, phase, task, and files.
2. Follow the recommended next calls.
3. Load required rules before editing code.
4. Load guides, lessons, runbooks, or full items only when routed to them.
5. Cite knowledge IDs used in final responses.
```

## Progressive Knowledge Disclosure

AI should not search or load the entire knowledge base. Argos exposes knowledge in stages:

```text
brief
-> route
-> summaries
-> specific full item by ID
-> citations
```

Rules:

- Default responses return summaries, not full bodies.
- Full knowledge content requires a second request with a specific ID.
- Each result includes `why_matched`, `scope`, `priority`, and `status`.
- Deprecated knowledge is excluded by default.
- Conflicting rules are returned explicitly in `conflicts`.
- Result counts and output size are limited by default.

This design follows the direction used by current AI development tools: small static context at startup, then dynamic MCP access when the task needs more.

## MCP Interface

Argos exposes MCP tools, resources, and prompts.

Tools:

```text
route_knowledge(project, task, files?)
get_relevant_rules(project, task, files?, limit?)
get_guides(project, task_type, domains?, limit?)
get_lessons(project, topic_or_error, files?, limit?)
get_runbook(project, service, operation, environment?)
get_knowledge_item(id)
cite_knowledge(ids)
```

Resources:

```text
argos://project/{project}/brief
argos://project/{project}/rules
argos://domain/{domain}/map
argos://knowledge/{id}
```

Prompts:

```text
prepare_feature_work
debug_with_lessons
review_with_standards
prepare_deployment
```

MCP tools are for agent-driven dynamic retrieval. MCP resources are for `@`-style context references where supported. MCP prompts are reusable workflow templates.

## Workflow Integration Contract

Argos also exposes higher-level workflow-friendly calls. These are intended for superpowers-like skill systems, enterprise agent frameworks, and open-source workflow packs.

These calls are public integration contracts, not only internal concepts. The MVP should expose them as MCP tools and should also make them available through CLI JSON output for workflows that do not use MCP.

```text
argos_context(task_context)
```

Used at the start of any substantial workflow. Returns project brief, relevant domains, and recommended next calls.

```text
argos_requirements(project, phase, domains?, task?)
```

Used during design and planning. Returns business constraints, technical constraints, historical decisions, and domain knowledge that must be considered.

```text
argos_standards(project, task_type?, files?)
```

Used during implementation and review. Returns required rules, coding standards, testing expectations, and security constraints.

```text
argos_risks(project, task?, files?, symptoms?)
```

Used during planning, debugging, risky changes, and pre-deployment checks. Returns lessons, incident notes, conflict warnings, and sensitive-module risks.

```text
argos_operations(project, service?, operation?, environment?)
```

Used for deployment, rollback, database operations, on-call work, and operational troubleshooting. Returns runbooks and environment-specific notes.

These workflow calls internally use the same progressive disclosure model. They return workflow-ready summaries and recommended next calls instead of raw search results.

Example response:

```json
{
  "phase": "planning",
  "project": "mall-api",
  "must_consider": [
    {
      "id": "decision:auth.session-boundary.v1",
      "type": "decision",
      "summary": "User session ownership belongs to account service.",
      "why_matched": "task mentions refresh token and account domain",
      "priority": "must"
    }
  ],
  "recommended_next_calls": [
    {
      "tool": "argos_standards",
      "reason": "implementation will touch src/auth/**"
    }
  ],
  "conflicts": []
}
```

## Index Store

SQLite is the default MVP index backend, but it is not the system of record and is not a deployment requirement.

The index stores:

- item ID
- title
- type
- status
- priority
- domains
- projects
- applicability scope
- summaries
- references
- full-text search data
- optional embeddings

Future index backends can include Postgres with pgvector, Meilisearch, Typesense, or OpenSearch. The source format and public contracts should remain stable when the index backend changes.

## MVP Scope

MVP includes:

- Markdown knowledge items with YAML frontmatter.
- Controlled registries for domains, projects, and types.
- CLI for initialization, creation, validation, indexing, adapter generation, and MCP startup.
- Local rebuildable SQLite index.
- Static adapters for AGENTS.md, CLAUDE.md, GEMINI.md, and Cursor rules.
- MCP server with tools, resources, and prompts.
- Workflow integration contract.
- Inbox review flow for AI-generated or imported knowledge.

MVP does not include:

- multi-tenant enterprise permissions
- web management UI
- online collaborative editing
- automatic full synchronization from Notion, Feishu, Confluence, or similar systems
- complex graph visualization
- cloud database deployment
- automatic unreviewed writes into official knowledge

## Validation And Testing

Testing should cover:

- frontmatter schema validation
- registry reference validation
- unique knowledge IDs
- valid status, priority, and type values
- index rebuild from source files
- generated adapter content
- MCP tool response shape and result limits
- deprecated items excluded by default
- conflict detection for overlapping mandatory rules
- workflow contract responses containing `why_matched` and recommended next calls

## Evolution Path

Phase 2:

- lightweight web UI for inbox review and knowledge browsing
- knowledge usage analytics
- improved citation tracking

Phase 3:

- pluggable index backends
- Postgres and vector search support
- search quality evaluation

Phase 4:

- organization-level knowledge packages
- permissions, audit logs, and policy management
- managed remote MCP deployment

Phase 5:

- semi-automatic extraction from sessions, PRs, incident reviews, and code reviews
- stronger drift detection between codebase state and knowledge items
