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
	TriggerRequests  []string `json:"trigger_requests"`
	NegativeTriggers []string `json:"negative_triggers"`
	Phases           []string `json:"phases"`
	QueryPhrases     []string `json:"query_phrases"`
	ExpectedUse      string   `json:"expected_use"`
	CitationPolicy   string   `json:"citation_policy"`
	MissingNeeds     []string `json:"missing_needs"`
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
	} else if !validAudiencePrimaryV2(proposal.Audience.Primary) {
		addFail("audience.primary must be implementer_agent, maintainer_agent, reviewer_agent, consumer_agent, operator_agent, or cross_project_agent")
	}
	if len(nonEmpty(proposal.Audience.AgentActionsSupported)) == 0 {
		addReview("audience.agent_actions_supported should describe future agent actions")
	}
	if len(nonEmpty(proposal.Scope.Projects)) == 0 && strings.TrimSpace(proposal.Scope.Distribution) == "" {
		addFail("scope must include projects or distribution")
	}
	if strings.TrimSpace(proposal.Scope.Stability) == "" {
		addReview("scope.stability is required")
	} else if !validScopeStabilityV2(proposal.Scope.Stability) {
		addFail("scope.stability must be draft, experimental, reviewed, or official_candidate")
	}
	if strings.TrimSpace(proposal.Scope.Distribution) != "" && !validScopeDistributionV2(proposal.Scope.Distribution) {
		addFail("scope.distribution must be personal, project, small_team, multi_project, organization, or public_consumer")
	}
	validateSourceProfileV2(proposal.SourceProfile, addFail, addReview)
	validateProposedShapeV2(proposal.ProposedShape, proposal.Delivery, addFail, addReview)
	validateFutureUseV2(proposal.FutureUse, addFail, addReview)
	if len(nonEmpty(proposal.Applicability.WhenToUse)) == 0 || len(nonEmpty(proposal.Applicability.WhenNotToUse)) == 0 {
		addReview("applicability must include when_to_use and when_not_to_use")
	}
	validateOverlapDecisionV2(proposal.OverlapDecision, addFail, addReview)
	validateDeliveryV2(proposal.Delivery, addFail)
	validateCandidateAndPlanV2(proposal, addFail, addReview)
	if len(nonEmpty(proposal.HumanReview.ReviewQuestions)) == 0 {
		addReview("human_review.review_questions should include reviewer decisions")
	}
	return findings
}

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
		if !validSourceClaimKindV2(claim.Kind) {
			addFail(fmt.Sprintf("unknown source_profile.claims kind at index %d", i))
		}
		if !validSourceClaimTrustV2(claim.Trust) {
			addFail(fmt.Sprintf("unknown source_profile.claims trust at index %d", i))
		}
		if claim.Kind != "question" && len(nonEmpty(claim.Source)) == 0 {
			addReview(fmt.Sprintf("source_profile.claims source is required at index %d", i))
		}
		if claim.Trust == "synthesized" && len(nonEmpty(source.Assumptions)) == 0 && !claim.RequiresReview {
			addReview("synthesized claim requires assumptions or review")
		}
		if claim.Trust == "user_confirmed" && len(nonEmpty(source.UserConfirmed)) == 0 {
			addReview("user_confirmed claim requires user_confirmed source")
		}
		if claim.Trust == "user_confirmed" && len(nonEmpty(claim.Source)) == 0 {
			addReview("user_confirmed claim requires source")
		}
		if claim.Trust == "observed" && len(nonEmpty(source.Observed)) == 0 {
			addReview("observed claim requires observed source")
		}
		if claim.Trust == "imported" && len(nonEmpty(source.Imported)) == 0 {
			addReview("imported claim requires imported source")
		}
	}
}

func validSourceClaimKindV2(kind string) bool {
	switch kind {
	case "fact", "decision", "recommendation", "example", "template", "assumption", "question":
		return true
	default:
		return false
	}
}

func validSourceClaimTrustV2(trust string) bool {
	switch trust {
	case "observed", "user_confirmed", "imported", "synthesized", "illustrative", "unknown":
		return true
	default:
		return false
	}
}

func validateProposedShapeV2(shape ProposedShapeV2, delivery DeliveryV2, addFail func(string), addReview func(string)) {
	if strings.TrimSpace(shape.Kind) == "" || strings.TrimSpace(shape.Type) == "" || strings.TrimSpace(shape.ID) == "" || strings.TrimSpace(shape.Path) == "" {
		addFail("proposed_shape must include kind, type, id, and path")
	}
	if strings.TrimSpace(shape.Kind) != "" && !validProposedShapeKindV2(shape.Kind) {
		addFail("proposed_shape.kind must be item or package")
	}
	if strings.TrimSpace(shape.Type) != "" && !validProposedShapeTypeV2(shape.Type) {
		addFail("proposed_shape.type must be rule, decision, lesson, runbook, reference, template, checklist, or package")
	}
	if strings.TrimSpace(shape.Status) == "" || strings.TrimSpace(shape.Priority) == "" {
		addFail("proposed_shape must include status and priority")
	}
	if strings.TrimSpace(shape.Priority) == "must" && !delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	if strings.TrimSpace(shape.EntrypointLoad) != "" && !validEntrypointLoadV2(shape.EntrypointLoad) {
		addFail("proposed_shape.entrypoint_load must be start_here, read_before_implementation, read_before_review, on_demand, or reference_only")
	}
	if strings.TrimSpace(shape.Rationale) == "" {
		addReview("proposed_shape.rationale should explain item or package choice")
	}
}

func validAudiencePrimaryV2(primary string) bool {
	switch primary {
	case "implementer_agent", "maintainer_agent", "reviewer_agent", "consumer_agent", "operator_agent", "cross_project_agent":
		return true
	default:
		return false
	}
}

func validScopeStabilityV2(stability string) bool {
	switch stability {
	case "draft", "experimental", "reviewed", "official_candidate":
		return true
	default:
		return false
	}
}

func validScopeDistributionV2(distribution string) bool {
	switch distribution {
	case "personal", "project", "small_team", "multi_project", "organization", "public_consumer":
		return true
	default:
		return false
	}
}

func validProposedShapeKindV2(kind string) bool {
	switch kind {
	case "item", "package":
		return true
	default:
		return false
	}
}

func validProposedShapeTypeV2(shapeType string) bool {
	switch shapeType {
	case "rule", "decision", "lesson", "runbook", "reference", "template", "checklist", "package":
		return true
	default:
		return false
	}
}

func validEntrypointLoadV2(load string) bool {
	switch load {
	case "start_here", "read_before_implementation", "read_before_review", "on_demand", "reference_only":
		return true
	default:
		return false
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

func validateDeliveryV2(delivery DeliveryV2, addFail func(string)) {
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
	for i, scenario := range proposal.VerificationPlan.FindabilityScenarios {
		if strings.TrimSpace(scenario.Project) == "" || (strings.TrimSpace(scenario.Task) == "" && strings.TrimSpace(scenario.Query) == "") {
			addFail(fmt.Sprintf("verification_plan.findability_scenarios[%d] must include project and task or query", i))
		}
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
