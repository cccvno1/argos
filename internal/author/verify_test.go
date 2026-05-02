package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyValidInboxCandidatePassesWithoutMutatingWorkspaceIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	writeProposal(t, root, proposalPath, validVerifyProposal(candidatePath, "product list cache ttl"))

	indexPath := filepath.Join(root, "argos", "index.db")
	writeAuthorFile(t, root, "argos/index.db", "workspace-index-sentinel")

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "pass" {
		t.Fatalf("expected pass result, got %#v", result)
	}
	if result.Proposal.Validation != "pass" {
		t.Fatalf("expected proposal validation pass, got %#v", result.Proposal)
	}
	if result.Candidate.Validation != "pass" {
		t.Fatalf("expected candidate validation pass, got %#v", result.Candidate)
	}
	if result.Policy.Result != "pass" {
		t.Fatalf("expected policy pass, got %#v", result.Policy)
	}
	if result.Findability.Result != "pass" {
		t.Fatalf("expected findability pass, got %#v", result.Findability)
	}
	if len(result.Findability.Checks) != 1 || !result.Findability.Checks[0].MatchedCandidate || result.Findability.Checks[0].CandidateID != "rule:backend.cache.v1" {
		t.Fatalf("expected candidate findability match, got %#v", result.Findability.Checks)
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read workspace index: %v", err)
	}
	if string(data) != "workspace-index-sentinel" {
		t.Fatalf("Verify mutated workspace argos/index.db: %q", data)
	}
}

func TestVerifyValidInboxCandidatePassesWithoutCreatingWorkspaceIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	writeProposal(t, root, proposalPath, validVerifyProposal(candidatePath, "product list cache ttl"))

	indexPath := filepath.Join(root, "argos", "index.db")
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace index to be absent before Verify, got err %v", err)
	}

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "pass" {
		t.Fatalf("expected pass result, got %#v", result)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace index to remain absent after Verify, got err %v", err)
	}
}

func TestVerifyCandidateOutsideApprovedDeliveryBoundaryFails(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	writeProposal(t, root, proposalPath, validVerifyProposal(candidatePath, "product list cache ttl"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "fail" {
		t.Fatalf("expected fail result, got %#v", result)
	}
	if result.Policy.Result != "fail" {
		t.Fatalf("expected policy fail, got %#v", result.Policy)
	}
	if result.Findability.Result != "not-run" {
		t.Fatalf("expected findability not-run after policy fail, got %#v", result.Findability)
	}
	if !hasVerifyFinding(result.Findings, "fail", "approved delivery boundary") {
		t.Fatalf("expected approved delivery boundary finding, got %#v", result.Findings)
	}
}

func TestVerifyMissingCandidatePathReturnsStructuredFailure(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/missing.md"
	proposalPath := "authoring/proposals/cache.json"
	writeProposal(t, root, proposalPath, validVerifyProposal(candidatePath, "product list cache ttl"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "fail" {
		t.Fatalf("expected fail result, got %#v", result)
	}
	if result.Candidate.Validation != "fail" {
		t.Fatalf("expected candidate validation fail, got %#v", result.Candidate)
	}
	if result.Policy.Result != "not-run" {
		t.Fatalf("expected policy not-run after candidate load failure, got %#v", result.Policy)
	}
	if result.Findability.Result != "not-run" {
		t.Fatalf("expected findability not-run after candidate load failure, got %#v", result.Findability)
	}
	if !hasVerifyFinding(result.Findings, "fail", "candidate validation/load failure") {
		t.Fatalf("expected candidate validation/load failure finding, got %#v", result.Findings)
	}
}

func TestVerifyOfficialReviewCandidateOutsideApprovedDeliveryBoundaryFails(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.Delivery.Path = "official_review"
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "fail" {
		t.Fatalf("expected fail result, got %#v", result)
	}
	if result.Policy.Result != "fail" {
		t.Fatalf("expected policy fail, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "fail", "approved delivery boundary") {
		t.Fatalf("expected approved delivery boundary finding, got %#v", result.Findings)
	}
}

func TestVerifyCandidateMissingFutureRetrievalTagReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.FutureRetrievalContract.Tags = []string{"cache", "redis"}
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "redis") {
		t.Fatalf("expected missing redis tag finding, got %#v", result.Findings)
	}
}

func TestVerifyCandidateMissingFutureRetrievalProjectReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.FutureRetrievalContract.Projects = []string{"mall-api", "catalog-web"}
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "catalog-web") {
		t.Fatalf("expected missing catalog-web project finding, got %#v", result.Findings)
	}
}

func TestVerifyCandidateMissingFutureRetrievalDomainsReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.FutureRetrievalContract.TechDomains = []string{"backend", "database"}
	proposal.FutureRetrievalContract.BusinessDomains = []string{"catalog", "checkout"}
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "tech domain") || !hasVerifyFinding(result.Findings, "review-needed", "database") {
		t.Fatalf("expected missing database tech domain finding, got %#v", result.Findings)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "business domain") || !hasVerifyFinding(result.Findings, "review-needed", "checkout") {
		t.Fatalf("expected missing checkout business domain finding, got %#v", result.Findings)
	}
}

func TestVerifyCandidateMissingFutureRetrievalFileScopeReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.FutureRetrievalContract.Files = []string{"internal/catalog/products.go", "internal/payments/webhook.go"}
	proposal.FutureRetrievalContract.FileGlobs = []string{"internal/catalog/**", "internal/payments/**"}
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "future retrieval file") || !hasVerifyFinding(result.Findings, "review-needed", "internal/payments/webhook.go") {
		t.Fatalf("expected missing future retrieval file finding, got %#v", result.Findings)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "future retrieval file_glob") || !hasVerifyFinding(result.Findings, "review-needed", "internal/payments/**") {
		t.Fatalf("expected missing future retrieval file_glob finding, got %#v", result.Findings)
	}
}

func TestVerifyInvalidCandidateFileScopePatternReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	body := strings.ReplaceAll(authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"), "files: [internal/catalog/**]", `files: ["["]`)
	writeAuthorFile(t, root, candidatePath, body)
	proposalPath := "authoring/proposals/cache.json"
	writeProposal(t, root, proposalPath, validVerifyProposal(candidatePath, "product list cache ttl"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "invalid applies_to.files pattern") || !hasVerifyFinding(result.Findings, "review-needed", "[") {
		t.Fatalf("expected invalid pattern finding, got %#v", result.Findings)
	}
}

func TestVerifyInvalidCandidateFileScopePatternWithOnlyFileGlobsReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	body := strings.ReplaceAll(authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"), "files: [internal/catalog/**]", `files: ["["]`)
	writeAuthorFile(t, root, candidatePath, body)
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.FutureRetrievalContract.Files = nil
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Policy.Result != "review-needed" {
		t.Fatalf("expected policy review-needed, got %#v", result.Policy)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "invalid applies_to.files pattern") || !hasVerifyFinding(result.Findings, "review-needed", "[") {
		t.Fatalf("expected invalid pattern finding, got %#v", result.Findings)
	}
}

func TestVerifyValidCandidateNotFindableReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.VerificationPlan.FindabilityScenarios = []FindabilityScenario{{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "warehouse barcode scanner",
		Files:   []string{"internal/warehouse/scanner.go"},
	}}
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Candidate.Validation != "pass" {
		t.Fatalf("expected candidate validation pass, got %#v", result.Candidate)
	}
	if result.Policy.Result != "pass" {
		t.Fatalf("expected policy pass, got %#v", result.Policy)
	}
	if result.Findability.Result != "review-needed" {
		t.Fatalf("expected findability review-needed, got %#v", result.Findability)
	}
	if len(result.Findability.Checks) != 1 || result.Findability.Checks[0].MatchedCandidate || result.Findability.Checks[0].Support != "none" {
		t.Fatalf("expected unmatched candidate with no support, got %#v", result.Findability.Checks)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "not findable") {
		t.Fatalf("expected not findable review finding, got %#v", result.Findings)
	}
}

func TestVerifyEmptyFindabilityScenariosDoesNotPassFindability(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeAuthorFile(t, root, candidatePath, authorItem("rule:backend.cache.v1", "draft", "Product list cache TTL rule"))
	proposalPath := "authoring/proposals/cache.json"
	proposal := validVerifyProposal(candidatePath, "product list cache ttl")
	proposal.VerificationPlan.FindabilityScenarios = nil
	writeProposal(t, root, proposalPath, proposal)

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result from proposal validation, got %#v", result)
	}
	if result.Proposal.Validation != "review-needed" {
		t.Fatalf("expected proposal validation review-needed, got %#v", result.Proposal)
	}
	if result.Findability.Result != "review-needed" {
		t.Fatalf("expected findability review-needed without scenarios, got %#v", result.Findability)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "verification_plan.findability_scenarios is empty") {
		t.Fatalf("expected empty findability scenarios finding, got %#v", result.Findings)
	}
}

func TestVerifyWeakButMatchedFindabilityReturnsReviewNeeded(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	candidatePath := "knowledge/.inbox/items/backend/token.md"
	writeAuthorFile(t, root, candidatePath, weakMatchedAuthorItem())
	proposalPath := "authoring/proposals/token.json"
	writeProposal(t, root, proposalPath, weakMatchedProposal(candidatePath))

	result, err := Verify(root, VerifyRequest{
		ProposalPath:  proposalPath,
		CandidatePath: candidatePath,
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}

	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed result, got %#v", result)
	}
	if result.Findability.Result != "review-needed" {
		t.Fatalf("expected findability review-needed, got %#v", result.Findability)
	}
	if len(result.Findability.Checks) != 1 || !result.Findability.Checks[0].MatchedCandidate || result.Findability.Checks[0].Support != "weak" {
		t.Fatalf("expected weak matched candidate, got %#v", result.Findability.Checks)
	}
	if !hasVerifyFinding(result.Findings, "review-needed", "weak") {
		t.Fatalf("expected weak support review finding, got %#v", result.Findings)
	}
}

func validVerifyProposal(candidatePath string, findQuery string) Proposal {
	return Proposal{
		SchemaVersion:  ProposalSchemaVersion,
		KnowledgeGoal:  "Document product list cache TTL guidance.",
		AuthoringMode:  ModeUserSpecified,
		Project:        "mall-api",
		ProposedShape:  ProposedShape{Kind: "item", Type: "rule", Title: "Product list cache TTL rule", ID: "rule:backend.cache.v1", Path: candidatePath, Status: "draft", Priority: "should"},
		SourceAndTrust: SourceAndTrust{UserProvided: []string{"Human supplied the cache TTL requirement."}},
		FutureRetrievalContract: FutureRetrievalContract{
			Tasks:           []string{"implement product list cache"},
			Phases:          []string{"implementation"},
			Files:           []string{"internal/catalog/products.go"},
			FileGlobs:       []string{"internal/catalog/**"},
			QueryPhrases:    []string{findQuery},
			Projects:        []string{"mall-api"},
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"catalog"},
			Tags:            []string{"cache"},
			ExpectedUse:     "Find this guidance before changing product list cache behavior.",
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When implementing product list cache behavior."},
			WhenNotToUse: []string{"When changing unrelated warehouse scanning behavior."},
		},
		OverlapDecision: OverlapDecision{Decision: "create_new", Reason: "No official knowledge covers product list cache TTLs."},
		Delivery:        Delivery{Path: "inbox", RequiresHumanApproval: true},
		CandidateFiles:  []CandidateFile{{Path: candidatePath, Purpose: "Candidate knowledge item.", Load: "autoload"}},
		VerificationPlan: VerificationPlan{
			ValidatePath: candidatePath,
			FindabilityScenarios: []FindabilityScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement product list cache ttl",
				Query:   findQuery,
				Files:   []string{"internal/catalog/products.go"},
			}},
		},
	}
}

func weakMatchedProposal(candidatePath string) Proposal {
	proposal := validVerifyProposal(candidatePath, "barcode scanner token")
	proposal.KnowledgeGoal = "Document generic token guidance."
	proposal.ProposedShape = ProposedShape{Kind: "item", Type: "reference", Title: "Generic token reference", ID: "reference:backend.token.v1", Path: candidatePath, Status: "draft", Priority: "may"}
	proposal.FutureRetrievalContract = FutureRetrievalContract{
		Tasks:           []string{"add warehouse barcode scanner"},
		Phases:          []string{"implementation"},
		QueryPhrases:    []string{"barcode scanner token"},
		Projects:        []string{"mall-api"},
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"catalog"},
		Tags:            []string{"token"},
		ExpectedUse:     "Find this only as weak supporting token context.",
	}
	proposal.CandidateFiles = []CandidateFile{{Path: candidatePath, Purpose: "Candidate weak token reference.", Load: "autoload"}}
	proposal.VerificationPlan = VerificationPlan{
		ValidatePath: candidatePath,
		FindabilityScenarios: []FindabilityScenario{{
			Project: "mall-api",
			Phase:   "implementation",
			Task:    "add warehouse barcode scanner",
			Query:   "barcode scanner token",
		}},
	}
	return proposal
}

func weakMatchedAuthorItem() string {
	return `---
id: reference:backend.token.v1
title: Generic token reference
type: reference
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: may
updated_at: 2026-05-02
tags: [token]
---
Token guidance applies to platform work.
`
}

func writeProposal(t *testing.T, root string, rel string, proposal Proposal) {
	t.Helper()
	data, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir proposal dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
}

func hasVerifyFinding(findings []Finding, severity string, messageContains string) bool {
	for _, finding := range findings {
		if finding.Severity == severity && strings.Contains(finding.Message, messageContains) {
			return true
		}
	}
	return false
}
