# Argos Authoring Dogfood Round 0

Date: 2026-05-03
Argos Commit: `84ea0ec`
Status: `evaluated-fail`
Round Root: `/tmp/argos-authoring-dogfood.FblCEm`

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
| `case-001` | `fail` | `$ROUND_ROOT/packets/case-001.md` | `$ROUND_ROOT/reports/case-001.md` | `fail` |

## Evaluation Commands

```bash
"$ROUND_ROOT/argos" dogfood authoring evaluate --case case-001 --report "$ROUND_ROOT/reports/case-001.md" --workspace "$ROUND_ROOT/case-001" --json
```

## Results

Overall result: `fail`

Runner artifacts:

- Packet: `/tmp/argos-authoring-dogfood.FblCEm/packets/case-001.md`
- Workspace: `/tmp/argos-authoring-dogfood.FblCEm/case-001`
- Runner report: `/tmp/argos-authoring-dogfood.FblCEm/reports/case-001.md`
- Evaluation JSON: `/tmp/argos-authoring-dogfood.FblCEm/evaluation-case-001.json`

Evaluator output:

```json
{
  "case_id": "case-001",
  "result": "fail",
  "findings": [
    {
      "severity": "fail",
      "message": "proposal evidence does not satisfy hidden evaluation requirements"
    }
  ]
}
```

The fresh runner completed the visible flow: it wrote a proposal, wrote an inbox
candidate package, kept official knowledge unchanged, did not run promote, and
reported `author verify` as `pass`.

The failure is that the proposal did not distinguish user-provided confirmation
from observed template evidence strongly enough for the hidden evaluator.

## Failure Classification

- `product`: authoring UX or contract made correct behavior hard.
- `harness`: packet, template, parser, fixture, or evaluator was wrong.
- `runner`: the fresh agent ignored clear instructions.
- `case`: the case expectation or fixture does not represent a useful scenario.
- `blocked`: the run could not execute because tools or workspace setup failed.

## Next Development Decision

Classify `case-001` as `product`.

Next development slice: make the authoring proposal contract and runner packet
clearer about source trust for user-designed or user-confirmed material. A
fresh runner should know how to represent a human-provided template separately
from repository-observed facts before it writes candidate knowledge.
