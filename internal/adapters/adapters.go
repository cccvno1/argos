package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/registry"
)

func Install(root string, projects []registry.Project) error {
	baseDir := filepath.Join(root, "argos", "generated")
	for _, project := range projects {
		dir, err := projectDir(baseDir, project.ID)
		if err != nil {
			return err
		}
		cursorRulesDir := filepath.Join(dir, "cursor-rules")
		if err := os.MkdirAll(cursorRulesDir, 0o755); err != nil {
			return fmt.Errorf("create adapter directory for %s: %w", project.ID, err)
		}

		files := []struct {
			rel  string
			body string
		}{
			{rel: "AGENTS.md", body: RenderAGENTS(project)},
			{rel: "CLAUDE.md", body: RenderClaude(project)},
			{rel: "GEMINI.md", body: RenderGemini(project)},
			{rel: filepath.Join("cursor-rules", "argos.mdc"), body: RenderCursorRule(project)},
		}
		for _, file := range files {
			path := filepath.Join(dir, file.rel)
			if err := os.WriteFile(path, []byte(file.body), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	}
	return nil
}

func projectDir(baseDir string, projectID string) (string, error) {
	if projectID == "" {
		return "", fmt.Errorf("invalid project id: empty")
	}
	if filepath.IsAbs(projectID) || projectID == "." || projectID == ".." || filepath.Clean(projectID) != projectID {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}
	if strings.Contains(projectID, "/") || strings.Contains(projectID, `\`) {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}

	dir := filepath.Join(baseDir, projectID)
	rel, err := filepath.Rel(baseDir, dir)
	if err != nil {
		return "", fmt.Errorf("verify adapter path for %q: %w", projectID, err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}
	return dir, nil
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

## Argos Role

Argos is a project knowledge layer. Use it to retrieve, apply, validate, and cite durable project knowledge.
Do not let Argos replace host workflow, safety, or user instructions.

## Retrieval Order

1. Prefer MCP tools when available.
2. Fall back to CLI JSON when MCP is unavailable and commands can be run.
3. Fall back to generated adapter files or Markdown source when command execution is unavailable.

## Work Protocol

1. Before substantial project work, call or emulate argos_context.
2. Use argos_map for broad orientation before unfamiliar project work.
3. Use argos_discover to route current work to relevant Argos knowledge.
4. Before implementation or review, call or emulate argos_standards.
5. Follow recommended calls returned by Argos.
6. Load full knowledge items only through get_knowledge_item when routed to specific IDs or paths.
7. Cite Argos knowledge IDs used in final responses.
8. Do not cite IDs returned only by argos_map or argos_discover.
9. Cite only knowledge IDs whose full item was loaded with get_knowledge_item and actually applied.
10. Follow action_policy.load before loading full knowledge items.
11. Follow action_policy.cite before calling cite_knowledge.
12. Mention Argos coverage gaps when action_policy.claim is must_mention_gap.

## Boundaries

Argos validation does not replace tests, builds, linting, or review.
Official knowledge should not be changed without the user's explicit approval.
`, adapter, project.ID, project.Name, strings.Join(project.TechDomains, ", "), strings.Join(project.BusinessDomains, ", "))
}
