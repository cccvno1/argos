package authoringtest

import (
	"strings"
	"testing"
)

func TestLoadCasesIncludesRequiredAuthoringKinds(t *testing.T) {
	cases, err := LoadCases("../../testdata/authoring-golden/cases.json")
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"user_specified": false,
		"observed":       false,
		"synthesized":    false,
		"overlap":        false,
		"findability":    false,
		"policy":         false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing authoring case kind %s in %#v", kind, cases)
		}
	}
}

func TestSummariesHideExpectedOracles(t *testing.T) {
	cases, err := LoadCases("../../testdata/authoring-golden/cases.json")
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	text := stringifySummaries(summaries)
	for _, forbidden := range []string{
		"expected",
		"result",
		"purpose_built_rule_user_specified",
		"candidate_validates_but_not_findable",
		"priority_must_requires_authorization",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "case-001") || !strings.Contains(text, "Create a backend rule") {
		t.Fatalf("summary should expose public handles and inputs: %s", text)
	}
}
