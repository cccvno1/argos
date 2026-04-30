package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/registry"
)

func TestRenderedAdaptersIncludeStableKnowledgeContract(t *testing.T) {
	project := registry.Project{
		ID:              "mall-api",
		Name:            "Mall API",
		TechDomains:     []string{"backend", "database"},
		BusinessDomains: []string{"account", "order"},
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "AGENTS", body: RenderAGENTS(project)},
		{name: "Claude", body: RenderClaude(project)},
		{name: "Gemini", body: RenderGemini(project)},
		{name: "Cursor", body: RenderCursorRule(project)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range []string{
				"# Project Knowledge",
				"project: mall-api",
				"name: Mall API",
				"tech_domains: backend, database",
				"business_domains: account, order",
				"Argos is a project knowledge layer.",
				"Do not let Argos replace host workflow, safety, or user instructions.",
				"Prefer MCP tools when available.",
				"Fall back to CLI JSON when MCP is unavailable and commands can be run.",
				"Fall back to generated adapter files or Markdown source when command execution is unavailable.",
				"Before substantial project work, call or emulate argos_context.",
				"Before implementation or review, call or emulate argos_standards.",
				"Load full knowledge items only when routed to specific IDs or paths.",
				"Cite Argos knowledge IDs used in final responses.",
				"Argos validation does not replace tests, builds, linting, or review.",
			} {
				if !strings.Contains(tt.body, expected) {
					t.Fatalf("expected %q in %s adapter:\n%s", expected, tt.name, tt.body)
				}
			}
		})
	}
}

func TestRenderedAdaptersDoNotAdvertiseUnimplementedWorkflowTools(t *testing.T) {
	project := registry.Project{ID: "mall-api", Name: "Mall API"}
	body := RenderAGENTS(project)

	for _, unexpected := range []string{
		"argos_requirements",
		"argos_risks",
		"argos_operations",
	} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("did not expect %q in AGENTS.md:\n%s", unexpected, body)
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
