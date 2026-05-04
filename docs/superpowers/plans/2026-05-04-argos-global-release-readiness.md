# Argos Global Release Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit Argos end-to-end first-release readiness from a clean user workspace, across CLI, docs, MCP, adapters, and agent-facing workflows.

**Architecture:** Use the public binary and temporary workspaces for runtime checks, then record findings in a release-readiness report. Keep product changes minimal and only fix blockers discovered by real commands.

**Tech Stack:** Go CLI, JSON-RPC over stdio for MCP, Markdown reports, shell smoke commands, existing Go tests.

---

### Task 1: Fresh Workspace Smoke

**Files:**
- Create or modify only if needed: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`
- Runtime artifacts: `/tmp/argos-global-readiness`

- [x] **Step 1: Build public binary**

Run:

```bash
rm -rf /tmp/argos-global-readiness
mkdir -p /tmp/argos-global-readiness
go build -o /tmp/argos-global-readiness/argos ./cmd/argos
```

Expected: command exits 0.

- [x] **Step 2: Initialize and validate a clean workspace**

Run from a new temporary directory:

```bash
SMOKE_ROOT=$(mktemp -d /tmp/argos-global-readiness.workspace.XXXXXX)
cd "$SMOKE_ROOT"
/tmp/argos-global-readiness/argos init
/tmp/argos-global-readiness/argos validate
```

Expected: `init` succeeds and `validate` exits 0.

- [x] **Step 3: Verify empty discovery behavior**

Run:

```bash
/tmp/argos-global-readiness/argos index
/tmp/argos-global-readiness/argos knowledge find --json --project mall-api --query "redis cache"
```

Expected: no crash; response uses a non-strong support state and does not claim Argos-backed knowledge.

- [x] **Step 4: Verify write-to-query loop**

Run the public sequence:

```bash
/tmp/argos-global-readiness/argos knowledge design --json --project mall-api --intent "Create Redis cache best practices for future backend agents."
```

Then write a reviewed design and draft package, run:

```bash
/tmp/argos-global-readiness/argos knowledge check --json --design <design.json> --draft <draft-path>
/tmp/argos-global-readiness/argos knowledge publish --design <design.json> --path <draft-path>
/tmp/argos-global-readiness/argos index
/tmp/argos-global-readiness/argos knowledge find --json --project mall-api --query "redis cache best practices"
```

Expected: check result `pass`, publish succeeds, findback includes `package:mall-api.redis-cache.v1`.

### Task 2: CLI And Documentation Consistency

**Files:**
- Inspect: `README.md`, `skills/capture-knowledge/SKILL.md`, `internal/cli/cli.go`
- Create or modify only if needed: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`

- [x] **Step 1: Capture CLI help surface**

Run:

```bash
/tmp/argos-global-readiness/argos
/tmp/argos-global-readiness/argos knowledge design --help
/tmp/argos-global-readiness/argos knowledge check --help
/tmp/argos-global-readiness/argos knowledge publish --help
/tmp/argos-global-readiness/argos knowledge find --help
/tmp/argos-global-readiness/argos dogfood write cases --json
```

Expected: command names and flags match README and skill wording.

- [x] **Step 2: Scan for removed write vocabulary**

Run:

```bash
rg -n "knowledge/\\.inbox/proposals|verify the draft|authoring_packet|proposal_scaffold|source_profile|proposed_shape|overlap_decision|verification_plan|human_review|artifact_state|author inspect|author verify|authoring\\.proposal\\.v2|argos author|dogfood authoring|authoring-golden|argos-authoring-dogfood|\\bpromote\\b|Source Profile|Proposed Shape|Candidate Files|Verification Plan|Human Review Decisions|proposal artifact|canonical JSON proposal" --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**' --glob '!docs/superpowers/reports/**' --glob '!.worktrees/**'
```

Expected: no live-surface matches.

- [x] **Step 3: Record documentation gaps**

Record whether README and capture skill clearly explain:

```text
agent-first purpose
fresh workspace setup
query path
write design/check/publish path
MCP path
adapter fallback path
release-blocking limitations
```

Expected: gaps are classified as blocker, should-fix, or later.

### Task 3: MCP And Agent Integration Smoke

**Files:**
- Inspect: `internal/mcp/server.go`, `internal/mcp/server_test.go`, `internal/adapters/adapters.go`
- Create or modify only if needed: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`

- [x] **Step 1: Verify MCP tool discovery**

Run:

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | /tmp/argos-global-readiness/argos mcp
```

Expected: response includes implemented tool names only, with strict input schemas.

- [x] **Step 2: Verify MCP query tool call**

Run from the smoke workspace after indexing:

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"argos_find_knowledge","arguments":{"project":"mall-api","query":"redis cache best practices"}}}' | /tmp/argos-global-readiness/argos mcp
```

Expected: response returns the same findback knowledge ID as CLI.

- [x] **Step 3: Verify adapter generation**

Run:

```bash
/tmp/argos-global-readiness/argos install-adapters
```

Expected: generated adapters preserve host workflow control, prefer MCP, fall back to CLI JSON or Markdown, and cite Argos IDs.

### Task 4: Release Decision Report

**Files:**
- Create: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`

- [x] **Step 1: Write release-readiness report**

Include:

```markdown
Status: ready | conditional | blocked
Fresh workspace smoke
CLI/docs consistency
MCP/agent integration
Blockers
Should-fix before first release
Later
```

- [x] **Step 2: Run final verification**

Run:

```bash
go test ./... -count=1
git diff --check
```

Expected: both pass.

- [x] **Step 3: Commit**

Run:

```bash
git add docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md docs/superpowers/plans/2026-05-04-argos-global-release-readiness.md
git commit -m "docs: record global release readiness"
```
