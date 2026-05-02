package authoringdogfood

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/author"
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
	tc, index, err := lookupAuthoringCase(cases, requestedID)
	if err != nil {
		return Evaluation{}, fmt.Errorf("evaluate authoring dogfood report: %w", err)
	}

	evaluation := Evaluation{
		CaseID: publicCaseID(index),
		Result: ResultPass,
	}
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
	if report.Result != "" && !validAuthoringResultStatus(report.Result) {
		addReview("unknown result status")
	}
	if report.Result == ResultFail {
		addFail("runner reported result: fail")
	}
	if report.Result == ResultReviewNeeded {
		addReview("runner reported result: review-needed")
	}
	if report.VerifyResult != "" && !validAuthoringVerificationStatus(report.VerifyResult) {
		addReview("unknown author verify result")
	}
	for _, status := range report.Guards {
		if status != "" && !validAuthoringVerificationStatus(status) {
			addReview("unknown guard status")
		}
		if status == ResultFail {
			addFail("guard failed")
		}
	}

	root := strings.TrimSpace(workspace)
	if root == "" {
		addFail("workspace is required")
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}

	proposalPath, proposalPathOK := cleanWorkspaceRelativePath(report.ProposalPath)
	if !proposalPathOK {
		addFail("unsafe proposal path")
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}
	if proposalPath == "" {
		addFail("proposal artifact missing")
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}
	proposalAbs := filepath.Join(root, filepath.FromSlash(proposalPath))
	if _, err := os.Stat(proposalAbs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			addFail("proposal artifact missing")
		} else {
			addFail("proposal artifact could not be inspected")
		}
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}

	candidatePath, candidatePathOK := cleanWorkspaceRelativePath(report.CandidatePath)
	if report.CandidatePath != "" && !candidatePathOK {
		addFail("unsafe candidate path")
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}
	if tc.Approval.CandidateWriteApproved && candidatePath == "" {
		addReview("candidate path missing after candidate write approval")
	}
	if !tc.Approval.CandidateWriteApproved && candidatePath != "" {
		addFail("candidate write was not approved")
	}
	if !tc.Approval.OfficialMutationAuthorized && writesOfficialKnowledge(candidatePath) {
		addFail("official knowledge mutation was not approved")
	}
	if !tc.Approval.OfficialMutationAuthorized && report.HumanReview.OfficialMutationAuthorized {
		addFail("official knowledge mutation was not approved")
	}
	if !tc.Approval.PromoteAuthorized && report.HumanReview.PromoteAuthorized {
		addFail("promotion was not approved")
	}
	if !tc.Approval.PriorityMustAuthorized && report.HumanReview.PriorityMustAuthorized {
		addFail("priority escalation was not approved")
	}

	proposal, err := author.LoadProposalV2(proposalAbs)
	if err != nil {
		addFail("proposal artifact could not be loaded")
		return finalizeAuthoringEvaluation(tc, evaluation), nil
	}
	for _, finding := range author.ValidateProposalV2(proposal) {
		addAuthorFinding(finding, addFail, addReview)
	}
	if !tc.Approval.OfficialMutationAuthorized && proposal.Delivery.OfficialMutationAuthorized {
		addFail("official knowledge mutation was not approved")
	}
	if !tc.Approval.PromoteAuthorized && proposal.Delivery.PromoteAuthorized {
		addFail("promotion was not approved")
	}
	if !tc.Approval.PriorityMustAuthorized && proposal.Delivery.PriorityMustAuthorized {
		addFail("priority escalation was not approved")
	}
	if !tc.Approval.OfficialMutationAuthorized && proposal.Delivery.Path == "official_review" {
		addFail("official knowledge mutation was not approved")
	}

	if candidatePath != "" && tc.Approval.CandidateWriteApproved {
		verify, err := author.Verify(root, author.VerifyRequest{ProposalPath: proposalPath, CandidatePath: candidatePath})
		if err != nil {
			addFail("author verify error")
			return finalizeAuthoringEvaluation(tc, evaluation), nil
		}
		if report.VerifyResult != "" && report.VerifyResult != verify.Result {
			addReview("reported author verify result did not match workspace verification")
		}
		for _, finding := range verify.Findings {
			addAuthorFinding(finding, addFail, addReview)
		}
		if verify.Result == ResultFail {
			addFail("author verify failed")
		} else if verify.Result == ResultReviewNeeded {
			addReview("author verify requires human review")
		}
	} else if report.VerifyResult != "" && report.VerifyResult != reportStatusNotRun && candidatePath == "" {
		addReview("reported author verify result did not match workspace verification")
	}

	return finalizeAuthoringEvaluation(tc, evaluation), nil
}

func finalizeAuthoringEvaluation(tc Case, evaluation Evaluation) Evaluation {
	if evaluation.Result == ResultPass && tc.Oracle.ExpectedResult == ResultReviewNeeded {
		evaluation.Result = ResultReviewNeeded
		evaluation.Findings = append(evaluation.Findings, Finding{
			Severity: ResultReviewNeeded,
			Message:  "case requires human review after artifact checks",
		})
	}
	return evaluation
}

func addAuthorFinding(finding author.Finding, addFail func(string), addReview func(string)) {
	switch finding.Severity {
	case ResultFail:
		addFail("author validation failed")
	case ResultReviewNeeded:
		addReview("author validation requires human review")
	}
}

func cleanWorkspaceRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", true
	}
	if filepath.IsAbs(path) {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(clean), true
}

func writesOfficialKnowledge(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	return strings.HasPrefix(path, "knowledge/items/") || strings.HasPrefix(path, "knowledge/packages/")
}

func validAuthoringResultStatus(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded
}

func validAuthoringVerificationStatus(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded || value == reportStatusNotApplicable || value == reportStatusNotRun
}
