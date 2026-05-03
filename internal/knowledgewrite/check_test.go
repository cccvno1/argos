package knowledgewrite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/workspace"
)

func TestCheckKnowledgeReturnsNewVocabulary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	designPath := writeDesignFile(t, root, validKnowledgeDesign())
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackage(t, root, draftPath)

	result, err := Check(root, CheckRequest{DesignPath: designPath, DraftPath: draftPath})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Result != "pass" && result.Result != "review-needed" {
		t.Fatalf("unexpected check result: %#v", result)
	}
	if result.Design.Path != designPath {
		t.Fatalf("design path mismatch: %#v", result.Design)
	}
	if result.Draft.Path != draftPath {
		t.Fatalf("draft path mismatch: %#v", result.Draft)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal check result: %v", err)
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("check JSON contains removed term %q: %s", forbidden, string(body))
		}
	}
}

func TestCheckKnowledgeRejectsDraftOutsideWriteBoundary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	design := validKnowledgeDesign()
	design.DraftOutput.Path = "knowledge/.inbox/packages/mall-api/redis-cache"
	design.CheckPlan.ValidatePath = "knowledge/.inbox/packages/mall-api/redis-cache"
	designPath := writeDesignFile(t, root, design)
	draftPath := "knowledge/packages/mall-api/redis-cache"
	writeDraftPackage(t, root, draftPath)

	result, err := Check(root, CheckRequest{DesignPath: designPath, DraftPath: draftPath})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected fail for official draft outside inbox boundary, got %#v", result)
	}
	if !hasFinding(result.Findings, "draft path is outside approved write boundary") {
		t.Fatalf("missing boundary finding: %#v", result.Findings)
	}
}

func TestCheckKnowledgeOfficialDraftWithoutPublishApprovalNeedsReview(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	design := validKnowledgeDesign()
	design.WriteBoundary.Path = "official_review"
	design.Review.OfficialWriteApproved = true
	design.Review.PublishApproved = false
	design.DraftOutput.Path = "knowledge/packages/mall-api/redis-cache"
	design.CheckPlan.ValidatePath = "knowledge/packages/mall-api/redis-cache"
	design.DraftFiles = []DraftFile{{
		Path:    "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md",
		Purpose: "entrypoint",
		Load:    "read_before_implementation",
	}}
	designPath := writeDesignFile(t, root, design)
	draftPath := "knowledge/packages/mall-api/redis-cache"
	writeDraftPackageWithStatus(t, root, draftPath, "active")

	result, err := Check(root, CheckRequest{DesignPath: designPath, DraftPath: draftPath})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Result != "review-needed" {
		t.Fatalf("expected review-needed for official draft without publish approval, got %#v", result)
	}
	if !hasFinding(result.Findings, "review.publish_approved") {
		t.Fatalf("missing publish approval finding: %#v", result.Findings)
	}
}

func TestValidateDesignUsesReviewApprovalForPriorityMust(t *testing.T) {
	design := validKnowledgeDesign()
	design.DraftOutput.Priority = "must"
	design.Review.PriorityMustApproved = true
	design.WriteBoundary.PriorityMustApproved = false

	findings := ValidateDesign(design)

	if hasFindingWithSeverity(findings, "fail", "priority: must") {
		t.Fatalf("priority must should be approved by review.priority_must_approved: %#v", findings)
	}
}

func TestValidateDesignUsesReviewApprovalForOfficialReview(t *testing.T) {
	design := validKnowledgeDesign()
	design.WriteBoundary.Path = "official_review"
	design.WriteBoundary.OfficialWriteApproved = false
	design.Review.OfficialWriteApproved = true
	design.DraftOutput.Path = "knowledge/packages/mall-api/redis-cache"
	design.CheckPlan.ValidatePath = "knowledge/packages/mall-api/redis-cache"
	design.DraftFiles = []DraftFile{{
		Path:    "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md",
		Purpose: "entrypoint",
		Load:    "read_before_implementation",
	}}

	findings := ValidateDesign(design)

	if hasFindingWithSeverity(findings, "fail", "official writing requires explicit approval") {
		t.Fatalf("official review should be approved by review.official_write_approved: %#v", findings)
	}
}

func validKnowledgeDesign() KnowledgeDesign {
	return KnowledgeDesign{
		SchemaVersion: KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices for future backend agents.",
		KnowledgeGoal: "Design reusable knowledge for future agents: Create Redis cache best practices.",
		Project:       "mall-api",
		Audience: Audience{
			Primary: "implementer_agent",
			AgentActionsSupported: []string{
				"apply Redis cache guidance during implementation",
			},
		},
		Scope: Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			FileGlobs:    []string{"**/*"},
		},
		Sources: Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			AISuggested: []string{
				"Use reviewed Redis recommendations as draft guidance.",
			},
			Assumptions: []string{"Team will review before publishing."},
			Claims: []SourceClaim{{
				Claim:          "Redis cache best practices are being designed for future agents.",
				Kind:           "recommendation",
				Trust:          "ai_suggested",
				Source:         []string{"argos knowledge design"},
				RequiresReview: true,
			}},
		},
		DraftOutput: DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             "package:mall-api.redis-cache.v1",
			Path:           "knowledge/.inbox/packages/mall-api/redis-cache",
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package gives future agents one entrypoint for Redis cache guidance.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: FutureUse{
			TriggerRequests: []string{"implement Redis cache behavior"},
			QueryPhrases:    []string{"redis cache best practices"},
			ExpectedUse:     "Future agents should read this before implementing Redis cache behavior.",
			CitationPolicy:  "cite_after_use",
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When implementing Redis-backed caching."},
			WhenNotToUse: []string{"When changing unrelated storage behavior."},
		},
		ExistingKnowledge: ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing knowledge covers this exact guidance.",
		},
		WriteBoundary: WriteBoundary{
			Path:                        "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:        true,
		},
		DraftFiles: []DraftFile{{
			Path:    "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
			Purpose: "entrypoint",
			Load:    "read_before_implementation",
		}},
		CheckPlan: CheckPlan{
			ValidatePath: "knowledge/.inbox/packages/mall-api/redis-cache",
			FindabilityChecks: []FindabilityCheckScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: Review{
			Questions:          []string{"Is draft writing approved?"},
			DesignApproved:     true,
			DraftWriteApproved: true,
		},
	}
}

func writeDesignFile(t *testing.T, root string, design KnowledgeDesign) string {
	t.Helper()
	rel := "knowledge/.inbox/designs/mall-api/redis-cache/design.json"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir design parent: %v", err)
	}
	body, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	if err := os.WriteFile(abs, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write design: %v", err)
	}
	return rel
}

func writeDraftPackage(t *testing.T, root string, rel string) {
	t.Helper()
	writeDraftPackageWithStatus(t, root, rel, "draft")
}

func writeDraftPackageWithStatus(t *testing.T, root string, rel string, status string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir draft: %v", err)
	}
	body := `---
id: package:mall-api.redis-cache.v1
type: package
title: Redis Cache Best Practices
status: ` + status + `
priority: should
projects: [mall-api]
tech_domains: [redis]
tags: [cache]
updated_at: 2026-05-03
applies_to:
  files: ["**/*"]
---

# Redis Cache Best Practices

## Purpose

Use reviewed Redis cache guidance for future backend implementation.

## When To Use

When implementing Redis-backed caching.

## Start Here

Read this package before changing cache behavior.

## Load On Demand

Load deeper examples only when the implementation needs them.
`
	if err := os.WriteFile(filepath.Join(dir, "KNOWLEDGE.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write draft package: %v", err)
	}
}

func hasFinding(findings []Finding, text string) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}

func hasFindingWithSeverity(findings []Finding, severity string, text string) bool {
	for _, finding := range findings {
		if finding.Severity == severity && strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}
