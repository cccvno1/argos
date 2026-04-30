package discoverytest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

type CaseFile struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID        string   `json:"id"`
	Fixture   string   `json:"fixture"`
	Operation string   `json:"operation"`
	Input     Input    `json:"input"`
	Expected  Expected `json:"expected"`
}

type Input struct {
	Project           string   `json:"project"`
	Phase             string   `json:"phase"`
	Task              string   `json:"task"`
	Query             string   `json:"query"`
	Files             []string `json:"files"`
	Types             []string `json:"types"`
	Tags              []string `json:"tags"`
	Domains           []string `json:"domains"`
	Status            []string `json:"status"`
	IncludeDeprecated bool     `json:"include_deprecated"`
	Limit             int      `json:"limit"`
}

type Expected struct {
	Coverage                         string         `json:"coverage"`
	ActionAuthority                  string         `json:"action_authority"`
	ActionLoad                       string         `json:"action_load"`
	ActionCite                       string         `json:"action_cite"`
	ActionClaim                      string         `json:"action_claim"`
	TopID                            string         `json:"top_id"`
	IncludeIDs                       []string       `json:"include_ids"`
	ExcludeIDs                       []string       `json:"exclude_ids"`
	LoadIDs                          []string       `json:"load_ids"`
	CiteIDs                          []string       `json:"cite_ids"`
	IncludeDomains                   []string       `json:"include_domains"`
	IncludeTags                      []string       `json:"include_tags"`
	IncludeText                      []string       `json:"include_text"`
	ExcludeText                      []string       `json:"exclude_text"`
	InventoryTypesMin                map[string]int `json:"inventory_types_min"`
	IncludeDeprecatedIDWhenRequested string         `json:"include_deprecated_id_when_requested"`
	RequireNextCallTools             []string       `json:"require_next_call_tools"`
	ForbidNextCallTools              []string       `json:"forbid_next_call_tools"`
	WhyContains                      []string       `json:"why_contains"`
	NoBodies                         bool           `json:"no_bodies"`
	GroupsEmpty                      bool           `json:"groups_empty"`
	ItemsEmpty                       bool           `json:"items_empty"`
	RequireMissingHints              bool           `json:"require_missing_hints"`
	RejectUnknownArguments           bool           `json:"reject_unknown_arguments"`
	RejectMissingTaskAndQuery        bool           `json:"reject_missing_task_and_query"`
	RejectOutOfRangeLimit            bool           `json:"reject_out_of_range_limit"`
}

func FixtureRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve discoverytest caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "discovery-golden"))
	if _, err := os.Stat(filepath.Join(root, "cases.json")); err != nil {
		t.Fatalf("find discovery golden fixture: %v", err)
	}
	return root
}

func LoadCases(t testing.TB) []Case {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(FixtureRoot(t), "cases.json"))
	if err != nil {
		t.Fatalf("read cases.json: %v", err)
	}
	var file CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse cases.json: %v", err)
	}
	return file.Cases
}

func CaseByID(t testing.TB, cases []Case, id string) Case {
	t.Helper()
	for _, tc := range cases {
		if tc.ID == id {
			return tc
		}
	}
	t.Fatalf("missing golden case %q", id)
	return Case{}
}

func CopyWorkspace(t testing.TB) string {
	t.Helper()
	dst := t.TempDir()
	if err := copyDir(FixtureRoot(t), dst); err != nil {
		t.Fatalf("copy discovery golden fixture: %v", err)
	}
	return dst
}

func CopyEmptyWorkspace(t testing.TB) string {
	t.Helper()
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dst, "knowledge"), 0o755); err != nil {
		t.Fatalf("create empty workspace knowledge directory: %v", err)
	}
	files := map[string]string{
		"cases.json":              `{"cases":[]}` + "\n",
		"knowledge/domains.yaml":  "tech_domains: [backend, security, database, payments, platform]\nbusiness_domains: [account, order, billing]\n",
		"knowledge/projects.yaml": "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend, security]\n    business_domains: [account]\n",
		"knowledge/types.yaml":    "types: [rule, decision, lesson, runbook, reference, package]\n",
	}
	for rel, body := range files {
		path := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create empty workspace directory for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write empty workspace file %s: %v", rel, err)
		}
	}
	return dst
}

func BuildIndexedWorkspace(t testing.TB) (string, *index.Store) {
	t.Helper()
	root := CopyWorkspace(t)
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("load golden knowledge: %v", err)
	}
	dbPath := filepath.Join(root, "argos", "index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		t.Fatalf("rebuild golden index: %v", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("open golden index: %v", err)
	}
	return root, store
}

func BuildIndexedEmptyWorkspace(t testing.TB) (string, *index.Store) {
	t.Helper()
	root := CopyEmptyWorkspace(t)
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("load empty knowledge workspace: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty knowledge workspace, got %#v", items)
	}
	dbPath := filepath.Join(root, "argos", "index.db")
	if err := index.Rebuild(dbPath, nil); err != nil {
		t.Fatalf("rebuild empty index: %v", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("open empty index: %v", err)
	}
	return root, store
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
