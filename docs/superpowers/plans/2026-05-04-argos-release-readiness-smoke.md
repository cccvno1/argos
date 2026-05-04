# Argos Release Readiness Smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Validate the first-release write flow with real CLI scenarios and record the result.

**Architecture:** Run Argos from a clean temporary workspace using only public CLI commands. Keep fixes limited to product issues exposed by the smoke run: command behavior, user-facing docs, skill wording, and the write dogfood round report.

**Tech Stack:** Go CLI, `go test`, shell smoke scripts, Argos knowledge registry/index files, Markdown report.

---

### Task 1: Run Three Real CLI Smoke Scenarios

**Files:**
- Modify only if failures require it: `internal/cli/cli.go`, `internal/knowledgewrite/*.go`, `README.md`, `skills/capture-knowledge/SKILL.md`
- Create temporary smoke workspace outside the repo, under `/tmp`

- [x] **Step 1: Build the CLI**

Run:

```bash
go build -o /tmp/argos-release-smoke/argos ./cmd/argos
```

Expected: exit 0.

- [x] **Step 2: Create a clean smoke workspace**

Run:

```bash
SMOKE_ROOT=$(mktemp -d /tmp/argos-release-smoke.workspace.XXXXXX)
/tmp/argos-release-smoke/argos init
```

Expected: `knowledge/.inbox/designs`, `knowledge/.inbox/packages`, registry files, and `argos/generated` exist.

- [x] **Step 3: Run Redis best practices flow**

Run the public command sequence:

```bash
/tmp/argos-release-smoke/argos knowledge design --json --project mall-api --intent "Create Redis cache best practices for future backend agents."
```

Then write a reviewed `knowledge.design.v1` design and draft package, run:

```bash
/tmp/argos-release-smoke/argos knowledge check --json --design <design.json> --draft <draft-path>
/tmp/argos-release-smoke/argos knowledge publish --design <design.json> --path <draft-path>
/tmp/argos-release-smoke/argos index
/tmp/argos-release-smoke/argos knowledge find --json --project mall-api --query "redis cache best practices"
```

Expected: check result `pass`; publish succeeds; find output includes `package:mall-api.redis-cache.v1`.

- [x] **Step 4: Run Go template flow**

Use the same command sequence for:

```text
I designed a Go service template. Turn it into reusable knowledge so future agents write Go services in this style.
```

Expected: check result `pass`; publish succeeds; find output includes `package:mall-api.go-service-template.v1`.

- [x] **Step 5: Run business interface flow**

Use the same command sequence for:

```text
This project is called by many people. Preserve the business points and interfaces so future agents can help other developers use it.
```

Expected: check result `pass`; publish succeeds; find output includes `package:mall-api.consumer-api.v1`.

### Task 2: Record Release Readiness Result

**Files:**
- Modify: `docs/superpowers/reports/2026-05-03-argos-write-dogfood-round-0.md`

- [x] **Step 1: Update report status**

Set status based on actual smoke results:

```markdown
Status: pass
```

or:

```markdown
Status: partial
```

Include scenario table with command outcome, check result, publish result, and findback result.

- [x] **Step 2: Record any fixes**

If code/docs were changed during the smoke run, add a short `Fixes Made` section listing changed files and reason.

### Task 3: Final Verification

**Files:**
- No new files unless failures require fixes.

- [x] **Step 1: Run full tests**

Run:

```bash
go test ./... -count=1
```

Expected: all packages pass.

- [x] **Step 2: Run live old-vocabulary scan**

Run:

```bash
rg -n "knowledge/\\.inbox/proposals|verify the draft|authoring_packet|proposal_scaffold|source_profile|proposed_shape|overlap_decision|verification_plan|human_review|artifact_state|author inspect|author verify|authoring\\.proposal\\.v2|argos author|dogfood authoring|authoring-golden|argos-authoring-dogfood|\\bpromote\\b|Source Profile|Proposed Shape|Candidate Files|Verification Plan|Human Review Decisions|proposal artifact|canonical JSON proposal" --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**' --glob '!docs/superpowers/reports/**' --glob '!.worktrees/**'
```

Expected: no output.

- [x] **Step 3: Commit**

Commit the report and any minimal fixes:

```bash
git add docs/superpowers/reports/2026-05-03-argos-write-dogfood-round-0.md
git commit -m "test: record write release smoke"
```
