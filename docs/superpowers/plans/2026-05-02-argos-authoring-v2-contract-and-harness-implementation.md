# Argos Authoring V2 Contract And Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the authoring v2 proposal contract and a productized authoring dogfood harness that validates realistic write-side workflows from natural user requests.

**Architecture:** Keep deterministic authoring behavior in `internal/author`, add v2 proposal validation beside the existing v1 contract, and normalize v2 proposals into the existing verification pipeline. Add a separate `internal/authoringdogfood` package for natural authoring cases, runner packets, report parsing, and workspace-aware evaluation, then expose it under `argos dogfood authoring`.

**Tech Stack:** Go, standard `flag` CLI parsing, JSON serialization, existing Argos knowledge validation, existing `author.Verify`, Markdown report parsing, `testdata/authoring-golden`, README, and the local `capture-knowledge` skill.

---

## Scope Check

This plan intentionally implements the contract and harness surface only. It does not add automatic proposal generation, MCP authoring tools, runner orchestration, or scaffold commands. Those can be separate specs after this surface proves stable.

## File Structure

- Create `internal/author/proposal_v2.go`
  - Define `authoring.proposal.v2` structs.
  - Validate audience, scope, source profile, future use, delivery, candidate files, verification plan, and human review decisions.
  - Normalize v2 proposals into the existing v1-shaped `Proposal` used by `Verify`.
- Create `internal/author/proposal_v2_test.go`
  - Lock required v2 fields and source/trust validation.
  - Cover synthesized claims, user-confirmed claims, unresolved overlap, and unauthorized authority.
- Modify `internal/author/verify.go`
  - Load v1 or v2 proposal based on `schema_version`.
  - Preserve v1 behavior.
  - Use v2 validation findings and normalized candidate policy fields for v2 proposals.
- Modify `internal/author/verify_test.go`
  - Add v2 verification tests for pass, missing future use, unresolved overlap, and unauthorized `priority: must`.
- Create `internal/authoringdogfood/cases.go`
  - Load natural authoring cases from `testdata/authoring-golden/cases.json`.
  - Expose public summaries that hide hidden oracle fields.
- Create `internal/authoringdogfood/packet.go`
  - Build authoring runner packets from public case input.
  - Include authoring CLI equivalents and required report shape.
- Create `internal/authoringdogfood/report.go`
  - Parse authoring runner reports with proposal path, candidate path, verification result, human decisions, guards, and result.
- Create `internal/authoringdogfood/evaluate.go`
  - Evaluate report plus workspace artifacts.
  - Load proposal, check candidate writes, call `author.Verify`, and apply hidden scenario guards.
- Create `internal/authoringdogfood/authoringdogfood_test.go`
  - Cover public summary secrecy, packet content, report parsing, evaluator behavior, and workspace-backed failure when artifacts are missing.
- Replace `testdata/authoring-golden/cases.json`
  - Move public inputs to natural `user_request`.
  - Add hidden oracle and simulated approval boundaries.
  - Cover Go template, Redis best practices, API consumer knowledge, observed lesson, overlap, findability, unauthorized authority, and personal convention.
- Delete `internal/authoringtest/cases.go`
  - The old seed-only loader is replaced by `internal/authoringdogfood`.
- Delete `internal/authoringtest/cases_test.go`
  - Coverage moves into `internal/authoringdogfood/authoringdogfood_test.go`.
- Modify `internal/cli/cli.go`
  - Add `argos dogfood authoring cases|packet|evaluate`.
  - Keep existing discovery dogfood commands unchanged.
- Modify `internal/cli/cli_test.go`
  - Cover authoring dogfood CLI cases, packet, evaluate, required flags, and hidden oracle secrecy.
- Modify `README.md`
  - Add v2 wording around natural authoring input, source profile, and authoring dogfood commands.
- Modify `skills/capture-knowledge/SKILL.md`
  - Reframe proposal sections around v2 audience, scope, source profile, future use, and human review decisions.

---

### Task 1: Proposal V2 Contract

**Files:**
- Create: `internal/author/proposal_v2.go`
- Create: `internal/author/proposal_v2_test.go`

- [ ] **Step 1: Write failing v2 validation tests**

Create `internal/author/proposal_v2_test.go`:

```go
package author

import "testing"

func TestValidateProposalV2AcceptsCompleteMixedSourceDraft(t *testing.T) {
	proposal := validProposalV2()

	findings := ValidateProposalV2(proposal)

	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestValidateProposalV2RequiresNaturalUserRequestAudienceScopeAndFutureUse(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProposalV2)
		want string
	}{
		{name: "user request", edit: func(p *ProposalV2) { p.UserRequest = "" }, want: "user_request is required"},
		{name: "audience", edit: func(p *ProposalV2) { p.Audience.Primary = "" }, want: "audience.primary is required"},
		{name: "scope", edit: func(p *ProposalV2) { p.Scope.Projects = nil; p.Scope.Distribution = "" }, want: "scope must include projects or distribution"},
		{name: "future use", edit: func(p *ProposalV2) { p.FutureUse.TriggerRequests = nil; p.FutureUse.QueryPhrases = nil }, want: "future_use must include trigger_requests or query_phrases"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			tt.edit(&proposal)

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "fail", tt.want) {
				t.Fatalf("expected finding containing %q, got %#v", tt.want, findings)
			}
		})
	}
}

func TestValidateProposalV2RequiresSourceProfileClaimsForSynthesizedRecommendations(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.Assumptions = nil
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{Claim: "Use Redis locks for stampede protection.", Kind: "recommendation", Trust: "synthesized", RequiresReview: false},
	}

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "synthesized claim requires assumptions or review") {
		t.Fatalf("expected synthesized review finding, got %#v", findings)
	}
}

func TestValidateProposalV2RequiresUserConfirmedSourceForDecisions(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.UserConfirmed = nil
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{Claim: "All generated services use the standard layout.", Kind: "decision", Trust: "user_confirmed"},
	}

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "user_confirmed claim requires user_confirmed source") {
		t.Fatalf("expected user-confirmed source finding, got %#v", findings)
	}
}

func TestValidateProposalV2RejectsUnauthorizedPriorityMustAndOfficialMutation(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProposalV2)
		want string
	}{
		{
			name: "priority must",
			edit: func(p *ProposalV2) {
				p.ProposedShape.Priority = "must"
				p.Delivery.PriorityMustAuthorized = false
			},
			want: "priority: must requires explicit authorization",
		},
		{
			name: "official mutation",
			edit: func(p *ProposalV2) {
				p.Delivery.Path = "official_review"
				p.Delivery.OfficialMutationAuthorized = false
			},
			want: "official mutation requires explicit authorization",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			tt.edit(&proposal)

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "fail", tt.want) {
				t.Fatalf("expected finding containing %q, got %#v", tt.want, findings)
			}
		})
	}
}

func TestValidateProposalV2MarksUnresolvedOverlapReviewNeeded(t *testing.T) {
	proposal := validProposalV2()
	proposal.OverlapDecision.Decision = "unresolved"
	proposal.OverlapDecision.HumanChoiceRequired = true

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "overlap decision is unresolved") {
		t.Fatalf("expected unresolved overlap review finding, got %#v", findings)
	}
}

func validProposalV2() ProposalV2 {
	return ProposalV2{
		SchemaVersion: "authoring.proposal.v2",
		UserRequest:   "I designed a Go service template. Turn it into reusable knowledge for future agents.",
		KnowledgeGoal: "Help future agents generate Go services using the approved project template.",
		Project:       "mall-api",
		Audience: AudienceV2{
			Primary: "implementer_agent",
			AgentActionsSupported: []string{
				"generate a new Go service using the approved project style",
			},
		},
		Scope: ScopeV2{
			Projects:       []string{"mall-api"},
			Stability:      "draft",
			Distribution:   "project",
			SubjectDomains: []string{"service-template"},
			TechDomains:    []string{"backend"},
			FileGlobs:      []string{"templates/go-service/**"},
			OutOfScope:     []string{"legacy services outside the template path"},
		},
		SourceProfile: SourceProfileV2{
			UserConfirmed: []string{"The user wants this template to guide future Go service generation."},
			Observed:      []string{"templates/go-service"},
			Templates:     []string{"templates/go-service"},
			Assumptions:   []string{"Template examples are draft until the user reviews generated knowledge."},
			Claims: []SourceClaimV2{
				{Claim: "Future Go services should use the template layout.", Kind: "decision", Trust: "user_confirmed", Source: []string{"user request"}},
				{Claim: "The directory layout comes from templates/go-service.", Kind: "fact", Trust: "observed", Source: []string{"templates/go-service"}},
			},
		},
		ProposedShape: ProposedShapeV2{
			Kind:           "package",
			Type:           "package",
			Title:          "Go Service Template Knowledge",
			ID:             "package:backend.go-service-template.v1",
			Path:           "knowledge/.inbox/packages/backend/go-service-template",
			Status:         "draft",
			Priority:       "should",
			Rationale:      "The template needs entrypoint guidance plus examples.",
			EntrypointLoad: "start_here",
		},
		FutureUse: FutureUseV2{
			TriggerRequests: []string{"generate a Go service", "create a backend service from the standard template"},
			NegativeTriggers: []string{"write a one-off Go script"},
			Phases: []string{"planning", "implementation", "review"},
			QueryPhrases: []string{"go service template", "standard go service layout"},
			ExpectedUse: "read_before_implementation",
			CitationPolicy: "cite_after_use",
		},
		Applicability: Applicability{
			WhenToUse: []string{"When creating a new Go service in mall-api."},
			WhenNotToUse: []string{"When changing an existing legacy service that does not follow the template."},
			Tradeoffs: []string{"The template improves consistency but may not fit small scripts."},
		},
		OverlapDecision: OverlapDecisionV2{
			Decision: "create_new",
			Reason: "No existing Go service template knowledge covers this future task.",
		},
		Delivery: DeliveryV2{
			Path:                       "inbox",
			WriteRequiresHumanApproval: true,
			ReviewPacketRequired:       true,
		},
		CandidateFiles: []CandidateFile{
			{Path: "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", Purpose: "entrypoint", Load: "start_here"},
		},
		VerificationPlan: VerificationPlan{
			ValidatePath: "knowledge/.inbox/packages/backend/go-service-template",
			FindabilityScenarios: []FindabilityScenario{
				{Project: "mall-api", Phase: "implementation", Task: "generate a Go service", Query: "go service template"},
			},
		},
		HumanReview: HumanReviewV2{
			ReviewQuestions: []string{"Is this the right future-agent audience?"},
		},
	}
}

```

- [ ] **Step 2: Run the new tests and confirm they fail**

Run:

```bash
go test ./internal/author -run 'TestValidateProposalV2' -count=1
```

Expected: FAIL with undefined v2 proposal types or functions.

- [ ] **Step 3: Add the v2 proposal structs and validation**

Create `internal/author/proposal_v2.go`:

```go
package author

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const ProposalV2SchemaVersion = "authoring.proposal.v2"

type ProposalV2 struct {
	SchemaVersion    string            `json:"schema_version"`
	UserRequest      string            `json:"user_request"`
	KnowledgeGoal    string            `json:"knowledge_goal"`
	Project          string            `json:"project"`
	Audience         AudienceV2        `json:"audience"`
	Scope            ScopeV2           `json:"scope"`
	SourceProfile    SourceProfileV2   `json:"source_profile"`
	ProposedShape    ProposedShapeV2   `json:"proposed_shape"`
	FutureUse        FutureUseV2       `json:"future_use"`
	Applicability    Applicability     `json:"applicability"`
	OverlapDecision  OverlapDecisionV2 `json:"overlap_decision"`
	Delivery         DeliveryV2        `json:"delivery"`
	CandidateFiles   []CandidateFile   `json:"candidate_files"`
	VerificationPlan VerificationPlan  `json:"verification_plan"`
	HumanReview      HumanReviewV2     `json:"human_review"`
}

type AudienceV2 struct {
	Primary               string   `json:"primary"`
	Secondary             []string `json:"secondary,omitempty"`
	HumanReviewers        []string `json:"human_reviewers,omitempty"`
	AgentActionsSupported []string `json:"agent_actions_supported"`
}

type ScopeV2 struct {
	Projects       []string `json:"projects"`
	Stability      string   `json:"stability"`
	Distribution   string   `json:"distribution"`
	SubjectDomains []string `json:"subject_domains"`
	TechDomains    []string `json:"tech_domains"`
	Files          []string `json:"files"`
	FileGlobs      []string `json:"file_globs"`
	OutOfScope     []string `json:"out_of_scope"`
}

type SourceProfileV2 struct {
	UserConfirmed []string        `json:"user_confirmed"`
	Observed      []string        `json:"observed"`
	Imported      []string        `json:"imported"`
	Synthesized   []string        `json:"synthesized"`
	Templates     []string        `json:"templates"`
	Examples      []string        `json:"examples"`
	Assumptions   []string        `json:"assumptions"`
	OpenQuestions []string        `json:"open_questions"`
	Claims        []SourceClaimV2 `json:"claims"`
}

type SourceClaimV2 struct {
	Claim          string   `json:"claim"`
	Kind           string   `json:"kind"`
	Source         []string `json:"source"`
	Trust          string   `json:"trust"`
	RequiresReview bool     `json:"requires_review"`
}

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
}

type FutureUseV2 struct {
	TriggerRequests []string `json:"trigger_requests"`
	NegativeTriggers []string `json:"negative_triggers"`
	Phases          []string `json:"phases"`
	QueryPhrases    []string `json:"query_phrases"`
	ExpectedUse     string   `json:"expected_use"`
	CitationPolicy  string   `json:"citation_policy"`
	MissingNeeds    []string `json:"missing_needs"`
}

type OverlapDecisionV2 struct {
	OfficialOverlap     []string `json:"official_overlap"`
	InboxOverlap        []string `json:"inbox_overlap"`
	PossibleOverlap     []string `json:"possible_overlap"`
	Decision            string   `json:"decision"`
	Reason              string   `json:"reason"`
	HumanChoiceRequired bool     `json:"human_choice_required"`
}

type DeliveryV2 struct {
	Path                       string `json:"path"`
	WriteRequiresHumanApproval bool   `json:"write_requires_human_approval"`
	PriorityMustAuthorized     bool   `json:"priority_must_authorized"`
	PromoteAuthorized          bool   `json:"promote_authorized"`
	OfficialMutationAuthorized bool   `json:"official_mutation_authorized"`
	ReviewPacketRequired       bool   `json:"review_packet_required"`
}

type HumanReviewV2 struct {
	ReviewQuestions            []string `json:"review_questions"`
	ProposalApproved           bool     `json:"proposal_approved,omitempty"`
	CandidateWriteApproved     bool     `json:"candidate_write_approved,omitempty"`
	PriorityMustAuthorized     bool     `json:"priority_must_authorized,omitempty"`
	OfficialMutationAuthorized bool     `json:"official_mutation_authorized,omitempty"`
	PromoteAuthorized          bool     `json:"promote_authorized,omitempty"`
	RequestedEdits             []string `json:"requested_edits,omitempty"`
	UnresolvedBlockers         []string `json:"unresolved_blockers,omitempty"`
}

func LoadProposalV2(path string) (ProposalV2, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProposalV2{}, fmt.Errorf("read proposal: %w", err)
	}
	var proposal ProposalV2
	if err := json.Unmarshal(data, &proposal); err != nil {
		return ProposalV2{}, fmt.Errorf("parse proposal JSON: %w", err)
	}
	return proposal, nil
}

func ValidateProposalV2(proposal ProposalV2) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	if strings.TrimSpace(proposal.SchemaVersion) != ProposalV2SchemaVersion {
		addFail("schema_version must be authoring.proposal.v2")
	}
	if strings.TrimSpace(proposal.UserRequest) == "" {
		addFail("user_request is required")
	}
	if strings.TrimSpace(proposal.KnowledgeGoal) == "" {
		addFail("knowledge_goal is required")
	}
	if strings.TrimSpace(proposal.Project) == "" {
		addFail("project is required")
	}
	if strings.TrimSpace(proposal.Audience.Primary) == "" {
		addFail("audience.primary is required")
	}
	if len(nonEmpty(proposal.Audience.AgentActionsSupported)) == 0 {
		addReview("audience.agent_actions_supported should describe future agent actions")
	}
	if len(nonEmpty(proposal.Scope.Projects)) == 0 && strings.TrimSpace(proposal.Scope.Distribution) == "" {
		addFail("scope must include projects or distribution")
	}
	if strings.TrimSpace(proposal.Scope.Stability) == "" {
		addReview("scope.stability is required")
	}
	validateSourceProfileV2(proposal.SourceProfile, addFail, addReview)
	validateProposedShapeV2(proposal.ProposedShape, proposal.Delivery, addFail, addReview)
	validateFutureUseV2(proposal.FutureUse, addFail, addReview)
	if len(nonEmpty(proposal.Applicability.WhenToUse)) == 0 || len(nonEmpty(proposal.Applicability.WhenNotToUse)) == 0 {
		addReview("applicability must include when_to_use and when_not_to_use")
	}
	validateOverlapDecisionV2(proposal.OverlapDecision, addFail, addReview)
	validateDeliveryV2(proposal.Delivery, proposal.ProposedShape, addFail)
	validateCandidateAndPlanV2(proposal, addFail, addReview)
	if len(nonEmpty(proposal.HumanReview.ReviewQuestions)) == 0 {
		addReview("human_review.review_questions should include reviewer decisions")
	}
	return findings
}
```

In the same file, add helper functions used above:

```go
func validateSourceProfileV2(source SourceProfileV2, addFail func(string), addReview func(string)) {
	if !hasAnySourceV2(source) {
		addReview("source_profile must include at least one source bucket")
	}
	if len(source.Claims) == 0 {
		addReview("source_profile.claims should include claim-level trust")
	}
	for i, claim := range source.Claims {
		if strings.TrimSpace(claim.Claim) == "" || strings.TrimSpace(claim.Kind) == "" || strings.TrimSpace(claim.Trust) == "" {
			addFail(fmt.Sprintf("source_profile.claims[%d] must include claim, kind, and trust", i))
			continue
		}
		if claim.Trust == "synthesized" && len(nonEmpty(source.Assumptions)) == 0 && !claim.RequiresReview {
			addReview("synthesized claim requires assumptions or review")
		}
		if claim.Trust == "user_confirmed" && len(nonEmpty(source.UserConfirmed)) == 0 {
			addReview("user_confirmed claim requires user_confirmed source")
		}
		if claim.Trust == "observed" && len(nonEmpty(source.Observed)) == 0 {
			addReview("observed claim requires observed source")
		}
		if claim.Trust == "imported" && len(nonEmpty(source.Imported)) == 0 {
			addReview("imported claim requires imported source")
		}
	}
}

func validateProposedShapeV2(shape ProposedShapeV2, delivery DeliveryV2, addFail func(string), addReview func(string)) {
	if strings.TrimSpace(shape.Kind) == "" || strings.TrimSpace(shape.Type) == "" || strings.TrimSpace(shape.ID) == "" || strings.TrimSpace(shape.Path) == "" {
		addFail("proposed_shape must include kind, type, id, and path")
	}
	if strings.TrimSpace(shape.Status) == "" || strings.TrimSpace(shape.Priority) == "" {
		addFail("proposed_shape must include status and priority")
	}
	if strings.TrimSpace(shape.Priority) == "must" && !delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	if strings.TrimSpace(shape.Rationale) == "" {
		addReview("proposed_shape.rationale should explain item or package choice")
	}
}

func validateFutureUseV2(future FutureUseV2, addFail func(string), addReview func(string)) {
	if len(nonEmpty(future.TriggerRequests)) == 0 && len(nonEmpty(future.QueryPhrases)) == 0 {
		addFail("future_use must include trigger_requests or query_phrases")
	}
	if strings.TrimSpace(future.ExpectedUse) == "" {
		addFail("future_use.expected_use is required")
	}
	if len(nonEmpty(future.NegativeTriggers)) == 0 {
		addReview("future_use.negative_triggers should include at least one boundary")
	}
}

func validateOverlapDecisionV2(overlap OverlapDecisionV2, addFail func(string), addReview func(string)) {
	switch strings.TrimSpace(overlap.Decision) {
	case "create_new", "update_existing", "merge_with_existing", "stop":
	case "unresolved":
		addReview("overlap decision is unresolved")
	default:
		addFail("overlap_decision.decision must be create_new, update_existing, merge_with_existing, stop, or unresolved")
	}
	if strings.TrimSpace(overlap.Reason) == "" {
		addReview("overlap_decision.reason is required")
	}
}

func validateDeliveryV2(delivery DeliveryV2, shape ProposedShapeV2, addFail func(string)) {
	if delivery.Path != "inbox" && delivery.Path != "official_review" {
		addFail("delivery.path must be inbox or official_review")
	}
	if !delivery.WriteRequiresHumanApproval {
		addFail("delivery.write_requires_human_approval must be true")
	}
	if !delivery.ReviewPacketRequired {
		addFail("delivery.review_packet_required must be true")
	}
	if delivery.Path == "official_review" && !delivery.OfficialMutationAuthorized {
		addFail("official mutation requires explicit authorization")
	}
	if strings.TrimSpace(shape.Priority) == "must" && !delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
}

func validateCandidateAndPlanV2(proposal ProposalV2, addFail func(string), addReview func(string)) {
	if len(proposal.CandidateFiles) == 0 {
		addFail("candidate_files must include at least one file")
	}
	for i, file := range proposal.CandidateFiles {
		if strings.TrimSpace(file.Path) == "" || strings.TrimSpace(file.Purpose) == "" || strings.TrimSpace(file.Load) == "" {
			addFail(fmt.Sprintf("candidate_files[%d] must include path, purpose, and load", i))
		}
	}
	if strings.TrimSpace(proposal.VerificationPlan.ValidatePath) == "" {
		addFail("verification_plan.validate_path is required")
	}
	if len(proposal.VerificationPlan.FindabilityScenarios) == 0 {
		addReview("verification_plan.findability_scenarios should include at least one scenario")
	}
}

func hasAnySourceV2(source SourceProfileV2) bool {
	return len(nonEmpty(source.UserConfirmed)) > 0 ||
		len(nonEmpty(source.Observed)) > 0 ||
		len(nonEmpty(source.Imported)) > 0 ||
		len(nonEmpty(source.Synthesized)) > 0 ||
		len(nonEmpty(source.Templates)) > 0 ||
		len(nonEmpty(source.Examples)) > 0
}
```

- [ ] **Step 4: Run the v2 proposal tests**

Run:

```bash
go test ./internal/author -run 'TestValidateProposalV2' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit task 1**

Run:

```bash
git add internal/author/proposal_v2.go internal/author/proposal_v2_test.go
git commit -m "feat: add authoring proposal v2 contract"
```

---

### Task 2: Verification Supports V2 Proposals

**Files:**
- Modify: `internal/author/proposal_v2.go`
- Modify: `internal/author/verify.go`
- Modify: `internal/author/verify_test.go`

- [ ] **Step 1: Add failing v2 verify tests**

Append to `internal/author/verify_test.go`:

```go
func TestVerifyAcceptsV2ProposalAndValidInboxCandidate(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/packages/backend/go-service-template"
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", authorPackage("package:backend.go-service-template.v1", "draft", "Go Service Template Knowledge"))
	proposalPath := "knowledge/.inbox/proposals/go-service-template/proposal.json"
	writeProposalV2(t, root, proposalPath, validVerifyProposalV2(candidatePath))

	result, err := Verify(root, VerifyRequest{ProposalPath: proposalPath, CandidatePath: candidatePath})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "pass" {
		t.Fatalf("expected pass result, got %#v", result)
	}
	if result.Proposal.Validation != "pass" {
		t.Fatalf("expected proposal pass, got %#v", result.Proposal)
	}
	if result.Candidate.Validation != "pass" {
		t.Fatalf("expected candidate pass, got %#v", result.Candidate)
	}
}

func TestVerifyV2ReturnsReviewNeededForUnresolvedOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/packages/backend/go-service-template"
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", authorPackage("package:backend.go-service-template.v1", "draft", "Go Service Template Knowledge"))
	proposalPath := "knowledge/.inbox/proposals/go-service-template/proposal.json"
	proposal := validVerifyProposalV2(candidatePath)
	proposal.OverlapDecision.Decision = "unresolved"
	proposal.OverlapDecision.HumanChoiceRequired = true
	writeProposalV2(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{ProposalPath: proposalPath, CandidatePath: candidatePath})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed, got %#v", result)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "overlap decision is unresolved") {
		t.Fatalf("expected unresolved overlap finding, got %#v", result.Findings)
	}
}

func TestVerifyV2RejectsUnauthorizedPriorityMust(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/packages/backend/go-service-template"
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", authorPackage("package:backend.go-service-template.v1", "draft", "Go Service Template Knowledge"))
	proposalPath := "knowledge/.inbox/proposals/go-service-template/proposal.json"
	proposal := validVerifyProposalV2(candidatePath)
	proposal.ProposedShape.Priority = "must"
	proposal.Delivery.PriorityMustAuthorized = false
	writeProposalV2(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{ProposalPath: proposalPath, CandidatePath: candidatePath})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected fail, got %#v", result)
	}
	if !hasVerifyFinding(result.Findings, "fail", "priority: must") {
		t.Fatalf("expected priority must finding, got %#v", result.Findings)
	}
}
```

Also add helpers to the same test file:

```go
func writeProposalV2(t *testing.T, root, rel string, proposal ProposalV2) {
	t.Helper()
	data, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatalf("marshal v2 proposal: %v", err)
	}
	writeAuthorFile(t, root, rel, string(data))
}

func validVerifyProposalV2(candidatePath string) ProposalV2 {
	proposal := validProposalV2()
	proposal.ProposedShape.Path = candidatePath
	proposal.VerificationPlan.ValidatePath = candidatePath
	proposal.CandidateFiles = []CandidateFile{{Path: candidatePath + "/KNOWLEDGE.md", Purpose: "entrypoint", Load: "start_here"}}
	return proposal
}

func authorPackage(id, status, title string) string {
	return `---
id: ` + id + `
title: ` + title + `
type: package
tech_domains: [backend]
business_domains: [catalog]
tags: [service-template]
projects: [mall-api]
status: ` + status + `
priority: should
updated_at: 2026-05-02
applies_to:
  files:
    - templates/go-service/**
---
## Purpose

Guide future agents when generating Go services.

## Start Here

Use this package before creating a new Go service.

## Load On Demand

- examples/template.md
`
}
```

- [ ] **Step 2: Run the verify tests and confirm they fail**

Run:

```bash
go test ./internal/author -run 'TestVerify.*V2' -count=1
```

Expected: FAIL because `Verify` still expects v1 proposals.

- [ ] **Step 3: Add v2 normalization**

Append to `internal/author/proposal_v2.go`:

```go
func NormalizeProposalV2(proposal ProposalV2) Proposal {
	return Proposal{
		SchemaVersion: ProposalSchemaVersion,
		KnowledgeGoal: proposal.KnowledgeGoal,
		AuthoringMode: "synthesized",
		Project: proposal.Project,
		ProposedShape: ProposedShape{
			Kind:     proposal.ProposedShape.Kind,
			Type:     proposal.ProposedShape.Type,
			Title:    proposal.ProposedShape.Title,
			ID:       proposal.ProposedShape.ID,
			Path:     proposal.ProposedShape.Path,
			Status:   proposal.ProposedShape.Status,
			Priority: proposal.ProposedShape.Priority,
		},
		FutureRetrievalContract: FutureRetrievalContract{
			Tasks:           append([]string{}, proposal.FutureUse.TriggerRequests...),
			Phases:          append([]string{}, proposal.FutureUse.Phases...),
			Files:           append([]string{}, proposal.Scope.Files...),
			FileGlobs:       append([]string{}, proposal.Scope.FileGlobs...),
			QueryPhrases:    append([]string{}, proposal.FutureUse.QueryPhrases...),
			Projects:        append([]string{}, proposal.Scope.Projects...),
			TechDomains:     append([]string{}, proposal.Scope.TechDomains...),
			BusinessDomains: nil,
			Tags:            append([]string{}, proposal.Scope.SubjectDomains...),
			ExpectedUse:     proposal.FutureUse.ExpectedUse,
			CiteAfterUse:    proposal.FutureUse.CitationPolicy == "cite_after_use",
		},
		SourceAndTrust: SourceAndTrust{
			UserProvided:  append([]string{}, proposal.SourceProfile.UserConfirmed...),
			Observed:      append([]string{}, proposal.SourceProfile.Observed...),
			Imported:      append([]string{}, proposal.SourceProfile.Imported...),
			Synthesized:   append([]string{}, proposal.SourceProfile.Synthesized...),
			Assumptions:   append([]string{}, proposal.SourceProfile.Assumptions...),
			OpenQuestions: append([]string{}, proposal.SourceProfile.OpenQuestions...),
		},
		Applicability: proposal.Applicability,
		OverlapDecision: OverlapDecision{
			OfficialOverlap: append([]string{}, proposal.OverlapDecision.OfficialOverlap...),
			InboxOverlap:    append([]string{}, proposal.OverlapDecision.InboxOverlap...),
			Decision:        normalizeOverlapDecisionV2(proposal.OverlapDecision.Decision),
			Reason:          proposal.OverlapDecision.Reason,
		},
		Delivery: Delivery{
			Path:                   proposal.Delivery.Path,
			RequiresHumanApproval:  proposal.Delivery.WriteRequiresHumanApproval,
			PriorityMustAuthorized: proposal.Delivery.PriorityMustAuthorized,
			PromoteAuthorized:      proposal.Delivery.PromoteAuthorized,
		},
		CandidateFiles:   append([]CandidateFile{}, proposal.CandidateFiles...),
		VerificationPlan: proposal.VerificationPlan,
	}
}

func normalizeOverlapDecisionV2(decision string) string {
	switch decision {
	case "merge_with_existing":
		return "update_existing"
	default:
		return decision
	}
}
```

- [ ] **Step 4: Update `Verify` to load either schema**

Modify the proposal loading portion in `internal/author/verify.go` by adding this helper:

```go
func loadProposalForVerify(path string) (Proposal, []Finding, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Proposal{}, nil, "", fmt.Errorf("read proposal: %w", err)
	}
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return Proposal{}, nil, "", fmt.Errorf("parse proposal JSON: %w", err)
	}
	switch strings.TrimSpace(header.SchemaVersion) {
	case ProposalSchemaVersion:
		var proposal Proposal
		if err := json.Unmarshal(data, &proposal); err != nil {
			return Proposal{}, nil, "", fmt.Errorf("parse proposal JSON: %w", err)
		}
		return proposal, ValidateProposal(proposal), ProposalSchemaVersion, nil
	case ProposalV2SchemaVersion:
		var proposal ProposalV2
		if err := json.Unmarshal(data, &proposal); err != nil {
			return Proposal{}, nil, "", fmt.Errorf("parse proposal JSON: %w", err)
		}
		return NormalizeProposalV2(proposal), ValidateProposalV2(proposal), ProposalV2SchemaVersion, nil
	default:
		return Proposal{}, []Finding{{Severity: "fail", Message: "schema_version must be authoring.proposal.v1 or authoring.proposal.v2"}}, strings.TrimSpace(header.SchemaVersion), nil
	}
}
```

Add `encoding/json` to the imports in `internal/author/verify.go`.

Then replace:

```go
proposal, err := LoadProposal(filepath.Join(root, proposalPath))
if err != nil {
	return VerifyResponse{}, err
}
```

with:

```go
proposal, proposalFindings, _, err := loadProposalForVerify(filepath.Join(root, proposalPath))
if err != nil {
	return VerifyResponse{}, err
}
```

Remove the later line that recalculates `proposalFindings := ValidateProposal(proposal)` so the v2 findings are preserved.

- [ ] **Step 5: Run focused author tests**

Run:

```bash
go test ./internal/author -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit task 2**

Run:

```bash
git add internal/author/proposal_v2.go internal/author/verify.go internal/author/verify_test.go
git commit -m "feat: verify authoring proposal v2"
```

---

### Task 3: Authoring Harness Case Model And Fixtures

**Files:**
- Create: `internal/authoringdogfood/cases.go`
- Create: `internal/authoringdogfood/authoringdogfood_test.go`
- Replace: `testdata/authoring-golden/cases.json`
- Delete: `internal/authoringtest/cases.go`
- Delete: `internal/authoringtest/cases_test.go`

- [ ] **Step 1: Write failing case loader tests**

Create `internal/authoringdogfood/authoringdogfood_test.go`:

```go
package authoringdogfood

import (
	"encoding/json"
	"strings"
	"testing"
)

const authoringCasesPath = "../../testdata/authoring-golden/cases.json"

func TestLoadCasesIncludesRealScenarioMatrix(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"go_template_standard": false,
		"redis_best_practices": false,
		"api_consumer_knowledge": false,
		"observed_repo_lesson": false,
		"overlap_requires_choice": false,
		"candidate_not_findable": false,
		"unauthorized_authority": false,
		"personal_project_convention": false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing authoring case kind %s", kind)
		}
	}
}

func TestSummariesExposeNaturalInputAndHideOracle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	text := string(data)
	for _, want := range []string{"case-001", "user_request", "I designed a Go service template"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q: %s", want, text)
		}
	}
	for _, forbidden := range []string{
		"expected",
		"oracle",
		"required_guards",
		"go_template_standard",
		"candidate_write_approved",
		"proposal_must_precede_candidate",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}
}

func TestFindCaseAcceptsPublicHandle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, publicID, err := FindCase(cases, "case-001")
	if err != nil {
		t.Fatalf("FindCase returned error: %v", err)
	}
	if publicID != "case-001" {
		t.Fatalf("publicID = %q, want case-001", publicID)
	}
	if !strings.Contains(tc.Input.UserRequest, "Go service template") {
		t.Fatalf("unexpected case input: %#v", tc.Input)
	}
}
```

- [ ] **Step 2: Run the new tests and confirm they fail**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestLoadCases|TestSummaries|TestFindCase' -count=1
```

Expected: FAIL because `internal/authoringdogfood` does not exist.

- [ ] **Step 3: Implement the case model**

Create `internal/authoringdogfood/cases.go`:

```go
package authoringdogfood

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type CaseFile struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Fixture   string   `json:"fixture"`
	Input     Input    `json:"input"`
	Approval  Approval `json:"approval"`
	Oracle    Oracle   `json:"oracle"`
}

type Input struct {
	UserRequest      string   `json:"user_request"`
	Project          string   `json:"project"`
	ContextHints     []string `json:"context_hints"`
	AvailableSources []Source `json:"available_sources"`
	Constraints      []string `json:"constraints"`
}

type Source struct {
	Kind        string `json:"kind"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description"`
}

type Approval struct {
	ProposalApproved           bool     `json:"proposal_approved"`
	CandidateWriteApproved     bool     `json:"candidate_write_approved"`
	PriorityMustAuthorized     bool     `json:"priority_must_authorized"`
	OfficialMutationAuthorized bool     `json:"official_mutation_authorized"`
	PromoteAuthorized          bool     `json:"promote_authorized"`
	Notes                      []string `json:"notes"`
}

type Oracle struct {
	ExpectedResult              string   `json:"expected_result"`
	RequiredGuards              []string `json:"required_guards"`
	RequiredProposalProperties  []string `json:"required_proposal_properties"`
	ForbiddenMutations          []string `json:"forbidden_mutations"`
	RequiredEvidenceCategories  []string `json:"required_evidence_categories"`
}

type Summary struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Fixture string `json:"fixture"`
	Input   Input  `json:"input"`
}

func LoadCases(path string) ([]Case, error) {
	fileHandle, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read authoring cases %q: %w", path, err)
	}
	defer fileHandle.Close()

	var file CaseFile
	decoder := json.NewDecoder(fileHandle)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
		}
		return nil, fmt.Errorf("parse authoring cases %q: multiple JSON values", path)
	}
	if err := validateCaseFile(file); err != nil {
		return nil, fmt.Errorf("validate authoring cases %q: %w", path, err)
	}
	return file.Cases, nil
}

func Summaries(cases []Case) []Summary {
	summaries := make([]Summary, 0, len(cases))
	for i, tc := range cases {
		summaries = append(summaries, Summary{
			ID:      publicCaseID(i),
			Kind:    publicKind(tc.Kind),
			Fixture: fixtureName(tc.Fixture),
			Input:   tc.Input,
		})
	}
	return summaries
}

func FindCase(cases []Case, id string) (Case, string, error) {
	id = strings.TrimSpace(id)
	for i, tc := range cases {
		publicID := publicCaseID(i)
		if id == publicID || id == tc.ID {
			return tc, publicID, nil
		}
	}
	return Case{}, "", fmt.Errorf("unknown authoring case %q", id)
}

func publicCaseID(index int) string {
	return fmt.Sprintf("case-%03d", index+1)
}

func publicKind(kind string) string {
	switch kind {
	case "go_template_standard":
		return "template_standard"
	case "redis_best_practices":
		return "best_practices"
	case "api_consumer_knowledge":
		return "consumer_reference"
	case "observed_repo_lesson":
		return "observed_lesson"
	case "overlap_requires_choice":
		return "overlap"
	case "candidate_not_findable":
		return "findability"
	case "unauthorized_authority":
		return "authority"
	case "personal_project_convention":
		return "personal_convention"
	default:
		return "authoring"
	}
}

func fixtureName(fixture string) string {
	if strings.TrimSpace(fixture) == "" {
		return "full"
	}
	return strings.TrimSpace(fixture)
}
```

Add validation helpers in the same file:

```go
func validateCaseFile(file CaseFile) error {
	if len(file.Cases) == 0 {
		return fmt.Errorf("cases list is empty")
	}
	for i, tc := range file.Cases {
		context := fmt.Sprintf("case[%d] id %q", i, tc.ID)
		if strings.TrimSpace(tc.ID) == "" {
			return fmt.Errorf("case[%d]: id is required", i)
		}
		if strings.TrimSpace(tc.Kind) == "" {
			return fmt.Errorf("%s: kind is required", context)
		}
		if strings.TrimSpace(tc.Input.UserRequest) == "" {
			return fmt.Errorf("%s: input.user_request is required", context)
		}
		if strings.TrimSpace(tc.Input.Project) == "" {
			return fmt.Errorf("%s: input.project is required", context)
		}
		if len(tc.Oracle.RequiredGuards) == 0 {
			return fmt.Errorf("%s: oracle.required_guards is required", context)
		}
		if strings.TrimSpace(tc.Oracle.ExpectedResult) == "" {
			return fmt.Errorf("%s: oracle.expected_result is required", context)
		}
	}
	return nil
}
```

- [ ] **Step 4: Replace the authoring golden fixture**

Replace `testdata/authoring-golden/cases.json` with this structure:

```json
{
  "cases": [
    {
      "id": "go_template_standard",
      "kind": "go_template_standard",
      "fixture": "full",
      "input": {
        "user_request": "I designed a Go service template. Turn it into reusable knowledge so future agents write Go services in this style.",
        "project": "mall-api",
        "context_hints": ["go", "service-template", "backend"],
        "available_sources": [
          {"kind": "file", "path": "templates/go-service", "description": "User-provided Go service template"}
        ],
        "constraints": ["Candidate knowledge must stay in inbox.", "Examples are template material only after review."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Proposal approved for inbox draft only."]
      },
      "oracle": {
        "expected_result": "pass",
        "required_guards": ["proposal_before_write", "source_profile", "future_use", "candidate_inbox_only"],
        "required_proposal_properties": ["audience.implementer_agent", "source_profile.templates", "future_use.trigger_requests"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["user_confirmed", "template"]
      }
    },
    {
      "id": "redis_best_practices",
      "kind": "redis_best_practices",
      "fixture": "full",
      "input": {
        "user_request": "We use Redis heavily, but I am not an expert. Help me and the agent design safe draft practices for future agents.",
        "project": "mall-api",
        "context_hints": ["redis", "cache", "best-practices"],
        "available_sources": [],
        "constraints": ["Synthesized recommendations must stay draft.", "Do not use priority must."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Draft candidate approved if assumptions and open questions are visible."]
      },
      "oracle": {
        "expected_result": "pass",
        "required_guards": ["assumptions_visible", "no_unauthorized_priority_must", "candidate_inbox_only"],
        "required_proposal_properties": ["source_profile.synthesized", "source_profile.assumptions", "scope.stability.draft"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["synthesized", "assumption"]
      }
    },
    {
      "id": "api_consumer_knowledge",
      "kind": "api_consumer_knowledge",
      "fixture": "full",
      "input": {
        "user_request": "This project is called by many people. I want future agents to understand the business points and interfaces so they can help other developers use it.",
        "project": "mall-api",
        "context_hints": ["api", "consumer", "business-capability"],
        "available_sources": [
          {"kind": "file", "path": "internal/api", "description": "API interface code"}
        ],
        "constraints": ["Separate API facts from user interpretation."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Consumer-facing draft candidate is approved."]
      },
      "oracle": {
        "expected_result": "pass",
        "required_guards": ["consumer_audience", "source_profile", "candidate_inbox_only"],
        "required_proposal_properties": ["audience.consumer_agent", "source_profile.observed", "future_use.negative_triggers"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["observed", "user_confirmed"]
      }
    },
    {
      "id": "observed_repo_lesson",
      "kind": "observed_repo_lesson",
      "fixture": "full",
      "input": {
        "user_request": "Turn the retry handling pattern in the repository into reusable knowledge for future agents.",
        "project": "mall-api",
        "context_hints": ["retry", "observed-pattern"],
        "available_sources": [
          {"kind": "file", "path": "internal/retry", "description": "Retry implementation files"}
        ],
        "constraints": ["Record evidence paths and separate assumptions."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Observed lesson candidate is approved."]
      },
      "oracle": {
        "expected_result": "pass",
        "required_guards": ["observed_evidence_paths", "assumptions_separated", "candidate_inbox_only"],
        "required_proposal_properties": ["source_profile.observed", "source_profile.claims"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["observed"]
      }
    },
    {
      "id": "overlap_requires_choice",
      "kind": "overlap_requires_choice",
      "fixture": "full",
      "input": {
        "user_request": "Create another cache TTL rule that may overlap existing cache knowledge.",
        "project": "mall-api",
        "context_hints": ["cache", "ttl", "overlap"],
        "available_sources": [],
        "constraints": ["Do not write a candidate while overlap is unresolved."]
      },
      "approval": {
        "proposal_approved": false,
        "candidate_write_approved": false,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Overlap decision is unresolved."]
      },
      "oracle": {
        "expected_result": "review-needed",
        "required_guards": ["overlap_checked", "no_write_before_decision"],
        "required_proposal_properties": ["overlap_decision.unresolved"],
        "forbidden_mutations": ["candidate_write", "official_write", "promotion"],
        "required_evidence_categories": ["overlap"]
      }
    },
    {
      "id": "candidate_not_findable",
      "kind": "candidate_not_findable",
      "fixture": "full",
      "input": {
        "user_request": "Create knowledge whose metadata is too weak to be found for the intended future task.",
        "project": "mall-api",
        "context_hints": ["findability", "metadata"],
        "available_sources": [],
        "constraints": ["Report review-needed when verification cannot find the candidate."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Candidate may be written to demonstrate findability failure."]
      },
      "oracle": {
        "expected_result": "review-needed",
        "required_guards": ["author_verify_run", "findability_reported"],
        "required_proposal_properties": ["future_use.trigger_requests"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["verification"]
      }
    },
    {
      "id": "unauthorized_authority",
      "kind": "unauthorized_authority",
      "fixture": "full",
      "input": {
        "user_request": "Create a must-follow engineering rule for future agents.",
        "project": "mall-api",
        "context_hints": ["policy", "priority-must"],
        "available_sources": [],
        "constraints": ["The user has not authorized priority must or official mutation."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Must authority is not authorized."]
      },
      "oracle": {
        "expected_result": "review-needed",
        "required_guards": ["priority_must_authorization_checked", "human_authorization_required"],
        "required_proposal_properties": ["delivery.priority_must_authorized.false"],
        "forbidden_mutations": ["official_write", "promotion", "priority_must"],
        "required_evidence_categories": ["human_review"]
      }
    },
    {
      "id": "personal_project_convention",
      "kind": "personal_project_convention",
      "fixture": "full",
      "input": {
        "user_request": "I have a personal convention for this project. Preserve it for future agents without making it global truth.",
        "project": "mall-api",
        "context_hints": ["personal", "project-convention"],
        "available_sources": [],
        "constraints": ["Keep scope narrow and draft unless reviewed."]
      },
      "approval": {
        "proposal_approved": true,
        "candidate_write_approved": true,
        "priority_must_authorized": false,
        "official_mutation_authorized": false,
        "promote_authorized": false,
        "notes": ["Personal project convention may be written as a draft inbox candidate."]
      },
      "oracle": {
        "expected_result": "pass",
        "required_guards": ["scope_not_global", "candidate_inbox_only"],
        "required_proposal_properties": ["scope.distribution.personal", "scope.projects"],
        "forbidden_mutations": ["official_write", "promotion"],
        "required_evidence_categories": ["user_confirmed"]
      }
    }
  ]
}
```

- [ ] **Step 5: Delete the old seed-only package**

Run:

```bash
git rm internal/authoringtest/cases.go internal/authoringtest/cases_test.go
```

Expected: files are removed because `internal/authoringdogfood` now owns authoring harness behavior.

- [ ] **Step 6: Run authoring dogfood package tests**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit task 3**

Run:

```bash
git add internal/authoringdogfood/cases.go internal/authoringdogfood/authoringdogfood_test.go testdata/authoring-golden/cases.json
git commit -m "feat: add natural authoring dogfood cases"
```

---

### Task 4: Authoring Packet, Report Parser, And Evaluator

**Files:**
- Create: `internal/authoringdogfood/packet.go`
- Create: `internal/authoringdogfood/report.go`
- Create: `internal/authoringdogfood/evaluate.go`
- Modify: `internal/authoringdogfood/authoringdogfood_test.go`

- [ ] **Step 1: Add failing packet, report, and evaluator tests**

Append to `internal/authoringdogfood/authoringdogfood_test.go`:

```go
func TestBuildPacketIncludesNaturalRequestAndAuthoringCommands(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	packet, err := BuildPacket(cases, PacketOptions{CaseID: "case-001", Workspace: "/tmp/argos-authoring", ArgosBinary: "/tmp/argos"})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}

	text := packet.Markdown
	for _, want := range []string{
		"Case: `case-001`",
		"Workspace: `/tmp/argos-authoring`",
		"I designed a Go service template",
		"/tmp/argos author inspect --json",
		"/tmp/argos author verify --json --proposal <proposal.json> --path <candidate>",
		"Include these sections: `## Inputs`, `## Tool Transcript Summary`, `## Artifacts`, `## Human Review Decisions`, `## Guards`, and `## Result`.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("packet missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"go_template_standard", "required_guards", "expected_result", "proposal_must_precede_candidate"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("packet leaked %q:\n%s", forbidden, text)
		}
	}
}

func TestParseMarkdownReportExtractsAuthoringArtifacts(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	if report.CaseID != "case-001" {
		t.Fatalf("unexpected case id: %q", report.CaseID)
	}
	if report.ProposalPath != "knowledge/.inbox/proposals/go-service-template/proposal.json" {
		t.Fatalf("unexpected proposal path: %q", report.ProposalPath)
	}
	if report.CandidatePath != "knowledge/.inbox/packages/backend/go-service-template" {
		t.Fatalf("unexpected candidate path: %q", report.CandidatePath)
	}
	if report.VerifyResult != "pass" {
		t.Fatalf("unexpected verify result: %q", report.VerifyResult)
	}
	if report.Guards["proposal before write"] != "pass" {
		t.Fatalf("unexpected guards: %#v", report.Guards)
	}
}

func TestEvaluateCaseRequiresWorkspaceArtifacts(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	root := t.TempDir()
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-001", root, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}
	if evaluation.Result != ResultFail {
		t.Fatalf("expected fail when artifacts are missing, got %#v", evaluation)
	}
	if !hasEvaluationFinding(evaluation.Findings, "fail", "proposal artifact is missing") {
		t.Fatalf("expected missing proposal finding, got %#v", evaluation.Findings)
	}
}
```

Add sample helpers in the same file:

```go
func sampleAuthoringReport(caseID, result string) string {
	return `Case: ` + caseID + `

## Inputs

- User request: I designed a Go service template.

## Tool Transcript Summary

- Ran author inspect.
- Wrote proposal after review.
- Ran author verify.

## Artifacts

- Proposal path: knowledge/.inbox/proposals/go-service-template/proposal.json
- Candidate path: knowledge/.inbox/packages/backend/go-service-template
- Verify result: pass

## Human Review Decisions

- Proposal approved: true
- Candidate write approved: true
- Priority must authorized: false
- Official mutation authorized: false
- Promote authorized: false

## Guards

- Proposal before write: pass
- Source profile: pass
- Future use: pass
- Candidate inbox only: pass

## Result

Result: ` + result + `
`
}

func hasEvaluationFinding(findings []Finding, severity, text string) bool {
	for _, finding := range findings {
		if finding.Severity == severity && strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests and confirm they fail**

Run:

```bash
go test ./internal/authoringdogfood -run 'TestBuildPacket|TestParseMarkdownReport|TestEvaluateCase' -count=1
```

Expected: FAIL because packet, report, and evaluator functions do not exist.

- [ ] **Step 3: Implement packet generation**

Create `internal/authoringdogfood/packet.go`:

```go
package authoringdogfood

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PacketOptions struct {
	CaseID      string
	Workspace   string
	ArgosBinary string
}

type Packet struct {
	CaseID      string `json:"case_id"`
	Kind        string `json:"kind"`
	Fixture     string `json:"fixture"`
	Workspace   string `json:"workspace"`
	ArgosBinary string `json:"argos_binary"`
	Input       Input  `json:"input"`
	Markdown    string `json:"markdown"`
}

func BuildPacket(cases []Case, options PacketOptions) (Packet, error) {
	tc, publicID, err := FindCase(cases, options.CaseID)
	if err != nil {
		return Packet{}, fmt.Errorf("build authoring packet: %w", err)
	}
	workspace := strings.TrimSpace(options.Workspace)
	if workspace == "" {
		return Packet{}, fmt.Errorf("build authoring packet: workspace is required")
	}
	argosBinary := strings.TrimSpace(options.ArgosBinary)
	if argosBinary == "" {
		return Packet{}, fmt.Errorf("build authoring packet: argos binary is required")
	}
	packet := Packet{
		CaseID:      publicID,
		Kind:        publicKind(tc.Kind),
		Fixture:     fixtureName(tc.Fixture),
		Workspace:   workspace,
		ArgosBinary: argosBinary,
		Input:       tc.Input,
	}
	packet.Markdown = renderPacketMarkdown(packet, tc.Approval)
	return packet, nil
}

func renderPacketMarkdown(packet Packet, approval Approval) string {
	input, err := json.MarshalIndent(packet.Input, "", "  ")
	if err != nil {
		input = []byte("{}")
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Argos Authoring Dogfood Runner Packet\n\n")
	fmt.Fprintf(&builder, "Case: `%s`\n", packet.CaseID)
	fmt.Fprintf(&builder, "Kind: `%s`\n", packet.Kind)
	fmt.Fprintf(&builder, "Fixture: `%s`\n", packet.Fixture)
	fmt.Fprintf(&builder, "Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Runner Instructions\n\n")
	fmt.Fprintf(&builder, "- Start from the user request below; do not inspect authoring cases or hidden expectations.\n")
	fmt.Fprintf(&builder, "- Produce a Knowledge Design Proposal before writing candidate knowledge.\n")
	fmt.Fprintf(&builder, "- Write candidate files only when simulated approval allows it.\n")
	fmt.Fprintf(&builder, "- Keep candidates in inbox unless approval explicitly allows official mutation.\n")
	fmt.Fprintf(&builder, "- Run author verification for any written candidate.\n\n")

	fmt.Fprintf(&builder, "## Simulated Human Approval\n\n")
	fmt.Fprintf(&builder, "- Proposal approved: %t\n", approval.ProposalApproved)
	fmt.Fprintf(&builder, "- Candidate write approved: %t\n", approval.CandidateWriteApproved)
	fmt.Fprintf(&builder, "- Priority must authorized: %t\n", approval.PriorityMustAuthorized)
	fmt.Fprintf(&builder, "- Official mutation authorized: %t\n", approval.OfficialMutationAuthorized)
	fmt.Fprintf(&builder, "- Promote authorized: %t\n", approval.PromoteAuthorized)
	for _, note := range approval.Notes {
		fmt.Fprintf(&builder, "- Note: %s\n", note)
	}
	fmt.Fprintf(&builder, "\n")

	fmt.Fprintf(&builder, "## CLI Equivalents\n\n")
	fmt.Fprintf(&builder, "```bash\n")
	fmt.Fprintf(&builder, "%s author inspect --json --project <project> --goal \"<knowledge goal>\" [--future-task \"<future task>\"] [--phase <phase>] [--files <path>] [--domains <domain>] [--tags <tag>]\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s author verify --json --proposal <proposal.json> --path <candidate>\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "```\n\n")

	fmt.Fprintf(&builder, "## Input JSON\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", input)

	fmt.Fprintf(&builder, "## Required Report Shape\n\n")
	fmt.Fprintf(&builder, "Include these sections: `## Inputs`, `## Tool Transcript Summary`, `## Artifacts`, `## Human Review Decisions`, `## Guards`, and `## Result`.\n")
	return builder.String()
}
```

- [ ] **Step 4: Implement report parsing**

Create `internal/authoringdogfood/report.go`:

```go
package authoringdogfood

import "strings"

type Report struct {
	CaseID                string            `json:"case_id"`
	ProposalPath          string            `json:"proposal_path"`
	CandidatePath         string            `json:"candidate_path"`
	VerifyResult          string            `json:"verify_result"`
	HumanReviewDecisions  map[string]string `json:"human_review_decisions"`
	Guards                map[string]string `json:"guards"`
	Result                string            `json:"result"`
	MissingSections       []string          `json:"missing_sections"`
	MissingFields         []string          `json:"missing_fields"`
}

var requiredSections = []string{"inputs", "tool transcript summary", "artifacts", "human review decisions", "guards", "result"}

func ParseMarkdownReport(data []byte) (Report, error) {
	text := string(data)
	report := Report{
		CaseID:               parseCaseID(text),
		HumanReviewDecisions: map[string]string{},
		Guards:               map[string]string{},
	}
	sections := parseSections(text)
	for _, section := range requiredSections {
		if _, ok := sections[section]; !ok {
			report.MissingSections = append(report.MissingSections, section)
		}
	}
	parseArtifacts(sections["artifacts"], &report)
	parseDecisions(sections["human review decisions"], &report)
	parseGuards(sections["guards"], &report)
	report.Result = parseResult(sections["result"])
	report.MissingFields = missingFields(report)
	return report, nil
}
```

Add parser helpers in the same file:

```go
func parseCaseID(text string) string {
	for _, line := range strings.Split(text, "\n") {
		label, value, ok := splitLabel(line)
		if ok && strings.EqualFold(label, "case") {
			return cleanValue(value)
		}
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			return ""
		}
	}
	return ""
}

func parseSections(text string) map[string]string {
	sections := map[string]string{}
	var current string
	var builder strings.Builder
	flush := func() {
		if current != "" {
			sections[current] = strings.TrimSpace(builder.String())
			builder.Reset()
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			current = normalize(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			continue
		}
		if current != "" {
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
	}
	flush()
	return sections
}

func parseArtifacts(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitBullet(line)
		if !ok {
			continue
		}
		switch normalize(label) {
		case "proposal path":
			report.ProposalPath = cleanValue(value)
		case "candidate path":
			report.CandidatePath = cleanValue(value)
		case "verify result":
			report.VerifyResult = cleanStatus(value)
		}
	}
}

func parseDecisions(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitBullet(line)
		if ok {
			report.HumanReviewDecisions[normalize(label)] = cleanValue(value)
		}
	}
}

func parseGuards(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitBullet(line)
		if ok {
			report.Guards[normalize(label)] = cleanStatus(value)
		}
	}
}

func parseResult(section string) string {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitLabel(line)
		if ok && strings.EqualFold(label, "result") {
			return cleanStatus(value)
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return cleanStatus(trimmed)
		}
	}
	return ""
}
```

Add string helpers:

```go
func splitBullet(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "- ") {
		return "", "", false
	}
	return splitLabel(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
}

func splitLabel(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func normalize(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	if strings.EqualFold(value, "none") || value == "-" {
		return ""
	}
	return value
}

func cleanStatus(value string) string {
	value = strings.ToLower(cleanValue(value))
	for _, status := range []string{"pass", "fail", "review-needed", "not-applicable", "not-run"} {
		if strings.Contains(value, status) {
			return status
		}
	}
	return value
}

func missingFields(report Report) []string {
	var missing []string
	if report.ProposalPath == "" {
		missing = append(missing, "proposal path")
	}
	if report.Result == "" {
		missing = append(missing, "result")
	}
	return missing
}
```

- [ ] **Step 5: Implement workspace-aware evaluation**

Create `internal/authoringdogfood/evaluate.go`:

```go
package authoringdogfood

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/author"
)

const (
	ResultPass         = "pass"
	ResultFail         = "fail"
	ResultReviewNeeded = "review-needed"
)

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Evaluation struct {
	CaseID   string    `json:"case_id"`
	Result   string    `json:"result"`
	Findings []Finding `json:"findings,omitempty"`
}

func EvaluateCase(cases []Case, requestedID string, workspace string, report Report) (Evaluation, error) {
	tc, publicID, err := FindCase(cases, requestedID)
	if err != nil {
		return Evaluation{}, fmt.Errorf("evaluate authoring report: %w", err)
	}
	evaluation := Evaluation{CaseID: publicID, Result: ResultPass}
	addFail := func(message string) {
		evaluation.Result = ResultFail
		evaluation.Findings = append(evaluation.Findings, Finding{Severity: ResultFail, Message: message})
	}
	addReview := func(message string) {
		if evaluation.Result == ResultPass {
			evaluation.Result = ResultReviewNeeded
		}
		evaluation.Findings = append(evaluation.Findings, Finding{Severity: ResultReviewNeeded, Message: message})
	}

	if report.CaseID != "" && report.CaseID != publicID && report.CaseID != tc.ID {
		addFail("report case id does not match requested case")
	}
	for _, section := range report.MissingSections {
		addReview("missing section: " + section)
	}
	for _, field := range report.MissingFields {
		addReview("missing field: " + field)
	}
	if report.Result == ResultFail {
		addFail("runner reported result: fail")
	}
	checkWorkspaceArtifacts(workspace, tc, report, addFail, addReview)
	if tc.Oracle.ExpectedResult != "" && evaluation.Result == ResultPass && report.Result == ResultReviewNeeded {
		addReview("runner reported review-needed")
	}
	if tc.Oracle.ExpectedResult == ResultReviewNeeded && evaluation.Result == ResultPass {
		addReview("case expected review-needed but evaluator found pass")
	}
	if tc.Oracle.ExpectedResult == ResultFail && evaluation.Result != ResultFail {
		addFail("case expected fail")
	}
	return evaluation, nil
}
```

Add evaluator helpers:

```go
func checkWorkspaceArtifacts(workspace string, tc Case, report Report, addFail func(string), addReview func(string)) {
	root := strings.TrimSpace(workspace)
	if root == "" {
		addFail("workspace is required")
		return
	}
	if report.ProposalPath == "" {
		addFail("proposal artifact is missing")
		return
	}
	proposalPath, err := safeJoin(root, report.ProposalPath)
	if err != nil {
		addFail(err.Error())
		return
	}
	if _, err := os.Stat(proposalPath); err != nil {
		addFail("proposal artifact is missing")
		return
	}
	proposal, err := author.LoadProposalV2(proposalPath)
	if err != nil {
		addFail("load v2 proposal: " + err.Error())
		return
	}
	for _, finding := range author.ValidateProposalV2(proposal) {
		if finding.Severity == ResultFail {
			addFail("proposal verification failed: " + finding.Message)
		} else {
			addReview("proposal review needed: " + finding.Message)
		}
	}
	if !tc.Approval.CandidateWriteApproved && report.CandidatePath != "" {
		addFail("candidate write was not approved")
		return
	}
	if report.CandidatePath == "" {
		if tc.Approval.CandidateWriteApproved {
			addReview("candidate path is missing")
		}
		return
	}
	if _, err := safeJoin(root, report.CandidatePath); err != nil {
		addFail(err.Error())
		return
	}
	if !tc.Approval.OfficialMutationAuthorized && isOfficialPath(report.CandidatePath) {
		addFail("official mutation was not authorized")
	}
	if !tc.Approval.PromoteAuthorized && pathExists(filepath.Join(root, "knowledge/packages")) && strings.Contains(strings.Join(tc.Oracle.ForbiddenMutations, ","), "promotion") && isOfficialPath(report.CandidatePath) {
		addFail("promotion was not authorized")
	}
	verifyResult, err := author.Verify(root, author.VerifyRequest{ProposalPath: report.ProposalPath, CandidatePath: report.CandidatePath})
	if err != nil {
		addFail("author verify failed: " + err.Error())
		return
	}
	if report.VerifyResult != "" && report.VerifyResult != verifyResult.Result {
		addReview("reported verify result does not match workspace verification")
	}
	for _, finding := range verifyResult.Findings {
		if finding.Severity == ResultFail {
			addFail("author verify finding: " + finding.Message)
		} else {
			addReview("author verify finding: " + finding.Message)
		}
	}
}

func safeJoin(root string, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("relative path is required")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("artifact path must be relative")
	}
	clean := filepath.Clean(rel)
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if part == ".." {
			return "", fmt.Errorf("artifact path must stay inside workspace")
		}
	}
	return filepath.Join(root, clean), nil
}

func isOfficialPath(path string) bool {
	slash := filepath.ToSlash(path)
	return strings.HasPrefix(slash, "knowledge/items/") || strings.HasPrefix(slash, "knowledge/packages/")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

- [ ] **Step 6: Run authoring dogfood tests**

Run:

```bash
go test ./internal/authoringdogfood -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit task 4**

Run:

```bash
git add internal/authoringdogfood/packet.go internal/authoringdogfood/report.go internal/authoringdogfood/evaluate.go internal/authoringdogfood/authoringdogfood_test.go
git commit -m "feat: evaluate authoring dogfood reports"
```

---

### Task 5: CLI Wiring For `argos dogfood authoring`

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add failing CLI tests**

Add `authoringdogfood` to imports in `internal/cli/cli_test.go` only if direct helper types are needed. Then append tests:

```go
func TestRunDogfoodAuthoringCasesReturnsNaturalPublicInput(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "authoring", "cases", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "I designed a Go service template") {
		t.Fatalf("expected natural authoring request, got %s", stdout.String())
	}
	for _, forbidden := range []string{"expected_result", "required_guards", "go_template_standard"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("authoring cases leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDogfoodAuthoringPacketReturnsMarkdown(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "authoring", "packet",
		"--case", "case-001",
		"--workspace", "/tmp/argos-authoring",
		"--argos-binary", "/tmp/argos",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"Argos Authoring Dogfood Runner Packet", "author inspect --json", "author verify --json"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("packet missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunDogfoodAuthoringEvaluateRequiresWorkspaceBackedArtifacts(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "report.md")
	writeCLIFile(t, root, "report.md", `Case: case-001

## Inputs

- User request: I designed a Go service template.

## Tool Transcript Summary

- Ran author inspect.

## Artifacts

- Proposal path: knowledge/.inbox/proposals/go-service-template/proposal.json
- Candidate path: knowledge/.inbox/packages/backend/go-service-template
- Verify result: pass

## Human Review Decisions

- Proposal approved: true
- Candidate write approved: true
- Priority must authorized: false
- Official mutation authorized: false
- Promote authorized: false

## Guards

- Proposal before write: pass

## Result

Result: pass
`)
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "authoring", "evaluate",
		"--case", "case-001",
		"--report", reportPath,
		"--workspace", root,
		"--json",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"result":"fail"`) || !strings.Contains(stdout.String(), "proposal artifact is missing") {
		t.Fatalf("expected missing artifact failure, got %s", stdout.String())
	}
}

func TestRunDogfoodAuthoringRequiresFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "cases json", args: []string{"dogfood", "authoring", "cases"}, want: "dogfood authoring cases: --json is required"},
		{name: "packet case", args: []string{"dogfood", "authoring", "packet", "--workspace", "/tmp/ws", "--argos-binary", "/tmp/argos"}, want: "dogfood authoring packet: --case is required"},
		{name: "evaluate workspace", args: []string{"dogfood", "authoring", "evaluate", "--json", "--case", "case-001", "--report", "report.md"}, want: "dogfood authoring evaluate: --workspace is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run CLI tests and confirm they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunDogfoodAuthoring' -count=1
```

Expected: FAIL because `dogfood authoring` is not wired.

- [ ] **Step 3: Wire the nested authoring command**

In `internal/cli/cli.go`, add the import:

```go
"argos/internal/authoringdogfood"
```

Add a new constant:

```go
const defaultAuthoringDogfoodCasesPath = "testdata/authoring-golden/cases.json"
```

Modify `runDogfood`:

```go
case "authoring":
	return runDogfoodAuthoring(args[1:], stdout, stderr)
```

Add the nested command functions:

```go
func runDogfoodAuthoring(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood authoring: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "cases":
		return runDogfoodAuthoringCases(args[1:], stdout, stderr)
	case "packet":
		return runDogfoodAuthoringPacket(args[1:], stdout, stderr)
	case "evaluate":
		return runDogfoodAuthoringEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dogfood authoring: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfoodAuthoringCases(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood authoring cases: --json is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring cases: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, authoringdogfood.Summaries(cases))
}
```

Add packet and evaluate functions:

```go
func runDogfoodAuthoringPacket(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring packet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "authoring dogfood case id or public handle")
	workspacePath := flags.String("workspace", "", "fixture workspace path")
	argosBinary := flags.String("argos-binary", "", "argos binary path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood authoring packet: --case is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring packet: %v\n", err)
		return 1
	}
	packet, err := authoringdogfood.BuildPacket(cases, authoringdogfood.PacketOptions{
		CaseID:      *caseID,
		Workspace:   *workspacePath,
		ArgosBinary: *argosBinary,
	})
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring packet: %v\n", err)
		return 2
	}
	if *jsonOut {
		return printJSON(stdout, stderr, packet)
	}
	fmt.Fprint(stdout, packet.Markdown)
	return 0
}

func runDogfoodAuthoringEvaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "authoring dogfood case id or public handle")
	reportPath := flags.String("report", "", "markdown report path")
	workspacePath := flags.String("workspace", "", "workspace path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --json is required")
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --case is required")
		return 2
	}
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --report is required")
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --workspace is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: %v\n", err)
		return 1
	}
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: read report: %v\n", err)
		return 1
	}
	report, err := authoringdogfood.ParseMarkdownReport(data)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: parse report: %v\n", err)
		return 1
	}
	evaluation, err := authoringdogfood.EvaluateCase(cases, *caseID, *workspacePath, report)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: %v\n", err)
		return 2
	}
	return printJSON(stdout, stderr, evaluation)
}
```

- [ ] **Step 4: Run CLI focused tests**

Run:

```bash
go test ./internal/cli -run 'TestRunDogfoodAuthoring' -count=1
```

Expected: PASS.

- [ ] **Step 5: Run discovery dogfood CLI regression tests**

Run:

```bash
go test ./internal/cli -run 'TestRunDogfood' -count=1
```

Expected: PASS. Existing `argos dogfood cases|packet|evaluate` behavior remains unchanged.

- [ ] **Step 6: Commit task 5**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: expose authoring dogfood CLI"
```

---

### Task 6: Documentation And Final Verification

**Files:**
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`

- [ ] **Step 1: Update README wording**

In `README.md`, update the authoring section to mention v2 source profiles and authoring dogfood. The inserted text should include:

```markdown
Authoring v2 proposals describe mixed source and trust through `source_profile`
instead of forcing a single authoring mode. A proposal should name the future
agent audience, scope, future-use triggers, negative triggers, source profile,
applicability boundaries, delivery path, and human review decisions.
```

Also add the authoring harness commands near existing dogfood commands:

````markdown
Authoring dogfood validates the write side from natural user requests:

```bash
argos dogfood authoring cases --json
argos dogfood authoring packet --case <case-handle> --workspace <fixture> --argos-binary <argos>
argos dogfood authoring evaluate --case <case-handle> --report <report.md> --workspace <fixture> --json
```
````

- [ ] **Step 2: Update `capture-knowledge` skill proposal sections**

In `skills/capture-knowledge/SKILL.md`, update the proposal section list to include:

```text
User Request
Knowledge Goal
Future Agent Audience
Scope
Source Profile
Proposed Shape
Future Use
Applicability Boundaries
Overlap Decision
Delivery
Candidate Files
Verification Plan
Human Review Decisions
```

Also add this rule under hard gates:

```markdown
Treat `source_profile` as multi-source. Do not force the request into one mode
when it combines user-confirmed standards, observed files, imported references,
templates, examples, synthesized recommendations, assumptions, and open
questions.
```

- [ ] **Step 3: Run docs/skill text scan**

Run:

```bash
rg -n "authoring_mode|business_domains|goal.*mode" README.md skills/capture-knowledge/SKILL.md docs/superpowers/specs/2026-05-02-argos-authoring-v2-contract-and-harness-design.md
```

Expected: remaining matches are either compatibility explanations or examples that are intentionally contrasted with v2. No new authoring guidance should tell users to start from `mode`.

- [ ] **Step 4: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Run CLI smoke commands**

Run:

```bash
go run . dogfood authoring cases --json
go run . dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring --argos-binary ./argos
```

Expected: first command emits public case summaries with natural `user_request` and no hidden oracle fields. Second command emits a Markdown runner packet with authoring commands and no hidden expected values.

- [ ] **Step 6: Check diff hygiene**

Run:

```bash
git diff --check
git status --short
```

Expected: `git diff --check` exits 0. `git status --short` shows only the intended files for this task.

- [ ] **Step 7: Commit task 6**

Run:

```bash
git add README.md skills/capture-knowledge/SKILL.md
git commit -m "docs: document authoring v2 workflow"
```

---

## Final Verification

- [ ] **Step 1: Run all tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Verify authoring dogfood secrecy**

Run:

```bash
go run . dogfood authoring cases --json
```

Expected: output contains `user_request` and public handles such as `case-001`; output does not contain `oracle`, `expected_result`, `required_guards`, or internal case IDs such as `go_template_standard`.

- [ ] **Step 3: Verify authoring packet**

Run:

```bash
go run . dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring --argos-binary ./argos
```

Expected: output includes `author inspect --json`, `author verify --json`, simulated approval boundaries, required report sections, and the natural Go template request. It does not include hidden oracle fields.

- [ ] **Step 4: Verify repository state**

Run:

```bash
git status --short
git log --oneline -6
```

Expected: worktree is clean after commits. Recent commits show the six implementation commits from this plan.

---

## Self-Review Notes

- Spec coverage: v2 proposal contract is covered by Tasks 1 and 2; natural public harness input is covered by Task 3; workspace-aware evaluator and `author.Verify` integration are covered by Task 4; CLI surface is covered by Task 5; README and skill wording are covered by Task 6.
- Scope boundary: proposal generation, scaffolding, MCP authoring tools, and automatic runner orchestration are intentionally outside this plan.
- Type consistency: v2 public schema uses `ProposalV2`, `SourceProfileV2`, `FutureUseV2`, `DeliveryV2`, and `HumanReviewV2`; harness schema uses `authoringdogfood.Case`, `Input`, `Approval`, and `Oracle`; CLI imports only `internal/authoringdogfood`.
