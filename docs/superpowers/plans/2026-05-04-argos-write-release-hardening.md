# Argos Write Release Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the write side release-ready by adding project registration commands, enforcing storage status contracts, and documenting the final write contract.

**Architecture:** Keep Argos filesystem-first. The registry package owns structured reads/writes of `knowledge/projects.yaml`; the CLI owns command parsing and user-facing output; validation scope logic stays in the CLI because it is tied to `argos validate`, `knowledge check`, and `knowledge publish` behavior. Audit/review-history storage is intentionally deferred.

**Tech Stack:** Go CLI, `gopkg.in/yaml.v3`, existing registry/knowledge/knowledgewrite packages, JSON CLI output, Markdown docs.

---

## File Structure

- Modify `internal/registry/registry.go`: add JSON tags to `Project` and reuse existing YAML helpers.
- Create `internal/registry/projects.go`: project list/add registry API and structured YAML write.
- Modify `internal/registry/registry_test.go`: registry-level project add/list tests.
- Modify `internal/cli/cli.go`: add `project` command group, project CLI handlers, and storage scope validation.
- Modify `internal/cli/cli_test.go`: CLI project tests, fresh write flow test updates, validate scope tests, publish guard tests.
- Modify `README.md`: project CLI, storage contract, publish contract, dogfood source-tree decision.
- Modify `skills/capture-knowledge/SKILL.md`: agent workflow uses `project add/list` and storage contract.
- Modify `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`: mark write side release-ready and defer audit trail.

---

### Task 1: Registry Project Add/List API

**Files:**
- Create: `internal/registry/projects.go`
- Modify: `internal/registry/registry.go`
- Modify: `internal/registry/registry_test.go`

- [ ] **Step 1: Write failing registry tests**

Append these tests to `internal/registry/registry_test.go` before `writeRegistryFile`:

```go
func TestAddProjectWritesStructuredProjectsYAML(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, database]
business_domains: [account, catalog]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"backend", "database", "backend", " "},
		BusinessDomains: []string{"account", "catalog", "account"},
	})
	if err != nil {
		t.Fatalf("AddProject returned error: %v", err)
	}

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.Projects) != 1 {
		t.Fatalf("expected one project, got %#v", reg.Projects)
	}
	got := reg.Projects[0]
	if got.ID != "mall-api" || got.Name != "Mall API" || got.Path != "services/mall-api" {
		t.Fatalf("unexpected project identity: %#v", got)
	}
	if strings.Join(got.TechDomains, ",") != "backend,database" {
		t.Fatalf("unexpected tech domains: %#v", got.TechDomains)
	}
	if strings.Join(got.BusinessDomains, ",") != "account,catalog" {
		t.Fatalf("unexpected business domains: %#v", got.BusinessDomains)
	}
}

func TestAddProjectRejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{ID: "mall-api", Name: "Mall API 2", Path: "services/mall-api-2"})
	if err == nil {
		t.Fatal("expected duplicate project error")
	}
	if !strings.Contains(err.Error(), "project already exists: mall-api") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddProjectRejectsUnknownDomains(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"mobile"},
		BusinessDomains: []string{"catalog"},
	})
	if err == nil {
		t.Fatal("expected unknown domain error")
	}
	for _, want := range []string{"unknown tech domain: mobile", "unknown business domain: catalog"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestListProjectsReturnsRegisteredProjects(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	projects, err := ListProjects(root)
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}
```

- [ ] **Step 2: Run registry tests and verify failure**

Run:

```bash
go test ./internal/registry -run 'TestAddProject|TestListProjects' -count=1
```

Expected: fail with undefined `AddProject` and `ListProjects`.

- [ ] **Step 3: Add JSON tags to registry projects**

In `internal/registry/registry.go`, update `Project` to carry both YAML and JSON tags:

```go
type Project struct {
	ID              string   `yaml:"id" json:"id"`
	Name            string   `yaml:"name" json:"name"`
	Path            string   `yaml:"path" json:"path"`
	TechDomains     []string `yaml:"tech_domains" json:"tech_domains"`
	BusinessDomains []string `yaml:"business_domains" json:"business_domains"`
}
```

- [ ] **Step 4: Implement registry project API**

Create `internal/registry/projects.go` with:

```go
package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type projectsFile struct {
	Projects []Project `yaml:"projects"`
}

func ListProjects(root string) ([]Project, error) {
	file, err := loadProjectsFile(root)
	if err != nil {
		return nil, err
	}
	return append([]Project{}, file.Projects...), nil
}

func AddProject(root string, project Project) error {
	project = normalizeProject(project)
	if err := validateProjectRequired(project); err != nil {
		return err
	}
	reg, err := Load(root)
	if err != nil {
		return err
	}
	for _, existing := range reg.Projects {
		if existing.ID == project.ID {
			return fmt.Errorf("project already exists: %s", project.ID)
		}
	}
	var failures []string
	for _, domain := range project.TechDomains {
		if !containsRegistryValue(reg.TechDomains, domain) {
			failures = append(failures, fmt.Sprintf("unknown tech domain: %s", domain))
		}
	}
	for _, domain := range project.BusinessDomains {
		if !containsRegistryValue(reg.BusinessDomains, domain) {
			failures = append(failures, fmt.Sprintf("unknown business domain: %s", domain))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	file, err := loadProjectsFile(root)
	if err != nil {
		return err
	}
	file.Projects = append(file.Projects, project)
	return writeProjectsFile(root, file)
}

func loadProjectsFile(root string) (projectsFile, error) {
	var file projectsFile
	if err := readYAML(filepath.Join(root, "knowledge", "projects.yaml"), &file); err != nil {
		return projectsFile{}, err
	}
	return file, nil
}

func writeProjectsFile(root string, file projectsFile) error {
	var body bytes.Buffer
	encoder := yaml.NewEncoder(&body)
	encoder.SetIndent(2)
	if err := encoder.Encode(file); err != nil {
		return fmt.Errorf("encode projects registry: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("encode projects registry: %w", err)
	}
	path := filepath.Join(root, "knowledge", "projects.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create projects registry parent: %w", err)
	}
	if err := os.WriteFile(path, body.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func normalizeProject(project Project) Project {
	project.ID = strings.TrimSpace(project.ID)
	project.Name = strings.TrimSpace(project.Name)
	project.Path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(project.Path)))
	if project.Path == "." {
		project.Path = ""
	}
	project.TechDomains = uniqueRegistryValues(project.TechDomains)
	project.BusinessDomains = uniqueRegistryValues(project.BusinessDomains)
	return project
}

func validateProjectRequired(project Project) error {
	var failures []string
	if project.ID == "" {
		failures = append(failures, "project id is required")
	}
	if project.Name == "" {
		failures = append(failures, "project name is required")
	}
	if project.Path == "" {
		failures = append(failures, "project path is required")
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func uniqueRegistryValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsRegistryValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run registry tests and verify pass**

Run:

```bash
go test ./internal/registry -run 'TestAddProject|TestListProjects|TestLoad' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit registry API**

Run:

```bash
git add internal/registry/registry.go internal/registry/projects.go internal/registry/registry_test.go
git commit -m "feat: add project registry writer"
```

---

### Task 2: CLI Project Add/List

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI project tests**

Add these tests near the other CLI command tests in `internal/cli/cli_test.go`:

```go
func TestRunProjectAddCreatesProject(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--tech-domain", "database",
		"--business-domain", "account",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if strings.TrimSpace(stdout.String()) != "added project mall-api" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	reg, err := registry.Load(root)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if len(reg.Projects) != 1 || reg.Projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects: %#v", reg.Projects)
	}
}

func TestRunProjectListReturnsJSON(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})
	runOK(t, root, []string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--business-domain", "account",
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "list", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var result struct {
		Projects []registry.Project `json:"projects"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Projects) != 1 || result.Projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects JSON: %s", stdout.String())
	}
	for _, want := range []string{`"id": "mall-api"`, `"tech_domains": [`, `"business_domains": [`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected JSON field %q, got: %s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), `"ID"`) || strings.Contains(stdout.String(), `"TechDomains"`) {
		t.Fatalf("project list should use snake_case JSON fields, got: %s", stdout.String())
	}
}

func TestRunProjectAddRequiresFields(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "add", "--id", "mall-api"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "project add: --name is required") {
		t.Fatalf("expected missing name error, got %q", stderr.String())
	}
}

func TestRunProjectListRequiresJSON(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "list"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "project list: --json is required") {
		t.Fatalf("expected missing json error, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run CLI project tests and verify failure**

Run:

```bash
go test ./internal/cli -run 'TestRunProject' -count=1
```

Expected: fail with unknown command `project`.

- [ ] **Step 3: Add project command handlers**

Modify `internal/cli/cli.go`:

Add a top-level switch case in `run`:

```go
	case "project":
		return runProject(args[1:], stdout, stderr)
```

Add these functions after `runDogfoodEvaluate` and before `runKnowledge`:

```go
func runProject(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "project: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "add":
		return runProjectAdd(args[1:], stdout, stderr)
	case "list":
		return runProjectList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "project: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runProjectAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("project add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	id := flags.String("id", "", "project id")
	name := flags.String("name", "", "project name")
	projectPath := flags.String("path", "", "project source path")
	var techDomains multiValueFlag
	var businessDomains multiValueFlag
	flags.Var(&techDomains, "tech-domain", "tech domain; may be repeated")
	flags.Var(&businessDomains, "business-domain", "business domain; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(stderr, "project add: --id is required")
		return 2
	}
	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(stderr, "project add: --name is required")
		return 2
	}
	if strings.TrimSpace(*projectPath) == "" {
		fmt.Fprintln(stderr, "project add: --path is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "project add: get current directory: %v\n", err)
		return 1
	}
	project := registry.Project{
		ID:              *id,
		Name:            *name,
		Path:            *projectPath,
		TechDomains:     techDomains,
		BusinessDomains: businessDomains,
	}
	if err := registry.AddProject(root, project); err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "added project %s\n", strings.TrimSpace(*id))
	return 0
}

func runProjectList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("project list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "project list: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "project list: get current directory: %v\n", err)
		return 1
	}
	projects, err := registry.ListProjects(root)
	if err != nil {
		fmt.Fprintf(stderr, "project list: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, struct {
		Projects []registry.Project `json:"projects"`
	}{Projects: projects})
}
```

In `printUsage`, add:

```go
	fmt.Fprintln(w, "  project")
```

Add examples:

```go
	fmt.Fprintln(w, "  argos project add --id <project> --name <name> --path <path>")
	fmt.Fprintln(w, "  argos project list --json")
```

- [ ] **Step 4: Run CLI project tests and verify pass**

Run:

```bash
go test ./internal/cli -run 'TestRunProject|TestUsageUsesWriteVocabulary' -count=1
```

Expected: pass.

- [ ] **Step 5: Commit CLI project commands**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: add project CLI commands"
```

---

### Task 3: Fresh Workspace Write Flow Uses Project Command

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Update the write findback flow test**

In `TestKnowledgeWritePublishAndFindbackFlow`, replace `initWorkspace(t, root)` with a real `init` plus `project add` path:

```go
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})
	runOK(t, root, []string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--business-domain", "account",
	})
```

Keep the rest of the test unchanged. The draft and design already use `mall-api`
and the helper draft must use the default `account` business domain.

In `writeDraftPackageForCLI`, change the frontmatter business domain from
`catalog` to `account`:

```yaml
business_domains: [account]
```

- [ ] **Step 2: Run the updated write flow test**

Run:

```bash
go test ./internal/cli -run TestKnowledgeWritePublishAndFindbackFlow -count=1
```

Expected: pass.

- [ ] **Step 3: Commit fresh workspace flow**

Run:

```bash
git add internal/cli/cli_test.go
git commit -m "test: cover fresh write flow with project registration"
```

---

### Task 4: Storage Contract Validation

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing storage validation tests**

Add these tests near `TestRunValidatePrintsValidationErrors` in `internal/cli/cli_test.go`:

```go
func TestRunValidateRejectsOfficialDraftStatus(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/items/backend/draft.md", `---
id: rule:backend.draft.v1
title: Draft Official Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: draft
priority: should
updated_at: 2026-05-04
---
Official roots must not carry draft status.
`)
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "official knowledge must not use status: draft") {
		t.Fatalf("expected official draft error, got %q", stderr.String())
	}
}

func TestRunValidateInboxRejectsActiveStatus(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/items/backend/active.md", `---
id: rule:backend.active-draft.v1
title: Active Inbox Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: should
updated_at: 2026-05-04
---
Inbox roots must carry draft status.
`)
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", "--inbox"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "inbox knowledge must use status: draft") {
		t.Fatalf("expected inbox status error, got %q", stderr.String())
	}
}

func TestRunValidatePathDerivesStorageScope(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/items/backend/active.md", `---
id: rule:backend.active-draft.v1
title: Active Inbox Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: should
updated_at: 2026-05-04
---
Inbox roots must carry draft status.
`)
	writeCLIFile(t, root, "knowledge/items/backend/draft.md", `---
id: rule:backend.official-draft.v1
title: Draft Official Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: draft
priority: should
updated_at: 2026-05-04
---
Official roots must not carry draft status.
`)
	chdir(t, root)

	var inboxStdout, inboxStderr bytes.Buffer
	inboxCode := Run([]string{"validate", "--path", "knowledge/.inbox/items/backend/active.md"}, &inboxStdout, &inboxStderr)
	if inboxCode != 1 || !strings.Contains(inboxStderr.String(), "inbox knowledge must use status: draft") {
		t.Fatalf("expected inbox path status error, code=%d stderr=%q", inboxCode, inboxStderr.String())
	}

	var officialStdout, officialStderr bytes.Buffer
	officialCode := Run([]string{"validate", "--path", "knowledge/items/backend/draft.md"}, &officialStdout, &officialStderr)
	if officialCode != 1 || !strings.Contains(officialStderr.String(), "official knowledge must not use status: draft") {
		t.Fatalf("expected official path status error, code=%d stderr=%q", officialCode, officialStderr.String())
	}
}
```

- [ ] **Step 2: Run storage validation tests and verify failure**

Run:

```bash
go test ./internal/cli -run 'TestRunValidateRejectsOfficialDraftStatus|TestRunValidateInboxRejectsActiveStatus|TestRunValidatePathDerivesStorageScope' -count=1
```

Expected: fail because status storage scope is not enforced.

- [ ] **Step 3: Add storage scope validation**

Modify `internal/cli/cli.go`.

Replace `validationScope` with:

```go
type validationScope struct {
	Inbox bool
	Path  string
}

type knowledgeStorageScope string

const (
	knowledgeStorageOfficial knowledgeStorageScope = "official"
	knowledgeStorageInbox    knowledgeStorageScope = "inbox"
	knowledgeStorageUnknown  knowledgeStorageScope = "unknown"
)
```

In `loadAndValidateKnowledge`, after `knowledge.ValidateItems(items, reg)`, append storage scope errors:

```go
	errs := knowledge.ValidateItems(items, reg)
	errs = append(errs, validateKnowledgeStorageScope(items, storageScopeForValidation(scope))...)
```

Add helper functions near `loadAndValidateKnowledge`:

```go
func storageScopeForValidation(scope validationScope) knowledgeStorageScope {
	if strings.TrimSpace(scope.Path) != "" {
		return storageScopeForPath(scope.Path)
	}
	if scope.Inbox {
		return knowledgeStorageInbox
	}
	return knowledgeStorageOfficial
}

func storageScopeForPath(path string) knowledgeStorageScope {
	slash := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if slash == "." || slash == "" {
		return knowledgeStorageUnknown
	}
	if slash == "knowledge/.inbox/items" ||
		slash == "knowledge/.inbox/packages" ||
		strings.HasPrefix(slash, "knowledge/.inbox/items/") ||
		strings.HasPrefix(slash, "knowledge/.inbox/packages/") {
		return knowledgeStorageInbox
	}
	if slash == "knowledge/items" ||
		slash == "knowledge/packages" ||
		strings.HasPrefix(slash, "knowledge/items/") ||
		strings.HasPrefix(slash, "knowledge/packages/") {
		return knowledgeStorageOfficial
	}
	return knowledgeStorageUnknown
}

func validateKnowledgeStorageScope(items []knowledge.Item, scope knowledgeStorageScope) []error {
	var errs []error
	for _, item := range items {
		status := strings.TrimSpace(item.Status)
		switch scope {
		case knowledgeStorageInbox:
			if status != "draft" {
				errs = append(errs, fmt.Errorf("%s: inbox knowledge must use status: draft; publish active knowledge through argos knowledge publish", item.Path))
			}
		case knowledgeStorageOfficial:
			if status == "draft" {
				errs = append(errs, fmt.Errorf("%s: official knowledge must not use status: draft; publish from inbox or set status: active after review", item.Path))
			}
		}
	}
	return errs
}
```

- [ ] **Step 4: Run storage validation tests and full affected CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRunValidate|TestRunKnowledgePublish|TestKnowledgeWritePublishAndFindbackFlow' -count=1
```

Expected: pass.

- [ ] **Step 5: Commit storage validation**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "fix: enforce knowledge storage status contract"
```

---

### Task 5: Publish Contract Guard Coverage

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add publish non-inbox path regression test**

Add this test near the other publish tests:

```go
func TestRunKnowledgePublishRejectsNonInboxPath(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	design := validCLIKnowledgeDesign(draftPath, draftID)
	design.WriteBoundary.Path = "official_review"
	design.Review.OfficialWriteApproved = true
	design.DraftOutput.Status = "active"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", design)
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", strings.Replace(validCLICheckDraftPackage(draftID), "status: draft", "status: active", 1))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "draft must be under knowledge/.inbox/items or knowledge/.inbox/packages") {
		t.Fatalf("expected non-inbox publish error, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run publish tests**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgePublish|TestKnowledgeWritePublishAndFindbackFlow' -count=1
```

Expected: pass. The test may already pass because `publishTarget` rejects
non-inbox paths; this task makes that release contract explicit.

- [ ] **Step 3: Commit publish guard coverage**

Run:

```bash
git add internal/cli/cli_test.go
git commit -m "test: cover publish inbox boundary"
```

---

### Task 6: Documentation, Skill, And Readiness Updates

**Files:**
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`
- Modify: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Update README write contract**

In `README.md`, update the project registry and write sections so they include:

```markdown
### Project Registry Setup

Use project commands instead of hand-editing `knowledge/projects.yaml`:

```bash
argos project add --id mall-api --name "Mall API" --path services/mall-api \
  --tech-domain backend --business-domain account
argos project list --json
```

`project add` validates the selected domains against `knowledge/domains.yaml`.
Edit `knowledge/domains.yaml` only when the allowed domain vocabulary itself
needs to change.

### Storage Contract

Inbox drafts live under `knowledge/.inbox/items/` and
`knowledge/.inbox/packages/` and must use `status: draft`.

Official knowledge lives under `knowledge/items/` and `knowledge/packages/` and
must not use `status: draft`. Published knowledge normally uses
`status: active`; deprecated official knowledge uses `status: deprecated`.

Packages use `KNOWLEDGE.md` as the indexed entrypoint. Supporting files under
`references/`, `examples/`, `checklists/`, `scripts/`, and `assets/` are loaded
only when the package points to them.
```

In the write flow list, make step 1:

```markdown
1. Run `argos project list --json`; if the project is missing, run
   `argos project add`.
```

Add a dogfood note near the dogfood section:

```markdown
Write dogfood is a source-tree internal release-validation harness. It may read
repository `testdata` and is not required for normal installed-binary knowledge
writing.
```

- [ ] **Step 2: Update capture skill**

In `skills/capture-knowledge/SKILL.md`, add project setup guidance before design:

```markdown
Before designing project-scoped knowledge, run `argos project list --json`.
If the target project is missing and the user has provided its id, name, path,
and domains, register it with `argos project add`. Do not hand-edit
`knowledge/projects.yaml` unless the CLI is unavailable.
```

Add storage contract text:

```markdown
Inbox drafts must stay under `knowledge/.inbox/items/` or
`knowledge/.inbox/packages/` with `status: draft`. Official knowledge under
`knowledge/items/` or `knowledge/packages/` must not use `status: draft`.
Use `argos knowledge publish` for the standard inbox-to-official transition.
```

- [ ] **Step 3: Update readiness report**

In `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`:

- Change the write-to-query loop note from requiring manual project setup to
  using `argos project add`.
- Change `Status: conditional` to `Status: ready for write/query core` if the
  report still has non-write release gaps, or `Status: ready` if no blocking
  release gaps remain.
- Move fresh manual `knowledge/projects.yaml` setup out of should-fix.
- Move dogfood cwd dependence out of should-fix and into an explicit decision:
  source-tree internal harness.
- Add a later item for audit/review-history storage.

- [ ] **Step 4: Add docs surface regression test**

In `TestUsageUsesWriteVocabulary`, add expected usage lines:

```go
		"argos project add --id <project> --name <name> --path <path>",
		"argos project list --json",
```

The existing removed-term assertions should continue to guard README/skill-like
surfaces through direct command help. Do not add old write vocabulary.

- [ ] **Step 5: Run docs and CLI surface tests**

Run:

```bash
go test ./internal/cli -run 'TestUsageUsesWriteVocabulary|TestRunProject' -count=1
rg -n "knowledge/\\.inbox/proposals|verify the draft|authoring_packet|proposal_scaffold|source_profile|proposed_shape|overlap_decision|verification_plan|human_review|artifact_state|author inspect|author verify|authoring\\.proposal\\.v2|argos author|dogfood authoring|authoring-golden|argos-authoring-dogfood|\\bpromote\\b|Source Profile|Proposed Shape|Candidate Files|Verification Plan|Human Review Decisions|proposal artifact|canonical JSON proposal" --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**' --glob '!docs/superpowers/reports/**' --glob '!.worktrees/**'
```

Expected:

- Go test passes.
- `rg` exits 1 with no output.

- [ ] **Step 6: Commit docs and readiness**

Run:

```bash
git add README.md skills/capture-knowledge/SKILL.md docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md internal/cli/cli_test.go
git commit -m "docs: finalize write release contract"
```

---

### Task 7: Final Verification

**Files:**
- Inspect all changed files.

- [ ] **Step 1: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: pass for all packages.

- [ ] **Step 2: Run diff checks**

Run:

```bash
git diff --check
```

Expected: no output, exit 0.

- [ ] **Step 3: Review changed files**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
git diff --stat main...HEAD
```

Expected:

- Branch is `write-release-hardening`.
- Only planned files changed.
- Commits match the task boundaries.

- [ ] **Step 4: Summarize readiness**

Prepare a concise final note with:

- Project registration command added.
- Storage status contract enforced.
- Fresh write-to-query flow covered without hand-written project YAML.
- Dogfood write classified as source-tree internal harness.
- Audit/review-history deferred to next module.
- Verification commands and results.
