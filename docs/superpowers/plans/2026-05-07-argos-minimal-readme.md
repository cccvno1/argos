# Argos Minimal README Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current command-heavy README with a short English product README that explains how humans use Argos through AI agents.

**Architecture:** Keep README focused on product positioning, quick start, human intent, fit with workflow systems, and maintainer commands. Do not move the removed command reference into new public docs in this pass.

**Tech Stack:** Markdown, Go test verification.

---

## File Structure

- Modify: `README.md` — replace with the minimal English README.

---

### Task 1: Replace README With Minimal Product Guide

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace README content**

Replace `README.md` with:

```markdown
# Argos

Argos gives AI coding agents durable project memory.

## Quick Start

Install Argos, initialize your repository, and install generated agent guidance.
After that, keep working normally with your AI agent.

For normal use, you should not need to learn the Argos CLI. Your agent uses
Argos in the background when project knowledge is relevant.

## How You Use It

You express intent in natural language:

- Remember this for future agents.
- Preserve this decision.
- Use project knowledge before changing this.
- Audit Argos knowledge before release.

The agent decides when to query, read, write, validate, publish, or cite Argos
knowledge. When writing or publishing durable knowledge, the agent should ask
for your approval before changing trusted knowledge.

## What Argos Does

Argos stores project knowledge as local repository files, builds a local index,
and lets future agents find, review, publish, and cite that knowledge.

Use Argos for durable knowledge such as:

- project standards
- decisions
- lessons
- examples
- runbooks
- references
- knowledge packages

## How It Fits

Workflow systems like Superpowers and OpenSpec decide how work proceeds: design,
planning, testing, debugging, review, and branch completion.

Argos supplies the durable project knowledge those workflows can use.

## Development

Build the CLI:

```bash
go build ./cmd/argos
```

Run the test suite:

```bash
go test ./...
```

## License

MIT
```

- [ ] **Step 2: Verify README is concise**

Run:

```bash
wc -l README.md
```

Expected: fewer than 80 lines.

- [ ] **Step 3: Run tests**

Run:

```bash
go test ./... -count=1
```

Expected: all packages pass.

- [ ] **Step 4: Commit**

Run:

```bash
git add README.md docs/superpowers/plans/2026-05-07-argos-minimal-readme.md
git commit -m "docs: simplify README for agent-operated usage"
```
