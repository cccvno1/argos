package authoringdogfood

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
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Fixture  string   `json:"fixture"`
	Input    Input    `json:"input"`
	Approval Approval `json:"approval"`
	Oracle   Oracle   `json:"oracle"`
}

type Input struct {
	UserRequest      string   `json:"user_request"`
	Project          string   `json:"project"`
	ContextHints     []string `json:"context_hints"`
	AvailableSources []Source `json:"available_sources"`
	Constraints      []string `json:"constraints"`
}

type Source struct {
	Kind        string `json:"kind"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description"`
}

type Approval struct {
	ProposalApproved           bool     `json:"proposal_approved"`
	CandidateWriteApproved     bool     `json:"candidate_write_approved"`
	PriorityMustAuthorized     bool     `json:"priority_must_authorized"`
	OfficialMutationAuthorized bool     `json:"official_mutation_authorized"`
	PromoteAuthorized          bool     `json:"promote_authorized"`
	Notes                      []string `json:"notes"`
}

type Oracle struct {
	ExpectedResult             string   `json:"expected_result"`
	RequiredGuards             []string `json:"required_guards"`
	RequiredProposalProperties []string `json:"required_proposal_properties"`
	ForbiddenMutations         []string `json:"forbidden_mutations"`
	RequiredEvidenceCategories []string `json:"required_evidence_categories"`
}

type Summary struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Fixture string `json:"fixture"`
	Input   Input  `json:"input"`
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
			ID:      publicCaseID(i),
			Kind:    publicKind(tc.Kind),
			Fixture: fixtureName(tc.Fixture),
			Input:   tc.Input,
		})
	}
	return summaries
}

func FindCase(cases []Case, id string) (Case, string, error) {
	id = strings.TrimSpace(id)
	for i, tc := range cases {
		publicID := publicCaseID(i)
		if id == publicID || id == tc.ID {
			return tc, publicID, nil
		}
	}
	return Case{}, "", fmt.Errorf("unknown authoring case %q", id)
}

func publicCaseID(index int) string {
	return fmt.Sprintf("case-%03d", index+1)
}

func publicKind(kind string) string {
	switch kind {
	case "go_template_standard":
		return "template_standard"
	case "redis_best_practices":
		return "best_practices"
	case "api_consumer_knowledge":
		return "consumer_reference"
	case "observed_repo_lesson":
		return "observed_lesson"
	case "overlap_requires_choice":
		return "overlap"
	case "candidate_not_findable":
		return "findability"
	case "unauthorized_" + "author" + "ity":
		return "authz"
	case "personal_project_convention":
		return "personal_convention"
	default:
		return "authoring"
	}
}

func fixtureName(fixture string) string {
	if strings.TrimSpace(fixture) == "" {
		return "full"
	}
	return strings.TrimSpace(fixture)
}

func validateCaseFile(file CaseFile) error {
	if len(file.Cases) == 0 {
		return fmt.Errorf("cases list is empty")
	}
	for i, tc := range file.Cases {
		context := fmt.Sprintf("case[%d] id %q", i, tc.ID)
		if strings.TrimSpace(tc.ID) == "" {
			return fmt.Errorf("case[%d]: id is required", i)
		}
		if strings.TrimSpace(tc.Kind) == "" {
			return fmt.Errorf("%s: kind is required", context)
		}
		if strings.TrimSpace(tc.Input.UserRequest) == "" {
			return fmt.Errorf("%s: input.user_request is required", context)
		}
		if strings.TrimSpace(tc.Input.Project) == "" {
			return fmt.Errorf("%s: input.project is required", context)
		}
		if len(tc.Oracle.RequiredGuards) == 0 {
			return fmt.Errorf("%s: oracle.required_guards is required", context)
		}
		if strings.TrimSpace(tc.Oracle.ExpectedResult) == "" {
			return fmt.Errorf("%s: oracle.expected_result is required", context)
		}
	}
	return nil
}
