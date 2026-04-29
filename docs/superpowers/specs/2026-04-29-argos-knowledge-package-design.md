# Argos Knowledge Package Design

Date: 2026-04-29

## Purpose

Argos needs a first-class way to represent deliberate knowledge assets, not only single Markdown knowledge items. Some knowledge is small enough to be a rule, lesson, decision, or runbook. Other knowledge is a structured package: project information, API contracts, database practices, Redis conventions, templates, examples, checklists, scripts, and supporting material.

This design adds a knowledge package protocol that works like progressive disclosure in agent skills. A package has a small indexed entrypoint and optional supporting files that agents load only when the current task needs them.

The goal is not to classify every kind of knowledge for users. The goal is to provide a reliable structure users and agents can use for many kinds of knowledge.

## Design Principles

Argos should strictly constrain the parts machines must understand and leave semantic organization flexible for users.

Strict protocol:

- Packages have one entrypoint named `KNOWLEDGE.md`.
- Only `KNOWLEDGE.md` has Argos frontmatter and an Argos ID.
- Package assets stay inside the package directory.
- Candidate packages are created in inbox and reviewed before becoming official knowledge.
- Package creation is proposal-first.
- Validation supports exact path-level checks.
- Promotion is handled by a safe CLI operation.

Flexible semantics:

- Users choose package paths.
- Users choose tags.
- Users choose how to describe purpose and use cases.
- Argos does not require fixed `kind`, `shape`, `source`, or `layer` fields.
- Optional directories are created only when the package needs them.

In short:

```text
Path: human organization
Frontmatter: machine metadata
Body sections: agent protocol
Assets: progressive disclosure material
```

## Repository Layout

Official packages live under:

```text
knowledge/packages/
```

Candidate packages live under:

```text
knowledge/.inbox/packages/
```

Package paths are free-form multi-level paths. They are for human organization and do not have to match project IDs, technical domains, or business domains.

Examples:

```text
knowledge/packages/backend/redis/best-practices/
knowledge/packages/backend/auth/jwt-refresh-token/
knowledge/packages/mall-api/api/public-contract/
knowledge/packages/mall-api/domain/order-lifecycle/
knowledge/packages/templates/go/service/
knowledge/packages/team/code-review/process/
```

Path safety is strict:

- Paths must stay under `knowledge/.inbox/packages/` or `knowledge/packages/`.
- Paths must be relative, not absolute.
- Paths must not contain `..`.
- Paths must not contain empty segments.
- Package directories must not be hidden directories.
- Package path segments should use lowercase letters, numbers, and hyphens.
- Creating or promoting a package must not overwrite existing content.

## Package Structure

A package must contain:

```text
KNOWLEDGE.md
```

A package may contain:

```text
references/
examples/
checklists/
scripts/
assets/
```

Only create optional directories when they carry useful knowledge. A package should not contain empty structure for the sake of looking complete.

Directory meanings:

- `references/`: deeper explanations, background, trade-offs, and long-form guidance.
- `examples/`: copyable code, configuration, API examples, or template fragments.
- `checklists/`: review, release, production-readiness, or operational checklists.
- `scripts/`: explicit verification, generation, or inspection scripts.
- `assets/`: supporting files that do not fit the other directories.

MVP packages use one indexed file only:

```text
KNOWLEDGE.md
```

Markdown files inside `references/` or `checklists/` do not have Argos frontmatter in the MVP and are not indexed independently.

## KNOWLEDGE.md Protocol

`KNOWLEDGE.md` is the package entrypoint. It should be short enough to load early and specific enough to tell an agent when to load deeper files.

Frontmatter uses the existing knowledge metadata model, with `type: package`.

Example:

```yaml
---
id: package:backend.redis.best-practices.v1
title: Redis Best Practices
type: package
tech_domains: [backend, database]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-04-29
---
```

Required frontmatter:

- `id`
- `title`
- `type`
- `status`
- `priority`
- `updated_at`

Registry-backed frontmatter remains registry-backed:

- `type`
- `tech_domains`
- `business_domains`
- `projects`

`tags` are intentionally free-form and not registry-backed.

The body must include these sections:

```md
## Purpose
## When To Use
## Start Here
## Load On Demand
```

`## Verification` is optional.

The body-section requirement is a medium-strength validation rule. Argos validates that the sections exist, but the MVP does not parse `Load On Demand` links or require exact wording inside the sections.

Example:

```md
## Purpose

Document Redis usage guidance for services that use Redis for caching,
coordination, or rate limiting.

## When To Use

Use this package when designing, implementing, reviewing, or debugging Redis
usage.

## Start Here

- Prefer explicit key namespaces.
- Set TTL intentionally.
- Do not use Redis as the source of truth unless the design says so.

## Load On Demand

- `references/key-design.md`: when designing Redis key structure.
- `references/cache-invalidation.md`: when changing invalidation behavior.
- `examples/go/redis_client.go`: when writing Go Redis client setup.
- `checklists/production-readiness.md`: before production deployment.

## Verification

Run scripts only when this package explicitly says they apply to the current
change. Scripts are never executed automatically by Argos.
```

## Creation Workflow

Knowledge package creation is deliberate. Agents must not silently create packages.

The flow:

```text
user intent
-> agent gathers context
-> agent proposes package structure
-> user confirms proposal
-> agent writes candidate package to knowledge/.inbox/packages/
-> agent runs argos validate --path <candidate>
-> agent fixes protocol issues if validation fails
-> package remains ready for human review
```

The proposal must come before file creation.

Required proposal fields:

```text
Title
Target path
ID
Why this should be a package
Intended use
Frontmatter
Planned files
Omitted directories
Validation plan
Open questions
```

The proposal should explain both what will be created and what will be deliberately omitted. This keeps packages focused and prevents ornamental directory sprawl.

Example:

```text
Title: Redis Best Practices
Target path: knowledge/.inbox/packages/backend/redis/best-practices/
ID: package:backend.redis.best-practices.v1

Why this should be a package:
It needs progressive disclosure: entry guidance, detailed references,
implementation examples, and a production checklist.

Planned files:
- KNOWLEDGE.md
- references/key-design.md
- references/cache-invalidation.md
- examples/go/redis_client.go
- checklists/production-readiness.md

Omitted directories:
- scripts/ because no automated verification is defined yet
- assets/ because examples are plain text/code files
```

## Validation

Argos validation should support three scopes:

```bash
argos validate
argos validate --inbox
argos validate --path <path>
```

`argos validate` validates official knowledge only:

```text
knowledge/items/
knowledge/packages/
```

It should not fail because of half-finished inbox candidates.

`argos validate --inbox` validates inbox candidates:

```text
knowledge/.inbox/items/
knowledge/.inbox/packages/
```

`argos validate --path <path>` validates exactly one item or package path. This is the required validation mode for agent-created packages.

For package paths, validation checks:

- The path is inside an allowed package root.
- `KNOWLEDGE.md` exists.
- `KNOWLEDGE.md` frontmatter is valid.
- `type` is `package`.
- Required frontmatter fields exist.
- Registry-backed fields reference known registry values.
- The body is non-empty.
- The body contains required package protocol sections.
- Optional package directories stay inside the package.

The MVP does not parse or validate every `Load On Demand` path. That can be added later as a stronger validation mode.

## Promotion

Review turns an inbox candidate into official knowledge.

Argos should provide:

```bash
argos promote --path <candidate>
```

For packages:

```text
knowledge/.inbox/packages/backend/redis/best-practices/
-> knowledge/packages/backend/redis/best-practices/
```

For items:

```text
knowledge/.inbox/items/backend/auth-rule.md
-> knowledge/items/backend/auth-rule.md
```

Promotion rules:

- The candidate path must be under an allowed inbox root.
- The candidate must validate before moving.
- The official target is computed by preserving the path under `.inbox`.
- The command must refuse to overwrite an existing official item or package.
- The command validates official knowledge after moving.
- The command reports the promoted path and recommends `argos index`.

`argos promote` should not automatically rebuild the index. Agents may run `argos index` after promotion when the user wants the promoted knowledge to become queryable immediately.

Agent skills can orchestrate promotion, but CLI owns the safe filesystem operation.

## MCP And Query Behavior

Packages participate in query results through their `KNOWLEDGE.md` entrypoint.

Initial MVP behavior:

- Index `KNOWLEDGE.md` as a normal knowledge record with `type: package`.
- Query results can return package summaries.
- `get_knowledge_item(id)` can return the `KNOWLEDGE.md` body.
- Package assets are discovered from `Load On Demand` text and local paths by the agent.

Future MCP extensions may add:

```text
list_package_assets(id)
get_package_asset(id, path)
```

These are not required for the first package implementation.

## Agent Skill Behavior

The user should not need to learn package creation commands. A capture or package-creation skill should translate user intent into CLI-backed operations.

The skill should:

- Read the current development context when relevant.
- Decide whether the knowledge should be an item or package.
- Propose package structure before writing files.
- Ask only for missing decisions that materially affect package quality.
- Use `argos validate --path` after writing a candidate.
- Leave candidates in inbox for review.
- Use `argos promote --path` only after the user confirms review.

The skill should not:

- Create packages silently.
- Promote packages silently.
- Execute package scripts unless the user or package protocol explicitly asks for verification.
- Treat tags or paths as fixed taxonomy.

## Non-Goals

This design does not implement:

- fixed `kind`, `shape`, `source`, or `layer` taxonomies
- source-policy priority systems
- remote package installation
- package-internal indexed Markdown files
- automatic promotion
- automatic index rebuild after promotion
- automatic script execution
- strict parsing of `Load On Demand` paths

These may be added later when real usage shows they are needed.
