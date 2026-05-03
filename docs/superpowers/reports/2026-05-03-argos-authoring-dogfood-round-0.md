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

Packet output: `$ROUND_ROOT/packets/case-001.md`
Runner report output: `$ROUND_ROOT/reports/case-001.md`

```bash
ROUND_ROOT=$(mktemp -d /tmp/argos-authoring-dogfood.XXXXXX)
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports" "$ROUND_ROOT/case-001"
go build -o "$ROUND_ROOT/argos" ./cmd/argos
cp -R testdata/authoring-golden/fixtures/full/. "$ROUND_ROOT/case-001/"
"$ROUND_ROOT/argos" dogfood authoring packet --case case-001 --workspace "$ROUND_ROOT/case-001" --argos-binary "$ROUND_ROOT/argos" > "$ROUND_ROOT/packets/case-001.md"
```

## Case Matrix

| Case | Status | Packet | Runner Report | Evaluation |
| --- | --- | --- | --- | --- |
| `case-001` | `not-run` | `$ROUND_ROOT/packets/case-001.md` | `$ROUND_ROOT/reports/case-001.md` | `not-run` |

## Evaluation Commands

```bash
"$ROUND_ROOT/argos" dogfood authoring evaluate --case case-001 --report "$ROUND_ROOT/reports/case-001.md" --workspace "$ROUND_ROOT/case-001" --json
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
