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
- Start a fresh runner with `/tmp/argos-authoring-dogfood/packets/case-001.md`, `docs/superpowers/templates/argos-authoring-dogfood-report.md`, `/tmp/argos-authoring-dogfood/case-001`, and `/tmp/argos-authoring-dogfood/argos`.
- Fresh runner saves the completed report at `/tmp/argos-authoring-dogfood/reports/case-001.md`.

## Coordinator Commands

```bash
mkdir -p /tmp/argos-authoring-dogfood/packets /tmp/argos-authoring-dogfood/reports /tmp/argos-authoring-dogfood/case-001
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
/tmp/argos-authoring-dogfood/argos dogfood authoring cases --json
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
/tmp/argos-authoring-dogfood/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos > /tmp/argos-authoring-dogfood/packets/case-001.md
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report /tmp/argos-authoring-dogfood/reports/case-001.md --workspace /tmp/argos-authoring-dogfood/case-001 --json
```

## Runner Requirements

- Start from the generated packet, not from repository history.
- Produce canonical `authoring.proposal.v2` JSON before writing candidate knowledge.
- Write candidate knowledge only after the packet's simulated approval boundary allows it.
- Keep candidates in inbox unless the packet explicitly authorizes another boundary.
- Run `author verify --json --proposal <proposal-path> --path <candidate-path>` when a candidate is written.
- Fill every section in the authoring dogfood report template.

## Evaluator Handoff

Copy the evaluator JSON into the round report and classify each failure as `product`, `harness`, `runner`, `case`, or `blocked`.
