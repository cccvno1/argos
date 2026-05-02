# Argos Authoring Dogfood Round 0

Date: 2026-05-03
Argos Commit: `record-before-run`
Status: `not-run`

## Purpose

Record the first fresh-runner authoring dogfood round after the process assets are productized.

## Runner Isolation

- One fresh AI session per case.
- Runner receives only the generated packet, report template path, prepared workspace path, and Argos binary path.
- Coordinator keeps golden cases and evaluator details private.

## Fixture Preparation

Packet output: `/tmp/argos-authoring-dogfood/packets/case-001.md`
Runner report output: `/tmp/argos-authoring-dogfood/reports/case-001.md`

```bash
mkdir -p /tmp/argos-authoring-dogfood/packets /tmp/argos-authoring-dogfood/reports /tmp/argos-authoring-dogfood/case-001
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
/tmp/argos-authoring-dogfood/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos > /tmp/argos-authoring-dogfood/packets/case-001.md
```

## Case Matrix

| Case | Status | Packet | Runner Report | Evaluation |
| --- | --- | --- | --- | --- |
| `case-001` | `not-run` | `/tmp/argos-authoring-dogfood/packets/case-001.md` | `/tmp/argos-authoring-dogfood/reports/case-001.md` | `not-run` |

## Evaluation Commands

```bash
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report /tmp/argos-authoring-dogfood/reports/case-001.md --workspace /tmp/argos-authoring-dogfood/case-001 --json
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
