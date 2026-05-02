package authoringtest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	fileHandle, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read authoring cases %q: %w", path, err)
	}
	defer fileHandle.Close()

	var file CaseFile
	decoder := json.NewDecoder(fileHandle)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
		}
		return nil, fmt.Errorf("parse authoring cases %q: multiple JSON values", path)
	}
	if err := validateCaseFile(file); err != nil {
		return nil, fmt.Errorf("validate authoring cases %q: %w", path, err)
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

func validateCaseFile(file CaseFile) error {
	if len(file.Cases) == 0 {
		return fmt.Errorf("cases list is empty")
	}
	for i, tc := range file.Cases {
		context := caseContext(i, tc.ID)
		if isBlank(tc.ID) {
			return fmt.Errorf("%s: id is required", context)
		}
		if isBlank(tc.Kind) {
			return fmt.Errorf("%s: kind is required", context)
		}
		if isBlank(tc.Input.Project) {
			return fmt.Errorf("%s: input.project is required", context)
		}
		if isBlank(tc.Input.Goal) {
			return fmt.Errorf("%s: input.goal is required", context)
		}
		if isBlank(tc.Input.Mode) {
			return fmt.Errorf("%s: input.mode is required", context)
		}
		if len(tc.RequiredGuards) == 0 {
			return fmt.Errorf("%s: required_guards is required", context)
		}
		if isBlank(tc.Expected.Result) {
			return fmt.Errorf("%s: expected.result is required", context)
		}
	}
	return nil
}

func caseContext(index int, id string) string {
	if !isBlank(id) {
		return fmt.Sprintf("case[%d] id %q", index, id)
	}
	return fmt.Sprintf("case[%d]", index)
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}
