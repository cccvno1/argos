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
				"Argos is a shared knowledge layer built from uploaded reusable knowledge.",
				"Do not let Argos replace host workflow, safety, or user instructions.",
				"Prefer MCP tools when available.",
				"Fall back to CLI JSON when MCP is unavailable and commands can be run.",
				"Fall back to generated adapter files or Markdown source when command execution is unavailable.",
				"Before substantial project work, call argos_context when available; otherwise follow equivalent adapter or Markdown guidance.",
				"Use argos_list_knowledge when available; otherwise use equivalent CLI JSON or adapter guidance for broad orientation before unfamiliar project work.",
				"Use argos_find_knowledge when available; otherwise use equivalent CLI JSON or adapter guidance to find shared knowledge for current work.",
				"Before implementation or review, call argos_standards when available; otherwise follow equivalent adapter or Markdown guidance.",
				"Follow next_steps returned by Argos.",
				"Read full knowledge items only through argos_read_knowledge when available; otherwise use equivalent CLI JSON or adapter guidance when routed to specific IDs or paths.",
				"Cite Argos knowledge IDs used in final responses only after reading and applying them.",
				"Do not cite IDs returned only by argos_list_knowledge or argos_find_knowledge.",
				"Cite only knowledge IDs whose full item was read with argos_read_knowledge or equivalent CLI JSON and actually applied.",
				"Follow usage.read before reading full knowledge items.",
				"Follow usage.cite before calling argos_cite_knowledge or equivalent CLI JSON citation.",
				"When missing_needs are present, separate Argos-backed claims from general reasoning.",
				"Do not cite missing_needs; they are unsupported needs, not knowledge items.",
				"Do not start upload, capture, or inbox creation from find results alone.",
				"Semantic search status never overrides usage guidance.",
				"Argos validation does not replace tests, builds, linting, or review.",
			} {
				if !strings.Contains(tt.body, expected) {
					t.Fatalf("expected %q in %s adapter:\n%s", expected, tt.name, tt.body)
				}
			}
			retired := []string{
				retiredAdapterTerm("argos", "map"),
				retiredAdapterTerm("argos", "discover"),
				retiredAdapterTerm("get", "knowledge", "item"),
				retiredAdapterTerm("coverage", "gaps"),
				retiredAdapterTerm("action", "policy"),
				"Semantic " + retiredAdapterTerm("re"+"call"),
			}
			for _, forbidden := range retired {
				if strings.Contains(tt.body, forbidden) {
					t.Fatalf("did not expect retired term %q in %s adapter:\n%s", forbidden, tt.name, tt.body)
				}
			}
			for _, forbidden := range legacyDiscoveryTerms() {
				if strings.Contains(tt.body, forbidden) {
					t.Fatalf("did not expect %q in %s adapter:\n%s", forbidden, tt.name, tt.body)
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

func TestGeneratedAdaptersDoNotRecommendDirectStorageQueries(t *testing.T) {
	root := t.TempDir()
	projects := []registry.Project{{
		ID:              "mall-api",
		Name:            "Mall API",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
	}}
	if err := Install(root, projects); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	body := readFile(t, filepath.Join(root, "argos", "generated", "mall-api", "cursor-rules", "argos.mdc"))
	for _, forbidden := range []string{"query SQLite", "query sqlite", "query vector", "knowledge_vectors", "knowledge_fts"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("adapter should not recommend direct storage access %q:\n%s", forbidden, body)
		}
	}
}

func TestRenderCursorRuleKeepsCursorFrontmatter(t *testing.T) {
	project := registry.Project{ID: "mall-api", Name: "Mall API"}
	body := RenderCursorRule(project)

	for _, expected := range []string{
		"---\n",
		"description: Argos progressive knowledge protocol for mall-api",
		"alwaysApply: true",
		"# Project Knowledge",
		"Argos is a shared knowledge layer built from uploaded reusable knowledge.",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in Cursor rule:\n%s", expected, body)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func legacyDiscoveryTerms() []string {
	return []string{
		legacyDiscoveryTerm("gap", "candidates"),
		legacyDiscoveryTerm("capture", "candidate"),
		legacyDiscoveryTerm("candidate", "only"),
		legacyDiscoveryTerm("proposal", "required"),
		"Start capture-knowledge only with user approval.",
		"official knowledge",
	}
}

func legacyDiscoveryTerm(parts ...string) string {
	// Build retired discovery terms from parts so repo-wide legacy-term scans stay focused on active surfaces.
	return strings.Join(parts, "_")
}

func retiredAdapterTerm(parts ...string) string {
	return strings.Join(parts, "_")
}
