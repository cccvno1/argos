package cli

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"argos/internal/discoverytest"
	"argos/internal/query"
)

func TestGoldenCLIKnowledgeFindMatchesQueryBehavior(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_find_matches_query")

	args := []string{
		"knowledge", "find", "--json",
		"--project", tc.Input.Project,
		"--phase", tc.Input.Phase,
		"--task", tc.Input.Task,
		"--query", tc.Input.Query,
		"--limit", strconv.Itoa(tc.Input.Limit),
	}
	for _, file := range tc.Input.Files {
		args = append(args, "--files", file)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.FindKnowledgeResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse knowledge find JSON: %v\n%s", err, stdout.String())
	}
	if result.Support.Level != tc.Expected.SupportLevel {
		t.Fatalf("expected support level %q, got %#v", tc.Expected.SupportLevel, result.Support)
	}
	if result.Usage.Read != tc.Expected.UsageRead {
		t.Fatalf("expected read guidance %q, got %#v", tc.Expected.UsageRead, result.Usage)
	}
	if result.SearchStatus.Semantic.Status != tc.Expected.SearchSemanticStatus {
		t.Fatalf("expected semantic status %q, got %#v", tc.Expected.SearchSemanticStatus, result.SearchStatus)
	}
	if !containsGoldenKnowledgeID(result.Items, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Items)
	}
	assertCLINextSteps(t, result.NextSteps, tc.Expected.RequireNextStepTools, tc.Expected.ForbidNextStepTools)
	for _, item := range result.Items {
		if tc.Expected.NoBodies && item.Body != "" {
			t.Fatalf("CLI knowledge find returned body for %s", item.ID)
		}
	}
}

func TestGoldenCLIKnowledgeFindNoneIncludesMissingNeeds(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "none_payment_webhook")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"knowledge", "find", "--json",
		"--project", tc.Input.Project,
		"--phase", tc.Input.Phase,
		"--task", tc.Input.Task,
		"--query", tc.Input.Query,
		"--limit", strconv.Itoa(tc.Input.Limit),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.FindKnowledgeResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse knowledge find JSON: %v\n%s", err, stdout.String())
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none support, got %#v", result.Support)
	}
	if result.Usage.Read != tc.Expected.UsageRead {
		t.Fatalf("expected read guidance %q, got %#v", tc.Expected.UsageRead, result.Usage)
	}
	if result.SearchStatus.Semantic.Status != tc.Expected.SearchSemanticStatus {
		t.Fatalf("expected semantic status %q, got %#v", tc.Expected.SearchSemanticStatus, result.SearchStatus)
	}
	assertCLIMissingNeedSources(t, result.MissingNeeds, tc.Expected.MissingNeedSources)
	for _, need := range result.MissingNeeds {
		if need.ArgosBacked {
			t.Fatalf("expected CLI missing need not to be Argos-backed, got %#v", need)
		}
	}
	assertCLINextSteps(t, result.NextSteps, tc.Expected.RequireNextStepTools, tc.Expected.ForbidNextStepTools)
	legacyKey := `"gap_` + `candidates"`
	if strings.Contains(stdout.String(), legacyKey) {
		t.Fatalf("CLI knowledge find should not return legacy gap candidates: %s", stdout.String())
	}
}

func TestGoldenCLIKnowledgeListMatchesQueryBehavior(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_list_matches_query")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "list", "--json", "--project", tc.Input.Project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.ListKnowledgeResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse knowledge list JSON: %v\n%s", err, stdout.String())
	}
	if result.Usage.Read != tc.Expected.UsageRead {
		t.Fatalf("expected read guidance %q, got %#v", tc.Expected.UsageRead, result.Usage)
	}
	for typ, min := range tc.Expected.InventoryTypesMin {
		if result.Inventory.Types[typ] < min {
			t.Fatalf("expected type %s >= %d, got %#v", typ, min, result.Inventory.Types)
		}
	}
	if !containsGoldenListID(result.Groups, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Groups)
	}
}

func TestGoldenCLIKnowledgeReadReturnsFullBody(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_read_returns_body")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "read", "--json", tc.Expected.LoadIDs[0]}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.KnowledgeItemResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse knowledge read JSON: %v\n%s", err, stdout.String())
	}
	if result.ID != tc.Expected.LoadIDs[0] || result.Body == "" {
		t.Fatalf("expected full knowledge item body, got %#v", result)
	}
}

func TestGoldenCLIKnowledgeCiteReturnsCitation(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_cite_returns_citation")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "cite", "--json", tc.Expected.CiteIDs[0]}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.CitationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse knowledge cite JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Citations) != 1 || result.Citations[0].ID != tc.Expected.CiteIDs[0] {
		t.Fatalf("expected citation for auth refresh rule, got %#v", result)
	}
	if len(result.Missing) != 0 {
		t.Fatalf("expected no missing citations, got %#v", result.Missing)
	}
}

func TestGoldenCLIValidationErrorsStayExplicit(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "find missing task and query", args: []string{"knowledge", "find", "--json", "--project", "mall-api"}, want: "knowledge find: --task or --query is required"},
		{name: "find bad limit", args: []string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "auth", "--limit", "99"}, want: "knowledge find: --limit must be between 1 and 20"},
		{name: "list missing project", args: []string{"knowledge", "list", "--json"}, want: "knowledge list: --project is required"},
		{name: "read missing id", args: []string{"knowledge", "read", "--json"}, want: "knowledge read: id is required"},
		{name: "cite missing ids", args: []string{"knowledge", "cite", "--json"}, want: "knowledge cite: at least one id is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("expected nonzero exit code")
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.want, stderr.String())
			}
		})
	}
}

func containsGoldenKnowledgeID(items []query.KnowledgeSummary, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func assertCLIMissingNeedSources(t *testing.T, got []query.MissingNeed, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected missing need sources %v, got %#v", want, got)
	}
	seen := map[string]bool{}
	for _, need := range got {
		seen[need.Source] = true
	}
	for _, source := range want {
		if !seen[source] {
			t.Fatalf("expected missing need source %q in %#v", source, got)
		}
	}
}

func assertCLINextSteps(t *testing.T, got []query.NextStep, requireTools []string, forbidTools []string) {
	t.Helper()
	tools := map[string]bool{}
	for _, step := range got {
		tools[step.Tool] = true
	}
	for _, tool := range requireTools {
		if !tools[tool] {
			t.Fatalf("expected next step tool %q in %#v", tool, got)
		}
	}
	for _, tool := range forbidTools {
		if tools[tool] {
			t.Fatalf("did not expect next step tool %q in %#v", tool, got)
		}
	}
}

func containsGoldenListID(groups []query.ListGroup, id string) bool {
	for _, group := range groups {
		for _, item := range group.Items {
			if item.ID == id {
				return true
			}
		}
	}
	return false
}
