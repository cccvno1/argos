# Argos Authoring Dogfood Productization Design

Date: 2026-05-03

## Purpose

Build a complete authoring dogfood workflow that works like the existing
discovery dogfood loop: a coordinator can prepare an isolated case, hand a
fresh runner only public inputs, collect a structured report, evaluate it with
Argos, and record the round outcome for the next development decision.

This is not another hand-run smoke test. The goal is a repeatable process for
testing whether agents can design and write durable Argos knowledge for future
agent work.

## Background

The authoring v2 slice added the core harness:

```bash
argos dogfood authoring cases --json
argos dogfood authoring packet --case <case-handle> --workspace <fixture> --argos-binary <argos>
argos dogfood authoring evaluate --case <case-handle> --report <report.md> --workspace <fixture> --json
```

It also added natural authoring cases, public runner packets, a report parser,
workspace-aware evaluation, and hidden oracle enforcement.

What is still missing is the outer workflow that made discovery dogfood useful:

- a coordinator checklist;
- a report template aligned with the parser;
- a fixture preparation convention;
- a fresh-runner handoff protocol;
- a round report shape;
- consistency tests that keep docs, packet text, parser fields, and CLI commands
  from drifting apart.

## Non-Goals

- Do not automate AI runner execution.
- Do not call a model from Argos.
- Do not expand authoring evaluator semantics unless template/packet
  consistency requires a small parser alias.
- Do not change proposal generation, `author verify`, ranking, indexing, MCP, or
  promotion behavior.
- Do not make the current coordinator session count as a valid dogfood runner.
- Do not expose hidden oracle fields, internal case IDs, or expected results to
  runner packets or checklist instructions.

## Design Principles

### Fresh Runner Or It Does Not Count

A dogfood result only counts when the runner starts from the generated packet,
the report template, the prepared workspace, and the Argos binary. A coordinator
who already knows the harness design may prepare fixtures and evaluate reports,
but should not author the candidate for a counted run.

### Public Workspace, Hidden Oracle

The prepared workspace is public runner context. It may include registry files,
source material, templates, and official knowledge needed for overlap checks.
`testdata/authoring-golden/cases.json` and hidden oracle fields remain private
to the evaluator.

### Process Assets Are Part Of The Harness

The checklist and report template are not decorative documentation. They are
versioned harness artifacts and should be tested against the packet renderer and
report parser.

### Failures Drive The Next Slice

Round reports should classify failures as one of:

- `product`: authoring UX or contract made the correct behavior hard;
- `harness`: packet, template, parser, fixture, or evaluator was wrong;
- `runner`: the fresh agent ignored clear instructions;
- `case`: the case expectation or fixture does not represent a useful scenario;
- `blocked`: the run could not execute because tools or workspace setup failed.

## Artifacts To Add

### Authoring Checklist

Create:

```text
docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md
```

The checklist should mirror the discovery checklist, but use authoring-specific
rules:

- run one case per fresh AI session;
- give the runner only the packet, the authoring report template, the prepared
  workspace path, and the Argos binary path;
- do not give the runner hidden oracle fields, internal case IDs, prior
  transcripts, this design history, or `testdata/authoring-golden/cases.json`;
- copy a fixture seed into a temp workspace before each case;
- allow regular workspace file edits only inside that temp workspace;
- require canonical `authoring.proposal.v2` JSON before candidate writing;
- require inbox candidate writes unless the packet explicitly authorizes another
  boundary;
- require `author verify --json` when a candidate is written;
- save the runner report and evaluate it in a separate coordinator context.

It should include exact command examples:

```bash
go build -o /tmp/argos-authoring-dogfood/argos ./cmd/argos
mkdir -p /tmp/argos-authoring-dogfood/case-001
cp -R testdata/authoring-golden/fixtures/full/. /tmp/argos-authoring-dogfood/case-001/
/tmp/argos-authoring-dogfood/argos dogfood authoring packet --case case-001 --workspace /tmp/argos-authoring-dogfood/case-001 --argos-binary /tmp/argos-authoring-dogfood/argos
/tmp/argos-authoring-dogfood/argos dogfood authoring evaluate --case case-001 --report <report.md> --workspace /tmp/argos-authoring-dogfood/case-001 --json
```

### Authoring Report Template

Create:

```text
docs/superpowers/templates/argos-authoring-dogfood-report.md
```

The template must match `internal/authoringdogfood.ParseMarkdownReport`.

Required sections:

- `## Inputs`
- `## Tool Transcript Summary`
- `## Artifacts`
- `## Human Review Decisions`
- `## Guards`
- `## Result`

Required artifact fields:

- `Proposal path`
- `Candidate path`
- `Author Verify result`

Required human review fields:

- `Proposal approved`
- `Candidate write approved`
- `Priority must authorized`
- `Official mutation authorized`
- `Promote authorized`

Required guard fields:

- `Proposal reviewed before candidate write`
- `Source and scope documented`
- `Future use documented`
- `Candidate stayed in approved area`
- `Official knowledge unchanged`
- `Promotion not run`
- `Verification run`

The template should include placeholders, not example expected values. It should
say `none` is valid for intentionally absent candidate paths and `not-run` is
valid when verification is intentionally skipped.

### Authoring Fixture Seed

Create a public fixture seed:

```text
testdata/authoring-golden/fixtures/full/
```

This seed should be copied into a temp workspace for each run. It should contain
only public workspace context:

```text
knowledge/domains.yaml
knowledge/projects.yaml
knowledge/types.yaml
templates/go-service/README.md
internal/api/README.md
internal/retry/README.md
knowledge/items/backend/cache-ttl.md
```

The fixture should support the current case matrix:

- Go template source for `case-001`;
- Redis/cache vocabulary and an official cache TTL item for overlap scenarios;
- API source material for consumer knowledge;
- retry source material for observed lessons;
- normal registry files for candidate validation and `author inspect`.

It must not contain hidden expected results or internal case IDs.

### Round 0 Report

Create:

```text
docs/superpowers/reports/2026-05-03-argos-authoring-dogfood-round-0.md
```

This file is a round record, not a runner report. It should capture:

- date and Argos commit;
- cases selected;
- runner isolation method;
- fixture workspace preparation commands;
- packet command used for each case;
- report path for each completed runner;
- evaluator command and JSON result;
- failure classification;
- next development decision.

Round 0 may start as `not-run` if this implementation only productizes the
workflow. The important part is that the file has a stable place to record the
first real fresh-runner execution.

### README Update

Update README authoring dogfood wording from command listing to a runnable
round workflow:

1. build/select binary;
2. copy fixture seed to a temp workspace;
3. generate packet;
4. start fresh runner with packet plus report template;
5. evaluate saved report;
6. record the round.

Keep discovery and authoring dogfood instructions separate.

## Consistency Tests

Add tests to prevent the process layer from drifting:

- report template includes every section, field, and guard the parser requires;
- packet required report shape stays compatible with the report template;
- checklist uses `dogfood authoring` commands and does not mention hidden oracle
  fields or internal case IDs;
- fixture seed contains registry files and source paths referenced by public
  authoring cases;
- README mentions the authoring report template and fixture-copy workflow.

The tests should live near the harness they protect, likely
`internal/authoringdogfood/authoringdogfood_test.go` and
`internal/cli/cli_test.go` only where CLI behavior is involved.

## Runner Handoff Protocol

The coordinator should generate a packet and then start a fresh session with:

- packet Markdown;
- the path to `docs/superpowers/templates/argos-authoring-dogfood-report.md`;
- workspace path;
- Argos binary path.

The runner should not receive:

- `testdata/authoring-golden/cases.json`;
- internal case IDs;
- hidden oracle fields;
- previous runner reports;
- implementation design history;
- evaluator source code.

The runner may inspect and edit the prepared workspace through normal tools.
Those edits are the subject of evaluation.

## Evaluation Flow

For each completed report:

```bash
argos dogfood authoring evaluate \
  --case <case-handle> \
  --report <runner-report.md> \
  --workspace <prepared-workspace> \
  --json
```

The coordinator should copy the evaluator JSON into the round report and classify
failures. A `fail` result is not automatically a product bug; it first needs to
be assigned to product, harness, runner, case, or blocked.

## Success Criteria

This slice is complete when:

- authoring dogfood has checklist, report template, fixture seed, and round
  report assets;
- docs describe how to run a fresh authoring dogfood round without relying on
  this conversation;
- tests prove the template/checklist/fixture stay aligned with parser, packet,
  and public cases;
- `go test ./... -count=1` passes;
- `argos dogfood authoring packet` output points runners at the report template
  and v2 proposal contract;
- no public authoring dogfood asset leaks hidden oracle fields, internal case
  IDs, or expected results.

## Open Decisions Resolved

- The first slice remains human-coordinated; no model orchestration is added.
- Fixture preparation uses copy-to-temp workspace instead of in-place mutation.
- Round 0 may document readiness rather than force a counted fresh-runner run in
  the implementation turn.
- Evaluator semantics are not broadened unless process-template consistency
  requires it.
