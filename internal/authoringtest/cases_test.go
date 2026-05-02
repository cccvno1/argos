package authoringtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	text := marshalSummaries(t, summaries)
	summaryObjects := unmarshalSummaryObjects(t, text)
	if len(summaries) != len(cases) {
		t.Fatalf("expected %d summaries, got %d", len(cases), len(summaries))
	}
	for i, summary := range summaryObjects {
		if _, ok := summary["expected"]; ok {
			t.Fatalf("summary %d exposed expected field: %s", i, text)
		}
		if _, ok := summary["required_guards"]; ok {
			t.Fatalf("summary %d exposed required_guards field: %s", i, text)
		}
	}

	for i, summary := range summaries {
		tc := cases[i]
		wantID := fmt.Sprintf("case-%03d", i+1)
		if summary.ID != wantID {
			t.Fatalf("summary %d ID = %q, want %q", i, summary.ID, wantID)
		}
		if summary.ID == tc.ID {
			t.Fatalf("summary %d ID reused internal case ID %q", i, tc.ID)
		}
		if summary.Kind != tc.Kind {
			t.Fatalf("summary %d Kind = %q, want %q", i, summary.Kind, tc.Kind)
		}
		if summary.Input.Project != tc.Input.Project {
			t.Fatalf("summary %d Input.Project = %q, want %q", i, summary.Input.Project, tc.Input.Project)
		}
		if summary.Input.Goal != tc.Input.Goal {
			t.Fatalf("summary %d Input.Goal = %q, want %q", i, summary.Input.Goal, tc.Input.Goal)
		}
		if summary.Input.Mode != tc.Input.Mode {
			t.Fatalf("summary %d Input.Mode = %q, want %q", i, summary.Input.Mode, tc.Input.Mode)
		}
		for field, value := range exposedSummaryFields(summary) {
			assertSummaryFieldDoesNotLeak(t, i, field, value, cases)
		}
	}

	if summaries[0].ID != "case-001" {
		t.Fatalf("first summary should expose public handle case-001, got %q", summaries[0].ID)
	}
	if summaries[0].Input.Goal != cases[0].Input.Goal {
		t.Fatalf("first summary should expose input goal %q, got %q", cases[0].Input.Goal, summaries[0].Input.Goal)
	}
}

func TestLoadCasesRejectsUnknownFields(t *testing.T) {
	path := writeCasesFile(t, `{
  "cases": [
    {
      "id": "unknown_field_case",
      "kind": "user_specified",
      "input": {
        "project": "mall-api",
        "goal": "Create a backend rule.",
        "mode": "user_specified",
        "goall": "misspelled"
      },
      "required_guards": ["proposal_before_write"],
      "expected": {"result": "pass"}
    }
  ]
}`)

	_, err := LoadCases(path)
	if err == nil {
		t.Fatalf("expected LoadCases to reject unknown JSON fields")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field context, got %v", err)
	}
}

func TestLoadCasesValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty cases list",
			body: `{"cases":[]}`,
			want: []string{"cases list is empty"},
		},
		{
			name: "empty case id",
			body: validCaseJSON(`"id": ""`),
			want: []string{"case[0]", "id is required"},
		},
		{
			name: "empty kind",
			body: validCaseJSON(`"kind": ""`),
			want: []string{"case[0]", `id "valid_case"`, "kind is required"},
		},
		{
			name: "empty input project",
			body: validCaseJSON(`"input": {"project": "", "goal": "Create a backend rule.", "mode": "user_specified"}`),
			want: []string{"case[0]", `id "valid_case"`, "input.project is required"},
		},
		{
			name: "empty input goal",
			body: validCaseJSON(`"input": {"project": "mall-api", "goal": "", "mode": "user_specified"}`),
			want: []string{"case[0]", `id "valid_case"`, "input.goal is required"},
		},
		{
			name: "empty input mode",
			body: validCaseJSON(`"input": {"project": "mall-api", "goal": "Create a backend rule.", "mode": ""}`),
			want: []string{"case[0]", `id "valid_case"`, "input.mode is required"},
		},
		{
			name: "empty required guards",
			body: validCaseJSON(`"required_guards": []`),
			want: []string{"case[0]", `id "valid_case"`, "required_guards is required"},
		},
		{
			name: "empty expected result",
			body: validCaseJSON(`"expected": {"result": ""}`),
			want: []string{"case[0]", `id "valid_case"`, "expected.result is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCases(writeCasesFile(t, tt.body))
			if err == nil {
				t.Fatalf("expected validation error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected error to contain %q, got %v", want, err)
				}
			}
		})
	}
}

func marshalSummaries(t *testing.T, summaries []Summary) string {
	t.Helper()
	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	return string(data)
}

func unmarshalSummaryObjects(t *testing.T, text string) []map[string]json.RawMessage {
	t.Helper()
	var objects []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &objects); err != nil {
		t.Fatalf("unmarshal summaries: %v", err)
	}
	return objects
}

func exposedSummaryFields(summary Summary) map[string]string {
	return map[string]string{
		"id":            summary.ID,
		"kind":          summary.Kind,
		"input.project": summary.Input.Project,
		"input.goal":    summary.Input.Goal,
		"input.mode":    summary.Input.Mode,
	}
}

func assertSummaryFieldDoesNotLeak(t *testing.T, summaryIndex int, field string, value string, cases []Case) {
	t.Helper()
	for _, tc := range cases {
		if strings.Contains(value, tc.ID) {
			t.Fatalf("summary %d field %s leaked case id %q in %q", summaryIndex, field, tc.ID, value)
		}
		for _, guard := range tc.RequiredGuards {
			if value == guard {
				t.Fatalf("summary %d field %s leaked required guard %q", summaryIndex, field, guard)
			}
		}
		if value == tc.Expected.Result {
			t.Fatalf("summary %d field %s leaked expected result %q", summaryIndex, field, tc.Expected.Result)
		}
	}
}

func writeCasesFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write cases file: %v", err)
	}
	return path
}

func validCaseJSON(replacement string) string {
	fields := []string{
		`"id": "valid_case"`,
		`"kind": "user_specified"`,
		`"input": {"project": "mall-api", "goal": "Create a backend rule.", "mode": "user_specified"}`,
		`"required_guards": ["proposal_before_write"]`,
		`"expected": {"result": "pass"}`,
	}
	replacementKey := strings.SplitN(replacement, ":", 2)[0]
	for i, field := range fields {
		if strings.HasPrefix(field, replacementKey+":") {
			fields[i] = replacement
			break
		}
	}
	return `{"cases":[{` + strings.Join(fields, ",") + `}]}`
}
