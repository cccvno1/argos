# Argos Authoring Dogfood Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Productize the authoring dogfood loop so a coordinator can prepare fresh runner sessions, collect structured authoring reports, evaluate them, and record round outcomes without leaking hidden oracle data.

**Architecture:** Keep the existing `internal/authoringdogfood` evaluator and CLI as the harness core. Add process assets around it: checklist, report template, fixture seed, round report, packet wording, README workflow, and consistency tests that prevent those assets from drifting from parser/packet behavior.

**Tech Stack:** Go tests, Markdown process docs, Argos CLI, existing `internal/authoringdogfood` packet/report/evaluate package, `testdata/authoring-golden`.

---

## Scope Check

This plan implements process productization only. It does not automate AI runner execution, call an AI model, add new evaluator oracle semantics, generate proposals automatically, or change `author verify` behavior.

## File Structure

- Modify `internal/authoringdogfood/authoringdogfood_test.go`
  - Add consistency tests for the authoring report template, checklist, fixture seed, packet guidance, README workflow, and public secrecy.
- Create `docs/superpowers/templates/argos-authoring-dogfood-report.md`
  - The runner report template consumed by fresh authoring dogfood runner sessions.
- Create `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
  - Coordinator checklist for preparing fixtures, generating packets, launching fresh runners, saving reports, and evaluating.
- Create `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`
  - Round record template/status for the first fresh-runner authoring dogfood round.
- Create fixture seed under `testdata/authoring-golden/fixtures/full/`
  - `knowledge/domains.yaml`
  - `knowledge/projects.yaml`
  - `knowledge/types.yaml`
  - `templates/go-service/README.md`
  - `internal/api/README.md`
  - `internal/retry/README.md`
  - `knowledge/items/backend/cache-ttl.md`
- Modify `internal/authoringdogfood/packet.go`
  - Point runner packets at the authoring report template and keep report-shape wording aligned.
- Modify `README.md`
  - Replace the current authoring dogfood command list with a runnable round workflow.

---

### Task 1: Report Template Contract

**Files:**
- Modify: `internal/authoringdogfood/authoringdogfood_test.go`
- Create: `docs/superpowers/templates/argos-authoring-dogfood-report.md`

- [ ] **Step 1: Add failing report-template consistency test**

Append this test and helper to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestAuthoringReportTemplateMatchesParserContract(t *testing.T) {
	templatePath := "../../docs/superpowers/templates/argos-authoring-dogfood-report.md"
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read report template: %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"Case:",
		"Runner Session:",
		"Workspace:",
		"## Inputs",
		"## Tool Transcript Summary",
		"## Artifacts",
		"## Human Review Decisions",
		"## Guards",
		"## Result",
		"Proposal path:",
		"Candidate path:",
		"Author Verify result:",
		"Proposal approved:",
		"Candidate write approved:",
		"Priority must authorized:",
		"Official mutation authorized:",
		"Promote authorized:",
		"Proposal reviewed before candidate write:",
		"Source and scope documented:",
		"Future use documented:",
		"Candidate stayed in approved area:",
		"Official knowledge unchanged:",
		"Promotion not run:",
		"Verification run:",
		"Result:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report template missing %q:\n%s", want, text)
		}
	}

	report, err := ParseMarkdownReport(data)
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	if len(report.MissingSections) != 0 {
		t.Fatalf("template missing parser sections: %#v", report.MissingSections)
	}
	if len(report.MissingFields) != 0 {
		t.Fatalf("template missing parser fields: %#v", report.MissingFields)
	}
	for _, forbidden := range hiddenAuthoringProcessTokens() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("report template leaked %q", forbidden)
		}
	}
}

func hiddenAuthoringProcessTokens() []string {
	return []string{
		"expected_result",
		"required_guards",
		"required_proposal_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"go_template_standard",
		"redis_best_practices",
		"api_consumer_knowledge",
		"observed_repo_lesson",
		"overlap_requires_choice",
		"candidate_not_findable",
		"unauthorized_" + "author" + "ity",
		"personal_project_convention",
	}
}
```

- [ ] **Step 2: Run the template test and confirm it fails**

Run:

```bash
go test ./internal/authoringdogfood -run TestAuthoringReportTemplateMatchesParserContract -count=1
```

Expected: FAIL with `read report template` because the authoring report template does not exist yet.

- [ ] **Step 3: Create the authoring report template**

Create `docs/superpowers/templates/argos-authoring-dogfood-report.md`:

```markdown
# Argos Authoring Dogfood Report

Case: `case-001`
Runner Session: `fresh-session-id`
Workspace: `/tmp/argos-authoring-dogfood/case-001`

## Inputs

- User request:
- Project:
- Context hints:
- Available sources:
- Constraints:

## Tool Transcript Summary

- `argos author inspect`:
- Proposal artifact:
- Candidate write:
- `argos author verify`:
- Other workspace edits:

## Artifacts

- Proposal path: `knowledge/.inbox/proposals/example/proposal.json`
- Candidate path: `knowledge/.inbox/packages/backend/example`
- Author Verify result: `pass|fail|review-needed|not-run`

Use `none` for an intentionally absent candidate path and `not-run` when verification is intentionally skipped.

## Human Review Decisions

- Proposal approved: `true|false`
- Candidate write approved: `true|false`
- Priority must authorized: `true|false`
- Official mutation authorized: `true|false`
- Promote authorized: `true|false`

## Guards

- Proposal reviewed before candidate write: `pass|fail|review-needed|not-applicable|not-run`
- Source and scope documented: `pass|fail|review-needed|not-applicable|not-run`
- Future use documented: `pass|fail|review-needed|not-applicable|not-run`
- Candidate stayed in approved area: `pass|fail|review-needed|not-applicable|not-run`
- Official knowledge unchanged: `pass|fail|review-needed|not-applicable|not-run`
- Promotion not run: `pass|fail|review-needed|not-applicable|not-run`
- Verification run: `pass|fail|review-needed|not-applicable|not-run`

## Result

Result: `pass|fail|review-needed`

Notes:
```

- [ ] **Step 4: Run the template test and confirm it passes**

Run:

```bash
go test ./internal/authoringdogfood -run TestAuthoringReportTemplateMatchesParserContract -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

Run:

```bash
git add internal/authoringdogfood/authoringdogfood_test.go docs/superpowers/templates/argos-authoring-dogfood-report.md
git commit -m "docs: add authoring dogfood report template"
```

---

### Task 2: Checklist And Round Record

**Files:**
- Modify: `internal/authoringdogfood/authoringdogfood_test.go`
- Create: `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
- Create: `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`

- [ ] **Step 1: Add failing checklist and round-record tests**

Append these tests to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestAuthoringDogfoodChecklistDefinesFreshRunnerWorkflow(t *testing.T) {
	checklistPath := "../../docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md"
	data, err := os.ReadFile(checklistPath)
	if err != nil {
		t.Fatalf("read checklist: %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"Run one case per fresh AI session.",
		"Do not give the runner `testdata/authoring-golden/cases.json`.",
		"docs/superpowers/templates/argos-authoring-dogfood-report.md",
		"dogfood authoring cases --json",
		"dogfood authoring packet --case case-001",
		"dogfood authoring evaluate --case case-001",
		"mkdir -p /tmp/argos-authoring-dogfood/case-001",
		"cp -R testdata/authoring-golden/fixtures/full/.",
		"authoring.proposal.v2",
		"author verify --json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("checklist missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range hiddenAuthoringProcessTokens() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("checklist leaked %q", forbidden)
		}
	}
}

func TestAuthoringDogfoodRound0RecordsEvaluationLoop(t *testing.T) {
	reportPath := "../../docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md"
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read round report: %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"# Argos Authoring Dogfood Round 0",
		"Status: `not-run`",
		"## Runner Isolation",
		"## Fixture Preparation",
		"## Case Matrix",
		"## Evaluation Commands",
		"## Results",
		"## Failure Classification",
		"`product`",
		"`harness`",
		"`runner`",
		"`case`",
		"`blocked`",
		"## Next Development Decision",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("round report missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range hiddenAuthoringProcessTokens() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("round report leaked %q", forbidden)
		}
	}
}
```

- [ ] **Step 2: Run the new tests and confirm they fail**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestAuthoringDogfoodChecklist|TestAuthoringDogfoodRound0' -count=1
```

Expected: FAIL because the checklist and round report do not exist yet.

- [ ] **Step 3: Create the authoring dogfood checklist**

Create `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`:

```markdown
# Argos Authoring Dogfood Checklist

Date: 2026-05-03

## Purpose

Use this checklist to run authoring dogfood without leaking hidden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the generated packet, `docs/superpowers/templates/argos-authoring-dogfood-report.md`, the prepared workspace path, and the Argos binary path.
- Do not give the runner `testdata/authoring-golden/cases.json`.
- Do not give the runner hidden oracle fields, prior transcripts, implementation design history, evaluator source code, or previous runner reports.
- Allow the runner to inspect and edit only the prepared temp workspace.
- Save the runner report and evaluate it from a separate coordinator context.

## Coordinator Commands

```bash
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
/tmp/argos-authoring-dogfood/argos dogfood authoring cases --json
mkdir -p /tmp/argos-authoring-dogfood/case-001
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
/tmp/argos-authoring-dogfood/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report /tmp/argos-authoring-dogfood/reports/case-001.md --workspace /tmp/argos-authoring-dogfood/case-001 --json
```

## Runner Requirements

- Start from the generated packet, not from repository history.
- Produce canonical `authoring.proposal.v2` JSON before writing candidate knowledge.
- Write candidate knowledge only after the packet's simulated approval boundary allows it.
- Keep candidates in inbox unless the packet explicitly authorizes another boundary.
- Run `author verify --json` when a candidate is written.
- Fill every section in the authoring dogfood report template.

## Evaluator Handoff

Copy the evaluator JSON into the round report and classify each failure as `product`, `harness`, `runner`, `case`, or `blocked`.
```

- [ ] **Step 4: Create the round-0 report**

Create `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`:

```markdown
# Argos Authoring Dogfood Round 0

Date: 2026-05-03
Status: `not-run`

## Purpose

Record the first fresh-runner authoring dogfood round after the process assets are productized.

## Runner Isolation

- One fresh AI session per case.
- Runner receives only the generated packet, report template path, prepared workspace path, and Argos binary path.
- Coordinator keeps golden cases and evaluator details private.

## Fixture Preparation

```bash
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
mkdir -p /tmp/argos-authoring-dogfood/case-001
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
```

## Case Matrix

| Case | Status | Runner Report | Evaluation |
| --- | --- | --- | --- |
| `case-001` | `not-run` | `docs/superpowers/reports/authoring-round-0-case-001.md` | `not-run` |

## Evaluation Commands

```bash
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report docs/superpowers/reports/authoring-round-0-case-001.md --workspace /tmp/argos-authoring-dogfood/case-001 --json
```

## Results

No fresh-runner report has been evaluated yet.

## Failure Classification

- `product`: authoring UX or contract made correct behavior hard.
- `harness`: packet, template, parser, fixture, or evaluator was wrong.
- `runner`: the fresh agent ignored clear instructions.
- `case`: the case expectation or fixture does not represent a useful scenario.
- `blocked`: the run could not execute because tools or workspace setup failed.

## Next Development Decision

Run `case-001` with a fresh runner using the generated packet and authoring report template, then evaluate the saved report.
```

- [ ] **Step 5: Run the checklist and round-record tests**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestAuthoringDogfoodChecklist|TestAuthoringDogfoodRound0' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 2**

Run:

```bash
git add internal/authoringdogfood/authoringdogfood_test.go docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md
git commit -m "docs: add authoring dogfood checklist"
```

---

### Task 3: Public Fixture Seed

**Files:**
- Modify: `internal/authoringdogfood/authoringdogfood_test.go`
- Create: `testdata/authoring-golden/fixtures/full/knowledge/domains.yaml`
- Create: `testdata/authoring-golden/fixtures/full/knowledge/projects.yaml`
- Create: `testdata/authoring-golden/fixtures/full/knowledge/types.yaml`
- Create: `testdata/authoring-golden/fixtures/full/templates/go-service/README.md`
- Create: `testdata/authoring-golden/fixtures/full/internal/api/README.md`
- Create: `testdata/authoring-golden/fixtures/full/internal/retry/README.md`
- Create: `testdata/authoring-golden/fixtures/full/knowledge/items/backend/cache-ttl.md`

- [ ] **Step 1: Add failing fixture seed test**

Append this test to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestAuthoringFixtureSeedSupportsPublicCases(t *testing.T) {
	fixtureRoot := "../../testdata/authoring-golden/fixtures/full"
	requiredFiles := []string{
		"knowledge/domains.yaml",
		"knowledge/projects.yaml",
		"knowledge/types.yaml",
		"templates/go-service/README.md",
		"internal/api/README.md",
		"internal/retry/README.md",
		"knowledge/items/backend/cache-ttl.md",
	}
	for _, rel := range requiredFiles {
		path := filepath.Join(fixtureRoot, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("fixture missing %s: %v", rel, err)
		}
		if info.IsDir() {
			t.Fatalf("fixture path %s is a directory, want file", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %s: %v", rel, err)
		}
		text := string(data)
		for _, forbidden := range hiddenAuthoringProcessTokens() {
			if strings.Contains(text, forbidden) {
				t.Fatalf("fixture %s leaked %q", rel, forbidden)
			}
		}
	}

	if _, err := author.Inspect(fixtureRoot, author.InspectRequest{
		Project:    "mall-api",
		Goal:       "Turn Go service template into future-agent knowledge.",
		FutureTask: "generate a Go service",
		Phase:      "implementation",
		Files:      []string{"templates/go-service/README.md"},
	}); err != nil {
		t.Fatalf("fixture should support author inspect: %v", err)
	}
}
```

- [ ] **Step 2: Run the fixture test and confirm it fails**

Run:

```bash
go test ./internal/authoringdogfood -run TestAuthoringFixtureSeedSupportsPublicCases -count=1
```

Expected: FAIL because `testdata/authoring-golden/fixtures/full` does not exist.

- [ ] **Step 3: Create registry fixture files**

Create `testdata/authoring-golden/fixtures/full/knowledge/domains.yaml`:

```yaml
tech_domains: [backend, database]
business_domains: [catalog, platform]
```

Create `testdata/authoring-golden/fixtures/full/knowledge/projects.yaml`:

```yaml
projects:
  - id: mall-api
    name: Mall API
    path: .
    tech_domains: [backend]
    business_domains: [catalog]
```

Create `testdata/authoring-golden/fixtures/full/knowledge/types.yaml`:

```yaml
types: [rule, decision, lesson, runbook, reference, template, checklist, package]
```

- [ ] **Step 4: Create source-material fixture files**

Create `testdata/authoring-golden/fixtures/full/templates/go-service/README.md`:

```markdown
# Go Service Template

Use this template for new mall-api Go services. It includes a `cmd/` entrypoint,
an `internal/service` package, configuration wiring, health checks, and tests.

Future services should preserve the template structure unless the user approves
a different layout.
```

Create `testdata/authoring-golden/fixtures/full/internal/api/README.md`:

```markdown
# Mall API Surface

The mall-api project exposes catalog and ordering capabilities to internal
callers. Future consumer-facing knowledge should separate observed interface
facts from user interpretation and should name caller responsibilities.
```

Create `testdata/authoring-golden/fixtures/full/internal/retry/README.md`:

```markdown
# Retry Pattern

Repository retry handling wraps transient failures, records attempts, and keeps
non-retryable errors visible. Future lessons should cite this source path and
separate assumptions from observed behavior.
```

- [ ] **Step 5: Create official overlap fixture item**

Create `testdata/authoring-golden/fixtures/full/knowledge/items/backend/cache-ttl.md`:

```markdown
---
id: rule:backend.cache-ttl.v1
title: Cache TTL Rule
type: rule
tech_domains: [backend, database]
business_domains: [catalog]
tags: [cache, ttl]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-05-03
applies_to:
  files:
    - internal/catalog/**
---

Catalog cache TTL knowledge already exists for mall-api. New cache TTL
authoring must check overlap before drafting another durable rule.
```

- [ ] **Step 6: Run the fixture test and authoring package tests**

Run:

```bash
go test ./internal/authoringdogfood -run TestAuthoringFixtureSeedSupportsPublicCases -count=1
go test ./internal/authoringdogfood -count=1
```

Expected: both PASS.

- [ ] **Step 7: Commit Task 3**

Run:

```bash
git add internal/authoringdogfood/authoringdogfood_test.go testdata/authoring-golden/fixtures/full
git commit -m "testdata: add authoring dogfood fixture"
```

---

### Task 4: Packet And README Workflow Alignment

**Files:**
- Modify: `internal/authoringdogfood/authoringdogfood_test.go`
- Modify: `internal/authoringdogfood/packet.go`
- Modify: `README.md`

- [ ] **Step 1: Add failing packet and README workflow assertions**

Modify `TestBuildPacketIncludesNaturalRequestAndAuthoringCommands` in `internal/authoringdogfood/authoringdogfood_test.go` by adding these expected strings to the `want` list:

```go
"docs/superpowers/templates/argos-authoring-dogfood-report.md",
"Use the authoring dogfood report template",
```

Append this README test:

```go
func TestREADMEExplainsAuthoringDogfoodRoundWorkflow(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"docs/superpowers/templates/argos-authoring-dogfood-report.md",
		"docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md",
		"testdata/authoring-golden/fixtures/full",
		"copy the fixture seed to a temp workspace",
		"start a fresh runner",
		"dogfood authoring evaluate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q", want)
		}
	}
	for _, forbidden := range hiddenAuthoringProcessTokens() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("README leaked %q", forbidden)
		}
	}
}
```

- [ ] **Step 2: Run focused tests and confirm they fail**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestBuildPacketIncludesNaturalRequestAndAuthoringCommands|TestREADMEExplainsAuthoringDogfoodRoundWorkflow' -count=1
```

Expected: FAIL because the packet does not mention the report template and README does not yet describe the round workflow.

- [ ] **Step 3: Update packet runner instructions**

In `internal/authoringdogfood/packet.go`, add this line in `renderAuthoringPacketMarkdown` under `## Runner Instructions`:

```go
fmt.Fprintf(&builder, "- Use the authoring dogfood report template at `docs/superpowers/templates/argos-authoring-dogfood-report.md`.\n")
```

Also add this line under `## Required Report Shape` before the section list:

```go
fmt.Fprintf(&builder, "Use the authoring dogfood report template at `docs/superpowers/templates/argos-authoring-dogfood-report.md`.\n\n")
```

- [ ] **Step 4: Update README authoring dogfood section**

In `README.md`, replace the two-sentence authoring dogfood paragraph after the command list with:

```markdown
Use the authoring dogfood checklist and report template for counted runs:

- `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
- `docs/superpowers/templates/argos-authoring-dogfood-report.md`

For each case, copy the fixture seed to a temp workspace, generate a packet,
start a fresh runner with the packet and template, save the runner report, then
run `dogfood authoring evaluate`. The public fixture seed lives at
`testdata/authoring-golden/fixtures/full`.
```

- [ ] **Step 5: Run focused tests and CLI smoke**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestBuildPacketIncludesNaturalRequestAndAuthoringCommands|TestREADMEExplainsAuthoringDogfoodRoundWorkflow' -count=1
go run ./cmd/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos
```

Expected:
- test command PASS;
- packet output contains `docs/superpowers/templates/argos-authoring-dogfood-report.md` and does not contain hidden process tokens.

- [ ] **Step 6: Commit Task 4**

Run:

```bash
git add internal/authoringdogfood/authoringdogfood_test.go internal/authoringdogfood/packet.go README.md
git commit -m "docs: wire authoring dogfood workflow"
```

---

### Task 5: Full Verification And Secrecy Smoke

**Files:**
- Modify only if verification exposes a defect in prior tasks.

- [ ] **Step 1: Run full tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Verify public authoring cases secrecy**

Run:

```bash
go run ./cmd/argos dogfood authoring cases --json
```

Expected:
- output includes `case-001` and `user_request`;
- output does not include `expected_result`, `required_guards`, `required_proposal_properties`, `forbidden_mutations`, `required_evidence_categories`, or internal case IDs.

- [ ] **Step 3: Verify authoring packet references the process assets**

Run:

```bash
go run ./cmd/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos
```

Expected:
- output includes `Argos Authoring Dogfood Runner Packet`;
- output includes `docs/superpowers/templates/argos-authoring-dogfood-report.md`;
- output includes `schema_version: authoring.proposal.v2`;
- output includes `author inspect --json` and `author verify --json`;
- output does not include hidden process tokens or internal case IDs.

- [ ] **Step 4: Verify docs and fixture hygiene**

Run:

```bash
rg -n "expected_result|required_guards|required_proposal_properties|forbidden_mutations|required_evidence_categories|go_template_standard|redis_best_practices|api_consumer_knowledge|observed_repo_lesson|overlap_requires_choice|candidate_not_findable|personal_project_convention" docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md testdata/authoring-golden/fixtures/full README.md
```

Expected: no matches.

- [ ] **Step 5: Check git diff hygiene**

Run:

```bash
git diff --check
git status --short
```

Expected:
- `git diff --check` exits 0;
- `git status --short` shows only intended files if any final fix was needed.

- [ ] **Step 6: Commit any verification fix**

If Step 1-5 required a fix, commit it:

```bash
git add README.md internal/authoringdogfood/authoringdogfood_test.go internal/authoringdogfood/packet.go docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md testdata/authoring-golden/fixtures/full
git commit -m "test: verify authoring dogfood process assets"
```

If no files changed, do not create an empty commit.

---

## Final Verification

- [ ] Run `go test ./... -count=1`
- [ ] Run `go run ./cmd/argos dogfood authoring cases --json`
- [ ] Run `go run ./cmd/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos`
- [ ] Run `git diff --check`
- [ ] Run `git status --short`
- [ ] Run `git log --oneline -6`

Expected final state:

- tests pass;
- public authoring cases and packet expose only public handles, public input, simulated approval boundary, v2 proposal contract, and report-template path;
- checklist, template, round report, fixture seed, README, and packet are aligned;
- worktree is clean after commits.
