package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"argos/internal/knowledgewrite"
	"argos/internal/provenance"
)

func TestKnowledgeEmptyWorkspacePasses(t *testing.T) {
	root := t.TempDir()

	result, err := Knowledge(root, Request{})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "pass" {
		t.Fatalf("expected pass, got %#v", result)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no audit items, got %#v", result.Items)
	}
}

func TestKnowledgeDraftThroughCheckNeedsPublishDecisionForProject(t *testing.T) {
	root, provenanceID := createAuditProvenanceThroughCheck(t, "mall-api")

	result, err := Knowledge(root, Request{Project: "mall-api"})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "blocked" {
		t.Fatalf("expected blocked, got %#v", result)
	}
	if !auditHasItem(result.Items, "needs_publish_decision", provenanceID) {
		t.Fatalf("expected needs_publish_decision item for %s, got %#v", provenanceID, result.Items)
	}
}

func TestKnowledgeOfficialMissingPublishedProvenanceIsProblem(t *testing.T) {
	root := t.TempDir()
	writeAuditKnowledgeFile(t, root, "knowledge/items/backend/auth.md", "rule:backend.auth.v1", "mall-api", "active")

	result, err := Knowledge(root, Request{})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "problem" {
		t.Fatalf("expected problem, got %#v", result)
	}
	if !auditHasItem(result.Items, "official_missing_provenance", "") {
		t.Fatalf("expected official_missing_provenance item, got %#v", result.Items)
	}
}

func createAuditProvenanceThroughCheck(t *testing.T, project string) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeAuditRegistry(t, root, project)
	designPath := "knowledge/.inbox/designs/" + project + "/redis-cache/design.json"
	draftPath := "knowledge/.inbox/packages/" + project + "/redis-cache"
	writeAuditDesign(t, root, designPath, draftPath, project, "package:"+project+".redis-cache.v1")
	writeAuditKnowledgeFile(t, root, draftPath+"/KNOWLEDGE.md", "package:"+project+".redis-cache.v1", project, "draft")
	writeAuditFile(t, root, draftPath+"/references/redis.md", "Redis reference.\n")

	record, err := provenance.Start(root, provenance.StartRequest{
		DesignPath: designPath,
		DraftPath:  draftPath,
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	for _, stage := range []string{provenance.StageDesign, provenance.StageDraftWrite} {
		if _, err := provenance.RecordDecision(root, record.ProvenanceID, provenance.DecisionInput{
			Stage:      stage,
			Decision:   provenance.DecisionApproved,
			DecidedBy:  "chenchi",
			Role:       "knowledge_owner",
			Source:     "conversation",
			Reason:     stage + " approved.",
			RecordedBy: "codex",
		}); err != nil {
			t.Fatalf("record %s decision: %v", stage, err)
		}
	}
	if check, err := provenance.RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	} else if check.Result != "pass" {
		t.Fatalf("expected pass check, got %#v", check)
	}
	return root, record.ProvenanceID
}

func writeAuditRegistry(t *testing.T, root string, project string) {
	t.Helper()
	writeAuditFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [catalog]
`)
	writeAuditFile(t, root, "knowledge/projects.yaml", `projects:
  - id: `+project+`
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [catalog]
`)
	writeAuditFile(t, root, "knowledge/types.yaml", "types: [package]\n")
}

func writeAuditDesign(t *testing.T, root string, rel string, draftPath string, project string, knowledgeID string) {
	t.Helper()
	design := knowledgewrite.KnowledgeDesign{
		SchemaVersion: knowledgewrite.KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices.",
		KnowledgeGoal: "Document Redis cache best practices.",
		Project:       project,
		Audience: knowledgewrite.Audience{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"implement Redis cache behavior"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{project},
			Stability:    "draft",
			Distribution: "project",
			TechDomains:  []string{"backend"},
			FileGlobs:    []string{"**/*"},
		},
		Sources: knowledgewrite.Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			Claims: []knowledgewrite.SourceClaim{{
				Claim:  "Redis cache guidance is needed.",
				Kind:   "fact",
				Source: []string{"user request"},
				Trust:  "user_input",
			}},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             knowledgeID,
			Path:           draftPath,
			Status:         "draft",
			Priority:       "should",
			Rationale:      "Package entrypoint for cache knowledge.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: knowledgewrite.FutureUse{
			TriggerRequests:  []string{"implement Redis cache behavior"},
			NegativeTriggers: []string{"unrelated auth work"},
			QueryPhrases:     []string{"redis cache best practices"},
			ExpectedUse:      "Read before implementing Redis cache behavior.",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: knowledgewrite.Applicability{
			WhenToUse:    []string{"When implementing Redis cache behavior."},
			WhenNotToUse: []string{"When work is unrelated to Redis."},
		},
		ExistingKnowledge: knowledgewrite.ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing knowledge covers this.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path:                        "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:        true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path:    draftPath + "/KNOWLEDGE.md",
			Purpose: "Package entrypoint.",
			Load:    "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: draftPath,
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: project,
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{Questions: []string{"Is this ready?"}},
	}
	data, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	writeAuditFile(t, root, rel, string(data)+"\n")
}

func writeAuditKnowledgeFile(t *testing.T, root string, rel string, id string, project string, status string) {
	t.Helper()
	writeAuditFile(t, root, rel, `---
id: `+id+`
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [`+project+`]
status: `+status+`
priority: should
updated_at: 2026-05-03
applies_to:
  files: ["**/*"]
---
## Purpose

Use stable Redis cache keys, explicit TTLs, and safe fallback behavior.

## When To Use

Use when implementing Redis cache behavior.

## Start Here

Read this package before changing cache behavior.

## Load On Demand

- references/redis.md for supporting Redis details.
`)
}

func writeAuditFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func auditHasItem(items []Item, category string, provenanceID string) bool {
	for _, item := range items {
		if item.Category != category {
			continue
		}
		if provenanceID == "" || item.ProvenanceID == provenanceID {
			return true
		}
	}
	return false
}
