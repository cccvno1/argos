package dogfood

import (
	"fmt"
	"strings"

	"argos/internal/discoverytest"
)

const (
	ResultPass         = "pass"
	ResultFail         = "fail"
	ResultReviewNeeded = "review-needed"
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

func EvaluateCase(cases []discoverytest.Case, requestedID string, report Report) (Evaluation, error) {
	tc, index, err := lookupCase(cases, requestedID)
	if err != nil {
		return Evaluation{}, fmt.Errorf("evaluate dogfood report: %w", err)
	}
	publicID := publicCaseID(index)
	return evaluate(tc, report, publicID, []string{tc.ID, publicID, requestedID}), nil
}

func Evaluate(tc discoverytest.Case, report Report) Evaluation {
	return evaluate(tc, report, tc.ID, []string{tc.ID})
}

func evaluate(tc discoverytest.Case, report Report, evaluationCaseID string, allowedCaseIDs []string) Evaluation {
	evaluation := Evaluation{
		CaseID: evaluationCaseID,
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

	allowedCaseIDs = uniqueStrings(allowedCaseIDs)
	if report.CaseID != "" && !containsString(allowedCaseIDs, report.CaseID) {
		addFail(fmt.Sprintf("report case id does not match requested case: got %q, want one of %q", report.CaseID, strings.Join(allowedCaseIDs, ", ")))
	}
	for _, section := range report.MissingSections {
		addReview("missing section: " + section)
	}
	for _, field := range report.MissingFields {
		addReview("missing field: " + field)
	}
	if report.Result != "" && !isValidReportResult(report.Result) {
		addReview("unknown result status: " + report.Result)
	}
	if report.Result == ResultFail {
		addFail("runner reported result: fail")
	}
	for _, guard := range requiredReportGuards {
		if value := report.Guards[guard]; value != "" && !isValidGuardStatus(value) {
			addReview("unknown guard status for " + guard + ": " + value)
		}
	}

	expectedSupport := tc.Expected.Support
	if expectedSupport == "" {
		expectedSupport = tc.Expected.SupportLevel
	}
	if expectedSupport != "" && report.ActualSupport != "" && report.ActualSupport != expectedSupport {
		addFail("actual support does not match expected support: got " + report.ActualSupport + ", want " + expectedSupport)
	}

	idEvidenceComplete := report.hasField("discovered ids") && report.hasField("read ids") && report.hasField("cited ids")
	allReportedIDs := append(append([]string{}, report.DiscoveredIDs...), report.ReadIDs...)
	allReportedIDs = append(allReportedIDs, report.CitedIDs...)
	if idEvidenceComplete {
		for _, id := range tc.Expected.IncludeIDs {
			if !containsString(allReportedIDs, id) {
				addFail("expected ID was not reported: " + id)
			}
		}
	}
	for _, id := range tc.Expected.ExcludeIDs {
		if containsString(allReportedIDs, id) {
			addFail("excluded ID was reported: " + id)
		}
	}
	if report.hasField("read ids") {
		for _, id := range tc.Expected.LoadIDs {
			if !containsString(report.ReadIDs, id) {
				addFail("expected ID was not read: " + id)
			}
		}
	}
	if report.hasField("cited ids") {
		for _, id := range tc.Expected.CiteIDs {
			if !containsString(report.CitedIDs, id) {
				addFail("expected ID was not cited: " + id)
			}
		}
	}
	if report.hasField("read ids") && report.hasField("cited ids") {
		for _, id := range report.CitedIDs {
			if !containsString(report.ReadIDs, id) {
				addFail("cited ID was not read: " + id)
			}
		}
	}

	if tc.Expected.UsageRead == "forbidden" && len(report.ReadIDs) > 0 {
		addFail("read usage was forbidden but report includes read IDs: " + strings.Join(report.ReadIDs, ", "))
	}
	if tc.Expected.UsageCite == "forbidden" && len(report.CitedIDs) > 0 {
		addFail("citation usage was forbidden but report includes cited IDs: " + strings.Join(report.CitedIDs, ", "))
	}

	for _, guard := range requiredReportGuards {
		if report.Guards[guard] == ResultFail {
			addFail("guard failed: " + guard)
		}
	}

	return evaluation
}

func isValidReportResult(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded
}

func isValidGuardStatus(value string) bool {
	return value == ResultPass || value == ResultFail || value == ResultReviewNeeded || value == "not-applicable"
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
