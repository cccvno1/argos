# Argos Authoring Dogfood Checklist

Date: 2026-05-03

## Purpose

Use this checklist to run authoring dogfood without leaking hidden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the generated packet, `docs/superpowers/templates/argos-authoring-dogfood-report.md`, the prepared workspace path, and the Argos binary path.
- Do not give the runner `testdata/authoring-golden/cases.json`.
- Do not give the runner hidden oracle fields, prior transcripts, implementation design history, evaluator source code, or previous runner reports.
- Allow the runner to inspect and edit only the prepared temp workspace.
- Save the runner report and evaluate it from a separate coordinator context.
- Start a fresh runner with `$ROUND_ROOT/packets/case-001.md`, `docs/superpowers/templates/argos-authoring-dogfood-report.md`, `$ROUND_ROOT/case-001`, and `$ROUND_ROOT/argos`.
- Fresh runner saves the completed report at `$ROUND_ROOT/reports/case-001.md`.

## Coordinator Commands

```bash
ROUND_ROOT=$(mktemp -d /tmp/argos-authoring-dogfood.XXXXXX)
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports" "$ROUND_ROOT/case-001"
go build -o "$ROUND_ROOT/argos" ./cmd/argos
"$ROUND_ROOT/argos" dogfood authoring cases --json
cp -R testdata/authoring-golden/fixtures/full/. "$ROUND_ROOT/case-001/"
"$ROUND_ROOT/argos" dogfood authoring packet --case case-001 --workspace "$ROUND_ROOT/case-001" --argos-binary "$ROUND_ROOT/argos" > "$ROUND_ROOT/packets/case-001.md"
"$ROUND_ROOT/argos" dogfood authoring evaluate --case case-001 --report "$ROUND_ROOT/reports/case-001.md" --workspace "$ROUND_ROOT/case-001" --json
```

## Runner Requirements

- Start from the generated packet, not from repository history.
- Use the `proposal_scaffold` returned by `argos author inspect --json` as the canonical proposal shape before changing values.
- Produce canonical `authoring.proposal.v2` JSON before writing candidate knowledge.
- Use a review-only proposal when overlap, missing substantive content, or human approval blocks candidate writing.
- Write candidate knowledge only after the packet's simulated approval boundary allows it.
- Keep candidates in inbox unless the packet explicitly authorizes another boundary.
- Run `author verify --json --proposal <proposal-path> --path <candidate-path>` when a candidate is written.
- Fill every section in the authoring dogfood report template.
- Report artifact paths as workspace-relative paths, not absolute filesystem paths.

## Evaluator Handoff

Copy the evaluator JSON into the round report and classify each failure as `product`, `harness`, `runner`, `case`, or `blocked`.
- Rerun the template, consumer-reference, observed-lesson, overlap, and personal-convention cases after source/artifact/content contract changes.
- Treat missing actionable knowledge content and unapproved elevated priority as review-needed unless the runner violated a boundary.
