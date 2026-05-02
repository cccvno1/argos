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
