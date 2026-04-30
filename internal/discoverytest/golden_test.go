package discoverytest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCasesReadsGoldenCases(t *testing.T) {
	cases := LoadCases(t)
	if len(cases) != 21 {
		t.Fatalf("expected 21 golden cases, got %d", len(cases))
	}
	if CaseByID(t, cases, "strong_auth_refresh_full_signal").Expected.Coverage != "strong" {
		t.Fatalf("expected strong_auth_refresh_full_signal to expect strong coverage")
	}
}

func TestCopyWorkspaceCopiesKnowledgeAndCases(t *testing.T) {
	root := CopyWorkspace(t)
	for _, rel := range []string{
		"cases.json",
		"knowledge/domains.yaml",
		"knowledge/items/backend/auth-refresh-rule.md",
		"knowledge/packages/backend/auth-refresh/KNOWLEDGE.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected copied fixture %s: %v", rel, err)
		}
	}
}

func TestBuildIndexedWorkspaceCreatesQueryableStore(t *testing.T) {
	root, store := BuildIndexedWorkspace(t)
	defer store.Close()

	if _, err := os.Stat(filepath.Join(root, "argos", "index.db")); err != nil {
		t.Fatalf("expected index.db: %v", err)
	}
	item, err := store.GetItem("rule:backend.auth-refresh.v1")
	if err != nil {
		t.Fatalf("expected auth rule in index: %v", err)
	}
	if item.Body == "" {
		t.Fatalf("expected indexed item body")
	}
}
