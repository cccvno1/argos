package author

import (
	"strings"
	"testing"
)

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

func TestValidateProposalV2RequiresClaimSourceForUserConfirmedClaims(t *testing.T) {
	proposal := validProposalV2()
	proposal.SourceProfile.Claims = []SourceClaimV2{
		{Claim: "All generated services use the standard layout.", Kind: "decision", Trust: "user_confirmed"},
	}

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "review-needed", "user_confirmed claim requires source") {
		t.Fatalf("expected user-confirmed claim source finding, got %#v", findings)
	}
}

func TestValidateProposalV2ReviewsIncompleteSourceProfile(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProposalV2)
		want string
	}{
		{
			name: "no source buckets",
			edit: func(p *ProposalV2) {
				p.SourceProfile = SourceProfileV2{
					Claims: []SourceClaimV2{
						{Claim: "Future Go services should use the template layout.", Kind: "decision", Trust: "synthesized", RequiresReview: true},
					},
				}
			},
			want: "source_profile must include at least one source bucket",
		},
		{
			name: "no claims",
			edit: func(p *ProposalV2) {
				p.SourceProfile.Claims = nil
			},
			want: "source_profile.claims should include claim-level trust",
		},
		{
			name: "observed claim without observed bucket",
			edit: func(p *ProposalV2) {
				p.SourceProfile.Observed = nil
				p.SourceProfile.Claims = []SourceClaimV2{
					{Claim: "The directory layout comes from templates/go-service.", Kind: "fact", Trust: "observed", Source: []string{"templates/go-service"}},
				}
			},
			want: "observed claim requires observed source",
		},
		{
			name: "imported claim without imported bucket",
			edit: func(p *ProposalV2) {
				p.SourceProfile.Imported = nil
				p.SourceProfile.Claims = []SourceClaimV2{
					{Claim: "Use the imported service baseline.", Kind: "fact", Trust: "imported", Source: []string{"external template"}},
				}
			},
			want: "imported claim requires imported source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			tt.edit(&proposal)

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "review-needed", tt.want) {
				t.Fatalf("expected finding containing %q, got %#v", tt.want, findings)
			}
		})
	}
}

func TestValidateProposalV2ConstrainsSourceProfileClaimKindTrustAndSource(t *testing.T) {
	tests := []struct {
		name     string
		claim    SourceClaimV2
		severity string
		want     string
	}{
		{
			name:     "unknown kind",
			claim:    SourceClaimV2{Claim: "The template uses generated handlers.", Kind: "rule", Trust: "observed", Source: []string{"templates/go-service"}},
			severity: "fail",
			want:     "unknown source_profile.claims kind",
		},
		{
			name:     "unknown trust",
			claim:    SourceClaimV2{Claim: "The template uses generated handlers.", Kind: "fact", Trust: "manual", Source: []string{"templates/go-service"}},
			severity: "fail",
			want:     "unknown source_profile.claims trust",
		},
		{
			name:     "observed claim missing source",
			claim:    SourceClaimV2{Claim: "The template uses generated handlers.", Kind: "fact", Trust: "observed"},
			severity: "review-needed",
			want:     "source_profile.claims source is required",
		},
		{
			name:     "synthesized claim missing source",
			claim:    SourceClaimV2{Claim: "Use Redis locks for stampede protection.", Kind: "recommendation", Trust: "synthesized", RequiresReview: true},
			severity: "review-needed",
			want:     "source_profile.claims source is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			proposal.SourceProfile.Claims = []SourceClaimV2{tt.claim}

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, tt.severity, tt.want) {
				t.Fatalf("expected finding containing %q, got %#v", tt.want, findings)
			}
		})
	}
}

func TestValidateProposalV2RejectsUnknownContractEnums(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProposalV2)
		want string
	}{
		{
			name: "audience primary",
			edit: func(p *ProposalV2) { p.Audience.Primary = "writer_agent" },
			want: "audience.primary must be",
		},
		{
			name: "scope stability",
			edit: func(p *ProposalV2) { p.Scope.Stability = "stable" },
			want: "scope.stability",
		},
		{
			name: "scope distribution",
			edit: func(p *ProposalV2) { p.Scope.Distribution = "department" },
			want: "scope.distribution",
		},
		{
			name: "proposed shape kind",
			edit: func(p *ProposalV2) { p.ProposedShape.Kind = "bundle" },
			want: "proposed_shape.kind",
		},
		{
			name: "proposed shape type",
			edit: func(p *ProposalV2) { p.ProposedShape.Type = "guide" },
			want: "proposed_shape.type",
		},
		{
			name: "proposed shape entrypoint load",
			edit: func(p *ProposalV2) { p.ProposedShape.EntrypointLoad = "always" },
			want: "proposed_shape.entrypoint_load",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			tt.edit(&proposal)

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "fail", tt.want) {
				t.Fatalf("expected enum failure containing %q, got %#v", tt.want, findings)
			}
		})
	}
}

func TestValidateProposalV2ReportsPriorityMustOnce(t *testing.T) {
	proposal := validProposalV2()
	proposal.ProposedShape.Priority = "must"
	proposal.Delivery.PriorityMustAuthorized = false

	findings := ValidateProposalV2(proposal)

	count := 0
	for _, finding := range findings {
		if finding.Severity == "fail" && strings.Contains(finding.Message, "priority: must") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one priority: must finding, got %d in %#v", count, findings)
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

func TestValidateProposalV2RejectsIncompleteCandidateFiles(t *testing.T) {
	tests := []struct {
		name string
		file CandidateFile
	}{
		{
			name: "missing path",
			file: CandidateFile{Purpose: "entrypoint", Load: "start_here"},
		},
		{
			name: "missing purpose",
			file: CandidateFile{Path: "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", Load: "start_here"},
		},
		{
			name: "missing load",
			file: CandidateFile{Path: "knowledge/.inbox/packages/backend/go-service-template/KNOWLEDGE.md", Purpose: "entrypoint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			proposal.CandidateFiles = []CandidateFile{tt.file}

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "fail", "candidate_files") {
				t.Fatalf("expected candidate_files failure, got %#v", findings)
			}
		})
	}
}

func TestValidateProposalV2RejectsCandidateFilePathBoundaryViolations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProposalV2)
		want string
	}{
		{
			name: "absolute candidate file path",
			edit: func(p *ProposalV2) {
				p.CandidateFiles[0].Path = "/tmp/KNOWLEDGE.md"
			},
			want: "candidate_files[0].path must be relative",
		},
		{
			name: "parent candidate file path",
			edit: func(p *ProposalV2) {
				p.CandidateFiles[0].Path = "knowledge/.inbox/packages/backend/go-service-template/../other/KNOWLEDGE.md"
			},
			want: "candidate_files[0].path must stay inside workspace",
		},
		{
			name: "official file for inbox delivery",
			edit: func(p *ProposalV2) {
				p.CandidateFiles[0].Path = "knowledge/packages/backend/go-service-template/KNOWLEDGE.md"
			},
			want: "candidate_files[0].path is outside inbox delivery boundary",
		},
		{
			name: "file outside proposed path",
			edit: func(p *ProposalV2) {
				p.CandidateFiles[0].Path = "knowledge/.inbox/packages/backend/other-template/KNOWLEDGE.md"
			},
			want: "candidate_files[0].path must stay under proposed_shape.path",
		},
		{
			name: "verification path mismatch",
			edit: func(p *ProposalV2) {
				p.VerificationPlan.ValidatePath = "knowledge/.inbox/packages/backend/other-template"
			},
			want: "verification_plan.validate_path must match proposed_shape.path",
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

func TestValidateProposalV2RequiresVerificationPlanValidatePath(t *testing.T) {
	proposal := validProposalV2()
	proposal.VerificationPlan.ValidatePath = ""

	findings := ValidateProposalV2(proposal)

	if !hasFinding(findings, "fail", "verification_plan.validate_path is required") {
		t.Fatalf("expected verification validate path failure, got %#v", findings)
	}
}

func TestValidateProposalV2RejectsIncompleteFindabilityScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario FindabilityScenario
	}{
		{
			name:     "missing project",
			scenario: FindabilityScenario{Task: "generate a Go service", Query: "go service template"},
		},
		{
			name:     "missing task and query",
			scenario: FindabilityScenario{Project: "mall-api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposalV2()
			proposal.VerificationPlan.FindabilityScenarios = []FindabilityScenario{tt.scenario}

			findings := ValidateProposalV2(proposal)

			if !hasFinding(findings, "fail", "verification_plan.findability_scenarios") {
				t.Fatalf("expected findability scenario failure, got %#v", findings)
			}
		})
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
			TriggerRequests:  []string{"generate a Go service", "create a backend service from the standard template"},
			NegativeTriggers: []string{"write a one-off Go script"},
			Phases:           []string{"planning", "implementation", "review"},
			QueryPhrases:     []string{"go service template", "standard go service layout"},
			ExpectedUse:      "read_before_implementation",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When creating a new Go service in mall-api."},
			WhenNotToUse: []string{"When changing an existing legacy service that does not follow the template."},
			Tradeoffs:    []string{"The template improves consistency but may not fit small scripts."},
		},
		OverlapDecision: OverlapDecisionV2{
			Decision: "create_new",
			Reason:   "No existing Go service template knowledge covers this future task.",
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
