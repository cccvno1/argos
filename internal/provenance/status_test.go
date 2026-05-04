package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	recordStatusPublishApproval(t, root, id)
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

func TestStatusEvidenceReportsChangedDesignBound(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	writeStatusDesign(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", "knowledge/.inbox/packages/mall-api/redis-cache-updated")

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Evidence.DesignBound == "pass" {
		t.Fatalf("expected non-pass design evidence after design change, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "design_changed") {
		t.Fatalf("expected design_changed finding, got %#v", status.Findings)
	}
}

func TestStatusEvidenceReportsMissingRequiredHashes(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	loaded, err := Load(root, id)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	loaded.Record.Hashes.DesignSHA256 = ""
	loaded.Record.Hashes.DraftTreeSHA256 = ""
	loaded.Record.Hashes.LatestCheckSHA256 = ""
	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		t.Fatalf("resolve record path: %v", err)
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		t.Fatalf("write record: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Evidence.DesignBound == "pass" {
		t.Fatalf("expected non-pass design evidence with missing design hash, got %#v", status)
	}
	if status.Evidence.DraftBound == "pass" {
		t.Fatalf("expected non-pass draft evidence with missing draft tree hash, got %#v", status)
	}
	if status.Evidence.LatestCheck == "pass" {
		t.Fatalf("expected non-pass latest check evidence with missing check hash, got %#v", status)
	}
}

func TestStatusEvidenceReportsMissingLatestCheckArtifact(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	loaded, err := Load(root, id)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Record.LatestCheck == nil {
		t.Fatalf("expected latest check")
	}
	checkPath := filepath.Join(root, loaded.Dir, loaded.Record.LatestCheck.Path)
	if err := os.Remove(checkPath); err != nil {
		t.Fatalf("remove latest check artifact: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Evidence.LatestCheck != "missing" {
		t.Fatalf("expected missing latest check evidence, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "needs_check") && !statusHasActionContaining(status.Actions, "record-check") {
		t.Fatalf("expected needs_check finding or record-check action, got findings=%#v actions=%#v", status.Findings, status.Actions)
	}
	if status.ReadyToPublish {
		t.Fatalf("expected missing latest check artifact to block ready-to-publish, got %#v", status)
	}
}

func TestStatusEvidenceReportsChangedDecisionHashes(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	loaded, err := Load(root, id)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	loaded.Record.Hashes.DesignSHA256 = strings.Repeat("a", 64)
	loaded.Record.Hashes.DraftTreeSHA256 = strings.Repeat("b", 64)
	loaded.Record.Hashes.LatestCheckSHA256 = strings.Repeat("c", 64)
	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		t.Fatalf("resolve record path: %v", err)
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		t.Fatalf("write record: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Evidence.DesignDecision == "pass" {
		t.Fatalf("expected non-pass design decision evidence with stale approval hashes, got %#v", status)
	}
	if status.Evidence.DraftWriteDecision == "pass" {
		t.Fatalf("expected non-pass draft-write decision evidence with stale approval hashes, got %#v", status)
	}
	if status.Evidence.PublishDecision == "pass" {
		t.Fatalf("expected non-pass publish decision evidence with stale approval hashes, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "decision_mismatch") {
		t.Fatalf("expected decision_mismatch finding, got %#v", status.Findings)
	}
}

func TestStatusDoesNotDuplicateMissingPublishDecisionFinding(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if got := statusCategoryCount(status.Findings, "needs_publish_decision"); got != 1 {
		t.Fatalf("expected one needs_publish_decision finding, got %d: %#v", got, status.Findings)
	}
	if statusHasCategory(status.Findings, "decision_mismatch") {
		t.Fatalf("expected no generic decision_mismatch for missing publish decision, got %#v", status.Findings)
	}
}

func TestStatusBlocksDraftReadyWhenDraftPathMissing(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	draftPath := filepath.Join(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	officialPath := filepath.Join(root, "knowledge/packages/mall-api/redis-cache")
	if err := os.MkdirAll(filepath.Dir(officialPath), 0o755); err != nil {
		t.Fatalf("create official parent: %v", err)
	}
	if err := os.Rename(draftPath, officialPath); err != nil {
		t.Fatalf("move draft to official path: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.ReadyToPublish {
		t.Fatalf("expected missing draft to block ready-to-publish, got %#v", status)
	}
	if status.Result != "blocked" {
		t.Fatalf("expected blocked missing draft status, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "draft_missing") {
		t.Fatalf("expected draft_missing finding, got %#v", status.Findings)
	}
}

func TestStatusDoesNotPassUnsafeOfficialTarget(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	recordStatusPublishApproval(t, root, id)
	outside := filepath.Join(root, "..", "unsafe-official-target")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside target: %v", err)
	}
	loaded, err := Load(root, id)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	loaded.Record.State = StatePublished
	loaded.Record.Subject.OfficialPath = "../unsafe-official-target"
	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		t.Fatalf("resolve record path: %v", err)
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		t.Fatalf("write record: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Evidence.OfficialTarget == "pass" {
		t.Fatalf("expected unsafe official target not to pass, got %#v", status)
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

func statusCategoryCount(findings []StatusFinding, category string) int {
	count := 0
	for _, finding := range findings {
		if finding.Category == category {
			count++
		}
	}
	return count
}

func statusHasActionContaining(actions []string, want string) bool {
	for _, action := range actions {
		if strings.Contains(action, want) {
			return true
		}
	}
	return false
}

func recordStatusPublishApproval(t *testing.T, root string, id string) {
	t.Helper()
	if _, err := RecordDecision(root, id, DecisionInput{
		Stage: StagePublish, Decision: DecisionApproved, DecidedBy: "chenchi",
		Role: "knowledge_owner", Source: "conversation", Reason: "publish approved",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
}
