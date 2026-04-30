# Argos Discovery Dogfood Checklist

Date: 2026-04-30

## Purpose

Use this checklist to ask an AI agent to dogfood Argos Discovery without leaking
golden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the current case input, allowed tools, fixture workspace,
  and report template.
- Do not give the runner expected IDs, expected coverage, previous transcripts,
  or this design history.
- Use a separate evaluator session to compare the runner report against
  `testdata/discovery-golden/cases.json`.
- Fail the run if the runner mentions, loads, or cites a knowledge ID that did
  not appear in its tool transcript.

## Runner Prompt Template

You are validating one Argos Discovery case in a fresh context.

Workspace: `<fixture workspace path>`

Allowed flow:

1. Call `argos_context` if this looks like a workflow entrypoint.
2. Call `argos_map` if you need inventory awareness.
3. Call `argos_discover` with the case input.
4. Decide which IDs, if any, need full body loading.
5. Call `get_knowledge_item` only for selected IDs.
6. Call `cite_knowledge` only for IDs actually loaded and used.
7. Produce the report using `docs/superpowers/templates/argos-discovery-dogfood-report.md`.

Forbidden:

- Do not use prior knowledge of expected IDs.
- Do not cite IDs that were not loaded.
- Do not treat weak or none coverage as authoritative Argos guidance.
- Do not query SQLite, FTS tables, vector tables, or Markdown files directly
  unless the case explicitly validates fallback behavior.

Case input:

```json
<single case input without expected block>
```

## Evaluator Prompt Template

You are evaluating one Argos Discovery dogfood report.

Inputs:

- runner transcript
- runner report
- the matching case from `testdata/discovery-golden/cases.json`

Evaluate:

- Did actual coverage match expected coverage?
- Did discovered IDs include required IDs and exclude forbidden IDs?
- Did map/discover avoid full bodies?
- Did weak/none avoid load and citation recommendations?
- Did loaded IDs come from discovery output?
- Did cited IDs come from loaded and used knowledge?
- Did the runner show any sign of context contamination?

Return one result: `pass`, `fail`, or `review-needed`.
