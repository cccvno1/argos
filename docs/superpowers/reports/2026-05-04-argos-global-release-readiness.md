# Argos Global Release Readiness

Date: 2026-05-04
Updated: 2026-05-06

## Purpose

Audit Argos first-release readiness from a clean user workspace across:

- fresh workspace setup
- query and write CLI flow
- MCP tool discovery and calls
- generated adapter guidance
- agent-facing documentation

## Result

Status: ready for v0.1

Argos v0.1 is scoped as a local-first, agent-operated CLI/MCP core for durable
knowledge write and retrieval. It is ready for first release on that definition.
The release contract is:

```text
init -> validate -> index -> find none -> project add -> design -> provenance decisions -> check -> provenance verify -> publish -> index -> findback
```

MCP retrieval, MCP write design/check, generated adapters, provenance-gated
publish, and audit/status summaries are part of the v0.1 surface. Optional
semantic search, richer ranking, signed attestations, and external PR metadata
remain later work.

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
| Hidden `new` command | fixed | `argos new` now follows normal unknown-command handling. |

## MCP And Adapter Smoke

| Check | Result | Notes |
| --- | --- | --- |
| `tools/list` | pass | Exposes `argos_context`, `argos_standards`, discovery tools, `argos_design_knowledge`, and `argos_check_knowledge`. |
| MCP `argos_find_knowledge` | pass | Returned `package:mall-api.redis-cache.v1` and `support.level: strong` with structured context. |
| MCP `argos_design_knowledge` | pass | Returned write guidance and `knowledge.design.v1` template. |
| `argos install-adapters` | pass | Generated AGENTS, CLAUDE, and GEMINI guidance for `mall-api`. |
| Adapter protocol | pass | Preserves host workflow control, prefers MCP, falls back to CLI JSON or Markdown, and includes write-design gates. |

## Final V0.1 Smoke

Run date: 2026-05-06

| Check | Result | Notes |
| --- | --- | --- |
| Build public binary | pass | Built `/tmp/argos-v0.1-final/bin/argos`. |
| Repo-outside workspace | pass | Smoke workspace was `/tmp/argos-v0.1-final/workspace`. |
| Fresh setup | pass | `init`, `validate`, empty `index`, and empty `knowledge find` completed. |
| Project registration | pass | Added and listed `mall-api` through CLI commands. |
| Design and draft | pass | Wrote reviewed `knowledge.design.v1` and inbox package draft. |
| Provenance flow | pass | Recorded design, draft-write, check, and publish evidence. |
| Knowledge check | pass | `knowledge check --json` returned `result: pass`. |
| Provenance verify | pass | `provenance verify --json` returned `result: pass`. |
| Publish and index | pass | Published with `knowledge publish --provenance` and rebuilt index. |
| CLI findback | pass | `knowledge find --json` returned `package:mall-api.create-redis-cache-best-practices-for-future-backend-agents.v1`. |
| MCP findback | pass | `argos_find_knowledge` returned the same knowledge ID. |
| Adapter generation | pass | `install-adapters` completed in the smoke workspace. |
| Hidden placeholder removal | pass | `argos new` returned normal unknown-command output. |

## Fixes Made

- Removed the unimplemented `new` command from root usage.
- Added a regression assertion that root usage does not list `new`.
- Removed the hidden `new` command branch and added a regression assertion that
  `argos new` is treated as an unknown command.
- Documented project registry setup before publishing project-scoped knowledge.
- Documented write-side MCP tools in README.
- Made `knowledge publish` convert published official knowledge to `status: active`, then revalidate the official path.
- Added a regression test for published package activation.
- Documented draft-to-active publish semantics in README and the capture skill.

## Release Blockers

None for v0.1.

## Decisions

- Write dogfood is a source-tree internal release-validation harness. It may
  read repository `testdata`; install-safe fixture lookup is not part of the
  normal installed-binary knowledge writing contract.

## Should Fix Before First Formal Release

- None remaining for the v0.1 scope after the 2026-05-06 final smoke.

## Later

- Improve query-only findback from newly published packages. Structured context already reaches `strong`, but natural query alone remains `partial` for the smoke package.
- Add optional external PR metadata ingestion and signed attestations for stricter team environments.
- Add `provenance verify --all` if team CI should verify every visible
  provenance record through one command.

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
- Final v0.1 smoke root: `/tmp/argos-v0.1-final`
- Final v0.1 smoke summary: `/tmp/argos-v0.1-final/summary.json`
- Final v0.1 smoke check output: `/tmp/argos-v0.1-final/11-check.json`
- Final v0.1 smoke provenance verify output: `/tmp/argos-v0.1-final/14-provenance-verify.json`
- Final v0.1 smoke CLI findback: `/tmp/argos-v0.1-final/18-find.json`
- Final v0.1 smoke MCP findback: `/tmp/argos-v0.1-final/21-mcp-find.json`

## Next Action

Tag the v0.1 release after final repository verification.
