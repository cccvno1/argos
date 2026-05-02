package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProposalAcceptsCompleteSynthesizedDraft(t *testing.T) {
	proposal := validProposal()
	findings := ValidateProposal(proposal)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestValidateProposalRequiresFutureRetrievalContract(t *testing.T) {
	proposal := validProposal()
	proposal.FutureRetrievalContract.Tasks = nil
	proposal.FutureRetrievalContract.QueryPhrases = nil
	proposal.FutureRetrievalContract.Projects = nil

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "future_retrieval_contract") {
		t.Fatalf("expected future retrieval failure, got %#v", findings)
	}
}

func TestValidateProposalRequiresSourceAndTrust(t *testing.T) {
	proposal := validProposal()
	proposal.SourceAndTrust.Assumptions = nil

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "review-needed", "source_and_trust.assumptions") {
		t.Fatalf("expected synthesized assumptions review finding, got %#v", findings)
	}
}

func TestValidateProposalRejectsPriorityMustWithoutAuthorization(t *testing.T) {
	proposal := validProposal()
	proposal.ProposedShape.Priority = "must"
	proposal.Delivery.PriorityMustAuthorized = false

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "priority: must") {
		t.Fatalf("expected must authorization failure, got %#v", findings)
	}
}

func TestValidateProposalRejectsUnknownMode(t *testing.T) {
	proposal := validProposal()
	proposal.AuthoringMode = "memo"

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "authoring_mode") {
		t.Fatalf("expected authoring mode failure, got %#v", findings)
	}
}

func TestValidateProposalRequiresModeSpecificSourceAndTrust(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		source   SourceAndTrust
		wantText string
	}{
		{
			name:     "observed requires observed source",
			mode:     ModeObserved,
			source:   SourceAndTrust{Synthesized: []string{"Wrong bucket."}},
			wantText: "source_and_trust.observed",
		},
		{
			name:     "user specified requires user provided source",
			mode:     ModeUserSpecified,
			source:   SourceAndTrust{Observed: []string{"Wrong bucket."}},
			wantText: "source_and_trust.user_provided",
		},
		{
			name:     "imported requires imported source",
			mode:     ModeImported,
			source:   SourceAndTrust{UserProvided: []string{"Wrong bucket."}},
			wantText: "source_and_trust.imported",
		},
		{
			name:     "template requires any source",
			mode:     ModeTemplate,
			source:   SourceAndTrust{},
			wantText: "source_and_trust",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposal()
			proposal.AuthoringMode = tt.mode
			proposal.SourceAndTrust = tt.source

			findings := ValidateProposal(proposal)
			if !hasFinding(findings, "review-needed", tt.wantText) {
				t.Fatalf("expected source/trust review finding containing %q, got %#v", tt.wantText, findings)
			}
		})
	}
}

func TestValidateProposalRejectsIncompleteCandidateFiles(t *testing.T) {
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
			file: CandidateFile{Path: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", Load: "start_here"},
		},
		{
			name: "missing load",
			file: CandidateFile{Path: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", Purpose: "entrypoint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposal()
			proposal.CandidateFiles = []CandidateFile{tt.file}

			findings := ValidateProposal(proposal)
			if !hasFinding(findings, "fail", "candidate_files") {
				t.Fatalf("expected candidate_files failure, got %#v", findings)
			}
		})
	}
}

func TestValidateProposalRejectsIncompleteFindabilityScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario FindabilityScenario
	}{
		{
			name:     "missing project",
			scenario: FindabilityScenario{Task: "implement product list cache", Query: "cache ttl product list"},
		},
		{
			name:     "missing task and query",
			scenario: FindabilityScenario{Project: "mall-api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := validProposal()
			proposal.VerificationPlan.FindabilityScenarios = []FindabilityScenario{tt.scenario}

			findings := ValidateProposal(proposal)
			if !hasFinding(findings, "fail", "verification_plan.findability_scenarios") {
				t.Fatalf("expected findability scenario failure, got %#v", findings)
			}
		})
	}
}

func TestLoadProposalReadsValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proposal.json")
	data, err := json.Marshal(validProposal())
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	proposal, err := LoadProposal(path)
	if err != nil {
		t.Fatalf("load proposal: %v", err)
	}
	if proposal.SchemaVersion != ProposalSchemaVersion {
		t.Fatalf("expected schema version %q, got %q", ProposalSchemaVersion, proposal.SchemaVersion)
	}
	if proposal.ProposedShape.ID != "package:backend.product-list-cache.v1" {
		t.Fatalf("unexpected proposal id: %q", proposal.ProposedShape.ID)
	}
}

func TestLoadProposalRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	_, err := LoadProposal(path)
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
	if !strings.Contains(err.Error(), "parse proposal JSON") {
		t.Fatalf("expected parse proposal JSON error, got %v", err)
	}
}

func validProposal() Proposal {
	return Proposal{
		SchemaVersion: ProposalSchemaVersion,
		KnowledgeGoal: "Create product-list cache engineering knowledge for future backend agents.",
		AuthoringMode: ModeSynthesized,
		Project:       "mall-api",
		ProposedShape: ProposedShape{
			Kind:     "package",
			Type:     "package",
			Title:    "Product List Cache Engineering Knowledge",
			ID:       "package:backend.product-list-cache.v1",
			Path:     "knowledge/.inbox/packages/backend/product-list-cache",
			Status:   "draft",
			Priority: "should",
		},
		FutureRetrievalContract: FutureRetrievalContract{
			Tasks:           []string{"implement product list cache"},
			Phases:          []string{"implementation", "review"},
			Files:           []string{"internal/catalog/products.go"},
			FileGlobs:       []string{"internal/catalog/**"},
			QueryPhrases:    []string{"product list cache", "cache ttl product list"},
			Projects:        []string{"mall-api"},
			TechDomains:     []string{"backend", "database"},
			BusinessDomains: []string{"catalog"},
			Tags:            []string{"cache", "redis"},
			ExpectedUse:     "read_before_implementation",
			CiteAfterUse:    true,
		},
		SourceAndTrust: SourceAndTrust{
			Synthesized: []string{"General cache design practice adapted to the project context."},
			Assumptions: []string{"The project uses Redis for product-list cache storage."},
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When implementing product-list cache behavior."},
			WhenNotToUse: []string{"When changing unrelated catalog reads."},
			Tradeoffs:    []string{"Caching improves latency but can serve stale data."},
		},
		OverlapDecision: OverlapDecision{
			Decision: "create_new",
			Reason:   "No existing product-list cache knowledge covers this future task.",
		},
		Delivery: Delivery{
			Path:                   "inbox",
			RequiresHumanApproval:  true,
			PriorityMustAuthorized: false,
			PromoteAuthorized:      false,
		},
		CandidateFiles: []CandidateFile{
			{Path: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", Purpose: "entrypoint", Load: "start_here"},
		},
		VerificationPlan: VerificationPlan{
			ValidatePath: "knowledge/.inbox/packages/backend/product-list-cache",
			FindabilityScenarios: []FindabilityScenario{
				{
					Project: "mall-api",
					Phase:   "implementation",
					Task:    "implement product list cache",
					Query:   "cache ttl product list",
					Files:   []string{"internal/catalog/products.go"},
				},
			},
		},
	}
}

func hasFinding(findings []Finding, severity string, text string) bool {
	for _, finding := range findings {
		if finding.Severity == severity && strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}
