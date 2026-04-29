# Argos Knowledge Packages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add first-class knowledge packages with `KNOWLEDGE.md` entrypoints, path-level validation, safe promotion, and package indexing.

**Architecture:** Extend the existing Markdown knowledge model instead of adding a parallel package database. Package `KNOWLEDGE.md` files parse into the existing `knowledge.Item` type with `type: package`, while package-specific validation lives in `internal/knowledge`. CLI validation gains explicit scopes so official knowledge, inbox candidates, and a single path can be checked independently.

**Tech Stack:** Go 1.23, existing `internal/knowledge`, `internal/cli`, `internal/index`, `internal/query`, `internal/mcp`, YAML frontmatter via `gopkg.in/yaml.v3`, local filesystem operations.

---

## File Structure

- Modify `internal/knowledge/item.go`: add free-form `Tags []string` to frontmatter parsing.
- Modify `internal/knowledge/load.go`: add loaders for official items, official packages, inbox items, inbox packages, and a single path.
- Create `internal/knowledge/package.go`: package path safety and package section validation.
- Modify `internal/knowledge/validate.go`: validate package entrypoint sections and duplicate IDs across mixed items/packages.
- Modify `internal/knowledge/*_test.go`: focused tests for package parsing, loading, validation, path safety, and duplicate IDs.
- Modify `internal/workspace/workspace.go`: include `package` in default `knowledge/types.yaml`.
- Modify `internal/workspace/workspace_test.go`: assert default types include `package`.
- Modify `internal/cli/cli.go`: add `validate --inbox`, `validate --path`, and `promote --path`; update `index` to index official packages.
- Modify `internal/cli/cli_test.go`: CLI tests for validation scopes, path validation, promote behavior, and package indexing.
- Modify `internal/mcp/server_test.go`: assert package entrypoints can be fetched with `get_knowledge_item`.
- Modify `README.md`: document package layout, validate scopes, and promote flow.

## Task 1: Package Metadata And Loading

**Files:**
- Modify: `internal/knowledge/item.go`
- Modify: `internal/knowledge/load.go`
- Test: `internal/knowledge/item_test.go`
- Test: `internal/knowledge/load_test.go`

- [ ] **Step 1: Write failing parser test for tags**

Add this test to `internal/knowledge/item_test.go`:

```go
func TestParseItemReadsPackageTags(t *testing.T) {
	input := []byte(`---
id: package:backend.redis.best-practices.v1
title: Redis Best Practices
type: package
tech_domains: [backend, database]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-04-29
---
# Redis Best Practices
`)

	item, err := ParseItem("knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", input)
	if err != nil {
		t.Fatalf("ParseItem returned error: %v", err)
	}
	if item.Type != "package" {
		t.Fatalf("expected package type, got %q", item.Type)
	}
	if strings.Join(item.Tags, ",") != "redis,cache" {
		t.Fatalf("unexpected tags: %#v", item.Tags)
	}
}
```

- [ ] **Step 2: Run parser test to verify it fails**

Run:

```bash
go test ./internal/knowledge -run TestParseItemReadsPackageTags -count=1
```

Expected: FAIL because `knowledge.Item` has no `Tags` field.

- [ ] **Step 3: Add tags to the item model**

In `internal/knowledge/item.go`, add:

```go
Tags []string `yaml:"tags"`
```

to `type Item struct`.

- [ ] **Step 4: Run parser test to verify it passes**

Run:

```bash
gofmt -w internal/knowledge/item.go internal/knowledge/item_test.go
go test ./internal/knowledge -run TestParseItemReadsPackageTags -count=1
```

Expected: PASS.

- [ ] **Step 5: Write failing loader tests for official packages and inbox packages**

Add these tests to `internal/knowledge/load_test.go`:

```go
func TestLoadOfficialKnowledgeReadsItemsAndPackages(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/items/backend/auth.md", validKnowledgeItem("rule:backend.auth.v1", "rule"))
	writeKnowledgeFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validPackageItem("package:backend.redis.best-practices.v1"))
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/draft/KNOWLEDGE.md", validPackageItem("package:backend.draft.v1"))

	items, err := LoadOfficial(root)
	if err != nil {
		t.Fatalf("LoadOfficial returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected official item and package, got %d: %#v", len(items), items)
	}
	assertLoadedID(t, items, "rule:backend.auth.v1")
	assertLoadedID(t, items, "package:backend.redis.best-practices.v1")
}

func TestLoadInboxKnowledgeReadsInboxItemsAndPackages(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/.inbox/items/backend/auth.md", validKnowledgeItem("rule:backend.auth.draft.v1", "rule"))
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/redis/draft/KNOWLEDGE.md", validPackageItem("package:backend.redis.draft.v1"))
	writeKnowledgeFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validPackageItem("package:backend.redis.best-practices.v1"))

	items, err := LoadInbox(root)
	if err != nil {
		t.Fatalf("LoadInbox returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected inbox item and package, got %d: %#v", len(items), items)
	}
	assertLoadedID(t, items, "rule:backend.auth.draft.v1")
	assertLoadedID(t, items, "package:backend.redis.draft.v1")
}

func TestLoadPathReadsSinglePackageDirectory(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/redis/draft/KNOWLEDGE.md", validPackageItem("package:backend.redis.draft.v1"))

	items, err := LoadPath(root, "knowledge/.inbox/packages/backend/redis/draft")
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "package:backend.redis.draft.v1" {
		t.Fatalf("unexpected loaded items: %#v", items)
	}
}

func assertLoadedID(t *testing.T, items []Item, id string) {
	t.Helper()
	for _, item := range items {
		if item.ID == id {
			return
		}
	}
	t.Fatalf("expected loaded id %s in %#v", id, items)
}

func validPackageItem(id string) string {
	return `---
id: ` + id + `
title: Redis Best Practices
type: package
tech_domains: [backend]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis]
updated_at: 2026-04-29
---
## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read the short rules first.

## Load On Demand

- references/key-design.md when designing keys.
`
}
```

If the file already has helpers with overlapping names, reuse the existing helper style and keep the exact assertions.

- [ ] **Step 6: Run loader tests to verify they fail**

Run:

```bash
go test ./internal/knowledge -run 'TestLoadOfficialKnowledgeReadsItemsAndPackages|TestLoadInboxKnowledgeReadsInboxItemsAndPackages|TestLoadPathReadsSinglePackageDirectory' -count=1
```

Expected: FAIL because `LoadOfficial`, `LoadInbox`, and `LoadPath` do not exist.

- [ ] **Step 7: Implement package-aware loaders**

In `internal/knowledge/load.go`, replace `LoadItems` with wrappers around a shared loader:

```go
func LoadItems(root string) ([]Item, error) {
	return LoadOfficial(root)
}

func LoadOfficial(root string) ([]Item, error) {
	return loadFromRoots(root, []string{
		filepath.Join("knowledge", "items"),
		filepath.Join("knowledge", "packages"),
	})
}

func LoadInbox(root string) ([]Item, error) {
	return loadFromRoots(root, []string{
		filepath.Join("knowledge", ".inbox", "items"),
		filepath.Join("knowledge", ".inbox", "packages"),
	})
}

func LoadPath(root string, relPath string) ([]Item, error) {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return nil, err
	}
	abs := filepath.Join(root, clean)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", clean, err)
	}
	if info.IsDir() {
		entrypoint := filepath.Join(abs, "KNOWLEDGE.md")
		item, err := readItem(root, entrypoint)
		if err != nil {
			return nil, err
		}
		return []Item{item}, nil
	}
	if filepath.Ext(abs) != ".md" {
		return nil, fmt.Errorf("%s: expected markdown file or package directory", clean)
	}
	item, err := readItem(root, abs)
	if err != nil {
		return nil, err
	}
	return []Item{item}, nil
}

func loadFromRoots(root string, relRoots []string) ([]Item, error) {
	var items []Item
	for _, relRoot := range relRoots {
		absRoot := filepath.Join(root, relRoot)
		info, err := os.Stat(absRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", relRoot, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s exists but is not a directory", relRoot)
		}
		err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !isLoadableKnowledgeFile(path) {
				return nil
			}
			item, err := readItem(root, path)
			if err != nil {
				return err
			}
			items = append(items, item)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func isLoadableKnowledgeFile(path string) bool {
	return filepath.Base(path) == "KNOWLEDGE.md" || filepath.Ext(path) == ".md" && !strings.Contains(path, string(filepath.Separator)+"packages"+string(filepath.Separator))
}

func readItem(root string, path string) (Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Item{}, fmt.Errorf("read %s: %w", path, err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Item{}, fmt.Errorf("resolve relative path for %s: %w", path, err)
	}
	return ParseItem(rel, data)
}
```

Also add `cleanRelativePath` in `internal/knowledge/load.go`:

```go
func cleanRelativePath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("%s: path must be relative", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: path must stay inside workspace", relPath)
	}
	return clean, nil
}
```

Ensure `load.go` imports `strings`.

- [ ] **Step 8: Run loader tests and full knowledge package tests**

Run:

```bash
gofmt -w internal/knowledge/item.go internal/knowledge/load.go internal/knowledge/*_test.go
go test ./internal/knowledge
```

Expected: PASS.

- [ ] **Step 9: Commit Task 1**

```bash
git add internal/knowledge/item.go internal/knowledge/load.go internal/knowledge/item_test.go internal/knowledge/load_test.go
git commit -m "feat: load knowledge packages"
```

## Task 2: Package Validation Protocol

**Files:**
- Create: `internal/knowledge/package.go`
- Modify: `internal/knowledge/validate.go`
- Modify: `internal/workspace/workspace.go`
- Test: `internal/knowledge/validate_test.go`
- Test: `internal/knowledge/load_test.go`
- Test: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write failing validation tests**

Add these tests to `internal/knowledge/validate_test.go`:

```go
func TestValidatePackageRequiresProtocolSections(t *testing.T) {
	reg := registry.Registry{
		TechDomains: []string{"backend"},
		Types:       []string{"rule", "package"},
	}
	item := Item{
		Path:        "knowledge/packages/backend/redis/KNOWLEDGE.md",
		ID:          "package:backend.redis.v1",
		Title:       "Redis Practices",
		Type:        "package",
		TechDomains: []string{"backend"},
		Status:      "draft",
		Priority:    "should",
		UpdatedAt:   "2026-04-29",
		Body:        "## Purpose\n\nUse Redis well.\n",
	}

	errs := ValidateItems([]Item{item}, reg)
	if len(errs) != 3 {
		t.Fatalf("expected three missing section errors, got %d: %v", len(errs), errs)
	}
	assertValidationErrorContains(t, errs, "missing package section: When To Use")
	assertValidationErrorContains(t, errs, "missing package section: Start Here")
	assertValidationErrorContains(t, errs, "missing package section: Load On Demand")
}

func TestValidatePackageAcceptsProtocolSections(t *testing.T) {
	reg := registry.Registry{
		TechDomains: []string{"backend"},
		Types:       []string{"rule", "package"},
	}
	item := Item{
		Path:        "knowledge/packages/backend/redis/KNOWLEDGE.md",
		ID:          "package:backend.redis.v1",
		Title:       "Redis Practices",
		Type:        "package",
		TechDomains: []string{"backend"},
		Status:      "draft",
		Priority:    "should",
		UpdatedAt:   "2026-04-29",
		Body: `## Purpose

Use Redis well.

## When To Use

Use this when Redis is involved.

## Start Here

Read this first.

## Load On Demand

- references/key-design.md
`,
	}

	errs := ValidateItems([]Item{item}, reg)
	if len(errs) != 0 {
		t.Fatalf("expected valid package, got %v", errs)
	}
}

func TestValidatePackageRequiresKnowledgeEntrypointPath(t *testing.T) {
	reg := registry.Registry{Types: []string{"package"}}
	item := Item{
		Path:      "knowledge/packages/backend/redis/notes.md",
		ID:        "package:backend.redis.v1",
		Title:     "Redis Practices",
		Type:      "package",
		Status:    "draft",
		Priority:  "should",
		UpdatedAt: "2026-04-29",
		Body: `## Purpose
Text
## When To Use
Text
## Start Here
Text
## Load On Demand
Text
`,
	}

	errs := ValidateItems([]Item{item}, reg)
	assertValidationErrorContains(t, errs, "package entrypoint must be KNOWLEDGE.md")
}

func assertValidationErrorContains(t *testing.T, errs []error, want string) {
	t.Helper()
	for _, err := range errs {
		if strings.Contains(err.Error(), want) {
			return
		}
	}
	t.Fatalf("expected validation error containing %q, got %v", want, errs)
}
```

Ensure imports include `strings` if not already present.

- [ ] **Step 2: Run validation tests to verify they fail**

Run:

```bash
go test ./internal/knowledge -run 'TestValidatePackageRequiresProtocolSections|TestValidatePackageAcceptsProtocolSections|TestValidatePackageRequiresKnowledgeEntrypointPath' -count=1
```

Expected: FAIL because package-specific validation does not exist.

- [ ] **Step 3: Implement package validation helpers**

Create `internal/knowledge/package.go`:

```go
package knowledge

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var packageSegmentPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

var packageRequiredSections = []string{
	"Purpose",
	"When To Use",
	"Start Here",
	"Load On Demand",
}

func validatePackageItem(item Item) []error {
	var errs []error
	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", item.Path, fmt.Sprintf(format, args...)))
	}
	if filepath.Base(item.Path) != "KNOWLEDGE.md" {
		addErr("package entrypoint must be KNOWLEDGE.md")
	}
	if !strings.Contains(item.Path, "/packages/") && !strings.Contains(item.Path, `\packages\`) {
		addErr("package entrypoint must be under a packages directory")
	}
	for _, section := range packageRequiredSections {
		if !hasMarkdownSection(item.Body, section) {
			addErr("missing package section: %s", section)
		}
	}
	return errs
}

func hasMarkdownSection(body string, name string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")) == name {
			return true
		}
	}
	return false
}

func ValidatePackagePath(relPath string) error {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return err
	}
	parts := strings.Split(filepath.ToSlash(clean), "/")
	if len(parts) < 3 {
		return fmt.Errorf("%s: package path is too short", relPath)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("%s: package path contains unsafe segment", relPath)
		}
		if strings.HasPrefix(part, ".") && part != ".inbox" {
			return fmt.Errorf("%s: package path contains hidden segment %s", relPath, part)
		}
	}
	if len(parts) >= 3 && parts[0] == "knowledge" && parts[1] == "packages" {
		return validatePackageSegments(relPath, parts[2:])
	}
	if len(parts) >= 4 && parts[0] == "knowledge" && parts[1] == ".inbox" && parts[2] == "packages" {
		return validatePackageSegments(relPath, parts[3:])
	}
	return fmt.Errorf("%s: package path must be under knowledge/packages or knowledge/.inbox/packages", relPath)
}

func validatePackageSegments(relPath string, segments []string) error {
	if len(segments) == 0 {
		return fmt.Errorf("%s: package path must include at least one package segment", relPath)
	}
	for _, segment := range segments {
		if !packageSegmentPattern.MatchString(segment) {
			return fmt.Errorf("%s: invalid package path segment %s", relPath, segment)
		}
	}
	return nil
}
```

- [ ] **Step 4: Wire package validation into `ValidateItem`**

In `internal/knowledge/validate.go`, after body validation, add:

```go
if item.Type == "package" {
	errs = append(errs, validatePackageItem(item)...)
}
```

- [ ] **Step 5: Add path safety tests**

Add this test to `internal/knowledge/load_test.go` or a new `internal/knowledge/package_test.go`:

```go
func TestValidatePackagePathRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{
		"/knowledge/packages/backend/redis",
		"knowledge/packages/../items/backend",
		"knowledge/packages/backend/.hidden",
		"knowledge/packages/backend/Redis",
		"knowledge/items/backend/auth.md",
	} {
		if err := ValidatePackagePath(path); err == nil {
			t.Fatalf("expected %s to be rejected", path)
		}
	}
}

func TestValidatePackagePathAcceptsFreeMultiLevelPaths(t *testing.T) {
	for _, path := range []string{
		"knowledge/packages/backend/redis/best-practices",
		"knowledge/.inbox/packages/mall-api/api/public-contract",
		"knowledge/packages/templates/go/service",
	} {
		if err := ValidatePackagePath(path); err != nil {
			t.Fatalf("expected %s to be accepted: %v", path, err)
		}
	}
}
```

- [ ] **Step 6: Add package to default workspace types**

In `internal/workspace/workspace_test.go`, update the default type assertion:

```go
if !strings.Contains(string(typesBytes), "rule") ||
	!strings.Contains(string(typesBytes), "runbook") ||
	!strings.Contains(string(typesBytes), "package") {
	t.Fatalf("types.yaml missing default types: %s", string(typesBytes))
}
```

Run the focused test before implementation:

```bash
go test ./internal/workspace -run TestInitCreatesWorkspaceFiles -count=1
```

Expected: FAIL because `defaultTypesYAML` does not include `package`.

In `internal/workspace/workspace.go`, add `package` to `defaultTypesYAML`:

```yaml
  - package
```

- [ ] **Step 7: Run knowledge and workspace tests**

Run:

```bash
gofmt -w internal/knowledge
go test ./internal/knowledge
go test ./internal/workspace
```

Expected: PASS.

- [ ] **Step 8: Commit Task 2**

```bash
git add internal/knowledge internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: validate knowledge package protocol"
```

## Task 3: Validate CLI Scopes

**Files:**
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests for `validate --path` and `validate --inbox`**

Add these tests to `internal/cli/cli_test.go`:

```go
func TestRunValidatePathValidatesSingleInboxPackage(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "validated 1 knowledge item(s)") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunValidateInboxValidatesInboxOnly(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	writeCLIFile(t, root, "knowledge/packages/backend/broken/KNOWLEDGE.md", `---
id: package:backend.broken.v1
title: Broken
type: package
status: active
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate", "--inbox"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "validated 1 knowledge item(s)") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunValidateDefaultIgnoresInboxCandidates(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", validCLIKnowledgeItem("rule:backend.auth.v1", "rule"))
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/broken/KNOWLEDGE.md", `---
id: package:backend.broken.v1
title: Broken
type: package
status: draft
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "validated 1 knowledge item(s)" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func writeCLIRegistry(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, database]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule, package]\n")
}

func validCLIPackage(id string) string {
	return `---
id: ` + id + `
title: Redis Best Practices
type: package
tech_domains: [backend]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis]
updated_at: 2026-04-29
---
## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read the short rules first.

## Load On Demand

- references/key-design.md when designing keys.
`
}

func validCLIKnowledgeItem(id string, itemType string) string {
	return `---
id: ` + id + `
title: Auth rule
type: ` + itemType + `
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: must
updated_at: 2026-04-29
---
Use explicit auth middleware.
`
}
```

If equivalent helpers already exist, reuse them and keep the assertions.

- [ ] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunValidatePathValidatesSingleInboxPackage|TestRunValidateInboxValidatesInboxOnly|TestRunValidateDefaultIgnoresInboxCandidates' -count=1
```

Expected: FAIL because `validate` does not parse flags.

- [ ] **Step 3: Implement validation scopes in CLI**

In the `case "validate":` branch of `internal/cli/cli.go`, replace the current direct call with:

```go
flags := flag.NewFlagSet("validate", flag.ContinueOnError)
flags.SetOutput(stderr)
includeInbox := flags.Bool("inbox", false, "validate inbox candidates")
path := flags.String("path", "", "validate a single item or package path")
if err := flags.Parse(args[1:]); err != nil {
	return 2
}
if *includeInbox && strings.TrimSpace(*path) != "" {
	fmt.Fprintln(stderr, "validate: --inbox and --path cannot be used together")
	return 2
}
root, err := os.Getwd()
if err != nil {
	fmt.Fprintf(stderr, "get current directory: %v\n", err)
	return 1
}
items, err := loadAndValidateKnowledge(root, stderr, validationScope{Inbox: *includeInbox, Path: *path})
if err != nil {
	return 1
}
fmt.Fprintf(stdout, "validated %d knowledge item(s)\n", len(items))
return 0
```

Add:

```go
type validationScope struct {
	Inbox bool
	Path  string
}
```

Update `loadAndValidateKnowledge` signature:

```go
func loadAndValidateKnowledge(root string, stderr io.Writer, scope validationScope) ([]knowledge.Item, error) {
	reg, err := registry.Load(root)
	if err != nil {
		fmt.Fprintf(stderr, "load registry: %v\n", err)
		return nil, err
	}

	var items []knowledge.Item
	switch {
	case strings.TrimSpace(scope.Path) != "":
		items, err = knowledge.LoadPath(root, scope.Path)
	case scope.Inbox:
		items, err = knowledge.LoadInbox(root)
	default:
		items, err = knowledge.LoadOfficial(root)
	}
	if err != nil {
		fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
		return nil, err
	}

	errs := knowledge.ValidateItems(items, reg)
	for _, err := range errs {
		fmt.Fprintln(stderr, err)
	}
	if len(errs) > 0 {
		err := fmt.Errorf("validation failed with %d error(s)", len(errs))
		fmt.Fprintln(stderr, err)
		return nil, err
	}
	return items, nil
}
```

Update existing callers:

```go
items, err := loadAndValidateKnowledge(root, stderr, validationScope{})
```

Ensure `internal/cli/cli.go` imports `strings`.

- [ ] **Step 4: Run CLI validation tests**

Run:

```bash
gofmt -w internal/cli/cli.go internal/cli/cli_test.go
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit Task 3**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: add scoped knowledge validation"
```

## Task 4: Safe Promotion CLI

**Files:**
- Modify: `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing promote tests**

Add these tests to `internal/cli/cli_test.go`:

```go
func TestRunPromoteMovesInboxPackageToOfficialPackages(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected promoted package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/.inbox/packages/backend/redis/best-practices")); !os.IsNotExist(err) {
		t.Fatalf("expected inbox package removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "promoted knowledge/packages/backend/redis/best-practices") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "run argos index") {
		t.Fatalf("expected index recommendation, got %q", stdout.String())
	}
}

func TestRunPromoteRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	writeCLIFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.existing.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "target already exists") {
		t.Fatalf("expected overwrite error, got %q", stderr.String())
	}
}

func TestRunPromoteRejectsInvalidCandidate(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/broken/KNOWLEDGE.md", `---
id: package:backend.redis.broken.v1
title: Broken
type: package
status: draft
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/broken"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/.inbox/packages/backend/redis/broken/KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected invalid candidate to remain in inbox: %v", err)
	}
	if !strings.Contains(stderr.String(), "validation failed") {
		t.Fatalf("expected validation error, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run promote tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunPromoteMovesInboxPackageToOfficialPackages|TestRunPromoteRefusesOverwrite|TestRunPromoteRejectsInvalidCandidate' -count=1
```

Expected: FAIL because `promote` is an unknown command.

- [ ] **Step 3: Implement promote command**

In `internal/cli/cli.go`, add a `case "promote":` branch:

```go
case "promote":
	flags := flag.NewFlagSet("promote", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("path", "", "candidate item or package path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Fprintln(stderr, "promote: --path is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get current directory: %v\n", err)
		return 1
	}
	target, err := promoteCandidate(root, *path, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "promote: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "promoted %s\n", target)
	fmt.Fprintln(stdout, "run argos index to refresh query results")
	return 0
```

Add helper functions in `internal/cli/cli.go`:

```go
func promoteCandidate(root string, relPath string, stderr io.Writer) (string, error) {
	clean := filepath.Clean(relPath)
	if filepath.IsAbs(relPath) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: candidate path must be relative and inside workspace", relPath)
	}
	target, err := promotionTarget(clean)
	if err != nil {
		return "", err
	}
	if _, err := loadAndValidateKnowledge(root, stderr, validationScope{Path: clean}); err != nil {
		return "", err
	}
	targetAbs := filepath.Join(root, target)
	if _, err := os.Stat(targetAbs); err == nil {
		return "", fmt.Errorf("target already exists: %s", target)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target %s: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return "", fmt.Errorf("create target parent: %w", err)
	}
	if err := os.Rename(filepath.Join(root, clean), targetAbs); err != nil {
		return "", fmt.Errorf("move candidate: %w", err)
	}
	if _, err := loadAndValidateKnowledge(root, stderr, validationScope{}); err != nil {
		return "", fmt.Errorf("official validation failed after promotion: %w", err)
	}
	return target, nil
}

func promotionTarget(clean string) (string, error) {
	slash := filepath.ToSlash(clean)
	for _, mapping := range []struct {
		inbox    string
		official string
	}{
		{"knowledge/.inbox/packages/", "knowledge/packages/"},
		{"knowledge/.inbox/items/", "knowledge/items/"},
	} {
		if strings.HasPrefix(slash, mapping.inbox) {
			rest := strings.TrimPrefix(slash, mapping.inbox)
			if rest == "" || strings.Contains(rest, "../") {
				return "", fmt.Errorf("%s: invalid inbox candidate path", clean)
			}
			return filepath.FromSlash(mapping.official + rest), nil
		}
	}
	return "", fmt.Errorf("%s: candidate must be under knowledge/.inbox/items or knowledge/.inbox/packages", clean)
}
```

Update `printUsage` to include:

```go
fmt.Fprintln(w, "  promote")
```

- [ ] **Step 4: Run promote tests and CLI suite**

Run:

```bash
gofmt -w internal/cli/cli.go internal/cli/cli_test.go
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 5: Commit Task 4**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: promote inbox knowledge"
```

## Task 5: Index And Query Package Entrypoints

**Files:**
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing index CLI test for official packages**

Add this test to `internal/cli/cli_test.go`:

```go
func TestRunIndexIncludesOfficialPackages(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"index"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}

	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	item, err := store.GetItem("package:backend.redis.best-practices.v1")
	if err != nil {
		t.Fatalf("expected indexed package: %v", err)
	}
	if item.Type != "package" {
		t.Fatalf("expected package type, got %q", item.Type)
	}
	if !strings.Contains(item.Body, "## Load On Demand") {
		t.Fatalf("expected package body, got %q", item.Body)
	}
}
```

- [ ] **Step 2: Run index package test to verify it fails**

Run:

```bash
go test ./internal/cli -run TestRunIndexIncludesOfficialPackages -count=1
```

Expected: FAIL until `index` uses the official package-aware loader.

- [ ] **Step 3: Ensure index uses official package-aware validation**

In `internal/cli/cli.go`, the `case "index":` branch must call:

```go
items, err := loadAndValidateKnowledge(root, stderr, validationScope{})
```

Do not change the SQLite schema for tags in this plan. Tags are parsed metadata for future use, but current query and MCP results do not return tags.

- [ ] **Step 4: Add MCP test for fetching package entrypoint**

In `internal/mcp/server_test.go`, extend `buildMCPTestStore` or add a second helper so the test store includes:

```go
{
	Path:        "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md",
	ID:          "package:backend.redis.best-practices.v1",
	Title:       "Redis Best Practices",
	Type:        "package",
	TechDomains: []string{"backend"},
	Status:      "active",
	Priority:    "should",
	UpdatedAt:   "2026-04-29",
	Body: `## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read this first.

## Load On Demand

- references/key-design.md
`,
}
```

Add:

```go
func TestToolCallGetKnowledgeItemReturnsPackageEntrypoint(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_knowledge_item","arguments":{"id":"package:backend.redis.best-practices.v1"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"type": "package"`) || !strings.Contains(text, "## Load On Demand") {
		t.Fatalf("expected package entrypoint body: %s", text)
	}
}
```

- [ ] **Step 5: Run query/index/MCP tests**

Run:

```bash
gofmt -w internal/cli/cli.go internal/cli/cli_test.go internal/mcp/server_test.go
go test ./internal/cli ./internal/index ./internal/query ./internal/mcp
```

Expected: PASS.

- [ ] **Step 6: Commit Task 5**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go internal/mcp/server_test.go
git commit -m "feat: index knowledge packages"
```

## Task 6: Documentation And Final Verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README with package workflow**

Add a README section:

````md
## Knowledge Packages

Single knowledge items live under `knowledge/items/`.

Structured knowledge packages live under `knowledge/packages/` and use
`KNOWLEDGE.md` as their indexed entrypoint. Optional package directories include
`references/`, `examples/`, `checklists/`, `scripts/`, and `assets/`.

Package candidates are created under `knowledge/.inbox/packages/` and reviewed
before promotion:

```bash
argos validate --path knowledge/.inbox/packages/backend/redis/best-practices
argos promote --path knowledge/.inbox/packages/backend/redis/best-practices
argos index
```

`argos validate` checks official knowledge. `argos validate --inbox` checks inbox
candidates. `argos validate --path <path>` checks one item or package.
```
````

Also update the command list to include:

```text
argos promote --path <candidate>
```

- [ ] **Step 2: Run full verification**

Run:

```bash
gofmt -w internal/knowledge internal/cli internal/mcp internal/index internal/query
go test ./internal/knowledge
go test ./internal/cli
go test ./internal/mcp
go test ./...
go vet ./...
go test -race ./...
go build -o /tmp/argos-knowledge-packages ./cmd/argos
git diff --check
go mod tidy -diff
```

Expected: all commands exit 0 and `go mod tidy -diff` prints no diff.

- [ ] **Step 3: Commit Task 6**

```bash
git add README.md internal/knowledge internal/workspace internal/cli internal/mcp
git commit -m "docs: document knowledge packages"
```

## Requirement Coverage

- Package structure and `KNOWLEDGE.md` entrypoint: Tasks 1, 2, and 6.
- Optional directories and path safety: Task 2.
- Package protocol sections: Task 2.
- Tags as free-form metadata: Task 1.
- Official, inbox, and path-level validation scopes: Task 3.
- Safe CLI promotion: Task 4.
- Package indexing and MCP retrieval through `get_knowledge_item`: Task 5.
- README workflow documentation: Task 6.

## Non-Goals For This Plan

- `argos new` or `argos capture` generation commands.
- Agent skill implementation.
- Remote package installation.
- Source policy or package layering.
- Package asset MCP tools such as `list_package_assets` or `get_package_asset`.
- Parsing `Load On Demand` links.
- Automatically running package scripts.
