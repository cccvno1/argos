# Argos Discovery Dogfood Round 6

Date: 2026-05-02
Workspace root: `/tmp/argos-dogfood-20260502-tRfnX3`
Full fixture workspace: `/tmp/argos-dogfood-20260502-tRfnX3/full`
Empty fixture workspace: `/tmp/argos-dogfood-20260502-tRfnX3/empty`
Binary: `/tmp/argos-dogfood-20260502-tRfnX3/argos`

## Goal

Validate the renamed shared-knowledge workflow after the naming audit:

- list/find/read/cite public flow;
- `support`, `usage`, `search_status`, `missing_needs`, `read_status`, and
  `next_steps` vocabulary;
- progressive reading and citation accountability;
- weak/none attribution boundaries;
- inventory-only behavior without read or cite.

## Evaluation Method

Five fresh runner sessions executed one case each. Runners received only:

- fixture workspace path;
- Argos binary path;
- allowed flow and forbidden actions;
- one case input without the expected block;
- the current report template shape.

This evaluator compared each runner report against
`testdata/discovery-golden/cases.json`.

## Cases

### list_inventory_empty

Runner: `019de5b7-92ed-7d23-8186-3725b4dc92f1`

Inputs:

- Project: `unknown-api`
- Operation: `list-empty`
- Workspace: empty fixture

Observed Results:

- Inventory was empty.
- `argos_list_knowledge` usage guidance forbade read, cite, and claim.
- `argos_find_knowledge`, `argos_read_knowledge`, and `argos_cite_knowledge`
  were not called.
- No IDs were discovered, read, or cited.

Evaluator Result: `pass`

Notes:

- Matches expected empty inventory behavior.
- The runner labeled the observed support as `none`; evaluator treated this as
  report wording, because the behavior matched inventory-only expectations and
  no task guidance was claimed.

### progressive_read_status_and_citation_guard

Runner: `019de5b7-cdbb-7ec2-b132-2168d5c92816`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add refresh token endpoint`
- Query: `refresh token session renewal`
- Files: `internal/auth/session.go`
- Limit: `5`

Observed Results:

- Actual support: `strong`
- Usage guidance: read selected items before applying; cite only read-and-used
  knowledge.
- Search status: metadata enabled, FTS enabled, semantic disabled.
- Discovered IDs included:
  - `rule:backend.auth-refresh.v1`
  - `decision:backend.session-renewal.v1`
  - `package:backend.auth-refresh.v1`
  - `lesson:backend.auth-debug.v1`
  - `reference:backend.global-refresh.v1`
- Read IDs: `rule:backend.auth-refresh.v1`
- Cited IDs: `rule:backend.auth-refresh.v1`
- Missing needs: none.

Evaluator Result: `pass`

Notes:

- Matches expected strong support, selected read, and citation accountability.
- The runner did not read or cite IDs returned only by list/find.

### partial_domain_without_task_detail

Runner: `019de5b8-170b-7342-9936-23f6b1bf0851`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `tune cache ttl for product list`
- Query: `cache ttl product list`
- Files: `internal/catalog/products.go`
- Limit: `5`

Observed Results:

- Actual support: `partial`
- Usage guidance: read allowed; cite after read and used; claims must separate
  Argos-backed guidance from general reasoning.
- Search status: metadata enabled, FTS enabled, semantic disabled.
- Discovered IDs: `reference:backend.cache-policy.v1`
- Read IDs: `reference:backend.cache-policy.v1`
- Cited IDs: `reference:backend.cache-policy.v1`
- Missing needs: task-specific product-list TTL guidance.
- Attribution boundary: runner limited Argos-backed claims to the general cache
  policy and did not claim a specific TTL value from Argos.

Evaluator Result: `pass`

Notes:

- Matches expected partial support and missing-needs boundary.
- The runner preserved citation accountability.

### weak_single_generic_term

Runner: `019de5b8-6ef0-72b1-8175-39edef75abe5`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add warehouse barcode scanner`
- Query: `barcode scanner token`
- Limit: `5`

Observed Results:

- Actual support: `weak`
- Usage guidance: read `forbidden`, cite `forbidden`, claim `forbidden`.
- Search status: metadata enabled, FTS enabled, semantic disabled.
- Discovered IDs were weakly related auth/session summaries.
- Read IDs: none.
- Cited IDs: none.
- Missing needs: barcode scanner task/query, `argos_backed=false`.
- Attribution boundary: no Argos-backed task guidance claimed.

Evaluator Result: `pass`

Notes:

- Matches expected weak support.
- Runner correctly stopped at summaries and did not read or cite.

### none_payment_webhook

Runner: `019de5b8-c3a2-7850-b559-2c7a62a3fd6c`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add payment webhook signature verification`
- Query: `payment webhook signature`
- Limit: `5`

Observed Results:

- Actual support: `none`
- Usage guidance: read `forbidden`, cite `forbidden`, claim `forbidden`.
- Search status: metadata enabled, FTS enabled, semantic disabled.
- Discovered IDs: none.
- Read IDs: none.
- Cited IDs: none.
- Missing needs: payment webhook signature verification, source `not_found`,
  `argos_backed=false`.
- Attribution boundary: runner made no Argos-backed implementation claim.

Evaluator Result: `pass`

Notes:

- Matches expected none support and no-overclaim behavior.
- Runner did not start upload, capture, or inbox creation.

## Guard Summary

- Progressive reading: 5/5 pass.
- Weak/none no-overclaim: 3/3 applicable pass.
- Citation accountability: 5/5 pass.
- Cited IDs subset of read-and-used IDs: 5/5 pass.
- Missing needs not cited: 4/4 applicable pass.
- Attribution boundary: 5/5 pass.
- No Discovery-triggered upload/capture: 5/5 pass.
- Usage guidance followed: 5/5 pass.
- Context contamination: 5/5 pass.

## Product Observations

### What Worked

- The renamed list/find/read/cite flow was understandable enough for fresh
  runners to complete the cases.
- `usage` successfully prevented read/cite/claim behavior for inventory-only,
  weak, and none cases.
- `missing_needs` were treated as unsupported needs, not citable knowledge.
- Strong and partial cases preserved read-before-cite behavior.
- Semantic search being disabled was reported without blocking lexical and
  metadata results.

### Friction

- Several runners initially tried tool-style names before correcting to CLI
  subcommands such as `knowledge list` and `knowledge find`.
- The checklist says CLI equivalents exist, but it does not show complete
  command examples for list/find/read/cite in one compact block.
- `argos_context` CLI usage was less consistent than list/find/read/cite usage;
  some runners called it without carrying all case fields.

## Decision

Next round should be a narrow dogfood-harness usability pass, not another
runtime rename:

1. Update the dogfood checklist with explicit CLI command examples for
   `context`, `knowledge list`, `knowledge find`, `knowledge read`, and
   `knowledge cite`.
2. Add a runner prompt note that MCP-style tool names are conceptual names when
   MCP is unavailable; CLI execution must use `argos knowledge ...`.
3. Re-run a broader dogfood round across the remaining golden families after
   the checklist wording is tightened.

Runtime behavior does not need a fix from this round. The observed issues are
runner-instruction ergonomics, not support/usage/read/cite semantics.
