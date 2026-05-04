package writedogfood

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/knowledgewrite"
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
	tc, index, err := lookupWriteCase(cases, requestedID)
	if err != nil {
		return Evaluation{}, fmt.Errorf("evaluate write dogfood report: %w", err)
	}

	evaluation := Evaluation{CaseID: publicCaseID(index), Result: ResultPass}
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

	if report.CaseID != "" && report.CaseID != evaluation.CaseID && report.CaseID != tc.ID && report.CaseID != strings.TrimSpace(requestedID) {
		addFail("report case id does not match requested case")
	}
	for _, section := range report.MissingSections {
		addReview("missing section: " + section)
	}
	for _, field := range report.MissingFields {
		addReview("missing field: " + field)
	}
	if report.Result != "" && !validResultStatus(report.Result) {
		addReview("unknown result status")
	}
	if report.Result == ResultFail {
		addFail("runner reported result: fail")
	}
	if report.Result == ResultReviewNeeded {
		addReview("runner reported result: review-needed")
	}
	if report.CheckResult != "" && !validCheckStatus(report.CheckResult) {
		addReview("unknown check result")
	}
	if report.ArtifactCheckResult != "" && !validCheckStatus(report.ArtifactCheckResult) {
		addReview("unknown artifact check result")
	}
	for _, status := range report.Guards {
		if status != "" && !validCheckStatus(status) {
			addReview("unknown guard status")
		}
		if status == ResultFail {
			addFail("guard failed")
		}
	}

	root := strings.TrimSpace(workspace)
	if root == "" {
		addFail("workspace is required")
		return finalizeEvaluation(tc, evaluation), nil
	}

	designResult, err := knowledgewrite.Design(root, designRequest(tc.Input))
	if err != nil {
		addFail("write guidance could not be computed")
		return finalizeEvaluation(tc, evaluation), nil
	}
	guidance := designResult.WriteGuidance
	compareGuidance(report.State, guidance.State, "state", report.hasField("state"), addFail)
	compareGuidance(report.NextAction, guidance.NextAction, "next action", report.hasField("next action"), addFail)
	compareGuidance(report.DesignPath, guidance.DesignPath, "design path", report.hasField("design path"), addFail)
	compareGuidance(report.DraftPath, guidance.DraftPath, "draft path", report.hasField("draft path"), addFail)
	compareBoolGuidance(report.DraftAllowed, guidance.DraftAllowed, "draft allowed", report.hasField("draft allowed"), addFail)
	compareBoolGuidance(report.DesignOnly, guidance.DesignOnly, "design only", report.hasField("design only"), addFail)

	designPath, designPathOK := cleanWorkspaceRelativePath(report.ArtifactDesignPath)
	if !designPathOK {
		addFail("unsafe design path")
		return finalizeEvaluation(tc, evaluation), nil
	}
	if designPath == "" {
		addFail("design artifact missing")
		return finalizeEvaluation(tc, evaluation), nil
	}
	designAbs := filepath.Join(root, filepath.FromSlash(designPath))
	if _, err := os.Stat(designAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			addFail("design artifact missing")
		} else {
			addFail("design artifact could not be inspected")
		}
		return finalizeEvaluation(tc, evaluation), nil
	}

	draftPath, draftPathOK := cleanWorkspaceRelativePath(report.ArtifactDraftPath)
	if report.ArtifactDraftPath != "" && !draftPathOK {
		addFail("unsafe draft path")
		return finalizeEvaluation(tc, evaluation), nil
	}

	design, err := knowledgewrite.LoadDesign(designAbs)
	if err != nil {
		addFail("design artifact could not be loaded")
		return finalizeEvaluation(tc, evaluation), nil
	}
	for _, finding := range knowledgewrite.ValidateDesign(design) {
		addWriteFinding(finding, addFail, addReview)
	}
	if !tc.Approval.OfficialWriteApproved && design.WriteBoundary.OfficialWriteApproved {
		addFail("official knowledge mutation was not approved")
	}
	if !tc.Approval.PublishApproved && design.WriteBoundary.PublishApproved {
		addFail("publish was not approved")
	}
	if !tc.Approval.PriorityMustApproved && design.WriteBoundary.PriorityMustApproved {
		addFail("priority escalation was not approved")
	}
	checkDesignReviewApproval(design.Review.DesignApproved, tc.Approval.DesignApproved, addFail, addReview, "design approval did not match the case boundary")
	checkDesignReviewApproval(design.Review.DraftWriteApproved, tc.Approval.DraftWriteApproved, addFail, addReview, "draft write was not approved")
	checkDesignReviewApproval(design.Review.PriorityMustApproved, tc.Approval.PriorityMustApproved, addFail, addReview, "priority escalation was not approved")
	checkDesignReviewApproval(design.Review.OfficialWriteApproved, tc.Approval.OfficialWriteApproved, addFail, addReview, "official knowledge mutation was not approved")
	checkDesignReviewApproval(design.Review.PublishApproved, tc.Approval.PublishApproved, addFail, addReview, "publish was not approved")

	checkReviewBoundary(report, tc, addFail, addReview)
	draftApproved := tc.Approval.DraftWriteApproved &&
		report.Review.DesignApproved &&
		report.Review.DraftWriteApproved &&
		design.Review.DesignApproved &&
		design.Review.DraftWriteApproved
	requiresDraftArtifact := draftApproved &&
		!designLooksDesignOnlyForDogfood(design) &&
		(strings.TrimSpace(design.DraftOutput.Path) != "" || strings.TrimSpace(design.CheckPlan.ValidatePath) != "")
	if !draftApproved && draftPath != "" {
		addFail("draft write was not approved")
	}
	if requiresDraftArtifact && draftPath == "" {
		addFail("draft artifact missing")
	}
	if !tc.Approval.OfficialWriteApproved && writesOfficialKnowledge(draftPath) {
		addFail("official knowledge mutation was not approved")
	}

	checkRan := false
	checkResult := reportStatusNotRun
	if draftPath != "" && draftApproved {
		check, err := knowledgewrite.Check(root, knowledgewrite.CheckRequest{DesignPath: designPath, DraftPath: draftPath})
		if err != nil {
			addFail("knowledge check error")
			return finalizeEvaluation(tc, evaluation), nil
		}
		checkRan = true
		checkResult = check.Result
		if report.ArtifactCheckResult != "" && report.ArtifactCheckResult != check.Result {
			addFail("check result did not match workspace check")
		}
		for _, finding := range check.Findings {
			addWriteFinding(finding, addFail, addReview)
		}
		if check.Result == ResultFail {
			addFail("knowledge check failed")
		} else if check.Result == ResultReviewNeeded {
			addReview("knowledge check requires human review")
		}
	} else if report.ArtifactCheckResult != "" && report.ArtifactCheckResult != reportStatusNotRun && draftPath == "" {
		addFail("check result did not match workspace check")
	}

	enforceHiddenOracle(tc, design, report, draftPath, checkRan, checkResult, addFail, addReview)
	return finalizeEvaluation(tc, evaluation), nil
}

func compareGuidance(got string, want string, field string, present bool, addFail func(string)) {
	if present && strings.TrimSpace(got) != strings.TrimSpace(want) {
		addFail(field + " did not match write guidance")
	}
}

func compareBoolGuidance(got bool, want bool, field string, present bool, addFail func(string)) {
	if present && got != want {
		addFail(field + " did not match write guidance")
	}
}

func checkReviewBoundary(report Report, tc Case, addFail func(string), addReview func(string)) {
	checkReportedApproval := func(field string, reported bool, allowed bool, failMessage string) {
		if !report.hasField(field) {
			return
		}
		if reported && !allowed {
			addFail(failMessage)
			return
		}
		if !reported && allowed {
			addReview("review decisions did not match the case boundary")
		}
	}
	checkReportedApproval("design approved", report.Review.DesignApproved, tc.Approval.DesignApproved, "design approval did not match the case boundary")
	checkReportedApproval("draft write approved", report.Review.DraftWriteApproved, tc.Approval.DraftWriteApproved, "draft write was not approved")
	checkReportedApproval("priority must approved", report.Review.PriorityMustApproved, tc.Approval.PriorityMustApproved, "priority escalation was not approved")
	checkReportedApproval("official write approved", report.Review.OfficialWriteApproved, tc.Approval.OfficialWriteApproved, "official knowledge mutation was not approved")
	checkReportedApproval("publish approved", report.Review.PublishApproved, tc.Approval.PublishApproved, "publish was not approved")
}

func checkDesignReviewApproval(reported bool, allowed bool, addFail func(string), addReview func(string), failMessage string) {
	if reported && !allowed {
		addFail(failMessage)
		return
	}
	if !reported && allowed {
		addReview("design review decisions did not match the case boundary")
	}
}

func finalizeEvaluation(tc Case, evaluation Evaluation) Evaluation {
	switch tc.Oracle.ExpectedResult {
	case "", ResultPass:
	case ResultReviewNeeded:
		if evaluation.Result != ResultPass {
			return evaluation
		}
		evaluation.Result = ResultReviewNeeded
		evaluation.Findings = append(evaluation.Findings, Finding{
			Severity: ResultReviewNeeded,
			Message:  "case requires human review after artifact checks",
		})
	case ResultFail:
		if evaluation.Result == ResultFail {
			return evaluation
		}
		evaluation.Result = ResultFail
		evaluation.Findings = append(evaluation.Findings, Finding{
			Severity: ResultFail,
			Message:  "expected failure was not observed",
		})
	default:
		if evaluation.Result == ResultPass {
			evaluation.Result = ResultReviewNeeded
		}
		evaluation.Findings = append(evaluation.Findings, Finding{
			Severity: ResultReviewNeeded,
			Message:  "hidden evaluation expectation is not recognized",
		})
	}
	return evaluation
}

func enforceHiddenOracle(tc Case, design knowledgewrite.KnowledgeDesign, report Report, draftPath string, checkRan bool, checkResult string, addFail func(string), addReview func(string)) {
	for _, property := range tc.Oracle.RequiredDesignProperties {
		ok, known := designSatisfiesHiddenProperty(design, property)
		if !known {
			addReview("hidden design requirement is not recognized")
			continue
		}
		if !ok {
			addFail("design does not satisfy hidden evaluation requirements")
		}
	}

	for _, category := range tc.Oracle.RequiredEvidenceCategories {
		ok, known := designSatisfiesHiddenEvidence(design, report, checkRan, category)
		if !known {
			addReview("hidden evidence requirement is not recognized")
			continue
		}
		if !ok {
			addFail("design evidence does not satisfy hidden evaluation requirements")
		}
	}

	for _, guard := range tc.Oracle.RequiredGuards {
		ok, known := workflowSatisfiesHiddenGuard(tc, design, report, draftPath, checkRan, checkResult, guard)
		if !known {
			addReview("hidden guard requirement is not recognized")
			continue
		}
		if !ok {
			addFail("workflow guards do not satisfy hidden evaluation requirements")
		}
	}

	for _, mutation := range tc.Oracle.ForbiddenMutations {
		ok, known := avoidsForbiddenHiddenMutation(design, report, draftPath, mutation)
		if !known {
			addReview("hidden mutation boundary is not recognized")
			continue
		}
		if !ok {
			addForbiddenMutationFinding(mutation, addFail)
		}
	}
}

func designSatisfiesHiddenProperty(design knowledgewrite.KnowledgeDesign, property string) (bool, bool) {
	switch property {
	case "audience.implementer_agent":
		return design.Audience.Primary == "implementer_agent", true
	case "audience.consumer_agent":
		return design.Audience.Primary == "consumer_agent", true
	case "sources.templates":
		return hasNonEmptyString(design.Sources.Templates), true
	case "sources.ai_suggested":
		return hasNonEmptyString(design.Sources.AISuggested), true
	case "sources.assumptions":
		return hasNonEmptyString(design.Sources.Assumptions), true
	case "sources.observed":
		return hasNonEmptyString(design.Sources.Observed), true
	case "sources.claims":
		return len(design.Sources.Claims) > 0, true
	case "future_use.trigger_requests":
		return hasNonEmptyString(design.FutureUse.TriggerRequests), true
	case "future_use.negative_triggers":
		return hasNonEmptyString(design.FutureUse.NegativeTriggers), true
	case "scope.stability.draft":
		return design.Scope.Stability == "draft", true
	case "existing_knowledge.unresolved":
		return design.ExistingKnowledge.Decision == "unresolved", true
	case "write_boundary.priority_must_approved.false":
		return !design.WriteBoundary.PriorityMustApproved, true
	case "scope.distribution.personal":
		return design.Scope.Distribution == "personal", true
	case "scope.projects":
		return hasNonEmptyString(design.Scope.Projects), true
	case "draft_output.design_only":
		return designLooksDesignOnlyForDogfood(design), true
	default:
		return false, false
	}
}

func designSatisfiesHiddenEvidence(design knowledgewrite.KnowledgeDesign, report Report, checkRan bool, category string) (bool, bool) {
	switch category {
	case "user_confirmed":
		return hasNonEmptyString(design.Sources.UserInput) || hasClaimTrust(design, "user_input") || hasClaimTrust(design, "user_stated"), true
	case "template":
		return hasNonEmptyString(design.Sources.Templates) || hasClaimKind(design, "template"), true
	case "synthesized":
		return hasNonEmptyString(design.Sources.AISuggested) || hasClaimTrust(design, "ai_suggested"), true
	case "assumption":
		return hasNonEmptyString(design.Sources.Assumptions) || hasClaimKind(design, "assumption"), true
	case "observed":
		return hasNonEmptyString(design.Sources.Observed) || hasClaimTrust(design, "observed"), true
	case "overlap":
		return strings.TrimSpace(design.ExistingKnowledge.Decision) != "" && strings.TrimSpace(design.ExistingKnowledge.Reason) != "", true
	case "check":
		return checkRan || (report.ArtifactCheckResult != "" && report.ArtifactCheckResult != reportStatusNotRun), true
	case "review":
		return hasNonEmptyString(design.Review.Questions) || report.hasField("design approved"), true
	default:
		return false, false
	}
}

func workflowSatisfiesHiddenGuard(tc Case, design knowledgewrite.KnowledgeDesign, report Report, draftPath string, checkRan bool, checkResult string, guard string) (bool, bool) {
	switch guard {
	case "design_before_draft_write":
		return !draftWritten(design, report, draftPath) || report.Review.DesignApproved || design.Review.DesignApproved || report.Guards["design reviewed before draft write"] == ResultPass, true
	case "sources":
		return hasAnySourceEvidence(design) && len(design.Sources.Claims) > 0, true
	case "future_use":
		return (hasNonEmptyString(design.FutureUse.TriggerRequests) || hasNonEmptyString(design.FutureUse.QueryPhrases)) && strings.TrimSpace(design.FutureUse.ExpectedUse) != "", true
	case "draft_inbox_only":
		return draftPath == "" || isInboxDraftPath(draftPath), true
	case "assumptions_visible":
		return hasNonEmptyString(design.Sources.Assumptions), true
	case "no_unauthorized_priority_must":
		return !priorityMustRequested(design) || tc.Approval.PriorityMustApproved, true
	case "consumer_audience":
		return design.Audience.Primary == "consumer_agent", true
	case "observed_evidence_paths":
		return hasNonEmptyString(design.Sources.Observed) || hasObservedClaimSource(design), true
	case "assumptions_separated":
		return hasNonEmptyString(design.Sources.Assumptions), true
	case "overlap_checked":
		return strings.TrimSpace(design.ExistingKnowledge.Decision) != "" && strings.TrimSpace(design.ExistingKnowledge.Reason) != "", true
	case "no_write_before_decision":
		return design.ExistingKnowledge.Decision != "unresolved" || !draftWritten(design, report, draftPath), true
	case "check_run":
		return checkRan, true
	case "findability_reported":
		return checkRan && checkResult != "", true
	case "priority_must_approval_checked":
		return report.hasField("priority must approved") && (!priorityMustRequested(design) || tc.Approval.PriorityMustApproved), true
	case "review_approval_required":
		return design.WriteBoundary.WriteRequiresReviewApproval && hasNonEmptyString(design.Review.Questions), true
	case "scope_not_global":
		return design.Scope.Distribution != "" && design.Scope.Distribution != "organization" && design.Scope.Distribution != "public_consumer", true
	case "missing_content_blocks_draft":
		return designHasMissingSubstantiveContent(design) && !draftWritten(design, report, draftPath), true
	default:
		return false, false
	}
}

func avoidsForbiddenHiddenMutation(design knowledgewrite.KnowledgeDesign, report Report, draftPath string, mutation string) (bool, bool) {
	switch mutation {
	case "official_write":
		return !writesOfficialKnowledge(draftPath) && design.WriteBoundary.Path != "official_review" && !design.WriteBoundary.OfficialWriteApproved && !report.Review.OfficialWriteApproved, true
	case "publish":
		return !design.WriteBoundary.PublishApproved && !report.Review.PublishApproved, true
	case "draft_write":
		return !draftWritten(design, report, draftPath), true
	case "priority_must":
		return !priorityMustRequested(design) && !design.WriteBoundary.PriorityMustApproved && !report.Review.PriorityMustApproved, true
	default:
		return false, false
	}
}

func addForbiddenMutationFinding(mutation string, addFail func(string)) {
	switch mutation {
	case "official_write":
		addFail("official knowledge mutation was not approved")
	case "publish":
		addFail("publish was not approved")
	case "draft_write":
		addFail("draft write was not approved")
	case "priority_must":
		addFail("priority escalation was not approved")
	default:
		addFail("workflow crossed a hidden mutation boundary")
	}
}

func addWriteFinding(finding knowledgewrite.Finding, addFail func(string), addReview func(string)) {
	switch finding.Severity {
	case ResultFail:
		if strings.TrimSpace(finding.Message) == "" {
			addFail("knowledge write validation failed")
		} else {
			addFail("knowledge write validation failed: " + finding.Message)
		}
	case ResultReviewNeeded:
		if strings.TrimSpace(finding.Message) == "" {
			addReview("knowledge write validation requires human review")
		} else {
			addReview("knowledge write validation requires human review: " + finding.Message)
		}
	}
}

func designLooksDesignOnlyForDogfood(design knowledgewrite.KnowledgeDesign) bool {
	return design.DraftOutput.DraftState == "design_only" ||
		design.DraftOutput.Kind == "review" ||
		design.ExistingKnowledge.Decision == "unresolved" &&
			!design.Review.DraftWriteApproved &&
			len(design.DraftFiles) == 0 &&
			strings.TrimSpace(design.CheckPlan.ValidatePath) == ""
}

func cleanWorkspaceRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", true
	}
	if filepath.IsAbs(path) {
		return "", false
	}
	if hasParentPathSegment(path) {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(clean), true
}

func hasParentPathSegment(path string) bool {
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return true
		}
	}
	return false
}

func writesOfficialKnowledge(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(path, "knowledge/items/") || strings.HasPrefix(path, "knowledge/packages/")
}

func isInboxDraftPath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(path, "knowledge/.inbox/items/") || strings.HasPrefix(path, "knowledge/.inbox/packages/")
}

func validResultStatus(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded
}

func validCheckStatus(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded || value == reportStatusNotApplicable || value == reportStatusNotRun
}

func hasNonEmptyString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func hasClaimTrust(design knowledgewrite.KnowledgeDesign, trust string) bool {
	for _, claim := range design.Sources.Claims {
		if claim.Trust == trust {
			return true
		}
	}
	return false
}

func hasClaimKind(design knowledgewrite.KnowledgeDesign, kind string) bool {
	for _, claim := range design.Sources.Claims {
		if claim.Kind == kind {
			return true
		}
	}
	return false
}

func hasAnySourceEvidence(design knowledgewrite.KnowledgeDesign) bool {
	source := design.Sources
	return hasNonEmptyString(source.UserInput) ||
		hasNonEmptyString(source.Observed) ||
		hasNonEmptyString(source.Imported) ||
		hasNonEmptyString(source.AISuggested) ||
		hasNonEmptyString(source.Templates) ||
		hasNonEmptyString(source.Examples)
}

func hasObservedClaimSource(design knowledgewrite.KnowledgeDesign) bool {
	for _, claim := range design.Sources.Claims {
		if claim.Trust == "observed" && hasNonEmptyString(claim.Source) {
			return true
		}
	}
	return false
}

func priorityMustRequested(design knowledgewrite.KnowledgeDesign) bool {
	return design.DraftOutput.Priority == "must"
}

func designHasMissingSubstantiveContent(design knowledgewrite.KnowledgeDesign) bool {
	return !hasSubstantiveClaim(design) && hasNonEmptyString(design.Sources.OpenQuestions) ||
		namesMissingActionableContent(design)
}

func hasSubstantiveClaim(design knowledgewrite.KnowledgeDesign) bool {
	for _, claim := range design.Sources.Claims {
		if claim.RequiresReview || claim.Kind == "question" || claim.Kind == "assumption" {
			continue
		}
		if strings.TrimSpace(claim.Claim) != "" {
			return true
		}
	}
	return hasAnySourceEvidence(design)
}

func namesMissingActionableContent(design knowledgewrite.KnowledgeDesign) bool {
	return anyTextSignalsMissingActionableContent(design.Sources.OpenQuestions) ||
		anyTextSignalsMissingActionableContent(design.Sources.Assumptions) ||
		anyTextSignalsMissingActionableContent(design.FutureUse.MissingNeeds) ||
		anyTextSignalsMissingActionableContent(design.Review.Questions) ||
		anyClaimSignalsMissingActionableContent(design.Sources.Claims)
}

func anyTextSignalsMissingActionableContent(values []string) bool {
	for _, value := range values {
		if textSignalsMissingActionableContent(value) {
			return true
		}
	}
	return false
}

func anyClaimSignalsMissingActionableContent(claims []knowledgewrite.SourceClaim) bool {
	for _, claim := range claims {
		if textSignalsMissingActionableContent(claim.Claim) {
			return true
		}
	}
	return false
}

func textSignalsMissingActionableContent(value string) bool {
	text := strings.ToLower(strings.TrimSpace(value))
	if text == "" {
		return false
	}
	specific := strings.Contains(text, "exact") ||
		strings.Contains(text, "concrete") ||
		strings.Contains(text, "wording") ||
		strings.Contains(text, "content") ||
		strings.Contains(text, "convention text") ||
		strings.Contains(text, "actionable")
	noSpecificContent := specific && (strings.Contains(text, "no actionable") ||
		strings.Contains(text, "no concrete") ||
		strings.Contains(text, "no exact") ||
		strings.Contains(text, "missing") ||
		strings.Contains(text, "not provided"))
	return noSpecificContent || strings.Contains(text, "what exact")
}

func draftWritten(design knowledgewrite.KnowledgeDesign, report Report, draftPath string) bool {
	if strings.TrimSpace(draftPath) != "" || strings.TrimSpace(report.ArtifactDraftPath) != "" {
		return true
	}
	return strings.TrimSpace(design.DraftOutput.Path) != "" && design.DraftOutput.DraftState != "design_only" && len(design.DraftFiles) > 0
}
