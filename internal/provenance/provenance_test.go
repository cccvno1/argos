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
