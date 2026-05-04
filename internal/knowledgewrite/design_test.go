package knowledgewrite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/workspace"
)

func TestDesignKnowledgeReturnsWriteGuidanceAndTemplate(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Create Redis cache best practices for future backend agents.",
		Tags:    []string{"cache", "redis"},
		Domains: []string{"backend", "redis"},
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}

	if result.Intent == "" {
		t.Fatalf("intent missing from result: %#v", result)
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("state = %q, want ready_to_design: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if result.WriteGuidance.NextAction != "write_knowledge_design" {
		t.Fatalf("next_action = %q", result.WriteGuidance.NextAction)
	}
	if result.WriteGuidance.DraftAllowed {
		t.Fatalf("design response must not approve draft writing: %#v", result.WriteGuidance)
	}
	if result.KnowledgeDesignTemplate.SchemaVersion != KnowledgeDesignSchemaVersion {
		t.Fatalf("schema_version = %q", result.KnowledgeDesignTemplate.SchemaVersion)
	}
	if !strings.HasPrefix(result.WriteGuidance.DesignPath, "knowledge/.inbox/designs/mall-api/") {
		t.Fatalf("design path should be in inbox designs, got %q", result.WriteGuidance.DesignPath)
	}
	if !strings.HasPrefix(result.WriteGuidance.DraftPath, "knowledge/.inbox/packages/mall-api/") {
		t.Fatalf("draft path should be in inbox packages, got %q", result.WriteGuidance.DraftPath)
	}
}

func TestDesignKnowledgeJSONUsesNewVocabulary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{Project: "mall-api", Intent: "Document product list cache TTL."})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal design response: %v", err)
	}
	text := string(body)
	required := []string{
		`"write_guidance"`,
		`"knowledge_design_template"`,
		`"sources"`,
		`"draft_output"`,
		`"existing_knowledge"`,
		`"write_boundary"`,
		`"draft_files"`,
		`"check_plan"`,
		`"review"`,
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("design JSON missing %s: %s", want, text)
		}
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("design JSON contains removed term %q: %s", forbidden, text)
		}
	}
}

func TestDesignKnowledgeMissingContentIsDesignOnly(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Remember our project convention.",
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	if result.WriteGuidance.State != "design_only" {
		t.Fatalf("state = %q, want design_only: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if !result.WriteGuidance.DesignOnly {
		t.Fatalf("design_only flag should be true: %#v", result.WriteGuidance)
	}
	if result.WriteGuidance.DraftPath != "" {
		t.Fatalf("design-only response should not include draft path: %#v", result.WriteGuidance)
	}
	if len(result.KnowledgeDesignTemplate.Review.UnresolvedBlockers) == 0 {
		t.Fatalf("design-only template should include unresolved blocker: %#v", result.KnowledgeDesignTemplate.Review)
	}
}

func TestDesignKnowledgeRejectsUnsafeDraftPath(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{
		Project:   "mall-api",
		Intent:    "Create Redis guidance.",
		DraftPath: filepath.Join("..", "outside"),
	})
	if err != nil {
		t.Fatalf("unsafe draft path should produce guidance, not system error: %v", err)
	}
	if result.WriteGuidance.State != "design_only" {
		t.Fatalf("unsafe draft path should block draft writing: %#v", result.WriteGuidance)
	}
	if !containsText(result.WriteGuidance.StopConditions, "Do not write draft knowledge") {
		t.Fatalf("missing draft stop condition: %#v", result.WriteGuidance.StopConditions)
	}
}

func TestDesignKnowledgeIgnoresIndexOnlyGenericExistingText(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	writeKnowledgewriteTestFile(t, root, "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md", `---
id: package:mall-api.redis-cache.v1
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-05-04
---
## Purpose

Use stable Redis key namespaces, explicit TTLs, bounded payloads, write-adjacent invalidation, and safe fallbacks when Redis is unavailable.

## When To Use

Use when future agent work matches this scenario.
`)
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("load official knowledge: %v", err)
	}
	if err := index.Rebuild(filepath.Join(root, "argos", "index.db"), items); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "I designed a Go service template. Turn it into reusable knowledge so future agents write Go services in this style.",
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("state = %q, want ready_to_design: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if result.WriteGuidance.DraftPath == "" {
		t.Fatalf("draft path should be available for unrelated new knowledge: %#v", result.WriteGuidance)
	}
	if len(result.ExistingKnowledge.Index) != 0 {
		t.Fatalf("generic indexed text should not be reported as existing knowledge: %#v", result.ExistingKnowledge.Index)
	}
}

func TestDesignKnowledgeIgnoresSharedProjectAndBroadDomain(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	writeRedisCachePackage(t, root)

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Capture Go service template conventions for future agents.",
		Domains: []string{"backend"},
		Tags:    []string{"go", "template"},
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("state = %q, want ready_to_design: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if len(result.ExistingKnowledge.Official) != 0 {
		t.Fatalf("shared project and broad domain should not be reported as existing knowledge: %#v", result.ExistingKnowledge.Official)
	}
}

func TestDesignKnowledgeRelatedExistingKnowledgeRequiresReviewChoice(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	writeRedisCachePackage(t, root)

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Document Redis cache TTL best practices for future backend agents.",
		Domains: []string{"backend"},
		Tags:    []string{"redis", "cache"},
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	if result.WriteGuidance.State != "design_only" {
		t.Fatalf("state = %q, want design_only: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if len(result.ExistingKnowledge.Official) == 0 {
		t.Fatalf("related Redis package should be reported as existing knowledge")
	}
}

func TestDesignKnowledgeDoesNotScopeToAllRegistryDomainsByDefault(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	result, err := Design(root, DesignRequest{Project: "mall-api", Intent: "Document product list cache TTL."})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}

	scope := result.KnowledgeDesignTemplate.Scope
	if len(scope.TechDomains) != 0 {
		t.Fatalf("tech domains should be empty without explicit or project-specific domains, got %#v", scope.TechDomains)
	}
	if len(scope.SubjectDomains) != 0 {
		t.Fatalf("subject domains should be empty without explicit or project-specific domains, got %#v", scope.SubjectDomains)
	}
}

func TestDesignKnowledgeCarriesPhaseAndFilesIntoFindabilityCheck(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Document product list cache TTL.",
		Phase:   "implementation",
		Files:   []string{"internal/catalog/products.go"},
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	checks := result.KnowledgeDesignTemplate.CheckPlan.FindabilityChecks
	if len(checks) == 0 {
		t.Fatalf("missing findability checks: %#v", result.KnowledgeDesignTemplate.CheckPlan)
	}
	if checks[0].Phase != "implementation" {
		t.Fatalf("findability phase = %q", checks[0].Phase)
	}
	if len(checks[0].Files) != 1 || checks[0].Files[0] != "internal/catalog/products.go" {
		t.Fatalf("findability files = %#v", checks[0].Files)
	}
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func writeRedisCachePackage(t *testing.T, root string) {
	t.Helper()
	writeKnowledgewriteTestFile(t, root, "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md", `---
id: package:mall-api.redis-cache.v1
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-05-04
---
## Purpose

Document Redis cache best practices for backend agents.

## When To Use

Use when implementing Redis cache behavior for Mall API.
`)
}

func writeKnowledgewriteTestFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
