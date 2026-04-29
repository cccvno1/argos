package index

import (
	"path/filepath"
	"testing"

	"argos/internal/knowledge"
)

func TestRebuildStoresItems(t *testing.T) {
	root := t.TempDir()
	items := []knowledge.Item{testItem("backend.auth.jwt-refresh-token.v1", "JWT refresh token handling convention")}

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("backend.auth.jwt-refresh-token.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "JWT refresh token handling convention" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}

func TestCheckSchemaAcceptsRebuiltIndex(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{
		testItem("backend.auth.jwt-refresh-token.v1", "JWT refresh token handling convention"),
	}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if err := store.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema returned error: %v", err)
	}
}

func TestCheckSchemaRejectsDatabaseWithoutKnowledgeItems(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "empty.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	if err := store.CheckSchema(); err == nil {
		t.Fatal("expected CheckSchema to reject missing knowledge_items table")
	}
}

func TestFailedRebuildPreservesExistingIndex(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")

	if err := Rebuild(dbPath, []knowledge.Item{
		testItem("backend.auth.jwt-refresh-token.v1", "Original title"),
	}); err != nil {
		t.Fatalf("initial Rebuild returned error: %v", err)
	}

	err := Rebuild(dbPath, []knowledge.Item{
		testItem("backend.auth.jwt-refresh-token.v1", "Replacement title"),
		testItem("backend.auth.jwt-refresh-token.v1", "Duplicate title"),
	})
	if err == nil {
		t.Fatal("expected duplicate ID rebuild to fail")
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("backend.auth.jwt-refresh-token.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "Original title" {
		t.Fatalf("expected preserved original title, got %q", got.Title)
	}
}

func TestRebuildReplacesExistingIndex(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")

	if err := Rebuild(dbPath, []knowledge.Item{
		testItem("backend.auth.jwt-refresh-token.v1", "Original title"),
	}); err != nil {
		t.Fatalf("initial Rebuild returned error: %v", err)
	}
	if err := Rebuild(dbPath, []knowledge.Item{
		testItem("backend.cache.redis-convention.v1", "Replacement title"),
	}); err != nil {
		t.Fatalf("replacement Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("backend.cache.redis-convention.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "Replacement title" {
		t.Fatalf("expected replacement title, got %q", got.Title)
	}
	if _, err := store.GetItem("backend.auth.jwt-refresh-token.v1"); err == nil {
		t.Fatal("expected original item to be absent after rebuild")
	}
}

func testItem(id string, title string) knowledge.Item {
	return knowledge.Item{
		Path:            "knowledge/items/backend/auth.md",
		ID:              id,
		Title:           title,
		Type:            "rule",
		TechDomains:     []string{"backend", "security"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}
}
