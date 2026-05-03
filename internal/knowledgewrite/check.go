package knowledgewrite

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

func LoadDesign(path string) (KnowledgeDesign, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KnowledgeDesign{}, fmt.Errorf("read design: %w", err)
	}
	var design KnowledgeDesign
	if err := json.Unmarshal(data, &design); err != nil {
		return KnowledgeDesign{}, fmt.Errorf("parse design JSON: %w", err)
	}
	return design, nil
}

func ValidateDesign(design KnowledgeDesign) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	if strings.TrimSpace(design.SchemaVersion) != KnowledgeDesignSchemaVersion {
		addFail("schema_version must be " + KnowledgeDesignSchemaVersion)
	}
	if strings.TrimSpace(design.UserRequest) == "" {
		addFail("user_request is required")
	}
	if strings.TrimSpace(design.KnowledgeGoal) == "" {
		addFail("knowledge_goal is required")
	}
	if strings.TrimSpace(design.Project) == "" {
		addFail("project is required")
	}
	if strings.TrimSpace(design.Audience.Primary) == "" {
		addFail("audience.primary is required")
	} else if !validAudiencePrimary(design.Audience.Primary) {
		addFail("audience.primary must be implementer_agent, maintainer_agent, reviewer_agent, consumer_agent, operator_agent, or cross_project_agent")
	}
	if len(nonEmpty(design.Audience.AgentActionsSupported)) == 0 {
		addReview("audience.agent_actions_supported should describe future agent actions")
	}
	if len(nonEmpty(design.Scope.Projects)) == 0 && strings.TrimSpace(design.Scope.Distribution) == "" {
		addFail("scope must include projects or distribution")
	}
	if strings.TrimSpace(design.Scope.Stability) == "" {
		addReview("scope.stability is required")
	} else if !validScopeStability(design.Scope.Stability) {
		addFail("scope.stability must be draft, experimental, reviewed, or official_candidate")
	}
	if strings.TrimSpace(design.Scope.Distribution) != "" && !validScopeDistribution(design.Scope.Distribution) {
		addFail("scope.distribution must be personal, project, small_team, multi_project, organization, or public_consumer")
	}
	validateSources(design.Sources, addFail, addReview)
	validateDraftOutput(design.DraftOutput, design.WriteBoundary, addFail, addReview)
	validateFutureUse(design.FutureUse, addFail, addReview)
	if len(nonEmpty(design.Applicability.WhenToUse)) == 0 || len(nonEmpty(design.Applicability.WhenNotToUse)) == 0 {
		addReview("applicability must include when_to_use and when_not_to_use")
	}
	validateExistingKnowledge(design.ExistingKnowledge, addFail, addReview)
	validateWriteBoundary(design.WriteBoundary, addFail)
	validateDraftFilesAndPlan(design, addFail, addReview)
	if len(nonEmpty(design.Review.Questions)) == 0 {
		addReview("review.questions should include reviewer decisions")
	}
	return findings
}

func Check(root string, req CheckRequest) (CheckResponse, error) {
	designPath, err := cleanCheckPath("design", req.DesignPath)
	if err != nil {
		return CheckResponse{}, err
	}
	draftPath, err := cleanCheckPath("draft", req.DraftPath)
	if err != nil {
		return CheckResponse{}, err
	}

	design, err := LoadDesign(filepath.Join(root, designPath))
	if err != nil {
		return CheckResponse{}, err
	}
	designFindings := ValidateDesign(design)
	response := CheckResponse{
		Result: "pass",
		Design: DesignStatus{
			Path:       filepath.ToSlash(designPath),
			Validation: statusFromFindings(designFindings),
		},
		Draft: DraftStatus{
			Path:       filepath.ToSlash(draftPath),
			Validation: "pass",
		},
		Policy:      PolicyStatus{Result: "pass"},
		Findability: FindabilityStatus{Result: "pass"},
		Findings:    append([]Finding{}, designFindings...),
	}
	response.Result = aggregateResult(response.Findings)
	if response.Design.Validation == "fail" {
		response.Draft.Validation = "not-run"
		response.Policy.Result = "not-run"
		response.Findability.Result = "not-run"
		return response, nil
	}

	reg, err := registry.Load(root)
	if err != nil {
		return CheckResponse{}, fmt.Errorf("load registry: %w", err)
	}
	draftItems, err := knowledge.LoadPath(root, draftPath)
	if err != nil {
		response.Draft.Validation = "fail"
		response.Policy.Result = "not-run"
		response.Findability.Result = "not-run"
		response.Findings = append(response.Findings, Finding{
			Severity: "fail",
			Message:  fmt.Sprintf("draft validation/load failure: %v", err),
		})
		response.Result = aggregateResult(response.Findings)
		return response, nil
	}
	response.Draft.IDs = itemIDs(draftItems)
	if len(draftItems) == 0 {
		response.Draft.Validation = "fail"
		response.Findings = append(response.Findings, Finding{Severity: "fail", Message: "draft validation/load failure: draft path loaded no knowledge items"})
	} else if validationErrs := knowledge.ValidateItems(draftItems, reg); len(validationErrs) > 0 {
		response.Draft.Validation = "review-needed"
		for _, validationErr := range validationErrs {
			response.Findings = append(response.Findings, Finding{Severity: "review-needed", Message: validationErr.Error()})
		}
	}

	policyFindings := checkPolicy(design, draftPath, draftItems)
	response.Policy.Result = statusFromFindings(policyFindings)
	response.Findings = append(response.Findings, policyFindings...)
	response.Result = aggregateResult(response.Findings)

	if response.Result != "fail" && len(draftItems) > 0 {
		if err := checkFindability(root, design, draftItems, &response); err != nil {
			return CheckResponse{}, err
		}
	} else {
		response.Findability.Result = "not-run"
	}
	response.Result = aggregateResult(response.Findings)
	return response, nil
}

func validateSources(source Sources, addFail func(string), addReview func(string)) {
	if !hasAnySource(source) {
		addReview("sources must include at least one source bucket")
	}
	if len(source.Claims) == 0 {
		addReview("sources.claims should include claim-level trust")
	}
	for i, claim := range source.Claims {
		if strings.TrimSpace(claim.Claim) == "" || strings.TrimSpace(claim.Kind) == "" || strings.TrimSpace(claim.Trust) == "" {
			addFail(fmt.Sprintf("sources.claims[%d] must include claim, kind, and trust", i))
			continue
		}
		if !validSourceClaimKind(claim.Kind) {
			addFail(fmt.Sprintf("unknown sources.claims kind at index %d", i))
		}
		if !validSourceClaimTrust(claim.Trust) {
			addFail(fmt.Sprintf("unknown sources.claims trust at index %d", i))
		}
		if claim.Kind != "question" && len(nonEmpty(claim.Source)) == 0 {
			addReview(fmt.Sprintf("sources.claims source is required at index %d", i))
		}
		if claim.Trust == "ai_suggested" && len(nonEmpty(source.Assumptions)) == 0 && !claim.RequiresReview {
			addReview("ai_suggested claim requires assumptions or review")
		}
		if claim.Trust == "user_input" && len(nonEmpty(source.UserInput)) == 0 {
			addReview("user_input claim requires user_input source")
		}
		if claim.Trust == "observed" && len(nonEmpty(source.Observed)) == 0 {
			addReview("observed claim requires observed source")
		}
		if claim.Trust == "imported" && len(nonEmpty(source.Imported)) == 0 {
			addReview("imported claim requires imported source")
		}
		if claim.Kind == "assumption" && len(nonEmpty(source.Assumptions)) == 0 {
			addReview("assumption claim requires sources.assumptions")
		}
	}
	if !sourcesHaveSubstantiveContent(source) && len(nonEmpty(source.OpenQuestions)) > 0 {
		addReview("substantive knowledge content is missing; resolve open questions before draft writing")
	}
}

func validateDraftOutput(output DraftOutput, boundary WriteBoundary, addFail func(string), addReview func(string)) {
	if strings.TrimSpace(output.Kind) == "" || strings.TrimSpace(output.Type) == "" || strings.TrimSpace(output.ID) == "" {
		addFail("draft_output must include kind, type, and id")
	}
	if strings.TrimSpace(output.Kind) != "" && !validDraftKind(output.Kind) {
		addFail("draft_output.kind must be item, package, or review")
	}
	if strings.TrimSpace(output.Type) != "" && !validDraftType(output.Type) {
		addFail("draft_output.type must be rule, decision, lesson, runbook, reference, template, checklist, package, or review")
	}
	if strings.TrimSpace(output.DraftState) == "design_only" || output.Kind == "review" {
		return
	}
	if strings.TrimSpace(output.Path) == "" {
		addFail("draft_output.path is required")
	}
	if strings.TrimSpace(output.Status) == "" || strings.TrimSpace(output.Priority) == "" {
		addFail("draft_output must include status and priority")
	}
	if strings.TrimSpace(output.Priority) == "must" && !boundary.PriorityMustApproved {
		addFail("priority: must requires explicit approval")
	}
	if strings.TrimSpace(output.EntrypointLoad) != "" && !validEntrypointLoad(output.EntrypointLoad) {
		addFail("draft_output.entrypoint_load must be start_here, read_before_implementation, read_before_review, on_demand, or reference_only")
	}
	if strings.TrimSpace(output.Rationale) == "" {
		addReview("draft_output.rationale should explain item or package choice")
	}
}

func validateFutureUse(future FutureUse, addFail func(string), addReview func(string)) {
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

func validateExistingKnowledge(existing ExistingKnowledgeDecision, addFail func(string), addReview func(string)) {
	switch strings.TrimSpace(existing.Decision) {
	case "create_new", "update_existing", "merge_with_existing", "stop":
	case "unresolved":
		addReview("existing_knowledge decision is unresolved")
	default:
		addFail("existing_knowledge.decision must be create_new, update_existing, merge_with_existing, stop, or unresolved")
	}
	if strings.TrimSpace(existing.Reason) == "" {
		addReview("existing_knowledge.reason is required")
	}
}

func validateWriteBoundary(boundary WriteBoundary, addFail func(string)) {
	if boundary.Path != "inbox" && boundary.Path != "official_review" {
		addFail("write_boundary.path must be inbox or official_review")
	}
	if !boundary.WriteRequiresReviewApproval {
		addFail("write_boundary.write_requires_review_approval must be true")
	}
	if !boundary.ReviewPacketRequired {
		addFail("write_boundary.review_packet_required must be true")
	}
	if boundary.Path == "official_review" && !boundary.OfficialWriteApproved {
		addFail("official writing requires explicit approval")
	}
}

func validateDraftFilesAndPlan(design KnowledgeDesign, addFail func(string), addReview func(string)) {
	designOnly := design.DraftOutput.DraftState == "design_only" ||
		design.DraftOutput.Kind == "review" ||
		design.ExistingKnowledge.Decision == "unresolved" &&
			!design.Review.DraftWriteApproved &&
			len(design.DraftFiles) == 0 &&
			strings.TrimSpace(design.CheckPlan.ValidatePath) == ""
	draftPath, draftPathOK := cleanKnowledgeWritePath("draft_output.path", design.DraftOutput.Path, addFail)
	validatePath, validatePathOK := cleanKnowledgeWritePath("check_plan.validate_path", design.CheckPlan.ValidatePath, addFail)
	if !designOnly && draftPathOK {
		validateWritePathBoundary("draft_output.path", draftPath, design.WriteBoundary.Path, addFail)
	}
	if validatePathOK {
		validateWritePathBoundary("check_plan.validate_path", validatePath, design.WriteBoundary.Path, addFail)
	}
	if !designOnly && draftPathOK && validatePathOK && draftPath != validatePath {
		addFail("check_plan.validate_path must match draft_output.path")
	}
	if designOnly {
		if len(design.DraftFiles) > 0 {
			addReview("design-only review should not include draft_files before approval")
		}
		if strings.TrimSpace(design.CheckPlan.ValidatePath) != "" {
			addReview("design-only review should not set check_plan.validate_path before draft approval")
		}
		addReview("design is design-only until review decisions unblock draft writing")
		return
	}
	if len(design.DraftFiles) == 0 {
		addFail("draft_files must include at least one file")
	}
	for i, file := range design.DraftFiles {
		if strings.TrimSpace(file.Path) == "" || strings.TrimSpace(file.Purpose) == "" || strings.TrimSpace(file.Load) == "" {
			addFail(fmt.Sprintf("draft_files[%d] must include path, purpose, and load", i))
			continue
		}
		field := fmt.Sprintf("draft_files[%d].path", i)
		filePath, filePathOK := cleanKnowledgeWritePath(field, file.Path, addFail)
		if !filePathOK {
			continue
		}
		validateWritePathBoundary(field, filePath, design.WriteBoundary.Path, addFail)
		if draftPathOK && !pathWithinWriteRoot(filePath, draftPath) {
			addFail(fmt.Sprintf("draft_files[%d].path must stay under draft_output.path", i))
		}
	}
	if strings.TrimSpace(design.CheckPlan.ValidatePath) == "" {
		addFail("check_plan.validate_path is required")
	}
	if len(design.CheckPlan.FindabilityChecks) == 0 {
		addReview("check_plan.findability_checks should include at least one scenario")
	}
	for i, scenario := range design.CheckPlan.FindabilityChecks {
		if strings.TrimSpace(scenario.Project) == "" || (strings.TrimSpace(scenario.Task) == "" && strings.TrimSpace(scenario.Query) == "") {
			addFail(fmt.Sprintf("check_plan.findability_checks[%d] must include project and task or query", i))
		}
	}
}

func checkPolicy(design KnowledgeDesign, draftPath string, draftItems []knowledge.Item) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	draftSlash := filepath.ToSlash(draftPath)
	switch design.WriteBoundary.Path {
	case "inbox":
		if !isInboxDraftPath(draftSlash) {
			addFail("draft path is outside approved write boundary")
		}
	case "official_review":
		if !isOfficialDraftPath(draftSlash) {
			addFail("draft path is outside approved write boundary")
		}
		if !design.Review.OfficialWriteApproved {
			addFail("official draft path requires review.official_write_approved")
		}
	default:
		addFail("approved write boundary cannot be determined")
	}
	if !design.Review.DraftWriteApproved {
		addFail("draft writing requires review.draft_write_approved")
	}
	if expected := strings.TrimSpace(design.CheckPlan.ValidatePath); expected != "" && !pathsMatch(expected, draftPath) {
		addFail("draft path must match check_plan.validate_path")
	}
	if expected := strings.TrimSpace(design.DraftOutput.Path); expected != "" && !pathsMatch(expected, draftPath) {
		addFail("draft path must match draft_output.path")
	}
	if priorityMust(design, draftItems) && !design.Review.PriorityMustApproved {
		addFail("priority: must requires review.priority_must_approved")
	}
	if isInboxDraftPath(draftSlash) {
		for _, item := range draftItems {
			if strings.TrimSpace(item.Status) != "draft" {
				addFail(fmt.Sprintf("%s: inbox drafts must use status: draft", item.Path))
			}
		}
	}
	for _, item := range draftItems {
		if expectedID := strings.TrimSpace(design.DraftOutput.ID); expectedID != "" && item.ID != expectedID {
			addFail(fmt.Sprintf("%s: draft id must match draft_output.id", item.Path))
		}
		if expectedType := strings.TrimSpace(design.DraftOutput.Type); expectedType != "" && item.Type != expectedType {
			addFail(fmt.Sprintf("%s: draft type must match draft_output.type", item.Path))
		}
		for _, project := range missingDesignProjects(item, design.Scope.Projects) {
			addReview(fmt.Sprintf("%s: draft metadata is missing future project %q", item.Path, project))
		}
		for _, domain := range missingDesignTechDomains(item, design.Scope.TechDomains) {
			addReview(fmt.Sprintf("%s: draft metadata is missing future tech domain %q", item.Path, domain))
		}
		for _, domain := range missingDesignBusinessDomains(item, design.Scope.SubjectDomains) {
			addReview(fmt.Sprintf("%s: draft metadata is missing future business domain %q", item.Path, domain))
		}
		for _, pattern := range invalidFileScopePatterns(item) {
			addReview(fmt.Sprintf("%s: invalid applies_to.files pattern %q", item.Path, pattern))
		}
		for _, file := range missingDesignFiles(item, design.Scope.Files) {
			addReview(fmt.Sprintf("%s: draft file scope is missing future file %q", item.Path, file))
		}
		for _, glob := range missingDesignFileGlobs(item, design.Scope.FileGlobs) {
			addReview(fmt.Sprintf("%s: draft file scope is missing future file_glob %q", item.Path, glob))
		}
	}
	return findings
}

func checkFindability(root string, design KnowledgeDesign, draftItems []knowledge.Item, response *CheckResponse) error {
	if len(design.CheckPlan.FindabilityChecks) == 0 {
		response.Findability.Result = "review-needed"
		response.Findings = append(response.Findings, Finding{
			Severity: "review-needed",
			Message:  "check_plan.findability_checks is empty; findability simulation was not run",
		})
		return nil
	}
	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return fmt.Errorf("load official knowledge: %w", err)
	}
	items := mergeDraftItems(official, sanitizeDraftItemsForFindability(draftItems))
	tempDir, err := os.MkdirTemp("", "argos-check-index-*")
	if err != nil {
		return fmt.Errorf("create temporary check index directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		return fmt.Errorf("rebuild temporary check index: %w", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open temporary check index: %w", err)
	}
	defer store.Close()

	service := query.New(store)
	draftIDs := stringSet(itemIDs(draftItems))
	for i, scenario := range design.CheckPlan.FindabilityChecks {
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
			Scenario: scenario,
			Result:   "pass",
			DraftID:  firstItemID(draftItems),
			Support:  found.Support.Level,
		}
		for _, item := range found.Items {
			if draftIDs[item.ID] {
				check.MatchedDraft = true
				check.MatchedIDs = append(check.MatchedIDs, item.ID)
			}
		}
		if len(check.MatchedIDs) > 0 {
			check.DraftID = check.MatchedIDs[0]
		}
		if !check.MatchedDraft || supportNeedsReview(found.Support.Level) {
			check.Result = "review-needed"
			if !check.MatchedDraft || found.Support.Level == "none" {
				response.Findings = append(response.Findings, Finding{
					Severity: "review-needed",
					Message:  fmt.Sprintf("draft is not findable for check_plan.findability_checks[%d]", i),
				})
			} else {
				response.Findings = append(response.Findings, Finding{
					Severity: "review-needed",
					Message:  fmt.Sprintf("draft findability support is %s for check_plan.findability_checks[%d]", found.Support.Level, i),
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

func cleanCheckPath(field string, path string) (string, error) {
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

func cleanKnowledgeWritePath(field string, path string, addFail func(string)) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	if filepath.IsAbs(path) {
		addFail(field + " must be relative")
		return "", false
	}
	slash := strings.ReplaceAll(path, "\\", "/")
	for _, part := range strings.Split(slash, "/") {
		if part == ".." {
			addFail(field + " must stay inside workspace")
			return "", false
		}
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(slash)))
	if clean == "." {
		return "", false
	}
	return clean, true
}

func validateWritePathBoundary(field string, path string, boundary string, addFail func(string)) {
	switch boundary {
	case "inbox":
		if !isInboxDraftPath(path) {
			addFail(field + " is outside inbox write boundary")
		}
	case "official_review":
		if !isOfficialDraftPath(path) {
			addFail(field + " is outside official_review write boundary")
		}
	}
}

func pathWithinWriteRoot(path string, root string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	root = strings.TrimSuffix(filepath.ToSlash(strings.TrimSpace(root)), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func pathsMatch(expected string, actual string) bool {
	cleanExpected, err := cleanCheckPath("design path", expected)
	if err != nil {
		return false
	}
	return filepath.ToSlash(cleanExpected) == filepath.ToSlash(actual)
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

func priorityMust(design KnowledgeDesign, items []knowledge.Item) bool {
	if strings.TrimSpace(design.DraftOutput.Priority) == "must" {
		return true
	}
	for _, item := range items {
		if strings.TrimSpace(item.Priority) == "must" {
			return true
		}
	}
	return false
}

func nonEmpty(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func hasAnySource(source Sources) bool {
	return len(nonEmpty(source.UserInput)) > 0 ||
		len(nonEmpty(source.Observed)) > 0 ||
		len(nonEmpty(source.Imported)) > 0 ||
		len(nonEmpty(source.AISuggested)) > 0 ||
		len(nonEmpty(source.Templates)) > 0 ||
		len(nonEmpty(source.Examples)) > 0
}

func sourcesHaveSubstantiveContent(source Sources) bool {
	for _, claim := range source.Claims {
		if claim.RequiresReview || claim.Kind == "question" {
			continue
		}
		if strings.TrimSpace(claim.Claim) != "" && claim.Kind != "assumption" {
			return true
		}
	}
	return len(nonEmpty(source.Observed)) > 0 ||
		len(nonEmpty(source.Imported)) > 0 ||
		len(nonEmpty(source.AISuggested)) > 0 ||
		len(nonEmpty(source.Templates)) > 0 ||
		len(nonEmpty(source.Examples)) > 0
}

func validSourceClaimKind(kind string) bool {
	switch kind {
	case "fact", "decision", "recommendation", "example", "template", "assumption", "question":
		return true
	default:
		return false
	}
}

func validSourceClaimTrust(trust string) bool {
	switch trust {
	case "observed", "user_input", "imported", "ai_suggested", "illustrative", "unknown":
		return true
	default:
		return false
	}
}

func validAudiencePrimary(primary string) bool {
	switch primary {
	case "implementer_agent", "maintainer_agent", "reviewer_agent", "consumer_agent", "operator_agent", "cross_project_agent":
		return true
	default:
		return false
	}
}

func validScopeStability(stability string) bool {
	switch stability {
	case "draft", "experimental", "reviewed", "official_candidate":
		return true
	default:
		return false
	}
}

func validScopeDistribution(distribution string) bool {
	switch distribution {
	case "personal", "project", "small_team", "multi_project", "organization", "public_consumer":
		return true
	default:
		return false
	}
}

func validDraftKind(kind string) bool {
	switch kind {
	case "item", "package", "review":
		return true
	default:
		return false
	}
}

func validDraftType(draftType string) bool {
	switch draftType {
	case "rule", "decision", "lesson", "runbook", "reference", "template", "checklist", "package", "review":
		return true
	default:
		return false
	}
}

func validEntrypointLoad(load string) bool {
	switch load {
	case "start_here", "read_before_implementation", "read_before_review", "on_demand", "reference_only":
		return true
	default:
		return false
	}
}

func missingDesignProjects(item knowledge.Item, projects []string) []string {
	var missing []string
	for _, project := range nonEmpty(projects) {
		if !containsStringValue(item.Projects, project) {
			missing = append(missing, project)
		}
	}
	return missing
}

func missingDesignTechDomains(item knowledge.Item, domains []string) []string {
	var missing []string
	for _, domain := range nonEmpty(domains) {
		if !containsStringValue(item.TechDomains, domain) {
			missing = append(missing, domain)
		}
	}
	return missing
}

func missingDesignBusinessDomains(item knowledge.Item, domains []string) []string {
	var missing []string
	for _, domain := range nonEmpty(domains) {
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
		if _, err := doublestar.PathMatch(pattern, "argos/check/probe"); err != nil {
			if !seen[pattern] {
				seen[pattern] = true
				invalid = append(invalid, pattern)
			}
		}
	}
	return invalid
}

func missingDesignFiles(item knowledge.Item, files []string) []string {
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

func missingDesignFileGlobs(item knowledge.Item, globs []string) []string {
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

func sanitizeDraftItemsForFindability(items []knowledge.Item) []knowledge.Item {
	sanitized := make([]knowledge.Item, 0, len(items))
	for _, item := range items {
		cleaned := item
		var patterns []string
		for _, pattern := range item.AppliesTo.Files {
			if _, err := doublestar.PathMatch(pattern, "argos/check/probe"); err == nil {
				patterns = append(patterns, pattern)
			}
		}
		cleaned.AppliesTo.Files = patterns
		sanitized = append(sanitized, cleaned)
	}
	return sanitized
}

func mergeDraftItems(official []knowledge.Item, drafts []knowledge.Item) []knowledge.Item {
	draftIDs := stringSet(itemIDs(drafts))
	items := make([]knowledge.Item, 0, len(official)+len(drafts))
	for _, item := range official {
		if draftIDs[item.ID] {
			continue
		}
		items = append(items, item)
	}
	items = append(items, drafts...)
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

func supportNeedsReview(level string) bool {
	return level != "strong" && level != "partial"
}
