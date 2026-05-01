# Argos Discovery Dogfood Round 7

Date: 2026-05-02
Workspace root: `/tmp/argos-dogfood-20260502-round7-yN3KaP`
Full fixture workspace: `/tmp/argos-dogfood-20260502-round7-yN3KaP/full`
Empty fixture workspace: `/tmp/argos-dogfood-20260502-round7-yN3KaP/empty`
Binary: `/tmp/argos-dogfood-20260502-round7-yN3KaP/argos`

## Goal

Re-run a broader dogfood slice after tightening the checklist with explicit CLI
equivalents for MCP tool names.

This round checks whether fresh runners can execute list/find/read/cite without
tool-name confusion, and whether the renamed workflow holds across inventory,
deprecated filtering, strong matching, partial matching, none support, CLI,
MCP, and generated adapters.

## Checklist Change Under Test

The runner checklist now states that MCP names such as `argos_find_knowledge`
are conceptual when MCP is unavailable, and provides direct CLI examples:

- `argos context --json ...`
- `argos knowledge list --json ...`
- `argos knowledge find --json ...`
- `argos knowledge read --json <id>`
- `argos knowledge cite --json <id>...`

## Cases

### list_inventory_normal

Runner: `019de5c4-c750-7582-ab6a-733a423590a0`

Inputs:

- Project: `mall-api`
- Operation: `list`

Observed Results:

- `knowledge list --json --project mall-api` succeeded.
- Usage guard forbade read, cite, and claim from inventory output.
- Inventory counts:
  - `decision`: 1
  - `lesson`: 1
  - `package`: 1
  - `reference`: 2
  - `rule`: 2
- Domains included `backend` and `security`.
- No read or cite command was run.

Evaluator Result: `pass`

Notes:

- The explicit CLI examples avoided the tool-name retry seen in round 6.
- The runner loaded a local startup skill due session instructions, but did not
  inspect Argos cases, source, index tables, or Markdown knowledge files.

### list_hides_deprecated_by_default

Runner: `019de5c4-f07c-7653-8639-e992a28a0531`

Inputs:

- Project: `mall-api`
- Operation: `list`

Observed Results:

- Default `knowledge list --json --project mall-api` returned only active
  surfaced items.
- Read, cite, and claim were forbidden for list inventory.
- No read or cite command was run.

Evaluator Result: `pass-with-scope-note`

Notes:

- The runner verified default hiding behavior.
- It did not run the optional `--include-deprecated` follow-up, so this round
  does not fully validate deprecated visibility when explicitly requested.

### strong_file_scope_beats_generic

Runner: `019de5c5-2b24-7560-8aa0-51bf7595316f`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add backend auth handler`
- Query: `backend auth handler`
- Files: `internal/auth/handler.go`
- Limit: `5`

Observed Results:

- Support: `strong`
- Confidence: `0.7360001406827142`
- Top result: `rule:backend.auth-refresh.v1`
- Runner read and cited `rule:backend.auth-refresh.v1`.
- No weak or unread IDs were treated as guidance.

Evaluator Result: `pass-with-report-quality-note`

Notes:

- Behavior matched the expected file-scope preference.
- The report was shorter than the template and did not include the full guard
  matrix. Future automated harness work should enforce report shape.

### partial_lesson_without_rule

Runner: `019de5c6-2e45-7892-8345-894bf1f028cf`

Inputs:

- Project: `mall-api`
- Phase: `debugging`
- Task: `debug session renewal test failure`
- Query: `session renewal tests fail logs`
- Limit: `5`

Observed Results:

- Support: `partial`
- Confidence: `0.7555`
- Search: metadata and FTS enabled; semantic disabled.
- Found IDs included:
  - `lesson:backend.auth-debug.v1`
  - `decision:backend.session-renewal.v1`
  - `rule:backend.auth-refresh.v1`
- Read IDs: `lesson:backend.auth-debug.v1`
- Cited IDs: `lesson:backend.auth-debug.v1`
- Missing needs remained non-Argos-backed.

Evaluator Result: `pass`

Notes:

- Runner selected the high-confidence lesson, read before use, cited only that
  ID, and preserved the partial-support caveat.

### partial_package_entrypoint_without_detail

Runner: `019de5c6-612f-7ea2-8aa1-868bcf5279d5`

Inputs:

- Project: `mall-api`
- Phase: `planning`
- Task: `understand auth refresh package`
- Query: `auth refresh package examples`
- Types: `package`
- Limit: `5`

Observed Results:

- Support: `partial`
- Confidence: `0.7`
- Returned IDs: `package:backend.auth-refresh.v1`
- Read IDs: `package:backend.auth-refresh.v1`
- Cited IDs: `package:backend.auth-refresh.v1`
- Missing needs indicated example details were not fully covered by the package
  entrypoint.

Evaluator Result: `pass`

Notes:

- This confirms package entrypoints are useful without overclaiming detailed
  example coverage.

### none_project_scope_mismatch

Runner: `019de5c6-b491-7040-b266-d6b3b856806e`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `update warehouse picking flow`
- Query: `warehouse picking scanner`
- Limit: `5`

Observed Results:

- Support: `none`
- Confidence: `0`
- Usage forbade read, cite, and claim.
- IDs returned: none.
- Missing needs source: `wrong_scope`
- No read or cite command was run.

Evaluator Result: `pass`

Notes:

- Argos correctly refused to use wrong-project knowledge as Mall API support.

## Interface Checks

### CLI read/cite

Local command checks:

- `argos knowledge read --json rule:backend.auth-refresh.v1`
- `argos knowledge cite --json rule:backend.auth-refresh.v1`

Result: `pass`

The read command returned the full body only for the requested ID. The cite
command returned the active citation metadata for the requested ID.

### MCP strict schema

Local JSON-RPC checks:

- `tools/list` exposed `argos_find_knowledge`, `argos_list_knowledge`,
  `argos_read_knowledge`, and `argos_cite_knowledge`.
- `argos_find_knowledge` rejected unknown `include_inbox`.
- `argos_find_knowledge` rejected missing task/query.
- `argos_find_knowledge` rejected `limit=99`.

Result: `pass`

### Generated adapters

Local adapter check:

- `argos install-adapters`
- scanned generated adapter files for MCP-first, CLI JSON fallback, generated
  adapter files, Markdown fallback, list/find/read/cite guidance, and absence of
  direct SQLite/vector-query advice.

Result: `pass`

## Guard Summary

- Inventory-only read/cite prevention: pass.
- Default deprecated hiding: pass.
- Strong file-scope ranking: pass.
- Partial support attribution boundary: pass.
- Package entrypoint partial-support boundary: pass.
- Wrong-scope none support: pass.
- CLI read/cite interface: pass.
- MCP strict schema: pass.
- Generated adapter protocol: pass.
- CLI naming friction from round 6: improved.

## Product Observations

### What Improved

- Fresh runners no longer tried to execute MCP tool names as CLI subcommands
  when given explicit CLI equivalents.
- list/find/read/cite behavior remained consistent across broader cases.
- `usage` continued to prevent inventory-only, weak, and none overclaiming.

### Remaining Friction

- Some runner reports still drift from the full report template. The current
  human evaluator can handle this, but a repeatable harness should enforce
  structured output.
- Deprecated visibility needs a stronger two-step runner instruction or an
  automated evaluator check for `--include-deprecated`.
- Interface checks are still partly controller-run rather than fresh-runner
  executed, which is acceptable for this round but not ideal as a productized
  harness.

## Decision

Round 7 supports the same conclusion as round 6, with more confidence:

- Runtime shared-knowledge behavior is ready enough for the next development
  slice.
- The next slice should productize the dogfood harness rather than continue
  hand-running prompt/evaluator rounds.
- The first productization target should be structured runner packets and
  evaluator summaries, so future runtime work has a reliable validation loop.
