# Argos Discovery Dogfood Round 8

Date: 2026-05-02
Workspace root: `/tmp/argos-dogfood-20260502-round8-CCdPmM`
Binary: `/tmp/argos-dogfood-20260502-round8-CCdPmM/argos`

## Goal

Use the productized dogfood harness with fresh isolated runners before
resuming feature development. This round focused on whether public runner
packets, saved reports, and `argos dogfood evaluate` form a usable loop without
giving runners oracle values or source-level context.

## Runner Set

| Case | Surface | Result |
| --- | --- | --- |
| `case-001` | list inventory | pass |
| `case-002` | empty inventory | pass |
| `case-009` | partial find | pass |
| `case-017` | progressive find/read/cite workflow | pass |
| `case-020` | CLI read | pass |
| `case-021` | CLI cite | pass |
| `case-023` | adapter install | pass |

Reports were written under:

`/tmp/argos-dogfood-20260502-round8-CCdPmM/reports`

## Findings

The first evaluator pass exposed two harness defects rather than Discovery
behavior defects.

1. Real runner reports used natural Markdown shapes that the parser did not
   accept: `# Case: ...`, a bare `pass` under `## Result`, guard lines like
   `pass; reason`, and list fields like `none; reason`.
2. The CLI read/cite packets did not include a concrete task or query, so a
   fresh runner could not perform task-specific public discovery before reading
   or citing.

Both issues matter because they would make honest fresh-runner evidence look
like product failure, or force runners toward inventory-only behavior.

## Changes Made

- `internal/dogfood/report.go`
  - accepts `# Case: ...` labels;
  - accepts bare result statuses in the `## Result` section;
  - normalizes leading guard statuses with explanatory suffixes;
  - treats `none; reason` list fields as empty lists;
  - classifies natural support descriptions such as inventory/strong/partial.
- `testdata/discovery-golden/cases.json`
  - gives the CLI read/cite public cases concrete task/query input so fresh
    runners can discover IDs from public command output.
- `internal/dogfood/dogfood_test.go`
  - adds regression coverage for fresh-runner Markdown style;
  - adds regression coverage that CLI read/cite packets include task/query
    input for fresh discovery.

## Verification

Commands run:

```bash
go test ./... -count=1
/tmp/argos-dogfood-20260502-round8-CCdPmM/argos dogfood evaluate --case <case> --report /tmp/argos-dogfood-20260502-round8-CCdPmM/reports/<case>.md --json
```

Final evaluator results:

| Case | Evaluator result |
| --- | --- |
| `case-001` | pass |
| `case-002` | pass |
| `case-009` | pass |
| `case-017` | pass |
| `case-020` | pass |
| `case-021` | pass |
| `case-023` | pass |

## Decision

Round 8 is a complete harness-level tightening pass. The dogfood loop now has
enough evidence to support the next development slice: build product behavior
again, while keeping fresh-runner dogfood as the regression gate for Discovery
surfaces.

Next recommended development direction: resume feature work above the discovery
core, with the harness treated as a required gate for new retrieval, adapter, or
capture/upload behavior.
