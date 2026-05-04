package provenance

import (
	"encoding/json"
	"testing"

	"argos/internal/knowledgewrite"
)

func TestStatusReportsNeedsPublishDecision(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "blocked" {
		t.Fatalf("expected blocked, got %#v", status)
	}
	if status.Evidence.PublishDecision != "missing" {
		t.Fatalf("expected missing publish decision, got %#v", status.Evidence)
	}
	if !statusHasCategory(status.Findings, "needs_publish_decision") {
		t.Fatalf("missing needs_publish_decision finding: %#v", status.Findings)
	}
}

func TestStatusReportsReadyToPublish(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	if _, err := RecordDecision(root, id, DecisionInput{
		Stage: StagePublish, Decision: DecisionApproved, DecidedBy: "chenchi",
		Role: "knowledge_owner", Source: "conversation", Reason: "publish approved",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "pass" || !status.ReadyToPublish {
		t.Fatalf("expected ready pass, got %#v", status)
	}
}

func TestStatusBlocksWhenDraftChangesAfterPublishDecision(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	if _, err := RecordDecision(root, id, DecisionInput{
		Stage: StagePublish, Decision: DecisionApproved, DecidedBy: "chenchi",
		Role: "knowledge_owner", Source: "conversation", Reason: "publish approved",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed\n")

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "blocked" {
		t.Fatalf("expected blocked after draft change, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "draft_changed") {
		t.Fatalf("expected draft_changed finding, got %#v", status.Findings)
	}
}

func createStatusWorkspaceThroughCheck(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	designPath := "knowledge/.inbox/designs/mall-api/redis-cache/design.json"
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeTestFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend]\nbusiness_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/types.yaml", "types: [package]\n")
	writeStatusDesign(t, root, designPath, draftPath)
	writeTestFile(t, root, draftPath+"/KNOWLEDGE.md", validProvenanceDraftPackage())
	writeTestFile(t, root, draftPath+"/references/redis.md", "Redis reference.\n")

	record, err := Start(root, StartRequest{DesignPath: designPath, DraftPath: draftPath, CreatedBy: "codex"})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	for _, stage := range []string{StageDesign, StageDraftWrite} {
		if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
			Stage: stage, Decision: DecisionApproved, DecidedBy: "chenchi",
			Role: "knowledge_owner", Source: "conversation", Reason: stage + " approved",
			RecordedBy: "codex",
		}); err != nil {
			t.Fatalf("record %s decision: %v", stage, err)
		}
	}
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	return root, record.ProvenanceID
}

func writeStatusDesign(t *testing.T, root string, rel string, draftPath string) {
	t.Helper()
	design := knowledgewrite.KnowledgeDesign{
		SchemaVersion: knowledgewrite.KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices.",
		KnowledgeGoal: "Document Redis cache best practices.",
		Project:       "mall-api",
		Audience: knowledgewrite.Audience{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"implement Redis cache behavior"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			TechDomains:  []string{"backend"},
			FileGlobs:    []string{"**/*"},
		},
		Sources: knowledgewrite.Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			Claims: []knowledgewrite.SourceClaim{{
				Claim: "Redis cache guidance is needed.", Kind: "fact",
				Source: []string{"user request"}, Trust: "user_input",
			}},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind: "package", Type: "package", Title: "Redis Cache Best Practices",
			ID: "package:mall-api.redis-cache.v1", Path: draftPath, Status: "draft",
			Priority: "should", Rationale: "Package entrypoint for cache knowledge.",
			EntrypointLoad: "read_before_implementation", DraftState: "draft",
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
			Decision: "create_new", Reason: "No existing knowledge covers this.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path: "inbox", WriteRequiresReviewApproval: true, ReviewPacketRequired: true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path: draftPath + "/KNOWLEDGE.md", Purpose: "Package entrypoint.",
			Load: "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: draftPath,
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: "mall-api", Phase: "implementation",
				Task: "implement Redis cache behavior", Query: "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{Questions: []string{"Is this ready?"}},
	}
	data, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	writeTestFile(t, root, rel, string(data)+"\n")
}

func statusHasCategory(findings []StatusFinding, category string) bool {
	for _, finding := range findings {
		if finding.Category == category {
			return true
		}
	}
	return false
}
