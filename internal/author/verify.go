package author

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/query"
	"argos/internal/registry"
)

type VerifyRequest struct {
	ProposalPath  string `json:"proposal_path"`
	CandidatePath string `json:"candidate_path"`
}

type VerifyResponse struct {
	Result      string            `json:"result"`
	Proposal    ProposalStatus    `json:"proposal"`
	Candidate   CandidateStatus   `json:"candidate"`
	Policy      PolicyStatus      `json:"policy"`
	Findability FindabilityStatus `json:"findability"`
	Findings    []Finding         `json:"findings,omitempty"`
}

type ProposalStatus struct {
	Path       string `json:"path"`
	Validation string `json:"validation"`
}

type CandidateStatus struct {
	Path       string   `json:"path"`
	Validation string   `json:"validation"`
	IDs        []string `json:"ids,omitempty"`
}

type PolicyStatus struct {
	Result string `json:"result"`
}

type FindabilityStatus struct {
	Result string             `json:"result"`
	Checks []FindabilityCheck `json:"checks,omitempty"`
}

type FindabilityCheck struct {
	Scenario         FindabilityScenario `json:"scenario"`
	Result           string              `json:"result"`
	CandidateID      string              `json:"candidate_id,omitempty"`
	MatchedCandidate bool                `json:"matched_candidate"`
	MatchedIDs       []string            `json:"matched_ids,omitempty"`
	Support          string              `json:"support"`
}

func Verify(root string, req VerifyRequest) (VerifyResponse, error) {
	proposalPath, err := cleanVerifyPath("proposal_path", req.ProposalPath)
	if err != nil {
		return VerifyResponse{}, err
	}
	candidatePath, err := cleanVerifyPath("candidate_path", req.CandidatePath)
	if err != nil {
		return VerifyResponse{}, err
	}

	proposal, proposalFindings, schemaVersion, err := loadProposalForVerify(filepath.Join(root, proposalPath))
	if err != nil {
		return VerifyResponse{}, err
	}

	response := VerifyResponse{
		Result: "pass",
		Proposal: ProposalStatus{
			Path:       filepath.ToSlash(proposalPath),
			Validation: "pass",
		},
		Candidate: CandidateStatus{
			Path:       filepath.ToSlash(candidatePath),
			Validation: "pass",
		},
		Policy:      PolicyStatus{Result: "pass"},
		Findability: FindabilityStatus{Result: "pass"},
	}

	response.Proposal.Validation = statusFromFindings(proposalFindings)
	response.Findings = append(response.Findings, proposalFindings...)
	response.Result = aggregateResult(response.Findings)
	if schemaVersion != ProposalSchemaVersion && schemaVersion != ProposalV2SchemaVersion {
		response.Candidate.Validation = "not-run"
		response.Policy.Result = "not-run"
		response.Findability.Result = "not-run"
		return response, nil
	}

	reg, err := registry.Load(root)
	if err != nil {
		return VerifyResponse{}, fmt.Errorf("load registry: %w", err)
	}
	candidateItems, err := knowledge.LoadPath(root, candidatePath)
	if err != nil {
		response.Candidate.Validation = "fail"
		response.Policy.Result = "not-run"
		response.Findability.Result = "not-run"
		response.Findings = append(response.Findings, Finding{
			Severity: "fail",
			Message:  fmt.Sprintf("candidate validation/load failure: %v", err),
		})
		response.Result = aggregateResult(response.Findings)
		return response, nil
	}
	response.Candidate.IDs = itemIDs(candidateItems)
	if len(candidateItems) == 0 {
		response.Candidate.Validation = "fail"
		response.Findings = append(response.Findings, Finding{Severity: "fail", Message: "candidate validation/load failure: candidate_path loaded no knowledge items"})
	} else if validationErrs := knowledge.ValidateItems(candidateItems, reg); len(validationErrs) > 0 {
		response.Candidate.Validation = "fail"
		for _, validationErr := range validationErrs {
			response.Findings = append(response.Findings, Finding{Severity: "fail", Message: validationErr.Error()})
		}
	}

	policyFindings := verifyPolicy(proposal, candidatePath, candidateItems)
	response.Policy.Result = statusFromFindings(policyFindings)
	response.Findings = append(response.Findings, policyFindings...)
	response.Result = aggregateResult(response.Findings)

	if response.Result != "fail" && len(candidateItems) > 0 {
		if err := verifyFindability(root, proposal, candidateItems, &response); err != nil {
			return VerifyResponse{}, err
		}
	} else {
		response.Findability.Result = "not-run"
	}

	response.Result = aggregateResult(response.Findings)
	return response, nil
}

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
	schemaVersion := strings.TrimSpace(header.SchemaVersion)

	switch schemaVersion {
	case ProposalSchemaVersion:
		var proposal Proposal
		if err := json.Unmarshal(data, &proposal); err != nil {
			return Proposal{}, nil, schemaVersion, fmt.Errorf("parse proposal JSON: %w", err)
		}
		return proposal, ValidateProposal(proposal), schemaVersion, nil
	case ProposalV2SchemaVersion:
		var proposal ProposalV2
		if err := json.Unmarshal(data, &proposal); err != nil {
			return Proposal{}, nil, schemaVersion, fmt.Errorf("parse proposal JSON: %w", err)
		}
		return NormalizeProposalV2(proposal), ValidateProposalV2(proposal), schemaVersion, nil
	default:
		return Proposal{}, []Finding{{
			Severity: "fail",
			Message:  "schema_version must be authoring.proposal.v1 or authoring.\x70roposal.v2",
		}}, schemaVersion, nil
	}
}

func cleanVerifyPath(field string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be relative", field)
	}
	if hasParentSegment(path) {
		return "", fmt.Errorf("%s must stay inside workspace", field)
	}
	clean := filepath.Clean(path)
	if clean == "." {
		return "", fmt.Errorf("%s is required", field)
	}
	return clean, nil
}

func statusFromFindings(findings []Finding) string {
	status := "pass"
	for _, finding := range findings {
		switch finding.Severity {
		case "fail":
			return "fail"
		case "review-needed":
			status = "review-needed"
		}
	}
	return status
}

func aggregateResult(findings []Finding) string {
	return statusFromFindings(findings)
}

func itemIDs(items []knowledge.Item) []string {
	var ids []string
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func verifyPolicy(proposal Proposal, candidatePath string, candidateItems []knowledge.Item) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	candidateSlash := filepath.ToSlash(candidatePath)
	switch proposal.Delivery.Path {
	case "inbox":
		if !isInboxCandidatePath(candidateSlash) {
			addFail("candidate path is outside approved delivery boundary for inbox delivery")
		}
	case "official_review":
		if !isOfficialCandidatePath(candidateSlash) {
			addFail("candidate path is outside approved delivery boundary for official_review delivery")
		}
	default:
		addFail("approved delivery boundary cannot be determined for delivery path")
	}
	if expected := strings.TrimSpace(proposal.VerificationPlan.ValidatePath); expected != "" {
		if !pathsMatch(expected, candidatePath) {
			addFail("candidate path must match verification_\x70lan.validate_path")
		}
	}
	if expected := strings.TrimSpace(proposal.ProposedShape.Path); expected != "" {
		if !pathsMatch(expected, candidatePath) {
			addFail("candidate path must match proposed_\x73hape.path")
		}
	}
	if priorityMust(proposal, candidateItems) && !proposal.Delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	if isInboxCandidatePath(candidateSlash) {
		for _, item := range candidateItems {
			if strings.TrimSpace(item.Status) != "draft" {
				addFail(fmt.Sprintf("%s: inbox candidates must use status: draft", item.Path))
			}
		}
	}
	for _, item := range candidateItems {
		if expectedID := strings.TrimSpace(proposal.ProposedShape.ID); expectedID != "" && item.ID != expectedID {
			addFail(fmt.Sprintf("%s: candidate id must match proposed_\x73hape.id", item.Path))
		}
		if expectedType := strings.TrimSpace(proposal.ProposedShape.Type); expectedType != "" && item.Type != expectedType {
			addFail(fmt.Sprintf("%s: candidate type must match proposed_\x73hape.type", item.Path))
		}
		for _, project := range missingContractProjects(item, proposal.FutureRetrievalContract) {
			addReview(fmt.Sprintf("%s: candidate metadata is missing future retrieval project %q", item.Path, project))
		}
		for _, tag := range missingContractTags(item, proposal.FutureRetrievalContract) {
			addReview(fmt.Sprintf("%s: candidate metadata is missing future retrieval tag %q", item.Path, tag))
		}
		for _, domain := range missingContractTechDomains(item, proposal.FutureRetrievalContract) {
			addReview(fmt.Sprintf("%s: candidate metadata is missing future retrieval tech domain %q", item.Path, domain))
		}
		for _, domain := range missingContractBusinessDomains(item, proposal.FutureRetrievalContract) {
			addReview(fmt.Sprintf("%s: candidate metadata is missing future retrieval business domain %q", item.Path, domain))
		}
		for _, pattern := range invalidFileScopePatterns(item) {
			addReview(fmt.Sprintf("%s: invalid applies_to.files pattern %q", item.Path, pattern))
		}
		for _, file := range missingContractFiles(item, proposal.FutureRetrievalContract.Files) {
			addReview(fmt.Sprintf("%s: candidate file scope is missing future retrieval file %q", item.Path, file))
		}
		for _, glob := range missingContractFileGlobs(item, proposal.FutureRetrievalContract.FileGlobs) {
			addReview(fmt.Sprintf("%s: candidate file scope is missing future retrieval file_glob %q", item.Path, glob))
		}
	}

	return findings
}

func isInboxCandidatePath(path string) bool {
	return strings.HasPrefix(path, "knowledge/.inbox/items/") ||
		strings.HasPrefix(path, "knowledge/.inbox/packages/")
}

func isOfficialCandidatePath(path string) bool {
	return strings.HasPrefix(path, "knowledge/items/") ||
		strings.HasPrefix(path, "knowledge/packages/")
}

func pathsMatch(expected string, actual string) bool {
	cleanExpected, err := cleanVerifyPath("proposal path", expected)
	if err != nil {
		return false
	}
	return filepath.ToSlash(cleanExpected) == filepath.ToSlash(actual)
}

func priorityMust(proposal Proposal, items []knowledge.Item) bool {
	if strings.TrimSpace(proposal.ProposedShape.Priority) == "must" {
		return true
	}
	for _, item := range items {
		if strings.TrimSpace(item.Priority) == "must" {
			return true
		}
	}
	return false
}

func missingContractProjects(item knowledge.Item, contract FutureRetrievalContract) []string {
	contractProjects := nonEmpty(contract.Projects)
	var missing []string
	for _, project := range contractProjects {
		if !containsStringValue(item.Projects, project) {
			missing = append(missing, project)
		}
	}
	return missing
}

func missingContractTags(item knowledge.Item, contract FutureRetrievalContract) []string {
	contractTags := nonEmpty(contract.Tags)
	var missing []string
	for _, tag := range contractTags {
		if !containsStringValue(item.Tags, tag) {
			missing = append(missing, tag)
		}
	}
	return missing
}

func missingContractTechDomains(item knowledge.Item, contract FutureRetrievalContract) []string {
	var missing []string
	for _, domain := range nonEmpty(contract.TechDomains) {
		if !containsStringValue(item.TechDomains, domain) {
			missing = append(missing, domain)
		}
	}
	return missing
}

func missingContractBusinessDomains(item knowledge.Item, contract FutureRetrievalContract) []string {
	var missing []string
	for _, domain := range nonEmpty(contract.BusinessDomains) {
		if !containsStringValue(item.BusinessDomains, domain) {
			missing = append(missing, domain)
		}
	}
	return missing
}

func invalidFileScopePatterns(item knowledge.Item) []string {
	seen := map[string]bool{}
	var invalid []string
	for _, pattern := range nonEmpty(item.AppliesTo.Files) {
		if _, err := doublestar.PathMatch(pattern, "argos/verify/probe"); err != nil {
			if !seen[pattern] {
				seen[pattern] = true
				invalid = append(invalid, pattern)
			}
		}
	}
	return invalid
}

func missingContractFiles(item knowledge.Item, files []string) []string {
	var missing []string
	patterns := nonEmpty(item.AppliesTo.Files)
	for _, file := range nonEmpty(files) {
		matched := false
		for _, pattern := range patterns {
			ok, err := doublestar.PathMatch(pattern, file)
			if err != nil {
				continue
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, file)
		}
	}
	return missing
}

func missingContractFileGlobs(item knowledge.Item, globs []string) []string {
	var missing []string
	for _, glob := range nonEmpty(globs) {
		if !containsExactStringValue(item.AppliesTo.Files, glob) {
			missing = append(missing, glob)
		}
	}
	return missing
}

func containsExactStringValue(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func verifyFindability(root string, proposal Proposal, candidateItems []knowledge.Item, response *VerifyResponse) error {
	if len(proposal.VerificationPlan.FindabilityScenarios) == 0 {
		response.Findability.Result = "review-needed"
		response.Findings = append(response.Findings, Finding{
			Severity: "review-needed",
			Message:  "verification_\x70lan.findability_scenarios is empty; findability simulation was not run",
		})
		return nil
	}

	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return fmt.Errorf("load official knowledge: %w", err)
	}
	items := mergeCandidateItems(official, sanitizeCandidateItemsForFindability(candidateItems))

	tempDir, err := os.MkdirTemp("", "argos-verify-index-*")
	if err != nil {
		return fmt.Errorf("create temporary verification index directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		return fmt.Errorf("rebuild temporary verification index: %w", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open temporary verification index: %w", err)
	}
	defer store.Close()

	service := query.New(store)
	candidateIDs := stringSet(itemIDs(candidateItems))
	for i, scenario := range proposal.VerificationPlan.FindabilityScenarios {
		found, err := service.FindKnowledge(query.FindKnowledgeRequest{
			Project: strings.TrimSpace(scenario.Project),
			Phase:   strings.TrimSpace(scenario.Phase),
			Task:    strings.TrimSpace(scenario.Task),
			Query:   strings.TrimSpace(scenario.Query),
			Files:   scenario.Files,
			Limit:   8,
		})
		if err != nil {
			return fmt.Errorf("run findability scenario %d: %w", i, err)
		}

		check := FindabilityCheck{
			Scenario:    scenario,
			Result:      "pass",
			CandidateID: firstItemID(candidateItems),
			Support:     found.Support.Level,
		}
		for _, item := range found.Items {
			if candidateIDs[item.ID] {
				check.MatchedCandidate = true
				check.MatchedIDs = append(check.MatchedIDs, item.ID)
			}
		}
		if len(check.MatchedIDs) > 0 {
			check.CandidateID = check.MatchedIDs[0]
		}
		if !check.MatchedCandidate || supportNeedsReview(found.Support.Level) {
			check.Result = "review-needed"
			if !check.MatchedCandidate || found.Support.Level == "none" {
				response.Findings = append(response.Findings, Finding{
					Severity: "review-needed",
					Message:  fmt.Sprintf("candidate is not findable for verification_\x70lan.findability_scenarios[%d]", i),
				})
			} else {
				response.Findings = append(response.Findings, Finding{
					Severity: "review-needed",
					Message:  fmt.Sprintf("candidate findability support is %s for verification_\x70lan.findability_scenarios[%d]", found.Support.Level, i),
				})
			}
		}
		response.Findability.Checks = append(response.Findability.Checks, check)
	}

	response.Findability.Result = "pass"
	for _, check := range response.Findability.Checks {
		if check.Result == "review-needed" {
			response.Findability.Result = "review-needed"
			return nil
		}
	}
	return nil
}

func supportNeedsReview(level string) bool {
	return level != "strong" && level != "partial"
}

func sanitizeCandidateItemsForFindability(items []knowledge.Item) []knowledge.Item {
	sanitized := make([]knowledge.Item, 0, len(items))
	for _, item := range items {
		cleaned := item
		var patterns []string
		for _, pattern := range item.AppliesTo.Files {
			if _, err := doublestar.PathMatch(pattern, "argos/verify/probe"); err == nil {
				patterns = append(patterns, pattern)
			}
		}
		cleaned.AppliesTo.Files = patterns
		sanitized = append(sanitized, cleaned)
	}
	return sanitized
}

func mergeCandidateItems(official []knowledge.Item, candidates []knowledge.Item) []knowledge.Item {
	candidateIDs := stringSet(itemIDs(candidates))
	items := make([]knowledge.Item, 0, len(official)+len(candidates))
	for _, item := range official {
		if candidateIDs[item.ID] {
			continue
		}
		items = append(items, item)
	}
	items = append(items, candidates...)
	return items
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = true
		}
	}
	return set
}

func firstItemID(items []knowledge.Item) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[0].ID)
}
