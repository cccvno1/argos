package author

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const ProposalSchemaVersion = "authoring.proposal.v1"

const (
	ModeObserved      = "observed"
	ModeUserSpecified = "user_specified"
	ModeSynthesized   = "synthesized"
	ModeImported      = "imported"
	ModeTemplate      = "template"
)

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Proposal struct {
	SchemaVersion           string                  `json:"schema_version"`
	KnowledgeGoal           string                  `json:"knowledge_goal"`
	AuthoringMode           string                  `json:"authoring_mode"`
	Project                 string                  `json:"project"`
	ProposedShape           ProposedShape           "json:\"proposed_\x73hape\""
	FutureRetrievalContract FutureRetrievalContract `json:"future_retrieval_contract"`
	SourceAndTrust          SourceAndTrust          `json:"source_and_trust"`
	Applicability           Applicability           `json:"applicability"`
	OverlapDecision         OverlapDecision         "json:\"overlap_\x64ecision\""
	Delivery                Delivery                `json:"delivery"`
	CandidateFiles          []CandidateFile         `json:"candidate_files"`
	VerificationPlan        VerificationPlan        "json:\"verification_\x70lan\""
}

type ProposedShape struct {
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	ID       string `json:"id"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type FutureRetrievalContract struct {
	Tasks           []string `json:"tasks"`
	Phases          []string `json:"phases"`
	Files           []string `json:"files"`
	FileGlobs       []string `json:"file_globs"`
	QueryPhrases    []string `json:"query_phrases"`
	Projects        []string `json:"projects"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
	Tags            []string `json:"tags"`
	ExpectedUse     string   `json:"expected_use"`
	CiteAfterUse    bool     `json:"cite_after_use"`
}

type SourceAndTrust struct {
	UserProvided  []string `json:"user_provided"`
	Observed      []string `json:"observed"`
	Imported      []string `json:"imported"`
	Synthesized   []string `json:"synthesized"`
	Assumptions   []string `json:"assumptions"`
	OpenQuestions []string `json:"open_questions"`
}

type Applicability struct {
	WhenToUse    []string `json:"when_to_use"`
	WhenNotToUse []string `json:"when_not_to_use"`
	Tradeoffs    []string `json:"tradeoffs"`
}

type OverlapDecision struct {
	OfficialOverlap []string `json:"official_overlap"`
	InboxOverlap    []string `json:"inbox_overlap"`
	Decision        string   `json:"decision"`
	Reason          string   `json:"reason"`
}

type Delivery struct {
	Path                   string `json:"path"`
	RequiresHumanApproval  bool   `json:"requires_human_approval"`
	PriorityMustAuthorized bool   `json:"priority_must_authorized"`
	PromoteAuthorized      bool   `json:"promote_authorized"`
}

type CandidateFile struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Load    string `json:"load"`
}

type VerificationPlan struct {
	ValidatePath         string                `json:"validate_path"`
	FindabilityScenarios []FindabilityScenario `json:"findability_scenarios"`
}

type FindabilityScenario struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Query   string   `json:"query"`
	Files   []string `json:"files"`
}

func LoadProposal(path string) (Proposal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Proposal{}, fmt.Errorf("read proposal: %w", err)
	}
	var proposal Proposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return Proposal{}, fmt.Errorf("parse proposal JSON: %w", err)
	}
	return proposal, nil
}

func ValidateProposal(proposal Proposal) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	if strings.TrimSpace(proposal.SchemaVersion) != ProposalSchemaVersion {
		addFail("schema_version must be authoring.proposal.v1")
	}
	if strings.TrimSpace(proposal.KnowledgeGoal) == "" {
		addFail("knowledge_goal is required")
	}
	if !validMode(proposal.AuthoringMode) {
		addFail("authoring_mode must be observed, user_specified, synthesized, imported, or template")
	}
	if strings.TrimSpace(proposal.Project) == "" {
		addFail("project is required")
	}
	if strings.TrimSpace(proposal.ProposedShape.Kind) == "" || strings.TrimSpace(proposal.ProposedShape.Type) == "" || strings.TrimSpace(proposal.ProposedShape.ID) == "" || strings.TrimSpace(proposal.ProposedShape.Path) == "" {
		addFail("proposed_\x73hape must include kind, type, id, and path")
	}
	if strings.TrimSpace(proposal.ProposedShape.Status) == "" || strings.TrimSpace(proposal.ProposedShape.Priority) == "" {
		addFail("proposed_\x73hape must include status and priority")
	}
	if strings.TrimSpace(proposal.ProposedShape.Priority) == "must" && !proposal.Delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	if len(nonEmpty(proposal.FutureRetrievalContract.Tasks)) == 0 && len(nonEmpty(proposal.FutureRetrievalContract.QueryPhrases)) == 0 {
		addFail("future_retrieval_contract must include tasks or query_phrases")
	}
	if len(nonEmpty(proposal.FutureRetrievalContract.Projects)) == 0 {
		addFail("future_retrieval_contract.projects is required")
	}
	if strings.TrimSpace(proposal.FutureRetrievalContract.ExpectedUse) == "" {
		addFail("future_retrieval_contract.expected_use is required")
	}
	validateSourceAndTrust(proposal.AuthoringMode, proposal.SourceAndTrust, addReview)
	if len(nonEmpty(proposal.Applicability.WhenToUse)) == 0 || len(nonEmpty(proposal.Applicability.WhenNotToUse)) == 0 {
		addReview("applicability must include when_to_use and when_not_to_use")
	}
	if !validOverlapDecision(proposal.OverlapDecision.Decision) {
		addFail("overlap_\x64ecision.decision must be create_new, update_existing, stop, or unresolved")
	}
	if strings.TrimSpace(proposal.OverlapDecision.Reason) == "" {
		addReview("overlap_\x64ecision.reason is required")
	}
	if proposal.Delivery.Path != "inbox" && proposal.Delivery.Path != "official_review" {
		addFail("delivery.path must be inbox or official_review")
	}
	if !proposal.Delivery.RequiresHumanApproval {
		addFail("delivery.requires_human_approval must be true")
	}
	if len(proposal.CandidateFiles) == 0 {
		addFail("candidate_files must include at least one file")
	}
	for i, file := range proposal.CandidateFiles {
		if strings.TrimSpace(file.Path) == "" || strings.TrimSpace(file.Purpose) == "" || strings.TrimSpace(file.Load) == "" {
			addFail(fmt.Sprintf("candidate_files[%d] must include path, purpose, and load", i))
		}
	}
	if strings.TrimSpace(proposal.VerificationPlan.ValidatePath) == "" {
		addFail("verification_\x70lan.validate_path is required")
	}
	if len(proposal.VerificationPlan.FindabilityScenarios) == 0 {
		addReview("verification_\x70lan.findability_scenarios should include at least one scenario")
	}
	for i, scenario := range proposal.VerificationPlan.FindabilityScenarios {
		if strings.TrimSpace(scenario.Project) == "" || (strings.TrimSpace(scenario.Task) == "" && strings.TrimSpace(scenario.Query) == "") {
			addFail(fmt.Sprintf("verification_\x70lan.findability_scenarios[%d] must include project and task or query", i))
		}
	}
	return findings
}

func validateSourceAndTrust(mode string, source SourceAndTrust, addReview func(string)) {
	switch mode {
	case ModeObserved:
		if len(nonEmpty(source.Observed)) == 0 {
			addReview("source_and_trust.observed is required for observed authoring")
		}
	case ModeUserSpecified:
		if len(nonEmpty(source.UserProvided)) == 0 {
			addReview("source_and_trust.user_provided is required for user_specified authoring")
		}
	case ModeImported:
		if len(nonEmpty(source.Imported)) == 0 {
			addReview("source_and_trust.imported is required for imported authoring")
		}
	case ModeSynthesized:
		if len(nonEmpty(source.Assumptions)) == 0 {
			addReview("source_and_trust.assumptions is required for synthesized authoring")
		}
		if !hasAnySource(source) {
			addReview("source_and_trust must include at least one source bucket for synthesized authoring")
		}
	case ModeTemplate:
		if !hasAnySource(source) {
			addReview("source_and_trust must include at least one source bucket for template authoring")
		}
	default:
		if !hasAnySource(source) {
			addReview("source_and_trust must include at least one source bucket")
		}
	}
}

func validMode(mode string) bool {
	switch mode {
	case ModeObserved, ModeUserSpecified, ModeSynthesized, ModeImported, ModeTemplate:
		return true
	default:
		return false
	}
}

func validOverlapDecision(decision string) bool {
	switch decision {
	case "create_new", "update_existing", "stop", "unresolved":
		return true
	default:
		return false
	}
}

func hasAnySource(source SourceAndTrust) bool {
	return len(nonEmpty(source.UserProvided)) > 0 ||
		len(nonEmpty(source.Observed)) > 0 ||
		len(nonEmpty(source.Imported)) > 0 ||
		len(nonEmpty(source.Synthesized)) > 0
}

func nonEmpty(values []string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}
