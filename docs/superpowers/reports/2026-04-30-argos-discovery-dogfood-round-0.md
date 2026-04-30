# Argos Discovery Dogfood Round 0 Report

Date: 2026-04-30

## Summary

Round 0 ran all 21 golden Discovery cases with fresh runner contexts and a
separate evaluator pass.

Automated baseline passed before the run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp ./internal/adapters -count=1
```

Fresh runner rules were followed for the dogfood run:

- one case per isolated runner context
- no runner received `cases.json` expected values
- runners used temp workspaces copied from `testdata/discovery-golden/`
- runners did not query SQLite, FTS, vector tables, or source Markdown directly
- evaluator compared observed behavior against `cases.json`

## Result

Overall result: **19 pass, 2 review-needed/fail findings**.

The core Discovery behavior is strong enough to continue, but two harness and
agent-behavior gaps should be fixed before treating dogfood as fully reliable.

## Case Results

| Case | Result | Notes |
| --- | --- | --- |
| `map_inventory_normal` | Review needed | Map inventory was correct, but the runner cited all map IDs without loading or using them. This violates citation accountability for inventory-only work. |
| `map_inventory_empty` | Review needed | The case input used `project: unknown-api`, but global knowledge is visible to every project, so the map was not empty. This is a case design mismatch, not an implementation failure. |
| `map_hides_deprecated_by_default` | Pass | Deprecated item was hidden by default and visible only with `include_deprecated`. |
| `map_global_knowledge_visible` | Pass | Global knowledge appeared for `mall-api`; no full bodies loaded. |
| `strong_auth_refresh_full_signal` | Pass | Strong coverage, correct auth/session IDs, controlled loading, citation accountability passed. |
| `strong_auth_refresh_query_only` | Pass | Query-only discovery returned strong refresh/session guidance. |
| `strong_auth_refresh_task_only` | Pass | Task-only discovery returned strong refresh guidance and loaded only selected IDs. |
| `strong_file_scope_beats_generic` | Pass | File-scoped auth rule ranked above generic backend rule. |
| `partial_domain_without_task_detail` | Pass | Cache reference was found with partial coverage and missing hints. Runner marked review-needed, but evaluator confirmed golden expectations were satisfied. |
| `partial_lesson_without_rule` | Pass | Lesson surfaced as partial guidance with missing hints; citation used only the loaded lesson. |
| `partial_package_entrypoint_without_detail` | Pass | Package entrypoint surfaced with matched sections; package asset was not standalone. |
| `weak_single_generic_term` | Pass | Weak coverage did not load or cite any knowledge. |
| `weak_broad_tag_only` | Pass | Broad backend tag produced weak coverage only; no load or citation. |
| `none_payment_webhook` | Pass | No match, no load, no citation. |
| `none_project_scope_mismatch` | Pass | Warehouse project knowledge did not leak into `mall-api`. |
| `none_explicit_filter_excludes_match` | Pass | Explicit nonmatching `payments` tag excluded refresh-token matches. |
| `progressive_disclosure_and_citation_guard` | Pass | Full bodies were loaded only through `get_knowledge_item`; citations matched loaded/used IDs. |
| `interface_cli_discover_matches_query` | Pass | CLI discover returned strong coverage and expected IDs. |
| `interface_cli_map_matches_query` | Pass | CLI map returned expected inventory and package. |
| `interface_mcp_strict_schema` | Pass | Unknown args, missing task/query, and bad limits were rejected. |
| `adapter_flow_recommendations` | Pass | Generated adapters recommend MCP, CLI JSON, adapter files, and Markdown fallback; no direct SQLite/vector path. |

## Findings

### 1. `map_inventory_empty` Does Not Actually Model An Empty Inventory

The dogfood runner copied the normal golden corpus and called `argos_map` with
`project: unknown-api`. The result included `reference:backend.global-refresh.v1`
because global knowledge has no project list and is intentionally visible to all
project work.

This means the case name and expectation are misleading. The implementation is
behaving consistently with the current global-knowledge rule.

Recommended fix:

- Replace this dogfood case with a true empty-corpus workspace, or
- Rename it to `map_unknown_project_shows_global_knowledge` and add a separate
  empty-index case outside the normal corpus.

### 2. Runner Misused `cite_knowledge` In `map_inventory_normal`

The map inventory runner listed correct inventory, did not load any full bodies,
but then called `cite_knowledge` for all map IDs.

That violates the dogfood rule:

```text
cite_knowledge only for IDs actually loaded and used
```

This did not expose a query/MCP bug, because `cite_knowledge` currently returns
metadata for valid IDs and cannot know whether a runner actually used the
knowledge. It does expose an agent-behavior risk and a harness-instruction gap.

Recommended fix:

- Strengthen the dogfood checklist: inventory-only cases must not call
  `cite_knowledge`.
- Add evaluator wording that any citation for a non-loaded ID is a case failure.
- Consider whether final adapter instructions should say "Do not cite map or
  discover results unless you loaded and applied the item."

## Product Assessment

The route is not off track.

Discovery now demonstrates the important behavior:

- agents can inventory the knowledge base
- agents can route task/query/file context to relevant IDs
- strong, partial, weak, and none states behave distinctly
- weak/none avoids loading and citation
- project scope and explicit filters are respected
- package entrypoints remain progressive
- CLI, MCP, and adapters are aligned

The two findings are worth fixing, but they are harness/dogfood reliability
issues rather than core Discovery architecture failures.

Recommended next step: **fix the two Round 0 findings, then proceed to the next
Discovery milestone instead of adding vector search now.**

