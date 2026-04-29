package index

import (
	"path/filepath"
	"testing"

	"argos/internal/knowledge"
)

func TestRebuildStoresItems(t *testing.T) {
	root := t.TempDir()
	items := []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "backend.auth.jwt-refresh-token.v1",
		Title:           "JWT refresh token handling convention",
		Type:            "rule",
		TechDomains:     []string{"backend", "security"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}}

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
