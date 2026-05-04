package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
