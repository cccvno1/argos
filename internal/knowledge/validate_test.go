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
