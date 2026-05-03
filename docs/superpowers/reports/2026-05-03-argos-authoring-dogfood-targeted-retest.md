# Argos Authoring Dogfood Targeted Retest

Date: 2026-05-03
Argos Commit: `16c2884`
Status: `evaluated-fail`
Round Root: `/tmp/argos-authoring-dogfood.targeted.6ZdVf4`

## Purpose

Retest the three Round 1 problem cases after adding proposal scaffolds,
runner-facing review-only guidance, content-readiness diagnostics, and
authorization-sensitive result semantics.

## Runner Isolation

- One fresh runner session per case.
- Runners received only their generated packet, the report template, their temp
  workspace, and the Argos binary.
- Coordinator kept golden data, evaluator source, prior reports, and hidden
  expectations out of runner context.

## Case Matrix

| Case | Public Kind | Runner Result | Evaluation | Notes |
| --- | --- | --- | --- | --- |
| `case-005` | `overlap` | `review-needed` | `fail` | Runner used review-only shape and skipped candidate writing, but reported an absolute proposal path. |
| `case-007` | `authz` | `review-needed` | `review-needed` | Runner kept candidate as non-official draft, avoided priority must, and self-reported review-needed. |
| `case-008` | `personal_convention` | `review-needed` | `fail` | Runner correctly produced review-only proposal because exact convention content was missing; evaluator still expects candidate-shape evidence. |

## Evaluation Commands

```bash
ROUND_ROOT=/tmp/argos-authoring-dogfood.targeted.6ZdVf4

for n in 005 007 008; do
  "$ROUND_ROOT/argos" dogfood authoring evaluate \
    --case "case-$n" \
    --report "$ROUND_ROOT/reports/case-$n.md" \
    --workspace "$ROUND_ROOT/case-$n" \
    --json \
    > "$ROUND_ROOT/evaluation-case-$n.json"
done
```

## Results

Overall evaluator result: `fail`

Counts:

- `pass`: 0
- `review-needed`: 1
- `fail`: 2

Case summaries:

- `case-005`: behavior improved. The runner copied the canonical review-only
  shape, wrote a loadable proposal-only artifact, skipped candidate writing,
  and marked verify as `not-run`. The evaluator failed before loading the
  proposal because the report used an absolute proposal path. This is a harness
  guidance gap: packets and templates should explicitly require workspace-
  relative artifact paths in reports.
- `case-007`: behavior improved and evaluator matched the intended result. The
  runner no longer self-reported `pass` for an elevated-priority request when
  priority must was not authorized.
- `case-008`: product behavior improved but the case expectation is now stale.
  The runner correctly treated missing exact convention content as review-only
  and did not write a placeholder candidate. The evaluator failed because the
  existing case still expects a candidate and user-confirmed candidate evidence.

## Failure Classification

| Case | Classification | Rationale |
| --- | --- | --- |
| `case-005` | `harness` | Correct runner behavior was rejected because the report path was absolute. The packet/template need stronger artifact path wording and possibly path normalization diagnostics. |
| `case-008` | `case` + `product` | The intended behavior changed from "write draft placeholder candidate" to "review-only until substantive content exists". The golden case and hidden checks need to encode that new product decision. |

## Development Decision

The implementation direction is validated:

1. `proposal_scaffold` and review-only packet guidance improved runner behavior.
2. Result semantics fixed the authorization-sensitive self-report problem.
3. Missing-content guidance moved the runner away from placeholder candidate
   writing.

The next slice should be small and targeted:

1. Require workspace-relative artifact paths in runner reports and packet text.
2. Update the personal-convention case so missing substantive content expects
   review-only/no-candidate behavior.
3. Add evaluator coverage for the accepted missing-content review-only path.
4. Rerun `case-005` and `case-008`.
