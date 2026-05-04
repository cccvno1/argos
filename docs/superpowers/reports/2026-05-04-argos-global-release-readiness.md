# Argos Global Release Readiness

Date: 2026-05-04

## Purpose

Audit Argos first-release readiness from a clean user workspace across:

- fresh workspace setup
- query and write CLI flow
- MCP tool discovery and calls
- generated adapter guidance
- agent-facing documentation

## Result

Status: conditional

The core first-release loop is usable after project registry setup:

```text
init -> validate -> index -> find none -> design -> write reviewed design/draft -> check -> publish -> index -> findback
```

MCP retrieval, MCP write design, and generated adapters also work. The remaining
issues are release-quality gaps rather than proof that the core model is wrong.

## Fresh Workspace Smoke

| Check | Result | Notes |
| --- | --- | --- |
| `argos init` | pass | Creates inbox, official knowledge roots, generated adapter root, and registry files. |
| `argos validate` after init | pass | Validates zero knowledge items. |
| `argos index` after init | pass | Indexes zero knowledge items. |
| Empty `knowledge find` | pass | Returns `support.level: none` and forbids Argos-backed claims. |
| Write-to-query loop | pass after registry setup | Required adding `mall-api` to `knowledge/projects.yaml`; then design/check/publish/index/findback passed. |
| Findback with query only | pass, partial | Found `package:mall-api.redis-cache.v1`; support was `partial`. |
| Findback with phase/tags/domains | pass, strong | Found the same ID with `support.level: strong`. |

## CLI And Docs

| Check | Result | Notes |
| --- | --- | --- |
| Root usage | fixed | Removed unimplemented `new` command from help. |
| `knowledge design/check/publish/find --help` | pass | Flags match the current README and skill flow. |
| Removed write vocabulary scan | pass | No live-surface matches outside historical docs/reports/plans. |
| README registry guidance | fixed | Added `knowledge/projects.yaml` setup guidance. |
| README MCP tool list | fixed | Added `argos_design_knowledge` and `argos_check_knowledge`. |
| `dogfood write cases --json` from repo root | pass | Works when run in the repository. |
| `dogfood write cases --json` from a normal workspace | should-fix | Fails because the harness reads `testdata/write-golden/cases.json` relative to cwd. |

## MCP And Adapter Smoke

| Check | Result | Notes |
| --- | --- | --- |
| `tools/list` | pass | Exposes `argos_context`, `argos_standards`, discovery tools, `argos_design_knowledge`, and `argos_check_knowledge`. |
| MCP `argos_find_knowledge` | pass | Returned `package:mall-api.redis-cache.v1` and `support.level: strong` with structured context. |
| MCP `argos_design_knowledge` | pass | Returned write guidance and `knowledge.design.v1` template. |
| `argos install-adapters` | pass | Generated AGENTS, CLAUDE, and GEMINI guidance for `mall-api`. |
| Adapter protocol | pass | Preserves host workflow control, prefers MCP, falls back to CLI JSON or Markdown, and includes write-design gates. |

## Fixes Made

- Removed the unimplemented `new` command from root usage.
- Added a regression assertion that root usage does not list `new`.
- Documented project registry setup before publishing project-scoped knowledge.
- Documented write-side MCP tools in README.

## Release Blockers

None for the core agent-operated flow after project registry setup.

## Should Fix Before First Formal Release

- Published official knowledge can still carry `status: draft`; query can still work, but JSON says `status: draft` while support wording says active project knowledge. Decide whether publish should rewrite status to `active`, require official draft status, or adjust support wording.
- `argos dogfood write cases --json` depends on repo-root `testdata` paths. If dogfood commands are part of the released binary contract, make fixture lookup independent of cwd or clearly mark dogfood as source-tree-only.
- Fresh workspace setup requires manual `knowledge/projects.yaml` editing. README now documents this, but a future `argos project add` or stronger `init` guidance would reduce agent mistakes.

## Later

- Improve query-only findback from newly published packages. Structured context already reaches `strong`, but natural query alone remains `partial` for the smoke package.
- Decide whether `command "new" is not implemented yet` should remain as a hidden placeholder or be removed entirely.
- Add an install-style smoke for running the binary outside the source tree if Argos will be distributed independently of the repo.

## Evidence

- Fresh workspace root: `/tmp/argos-global-readiness/root.txt`
- Empty find output: `/tmp/argos-global-readiness/empty-find.json`
- Write smoke output: `/tmp/argos-global-readiness/write-smoke.txt`
- Query-only findback: `/tmp/argos-global-readiness/write-findback.json`
- Structured findback: `/tmp/argos-global-readiness/write-findback-filtered.json`
- MCP tools list: `/tmp/argos-global-readiness/mcp-tools-list.json`
- MCP find call: `/tmp/argos-global-readiness/mcp-find.json`
- MCP design call: `/tmp/argos-global-readiness/mcp-design.json`
- Adapter generation output: `/tmp/argos-global-readiness/install-adapters.out`

## Next Action

Fix the `status: draft` publish/query inconsistency before opening a new product module.
