# Argos Next Development Slice Decision

Date: 2026-05-02

## Context

The naming audit is merged locally and the renamed shared-knowledge workflow has
passed two dogfood rounds:

- Round 6: five fresh runner sessions across inventory, strong, partial, weak,
  and none support.
- Round 7: broader inventory, deprecated, strong file-scope, partial lesson,
  partial package, wrong-scope none, CLI, MCP, and adapter checks.

Runtime behavior held. The repeated friction is not support/usage semantics; it
is dogfood execution ergonomics and report consistency.

## Options

### 1. Productize the dogfood harness

Build a repeatable local workflow for generating runner packets, collecting
structured reports, and evaluating them against golden expectations.

This makes future runtime changes safer because dogfood stops depending on
manual prompt assembly and ad hoc evaluator judgment.

### 2. Tighten `argos_context`

Improve context request handling and runner guidance so agents carry project,
phase, task, and files through consistently before list/find/read/cite.

This is useful, but it should ride on top of a more repeatable dogfood harness.

### 3. Start semantic search or ranking enhancement work

Move into higher-value retrieval work, such as semantic provider integration or
ranking improvements.

This is strategically important, but it is riskier before the dogfood harness is
repeatable.

## Decision

Choose option 1 next: productize the dogfood harness.

## Recommended First Slice

Keep the first implementation slice small:

1. Add a command or test helper that emits one runner packet per selected golden
   case without expected values.
2. Add a structured report schema or parser for runner output.
3. Add an evaluator helper that compares runner reports with
   `testdata/discovery-golden/cases.json`.
4. Preserve human-in-the-loop execution for fresh AI sessions; do not attempt
   full AI automation in the first slice.

## Success Criteria

- A human can generate fresh runner prompts without hand-copying case inputs.
- Runner reports can be checked for missing sections and guard fields.
- Evaluator output clearly marks `pass`, `fail`, or `review-needed`.
- Existing golden tests and dogfood reports remain readable.
- The harness supports the current list/find/read/cite vocabulary and does not
  reintroduce retired terminology.
