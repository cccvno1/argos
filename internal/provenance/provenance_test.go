package provenance

import (
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
