package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListReturnsDraftAndPublishedRecords(t *testing.T) {
	root := t.TempDir()
	writeListRecord(t, root, "knowledge/provenance/package_mall-api.redis-cache.v1/prov-published/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-published",
		State:         StatePublished,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
		LatestCheck: &LatestCheck{Result: "pass"},
		CreatedAt:   "2026-05-04T00:00:00Z",
		CreatedBy:   "codex",
		PublishedAt: "2026-05-04T00:10:00Z",
	})
	writeListRecord(t, root, "knowledge/.inbox/provenance/prov-draft/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-draft",
		State:         StateDraft,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
		CreatedAt: "2026-05-04T00:00:00Z",
		CreatedBy: "codex",
	})

	result, err := List(root, ListFilter{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %#v", result.Records)
	}
	if result.Records[0].ProvenanceID != "prov-draft" || result.Records[1].ProvenanceID != "prov-published" {
		t.Fatalf("records should be sorted by path: %#v", result.Records)
	}
	if result.Records[0].Path != "knowledge/.inbox/provenance/prov-draft" ||
		result.Records[1].Path != "knowledge/provenance/package_mall-api.redis-cache.v1/prov-published" {
		t.Fatalf("records should be sorted by path: %#v", result.Records)
	}
}

func TestListFiltersByStateProjectAndKnowledgeID(t *testing.T) {
	root := t.TempDir()
	writeListRecord(t, root, "knowledge/.inbox/provenance/prov-mall/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-mall",
		State:         StateDraft,
		Subject:       Subject{Project: "mall-api", KnowledgeID: "package:mall-api.redis-cache.v1"},
	})
	writeListRecord(t, root, "knowledge/provenance/package_billing.redis.v1/prov-billing/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-billing",
		State:         StatePublished,
		Subject:       Subject{Project: "billing-api", KnowledgeID: "package:billing.redis.v1"},
	})

	result, err := List(root, ListFilter{
		State:       StateDraft,
		Project:     "mall-api",
		KnowledgeID: "package:mall-api.redis-cache.v1",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result.Records) != 1 || result.Records[0].ProvenanceID != "prov-mall" {
		t.Fatalf("unexpected filtered records: %#v", result.Records)
	}
}

func TestListRejectsInvalidState(t *testing.T) {
	_, err := List(t.TempDir(), ListFilter{State: "ready"})
	if err == nil || err.Error() != "state must be draft, published, or all" {
		t.Fatalf("expected invalid state error, got %v", err)
	}
}

func TestListRejectsSymlinkedProvenanceRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	inbox := filepath.Join(root, "knowledge/.inbox")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(inbox, "provenance")); err != nil {
		t.Fatalf("symlink provenance root: %v", err)
	}

	_, err := List(root, ListFilter{})
	if err == nil || !strings.Contains(err.Error(), "path must not contain symlinks") {
		t.Fatalf("expected symlinked provenance root error, got %v", err)
	}
}

func TestListRejectsSymlinkedIntermediateProvenanceRootComponent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "provenance"), 0o755); err != nil {
		t.Fatalf("mkdir outside provenance: %v", err)
	}
	knowledge := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(knowledge, 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(knowledge, ".inbox")); err != nil {
		t.Fatalf("symlink inbox: %v", err)
	}

	_, err := List(root, ListFilter{})
	if err == nil || !strings.Contains(err.Error(), "path must not contain symlinks") {
		t.Fatalf("expected symlinked provenance root component error, got %v", err)
	}
}

func TestListRejectsSymlinkedProvenanceRecordDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeListRecord(t, outside, "provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-outside",
		State:         StateDraft,
	})
	inboxRoot := filepath.Join(root, "knowledge/.inbox/provenance")
	if err := os.MkdirAll(inboxRoot, 0o755); err != nil {
		t.Fatalf("mkdir inbox provenance: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(inboxRoot, "prov-outside")); err != nil {
		t.Fatalf("symlink provenance record: %v", err)
	}

	_, err := List(root, ListFilter{})
	if err == nil || !strings.Contains(err.Error(), "path must not contain symlinks") {
		t.Fatalf("expected symlinked provenance record error, got %v", err)
	}
}

func writeListRecord(t *testing.T, root string, rel string, record Record) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir record parent: %v", err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(abs, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}
}
