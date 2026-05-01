# Argos Discovery Dogfood Checklist

Date: 2026-04-30

## Purpose

Use this checklist to ask an AI agent to dogfood Argos Discovery without leaking
golden expectations into the runner context.

## Context Isolation Rules

- Run one case per fresh AI session.
- Give the runner only the current case input, allowed tools, fixture workspace,
  and report template.
- If the case has `"fixture": "empty"`, give the runner an empty Argos
  workspace with registry files only and no `knowledge/items` or
  `knowledge/packages` content.
- Do not give the runner expected IDs, expected support, previous transcripts,
  or this design history.
- Use a separate evaluator session to compare the runner report against
  `testdata/discovery-golden/cases.json`.
- Fail the run if the runner mentions, reads, or cites a knowledge ID that did
  not appear in its tool transcript.

## Runner Prompt Template

You are validating one Argos Discovery case in a fresh context.

Workspace: `<fixture workspace path>`

Allowed flow:

1. Call `argos_context` if this looks like a workflow entrypoint.
2. Call `argos_list_knowledge` if you need inventory awareness.
3. Call `argos_find_knowledge` with the case input.
4. Decide which IDs, if any, need full body reading.
5. Call `argos_read_knowledge` only for selected IDs.
6. Call `argos_cite_knowledge` only for IDs actually read and used.
7. Produce the report using `docs/superpowers/templates/argos-discovery-dogfood-report.md`.

Forbidden:

- Do not use prior knowledge of expected IDs.
- Do not cite IDs that were not read.
- Do not call `argos_cite_knowledge` for inventory-only cases.
- Do not cite IDs returned only by `argos_list_knowledge` or `argos_find_knowledge`.
- Do not treat weak or none support as Argos-backed guidance.
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

- Did actual support match expected support?
- Did discovered IDs include required IDs and exclude forbidden IDs?
- Did list/find avoid full bodies?
- Did weak/none avoid read and citation recommendations?
- Did read IDs come from discovery output?
- Did cited IDs come from read and used knowledge?
- For inventory-only cases, did the runner avoid citation entirely?
- Did the runner show any sign of context contamination?

Return one result: `pass`, `fail`, or `review-needed`.
