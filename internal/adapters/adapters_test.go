package adapters

import (
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
