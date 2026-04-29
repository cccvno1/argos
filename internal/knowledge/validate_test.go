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

func TestValidatePackageRequiresProtocolSections(t *testing.T) {
	reg := registry.Registry{
		TechDomains: []string{"backend"},
		Types:       []string{"rule", "package"},
	}
	item := Item{
		Path:        "knowledge/packages/backend/redis/KNOWLEDGE.md",
		ID:          "package:backend.redis.v1",
		Title:       "Redis Practices",
		Type:        "package",
		TechDomains: []string{"backend"},
		Status:      "draft",
		Priority:    "should",
		UpdatedAt:   "2026-04-29",
		Body:        "## Purpose\n\nUse Redis well.\n",
	}

	errs := ValidateItems([]Item{item}, reg)
	if len(errs) != 3 {
		t.Fatalf("expected three missing section errors, got %d: %v", len(errs), errs)
	}
	assertValidationErrorContains(t, errs, "missing package section: When To Use")
	assertValidationErrorContains(t, errs, "missing package section: Start Here")
	assertValidationErrorContains(t, errs, "missing package section: Load On Demand")
}

func TestValidatePackageAcceptsProtocolSections(t *testing.T) {
	reg := registry.Registry{
		TechDomains: []string{"backend"},
		Types:       []string{"rule", "package"},
	}
	item := Item{
		Path:        "knowledge/packages/backend/redis/KNOWLEDGE.md",
		ID:          "package:backend.redis.v1",
		Title:       "Redis Practices",
		Type:        "package",
		TechDomains: []string{"backend"},
		Status:      "draft",
		Priority:    "should",
		UpdatedAt:   "2026-04-29",
		Body: `## Purpose

Use Redis well.

## When To Use

Use this when Redis is involved.

## Start Here

Read this first.

## Load On Demand

- references/key-design.md
`,
	}

	errs := ValidateItems([]Item{item}, reg)
	if len(errs) != 0 {
		t.Fatalf("expected valid package, got %v", errs)
	}
}

func TestValidatePackageRequiresKnowledgeEntrypointPath(t *testing.T) {
	reg := registry.Registry{Types: []string{"package"}}
	item := Item{
		Path:      "knowledge/packages/backend/redis/notes.md",
		ID:        "package:backend.redis.v1",
		Title:     "Redis Practices",
		Type:      "package",
		Status:    "draft",
		Priority:  "should",
		UpdatedAt: "2026-04-29",
		Body: `## Purpose
Text
## When To Use
Text
## Start Here
Text
## Load On Demand
Text
`,
	}

	errs := ValidateItems([]Item{item}, reg)
	assertValidationErrorContains(t, errs, "package entrypoint must be KNOWLEDGE.md")
}

func assertValidationErrorContains(t *testing.T, errs []error, want string) {
	t.Helper()
	for _, err := range errs {
		if strings.Contains(err.Error(), want) {
			return
		}
	}
	t.Fatalf("expected validation error containing %q, got %v", want, errs)
}
