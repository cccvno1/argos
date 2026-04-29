package query

import (
	"path/filepath"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

func TestStandardsReturnsActiveRulesWithMatchReason(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].WhyMatched == "" {
		t.Fatalf("expected match reason")
	}
}

func TestStandardsIncludesDraftRulesAndExcludesDeprecatedRules(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/draft.md",
			ID:              "rule:backend.draft.v1",
			Title:           "Draft rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "draft",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Draft guidance still applies.",
		},
		{
			Path:            "knowledge/items/backend/deprecated.md",
			ID:              "rule:backend.deprecated.v1",
			Title:           "Deprecated rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "deprecated",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Deprecated guidance should not apply.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "rule:backend.draft.v1" {
		t.Fatalf("expected draft rule, got %q", result.Items[0].ID)
	}
}
