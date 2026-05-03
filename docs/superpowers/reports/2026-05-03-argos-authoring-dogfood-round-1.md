# Argos Authoring Dogfood Round 1

Date: 2026-05-03
Argos Commit: `4b0e023`
Status: `evaluated-fail`
Round Root: `/tmp/argos-authoring-dogfood.full.bHW2K7`

## Purpose

Run the first full fresh-runner authoring dogfood round after tightening the
authoring source-state, artifact-state, and content-readiness contract.

This round validates whether the updated public packet and proposal contract
let fresh agents naturally produce correct authoring artifacts, not whether old
runner artifacts pass under a newer evaluator.

## Runner Isolation

- One fresh runner session per case.
- Each runner received only its generated packet, the authoring report template,
  its prepared temp workspace, and the Argos binary.
- The coordinator kept golden data, evaluator code, prior reports, and hidden
  expectations out of runner context.
- Runners edited only their case workspace under the round root.

## Fixture Preparation

```bash
ROUND_ROOT=/tmp/argos-authoring-dogfood.full.bHW2K7
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
| `case-001` | `template_standard` | `pass` | `pass` | Fresh runner encoded human-stated template intent and template source state correctly. |
| `case-002` | `best_practices` | `pass` | `pass` | Synthesized Redis draft stayed advisory and inbox-scoped. |
| `case-003` | `consumer_reference` | `pass` | `pass` | Consumer-facing API/business knowledge now separates user intent and observed API facts well enough. |
| `case-004` | `observed_lesson` | `pass` | `pass` | Observed retry lesson included source evidence and structured assumptions. |
| `case-005` | `overlap` | `review-needed` | `fail` | Runner correctly stopped before candidate writing, but wrote a non-loadable proposal shape for the review-only path. |
| `case-006` | `findability` | `review-needed` | `review-needed` | Weak metadata surfaced through verification as intended. |
| `case-007` | `authz` | `pass` | `review-needed` | Runner avoided must priority and official mutation, but its self-reported result underplayed the authorization review state. |
| `case-008` | `personal_convention` | `pass` | `pass` | False-green risk remains: candidate preserves a missing-content reminder rather than actionable convention knowledge. |

## Evaluation Commands

```bash
for n in 001 002 003 004 005 006 007 008; do
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

- `pass`: 5
- `review-needed`: 2
- `fail`: 1

Runner artifacts:

- Packets: `/tmp/argos-authoring-dogfood.full.bHW2K7/packets/case-001.md`
  through `case-008.md`
- Runner reports: `/tmp/argos-authoring-dogfood.full.bHW2K7/reports/case-001.md`
  through `case-008.md`
- Evaluation JSON: `/tmp/argos-authoring-dogfood.full.bHW2K7/evaluation-case-001.json`
  through `case-008.json`

Case summaries:

- `case-001`: passed. The source-state contract now gives runners enough
  public language to mark human-designed template material and template files.
- `case-002`: passed. The runner kept synthesized Redis practices as draft,
  documented assumptions, and avoided stronger authority.
- `case-003`: passed. The runner separated consumer-facing user intent from
  observed API facts.
- `case-004`: passed. The runner recorded repository evidence and assumptions
  in canonical fields.
- `case-005`: failed. The runner chose the correct workflow behavior for
  unresolved overlap, but its proposal JSON was not canonical and could not be
  loaded by `author verify` or the evaluator.
- `case-006`: returned `review-needed`. The candidate was intentionally weak
  for retrieval metadata, and verification surfaced that review state.
- `case-007`: returned `review-needed` under evaluation. The runner did not set
  must priority or mutate official knowledge, but the runner report should have
  treated missing elevated authorization as review-needed rather than pass.
- `case-008`: passed under the current evaluator but remains a product
  false-green. The candidate says the exact convention is missing and functions
  as a reminder/review boundary, not as reusable actionable knowledge.

## Failure Classification

| Case | Classification | Rationale |
| --- | --- | --- |
| `case-005` | `product` + `harness` | The review-only workflow is valid, but the public packet does not provide enough concrete schema guidance for fresh runners to encode a canonical no-candidate proposal. The evaluator failure is accurate but too late and too generic. |
| `case-007` | `product` | The artifact respected safety gates, but runner-facing result semantics are weak: an authorization-sensitive candidate can self-report pass while the counted evaluator correctly requires review. |
| `case-008` | `product` + `case` | The current case and gate still allow a missing-content placeholder package to pass. The desired behavior should be proposal-only or review-needed until substantive convention content exists. |

## Development Decision

The core source-state tightening worked: the previously failing template,
consumer-reference, and observed-lesson scenarios now pass under fresh-runner
conditions.

Do not broaden authoring features yet. The next development slice should close
the remaining production blockers:

1. Make review-only proposals first-class in runner-facing artifacts. The
   packet should show the exact canonical fields for no-candidate proposals,
   including `proposed_shape.kind`, `proposed_shape.artifact_state`,
   `overlap_decision.decision`, empty `candidate_files`, and an empty
   `verification_plan.validate_path`.
2. Add a machine-produced proposal scaffold or schema command so runners do not
   invent nested JSON shapes for canonical fields.
3. Tighten content-readiness semantics so "we know a convention exists, but not
   what it is" cannot pass as reusable knowledge. It should become a
   proposal-only review artifact or a candidate with evaluator result
   `review-needed`, not `pass`.
4. Align runner report result guidance with evaluator semantics so
   authorization-sensitive drafts self-report `review-needed` when elevated
   authority is requested but not granted.
