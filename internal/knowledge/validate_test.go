package knowledge

import (
	"strings"
	"testing"

	"argos/internal/registry"
)

func TestValidateItemRejectsUnknownDomain(t *testing.T) {
	item := Item{
		Path:            "knowledge/items/x.md",
		ID:              "backend.auth.jwt-refresh-token.v1",
		Title:           "JWT refresh token handling convention",
		Type:            "rule",
		TechDomains:     []string{"backend", "unknown"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}
	reg := registry.Registry{
		TechDomains:     []string{"backend", "security"},
		BusinessDomains: []string{"account"},
		Projects:        []registry.Project{{ID: "mall-api"}},
		Types:           []string{"rule", "guide", "runbook", "decision", "lesson", "reference"},
	}

	errs := ValidateItem(item, reg)
	if len(errs) == 0 {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(errs[0].Error(), "knowledge/items/x.md: unknown tech domain: unknown") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestValidateItemRejectsWhitespaceRequiredField(t *testing.T) {
	item := Item{
		Path:            "knowledge/items/x.md",
		ID:              "   ",
		Title:           "JWT refresh token handling convention",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}
	reg := registry.Registry{
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []registry.Project{{ID: "mall-api"}},
		Types:           []string{"rule"},
	}

	errs := ValidateItem(item, reg)
	if len(errs) == 0 {
		t.Fatal("expected validation error")
	}
	if errs[0].Error() != "knowledge/items/x.md: missing id" {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestValidateItemsRejectsDuplicateIDs(t *testing.T) {
	items := []Item{
		{
			Path:            "knowledge/items/backend/auth.md",
			ID:              "backend.auth.jwt-refresh-token.v1",
			Title:           "JWT refresh token handling convention",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Use short-lived access tokens.",
		},
		{
			Path:            "knowledge/items/backend/auth-copy.md",
			ID:              "backend.auth.jwt-refresh-token.v1",
			Title:           "JWT refresh token handling convention copy",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Use short-lived access tokens.",
		},
	}
	reg := registry.Registry{
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []registry.Project{{ID: "mall-api"}},
		Types:           []string{"rule"},
	}

	errs := ValidateItems(items, reg)
	if len(errs) != 1 {
		t.Fatalf("expected one duplicate id error, got %d: %v", len(errs), errs)
	}
	got := errs[0].Error()
	if !strings.Contains(got, "duplicate id backend.auth.jwt-refresh-token.v1") {
		t.Fatalf("expected duplicate id in error, got %q", got)
	}
	if !strings.Contains(got, "knowledge/items/backend/auth.md") || !strings.Contains(got, "knowledge/items/backend/auth-copy.md") {
		t.Fatalf("expected both paths in error, got %q", got)
	}
}
