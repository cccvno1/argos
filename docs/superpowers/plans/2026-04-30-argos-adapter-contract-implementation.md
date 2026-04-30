# Argos Adapter Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strengthen generated adapter files so different AI coding tools receive the same stable Argos knowledge-consumption contract.

**Architecture:** Keep adapter generation in `internal/adapters`. Add tests that define the shared contract text once, then update the renderer so AGENTS, Claude, Gemini, and Cursor outputs all include the same host-workflow compatibility rules, retrieval fallback order, progressive-loading behavior, and citation requirement. Do not add new CLI commands or MCP tools in this plan.

**Tech Stack:** Go, table-driven tests, existing `registry.Project`, Markdown adapter files, Cursor `.mdc` wrapper.

---

## File Structure

- Modify `internal/adapters/adapters_test.go`: add contract-focused assertions for all generated adapter outputs.
- Modify `internal/adapters/adapters.go`: centralize adapter contract rendering and update generated Markdown.
- Modify `README.md`: keep the agent-facing command section aligned with the stronger adapter contract.

## Task 1: Lock The Shared Adapter Contract With Tests

**Files:**
- Modify: `internal/adapters/adapters_test.go`

- [x] **Step 1: Replace the current single-output test with a table-driven contract test**

Replace `TestRenderAGENTSIncludesProgressiveProtocol` with:

```go
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
```

- [x] **Step 2: Keep unsupported-tool guardrails**

Add this test after the new table-driven contract test:

```go
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
```

- [x] **Step 3: Run the focused test and verify it fails**

Run:

```bash
go test ./internal/adapters -run 'TestRenderedAdapters' -count=1
```

Expected: FAIL because the current rendered adapters do not include the stronger contract strings.

- [x] **Step 4: Commit the failing tests**

Run:

```bash
git add internal/adapters/adapters_test.go
git commit -m "test: define adapter knowledge contract"
```

## Task 2: Render The Stronger Adapter Contract

**Files:**
- Modify: `internal/adapters/adapters.go`

- [x] **Step 1: Replace `renderMarkdown` with contract sections**

Update `renderMarkdown` to this shape:

```go
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
2. Before implementation or review, call or emulate argos_standards.
3. Follow recommended calls returned by Argos.
4. Load full knowledge items only when routed to specific IDs or paths.
5. Cite Argos knowledge IDs used in final responses.

## Boundaries

Argos validation does not replace tests, builds, linting, or review.
Official knowledge should not be changed without the user's explicit approval.
`, adapter, project.ID, project.Name, strings.Join(project.TechDomains, ", "), strings.Join(project.BusinessDomains, ", "))
}
```

- [x] **Step 2: Run the focused adapter tests**

Run:

```bash
go test ./internal/adapters -run 'TestRenderedAdapters' -count=1
```

Expected: PASS.

- [x] **Step 3: Run all adapter tests**

Run:

```bash
go test ./internal/adapters -count=1
```

Expected: PASS.

- [x] **Step 4: Commit the renderer change**

Run:

```bash
git add internal/adapters/adapters.go
git commit -m "feat: strengthen generated adapter contract"
```

## Task 3: Preserve Cursor Wrapper Behavior

**Files:**
- Modify: `internal/adapters/adapters_test.go`

- [x] **Step 1: Add a Cursor-specific wrapper test**

Add this test near the adapter rendering tests:

```go
func TestRenderCursorRuleKeepsCursorFrontmatter(t *testing.T) {
	project := registry.Project{ID: "mall-api", Name: "Mall API"}
	body := RenderCursorRule(project)

	for _, expected := range []string{
		"---\n",
		"description: Argos progressive knowledge protocol for mall-api",
		"alwaysApply: true",
		"# Project Knowledge",
		"Argos is a project knowledge layer.",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in Cursor rule:\n%s", expected, body)
		}
	}
}
```

- [x] **Step 2: Run the Cursor wrapper test**

Run:

```bash
go test ./internal/adapters -run TestRenderCursorRuleKeepsCursorFrontmatter -count=1
```

Expected: PASS.

- [x] **Step 3: Commit the wrapper test**

Run:

```bash
git add internal/adapters/adapters_test.go
git commit -m "test: cover cursor adapter wrapper"
```

## Task 4: Align README Adapter Documentation

**Files:**
- Modify: `README.md`

- [x] **Step 1: Update the Agent Experience section**

In `README.md`, keep the existing integration-path list and add this paragraph after the retrieval order block:

```md
Generated adapters define a minimum contract for tools that read project
instruction files: preserve host workflow control, prefer MCP, fall back to CLI
JSON or Markdown source, load full knowledge only when routed, and cite Argos
knowledge IDs that informed final responses.
```

- [x] **Step 2: Verify the README contains the contract summary**

Run:

```bash
rg -n "Generated adapters define a minimum contract|preserve host workflow control|prefer MCP|cite Argos knowledge IDs" README.md
```

Expected: the new paragraph is matched.

- [x] **Step 3: Commit the docs alignment**

Run:

```bash
git add README.md
git commit -m "docs: describe adapter contract"
```

## Task 5: Full Verification

**Files:**
- Verify all changed files.

- [x] **Step 1: Format Go files**

Run:

```bash
gofmt -w internal/adapters/adapters.go internal/adapters/adapters_test.go
```

Expected: command exits 0.

- [x] **Step 2: Run full tests**

Run:

```bash
go test ./...
```

Expected: PASS for all packages.

- [x] **Step 3: Run vet**

Run:

```bash
go vet ./...
```

Expected: exit 0.

- [x] **Step 4: Check diff hygiene**

Run:

```bash
git diff --check
```

Expected: exit 0.

- [x] **Step 5: Commit formatting-only changes if any exist**

Run:

```bash
git status --short
```

If `gofmt` changed files, run:

```bash
git add internal/adapters/adapters.go internal/adapters/adapters_test.go
git commit -m "style: format adapter contract changes"
```

If no files are changed, do not create a commit.

## Self-Review

Spec coverage:

- Host workflow non-conflict: Task 1 and Task 2 add adapter contract text and tests.
- Integration fallback order: Task 1 and Task 2 cover MCP, CLI JSON, generated adapters, and Markdown source.
- Progressive loading: Task 1 and Task 2 keep full item loading gated by routed IDs or paths.
- Citation requirement: Task 1 and Task 2 require final-response citations for used Argos knowledge IDs.
- Cursor compatibility: Task 3 covers the `.mdc` wrapper.
- README alignment: Task 4 documents the generated adapter contract.

No planned task adds new MCP tools or changes query behavior; guide, lesson, risk, and package discovery remain future work.
