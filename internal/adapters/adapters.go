package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/registry"
)

func Install(root string, projects []registry.Project) error {
	for _, project := range projects {
		dir := filepath.Join(root, "argos", "generated", project.ID)
		cursorRulesDir := filepath.Join(dir, "cursor-rules")
		if err := os.MkdirAll(cursorRulesDir, 0o755); err != nil {
			return fmt.Errorf("create adapter directory for %s: %w", project.ID, err)
		}

		files := map[string]string{
			filepath.Join(dir, "AGENTS.md"):            RenderAGENTS(project),
			filepath.Join(dir, "CLAUDE.md"):            RenderClaude(project),
			filepath.Join(dir, "GEMINI.md"):            RenderGemini(project),
			filepath.Join(cursorRulesDir, "argos.mdc"): RenderCursorRule(project),
		}
		for path, body := range files {
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	}
	return nil
}

func RenderAGENTS(project registry.Project) string {
	return renderMarkdown("AGENTS.md", project)
}

func RenderClaude(project registry.Project) string {
	return renderMarkdown("CLAUDE.md", project)
}

func RenderGemini(project registry.Project) string {
	return renderMarkdown("GEMINI.md", project)
}

func RenderCursorRule(project registry.Project) string {
	return fmt.Sprintf(`---
description: Argos progressive knowledge protocol for %s
alwaysApply: true
---

%s`, project.ID, renderMarkdown("Cursor", project))
}

func renderMarkdown(adapter string, project registry.Project) string {
	return fmt.Sprintf(`# Project Knowledge

Generated for %s by Argos.

project: %s
name: %s
tech_domains: %s
business_domains: %s

## Argos Protocol

1. Call argos_context to route the current request to relevant knowledge IDs.
2. Call argos_requirements to load requirement knowledge when implementation or behavior changes are requested.
3. Call argos_standards to load coding, architecture, and delivery standards before making changes.
4. Call argos_risks to check project-specific risks before finalizing work.
5. Call argos_operations to load operational runbooks when deployment, incidents, or maintenance are in scope.

Load full knowledge items only when routed to a specific ID.
Cite knowledge IDs used in final responses.
`, adapter, project.ID, project.Name, strings.Join(project.TechDomains, ", "), strings.Join(project.BusinessDomains, ", "))
}
