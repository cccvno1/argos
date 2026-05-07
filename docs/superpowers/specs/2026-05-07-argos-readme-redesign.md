# Argos Minimal README Redesign

Date: 2026-05-07

## Goal

Rewrite `README.md` as a minimal English product README. It should help a
human understand how to start using Argos with an AI coding agent, without
turning the README into a CLI manual or agent implementation guide.

## Design Principles

- Argos is agent-operated. Humans should not need to learn the CLI for normal
  use.
- README should explain the mental model, not every feature.
- Commands and protocol details belong in skills, generated adapter guidance,
  and maintainer/reference docs.
- The style should be closer to Superpowers and OpenSpec than to a CLI command
  reference.
- The README language should be English.

## README Structure

```markdown
# Argos

Argos gives AI coding agents durable project memory.

## Quick Start

Install Argos, initialize your repository, and install generated agent guidance.
After that, keep working normally with your AI agent.

## How You Use It

You do not usually call Argos directly.

Your agent uses Argos automatically when project knowledge is relevant. You only
express intent:

- Remember this for future agents.
- Preserve this decision.
- Use project knowledge before changing this.
- Audit Argos knowledge before release.

## What Argos Does

Argos stores project knowledge as local repository files, builds a local index,
and lets future agents find, review, publish, and cite that knowledge.

## How It Fits

Workflow systems like Superpowers and OpenSpec decide how work proceeds. Argos
supplies the durable knowledge those workflows can use.

## Development

Build, test, and release commands for maintainers.
```

## Scope

This redesign changes only `README.md` unless a verification step exposes a
broken command or stale reference elsewhere.

The rewrite should remove the current long agent reference from README. It
should not create new public reference documents unless explicitly requested.

## Acceptance Criteria

- README is short enough to read quickly.
- README is written in English.
- README describes Argos as an agent-operated project memory layer.
- README does not include CLI parameter tables.
- README does not describe MCP, provenance, or storage contracts in detail.
- README includes only the maintainer commands needed for build/test.
