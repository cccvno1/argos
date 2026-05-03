# Argos Authoring Contract Tightening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tighten Argos authoring so agents distinguish source state, artifact state, and substantive knowledge readiness before writing candidate knowledge.

**Architecture:** Keep the durable contract in `internal/author/proposal_v2.go` and normalize it through existing verify/evaluate paths. Add minimal public packet guidance so fresh runners can produce valid review-only proposals, while keeping hidden dogfood expectations private. Treat missing substantive content as a review blocker, not as a passed knowledge candidate.

**Tech Stack:** Go standard library, existing `internal/author`, `internal/authoringdogfood`, `internal/cli`, Markdown process assets, `go test`.

---

## File Structure

- Modify `internal/author/proposal_v2.go`
  - Add proposal artifact state and source-state validation helpers.
  - Allow review-only proposals without candidate files when the artifact state says no candidate should be written.
  - Add a substantive-content gate for missing concrete knowledge.
- Modify `internal/author/proposal_v2_test.go`
  - Add red tests for source-state aliases, no-candidate proposals, and missing substantive content.
- Modify `internal/author/verify.go`
  - Preserve review-only proposal behavior when verify is not supposed to run.
  - Keep candidate verification strict when a candidate path is provided.
- Modify `internal/author/verify_test.go`
  - Add tests showing review-only proposals validate without candidate files, and candidate verification still rejects missing candidate paths.
- Modify `internal/authoringdogfood/evaluate.go`
  - Stop turning valid review-only proposals into generic validation failure noise.
  - Surface public, non-oracle diagnostic categories for source trust, assumptions, artifact state, and content readiness.
- Modify `internal/authoringdogfood/authoringdogfood_test.go`
  - Add regression tests covering the public diagnostics and case behavior for unresolved overlap and missing substantive content.
- Modify `internal/authoringdogfood/packet.go`
  - Explain source-state and artifact-state fields in the public packet without exposing hidden oracle values.
- Modify `docs/superpowers/templates/argos-authoring-dogfood-report.md`
  - Keep the current parser-compatible shape; add one optional "Readiness notes" line if needed.
- Modify `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
  - Add coordinator guidance to rerun the failed and false-green cases.
- Modify `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`
  - Append the post-fix rerun result after implementation.

---

### Task 1: Source-State Contract

**Files:**
- Modify: `internal/author/proposal_v2.go`
- Test: `internal/author/proposal_v2_test.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`
- Modify: `internal/authoringdogfood/packet.go`

- [ ] **Step 1: Write failing tests for human-stated design and assumptions**

Add these tests to `internal/author/proposal_v2_test.go` near the existing source-profile tests:

```go
func TestValidateProposalV2AcceptsUserStatedSourceAlias(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.UserConfirmed = []string{"user request: I designed this Go service template"}
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{
			Claim:  "Future Go services should preserve the user-designed template shape.",
			Kind:   "decision",
			Trust:  "user_stated",
			Source: []string{"user request"},
		},
		{
			Claim:  "The concrete layout is documented in templates/go-service.",
			Kind:   "template",
			Trust:  "observed",
			Source: []string{"templates/go-service"},
		},
	}

	findings := ValidateProposalV2(proposal)

	if hasFinding(findings, "fail", "unknown source_profile.claims trust") {
		t.Fatalf("user_stated trust should be accepted, got %#v", findings)
	}
	if hasFinding(findings, "review-needed", "user_stated claim requires") {
		t.Fatalf("user_stated trust should use user_confirmed source bucket, got %#v", findings)
	}
}

func TestValidateProposalV2RequiresStructuredAssumptionsForAssumptionClaims(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.Assumptions = nil
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{
			Claim:  "Retry count and backoff behavior are not evidenced by the source.",
			Kind:   "assumption",
			Trust:  "unknown",
			Source: []string{"internal/retry/README.md"},
		},
	}

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "assumption claim requires source_profile.assumptions") {
		t.Fatalf("expected structured assumptions finding, got %#v", findings)
	}
}
```

Add this test to `internal/authoringdogfood/authoringdogfood_test.go` near packet tests:

```go
func TestAuthoringPacketExplainsPublicSourceState(t *testing.T) {
	cases := loadAuthoringCasesForTest(t)

	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      "case-001",
		Workspace:   "/tmp/argos-authoring/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}

	for _, want := range []string{
		"human-stated design or confirmation",
		"observed workspace facts",
		"synthesized recommendations",
		"assumptions or missing details",
	} {
		if !strings.Contains(packet.Markdown, want) {
			t.Fatalf("packet missing source-state guidance %q:\n%s", want, packet.Markdown)
		}
	}
	assertAuthoringPacketOmitsHiddenTokens(t, packet.Markdown)
}
```

- [ ] **Step 2: Run red tests**

Run:

```bash
go test ./internal/author -run 'TestValidateProposalV2AcceptsUserStatedSourceAlias|TestValidateProposalV2RequiresStructuredAssumptionsForAssumptionClaims' -count=1
go test ./internal/authoringdogfood -run TestAuthoringPacketExplainsPublicSourceState -count=1
```

Expected:

- First command fails because `user_stated` is not an accepted trust value and assumption claims do not require `source_profile.assumptions`.
- Second command fails because packet markdown does not explain source-state categories.

- [ ] **Step 3: Implement source-state validation**

In `internal/author/proposal_v2.go`, update `validSourceClaimTrustV2`:

```go
func validSourceClaimTrustV2(trust string) bool {
	switch trust {
	case "observed", "user_confirmed", "user_stated", "imported", "synthesized", "illustrative", "unknown":
		return true
	default:
		return false
	}
}
```

In `validateSourceProfileV2`, add handling after the existing synthesized and user-confirmed checks:

```go
		if claim.Trust == "user_stated" && len(nonEmpty(source.UserConfirmed)) == 0 {
			addReview("user_stated claim requires user_confirmed source")
		}
		if claim.Trust == "user_stated" && len(nonEmpty(claim.Source)) == 0 {
			addReview("user_stated claim requires source")
		}
		if claim.Kind == "assumption" && len(nonEmpty(source.Assumptions)) == 0 {
			addReview("assumption claim requires source_profile.assumptions")
		}
```

In `internal/authoringdogfood/evaluate.go`, update `proposalSatisfiesHiddenEvidence` so existing hidden evidence checks accept the public alias:

```go
	case "user_confirmed":
		return hasNonEmptyString(proposal.SourceProfile.UserConfirmed) ||
			hasClaimTrust(proposal, "user_confirmed") ||
			hasClaimTrust(proposal, "user_stated"), true
```

In `internal/authoringdogfood/packet.go`, add a short public section after "Proposal Contract":

```go
	fmt.Fprintf(&builder, "## Source State Guidance\n\n")
	fmt.Fprintf(&builder, "- Mark human-stated design or confirmation separately from observed workspace facts.\n")
	fmt.Fprintf(&builder, "- Mark synthesized recommendations as draft guidance and keep assumptions or missing details visible.\n")
	fmt.Fprintf(&builder, "- Use claim-level trust so reviewers can tell what came from the user, the workspace, or agent synthesis.\n\n")
```

- [ ] **Step 4: Run green tests**

Run:

```bash
go test ./internal/author -run 'TestValidateProposalV2AcceptsUserStatedSourceAlias|TestValidateProposalV2RequiresStructuredAssumptionsForAssumptionClaims' -count=1
go test ./internal/authoringdogfood -run TestAuthoringPacketExplainsPublicSourceState -count=1
```

Expected: both commands pass.

- [ ] **Step 5: Commit source-state contract**

Run:

```bash
git add internal/author/proposal_v2.go internal/author/proposal_v2_test.go internal/authoringdogfood/evaluate.go internal/authoringdogfood/packet.go internal/authoringdogfood/authoringdogfood_test.go
git commit -m "author: tighten source state contract"
```

---

### Task 2: Review-Only Artifact State

**Files:**
- Modify: `internal/author/proposal_v2.go`
- Test: `internal/author/proposal_v2_test.go`
- Test: `internal/author/verify_test.go`
- Modify: `internal/authoringdogfood/evaluate.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`

- [ ] **Step 1: Write failing tests for no-candidate proposals**

Add this test to `internal/author/proposal_v2_test.go`:

```go
func TestValidateProposalV2AllowsReviewOnlyProposalWithoutCandidateFiles(t *testing.T) {
	proposal := validProposalV2()
	proposal.ProposedShape.Kind = "review"
	proposal.ProposedShape.Type = "decision"
	proposal.ProposedShape.ID = "review:backend.cache-ttl-overlap"
	proposal.ProposedShape.Path = "knowledge/.inbox/proposals/cache-ttl-overlap/proposal.json"
	proposal.ProposedShape.Status = "review"
	proposal.ProposedShape.Priority = "may"
	proposal.OverlapDecision.Decision = "unresolved"
	proposal.OverlapDecision.HumanChoiceRequired = true
	proposal.CandidateFiles = nil
	proposal.VerificationPlan.ValidatePath = ""
	proposal.VerificationPlan.FindabilityScenarios = nil
	proposal.HumanReview.ProposalApproved = false
	proposal.HumanReview.CandidateWriteApproved = false
	proposal.HumanReview.UnresolvedBlockers = []string{"Existing cache TTL rule overlaps; choose update or distinct scope before writing a candidate."}

	findings := ValidateProposalV2(proposal)

	if hasFinding(findings, "fail", "candidate_files") {
		t.Fatalf("review-only proposal should not require candidate files, got %#v", findings)
	}
	if hasFinding(findings, "fail", "verification_plan.validate_path") {
		t.Fatalf("review-only proposal should not require validate path, got %#v", findings)
	}
	if !hasFinding(findings, "review-needed", "proposal is review-only") {
		t.Fatalf("expected review-only finding, got %#v", findings)
	}
}
```

Add this test to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestEvaluateCaseAcceptsReviewOnlyOverlapProposalWithoutCandidateNoise(t *testing.T) {
	cases := loadAuthoringCasesForTest(t)
	workspace := t.TempDir()
	proposalPath := "knowledge/.inbox/proposals/cache-ttl-overlap/proposal.json"
	writeAuthoringProposal(t, workspace, proposalPath, validReviewOnlyOverlapProposal())

	report := validAuthoringReport("case-005", ResultReviewNeeded, proposalPath, "")
	report.HumanReview.ProposalApproved = false
	report.HumanReview.CandidateWriteApproved = false
	report.VerifyResult = reportStatusNotRun

	evaluation, err := EvaluateCase(cases, "case-005", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("expected review-needed, got %#v", evaluation)
	}
	for _, finding := range evaluation.Findings {
		if strings.Contains(finding.Message, "author validation failed") {
			t.Fatalf("review-only proposal should not emit generic validation failure: %#v", evaluation.Findings)
		}
	}
}
```

Add helper in `internal/authoringdogfood/authoringdogfood_test.go` near other proposal helpers:

```go
func validReviewOnlyOverlapProposal() author.ProposalV2 {
	proposal := validAuthoringProposal("")
	proposal.ProposedShape.Kind = "review"
	proposal.ProposedShape.Type = "decision"
	proposal.ProposedShape.ID = "review:backend.cache-ttl-overlap"
	proposal.ProposedShape.Path = "knowledge/.inbox/proposals/cache-ttl-overlap/proposal.json"
	proposal.ProposedShape.Status = "review"
	proposal.ProposedShape.Priority = "may"
	proposal.OverlapDecision.Decision = "unresolved"
	proposal.OverlapDecision.Reason = "Existing cache TTL knowledge overlaps; choose a distinct scope or update path before writing a candidate."
	proposal.OverlapDecision.HumanChoiceRequired = true
	proposal.CandidateFiles = nil
	proposal.VerificationPlan.ValidatePath = ""
	proposal.VerificationPlan.FindabilityScenarios = nil
	proposal.HumanReview.ProposalApproved = false
	proposal.HumanReview.CandidateWriteApproved = false
	proposal.HumanReview.UnresolvedBlockers = []string{"Resolve overlap before writing a candidate."}
	return proposal
}
```

- [ ] **Step 2: Run red tests**

Run:

```bash
go test ./internal/author -run TestValidateProposalV2AllowsReviewOnlyProposalWithoutCandidateFiles -count=1
go test ./internal/authoringdogfood -run TestEvaluateCaseAcceptsReviewOnlyOverlapProposalWithoutCandidateNoise -count=1
```

Expected: both fail because review-only shape is not allowed and candidate files are required.

- [ ] **Step 3: Implement artifact-state support**

In `internal/author/proposal_v2.go`, extend `ProposedShapeV2`:

```go
type ProposedShapeV2 struct {
	Kind           string `json:"kind"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	ID             string `json:"id"`
	Path           string `json:"path"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	Rationale      string `json:"rationale"`
	EntrypointLoad string `json:"entrypoint_load"`
	ArtifactState  string `json:"artifact_state"`
}
```

Add helpers:

```go
func proposalV2ReviewOnly(proposal ProposalV2) bool {
	state := strings.TrimSpace(proposal.ProposedShape.ArtifactState)
	return state == "review_only" ||
		proposal.ProposedShape.Kind == "review" ||
		proposal.OverlapDecision.Decision == "unresolved" && !proposal.HumanReview.CandidateWriteApproved
}
```

Update `validProposedShapeKindV2`:

```go
	case "item", "package", "review":
		return true
```

Update `validProposedShapeTypeV2`:

```go
	case "rule", "decision", "lesson", "runbook", "reference", "template", "checklist", "package", "review":
		return true
```

Update `validateCandidateAndPlanV2`:

```go
	reviewOnly := proposalV2ReviewOnly(proposal)
	if reviewOnly {
		if len(proposal.CandidateFiles) > 0 {
			addReview("review-only proposal should not include candidate files before approval")
		}
		if strings.TrimSpace(proposal.VerificationPlan.ValidatePath) != "" {
			addReview("review-only proposal should not set verification_plan.validate_path before candidate approval")
		}
		addReview("proposal is review-only until human decisions unblock candidate writing")
		return
	}
```

Place that block after `proposedPath` path cleaning and before `candidate_files` length validation. Skip delivery path boundary checks for `proposed_shape.path` when review-only and the path is under `knowledge/.inbox/proposals/`.

- [ ] **Step 4: Run green tests**

Run:

```bash
go test ./internal/author -run TestValidateProposalV2AllowsReviewOnlyProposalWithoutCandidateFiles -count=1
go test ./internal/authoringdogfood -run TestEvaluateCaseAcceptsReviewOnlyOverlapProposalWithoutCandidateNoise -count=1
```

Expected: both pass.

- [ ] **Step 5: Commit review-only artifact state**

Run:

```bash
git add internal/author/proposal_v2.go internal/author/proposal_v2_test.go internal/authoringdogfood/evaluate.go internal/authoringdogfood/authoringdogfood_test.go
git commit -m "author: support review-only proposals"
```

---

### Task 3: Substantive Content Readiness Gate

**Files:**
- Modify: `internal/author/proposal_v2.go`
- Test: `internal/author/proposal_v2_test.go`
- Modify: `internal/author/verify.go`
- Test: `internal/author/verify_test.go`
- Modify: `internal/authoringdogfood/evaluate.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`

- [ ] **Step 1: Write failing tests for missing concrete knowledge**

Add this test to `internal/author/proposal_v2_test.go`:

```go
func TestValidateProposalV2BlocksCandidateWhenSubstantiveContentMissing(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.Observed = nil
	proposal.SourceProfile.Synthesized = nil
	proposal.SourceProfile.Assumptions = []string{"The concrete convention was not provided."}
	proposal.SourceProfile.OpenQuestions = []string{"What is the exact convention?"}
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{
			Claim:          "The requester wants a personal convention preserved, but did not provide the convention.",
			Kind:           "question",
			Trust:          "user_stated",
			Source:         []string{"user request"},
			RequiresReview: true,
		},
	}
	proposal.ProposedShape.ArtifactState = "candidate"
	proposal.HumanReview.CandidateWriteApproved = true

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "substantive knowledge content is missing") {
		t.Fatalf("expected missing substantive content finding, got %#v", findings)
	}
}
```

Add this test to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestEvaluateCaseFlagsPersonalConventionWithMissingContentAsReviewNeeded(t *testing.T) {
	cases := loadAuthoringCasesForTest(t)
	workspace := t.TempDir()
	proposalPath := "knowledge/.inbox/proposals/personal-convention/proposal.json"
	candidatePath := "knowledge/.inbox/packages/backend/personal-convention"
	proposal := validAuthoringProposal(candidatePath)
	proposal.Scope.Distribution = "personal"
	proposal.SourceProfile.Observed = nil
	proposal.SourceProfile.Synthesized = nil
	proposal.SourceProfile.Assumptions = []string{"The concrete personal convention was not provided."}
	proposal.SourceProfile.OpenQuestions = []string{"What is the exact convention?"}
	proposal.SourceProfile.Claims = []author.SourceClaimV2{
		{
			Claim:          "The user wants a personal convention preserved, but no concrete convention was provided.",
			Kind:           "question",
			Trust:          "user_stated",
			Source:         []string{"user request"},
			RequiresReview: true,
		},
	}
	writeAuthoringProposal(t, workspace, proposalPath, proposal)
	writeCandidatePackage(t, workspace, candidatePath, "package:backend.personal-convention.v1", "Personal Convention")

	report := validAuthoringReport("case-008", ResultPass, proposalPath, candidatePath)

	evaluation, err := EvaluateCase(cases, "case-008", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("expected review-needed, got %#v", evaluation)
	}
	if !hasEvaluationFinding(evaluation, ResultReviewNeeded, "substantive content needs review") {
		t.Fatalf("expected public substantive-content finding, got %#v", evaluation.Findings)
	}
}
```

- [ ] **Step 2: Run red tests**

Run:

```bash
go test ./internal/author -run TestValidateProposalV2BlocksCandidateWhenSubstantiveContentMissing -count=1
go test ./internal/authoringdogfood -run TestEvaluateCaseFlagsPersonalConventionWithMissingContentAsReviewNeeded -count=1
```

Expected: both fail because missing content is not yet a first-class readiness gate.

- [ ] **Step 3: Implement content readiness checks**

In `internal/author/proposal_v2.go`, add helper:

```go
func sourceProfileHasSubstantiveContent(source SourceProfileV2) bool {
	for _, claim := range source.Claims {
		if claim.RequiresReview || claim.Kind == "question" {
			continue
		}
		if strings.TrimSpace(claim.Claim) != "" && claim.Kind != "assumption" {
			return true
		}
	}
	return hasNonEmptyString(source.Observed) ||
		hasNonEmptyString(source.Imported) ||
		hasNonEmptyString(source.Synthesized) ||
		hasNonEmptyString(source.Templates) ||
		hasNonEmptyString(source.Examples)
}
```

At the end of `validateSourceProfileV2`, add:

```go
	if !sourceProfileHasSubstantiveContent(source) && len(nonEmpty(source.OpenQuestions)) > 0 {
		addReview("substantive knowledge content is missing; resolve open questions before candidate writing")
	}
```

In `internal/authoringdogfood/evaluate.go`, after `author.ValidateProposalV2(proposal)` processing, add a public diagnostic when this condition is present:

```go
	if proposalHasMissingSubstantiveContent(proposal) {
		addReview("substantive content needs review before this candidate is usable knowledge")
	}
```

Add helper:

```go
func proposalHasMissingSubstantiveContent(proposal author.ProposalV2) bool {
	return !hasSubstantiveAuthoringClaim(proposal) && hasNonEmptyString(proposal.SourceProfile.OpenQuestions)
}

func hasSubstantiveAuthoringClaim(proposal author.ProposalV2) bool {
	for _, claim := range proposal.SourceProfile.Claims {
		if claim.RequiresReview || claim.Kind == "question" || claim.Kind == "assumption" {
			continue
		}
		if strings.TrimSpace(claim.Claim) != "" {
			return true
		}
	}
	return hasNonEmptyString(proposal.SourceProfile.Observed) ||
		hasNonEmptyString(proposal.SourceProfile.Imported) ||
		hasNonEmptyString(proposal.SourceProfile.Synthesized) ||
		hasNonEmptyString(proposal.SourceProfile.Templates) ||
		hasNonEmptyString(proposal.SourceProfile.Examples)
}
```

- [ ] **Step 4: Run green tests**

Run:

```bash
go test ./internal/author -run TestValidateProposalV2BlocksCandidateWhenSubstantiveContentMissing -count=1
go test ./internal/authoringdogfood -run TestEvaluateCaseFlagsPersonalConventionWithMissingContentAsReviewNeeded -count=1
```

Expected: both pass.

- [ ] **Step 5: Commit substantive-content gate**

Run:

```bash
git add internal/author/proposal_v2.go internal/author/proposal_v2_test.go internal/authoringdogfood/evaluate.go internal/authoringdogfood/authoringdogfood_test.go
git commit -m "author: gate candidates without substantive content"
```

---

### Task 4: Diagnostics, Public Assets, and Rerun

**Files:**
- Modify: `internal/authoringdogfood/evaluate.go`
- Test: `internal/authoringdogfood/authoringdogfood_test.go`
- Modify: `docs/superpowers/templates/argos-authoring-dogfood-report.md`
- Modify: `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
- Modify: `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`

- [ ] **Step 1: Write failing diagnostic tests**

Add this test to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestEvaluateCaseUsesPublicDiagnosticCategories(t *testing.T) {
	cases := loadAuthoringCasesForTest(t)
	workspace := t.TempDir()
	proposalPath := sampleProposalPath
	proposal := validAuthoringProposal(sampleCandidatePath)
	proposal.SourceProfile.UserConfirmed = nil
	proposal.SourceProfile.Claims = []author.SourceClaimV2{
		{Claim: "Template should guide future services.", Kind: "template", Trust: "observed", Source: []string{"templates/go-service"}},
	}
	writeAuthoringProposal(t, workspace, proposalPath, proposal)
	writeCandidatePackage(t, workspace, sampleCandidatePath, "package:backend.go-service-template.v1", "Go Service Template")

	report := validAuthoringReport("case-001", ResultPass, proposalPath, sampleCandidatePath)

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if !hasEvaluationFinding(evaluation, ResultReviewNeeded, "source trust needs review") {
		t.Fatalf("expected source-trust diagnostic, got %#v", evaluation.Findings)
	}
	assertEvaluationFindingsOmit(t, evaluation, "source_profile.user_confirmed", "required_evidence_categories", "oracle")
}
```

- [ ] **Step 2: Run red test**

Run:

```bash
go test ./internal/authoringdogfood -run TestEvaluateCaseUsesPublicDiagnosticCategories -count=1
```

Expected: fails because hidden evaluator failures are still generic fail messages only.

- [ ] **Step 3: Implement public diagnostics without leaking private checks**

In `internal/authoringdogfood/evaluate.go`, add public review diagnostics next to hidden enforcement:

```go
func addPublicAuthoringDiagnostics(proposal author.ProposalV2, addReview func(string)) {
	if !hasNonEmptyString(proposal.SourceProfile.UserConfirmed) && hasClaimKind(proposal, "template") {
		addReview("source trust needs review: human-stated design and observed template facts are not clearly separated")
	}
	if hasClaimKind(proposal, "assumption") && !hasNonEmptyString(proposal.SourceProfile.Assumptions) {
		addReview("assumptions need review: assumption claims are not mirrored in the structured assumptions list")
	}
	if proposalV2LooksReviewOnlyForDogfood(proposal) {
		addReview("artifact state needs review: proposal is waiting for a human decision before candidate writing")
	}
	if proposalHasMissingSubstantiveContent(proposal) {
		addReview("substantive content needs review before this candidate is usable knowledge")
	}
}
```

Call it before `enforceHiddenOracle(...)`:

```go
	addPublicAuthoringDiagnostics(proposal, addReview)
	enforceHiddenOracle(tc, proposal, report, candidatePath, verifyRan, verifyResult, addFail, addReview)
```

Add helper:

```go
func proposalV2LooksReviewOnlyForDogfood(proposal author.ProposalV2) bool {
	return proposal.ProposedShape.ArtifactState == "review_only" ||
		proposal.ProposedShape.Kind == "review" ||
		proposal.OverlapDecision.Decision == "unresolved" && !proposal.HumanReview.CandidateWriteApproved
}
```

- [ ] **Step 4: Update packet/checklist wording**

In `docs/superpowers/templates/argos-authoring-dogfood-report.md`, add under `## Result`:

```markdown
Readiness notes: `none | source-state review | artifact-state review | content-readiness review`
```

In `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`, add under "Evaluator Handoff":

```markdown
- Rerun the template, consumer-reference, observed-lesson, overlap, and personal-convention cases after source/artifact/content contract changes.
```

- [ ] **Step 5: Run focused tests**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestEvaluateCaseUsesPublicDiagnosticCategories|TestAuthoringDogfoodReportTemplateIsParserCompatible|TestAuthoringDogfoodChecklistDefinesFreshRunnerWorkflow' -count=1
```

Expected: pass.

- [ ] **Step 6: Rerun dogfood cases**

Create a fresh round root and rerun the targeted cases:

```bash
ROUND_ROOT=$(mktemp -d /tmp/argos-authoring-tightening.XXXXXX)
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports"
go build -o "$ROUND_ROOT/argos" ./cmd/argos
for case_id in case-001 case-003 case-004 case-005 case-008; do
  mkdir -p "$ROUND_ROOT/$case_id"
  cp -R testdata/authoring-golden/fixtures/full/. "$ROUND_ROOT/$case_id/"
  "$ROUND_ROOT/argos" dogfood authoring packet --case "$case_id" --workspace "$ROUND_ROOT/$case_id" --argos-binary "$ROUND_ROOT/argos" > "$ROUND_ROOT/packets/$case_id.md"
done
```

Dispatch fresh runners only if the user approves subagent execution for the rerun. If running manually in the same session, record that the rerun is coordinator-assisted and do not claim fresh-runner evidence.

- [ ] **Step 7: Update round report**

Append a "Post-Fix Rerun" section to `docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md`:

```markdown
## Post-Fix Rerun

Date: 2026-05-03
Round Root: write the actual `/tmp/argos-authoring-tightening.*` directory used for this rerun.

| Case | Status | Evaluation | Notes |
| --- | --- | --- | --- |
```

Fill one row per rerun case using the actual runner report status, evaluator
result, and a public note that names only source-state, assumption-state,
review-only, or content-readiness behavior. Do not copy hidden oracle fields or
private evaluator property names into the report.

- [ ] **Step 8: Run final verification**

Run:

```bash
go test ./internal/author ./internal/authoringdogfood ./internal/cli -count=1
go test ./... -count=1
rg -n "expected_result|required_guards|required_proposal_properties|forbidden_mutations|required_evidence_categories|go_template_standard|redis_best_practices|api_consumer_knowledge|observed_repo_lesson|overlap_requires_choice|candidate_not_findable|proposal_must_precede_candidate|unauthorized_authority|personal_project_convention" docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md
git diff --check
```

Expected:

- Go tests pass.
- The `rg` command returns exit code `1` with no matches.
- `git diff --check` exits `0`.

- [ ] **Step 9: Commit diagnostics and rerun assets**

Run:

```bash
git add internal/authoringdogfood/evaluate.go internal/authoringdogfood/authoringdogfood_test.go docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md
git commit -m "authoringdogfood: improve authoring readiness diagnostics"
```

---

## Self-Review

- Spec coverage: The plan covers source trust gaps from template/API cases, structured assumptions from observed retry, review-only no-candidate overlap, substantive content false-green behavior, runner packet guidance, evaluator diagnostics, and rerun evidence.
- Placeholder scan: Implementation steps include exact file paths, test names, commands, and code snippets. The rerun report step requires actual rerun values and forbids committing empty template rows.
- Type consistency: New fields use `ProposedShapeV2.ArtifactState` with JSON `artifact_state`; source trust alias uses `user_stated`; diagnostics helpers live in `internal/authoringdogfood/evaluate.go`.
