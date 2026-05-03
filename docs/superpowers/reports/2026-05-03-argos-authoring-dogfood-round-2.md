# Argos Authoring Dogfood Round 2

Date: 2026-05-03
Argos Commit: `e8267a5`
Status: `evaluated-fail`
Round Root: `/tmp/argos-authoring-dogfood.round2.ubhVAm`

## Purpose

Run a full fresh-runner authoring dogfood round after tightening review-only
expectations, workspace-relative artifact reporting, and the personal convention
missing-content boundary.

This round checks whether fresh agents can naturally produce correct authoring
artifacts from the current packet, proposal scaffold, report template, and temp
workspace only.

## Runner Isolation

- One fresh runner session per case.
- Each runner received only its generated packet, the authoring report template,
  its prepared temp workspace, and the Argos binary.
- The coordinator kept golden data, evaluator source, prior reports, and hidden
  expectations out of runner context.
- Runners edited only their case workspace under the round root and their
  assigned report path.

## Fixture Preparation

```bash
ROUND_ROOT=/tmp/argos-authoring-dogfood.round2.ubhVAm
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports" "$ROUND_ROOT/evaluations"
go build -o "$ROUND_ROOT/argos" ./cmd/argos

for n in 001 002 003 004 005 006 007 008; do
  mkdir -p "$ROUND_ROOT/case-$n"
  cp -R testdata/authoring-golden/fixtures/full/. "$ROUND_ROOT/case-$n/"
  "$ROUND_ROOT/argos" dogfood authoring packet \
    --case "case-$n" \
    --workspace "$ROUND_ROOT/case-$n" \
    --argos-binary "$ROUND_ROOT/argos" \
    > "$ROUND_ROOT/packets/case-$n.md"
done
```

## Case Matrix

| Case | Public Kind | Runner Result | Evaluation | Notes |
| --- | --- | --- | --- | --- |
| `case-001` | `template_standard` | `pass` | `review-needed` | Candidate verified, but proposal JSON did not mirror the complete human-review boundary. |
| `case-002` | `best_practices` | `pass` | `review-needed` | Candidate verified, but proposal JSON did not mirror the complete human-review boundary. |
| `case-003` | `consumer_reference` | `pass` | `fail` | Candidate verified, but proposal kept the scaffold's implementer audience instead of consumer-agent audience. |
| `case-004` | `observed_lesson` | `pass` | `pass` | Observed retry lesson remained stable. |
| `case-005` | `overlap` | `review-needed` | `review-needed` | Review-only path now worked with workspace-relative proposal path. |
| `case-006` | `findability` | `review-needed` | `review-needed` | Weak metadata surfaced through verification as intended. |
| `case-007` | `authz` | `review-needed` | `review-needed` | Elevated-priority request stayed in review state. |
| `case-008` | `personal_convention` | `review-needed` | `fail` | Runner correctly skipped candidate writing, but the review-only proposal used invalid enum values and did not satisfy the missing-content guard. |

## Evaluation Commands

```bash
for n in 001 002 003 004 005 006 007 008; do
  "$ROUND_ROOT/argos" dogfood authoring evaluate \
    --case "case-$n" \
    --report "$ROUND_ROOT/reports/case-$n.md" \
    --workspace "$ROUND_ROOT/case-$n" \
    --json \
    > "$ROUND_ROOT/evaluations/case-$n.json"
done
```

## Results

Overall evaluator result: `fail`

Counts:

- `pass`: 1
- `review-needed`: 5
- `fail`: 2

Case summaries:

- `case-001`: The candidate and direct `author verify` path passed. The
  counted evaluator downgraded the case because `human_review.proposal_approved`
  and the other boundary booleans were absent from proposal JSON. The runner
  placed the complete boundary under a non-contract `approval_boundary` object.
- `case-002`: Same boundary-shape problem as `case-001`. The proposal scaffold
  and packet do not make it explicit enough that proposal JSON must carry the
  same five top-level `human_review` booleans as the report.
- `case-003`: The candidate verified, but hidden requirements failed because
  the proposal retained `audience.primary=implementer_agent`. For API/business
  consumer knowledge, the product should help runners choose consumer-agent
  audience from the request and source context.
- `case-004`: Passed without additional findings.
- `case-005`: The previous absolute-path failure is closed. The runner used a
  canonical review-only proposal, no candidate, `not-run` verification, and
  workspace-relative artifact paths.
- `case-006`: Returned `review-needed` through verification, as intended for a
  findability/metadata weakness scenario.
- `case-007`: Returned `review-needed` for unapproved elevated priority and
  verification review, matching the intended boundary.
- `case-008`: The runner made the right workflow decision: proposal-only and no
  candidate. The proposal itself failed validation because it used unsupported
  enum values such as `project_personal` and `proposal_review_only`. It also
  did not trip the evaluator's missing-content guard because the current
  detection is too dependent on exact "missing + exact/actionable" wording.

## Failure Classification

| Case | Classification | Rationale |
| --- | --- | --- |
| `case-001` | `product` + `harness` | Scaffold and packet let runners omit proposal-level human-review booleans even though evaluator expects them. |
| `case-002` | `product` + `harness` | Same human-review boundary shape gap as `case-001`. |
| `case-003` | `product` | `author inspect` defaults to implementer audience and does not help runners infer consumer-agent audience for API consumer knowledge. |
| `case-008` | `product` + `harness` | Review-only workflow is correct, but enum guidance and missing-content diagnostics are not concrete enough for fresh runners. |

## Development Decision

The write-side workflow is close, but not ready to declare stable. The remaining
issues are concentrated in proposal scaffold quality and runner-facing contract
clarity, not broad authoring architecture.

Next implementation slice should be:

1. Add all five human-review boundary booleans to the canonical proposal
   scaffold and packet proposal guidance.
2. Teach `author inspect` to infer consumer-agent audience for consumer/API
   knowledge requests when the goal or context clearly points that way.
3. Tighten review-only/missing-content scaffold guidance so runners use valid
   enums and direct missing-content wording.
4. Add evaluator/validator diagnostics that name invalid proposal enum fields in
   public findings where possible.
5. Retest only `case-001`, `case-002`, `case-003`, and `case-008` before
   running another full round.
