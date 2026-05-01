package dogfood

import (
	"encoding/json"
	"fmt"
	"os"

	"argos/internal/discoverytest"
)

type CaseSummary struct {
	ID        string              `json:"id"`
	Fixture   string              `json:"fixture"`
	Operation string              `json:"operation"`
	Input     discoverytest.Input `json:"input"`
}

func LoadCases(path string) ([]discoverytest.Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dogfood cases %q: %w", path, err)
	}

	var file discoverytest.CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse dogfood cases %q: %w", path, err)
	}

	return file.Cases, nil
}

func Summaries(cases []discoverytest.Case) []CaseSummary {
	summaries := make([]CaseSummary, 0, len(cases))
	for i, tc := range cases {
		summaries = append(summaries, CaseSummary{
			ID:        publicCaseID(i),
			Fixture:   fixtureName(tc.Fixture),
			Operation: tc.Operation,
			Input:     tc.Input,
		})
	}
	return summaries
}

func FindCase(cases []discoverytest.Case, id string) (discoverytest.Case, bool) {
	tc, _, err := lookupCase(cases, id)
	return tc, err == nil
}

func PublicCaseID(cases []discoverytest.Case, goldenID string) (string, bool) {
	for i, tc := range cases {
		if tc.ID == goldenID {
			return publicCaseID(i), true
		}
	}
	return "", false
}

func lookupCase(cases []discoverytest.Case, id string) (discoverytest.Case, int, error) {
	var matched discoverytest.Case
	matchedIndex := 0
	matches := 0
	for i, tc := range cases {
		if tc.ID == id || publicCaseID(i) == id {
			matched = tc
			matchedIndex = i
			matches++
		}
	}
	if matches == 0 {
		return discoverytest.Case{}, 0, fmt.Errorf("unknown case %q", id)
	}
	if matches > 1 {
		return discoverytest.Case{}, 0, fmt.Errorf("ambiguous case handle %q", id)
	}
	return matched, matchedIndex, nil
}

// Public case handles are derived from case order. Keep the case list append-only
// once reports reference handles so existing runner packets remain stable.
func publicCaseID(index int) string {
	return fmt.Sprintf("case-%03d", index+1)
}

func fixtureName(fixture string) string {
	if fixture == "" {
		return "full"
	}
	return fixture
}
