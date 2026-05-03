# Argos Authoring Round 1 Tightening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the Round 1 authoring dogfood blockers by making canonical proposal scaffolds available to runners, making review-only proposals explicit, and preventing missing substantive knowledge from passing as ready knowledge.

**Architecture:** Keep the product boundary stable: Argos does not run agent workflows, but it provides machine-readable contracts and deterministic evaluation. Extend `author inspect` with a canonical `authoring.proposal.v2` scaffold, tighten public packet/report guidance, and make evaluator diagnostics classify missing content and authorization review states consistently.

**Tech Stack:** Go standard library, existing Argos CLI, `internal/author`, `internal/authoringdogfood`, `go test`.

---

## File Structure

- Modify `internal/author/inspect.go`
  - Add `proposal_scaffold` to `InspectResponse`.
  - Build valid canonical proposal scaffolds from inspect inputs and overlap facts.
  - Return review-only scaffold shape when overlap is unresolved.
- Modify `internal/author/proposal_v2_test.go`
  - Add tests for scaffold validity and review-only scaffold fields.
- Modify `internal/authoringdogfood/packet.go`
  - Tell runners to use `author inspect`'s `proposal_scaffold`.
  - Add explicit public review-only JSON field guidance.
  - Add result semantics for authorization and content readiness.
- Modify `internal/authoringdogfood/authoringdogfood_test.go`
  - Add packet tests for proposal scaffold guidance and review-only fields.
  - Add evaluator test for missing substantive personal convention content.
  - Add evaluator test for authorization-sensitive self-reported pass becoming review-needed.
- Modify `internal/authoringdogfood/evaluate.go`
  - Make content-readiness diagnostics catch missing actionable convention content even when a runner writes a placeholder package.
  - Make authorization-sensitive cases add review-needed when priority-must is requested but not authorized, independent of runner self-report.
- Modify `docs/superpowers/templates/argos-authoring-dogfood-report.md`
  - Align report result guidance with evaluator semantics.
- Modify `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
  - Mention scaffold and review-only rerun focus.

## Task 1: Canonical Proposal Scaffold From Inspect

**Files:**
- Modify: `internal/author/inspect.go`
- Test: `internal/author/proposal_v2_test.go`

- [ ] **Step 1: Write failing tests**

Add tests that call `Inspect` on a fixture workspace and assert:

```go
func TestInspectIncludesValidProposalV2Scaffold(t *testing.T) {
	root := t.TempDir()
	writeAuthoringRegistryFixture(t, root)

	resp, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal: "Create reusable Redis draft practices for future agents.",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	if resp.ProposalScaffold.SchemaVersion != ProposalV2SchemaVersion {
		t.Fatalf("schema = %q", resp.ProposalScaffold.SchemaVersion)
	}
	if resp.ProposalScaffold.UserRequest != "Create reusable Redis draft practices for future agents." {
		t.Fatalf("user_request = %q", resp.ProposalScaffold.UserRequest)
	}
	if findings := ValidateProposalV2(resp.ProposalScaffold); hasFailingFinding(findings) {
		t.Fatalf("scaffold has failing findings: %#v", findings)
	}
}
```

Add a review-only overlap test:

```go
func TestInspectProposalScaffoldUsesReviewOnlyForUnresolvedOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthoringRegistryFixture(t, root)
	writeOfficialKnowledgeFixture(t, root, "knowledge/items/backend/cache-ttl.md")

	resp, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal: "Create another cache TTL rule that may overlap existing cache knowledge.",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := resp.ProposalScaffold
	if proposal.ProposedShape.Kind != "review" {
		t.Fatalf("kind = %q, want review", proposal.ProposedShape.Kind)
	}
	if proposal.ProposedShape.ArtifactState != "review_only" {
		t.Fatalf("artifact_state = %q, want review_only", proposal.ProposedShape.ArtifactState)
	}
	if proposal.OverlapDecision.Decision != "unresolved" {
		t.Fatalf("overlap decision = %q, want unresolved", proposal.OverlapDecision.Decision)
	}
	if len(proposal.CandidateFiles) != 0 {
		t.Fatalf("candidate files = %#v, want none", proposal.CandidateFiles)
	}
	if proposal.VerificationPlan.ValidatePath != "" {
		t.Fatalf("validate path = %q, want empty", proposal.VerificationPlan.ValidatePath)
	}
	if findings := ValidateProposalV2(proposal); hasFailingFinding(findings) {
		t.Fatalf("review-only scaffold has failing findings: %#v", findings)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/author -count=1
```

Expected: fails because `InspectResponse.ProposalScaffold` does not exist.

- [ ] **Step 3: Implement scaffold**

Add `ProposalScaffold ProposalV2 json:"proposal_scaffold"` to `InspectResponse`.

Build scaffold in `Inspect` after overlap is populated:

```go
response.ProposalScaffold = buildProposalScaffold(response, req)
```

Implementation details:

- Use slugged path `knowledge/.inbox/packages/<project>/<goal-slug>` for candidate scaffolds.
- Use slugged path `knowledge/.inbox/proposals/<project>/<goal-slug>/proposal.json` for review-only scaffolds.
- If official or inbox overlap exists, set:
  - `proposed_shape.kind: "review"`
  - `proposed_shape.type: "review"`
  - `proposed_shape.artifact_state: "review_only"`
  - `overlap_decision.decision: "unresolved"`
  - `candidate_files: []`
  - `verification_plan.validate_path: ""`
- Otherwise set a normal package candidate scaffold with one `KNOWLEDGE.md` candidate file.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
go test ./internal/author -count=1
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/author/inspect.go internal/author/proposal_v2_test.go
git commit -m "author: add proposal scaffold to inspect"
```

## Task 2: Runner-Facing Packet Guidance

**Files:**
- Modify: `internal/authoringdogfood/packet.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`

- [ ] **Step 1: Write failing packet tests**

Add tests that assert packet Markdown includes public, non-hidden guidance:

```go
func TestAuthoringPacketExplainsProposalScaffold(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	packet, err := BuildPacket(cases, PacketOptions{
		CaseID: "case-001", Workspace: "/tmp/ws", ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	for _, want := range []string{
		"`proposal_scaffold`",
		"Copy the scaffold shape before changing values",
		"`user_request` is a string",
		"`knowledge_goal` is a string",
	} {
		if !strings.Contains(packet.Markdown, want) {
			t.Fatalf("packet missing %q:\n%s", want, packet.Markdown)
		}
	}
}
```

Add review-only guidance test:

```go
func TestAuthoringPacketExplainsReviewOnlyShape(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	packet, err := BuildPacket(cases, PacketOptions{
		CaseID: "case-005", Workspace: "/tmp/ws", ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	for _, want := range []string{
		"`proposed_shape.kind`: `review`",
		"`proposed_shape.artifact_state`: `review_only`",
		"`overlap_decision.decision`: `unresolved`",
		"`candidate_files`: `[]`",
		"`verification_plan.validate_path`: empty string",
	} {
		if !strings.Contains(packet.Markdown, want) {
			t.Fatalf("packet missing %q:\n%s", want, packet.Markdown)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: fails because packet text does not mention scaffold or explicit review-only fields.

- [ ] **Step 3: Update packet renderer**

Add a `## Proposal Scaffold` section after CLI equivalents and expand `## Proposal Contract`.

Also add `## Result Semantics`:

- `pass`: candidate is ready for review under current approval boundary.
- `review-needed`: approval, authorization, source state, artifact state, findability, or substantive content still needs human decision.
- `fail`: workflow violated boundary or artifact cannot be inspected.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/authoringdogfood/packet.go internal/authoringdogfood/authoringdogfood_test.go
git commit -m "authoringdogfood: document scaffold and review-only shape"
```

## Task 3: Content And Authorization Readiness Evaluation

**Files:**
- Modify: `internal/authoringdogfood/evaluate.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`
- Modify: `docs/superpowers/templates/argos-authoring-dogfood-report.md`
- Modify: `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`

- [ ] **Step 1: Write failing evaluator tests**

Add a test mirroring Round 1 `case-008`: proposal has user-confirmed existence of a convention, open questions for exact wording, and candidate write approved. Expected evaluation: `review-needed`, not `pass`.

Add a test mirroring Round 1 `case-007`: runner reports `pass` but proposal requests or discusses elevated must authority while `priority_must_authorized` is false. Expected evaluation includes review-needed.

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: missing-content test currently passes incorrectly or lacks diagnostic; authorization self-report test lacks public diagnostic.

- [ ] **Step 3: Implement diagnostics**

In `evaluate.go`:

- Treat `source_profile.open_questions` containing exact missing convention/content wording as content-readiness review when candidate writing happened.
- Treat `future_use.missing_needs` and `human_review.review_questions` containing missing exact knowledge content as content-readiness review.
- Treat priority-must intent without authorization as review-needed even when the runner report says pass.
- Keep official mutation and promotion violations as fail.

- [ ] **Step 4: Update public template/checklist**

Add short wording to the report template:

- Missing actionable knowledge content means `review-needed`.
- Unauthorized elevated authority means `review-needed` unless a boundary was violated, which is `fail`.

- [ ] **Step 5: Verify GREEN**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/authoringdogfood/evaluate.go internal/authoringdogfood/authoringdogfood_test.go docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md
git commit -m "authoringdogfood: tighten readiness evaluation"
```

## Task 4: Verification And Round Follow-Up

**Files:**
- No production changes expected.

- [ ] **Step 1: Run focused tests**

```bash
go test ./internal/author ./internal/authoringdogfood ./internal/cli -count=1
```

Expected: pass.

- [ ] **Step 2: Run full test suite**

```bash
go test ./... -count=1
```

Expected: pass.

- [ ] **Step 3: Hidden-token scan**

```bash
if rg -n "expected_result|required_guards|required_proposal_properties|forbidden_mutations|required_evidence_categories|go_template_standard|redis_best_practices|api_consumer_knowledge|observed_repo_lesson|overlap_requires_choice|candidate_not_findable|proposal_must_precede_candidate|unauthorized_authority|personal_project_convention" docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/templates/argos-authoring-dogfood-report.md; then exit 1; else echo "hidden-token scan clean"; fi
```

Expected: `hidden-token scan clean`.

- [ ] **Step 4: Check formatting and status**

```bash
git diff --check
git status --short --branch
```

Expected: no whitespace errors; branch contains only intentional committed changes.

## Self-Review

- Spec coverage: Covers Round 1 decisions for scaffold, review-only shape, missing-content readiness, and authorization-sensitive result semantics.
- Placeholder scan: No TODO/TBD placeholders remain.
- Type consistency: Uses existing `ProposalV2`, `InspectResponse`, `Packet`, and evaluator types.
