# Argos Authoring Scaffold Tighten Retest

Date: 2026-05-03
Branch: `authoring-scaffold-tighten`
Status: `targeted-mostly-green`
Round Root: `/tmp/argos-authoring-dogfood.tighten.CLkJyX`

## Purpose

Retest the Round 2 authoring failures after tightening proposal scaffold JSON,
runner-facing proposal guidance, consumer-audience inference, and missing-content
review-only evaluation.

## Case Matrix

| Case | Public Kind | Runner Result | Evaluation After Tighten | Notes |
| --- | --- | --- | --- | --- |
| `case-001` | `template_standard` | `pass` | `pass` | Explicit `human_review` false booleans removed the scaffold ambiguity. |
| `case-002` | `best_practices` | `pass` | `pass` | Explicit `human_review` false booleans removed the scaffold ambiguity. |
| `case-003` | `consumer_reference` | `fail` | `fail` | Final proposal and candidate verify `pass`; failure is from a runner edit-boundary mistake before the artifacts were recreated in the assigned workspace. |
| `case-008` | `personal_convention` | `review-needed` | `review-needed` | Proposal-only missing-content workflow is now accepted as review-needed instead of fail. |

## Commands

```bash
ROUND_ROOT=/tmp/argos-authoring-dogfood.tighten.CLkJyX
GOCACHE="$ROUND_ROOT/gocache" go build -o "$ROUND_ROOT/argos" ./cmd/argos

for n in 001 002 003 008; do
  "$ROUND_ROOT/argos" dogfood authoring evaluate \
    --case "case-$n" \
    --report "$ROUND_ROOT/reports/case-$n.md" \
    --workspace "$ROUND_ROOT/case-$n" \
    --json > "$ROUND_ROOT/evaluations/case-$n.after-final.json"
done
```

`case-003` final artifact verification:

```bash
cd /tmp/argos-authoring-dogfood.tighten.CLkJyX/case-003
/tmp/argos-authoring-dogfood.tighten.CLkJyX/argos author verify \
  --proposal knowledge/.inbox/proposals/mall-api/consumer-interface-reference/proposal.json \
  --path knowledge/.inbox/packages/mall-api/consumer-interface-reference \
  --json
```

Result: proposal validation `pass`, candidate validation `pass`, policy `pass`,
findability `pass`.

## Findings

- `case-001` and `case-002` validate the explicit `human_review` boolean fix.
- `case-003` validates the product behavior: the proposal audience is
  `consumer_agent`, the candidate package is verifiable, and findability is
  strong. The remaining fail should be treated as a runner execution anomaly,
  not a product regression.
- `case-008` validates the review-only product decision: when substantive
  convention content is absent and candidate writing is not approved, the system
  should preserve a proposal for review instead of creating placeholder
  knowledge.

## Development Decision

This slice is ready for normal verification and commit. A future fresh-runner
pass for `case-003` would clear the runner anomaly, but it is not required to
validate the code changes in this branch.
