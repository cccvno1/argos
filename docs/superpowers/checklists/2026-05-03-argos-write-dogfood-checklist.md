# Argos Write Dogfood Checklist

Date: 2026-05-03

## Purpose

Use this checklist to run write dogfood without leaking hidden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the generated packet, `docs/superpowers/templates/argos-write-dogfood-report.md`, the prepared workspace path, and the Argos binary path.
- Do not give the runner `testdata/write-golden/cases.json`.
- Do not give the runner hidden oracle fields, prior transcripts, implementation design history, evaluator source code, or previous runner reports.
- Allow the runner to inspect and edit only the prepared temp workspace.
- Save the runner report and evaluate it from a separate coordinator context.
- Start a fresh runner with `$ROUND_ROOT/packets/case-001.md`, `docs/superpowers/templates/argos-write-dogfood-report.md`, `$ROUND_ROOT/case-001`, and `$ROUND_ROOT/argos`.
- Fresh runner saves the completed report at `$ROUND_ROOT/reports/case-001.md`.

## Coordinator Commands

```bash
ROUND_ROOT=$(mktemp -d /tmp/argos-write-dogfood.XXXXXX)
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports"
go build -o "$ROUND_ROOT/argos" ./cmd/argos
"$ROUND_ROOT/argos" dogfood write cases --json
"$ROUND_ROOT/argos" dogfood write packet --case case-001 --workspace "$ROUND_ROOT/case-001" --argos-binary "$ROUND_ROOT/argos" > "$ROUND_ROOT/packets/case-001.md"
"$ROUND_ROOT/argos" dogfood write evaluate --case case-001 --report "$ROUND_ROOT/reports/case-001.md" --workspace "$ROUND_ROOT/case-001" --json
```

The packet command seeds the selected public fixture into `$ROUND_ROOT/case-001`
before printing the runner packet.

## Runner Requirements

- Start from the generated packet, not from repository history.
- Use `argos knowledge design --json --project <project> --intent <intent>` as the canonical write guidance source.
- Write canonical `knowledge.design.v1` JSON before writing draft knowledge.
- Use a design-only artifact when overlap, missing substantive content, or review approval blocks draft writing.
- Write draft knowledge only after the packet's simulated approval boundary allows it.
- Keep drafts in inbox unless the packet explicitly authorizes another boundary.
- Run `argos knowledge check --json --design <design-path> --draft <draft-path>` when a draft is written.
- Fill every section in the write dogfood report template.
- Report artifact paths as workspace-relative paths, not absolute filesystem paths.

## Evaluator Handoff

Copy the evaluator JSON into the round report and classify each failure as `product`, `harness`, `runner`, `case`, or `blocked`.
- Rerun the template, consumer-reference, observed-lesson, overlap, and personal-convention cases after source/design/content contract changes.
- Treat missing actionable knowledge content and unapproved elevated priority as review-needed unless the runner violated a boundary.
