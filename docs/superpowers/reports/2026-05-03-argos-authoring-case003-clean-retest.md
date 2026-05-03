# Argos Authoring Case 003 Clean Retest

Date: 2026-05-03
Branch: `authoring-scaffold-tighten`
Status: `pass`

## Purpose

Rerun `case-003` after the previous targeted round produced a runner-level
edit-boundary failure. The goal was to verify the consumer-reference write path
without prior-run artifacts or manual workspace repair.

## First Clean Attempt

Round Root: `/tmp/argos-authoring-case003-clean.c4y40P`

Result: `review-needed`

The runner stayed inside the assigned workspace, but the workspace had not been
seeded with the public fixture. The packet advertised `internal/api` as an
available source, while the directory was absent. The runner initialized an empty
Argos workspace and produced weaker artifacts.

Root cause: harness misuse. `dogfood authoring packet` generated instructions
for a fixture-backed workspace but did not prepare that workspace. Earlier
counted rounds relied on coordinator-side manual fixture copying.

## Harness Fix

`dogfood authoring packet` now seeds the selected public fixture into the target
workspace before emitting the runner packet. This makes the CLI entrypoint match
the counted-run workflow and removes a high-risk manual step.

## Seeded Clean Retest

Round Root: `/tmp/argos-authoring-case003-seeded.y0P19q`

Runner result:

- Proposal path: `knowledge/.inbox/proposals/mall-api/consumer-reference-proposal.json`
- Candidate path: `knowledge/.inbox/packages/mall-api/consumer-reference`
- Author verify result: `pass`
- Final runner result: `pass`
- Blocker: `none`

Evaluation:

```bash
/tmp/argos-authoring-case003-seeded.y0P19q/argos dogfood authoring evaluate \
  --case case-003 \
  --report /tmp/argos-authoring-case003-seeded.y0P19q/reports/case-003.md \
  --workspace /tmp/argos-authoring-case003-seeded.y0P19q/case-003 \
  --json
```

Result: `pass`

## Decision

`case-003` no longer indicates a product issue. The write-side consumer-reference
flow can inspect the seeded API source, infer consumer audience, write an inbox
candidate, pass author verification, and satisfy the evaluator.
