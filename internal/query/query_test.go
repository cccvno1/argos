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

func TestStandardsRanksResultsByPriorityBeforeApplyingLimit(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		ruleWithPriority("rule:priority.may.v1", "may"),
		ruleWithPriority("rule:priority.must.v1", "must"),
		ruleWithPriority("rule:priority.should.v1", "should"),
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
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	got := []string{result.Items[0].ID, result.Items[1].ID}
	want := []string{"rule:priority.must.v1", "rule:priority.should.v1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected items %v, got %v", want, got)
		}
	}
}

func TestStandardsPrefersFileScopedMatchBeforeApplyingLimit(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/generic.md",
			ID:              "rule:priority.generic.v1",
			Title:           "Generic must rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Generic guidance applies.",
		},
		{
			Path:            "knowledge/items/backend/specific.md",
			ID:              "rule:priority.specific.v1",
			Title:           "Specific must rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
			UpdatedAt:       "2026-04-29",
			Body:            "File-specific guidance applies.",
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
		Files:   []string{"internal/auth/session.go"},
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "rule:priority.specific.v1" {
		t.Fatalf("expected file-scoped rule, got %q", result.Items[0].ID)
	}
}

func TestContextRecommendsNextCalls(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "add refresh token endpoint",
		Files:   []string{"internal/auth/session.go"},
	})

	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", result.Project)
	}
	if len(result.RecommendedNextCalls) == 0 {
		t.Fatal("expected recommended next calls")
	}
}

func TestGetKnowledgeItemReturnsFullBody(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	item, err := service.GetKnowledgeItem("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("GetKnowledgeItem returned error: %v", err)
	}
	if item.Body != "Require explicit auth middleware for account endpoints.\nThis is the full rule body." {
		t.Fatalf("expected full body, got %q", item.Body)
	}
}

func TestCiteKnowledgeReportsMissingIDs(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	result := service.CiteKnowledge([]string{"rule:backend.auth.v1", "missing.v1"})
	if len(result.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(result.Citations))
	}
	if result.Citations[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule citation, got %#v", result.Citations[0])
	}
	if len(result.Missing) != 1 || result.Missing[0] != "missing.v1" {
		t.Fatalf("expected missing id, got %#v", result.Missing)
	}
}

func ruleWithPriority(id string, priority string) knowledge.Item {
	return knowledge.Item{
		Path:            "knowledge/items/backend/priority.md",
		ID:              id,
		Title:           priority + " priority rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        priority,
		UpdatedAt:       "2026-04-29",
		Body:            "Priority guidance applies.",
	}
}

func buildQueryTestStore(t *testing.T) *index.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth middleware",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware for account endpoints.\nThis is the full rule body.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
