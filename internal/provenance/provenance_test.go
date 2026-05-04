package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHashFileChangesWhenFileChanges(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/redis/design.json", `{"schema_version":"knowledge.design.v1"}`)

	first, err := HashFile(root, "knowledge/.inbox/designs/redis/design.json")
	if err != nil {
		t.Fatalf("HashFile returned error: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/designs/redis/design.json", `{"schema_version":"knowledge.design.v1","changed":true}`)
	second, err := HashFile(root, "knowledge/.inbox/designs/redis/design.json")
	if err != nil {
		t.Fatalf("HashFile returned error after change: %v", err)
	}

	if first == second {
		t.Fatalf("expected hash to change, got %q", first)
	}
	if len(first) != 64 || len(second) != 64 {
		t.Fatalf("expected hex sha256 hashes, got %q and %q", first, second)
	}
}

func TestHashTreeIncludesSupportingFilesInStableOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", "entrypoint\n")
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "reference\n")

	first, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	if err != nil {
		t.Fatalf("HashTree returned error: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed\n")
	second, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	if err != nil {
		t.Fatalf("HashTree returned error after change: %v", err)
	}

	if first == second {
		t.Fatalf("expected tree hash to change when supporting file changes")
	}
}

func TestHashPublishedDraftAsDraftDoesNotNormalizeSupportingMarkdown(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", validProvenanceDraftPackage())
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", `---
status: draft
---
Reference evidence.
`)
	draftHash, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	if err != nil {
		t.Fatalf("HashTree returned error: %v", err)
	}

	writeTestFile(t, root, "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md", strings.Replace(validProvenanceDraftPackage(), "status: draft", "status: active", 1))
	writeTestFile(t, root, "knowledge/packages/mall-api/redis-cache/references/redis.md", `---
status: active
---
Reference evidence.
`)
	record := Record{
		Subject: Subject{
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
	}

	publishedHash, err := hashPublishedDraftAsDraft(root, record)
	if err != nil {
		t.Fatalf("hashPublishedDraftAsDraft returned error: %v", err)
	}
	if publishedHash == draftHash {
		t.Fatalf("support markdown status change was masked; hash = %s", publishedHash)
	}
}

func TestHashRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"", ".", "../outside", "/tmp/outside"} {
		if _, err := HashFile(root, rel); err == nil {
			t.Fatalf("HashFile(%q) expected error", rel)
		}
	}
}

func TestHashFileRejectsSymlinkOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(root, "knowledge/.inbox/packages/mall-api/redis-cache/secret.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir link parent: %v", err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink outside file: %v", err)
	}

	if _, err := HashFile(root, "knowledge/.inbox/packages/mall-api/redis-cache/secret.md"); err == nil {
		t.Fatalf("expected symlink escape error")
	}
}

func TestHashTreeRejectsSymlinkOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", "entrypoint\n")
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(root, "knowledge/.inbox/packages/mall-api/redis-cache/references/secret.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir link parent: %v", err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink outside file: %v", err)
	}

	if _, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache"); err == nil {
		t.Fatalf("expected tree symlink escape error")
	}
}

func TestHashTreeRejectsSymlinkedRootOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	outsideRoot := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(filepath.Join(outsideRoot, "empty-subdir"), 0o755); err != nil {
		t.Fatalf("mkdir outside root: %v", err)
	}
	link := filepath.Join(root, "knowledge/link")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir link parent: %v", err)
	}
	if err := os.Symlink(outsideRoot, link); err != nil {
		t.Fatalf("symlink outside root: %v", err)
	}

	if _, err := HashTree(root, "knowledge/link/empty-subdir"); err == nil {
		t.Fatalf("expected symlinked tree root escape error")
	}
}

func TestRecordJSONShape(t *testing.T) {
	record := Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-20260504-redis-cache-a1b2c3d4",
		State:         StateDraft,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
	}
	if record.SchemaVersion != "knowledge.provenance.v1" {
		t.Fatalf("unexpected schema version: %q", record.SchemaVersion)
	}
	if !strings.HasPrefix(record.ProvenanceID, "prov-") {
		t.Fatalf("unexpected provenance id: %q", record.ProvenanceID)
	}
}

func TestRecordOmitEmptyLatestCheck(t *testing.T) {
	record := Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-20260504-redis-cache-a1b2c3d4",
		State:         StateDraft,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if strings.Contains(string(data), "latest_check") {
		t.Fatalf("expected empty latest_check to be omitted, got %s", string(data))
	}
}

func TestStartCreatesInboxProvenanceRecord(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)

	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if record.State != StateDraft {
		t.Fatalf("expected draft state, got %#v", record)
	}
	if record.Subject.KnowledgeID != "package:mall-api.redis-cache.v1" {
		t.Fatalf("unexpected subject: %#v", record.Subject)
	}
	if record.Subject.OfficialPath != "knowledge/packages/mall-api/redis-cache" {
		t.Fatalf("unexpected official path: %#v", record.Subject)
	}
	if record.Hashes.DesignSHA256 == "" {
		t.Fatalf("expected design hash: %#v", record.Hashes)
	}
	loaded, err := Load(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Record.ProvenanceID != record.ProvenanceID {
		t.Fatalf("loaded wrong record: %#v", loaded.Record)
	}
}

func TestStartRejectsSymlinkedDesignBeforeParsing(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "design.json")
	if err := os.WriteFile(outside, []byte(`{not-json`), 0o644); err != nil {
		t.Fatalf("write outside design: %v", err)
	}
	link := filepath.Join(root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir design parent: %v", err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink design: %v", err)
	}

	_, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err == nil {
		t.Fatalf("expected symlinked design error")
	}
	if strings.Contains(err.Error(), "parse design JSON") {
		t.Fatalf("design was parsed before symlink rejection: %v", err)
	}
}

func TestStartRejectsSymlinkedProvenanceRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)
	outside := t.TempDir()
	inbox := filepath.Join(root, "knowledge/.inbox")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(inbox, "provenance")); err != nil {
		t.Fatalf("symlink provenance root: %v", err)
	}

	if _, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	}); err == nil {
		t.Fatalf("expected symlinked provenance root error")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatalf("read outside provenance dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected outside provenance dir to remain empty, got %d entries", len(entries))
	}
}

func TestLoadRejectsSymlinkedProvenanceRecord(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeTestFile(t, outside, "provenance.json", `{
  "schema_version": "knowledge.provenance.v1",
  "provenance_id": "prov-outside",
  "state": "draft",
  "subject": {},
  "hashes": {},
  "created_at": "2026-05-04T00:00:00Z",
  "created_by": "codex"
}`)
	inboxRoot := filepath.Join(root, "knowledge/.inbox/provenance")
	if err := os.MkdirAll(inboxRoot, 0o755); err != nil {
		t.Fatalf("mkdir inbox provenance: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(inboxRoot, "prov-outside")); err != nil {
		t.Fatalf("symlink provenance record: %v", err)
	}

	if _, err := Load(root, "prov-outside"); err == nil {
		t.Fatalf("expected symlinked provenance load error")
	}
}

func TestStartRetriesDuplicateProvenanceID(t *testing.T) {
	originalRandomBytes := randomBytes
	defer func() { randomBytes = originalRandomBytes }()
	calls := 0
	randomBytes = func(buf []byte) (int, error) {
		calls++
		if calls == 1 {
			copy(buf, []byte{0x01, 0x02, 0x03, 0x04})
			return len(buf), nil
		}
		copy(buf, []byte{0x05, 0x06, 0x07, 0x08})
		return len(buf), nil
	}

	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "title": "Redis Cache",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)
	existingID := "prov-" + time.Now().UTC().Format("20060102") + "-mall-api-redis-cache-01020304"
	writeTestFile(t, root, "knowledge/.inbox/provenance/"+existingID+"/provenance.json", `{"existing":true}`)

	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if record.ProvenanceID == existingID {
		t.Fatalf("expected duplicate id retry, got %s", record.ProvenanceID)
	}
	if calls != 2 {
		t.Fatalf("expected two random calls, got %d", calls)
	}
	existing, err := os.ReadFile(filepath.Join(root, "knowledge/.inbox/provenance", existingID, "provenance.json"))
	if err != nil {
		t.Fatalf("read existing provenance: %v", err)
	}
	if string(existing) != `{"existing":true}` {
		t.Fatalf("existing provenance was overwritten: %s", string(existing))
	}
}

func TestLoadAmbiguousPublishedProvenanceID(t *testing.T) {
	root := t.TempDir()
	for _, knowledgeID := range []string{"package_one", "package_two"} {
		writeTestFile(t, root, "knowledge/provenance/"+knowledgeID+"/prov-duplicate/provenance.json", `{
  "schema_version": "knowledge.provenance.v1",
  "provenance_id": "prov-duplicate",
  "state": "published",
  "subject": {},
  "hashes": {},
  "created_at": "2026-05-04T00:00:00Z",
  "created_by": "codex"
}`)
	}

	_, err := Load(root, "prov-duplicate")
	if err == nil {
		t.Fatalf("expected ambiguous provenance id error")
	}
	if !strings.Contains(err.Error(), "ambiguous provenance id") {
		t.Fatalf("expected ambiguous provenance id error, got %v", err)
	}
}

func TestRecordDecisionAppendsDecisionWithStageHashes(t *testing.T) {
	root := t.TempDir()
	record := startTestRecord(t, root)

	decision, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StageDesign,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Design scope is correct.",
		RecordedBy: "codex",
	})
	if err != nil {
		t.Fatalf("RecordDecision returned error: %v", err)
	}
	if decision.Hashes.DesignSHA256 == "" {
		t.Fatalf("expected design hash on decision: %#v", decision)
	}
	decisions, err := LoadDecisions(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("LoadDecisions returned error: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Stage != StageDesign {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}
}

func TestRecordDecisionRejectsInvalidDecisionInput(t *testing.T) {
	root := t.TempDir()
	record := startTestRecord(t, root)
	input := DecisionInput{
		Stage:      "unknown",
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Bad stage should fail.",
		RecordedBy: "codex",
	}
	if _, err := RecordDecision(root, record.ProvenanceID, input); err == nil {
		t.Fatalf("expected invalid stage error")
	}
}

func TestRecordCheckStoresCheckAndVerifyPasses(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)

	check, err := RecordCheck(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	if check.Result != "pass" {
		t.Fatalf("expected pass check, got %#v", check)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved after check passed.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "pass" {
		t.Fatalf("expected verify pass, got %#v", result)
	}
}

func TestVerifyFailsWhenDraftChangesAfterPublishDecision(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed after approval\n")

	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected verify fail, got %#v", result)
	}
	if !containsFinding(result.Findings, "draft tree hash changed") {
		t.Fatalf("expected draft hash finding, got %#v", result.Findings)
	}
}

func TestVerifyFailsWhenStoredLatestCheckDidNotPass(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	loaded, err := Load(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Record.LatestCheck == nil {
		t.Fatalf("expected latest check")
	}
	checkRel := filepath.ToSlash(filepath.Join(loaded.Dir, loaded.Record.LatestCheck.Path))
	writeTestFile(t, root, checkRel, `{"result":"fail"}`)
	checkHash, err := HashFile(root, checkRel)
	if err != nil {
		t.Fatalf("HashFile returned error: %v", err)
	}
	loaded.Record.Hashes.LatestCheckSHA256 = checkHash
	loaded.Record.LatestCheck.Result = "pass"
	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		t.Fatalf("resolve record path: %v", err)
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		t.Fatalf("write tampered record: %v", err)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}

	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected verify fail, got %#v", result)
	}
	if !containsFinding(result.Findings, "stored latest check must pass") {
		t.Fatalf("expected stored check finding, got %#v", result.Findings)
	}
}

func TestVerifyFailsWhenDesignChangesAfterDesignApproval(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", strings.Replace(validProvenanceDesignJSON(), "Create Redis cache best practices.", "Create updated Redis cache best practices.", 1))
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}

	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected verify fail, got %#v", result)
	}
	if !containsFinding(result.Findings, "design decision hashes do not match current record") {
		t.Fatalf("expected design decision hash finding, got %#v", result.Findings)
	}
	if !containsFinding(result.Findings, "draft_write decision hashes do not match current record") {
		t.Fatalf("expected draft_write decision hash finding, got %#v", result.Findings)
	}
}

func TestRecordCheckDoesNotOverwriteExistingCheckNumber(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	loaded, err := Load(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	checksDir := filepath.ToSlash(filepath.Join(loaded.Dir, "checks"))
	if err := os.Remove(filepath.Join(root, checksDir, "check-001.json")); err != nil {
		t.Fatalf("remove first check: %v", err)
	}
	writeTestFile(t, root, filepath.ToSlash(filepath.Join(checksDir, "check-002.json")), "existing evidence\n")

	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("second RecordCheck returned error: %v", err)
	}
	existing, err := os.ReadFile(filepath.Join(root, checksDir, "check-002.json"))
	if err != nil {
		t.Fatalf("read existing check: %v", err)
	}
	if string(existing) != "existing evidence\n" {
		t.Fatalf("existing check was overwritten: %s", string(existing))
	}
	loaded, err = Load(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Load after second check returned error: %v", err)
	}
	if loaded.Record.LatestCheck == nil || loaded.Record.LatestCheck.Path != "checks/check-003.json" {
		t.Fatalf("expected latest check-003.json, got %#v", loaded.Record.LatestCheck)
	}
}

func startPublishableTestRecord(t *testing.T, root string) Record {
	t.Helper()
	writeTestFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend]\nbusiness_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/types.yaml", "types: [package]\n")
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validProvenanceDesignJSON())
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", validProvenanceDraftPackage())
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "reference\n")
	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	return record
}

func recordDesignAndDraftApprovals(t *testing.T, root string, id string) {
	t.Helper()
	for _, stage := range []string{StageDesign, StageDraftWrite} {
		if _, err := RecordDecision(root, id, DecisionInput{
			Stage:      stage,
			Decision:   DecisionApproved,
			DecidedBy:  "chenchi",
			Role:       "knowledge_owner",
			Source:     "conversation",
			Reason:     stage + " approved.",
			RecordedBy: "codex",
		}); err != nil {
			t.Fatalf("RecordDecision(%s) returned error: %v", stage, err)
		}
	}
}

func containsFinding(findings []string, text string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, text) {
			return true
		}
	}
	return false
}

func validProvenanceDesignJSON() string {
	return `{
  "schema_version": "knowledge.design.v1",
  "user_request": "Create Redis cache best practices.",
  "knowledge_goal": "Document Redis cache best practices.",
  "project": "mall-api",
  "audience": {
    "primary": "implementer_agent",
    "agent_actions_supported": ["implement Redis cache behavior"]
  },
  "scope": {
    "projects": ["mall-api"],
    "stability": "reviewed",
    "distribution": "project",
    "tech_domains": ["backend"],
    "subject_domains": ["catalog"],
    "file_globs": ["**/*"]
  },
  "sources": {
    "user_input": ["User requested Redis cache best practices."],
    "observed": [],
    "imported": [],
    "ai_suggested": [],
    "templates": [],
    "examples": [],
    "assumptions": [],
    "open_questions": [],
    "claims": [{
      "claim": "Redis cache best practices are needed.",
      "kind": "fact",
      "source": ["user request"],
      "trust": "user_input",
      "requires_review": false
    }]
  },
  "draft_output": {
    "kind": "package",
    "type": "package",
    "title": "Redis Cache Best Practices",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache",
    "status": "draft",
    "priority": "should",
    "rationale": "A package can hold guidance and references.",
    "entrypoint_load": "read_before_implementation",
    "draft_state": "draft"
  },
  "future_use": {
    "trigger_requests": ["implement Redis cache behavior"],
    "negative_triggers": ["unrelated auth work"],
    "phases": ["implementation"],
    "query_phrases": ["redis cache best practices"],
    "expected_use": "Read before implementing Redis cache behavior.",
    "citation_policy": "cite_after_use",
    "missing_needs": []
  },
  "applicability": {
    "when_to_use": ["When implementing Redis cache behavior."],
    "when_not_to_use": ["When work does not touch Redis."],
    "tradeoffs": []
  },
  "existing_knowledge": {
    "official_matches": [],
    "inbox_matches": [],
    "possible_matches": [],
    "decision": "create_new",
    "reason": "No existing Redis cache package.",
    "review_choice_required": false
  },
  "write_boundary": {
    "path": "inbox",
    "write_requires_review_approval": true,
    "priority_must_approved": false,
    "publish_approved": true,
    "official_write_approved": false,
    "review_packet_required": true
  },
  "draft_files": [{
    "path": "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
    "purpose": "Package entrypoint.",
    "load": "read_before_implementation"
  }],
  "check_plan": {
    "validate_path": "knowledge/.inbox/packages/mall-api/redis-cache",
    "findability_checks": [{
      "project": "mall-api",
      "phase": "implementation",
      "task": "implement Redis cache behavior",
      "query": "redis cache best practices",
      "files": []
    }]
  },
  "review": {
    "questions": ["Is publication approved?"],
    "design_approved": true,
    "draft_write_approved": true,
    "priority_must_approved": false,
    "official_write_approved": false,
    "publish_approved": true
  }
}`
}

func validProvenanceDraftPackage() string {
	return `---
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
applies_to:
  files: ["**/*"]
---
## Purpose

Document Redis cache best practices.

## When To Use

Use when implementing Redis cache behavior.

## Start Here

Read before implementing Redis cache behavior.

## Load On Demand

- references/redis.md when deeper Redis detail is needed.
`
}

func startTestRecord(t *testing.T, root string) Record {
	t.Helper()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "id": "package:mall-api.redis-cache.v1",
    "title": "Redis Cache",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)
	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	return record
}

func writeTestFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
