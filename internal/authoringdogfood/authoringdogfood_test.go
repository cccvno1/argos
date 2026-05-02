package authoringdogfood

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const authoringCasesPath = "../../testdata/authoring-golden/cases.json"

func TestLoadCasesIncludesRealScenarioMatrix(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"go_template_standard":             false,
		"redis_best_practices":             false,
		"api_consumer_knowledge":           false,
		"observed_repo_lesson":             false,
		"overlap_requires_choice":          false,
		"candidate_not_findable":           false,
		"unauthorized_" + "author" + "ity": false,
		"personal_project_convention":      false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing authoring case kind %s", kind)
		}
	}
}

func TestSummariesExposeNaturalInputAndHideOracle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	text := string(data)
	for _, want := range []string{"case-001", "user_request", "I designed a Go service template"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q: %s", want, text)
		}
	}
	for _, forbidden := range []string{
		"expected",
		"oracle",
		"expected_result",
		"required_guards",
		"required_proposal_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"candidate_write_approved",
		"priority_must_authorized",
		"official_mutation_authorized",
		"promote_authorized",
		"review-needed",
		"proposal_must_precede_candidate",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}

	summaryValues := collectSummaryStrings(t, summaries)
	for _, tc := range cases {
		assertSummaryTextOmits(t, text, "internal case ID", tc.ID)
		if tc.Oracle.ExpectedResult != "pass" {
			assertSummaryTextOmits(t, text, "expected result", tc.Oracle.ExpectedResult)
		}
		for _, value := range appendHiddenValues(tc.Oracle.RequiredGuards, tc.Oracle.RequiredProposalProperties, tc.Oracle.ForbiddenMutations, tc.Oracle.RequiredEvidenceCategories) {
			if hiddenStructuredToken(value) {
				assertSummaryValuesOmit(t, summaryValues, "hidden oracle value", value)
			}
		}
	}
}

func TestFindCaseAcceptsPublicHandle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, publicID, err := FindCase(cases, "case-001")
	if err != nil {
		t.Fatalf("FindCase returned error: %v", err)
	}
	if publicID != "case-001" {
		t.Fatalf("publicID = %q, want case-001", publicID)
	}
	if !strings.Contains(tc.Input.UserRequest, "Go service template") {
		t.Fatalf("unexpected case input: %#v", tc.Input)
	}
}

func collectSummaryStrings(t *testing.T, summaries []Summary) []string {
	t.Helper()

	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal summaries: %v", err)
	}

	var values []string
	collectStrings(reflect.ValueOf(decoded), &values)
	return values
}

func collectStrings(value reflect.Value, values *[]string) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface {
		collectStrings(value.Elem(), values)
		return
	}
	switch value.Kind() {
	case reflect.String:
		*values = append(*values, value.String())
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			collectStrings(value.Index(i), values)
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			collectStrings(value.MapIndex(key), values)
		}
	}
}

func appendHiddenValues(groups ...[]string) []string {
	var values []string
	for _, group := range groups {
		values = append(values, group...)
	}
	return values
}

func assertSummaryTextOmits(t *testing.T, text, label, forbidden string) {
	t.Helper()
	if strings.Contains(text, forbidden) {
		t.Fatalf("summary leaked %s %q: %s", label, forbidden, text)
	}
}

func assertSummaryValuesOmit(t *testing.T, values []string, label, forbidden string) {
	t.Helper()
	for _, value := range values {
		if value == forbidden {
			t.Fatalf("summary leaked %s %q as JSON string value", label, forbidden)
		}
	}
}

func hiddenStructuredToken(value string) bool {
	return strings.Contains(value, "_") || strings.Contains(value, ".") || value == "review-needed" || value == "promotion"
}
