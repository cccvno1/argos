package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesWorkspaceFiles(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	required := []string{
		"knowledge/domains.yaml",
		"knowledge/projects.yaml",
		"knowledge/types.yaml",
		"knowledge/.inbox",
		"knowledge/items",
		"knowledge/packages",
		"argos/generated",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	typesBytes, err := os.ReadFile(filepath.Join(root, "knowledge/types.yaml"))
	if err != nil {
		t.Fatalf("read types.yaml: %v", err)
	}
	if !strings.Contains(string(typesBytes), "rule") ||
		!strings.Contains(string(typesBytes), "runbook") ||
		!strings.Contains(string(typesBytes), "package") {
		t.Fatalf("types.yaml missing default types: %s", string(typesBytes))
	}
}

func TestInitCreatesAuthoringInboxDirectories(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	required := []string{
		"knowledge/.inbox/items",
		"knowledge/.inbox/packages",
		"knowledge/.inbox/proposals",
	}
	for _, rel := range required {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", rel)
		}
	}
}

func TestInitPreservesExistingRegularConfigContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "knowledge", "types.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create knowledge dir: %v", err)
	}
	const existing = "types:\n  - custom\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	if err := Init(root); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	typesBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read types.yaml: %v", err)
	}
	if string(typesBytes) != existing {
		t.Fatalf("expected existing config to be preserved, got %q", string(typesBytes))
	}
}

func TestInitErrorsWhenRegistryPathIsDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "knowledge", "domains.yaml")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create registry directory: %v", err)
	}

	err := Init(root)
	if err == nil {
		t.Fatal("expected Init to return an error")
	}
	if !strings.Contains(err.Error(), "knowledge/domains.yaml") {
		t.Fatalf("expected error to include registry path, got %v", err)
	}
}
