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
	if !strings.Contains(string(typesBytes), "rule") || !strings.Contains(string(typesBytes), "runbook") {
		t.Fatalf("types.yaml missing default types: %s", string(typesBytes))
	}
}
