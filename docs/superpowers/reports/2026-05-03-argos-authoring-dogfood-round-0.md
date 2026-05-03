# Argos Authoring Dogfood Round 0

Date: 2026-05-03
Argos Commit: `84ea0ec`
Status: `evaluated-fail`
Round Root: `/tmp/argos-authoring-dogfood.FblCEm`; extended root `/tmp/argos-authoring-dogfood.multi.akKVlQ`

## Purpose

Record the first fresh-runner authoring dogfood round after the process assets
were productized, then extend it across the public case set before choosing the
next development slice.

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

The extended run reused the same built binary and copied the same fixture seed
into isolated workspaces for `case-002` through `case-008`.

## Case Matrix

| Case | Status | Packet | Runner Report | Evaluation |
| --- | --- | --- | --- | --- |
| `case-001` | `fail` | `$ROUND_ROOT/packets/case-001.md` | `$ROUND_ROOT/reports/case-001.md` | `fail` |
| `case-002` | `pass` | `$EXTENDED_ROOT/packets/case-002.md` | `$EXTENDED_ROOT/reports/case-002.md` | `pass` |
| `case-003` | `fail` | `$EXTENDED_ROOT/packets/case-003.md` | `$EXTENDED_ROOT/reports/case-003.md` | `fail` |
| `case-004` | `fail` | `$EXTENDED_ROOT/packets/case-004.md` | `$EXTENDED_ROOT/reports/case-004.md` | `fail` |
| `case-005` | `fail` | `$EXTENDED_ROOT/packets/case-005.md` | `$EXTENDED_ROOT/reports/case-005.md` | `fail` |
| `case-006` | `review-needed` | `$EXTENDED_ROOT/packets/case-006.md` | `$EXTENDED_ROOT/reports/case-006.md` | `review-needed` |
| `case-007` | `review-needed` | `$EXTENDED_ROOT/packets/case-007.md` | `$EXTENDED_ROOT/reports/case-007.md` | `review-needed` |
| `case-008` | `pass` | `$EXTENDED_ROOT/packets/case-008.md` | `$EXTENDED_ROOT/reports/case-008.md` | `pass` |

## Evaluation Commands

```bash
"$ROUND_ROOT/argos" dogfood authoring evaluate --case case-001 --report "$ROUND_ROOT/reports/case-001.md" --workspace "$ROUND_ROOT/case-001" --json

for case_id in case-002 case-003 case-004 case-005 case-006 case-007 case-008; do
  "$EXTENDED_ROOT/argos" dogfood authoring evaluate --case "$case_id" --report "$EXTENDED_ROOT/reports/$case_id.md" --workspace "$EXTENDED_ROOT/$case_id" --json
done
```

## Results

Overall result: `fail`

Runner artifacts:

- Initial packet: `/tmp/argos-authoring-dogfood.FblCEm/packets/case-001.md`
- Initial workspace: `/tmp/argos-authoring-dogfood.FblCEm/case-001`
- Initial runner report: `/tmp/argos-authoring-dogfood.FblCEm/reports/case-001.md`
- Initial evaluation JSON: `/tmp/argos-authoring-dogfood.FblCEm/evaluation-case-001.json`
- Extended packets: `/tmp/argos-authoring-dogfood.multi.akKVlQ/packets/case-002.md` through `case-008.md`
- Extended runner reports: `/tmp/argos-authoring-dogfood.multi.akKVlQ/reports/case-002.md` through `case-008.md`
- Extended evaluation JSON: `/tmp/argos-authoring-dogfood.multi.akKVlQ/evaluation-case-002.json` through `case-008.json`

Case summaries:

- `case-001`: runner wrote a proposal and inbox candidate, kept official knowledge unchanged, did not run promote, and reported verify as `pass`; evaluation failed because the proposal did not encode user-provided template trust clearly enough.
- `case-002`: passed. The runner kept synthesized Redis guidance draft/advisory, documented assumptions, referenced existing TTL overlap, and verified the inbox candidate.
- `case-003`: visible flow passed, but evaluation failed. The proposal treated consumer-facing business framing mostly as observed or synthesized, rather than clearly separating user-stated intent from API facts.
- `case-004`: visible flow passed, but evaluation failed. The candidate said unsupported retry details were out of scope, while the proposal did not put those assumptions in the canonical structured place.
- `case-005`: runner correctly stopped before writing a candidate because overlap was unresolved, but the proposal shape was not valid for a no-candidate path and the evaluator produced many generic validation findings.
- `case-006`: returned `review-needed` with useful evidence: candidate validation passed, but verification identified weak metadata and weak findability support for the intended future task.
- `case-007`: returned `review-needed`. The runner avoided elevated priority, official mutation, and promote, but the proposal needed stronger claim-level source trust before the candidate should be accepted.
- `case-008`: passed, but product review marks it as a false green risk. The runner did not invent the missing convention; however, the system accepted a candidate whose only real content is "ask for the concrete convention later."

## Failure Classification

- `product`: authoring UX or contract made correct behavior hard.
- `harness`: packet, template, parser, fixture, or evaluator was wrong.
- `runner`: the fresh agent ignored clear instructions.
- `case`: the case expectation or fixture does not represent a useful scenario.
- `blocked`: the run could not execute because tools or workspace setup failed.

Classifications:

| Case | Classification | Rationale |
| --- | --- | --- |
| `case-001` | `product` | Source trust for human-designed material is not obvious enough in the runner-facing contract. |
| `case-003` | `product` | Business/API consumer knowledge needs first-class handling for user-stated intent versus observed source facts. |
| `case-004` | `product` | Assumptions can be written in prose while missing the structured field the machine contract relies on. |
| `case-005` | `product` + `harness` | The no-candidate review path is a real workflow, but the proposal contract and diagnostics are shaped around candidate writes. |
| `case-008` | `case` + `product` | The harness accepted a scoped reminder, but the product should not treat missing substantive convention content as ready knowledge. |

## Next Development Decision

Do not continue broad feature work yet. The next slice should tighten the
authoring contract around source state and artifact state:

1. Make source trust explicit for human-stated design, observed project facts,
   synthesized recommendations, assumptions, and missing content.
2. Add a valid no-candidate proposal path for unresolved overlap or incomplete
   content, with actionable review diagnostics instead of generic validation
   noise.
3. Add a substantive-content gate so "ask the user later" can become a design
   note or review blocker, not a passed knowledge candidate.
4. Rerun the failed and false-green cases before returning to feature
   expansion.

## Post-Fix Rerun

Date: 2026-05-03
Round Root: `/tmp/argos-authoring-tightening.Tjl59s`

This was a coordinator-assisted re-evaluation of the previous fresh-runner
reports and workspaces using the tightened binary. New runner packets were
generated under the round root, but no new fresh runner sessions were launched
for this pass.

| Case | Status | Evaluation | Notes |
| --- | --- | --- | --- |
| `case-001` | `fail` | `fail` | Source-state diagnostic now names the human-stated design versus observed template split. |
| `case-003` | `fail` | `fail` | Source-state gap remains for consumer-facing business intent versus observed API facts. |
| `case-004` | `fail` | `fail` | Assumption-state gap remains for prose-only retry limitations. |
| `case-005` | `review-needed` | `fail` | Old artifact still lacks the new review-only proposal shape; fresh runner rerun is still needed. |
| `case-008` | `pass` | `pass` | Old artifact still passes because it frames scope reminders as content; content-readiness needs a fresh runner rerun and likely a stronger semantic gate. |
