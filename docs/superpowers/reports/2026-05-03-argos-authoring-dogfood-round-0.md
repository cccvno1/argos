# Argos Authoring Dogfood Round 0

Date: 2026-05-03
Status: `not-run`

## Purpose

Record the first fresh-runner authoring dogfood round after the process assets are productized.

## Runner Isolation

- One fresh AI session per case.
- Runner receives only the generated packet, report template path, prepared workspace path, and Argos binary path.
- Coordinator keeps golden cases and evaluator details private.

## Fixture Preparation

```bash
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
mkdir -p /tmp/argos-authoring-dogfood/case-001
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
```

## Case Matrix

| Case | Status | Runner Report | Evaluation |
| --- | --- | --- | --- |
| `case-001` | `not-run` | `docs/superpowers/reports/authoring-round-0-case-001.md` | `not-run` |

## Evaluation Commands

```bash
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report docs/superpowers/reports/authoring-round-0-case-001.md --workspace /tmp/argos-authoring-dogfood/case-001 --json
```

## Results

No fresh-runner report has been evaluated yet.

## Failure Classification

- `product`: authoring UX or contract made correct behavior hard.
- `harness`: packet, template, parser, fixture, or evaluator was wrong.
- `runner`: the fresh agent ignored clear instructions.
- `case`: the case expectation or fixture does not represent a useful scenario.
- `blocked`: the run could not execute because tools or workspace setup failed.

## Next Development Decision

Run `case-001` with a fresh runner using the generated packet and authoring report template, then evaluate the saved report.
