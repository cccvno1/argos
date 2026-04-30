# Argos Discovery Validation Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a repeatable Discovery validation harness with golden fixtures, automated query/CLI/MCP/adapter checks, and AI dogfood instructions that verify Argos can route, load, and cite knowledge without overclaiming.

**Architecture:** Store one canonical Markdown fixture workspace under `testdata/discovery-golden/`, describe expected behavior in `cases.json`, and add a small `internal/discoverytest` helper package that copies the fixture into temporary workspaces, loads cases, builds indexes, and exposes assertion helpers. Query, CLI, MCP, and adapter tests consume the same cases so the harness validates real entrypoints without duplicating scenario definitions. Dogfood documentation adds a fresh-context runner/evaluator procedure so AI testing is not polluted by design history.

**Tech Stack:** Go, `encoding/json`, `os`, `path/filepath`, existing `knowledge.LoadOfficial`, existing `index.Rebuild`, existing `query.Service`, existing CLI and MCP tests, Markdown fixture files, JSON golden cases.

---

## File Structure

- Create `testdata/discovery-golden/cases.json`: machine-readable golden case definitions for the 21 approved scenarios.
- Create `testdata/discovery-golden/knowledge/domains.yaml`: fixture domain registry.
- Create `testdata/discovery-golden/knowledge/projects.yaml`: fixture project registry.
- Create `testdata/discovery-golden/knowledge/types.yaml`: fixture type registry.
- Create `testdata/discovery-golden/knowledge/items/backend/auth-refresh-rule.md`: strong auth rule.
- Create `testdata/discovery-golden/knowledge/items/backend/session-decision.md`: auth/session decision.
- Create `testdata/discovery-golden/knowledge/items/backend/generic-backend.md`: generic backend guidance.
- Create `testdata/discovery-golden/knowledge/items/backend/cache-policy.md`: partial cache/domain guidance.
- Create `testdata/discovery-golden/knowledge/items/backend/auth-lesson.md`: lesson-only guidance.
- Create `testdata/discovery-golden/knowledge/items/backend/global-refresh-reference.md`: global knowledge with no project list.
- Create `testdata/discovery-golden/knowledge/items/backend/deprecated-auth.md`: deprecated guidance.
- Create `testdata/discovery-golden/knowledge/items/other/other-project-auth.md`: project-scope mismatch fixture.
- Create `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/KNOWLEDGE.md`: package entrypoint fixture.
- Create `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/references/example.md`: package asset that must not index as standalone.
- Create `internal/discoverytest/golden.go`: shared fixture/case loading and index setup helpers.
- Create `internal/discoverytest/golden_test.go`: tests for the helper package itself.
- Create `internal/query/discovery_golden_test.go`: query-service golden checks.
- Create `internal/cli/discovery_golden_test.go`: CLI JSON golden checks.
- Modify `internal/mcp/server_test.go`: add strict schema and golden interface checks using the shared fixture.
- Modify `internal/adapters/adapters_test.go`: add explicit guard that adapters do not recommend direct SQLite/vector access.
- Create `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`: AI runner/evaluator checklist.
- Create `docs/superpowers/templates/argos-discovery-dogfood-report.md`: stable report template.
- Modify `README.md`: document how to run automated validation and AI dogfood validation.

## Task 1: Add The Golden Corpus And Case Contract

**Files:**
- Create: `testdata/discovery-golden/cases.json`
- Create: `testdata/discovery-golden/knowledge/domains.yaml`
- Create: `testdata/discovery-golden/knowledge/projects.yaml`
- Create: `testdata/discovery-golden/knowledge/types.yaml`
- Create: `testdata/discovery-golden/knowledge/items/backend/auth-refresh-rule.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/session-decision.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/generic-backend.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/cache-policy.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/auth-lesson.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/global-refresh-reference.md`
- Create: `testdata/discovery-golden/knowledge/items/backend/deprecated-auth.md`
- Create: `testdata/discovery-golden/knowledge/items/other/other-project-auth.md`
- Create: `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/KNOWLEDGE.md`
- Create: `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/references/example.md`

- [ ] **Step 1: Create the fixture registries**

Add `testdata/discovery-golden/knowledge/domains.yaml`:

```yaml
tech_domains: [backend, security, database, payments, platform]
business_domains: [account, order, billing]
```

Add `testdata/discovery-golden/knowledge/projects.yaml`:

```yaml
projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend, security]
    business_domains: [account]
  - id: warehouse-api
    name: Warehouse API
    path: services/warehouse-api
    tech_domains: [backend]
    business_domains: [order]
```

Add `testdata/discovery-golden/knowledge/types.yaml`:

```yaml
types: [rule, decision, lesson, runbook, reference, package]
```

- [ ] **Step 2: Create the auth refresh rule fixture**

Add `testdata/discovery-golden/knowledge/items/backend/auth-refresh-rule.md`:

````markdown
---
id: rule:backend.auth-refresh.v1
title: Refresh Token Endpoint Rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  languages: [go]
  files: ["internal/auth/**"]
updated_at: 2026-04-30
tags: [auth, refresh-token, session-renewal]
---
# Refresh Token Endpoint Rule

Refresh token endpoints must authenticate the session, rotate refresh tokens,
and reject reuse attempts.

Implementation details:

- require auth middleware before touching account state
- rotate refresh token identifiers on every successful renewal
- emit an audit event when reuse is detected
````

- [ ] **Step 3: Create the session decision fixture**

Add `testdata/discovery-golden/knowledge/items/backend/session-decision.md`:

````markdown
---
id: decision:backend.session-renewal.v1
title: Session Renewal Decision
type: decision
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: should
applies_to:
  files: ["internal/auth/**", "internal/session/**"]
updated_at: 2026-04-30
tags: [auth, refresh-token, session]
---
# Session Renewal Decision

Mall API renews sessions through refresh token rotation instead of extending
access token lifetime.

The decision exists because short access token lifetimes limit exposure while
refresh rotation preserves user experience.
````

- [ ] **Step 4: Create generic, partial, lesson, global, deprecated, and mismatch fixtures**

Add `testdata/discovery-golden/knowledge/items/backend/generic-backend.md`:

```markdown
---
id: rule:backend.generic.v1
title: Generic Backend Rule
type: rule
tech_domains: [backend, platform]
business_domains: []
projects: [mall-api]
status: active
priority: should
updated_at: 2026-04-30
tags: [backend, platform]
---
# Generic Backend Rule

Backend changes should keep handlers small, validate inputs explicitly, and
return typed errors.
```

Add `testdata/discovery-golden/knowledge/items/backend/cache-policy.md`:

```markdown
---
id: reference:backend.cache-policy.v1
title: Cache Policy Reference
type: reference
tech_domains: [backend, database]
business_domains: []
projects: [mall-api]
status: active
priority: should
applies_to:
  files: ["internal/cache/**"]
updated_at: 2026-04-30
tags: [cache, redis, ttl]
---
# Cache Policy Reference

Cache entries should have explicit TTLs and should not hide source-of-truth
database writes.
```

Add `testdata/discovery-golden/knowledge/items/backend/auth-lesson.md`:

```markdown
---
id: lesson:backend.auth-debug.v1
title: Auth Debugging Lesson
type: lesson
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-04-30
tags: [auth, session, debugging]
---
# Auth Debugging Lesson

When session renewal tests fail, inspect token rotation logs before changing
middleware order.
```

Add `testdata/discovery-golden/knowledge/items/backend/global-refresh-reference.md`:

```markdown
---
id: reference:backend.global-refresh.v1
title: Global Refresh Token Reference
type: reference
tech_domains: [backend, security]
business_domains: []
projects: []
status: active
priority: may
updated_at: 2026-04-30
tags: [refresh-token, auth]
---
# Global Refresh Token Reference

Refresh token guidance applies across projects when project-specific guidance is
missing.
```

Add `testdata/discovery-golden/knowledge/items/backend/deprecated-auth.md`:

```markdown
---
id: rule:backend.deprecated-auth.v1
title: Deprecated Auth Rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: deprecated
priority: must
updated_at: 2026-04-30
tags: [auth, refresh-token]
---
# Deprecated Auth Rule

Deprecated auth guidance must not appear unless deprecated knowledge is
explicitly requested.
```

Add `testdata/discovery-golden/knowledge/items/other/other-project-auth.md`:

```markdown
---
id: rule:warehouse.auth.v1
title: Warehouse Auth Rule
type: rule
tech_domains: [backend, security]
business_domains: [order]
projects: [warehouse-api]
status: active
priority: must
updated_at: 2026-04-30
tags: [auth, warehouse]
---
# Warehouse Auth Rule

Warehouse-only auth guidance must not route to Mall API tasks.
```

- [ ] **Step 5: Create the package fixture and a package asset**

Add `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/KNOWLEDGE.md`:

```markdown
---
id: package:backend.auth-refresh.v1
title: Auth Refresh Package
type: package
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-04-30
tags: [auth, refresh-token, package]
---
## Purpose

Guide refresh token endpoint work with progressive disclosure.

## When To Use

Use for implementation or review of Mall API refresh token endpoint behavior.

## Start Here

Load the refresh token endpoint rule before editing `internal/auth/**`.

## Load On Demand

- `references/example.md` when examples are needed.
```

Add `testdata/discovery-golden/knowledge/packages/backend/auth-refresh/references/example.md`:

```markdown
# Auth Refresh Example

This package asset is intentionally not a standalone knowledge item.
```

- [ ] **Step 6: Create the first version of `cases.json`**

Add `testdata/discovery-golden/cases.json` with this structure:

```json
{
  "cases": [
    {
      "id": "map_inventory_normal",
      "operation": "map",
      "input": {"project": "mall-api"},
      "expected": {
        "inventory_types_min": {"rule": 2, "package": 1},
        "include_domains": ["backend", "security"],
        "include_tags": ["auth", "refresh-token"],
        "include_ids": ["package:backend.auth-refresh.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "map_inventory_empty",
      "operation": "map-empty",
      "input": {"project": "mall-api"},
      "expected": {"groups_empty": true, "no_bodies": true}
    },
    {
      "id": "map_hides_deprecated_by_default",
      "operation": "map",
      "input": {"project": "mall-api"},
      "expected": {
        "exclude_ids": ["rule:backend.deprecated-auth.v1"],
        "include_deprecated_id_when_requested": "rule:backend.deprecated-auth.v1",
        "no_bodies": true
      }
    },
    {
      "id": "map_global_knowledge_visible",
      "operation": "map",
      "input": {"project": "mall-api"},
      "expected": {
        "include_ids": ["reference:backend.global-refresh.v1"],
        "exclude_ids": ["rule:warehouse.auth.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "strong_auth_refresh_full_signal",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add refresh token endpoint",
        "query": "refresh token session renewal",
        "files": ["internal/auth/session.go"],
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "top_id": "rule:backend.auth-refresh.v1",
        "include_ids": ["rule:backend.auth-refresh.v1", "decision:backend.session-renewal.v1"],
        "exclude_ids": ["rule:backend.deprecated-auth.v1"],
        "require_next_call_tools": ["get_knowledge_item"],
        "no_bodies": true
      }
    },
    {
      "id": "strong_auth_refresh_query_only",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "planning",
        "query": "refresh token session renewal",
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "include_ids": ["rule:backend.auth-refresh.v1"],
        "exclude_ids": ["reference:backend.cache-policy.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "strong_auth_refresh_task_only",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add refresh token endpoint",
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "include_ids": ["rule:backend.auth-refresh.v1"],
        "exclude_ids": ["reference:backend.cache-policy.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "strong_file_scope_beats_generic",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add backend auth handler",
        "query": "backend auth handler",
        "files": ["internal/auth/handler.go"],
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "top_id": "rule:backend.auth-refresh.v1",
        "include_ids": ["rule:backend.generic.v1"],
        "why_contains": ["file scope matched"],
        "no_bodies": true
      }
    },
    {
      "id": "partial_domain_without_task_detail",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "tune cache ttl for product list",
        "query": "cache ttl product list",
        "files": ["internal/catalog/products.go"],
        "limit": 5
      },
      "expected": {
        "coverage": "partial",
        "include_ids": ["reference:backend.cache-policy.v1"],
        "require_missing_hints": true,
        "no_bodies": true
      }
    },
    {
      "id": "partial_lesson_without_rule",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "debugging",
        "task": "debug session renewal test failure",
        "query": "session renewal tests fail logs",
        "limit": 5
      },
      "expected": {
        "coverage": "partial",
        "include_ids": ["lesson:backend.auth-debug.v1"],
        "require_missing_hints": true,
        "no_bodies": true
      }
    },
    {
      "id": "partial_package_entrypoint_without_detail",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "planning",
        "task": "understand auth refresh package",
        "query": "auth refresh package examples",
        "types": ["package"],
        "limit": 5
      },
      "expected": {
        "coverage": "partial",
        "include_ids": ["package:backend.auth-refresh.v1"],
        "exclude_ids": ["references/example.md"],
        "no_bodies": true
      }
    },
    {
      "id": "weak_single_generic_term",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add warehouse barcode scanner",
        "query": "barcode scanner token",
        "limit": 5
      },
      "expected": {
        "coverage": "weak",
        "forbid_next_call_tools": ["get_knowledge_item", "cite_knowledge"],
        "no_bodies": true
      }
    },
    {
      "id": "weak_broad_tag_only",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add warehouse barcode scanner",
        "query": "backend platform",
        "tags": ["backend"],
        "limit": 5
      },
      "expected": {
        "coverage": "weak",
        "forbid_next_call_tools": ["get_knowledge_item", "cite_knowledge"],
        "no_bodies": true
      }
    },
    {
      "id": "none_payment_webhook",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add payment webhook signature verification",
        "query": "payment webhook signature",
        "limit": 5
      },
      "expected": {
        "coverage": "none",
        "items_empty": true,
        "forbid_next_call_tools": ["get_knowledge_item", "cite_knowledge"],
        "no_bodies": true
      }
    },
    {
      "id": "none_project_scope_mismatch",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "update warehouse auth flow",
        "query": "warehouse auth",
        "limit": 5
      },
      "expected": {
        "coverage": "none",
        "exclude_ids": ["rule:warehouse.auth.v1"],
        "forbid_next_call_tools": ["get_knowledge_item", "cite_knowledge"],
        "no_bodies": true
      }
    },
    {
      "id": "none_explicit_filter_excludes_match",
      "operation": "discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add refresh token endpoint",
        "query": "refresh token",
        "tags": ["payments"],
        "limit": 5
      },
      "expected": {
        "coverage": "none",
        "items_empty": true,
        "forbid_next_call_tools": ["get_knowledge_item", "cite_knowledge"],
        "no_bodies": true
      }
    },
    {
      "id": "progressive_disclosure_and_citation_guard",
      "operation": "workflow",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add refresh token endpoint",
        "query": "refresh token session renewal",
        "files": ["internal/auth/session.go"],
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "include_ids": ["rule:backend.auth-refresh.v1"],
        "load_ids": ["rule:backend.auth-refresh.v1"],
        "cite_ids": ["rule:backend.auth-refresh.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "interface_cli_discover_matches_query",
      "operation": "cli-discover",
      "input": {
        "project": "mall-api",
        "phase": "implementation",
        "task": "add refresh token endpoint",
        "query": "refresh token session renewal",
        "files": ["internal/auth/session.go"],
        "limit": 5
      },
      "expected": {
        "coverage": "strong",
        "include_ids": ["rule:backend.auth-refresh.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "interface_cli_map_matches_query",
      "operation": "cli-map",
      "input": {"project": "mall-api"},
      "expected": {
        "inventory_types_min": {"rule": 2, "package": 1},
        "include_ids": ["package:backend.auth-refresh.v1"],
        "no_bodies": true
      }
    },
    {
      "id": "interface_mcp_strict_schema",
      "operation": "mcp-schema",
      "input": {"project": "mall-api"},
      "expected": {
        "reject_unknown_arguments": true,
        "reject_missing_task_and_query": true,
        "reject_out_of_range_limit": true
      }
    },
    {
      "id": "adapter_flow_recommendations",
      "operation": "adapter",
      "input": {"project": "mall-api"},
      "expected": {
        "include_text": ["MCP", "CLI JSON", "generated adapter files", "Markdown source"],
        "exclude_text": ["query SQLite", "query vector"]
      }
    }
  ]
}
```

- [ ] **Step 7: Commit the fixture and case contract**

Run:

```bash
git add testdata/discovery-golden
git commit -m "test: add discovery golden fixtures"
```

Expected: commit succeeds.

## Task 2: Add Shared Golden Harness Helpers

**Files:**
- Create: `internal/discoverytest/golden_test.go`
- Create: `internal/discoverytest/golden.go`

- [ ] **Step 1: Write failing tests for the shared helper package**

Create `internal/discoverytest/golden_test.go`:

```go
package discoverytest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCasesReadsGoldenCases(t *testing.T) {
	cases := LoadCases(t)
	if len(cases) != 21 {
		t.Fatalf("expected 21 golden cases, got %d", len(cases))
	}
	if CaseByID(t, cases, "strong_auth_refresh_full_signal").Expected.Coverage != "strong" {
		t.Fatalf("expected strong_auth_refresh_full_signal to expect strong coverage")
	}
}

func TestCopyWorkspaceCopiesKnowledgeAndCases(t *testing.T) {
	root := CopyWorkspace(t)
	for _, rel := range []string{
		"cases.json",
		"knowledge/domains.yaml",
		"knowledge/items/backend/auth-refresh-rule.md",
		"knowledge/packages/backend/auth-refresh/KNOWLEDGE.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected copied fixture %s: %v", rel, err)
		}
	}
}

func TestBuildIndexedWorkspaceCreatesQueryableStore(t *testing.T) {
	root, store := BuildIndexedWorkspace(t)
	defer store.Close()

	if _, err := os.Stat(filepath.Join(root, "argos", "index.db")); err != nil {
		t.Fatalf("expected index.db: %v", err)
	}
	item, err := store.GetItem("rule:backend.auth-refresh.v1")
	if err != nil {
		t.Fatalf("expected auth rule in index: %v", err)
	}
	if item.Body == "" {
		t.Fatalf("expected indexed item body")
	}
}
```

- [ ] **Step 2: Run helper tests to verify they fail**

Run:

```bash
go test ./internal/discoverytest -count=1
```

Expected: FAIL because `internal/discoverytest` does not exist yet.

- [ ] **Step 3: Implement the shared helper package**

Create `internal/discoverytest/golden.go`:

```go
package discoverytest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

type CaseFile struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID        string   `json:"id"`
	Operation string   `json:"operation"`
	Input     Input    `json:"input"`
	Expected  Expected `json:"expected"`
}

type Input struct {
	Project           string   `json:"project"`
	Phase             string   `json:"phase"`
	Task              string   `json:"task"`
	Query             string   `json:"query"`
	Files             []string `json:"files"`
	Types             []string `json:"types"`
	Tags              []string `json:"tags"`
	Domains           []string `json:"domains"`
	Status            []string `json:"status"`
	IncludeDeprecated bool     `json:"include_deprecated"`
	Limit             int      `json:"limit"`
}

type Expected struct {
	Coverage                         string         `json:"coverage"`
	TopID                            string         `json:"top_id"`
	IncludeIDs                       []string       `json:"include_ids"`
	ExcludeIDs                       []string       `json:"exclude_ids"`
	LoadIDs                          []string       `json:"load_ids"`
	CiteIDs                          []string       `json:"cite_ids"`
	IncludeDomains                   []string       `json:"include_domains"`
	IncludeTags                      []string       `json:"include_tags"`
	IncludeText                      []string       `json:"include_text"`
	ExcludeText                      []string       `json:"exclude_text"`
	InventoryTypesMin                map[string]int `json:"inventory_types_min"`
	IncludeDeprecatedIDWhenRequested string         `json:"include_deprecated_id_when_requested"`
	RequireNextCallTools             []string       `json:"require_next_call_tools"`
	ForbidNextCallTools              []string       `json:"forbid_next_call_tools"`
	WhyContains                      []string       `json:"why_contains"`
	NoBodies                         bool           `json:"no_bodies"`
	GroupsEmpty                      bool           `json:"groups_empty"`
	ItemsEmpty                       bool           `json:"items_empty"`
	RequireMissingHints              bool           `json:"require_missing_hints"`
	RejectUnknownArguments           bool           `json:"reject_unknown_arguments"`
	RejectMissingTaskAndQuery        bool           `json:"reject_missing_task_and_query"`
	RejectOutOfRangeLimit            bool           `json:"reject_out_of_range_limit"`
}

func FixtureRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve discoverytest caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "discovery-golden"))
	if _, err := os.Stat(filepath.Join(root, "cases.json")); err != nil {
		t.Fatalf("find discovery golden fixture: %v", err)
	}
	return root
}

func LoadCases(t testing.TB) []Case {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(FixtureRoot(t), "cases.json"))
	if err != nil {
		t.Fatalf("read cases.json: %v", err)
	}
	var file CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse cases.json: %v", err)
	}
	return file.Cases
}

func CaseByID(t testing.TB, cases []Case, id string) Case {
	t.Helper()
	for _, tc := range cases {
		if tc.ID == id {
			return tc
		}
	}
	t.Fatalf("missing golden case %q", id)
	return Case{}
}

func CopyWorkspace(t testing.TB) string {
	t.Helper()
	dst := t.TempDir()
	if err := copyDir(FixtureRoot(t), dst); err != nil {
		t.Fatalf("copy discovery golden fixture: %v", err)
	}
	return dst
}

func BuildIndexedWorkspace(t testing.TB) (string, *index.Store) {
	t.Helper()
	root := CopyWorkspace(t)
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("load golden knowledge: %v", err)
	}
	dbPath := filepath.Join(root, "argos", "index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		t.Fatalf("rebuild golden index: %v", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("open golden index: %v", err)
	}
	return root, store
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
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
}
```

- [ ] **Step 4: Run helper tests**

Run:

```bash
go test ./internal/discoverytest -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the helper package**

Run:

```bash
git add internal/discoverytest
git commit -m "test: add discovery golden harness helpers"
```

Expected: commit succeeds.

## Task 3: Add Query-Service Golden Tests

**Files:**
- Create: `internal/query/discovery_golden_test.go`

- [ ] **Step 1: Write query-service golden tests**

Create `internal/query/discovery_golden_test.go`:

```go
package query

import (
	"strings"
	"testing"

	"argos/internal/discoverytest"
)

func TestGoldenDiscoveryCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "discover" && tc.Operation != "workflow" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.Discover(DiscoverRequest{
				Project:           tc.Input.Project,
				Phase:             tc.Input.Phase,
				Task:              tc.Input.Task,
				Query:             tc.Input.Query,
				Files:             tc.Input.Files,
				Types:             tc.Input.Types,
				Tags:              tc.Input.Tags,
				Domains:           tc.Input.Domains,
				Status:            tc.Input.Status,
				IncludeDeprecated: tc.Input.IncludeDeprecated,
				Limit:             tc.Input.Limit,
			})
			if err != nil {
				t.Fatalf("Discover returned error: %v", err)
			}
			assertCoverage(t, result.Coverage, tc.Expected.Coverage)
			assertDiscoveryIDs(t, result.Items, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertTopID(t, result.Items, tc.Expected.TopID)
			assertNoDiscoveryBodies(t, result.Items, tc.Expected.NoBodies)
			assertNextCalls(t, result.NextCalls, tc.Expected.RequireNextCallTools, tc.Expected.ForbidNextCallTools)
			assertMissingHints(t, result.Coverage, tc.Expected.RequireMissingHints)
			assertWhyContains(t, result.Items, tc.Expected.WhyContains)
			if tc.Expected.ItemsEmpty && len(result.Items) != 0 {
				t.Fatalf("expected empty items, got %#v", result.Items)
			}
		})
	}
}

func TestGoldenMapCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "map" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.Map(MapRequest{Project: tc.Input.Project})
			if err != nil {
				t.Fatalf("Map returned error: %v", err)
			}
			assertInventoryMinimums(t, result.Inventory.Types, tc.Expected.InventoryTypesMin)
			assertStringIncludes(t, result.Inventory.Domains, tc.Expected.IncludeDomains)
			assertStringIncludes(t, result.Inventory.Tags, tc.Expected.IncludeTags)
			assertMapIDs(t, result.Groups, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertNoMapBodies(t, result.Groups, tc.Expected.NoBodies)
		})
	}
}

func TestGoldenMapEmptyCase(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "map_inventory_empty")

	result, err := service.Map(MapRequest{Project: "unknown-api"})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	if tc.Expected.GroupsEmpty && len(result.Groups) != 0 {
		t.Fatalf("expected empty groups, got %#v", result.Groups)
	}
	assertNoMapBodies(t, result.Groups, tc.Expected.NoBodies)
}

func TestGoldenDeprecatedMapCase(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "map_hides_deprecated_by_default")

	result, err := service.Map(MapRequest{Project: tc.Input.Project, IncludeDeprecated: true})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	assertMapIDs(t, result.Groups, []string{tc.Expected.IncludeDeprecatedIDWhenRequested}, nil)
}

func TestGoldenProgressiveDisclosureAndCitationGuard(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "progressive_disclosure_and_citation_guard")

	discovered, err := service.Discover(DiscoverRequest{
		Project: tc.Input.Project,
		Phase:   tc.Input.Phase,
		Task:    tc.Input.Task,
		Query:   tc.Input.Query,
		Files:   tc.Input.Files,
		Limit:   tc.Input.Limit,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	assertNoDiscoveryBodies(t, discovered.Items, true)

	for _, id := range tc.Expected.LoadIDs {
		item, err := service.GetKnowledgeItem(id)
		if err != nil {
			t.Fatalf("GetKnowledgeItem(%s) returned error: %v", id, err)
		}
		if item.Body == "" {
			t.Fatalf("expected full body for loaded ID %s", id)
		}
	}
	cited := service.CiteKnowledge(tc.Expected.CiteIDs)
	if len(cited.Missing) != 0 {
		t.Fatalf("expected no missing citations, got %#v", cited.Missing)
	}
	if len(cited.Citations) != len(tc.Expected.CiteIDs) {
		t.Fatalf("expected citations for %v, got %#v", tc.Expected.CiteIDs, cited.Citations)
	}
}

func assertCoverage(t *testing.T, got Coverage, want string) {
	t.Helper()
	if want != "" && got.Status != want {
		t.Fatalf("expected coverage %q, got %#v", want, got)
	}
}

func assertDiscoveryIDs(t *testing.T, items []DiscoveryItem, include []string, exclude []string) {
	t.Helper()
	ids := discoveryIDs(items)
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected discovery ID %s in %#v", id, items)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect discovery ID %s in %#v", id, items)
		}
	}
}

func assertTopID(t *testing.T, items []DiscoveryItem, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if len(items) == 0 || items[0].ID != want {
		t.Fatalf("expected top ID %s, got %#v", want, items)
	}
}

func assertNoDiscoveryBodies(t *testing.T, items []DiscoveryItem, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, item := range items {
		if item.Body != "" {
			t.Fatalf("discover returned body for %s", item.ID)
		}
	}
}

func assertNoMapBodies(t *testing.T, groups []MapGroup, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, group := range groups {
		for _, item := range group.Items {
			if item.Body != "" {
				t.Fatalf("map returned body for %s", item.ID)
			}
		}
	}
}

func assertNextCalls(t *testing.T, calls []RecommendedCall, require []string, forbid []string) {
	t.Helper()
	tools := map[string]bool{}
	for _, call := range calls {
		tools[call.Tool] = true
	}
	for _, tool := range require {
		if !tools[tool] {
			t.Fatalf("expected next call %s in %#v", tool, calls)
		}
	}
	for _, tool := range forbid {
		if tools[tool] {
			t.Fatalf("did not expect next call %s in %#v", tool, calls)
		}
	}
}

func assertMissingHints(t *testing.T, coverage Coverage, required bool) {
	t.Helper()
	if required && len(coverage.MissingKnowledgeHints) == 0 {
		t.Fatalf("expected missing knowledge hints in %#v", coverage)
	}
}

func assertWhyContains(t *testing.T, items []DiscoveryItem, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		found := false
		for _, item := range items {
			for _, why := range item.WhyMatched {
				if strings.Contains(why, fragment) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("expected why_matched fragment %q in %#v", fragment, items)
		}
	}
}

func assertInventoryMinimums(t *testing.T, got map[string]int, minimums map[string]int) {
	t.Helper()
	for typ, min := range minimums {
		if got[typ] < min {
			t.Fatalf("expected inventory type %s >= %d, got %#v", typ, min, got)
		}
	}
}

func assertStringIncludes(t *testing.T, got []string, include []string) {
	t.Helper()
	present := map[string]bool{}
	for _, value := range got {
		present[value] = true
	}
	for _, value := range include {
		if !present[value] {
			t.Fatalf("expected %q in %#v", value, got)
		}
	}
}

func assertMapIDs(t *testing.T, groups []MapGroup, include []string, exclude []string) {
	t.Helper()
	ids := map[string]bool{}
	for _, group := range groups {
		for _, item := range group.Items {
			ids[item.ID] = true
		}
	}
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected map ID %s in %#v", id, groups)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect map ID %s in %#v", id, groups)
		}
	}
}

func discoveryIDs(items []DiscoveryItem) map[string]bool {
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	return ids
}
```

- [ ] **Step 2: Run query golden tests and record initial failures**

Run:

```bash
go test ./internal/query -run Golden -count=1
```

Expected: FAIL if any golden expectations expose current ranking/coverage gaps. Do not weaken expectations without inspecting the failure. If the failure is a real product issue, fix `internal/query/query.go` in the smallest focused patch and rerun this command.

- [ ] **Step 3: Make minimal query changes only if golden tests expose real gaps**

If the test failure shows that strong, partial, weak, none, disclosure, or filter behavior violates the approved spec, modify only `internal/query/query.go` and existing helper tests required to explain the behavior.

Run:

```bash
go test ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit query golden tests and any required query fixes**

Run:

```bash
git add internal/query/discovery_golden_test.go internal/query/query.go internal/query/query_test.go
git commit -m "test: add query discovery golden coverage"
```

Expected: commit succeeds. If no query implementation changed, `git add` ignores unchanged files.

## Task 4: Add CLI Golden Tests

**Files:**
- Create: `internal/cli/discovery_golden_test.go`

- [ ] **Step 1: Write CLI golden tests against the shared workspace**

Create `internal/cli/discovery_golden_test.go`:

```go
package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"argos/internal/discoverytest"
	"argos/internal/query"
)

func TestGoldenCLIDiscoverMatchesQueryBehavior(t *testing.T) {
	root, _ := discoverytest.BuildIndexedWorkspace(t)
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_discover_matches_query")

	args := []string{
		"discover", "--json",
		"--project", tc.Input.Project,
		"--phase", tc.Input.Phase,
		"--task", tc.Input.Task,
		"--query", tc.Input.Query,
		"--limit", "5",
	}
	for _, file := range tc.Input.Files {
		args = append(args, "--files", file)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.DiscoveryResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse discover JSON: %v\n%s", err, stdout.String())
	}
	if result.Coverage.Status != tc.Expected.Coverage {
		t.Fatalf("expected coverage %q, got %#v", tc.Expected.Coverage, result.Coverage)
	}
	if !containsDiscoveryID(result.Items, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Items)
	}
	for _, item := range result.Items {
		if tc.Expected.NoBodies && item.Body != "" {
			t.Fatalf("CLI discover returned body for %s", item.ID)
		}
	}
}

func TestGoldenCLIMapMatchesQueryBehavior(t *testing.T) {
	root, _ := discoverytest.BuildIndexedWorkspace(t)
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_map_matches_query")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"map", "--json", "--project", tc.Input.Project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.MapResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse map JSON: %v\n%s", err, stdout.String())
	}
	for typ, min := range tc.Expected.InventoryTypesMin {
		if result.Inventory.Types[typ] < min {
			t.Fatalf("expected type %s >= %d, got %#v", typ, min, result.Inventory.Types)
		}
	}
	if !containsMapID(result.Groups, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Groups)
	}
}

func TestGoldenCLIValidationErrorsStayExplicit(t *testing.T) {
	root, _ := discoverytest.BuildIndexedWorkspace(t)
	chdir(t, root)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "discover missing task and query", args: []string{"discover", "--json", "--project", "mall-api"}, want: "discover: --task or --query is required"},
		{name: "discover bad limit", args: []string{"discover", "--json", "--project", "mall-api", "--query", "auth", "--limit", "99"}, want: "discover: --limit must be between 1 and 20"},
		{name: "map missing project", args: []string{"map", "--json"}, want: "map: --project is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("expected nonzero exit code")
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.want, stderr.String())
			}
		})
	}
}

func containsDiscoveryID(items []query.DiscoveryItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsMapID(groups []query.MapGroup, id string) bool {
	for _, group := range groups {
		for _, item := range group.Items {
			if item.ID == id {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 2: Run CLI golden tests**

Run:

```bash
go test ./internal/cli -run Golden -count=1
```

Expected: PASS. If a failure shows CLI/query drift, fix `internal/cli/cli.go` without changing `cases.json`.

- [ ] **Step 3: Commit CLI golden tests**

Run:

```bash
git add internal/cli/discovery_golden_test.go internal/cli/cli.go
git commit -m "test: add cli discovery golden coverage"
```

Expected: commit succeeds.

## Task 5: Add MCP And Adapter Interface Guards

**Files:**
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/adapters/adapters_test.go`

- [ ] **Step 1: Add MCP golden strictness tests**

In `internal/mcp/server_test.go`, add this import to the existing import block:

```go
import "argos/internal/discoverytest"
```

Add this test near the existing discovery MCP tests:

```go
func TestGoldenMCPDiscoveryStrictSchema(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	server := NewServerWithStore(store)

	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{
			name: "discover unknown argument",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth","include_inbox":true}}}`,
			want: "unknown field",
		},
		{
			name: "discover missing task and query",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api"}}}`,
			want: "task or query is required",
		},
		{
			name: "discover bad limit",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth","limit":99}}}`,
			want: "limit must be between 1 and 20",
		},
		{
			name: "map unknown argument",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api","include_inbox":true}}}`,
			want: "unknown field",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := server.HandleLine([]byte(tc.line), &out); err != nil {
				t.Fatalf("HandleLine returned error: %v", err)
			}
			text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
			if !strings.Contains(text, tc.want) {
				t.Fatalf("expected %q in %s", tc.want, text)
			}
		})
	}
}
```

- [ ] **Step 2: Add adapter direct-storage guard**

In `internal/adapters/adapters_test.go`, add this new test after
`TestRenderedAdaptersDoNotAdvertiseUnimplementedWorkflowTools`:

```go
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
	body := readFile(t, filepath.Join(root, ".cursor", "rules", "argos-mall-api.mdc"))
	for _, forbidden := range []string{"query SQLite", "query sqlite", "query vector", "knowledge_vectors", "knowledge_fts"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("adapter should not recommend direct storage access %q:\n%s", forbidden, body)
		}
	}
}
```

- [ ] **Step 3: Run MCP and adapter tests**

Run:

```bash
go test ./internal/mcp ./internal/adapters -count=1
```

Expected: PASS. If a failure exposes an actual schema or adapter wording problem, fix the production file in the smallest patch.

- [ ] **Step 4: Commit MCP and adapter guards**

Run:

```bash
git add internal/mcp/server_test.go internal/mcp/server.go internal/adapters/adapters_test.go internal/adapters/adapters.go
git commit -m "test: guard discovery interfaces"
```

Expected: commit succeeds.

## Task 6: Add AI Dogfood Checklist And Report Template

**Files:**
- Create: `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- Create: `docs/superpowers/templates/argos-discovery-dogfood-report.md`

- [ ] **Step 1: Create the AI dogfood checklist**

Create `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`:

````markdown
# Argos Discovery Dogfood Checklist

Date: 2026-04-30

## Purpose

Use this checklist to ask an AI agent to dogfood Argos Discovery without leaking
golden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the current case input, allowed tools, fixture workspace,
  and report template.
- Do not give the runner expected IDs, expected coverage, previous transcripts,
  or this design history.
- Use a separate evaluator session to compare the runner report against
  `testdata/discovery-golden/cases.json`.
- Fail the run if the runner mentions, loads, or cites a knowledge ID that did
  not appear in its tool transcript.

## Runner Prompt Template

You are validating one Argos Discovery case in a fresh context.

Workspace: `<fixture workspace path>`

Allowed flow:

1. Call `argos_context` if this looks like a workflow entrypoint.
2. Call `argos_map` if you need inventory awareness.
3. Call `argos_discover` with the case input.
4. Decide which IDs, if any, need full body loading.
5. Call `get_knowledge_item` only for selected IDs.
6. Call `cite_knowledge` only for IDs actually loaded and used.
7. Produce the report using `docs/superpowers/templates/argos-discovery-dogfood-report.md`.

Forbidden:

- Do not use prior knowledge of expected IDs.
- Do not cite IDs that were not loaded.
- Do not treat weak or none coverage as authoritative Argos guidance.
- Do not query SQLite, FTS tables, vector tables, or Markdown files directly
  unless the case explicitly validates fallback behavior.

Case input:

```json
<single case input without expected block>
```

## Evaluator Prompt Template

You are evaluating one Argos Discovery dogfood report.

Inputs:

- runner transcript
- runner report
- the matching case from `testdata/discovery-golden/cases.json`

Evaluate:

- Did actual coverage match expected coverage?
- Did discovered IDs include required IDs and exclude forbidden IDs?
- Did map/discover avoid full bodies?
- Did weak/none avoid load and citation recommendations?
- Did loaded IDs come from discovery output?
- Did cited IDs come from loaded and used knowledge?
- Did the runner show any sign of context contamination?

Return one result: `pass`, `fail`, or `review-needed`.
````

- [ ] **Step 2: Create the report template**

Create `docs/superpowers/templates/argos-discovery-dogfood-report.md`:

```markdown
# Argos Discovery Dogfood Report

Case: `<case id>`
Runner Session: `<fresh session identifier or timestamp>`
Workspace: `<fixture workspace path>`

## Inputs

- Project:
- Phase:
- Task:
- Query:
- Files:
- Filters:

## Tool Transcript Summary

- `argos_context`:
- `argos_map`:
- `argos_discover`:
- `get_knowledge_item`:
- `cite_knowledge`:

## Observed Results

- Actual coverage:
- Discovered IDs:
- Loaded IDs:
- Cited IDs:
- Missing knowledge hints:
- Next calls:

## Guards

- Progressive disclosure: `pass|fail`
- Weak/none no-overclaim: `pass|fail|not-applicable`
- Citation accountability: `pass|fail|not-applicable`
- Context contamination: `pass|fail`

## Result

Result: `pass|fail|review-needed`

Notes:
```

- [ ] **Step 3: Run documentation red-flag scan**

Run:

```bash
rg -n "TB[D]|TO[D]O|placeholde[r]|fill in late[r]" docs/superpowers/checklists docs/superpowers/templates
```

Expected: no matches.

- [ ] **Step 4: Commit dogfood docs**

Run:

```bash
git add docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md docs/superpowers/templates/argos-discovery-dogfood-report.md
git commit -m "docs: add discovery dogfood checklist"
```

Expected: commit succeeds.

## Task 7: Document How To Run Validation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add a validation section to README**

In `README.md`, after the existing Discovery section, add:

````markdown
### Discovery Validation

Discovery has a golden validation harness under `testdata/discovery-golden/`.

Run automated validation with:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp ./internal/adapters
```

The golden corpus and `cases.json` verify inventory, strong/partial/weak/none
coverage, progressive disclosure, citation guardrails, and entrypoint
consistency.

AI dogfood validation uses:

- `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- `docs/superpowers/templates/argos-discovery-dogfood-report.md`

Dogfood runners must use fresh minimal context per case. Do not give runner
agents expected IDs, expected coverage, prior transcripts, or design history.
Evaluate reports separately against `testdata/discovery-golden/cases.json`.
````

- [ ] **Step 2: Run README scan**

Run:

```bash
rg -n "Discovery Validation|discovery-golden|dogfood" README.md
```

Expected: the new section appears and points to the checklist and template.

- [ ] **Step 3: Commit README docs**

Run:

```bash
git add README.md
git commit -m "docs: document discovery validation harness"
```

Expected: commit succeeds.

## Task 8: Final Verification

**Files:**
- Verify all files changed in previous tasks.

- [ ] **Step 1: Run targeted validation tests**

Run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp ./internal/adapters -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run plan/spec red-flag scans**

Run:

```bash
rg -n "TB[D]|TO[D]O|implement late[r]|fill in detail[s]|placeholde[r]" docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md docs/superpowers/plans/2026-04-30-argos-discovery-validation-harness-implementation.md docs/superpowers/checklists docs/superpowers/templates testdata/discovery-golden
```

Expected: no matches.

- [ ] **Step 4: Inspect changed files**

Run:

```bash
git status --short
git log --oneline -8
```

Expected: working tree clean after all task commits, and recent commits include:

```text
test: add discovery golden fixtures
test: add discovery golden harness helpers
test: add query discovery golden coverage
test: add cli discovery golden coverage
test: guard discovery interfaces
docs: add discovery dogfood checklist
docs: document discovery validation harness
```

## Self-Review Checklist

- Spec coverage:
  - Golden corpus and cases: Tasks 1 and 2.
  - Automated query harness: Task 3.
  - CLI harness: Task 4.
  - MCP and adapter consistency: Task 5.
  - AI dogfood runner/evaluator and context isolation: Task 6.
  - Run instructions: Task 7.
  - Final verification: Task 8.
- No new runtime dependency is introduced; JSON uses the standard library.
- The golden fixture stays local and lightweight.
- Runner context isolation is documented as a hard dogfood requirement.
- Weak and none cases forbid full loading and citation.
- Map/discover progressive disclosure is tested through body absence.
