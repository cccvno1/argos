# Argos Go MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first local-first Argos MVP as a Go CLI and MCP server that validates Markdown knowledge, rebuilds a SQLite index, generates AI adapter files, and exposes progressive knowledge queries.

**Architecture:** Argos is a single Go module with a small CLI in `cmd/argos` and focused internal packages for workspace layout, knowledge parsing, validation, indexing, adapter generation, querying, and MCP serving. Markdown files remain the source of truth; SQLite is a rebuildable cache under `argos/index.db`.

**Tech Stack:** Go 1.23+, `gopkg.in/yaml.v3`, `github.com/bmatcuk/doublestar/v4`, `modernc.org/sqlite`, Go standard `database/sql`, `encoding/json`, `net/rpc/jsonrpc`-style stdio handling implemented directly.

---

## File Structure

- Create `go.mod`: module and dependency declarations.
- Create `cmd/argos/main.go`: CLI entrypoint and command dispatch.
- Create `internal/cli/cli.go`: argument parsing and command routing.
- Create `internal/workspace/workspace.go`: repository paths, init scaffolding, and filesystem helpers.
- Create `internal/knowledge/item.go`: knowledge metadata structs and parser.
- Create `internal/knowledge/validate.go`: schema, registry, scope, and conflict validation.
- Create `internal/registry/registry.go`: loading `domains.yaml`, `projects.yaml`, and `types.yaml`.
- Create `internal/index/store.go`: SQLite schema and index rebuild logic.
- Create `internal/query/query.go`: progressive query service used by CLI and MCP.
- Create `internal/adapters/adapters.go`: generated `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and Cursor rule content.
- Create `internal/mcp/server.go`: minimal MCP-compatible stdio JSON-RPC server.
- Create `testdata/valid-workspace/...`: complete sample workspace used by tests.
- Create tests beside each package using Go's standard `testing` package.

## Task 1: Go Module And CLI Skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/argos/main.go`
- Create: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing CLI dispatch tests**

Create `internal/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsHelpWithoutCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{}, &out, &out)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage: argos <command>") {
		t.Fatalf("expected usage output, got %q", out.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"unknown"}, &out, &out)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(out.String(), "unknown command: unknown") {
		t.Fatalf("expected unknown command error, got %q", out.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/cli
```

Expected: FAIL because `internal/cli` does not exist.

- [ ] **Step 3: Add module and CLI skeleton**

Create `go.mod`:

```go
module argos

go 1.23
```

Create `cmd/argos/main.go`:

```go
package main

import (
	"os"

	"argos/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
```

Create `internal/cli/cli.go`:

```go
package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init", "new", "validate", "index", "install-adapters", "mcp":
		fmt.Fprintf(stderr, "command %q is not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: argos <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  new")
	fmt.Fprintln(w, "  validate")
	fmt.Fprintln(w, "  index")
	fmt.Fprintln(w, "  install-adapters")
	fmt.Fprintln(w, "  mcp")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go mod tidy
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/argos/main.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: scaffold go cli"
```

## Task 2: Workspace Initialization

**Files:**
- Modify: `internal/cli/cli.go`
- Create: `internal/workspace/workspace.go`
- Test: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write failing workspace init tests**

Create `internal/workspace/workspace_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/workspace
```

Expected: FAIL because `Init` is undefined.

- [ ] **Step 3: Implement workspace initialization**

Create `internal/workspace/workspace.go`:

```go
package workspace

import (
	"os"
	"path/filepath"
)

func Init(root string) error {
	dirs := []string{
		"knowledge/.inbox",
		"knowledge/items",
		"argos/generated",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"knowledge/domains.yaml":  defaultDomainsYAML,
		"knowledge/projects.yaml": defaultProjectsYAML,
		"knowledge/types.yaml":    defaultTypesYAML,
	}
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}

	return nil
}

const defaultDomainsYAML = `tech_domains:
  - backend
  - frontend
  - database
  - devops
  - testing
  - security
  - ai-engineering
business_domains:
  - account
  - order
  - payment
  - risk
  - content
  - analytics
`

const defaultProjectsYAML = `projects: []
`

const defaultTypesYAML = `types:
  - rule
  - guide
  - runbook
  - decision
  - lesson
  - reference
`
```

- [ ] **Step 4: Wire `argos init`**

Modify `internal/cli/cli.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"

	"argos/internal/workspace"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		if err := workspace.Init(root); err != nil {
			fmt.Fprintf(stderr, "init workspace: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "initialized Argos workspace")
		return 0
	case "new", "validate", "index", "install-adapters", "mcp":
		fmt.Fprintf(stderr, "command %q is not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: argos <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  new")
	fmt.Fprintln(w, "  validate")
	fmt.Fprintln(w, "  index")
	fmt.Fprintln(w, "  install-adapters")
	fmt.Fprintln(w, "  mcp")
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/workspace ./internal/cli
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: initialize argos workspace"
```

## Task 3: Knowledge Parsing And Validation

Note: `gopkg.in/yaml.v3` is introduced in this task when it is first imported; run `go mod tidy` after adding the parser code.

**Files:**
- Create: `internal/knowledge/item.go`
- Create: `internal/knowledge/validate.go`
- Create: `internal/registry/registry.go`
- Test: `internal/knowledge/item_test.go`
- Test: `internal/knowledge/validate_test.go`

- [ ] **Step 1: Write failing parser test**

Create `internal/knowledge/item_test.go`:

```go
package knowledge

import "testing"

func TestParseItemReadsFrontmatterAndBody(t *testing.T) {
	input := []byte(`---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  languages: [go]
  frameworks: [gin]
  files: ["internal/auth/**"]
updated_at: 2026-04-29
---

# Body

Use short-lived access tokens.
`)

	item, err := ParseItem("knowledge/items/backend/auth.md", input)
	if err != nil {
		t.Fatalf("ParseItem returned error: %v", err)
	}
	if item.ID != "backend.auth.jwt-refresh-token.v1" {
		t.Fatalf("unexpected ID: %s", item.ID)
	}
	if item.AppliesTo.Files[0] != "internal/auth/**" {
		t.Fatalf("unexpected file scope: %#v", item.AppliesTo.Files)
	}
	if item.Body != "# Body\n\nUse short-lived access tokens.\n" {
		t.Fatalf("unexpected body: %q", item.Body)
	}
}
```

- [ ] **Step 2: Write failing validation test**

Create `internal/knowledge/validate_test.go`:

```go
package knowledge

import (
	"strings"
	"testing"

	"argos/internal/registry"
)

func TestValidateItemRejectsUnknownDomain(t *testing.T) {
	item := Item{
		Path:            "knowledge/items/x.md",
		ID:              "backend.auth.jwt-refresh-token.v1",
		Title:           "JWT refresh token handling convention",
		Type:            "rule",
		TechDomains:     []string{"backend", "unknown"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}
	reg := registry.Registry{
		TechDomains:     []string{"backend", "security"},
		BusinessDomains: []string{"account"},
		Projects:        []registry.Project{{ID: "mall-api"}},
		Types:           []string{"rule", "guide", "runbook", "decision", "lesson", "reference"},
	}

	errs := ValidateItem(item, reg)
	if len(errs) == 0 {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(errs[0].Error(), "unknown tech domain: unknown") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/knowledge
```

Expected: FAIL because parser and validator do not exist.

- [ ] **Step 4: Implement registry types**

Create `internal/registry/registry.go`:

```go
package registry

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	TechDomains     []string  `yaml:"tech_domains"`
	BusinessDomains []string  `yaml:"business_domains"`
	Projects        []Project `yaml:"projects"`
	Types           []string  `yaml:"types"`
}

type Project struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Path            string   `yaml:"path"`
	TechDomains     []string `yaml:"tech_domains"`
	BusinessDomains []string `yaml:"business_domains"`
}

func Load(root string) (Registry, error) {
	var reg Registry

	if err := readYAML(filepath.Join(root, "knowledge/domains.yaml"), &reg); err != nil {
		return Registry{}, err
	}
	var projects struct {
		Projects []Project `yaml:"projects"`
	}
	if err := readYAML(filepath.Join(root, "knowledge/projects.yaml"), &projects); err != nil {
		return Registry{}, err
	}
	reg.Projects = projects.Projects

	var types struct {
		Types []string `yaml:"types"`
	}
	if err := readYAML(filepath.Join(root, "knowledge/types.yaml"), &types); err != nil {
		return Registry{}, err
	}
	reg.Types = types.Types

	return reg, nil
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}
```

- [ ] **Step 5: Implement parser**

Create `internal/knowledge/item.go`:

```go
package knowledge

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Item struct {
	Path            string   `yaml:"-"`
	ID              string   `yaml:"id"`
	Title           string   `yaml:"title"`
	Type            string   `yaml:"type"`
	TechDomains     []string `yaml:"tech_domains"`
	BusinessDomains []string `yaml:"business_domains"`
	Projects        []string `yaml:"projects"`
	Status          string   `yaml:"status"`
	Priority        string   `yaml:"priority"`
	AppliesTo       Scope    `yaml:"applies_to"`
	UpdatedAt       string   `yaml:"updated_at"`
	Body            string   `yaml:"-"`
}

type Scope struct {
	Languages  []string `yaml:"languages"`
	Frameworks []string `yaml:"frameworks"`
	Files      []string `yaml:"files"`
	Services   []string `yaml:"services"`
	Envs       []string `yaml:"envs"`
}

func ParseItem(path string, data []byte) (Item, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return Item{}, fmt.Errorf("%s: missing YAML frontmatter", path)
	}
	rest := data[len("---\n"):]
	idx := bytes.Index(rest, []byte("\n---\n"))
	if idx < 0 {
		return Item{}, fmt.Errorf("%s: frontmatter closing delimiter not found", path)
	}

	meta := rest[:idx]
	body := rest[idx+len("\n---\n"):]

	var item Item
	if err := yaml.Unmarshal(meta, &item); err != nil {
		return Item{}, fmt.Errorf("%s: parse frontmatter: %w", path, err)
	}
	item.Path = path
	item.Body = strings.TrimLeft(string(body), "\n")
	return item, nil
}
```

- [ ] **Step 6: Implement validation**

Create `internal/knowledge/validate.go`:

```go
package knowledge

import (
	"fmt"
	"strings"

	"argos/internal/registry"
)

func ValidateItem(item Item, reg registry.Registry) []error {
	var errs []error
	require := func(name string, value string) {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s: missing %s", item.Path, name))
		}
	}

	require("id", item.ID)
	require("title", item.Title)
	require("type", item.Type)
	require("status", item.Status)
	require("priority", item.Priority)
	require("updated_at", item.UpdatedAt)

	if !contains(reg.Types, item.Type) {
		errs = append(errs, fmt.Errorf("%s: unknown type: %s", item.Path, item.Type))
	}
	if !contains([]string{"draft", "active", "deprecated"}, item.Status) {
		errs = append(errs, fmt.Errorf("%s: invalid status: %s", item.Path, item.Status))
	}
	if !contains([]string{"must", "should", "may"}, item.Priority) {
		errs = append(errs, fmt.Errorf("%s: invalid priority: %s", item.Path, item.Priority))
	}
	for _, domain := range item.TechDomains {
		if !contains(reg.TechDomains, domain) {
			errs = append(errs, fmt.Errorf("%s: unknown tech domain: %s", item.Path, domain))
		}
	}
	for _, domain := range item.BusinessDomains {
		if !contains(reg.BusinessDomains, domain) {
			errs = append(errs, fmt.Errorf("%s: unknown business domain: %s", item.Path, domain))
		}
	}
	for _, project := range item.Projects {
		if !containsProject(reg.Projects, project) {
			errs = append(errs, fmt.Errorf("%s: unknown project: %s", item.Path, project))
		}
	}
	if strings.TrimSpace(item.Body) == "" {
		errs = append(errs, fmt.Errorf("%s: empty body", item.Path))
	}
	return errs
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsProject(projects []registry.Project, id string) bool {
	for _, project := range projects {
		if project.ID == id {
			return true
		}
	}
	return false
}
```

- [ ] **Step 7: Run tests**

Run:

```bash
go test ./internal/knowledge ./internal/registry
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/knowledge internal/registry
git commit -m "feat: parse and validate knowledge items"
```

## Task 4: Validate Command

**Files:**
- Modify: `internal/cli/cli.go`
- Create: `internal/knowledge/load.go`
- Test: `internal/knowledge/load_test.go`

- [ ] **Step 1: Write failing loader test**

Create `internal/knowledge/load_test.go`:

```go
package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadItemsReadsMarkdownFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "knowledge/items/backend")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "auth.md"), []byte(`---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := LoadItems(root)
	if err != nil {
		t.Fatalf("LoadItems returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "backend.auth.jwt-refresh-token.v1" {
		t.Fatalf("unexpected item ID: %s", items[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/knowledge
```

Expected: FAIL because `LoadItems` is undefined.

- [ ] **Step 3: Implement item loading**

Create `internal/knowledge/load.go`:

```go
package knowledge

import (
	"os"
	"path/filepath"
)

func LoadItems(root string) ([]Item, error) {
	base := filepath.Join(root, "knowledge/items")
	var items []Item
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		item, err := ParseItem(rel, data)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}
```

- [ ] **Step 4: Wire `argos validate`**

Modify `internal/cli/cli.go` so the `validate` case loads registry and items:

```go
	case "validate":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		reg, err := registry.Load(root)
		if err != nil {
			fmt.Fprintf(stderr, "load registry: %v\n", err)
			return 1
		}
		items, err := knowledge.LoadItems(root)
		if err != nil {
			fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
			return 1
		}
		var count int
		for _, item := range items {
			for _, validationErr := range knowledge.ValidateItem(item, reg) {
				fmt.Fprintln(stderr, validationErr.Error())
				count++
			}
		}
		if count > 0 {
			fmt.Fprintf(stderr, "validation failed with %d error(s)\n", count)
			return 1
		}
		fmt.Fprintf(stdout, "validated %d knowledge item(s)\n", len(items))
		return 0
```

Add imports:

```go
	"argos/internal/knowledge"
	"argos/internal/registry"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/knowledge/load.go internal/knowledge/load_test.go
git commit -m "feat: validate knowledge workspace"
```

## Task 5: SQLite Index Rebuild

Note: `modernc.org/sqlite` is introduced in this task when it is first imported; run `go mod tidy` after adding the index code.

**Files:**
- Create: `internal/index/store.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/index/store_test.go`

- [ ] **Step 1: Write failing index test**

Create `internal/index/store_test.go`:

```go
package index

import (
	"path/filepath"
	"testing"

	"argos/internal/knowledge"
)

func TestRebuildStoresItems(t *testing.T) {
	root := t.TempDir()
	items := []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "backend.auth.jwt-refresh-token.v1",
		Title:           "JWT refresh token handling convention",
		Type:            "rule",
		TechDomains:     []string{"backend", "security"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Use short-lived access tokens.",
	}}

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("backend.auth.jwt-refresh-token.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "JWT refresh token handling convention" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/index
```

Expected: FAIL because `Rebuild` is undefined.

- [ ] **Step 3: Implement SQLite store**

Create `internal/index/store.go`:

```go
package index

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"argos/internal/knowledge"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Rebuild(dbPath string, items []knowledge.Item) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dbPath)

	store, err := Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.createSchema(); err != nil {
		return err
	}
	for _, item := range items {
		if err := store.InsertItem(item); err != nil {
			return err
		}
	}
	return nil
}

func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS knowledge_items (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  title TEXT NOT NULL,
  type TEXT NOT NULL,
  tech_domains TEXT NOT NULL,
  business_domains TEXT NOT NULL,
  projects TEXT NOT NULL,
  status TEXT NOT NULL,
  priority TEXT NOT NULL,
  scope TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  summary TEXT NOT NULL,
  body TEXT NOT NULL
);
`)
	return err
}

func (s *Store) InsertItem(item knowledge.Item) error {
	tech, _ := json.Marshal(item.TechDomains)
	biz, _ := json.Marshal(item.BusinessDomains)
	projects, _ := json.Marshal(item.Projects)
	scope, _ := json.Marshal(item.AppliesTo)
	summary := firstNonEmptyLine(item.Body)

	_, err := s.db.Exec(`
INSERT INTO knowledge_items
(id, path, title, type, tech_domains, business_domains, projects, status, priority, scope, updated_at, summary, body)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Path, item.Title, item.Type, string(tech), string(biz), string(projects),
		item.Status, item.Priority, string(scope), item.UpdatedAt, summary, item.Body)
	return err
}

func (s *Store) GetItem(id string) (knowledge.Item, error) {
	var item knowledge.Item
	var tech, biz, projects, scope string
	err := s.db.QueryRow(`
SELECT id, path, title, type, tech_domains, business_domains, projects, status, priority, scope, updated_at, body
FROM knowledge_items WHERE id = ?`, id).Scan(
		&item.ID, &item.Path, &item.Title, &item.Type, &tech, &biz, &projects,
		&item.Status, &item.Priority, &scope, &item.UpdatedAt, &item.Body)
	if err != nil {
		return knowledge.Item{}, err
	}
	_ = json.Unmarshal([]byte(tech), &item.TechDomains)
	_ = json.Unmarshal([]byte(biz), &item.BusinessDomains)
	_ = json.Unmarshal([]byte(projects), &item.Projects)
	_ = json.Unmarshal([]byte(scope), &item.AppliesTo)
	return item, nil
}

func firstNonEmptyLine(body string) string {
	start := 0
	for start < len(body) {
		end := start
		for end < len(body) && body[end] != '\n' {
			end++
		}
		line := body[start:end]
		if line != "" && line[0] != '#' {
			return line
		}
		start = end + 1
	}
	return ""
}
```

- [ ] **Step 4: Wire `argos index`**

Modify `internal/cli/cli.go` and add an `index` command case:

```go
	case "index":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		items, err := knowledge.LoadItems(root)
		if err != nil {
			fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
			return 1
		}
		if err := index.Rebuild(filepath.Join(root, "argos/index.db"), items); err != nil {
			fmt.Fprintf(stderr, "rebuild index: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "indexed %d knowledge item(s)\n", len(items))
		return 0
```

Add imports:

```go
	"path/filepath"

	"argos/internal/index"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/index/store.go internal/index/store_test.go
git commit -m "feat: rebuild local knowledge index"
```

## Task 6: Static Adapter Generation

**Files:**
- Create: `internal/adapters/adapters.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/adapters/adapters_test.go`

- [ ] **Step 1: Write failing adapter test**

Create `internal/adapters/adapters_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/adapters
```

Expected: FAIL because package does not exist.

- [ ] **Step 3: Implement adapter rendering**

Create `internal/adapters/adapters.go`:

```go
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
		base := filepath.Join(root, "argos/generated", project.ID)
		if err := os.MkdirAll(filepath.Join(base, "cursor-rules"), 0o755); err != nil {
			return err
		}
		files := map[string]string{
			"AGENTS.md":                RenderAGENTS(project),
			"CLAUDE.md":                RenderClaude(project),
			"GEMINI.md":                RenderGemini(project),
			"cursor-rules/argos.mdc":   RenderCursorRule(project),
		}
		for rel, body := range files {
			if err := os.WriteFile(filepath.Join(base, rel), []byte(body), 0o644); err != nil {
				return err
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

- project: %s
- tech domains: %s
- business domains: %s

## Argos Protocol

Before substantial design, implementation, debugging, review, or operations work:

1. Call `+"`argos_context`"+` with project, phase, task, and files.
2. Follow the recommended next calls.
3. Call `+"`argos_requirements`"+` during design and planning.
4. Call `+"`argos_standards`"+` before implementation or review.
5. Call `+"`argos_risks`"+` for risky changes, debugging, and pre-deployment checks.
6. Call `+"`argos_operations`"+` for deployment, rollback, database, and on-call tasks.
7. Load full knowledge items only when routed to a specific ID.
8. Cite knowledge IDs used in final responses.
`, adapter, project.ID, strings.Join(project.TechDomains, ", "), strings.Join(project.BusinessDomains, ", "))
}
```

- [ ] **Step 4: Wire `argos install-adapters`**

Modify `internal/cli/cli.go` and add:

```go
	case "install-adapters":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get working directory: %v\n", err)
			return 1
		}
		reg, err := registry.Load(root)
		if err != nil {
			fmt.Fprintf(stderr, "load registry: %v\n", err)
			return 1
		}
		if err := adapters.Install(root, reg.Projects); err != nil {
			fmt.Fprintf(stderr, "install adapters: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "installed adapters for %d project(s)\n", len(reg.Projects))
		return 0
```

Add import:

```go
	"argos/internal/adapters"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/adapters internal/cli/cli.go
git commit -m "feat: generate ai adapter files"
```

## Task 7: Progressive Query Service

Note: `github.com/bmatcuk/doublestar/v4` is introduced in this task when it is first imported; run `go mod tidy` after adding the query code.

**Files:**
- Create: `internal/query/query.go`
- Modify: `internal/index/store.go`
- Test: `internal/query/query_test.go`

- [ ] **Step 1: Write failing query test**

Create `internal/query/query_test.go`:

```go
package query

import (
	"path/filepath"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

func TestStandardsReturnsActiveRulesWithMatchReason(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].WhyMatched == "" {
		t.Fatalf("expected match reason")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/query
```

Expected: FAIL because package does not exist.

- [ ] **Step 3: Add index list query**

Modify `internal/index/store.go`:

```go
func (s *Store) ListItems() ([]knowledge.Item, error) {
	rows, err := s.db.Query(`
SELECT id, path, title, type, tech_domains, business_domains, projects, status, priority, scope, updated_at, body
FROM knowledge_items ORDER BY priority, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []knowledge.Item
	for rows.Next() {
		var item knowledge.Item
		var tech, biz, projects, scope string
		if err := rows.Scan(&item.ID, &item.Path, &item.Title, &item.Type, &tech, &biz, &projects, &item.Status, &item.Priority, &scope, &item.UpdatedAt, &item.Body); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tech), &item.TechDomains)
		_ = json.Unmarshal([]byte(biz), &item.BusinessDomains)
		_ = json.Unmarshal([]byte(projects), &item.Projects)
		_ = json.Unmarshal([]byte(scope), &item.AppliesTo)
		items = append(items, item)
	}
	return items, rows.Err()
}
```

- [ ] **Step 4: Implement query service**

Create `internal/query/query.go`:

```go
package query

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"argos/internal/index"
	"argos/internal/knowledge"
)

type Service struct {
	store *index.Store
}

func New(store *index.Store) *Service {
	return &Service{store: store}
}

type StandardsRequest struct {
	Project  string   `json:"project"`
	TaskType string   `json:"task_type,omitempty"`
	Files    []string `json:"files,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type Response struct {
	Items                []ResultItem      `json:"items"`
	Conflicts            []string          `json:"conflicts"`
	RecommendedNextCalls []RecommendedCall `json:"recommended_next_calls"`
}

type ResultItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	WhyMatched string `json:"why_matched"`
}

type RecommendedCall struct {
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
}

func (s *Service) Standards(req StandardsRequest) (Response, error) {
	items, err := s.store.ListItems()
	if err != nil {
		return Response{}, err
	}
	limit := req.Limit
	if limit <= 0 || limit > 5 {
		limit = 5
	}

	var out Response
	for _, item := range items {
		if len(out.Items) >= limit {
			break
		}
		if item.Type != "rule" || item.Status == "deprecated" {
			continue
		}
		reason := matchReason(item, req.Project, req.Files)
		if reason == "" {
			continue
		}
		out.Items = append(out.Items, ResultItem{
			ID:         item.ID,
			Type:       item.Type,
			Title:      item.Title,
			Summary:    firstSentence(item.Body),
			Priority:   item.Priority,
			Status:     item.Status,
			WhyMatched: reason,
		})
	}
	return out, nil
}

func matchReason(item knowledge.Item, project string, files []string) string {
	if !contains(item.Projects, project) {
		return ""
	}
	for _, file := range files {
		for _, pattern := range item.AppliesTo.Files {
			ok, _ := doublestar.PathMatch(pattern, file)
			if ok {
				return "project matches and file matches " + pattern
			}
		}
	}
	if len(item.AppliesTo.Files) == 0 {
		return "project matches"
	}
	return ""
}

func firstSentence(body string) string {
	body = strings.TrimSpace(body)
	if idx := strings.Index(body, "\n"); idx >= 0 {
		return strings.TrimSpace(body[:idx])
	}
	return body
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/index ./internal/query
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/index/store.go internal/query
git commit -m "feat: add progressive query service"
```

## Task 8: Workflow Contract CLI JSON

**Files:**
- Modify: `internal/query/query.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/query/query_test.go`

- [ ] **Step 1: Add failing context query test**

Append to `internal/query/query_test.go`:

```go
func TestContextRecommendsNextCalls(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "add refresh token endpoint",
		Files:   []string{"internal/auth/session.go"},
	})

	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", result.Project)
	}
	if len(result.RecommendedNextCalls) == 0 {
		t.Fatal("expected recommended next calls")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/query
```

Expected: FAIL because `Context` is undefined.

- [ ] **Step 3: Implement context contract**

Add to `internal/query/query.go`:

```go
type ContextRequest struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Files   []string `json:"files,omitempty"`
}

type ContextResponse struct {
	Project              string            `json:"project"`
	Phase                string            `json:"phase"`
	RecommendedNextCalls []RecommendedCall `json:"recommended_next_calls"`
}

func (s *Service) Context(req ContextRequest) ContextResponse {
	calls := []RecommendedCall{{Tool: "argos_requirements", Reason: "workflow start should collect constraints"}}
	switch req.Phase {
	case "implementation", "review":
		calls = append(calls, RecommendedCall{Tool: "argos_standards", Reason: "implementation and review require active rules"})
	case "debugging":
		calls = append(calls, RecommendedCall{Tool: "argos_risks", Reason: "debugging should check lessons and incident history"})
	case "operations", "deployment":
		calls = append(calls, RecommendedCall{Tool: "argos_operations", Reason: "operations should use runbooks"})
	default:
		calls = append(calls, RecommendedCall{Tool: "argos_standards", Reason: "standards are useful before code changes"})
	}
	return ContextResponse{
		Project:              req.Project,
		Phase:                req.Phase,
		RecommendedNextCalls: calls,
	}
}
```

- [ ] **Step 4: Add `argos context --json` command**

Modify `internal/cli/cli.go` to add `context` to usage and command switch. Implement this command:

```go
	case "context":
		fs := flag.NewFlagSet("context", flag.ContinueOnError)
		fs.SetOutput(stderr)
		project := fs.String("project", "", "project id")
		phase := fs.String("phase", "planning", "workflow phase")
		task := fs.String("task", "", "task description")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		service := query.New(nil)
		result := service.Context(query.ContextRequest{Project: *project, Phase: *phase, Task: *task})
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode context response: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
```

Add imports:

```go
	"encoding/json"
	"flag"

	"argos/internal/query"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/query/query.go internal/query/query_test.go
git commit -m "feat: expose workflow context contract"
```

## Task 9: Minimal MCP Server

**Files:**
- Create: `internal/mcp/server.go`
- Modify: `internal/cli/cli.go`
- Test: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing MCP request test**

Create `internal/mcp/server_test.go`:

```go
package mcp

import (
	"bytes"
	"strings"
	"testing"

	"argos/internal/query"
)

func TestServerHandlesToolsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	if !strings.Contains(out.String(), "argos_context") {
		t.Fatalf("expected argos_context tool in response: %s", out.String())
	}
}

func TestServerHandlesResourcesAndPromptsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine resources/list returned error: %v", err)
	}
	if !strings.Contains(out.String(), "argos://project/{project}/brief") {
		t.Fatalf("expected project brief resource in response: %s", out.String())
	}

	out.Reset()
	err = server.HandleLine([]byte(`{"jsonrpc":"2.0","id":3,"method":"prompts/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine prompts/list returned error: %v", err)
	}
	if !strings.Contains(out.String(), "prepare_feature_work") {
		t.Fatalf("expected prepare_feature_work prompt in response: %s", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/mcp
```

Expected: FAIL because package does not exist.

- [ ] **Step 3: Implement minimal JSON-RPC MCP server**

Create `internal/mcp/server.go`:

```go
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"argos/internal/query"
)

type Server struct {
	query *query.Service
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func NewServer(service *query.Service) *Server {
	return &Server{query: service}
}

func (s *Server) Serve(stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		if err := s.HandleLine(scanner.Bytes(), stdout); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) HandleLine(line []byte, stdout io.Writer) error {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return err
	}

	var result any
	switch req.Method {
	case "tools/list":
		result = map[string]any{"tools": tools()}
	case "resources/list":
		result = map[string]any{"resources": resources()}
	case "prompts/list":
		result = map[string]any{"prompts": prompts()}
	default:
		result = map[string]any{"message": "method not implemented", "method": req.Method}
	}

	data, err := json.Marshal(response{JSONRPC: "2.0", ID: req.ID, Result: result})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(data))
	return err
}

func tools() []map[string]any {
	return []map[string]any{
		{"name": "argos_context", "description": "Start progressive Argos knowledge disclosure for a workflow."},
		{"name": "argos_requirements", "description": "Return design and planning constraints."},
		{"name": "argos_standards", "description": "Return active implementation and review standards."},
		{"name": "argos_risks", "description": "Return lessons, incident notes, and risk warnings."},
		{"name": "argos_operations", "description": "Return runbooks and operational notes."},
		{"name": "get_knowledge_item", "description": "Return full content for a routed knowledge ID."},
		{"name": "cite_knowledge", "description": "Return citation metadata for knowledge IDs."},
	}
}

func resources() []map[string]any {
	return []map[string]any{
		{"uri": "argos://project/{project}/brief", "name": "Project brief", "mimeType": "application/json"},
		{"uri": "argos://project/{project}/rules", "name": "Project rules", "mimeType": "application/json"},
		{"uri": "argos://domain/{domain}/map", "name": "Domain map", "mimeType": "application/json"},
		{"uri": "argos://knowledge/{id}", "name": "Knowledge item", "mimeType": "text/markdown"},
	}
}

func prompts() []map[string]any {
	return []map[string]any{
		{"name": "prepare_feature_work", "description": "Prepare context for feature implementation."},
		{"name": "debug_with_lessons", "description": "Debug using relevant lessons and risks."},
		{"name": "review_with_standards", "description": "Review code against active standards."},
		{"name": "prepare_deployment", "description": "Prepare deployment or rollback using runbooks."},
	}
}
```

- [ ] **Step 4: Wire `argos mcp`**

Modify `internal/cli/cli.go`:

```go
	case "mcp":
		server := mcp.NewServer(query.New(nil))
		if err := server.Serve(os.Stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "mcp server: %v\n", err)
			return 1
		}
		return 0
```

Add import:

```go
	"argos/internal/mcp"
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mcp internal/cli/cli.go
git commit -m "feat: expose minimal mcp server"
```

## Task 10: End-To-End Fixture And Documentation

**Files:**
- Create: `testdata/valid-workspace/knowledge/domains.yaml`
- Create: `testdata/valid-workspace/knowledge/projects.yaml`
- Create: `testdata/valid-workspace/knowledge/types.yaml`
- Create: `testdata/valid-workspace/knowledge/items/backend/auth.md`
- Create: `README.md`
- Test: `cmd/argos/e2e_test.go`

- [ ] **Step 1: Create fixture files**

Create `testdata/valid-workspace/knowledge/domains.yaml`:

```yaml
tech_domains:
  - backend
  - security
business_domains:
  - account
```

Create `testdata/valid-workspace/knowledge/projects.yaml`:

```yaml
projects:
  - id: mall-api
    name: Mall API
    path: .
    tech_domains: [backend, security]
    business_domains: [account]
```

Create `testdata/valid-workspace/knowledge/types.yaml`:

```yaml
types:
  - rule
  - guide
  - runbook
  - decision
  - lesson
  - reference
```

Create `testdata/valid-workspace/knowledge/items/backend/auth.md`:

```md
---
id: rule:backend.auth.v1
title: Auth middleware rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  languages: [go]
  frameworks: [gin]
  files: ["internal/auth/**"]
updated_at: 2026-04-29
---

Require explicit auth middleware for account endpoints.
```

- [ ] **Step 2: Write end-to-end CLI test**

Create `cmd/argos/e2e_test.go`:

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIValidateIndexAndAdapters(t *testing.T) {
	root := t.TempDir()
	copyDir(t, filepath.Join("..", "..", "testdata", "valid-workspace"), root)

	bin := filepath.Join(root, "argos-test")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	for _, args := range [][]string{{"validate"}, {"index"}, {"install-adapters"}} {
		cmd := exec.Command(bin, args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("argos %v failed: %v\n%s", args, err, out)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "argos/index.db")); err != nil {
		t.Fatalf("expected index.db: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "argos/generated/mall-api/AGENTS.md")); err != nil {
		t.Fatalf("expected generated AGENTS.md: %v", err)
	}
}

func copyDir(t *testing.T, src string, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Add README**

Create `README.md`:

```md
# Argos

Argos is a local-first knowledge substrate for AI development workflows.

## MVP

- Markdown knowledge items with YAML frontmatter
- Controlled registries for domains, projects, and types
- Rebuildable local SQLite index
- Static adapters for AI coding tools
- MCP server for progressive knowledge disclosure
- Workflow contracts for agent and skill systems

## Commands

```bash
argos init
argos validate
argos index
argos install-adapters
argos mcp
```
```

- [ ] **Step 4: Run full verification**

Run:

```bash
go test ./...
go build ./cmd/argos
```

Expected: PASS and build succeeds.

- [ ] **Step 5: Commit**

```bash
git add README.md cmd/argos/e2e_test.go testdata
git commit -m "test: add argos mvp fixture"
```

## Self-Review

Spec coverage:

- Markdown source with YAML frontmatter: Tasks 3 and 10.
- Controlled registries: Tasks 2, 3, and 10.
- CLI commands: Tasks 1, 2, 4, 5, 6, 8, and 9.
- SQLite rebuildable index: Task 5.
- Static adapters: Task 6.
- MCP tools, resources, and prompts discovery: Task 9.
- Workflow integration contract: Task 8.
- Progressive disclosure defaults: Tasks 6, 7, 8, and 9.
- Inbox review flow: Task 2 creates `.inbox`; full promotion workflow is represented by filesystem structure and validation in MVP.

Risk notes:

- The MCP server in this MVP starts with `tools/list`; tool invocation bodies should be added immediately after this plan if a connected MCP client needs live query calls during manual testing.
- Embeddings are excluded from this MVP because the approved design makes them optional.
- Web UI, cloud database, enterprise permissions, and external document sync are outside this MVP by design.
