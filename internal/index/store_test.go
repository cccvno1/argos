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

func TestRebuildStoresDiscoveryMetadata(t *testing.T) {
	root := t.TempDir()
	item := testItem("rule:backend.auth.v1", "Refresh token auth rule")
	item.Tags = []string{"auth", "refresh-token"}
	item.Body = "Refresh token rotation must be explicit."

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{item}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "auth" || got.Tags[1] != "refresh-token" {
		t.Fatalf("expected tags to round-trip, got %#v", got.Tags)
	}

	caps, err := store.DiscoveryCapabilities()
	if err != nil {
		t.Fatalf("DiscoveryCapabilities returned error: %v", err)
	}
	if caps.Metadata != "enabled" || caps.FTS != "enabled" || caps.Semantic != "disabled" {
		t.Fatalf("unexpected capabilities: %#v", caps)
	}
	if caps.SemanticReason != "semantic provider is not configured" {
		t.Fatalf("unexpected semantic reason: %q", caps.SemanticReason)
	}
}

func TestSearchTextFindsTitleBodyAndTags(t *testing.T) {
	root := t.TempDir()
	auth := testItem("rule:backend.auth.v1", "Refresh token auth rule")
	auth.Tags = []string{"session-renewal"}
	auth.Body = "Access tokens are short lived."
	cache := testItem("rule:backend.cache.v1", "Redis cache rule")
	cache.Tags = []string{"redis"}
	cache.Body = "Cache TTLs must be explicit."

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{auth, cache}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	matches, err := store.SearchText("session-renewal refresh", 10)
	if err != nil {
		t.Fatalf("SearchText returned error: %v", err)
	}
	if len(matches) == 0 || matches[0].ItemID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule match, got %#v", matches)
	}
	if matches[0].Score <= 0 {
		t.Fatalf("expected positive lexical score, got %#v", matches[0])
	}

	matches, err = store.SearchText("short lived", 10)
	if err != nil {
		t.Fatalf("SearchText for body returned error: %v", err)
	}
	if len(matches) == 0 || matches[0].ItemID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule body match, got %#v", matches)
	}
	if matches[0].Score <= 0 {
		t.Fatalf("expected positive body lexical score, got %#v", matches[0])
	}
}

func TestRebuildIndexesPackageEntrypointChunks(t *testing.T) {
	root := t.TempDir()
	pkg := testItem("package:backend.auth-refresh.v1", "Auth refresh package")
	pkg.Type = "package"
	pkg.Path = "knowledge/packages/backend/auth-refresh/KNOWLEDGE.md"
	pkg.Body = "## Purpose\nRefresh auth flows.\n\n## When To Use\nUse for refresh token endpoints.\n\n## Start Here\nRead the rule first.\n\n## Load On Demand\nOpen examples only when needed.\n"

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{pkg}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	chunks, err := store.ListChunks("package:backend.auth-refresh.v1")
	if err != nil {
		t.Fatalf("ListChunks returned error: %v", err)
	}
	gotSections := map[string]bool{}
	for _, chunk := range chunks {
		gotSections[chunk.Section] = true
		if chunk.Text == "" {
			t.Fatalf("expected chunk text: %#v", chunk)
		}
	}
	for _, section := range []string{"Purpose", "When To Use", "Start Here", "Load On Demand"} {
		if !gotSections[section] {
			t.Fatalf("expected section %q in chunks: %#v", section, chunks)
		}
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
