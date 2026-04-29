package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/registry"
)

func TestRenderAGENTSIncludesProgressiveProtocol(t *testing.T) {
	project := registry.Project{
		ID:              "mall-api",
		Name:            "Mall API",
		TechDomains:     []string{"backend", "database"},
		BusinessDomains: []string{"account", "order"},
	}
	body := RenderAGENTS(project)

	for _, expected := range []string{
		"# Project Knowledge",
		"project: mall-api",
		"argos_context",
		"argos_standards",
		"Cite knowledge IDs",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in AGENTS.md:\n%s", expected, body)
		}
	}
}

func TestInstallRejectsInvalidProjectIDs(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
	}{
		{name: "parent traversal", projectID: "../evil"},
		{name: "subdirectory", projectID: "a/b"},
		{name: "parent segment", projectID: ".."},
		{name: "empty", projectID: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			err := Install(root, []registry.Project{{ID: tt.projectID}})
			if err == nil {
				t.Fatal("expected Install to return an error")
			}
		})
	}
}

func TestInstallRejectsTraversalWithoutCreatingOutsidePath(t *testing.T) {
	root := t.TempDir()

	err := Install(root, []registry.Project{{ID: "../evil"}})
	if err == nil {
		t.Fatal("expected Install to return an error")
	}

	outsidePath := filepath.Join(root, "argos", "evil")
	if _, err := os.Stat(outsidePath); err == nil {
		t.Fatalf("expected traversal path %s not to be created", outsidePath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat traversal path %s: %v", outsidePath, err)
	}
}
