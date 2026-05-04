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

Status: ready for write/query core

The core first-release loop is usable with CLI project registration:

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
| Write-to-query loop | pass after `argos project add` | Required registering `mall-api`; design/check/provenance publish/index/findback passed. |
| Findback with query only | pass, partial | Found `package:mall-api.redis-cache.v1`; support was `partial`. |
| Findback with phase/tags/domains | pass, strong | Found the same ID with `support.level: strong`. |

## CLI And Docs

| Check | Result | Notes |
| --- | --- | --- |
| Root usage | fixed | Removed unimplemented `new` command from help. |
| `knowledge design/check/publish/find --help` | pass | Flags match the current README and skill flow. |
| Removed write vocabulary scan | pass | No live-surface matches outside historical docs/reports/plans. |
| README registry guidance | fixed | Added `argos project add` and `argos project list --json` setup guidance. |
| README MCP tool list | fixed | Added `argos_design_knowledge` and `argos_check_knowledge`. |
| Publish status semantics | fixed | Inbox drafts must stay `draft`; `knowledge publish` now writes official knowledge as `active`. |
| `dogfood write cases --json` from repo root | pass | Works when run in the repository. |
| `dogfood write cases --json` from a normal workspace | accepted | Source-tree internal release-validation harness; not required for installed-binary knowledge writing. |

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
- Made `knowledge publish` convert published official knowledge to `status: active`, then revalidate the official path.
- Added a regression test for published package activation.
- Documented draft-to-active publish semantics in README and the capture skill.

## Release Blockers

None for the core agent-operated write/query flow.

## Decisions

- Write dogfood is a source-tree internal release-validation harness. It may
  read repository `testdata`; install-safe fixture lookup is not part of the
  normal installed-binary knowledge writing contract.

## Should Fix Before First Formal Release

- Keep README, capture skill, and CLI usage synchronized as write commands evolve.

## Later

- Improve query-only findback from newly published packages. Structured context already reaches `strong`, but natural query alone remains `partial` for the smoke package.
- Decide whether `command "new" is not implemented yet` should remain as a hidden placeholder or be removed entirely.
- Add an install-style smoke for running the binary outside the source tree if Argos will be distributed independently of the repo.
- Add optional external PR metadata ingestion and signed attestations for stricter team environments.

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
- Publish activation regression: `go test ./internal/cli -run TestRunKnowledgePublishMovesInboxPackageToOfficialPackages -count=1`
- Full regression: `go test ./... -count=1`

## Next Action

Address remaining release polish outside the core write/query contract before
the first formal release.
