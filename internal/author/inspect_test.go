package author

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

func TestInspectReportsRegistryPolicyAndNoIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		Mode:       ModeSynthesized,
		FutureTask: "implement product list cache",
		Phase:      "implementation",
		Files:      []string{"internal/catalog/products.go"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Project != "mall-api" || result.Goal != "create product-list cache knowledge" || result.Mode != ModeSynthesized {
		t.Fatalf("unexpected inspect identity: %#v", result)
	}
	if result.Capabilities.Filesystem != "enabled" || result.Capabilities.Index != "unavailable" {
		t.Fatalf("unexpected capabilities: %#v", result.Capabilities)
	}
	if !result.Registry.ProjectKnown {
		t.Fatalf("expected known project: %#v", result.Registry)
	}
	if !reflect.DeepEqual(result.Registry.TechDomains, []string{"backend", "database"}) {
		t.Fatalf("unexpected tech domains: %#v", result.Registry.TechDomains)
	}
	if !reflect.DeepEqual(result.Registry.BusinessDomains, []string{"catalog"}) {
		t.Fatalf("unexpected business domains: %#v", result.Registry.BusinessDomains)
	}
	if result.Policy.Write != "after_proposal_approval" ||
		result.Policy.OfficialMutation != "requires_explicit_review_path" ||
		result.Policy.Promote != "requires_explicit_approval" ||
		result.Policy.PriorityMust != "requires_explicit_authorization" ||
		result.Policy.SynthesizedClaims != "must_mark_assumptions" {
		t.Fatalf("unexpected policy: %#v", result.Policy)
	}
	wantRequirements := []string{
		"schema_version:authoring.proposal.v2",
		"user_request",
		"future_agent_audience",
		"source_profile",
		"future_use",
		"applicability",
		"overlap_decision",
		"delivery",
		"candidate_files",
		"verification_plan",
		"human_review",
	}
	if !reflect.DeepEqual(result.ProposalRequirements, wantRequirements) {
		t.Fatalf("unexpected proposal requirements: %#v", result.ProposalRequirements)
	}
	if len(result.RecommendedNextSteps) != 1 ||
		result.RecommendedNextSteps[0].Step != "write_knowledge_design_proposal" ||
		result.RecommendedNextSteps[0].Reason != "Human review is required before durable writes." {
		t.Fatalf("unexpected next steps: %#v", result.RecommendedNextSteps)
	}
}

func TestInspectFindsOfficialAndInboxOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "draft", "Product List Cache Package"))

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		FutureTask: "product list cache ttl",
		Query:      "product list cache",
		Tags:       []string{"cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(result.Overlap.Official) == 0 {
		t.Fatalf("expected official overlap: %#v", result.Overlap)
	}
	if len(result.Overlap.Inbox) == 0 {
		t.Fatalf("expected inbox overlap: %#v", result.Overlap)
	}
	if result.Overlap.Official[0].Kind != "official" || result.Overlap.Official[0].ID != "rule:backend.cache.v1" {
		t.Fatalf("unexpected official overlap: %#v", result.Overlap.Official)
	}
	if result.Overlap.Inbox[0].Kind != "inbox" || result.Overlap.Inbox[0].ID != "package:backend.product-list-cache.v1" {
		t.Fatalf("unexpected inbox overlap: %#v", result.Overlap.Inbox)
	}
	if len(result.Overlap.Official[0].Reasons) == 0 || len(result.Overlap.Inbox[0].Reasons) == 0 {
		t.Fatalf("expected overlap reasons: %#v", result.Overlap)
	}
}

func TestInspectFindsIndexOverlapReadOnly(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("LoadOfficial returned error: %v", err)
	}
	if err := index.Rebuild(filepath.Join(root, "argos", "index.db"), items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		FutureTask: "product list cache ttl",
		Query:      "product list cache",
		Tags:       []string{"cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Capabilities.Index != "enabled" {
		t.Fatalf("expected enabled index capability, got %#v", result.Capabilities)
	}
	if !hasOverlap(result.Overlap.Index, "index", "rule:backend.cache.v1") {
		t.Fatalf("expected indexed product-list cache overlap, got %#v", result.Overlap.Index)
	}
}

func TestInspectReportsCandidatePathRisk(t *testing.T) {
	tests := []struct {
		name          string
		candidatePath string
		wantStatus    string
	}{
		{
			name:          "unsafe escaping path",
			candidatePath: "../outside",
			wantStatus:    "unsafe",
		},
		{
			name:          "standard inbox item path",
			candidatePath: "knowledge/.inbox/items/backend/cache.md",
			wantStatus:    "allowed",
		},
		{
			name:          "standard inbox package path",
			candidatePath: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md",
			wantStatus:    "allowed",
		},
		{
			name:          "official item path",
			candidatePath: "knowledge/items/backend/cache.md",
			wantStatus:    "official_review_required",
		},
		{
			name:          "official package path",
			candidatePath: "knowledge/packages/backend/product-list-cache/KNOWLEDGE.md",
			wantStatus:    "official_review_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeAuthorRegistry(t, root)

			result, err := Inspect(root, InspectRequest{
				Project:       "mall-api",
				Goal:          "create product-list cache knowledge",
				CandidatePath: tt.candidatePath,
			})
			if err != nil {
				t.Fatalf("Inspect returned error: %v", err)
			}
			if result.PathRisk.Status != tt.wantStatus {
				t.Fatalf("expected path risk status %q, got %#v", tt.wantStatus, result.PathRisk)
			}
		})
	}
}

func writeAuthorRegistry(t *testing.T, root string) {
	t.Helper()
	writeAuthorFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend, database]\nbusiness_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/types.yaml", "types: [rule, decision, lesson, runbook, reference, package]\n")
}

func writeAuthorFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func authorItem(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: rule
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache]
applies_to:
  files: [internal/catalog/**]
---
Product list cache TTL guidance for backend work.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}

func authorPackage(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache, redis]
applies_to:
  files: [internal/catalog/**]
---
## Purpose

Product list cache package.

## When To Use

Use when implementing product list cache behavior.

## Start Here

Read this entrypoint before implementation.

## Load On Demand

Load supporting references only when needed.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}

func hasOverlap(matches []OverlapMatch, kind string, id string) bool {
	for _, match := range matches {
		if match.Kind == kind && match.ID == id {
			return true
		}
	}
	return false
}
