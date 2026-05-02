package authoringtest

import (
	"encoding/json"
	"fmt"
	"os"
)

type CaseFile struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Input          Input    `json:"input"`
	RequiredGuards []string `json:"required_guards"`
	Expected       Expected `json:"expected"`
}

type Input struct {
	Project string `json:"project"`
	Goal    string `json:"goal"`
	Mode    string `json:"mode"`
}

type Expected struct {
	Result string `json:"result"`
}

type Summary struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Input Input  `json:"input"`
}

func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read authoring cases %q: %w", path, err)
	}
	var file CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
	}
	return file.Cases, nil
}

func Summaries(cases []Case) []Summary {
	summaries := make([]Summary, 0, len(cases))
	for i, tc := range cases {
		summaries = append(summaries, Summary{
			ID:    fmt.Sprintf("case-%03d", i+1),
			Kind:  tc.Kind,
			Input: tc.Input,
		})
	}
	return summaries
}

func stringifySummaries(summaries []Summary) string {
	data, err := json.Marshal(summaries)
	if err != nil {
		return ""
	}
	return string(data)
}
