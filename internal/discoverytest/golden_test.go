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
	if CaseByID(t, cases, "map_inventory_empty").Fixture != "empty" {
		t.Fatalf("expected map_inventory_empty to use empty fixture")
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

func TestBuildIndexedEmptyWorkspaceCreatesEmptyStore(t *testing.T) {
	root, store := BuildIndexedEmptyWorkspace(t)
	defer store.Close()

	if _, err := os.Stat(filepath.Join(root, "knowledge", "domains.yaml")); err != nil {
		t.Fatalf("expected empty workspace registry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge", "items")); !os.IsNotExist(err) {
		t.Fatalf("expected no official items directory, got err %v", err)
	}
	items, err := store.ListItems()
	if err != nil {
		t.Fatalf("ListItems returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty indexed store, got %#v", items)
	}
}
