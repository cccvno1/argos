# Argos Dogfood Harness Productization Design

Date: 2026-05-02

## Goal

Turn the current manual dogfood loop into a repeatable local harness that helps
a human coordinator prepare fresh AI runner sessions, collect structured
reports, and evaluate those reports against golden expectations.

The first slice keeps AI execution human-in-the-loop. Argos should generate the
runner packet and evaluate completed reports, but it should not call an AI model
or manage agent sessions.

## Background

Rounds 6 and 7 validated the renamed shared-knowledge workflow across inventory,
strong, partial, weak, none, CLI, MCP, and adapter cases. Runtime behavior held.
The recurring friction was harness ergonomics:

- runner prompts were assembled manually;
- some reports drifted from the template;
- evaluator judgment was partly ad hoc;
- deprecated visibility and other multi-step cases need clearer runner packets.

The next development slice should make the validation loop repeatable before
starting riskier retrieval work.

## Non-Goals

- Do not automate AI runner execution.
- Do not change ranking, search, support, usage, read, or citation behavior.
- Do not replace existing golden tests.
- Do not require network access.
- Do not parse free-form prose with broad NLP heuristics.
- Do not make historical dogfood reports conform to the new schema.

## User Workflow

The intended coordinator workflow is:

1. Build or select an Argos binary.
2. Prepare a fixture workspace and run `argos index`.
3. Generate one runner packet per selected golden case.
4. Paste each packet into a fresh AI session.
5. Save each runner's report as Markdown or JSON.
6. Evaluate each saved report against the matching golden case.
7. Use the evaluator summary to decide whether runtime behavior, harness
   wording, or test expectations need work.

## CLI Surface

Add a new top-level command group:

```bash
argos dogfood cases --json
argos dogfood packet --case <case-id> --workspace <path> --argos-binary <path> [--json]
argos dogfood evaluate --case <case-id> --report <path> [--json]
```

`argos dogfood cases` lists case IDs, operations, fixtures, and input summaries.
It must not print expected values unless a future explicit debug flag is added.

`argos dogfood packet` emits a runner packet for one case. The packet includes:

- fixture workspace path;
- Argos binary path;
- context isolation rules;
- MCP-to-CLI command equivalents;
- the case input without the `expected` block;
- report shape requirements;
- case-specific execution notes derived from operation and input.

`argos dogfood evaluate` reads one runner report and compares it with the golden
case expectation. The default output is human-readable. `--json` emits the same
result in machine-readable form.

## Runner Packet

The packet is a stable artifact, not just a prompt string.

JSON form:

```json
{
  "case_id": "partial_domain_without_task_detail",
  "operation": "find",
  "fixture": "full",
  "workspace": "/tmp/argos-dogfood/full",
  "argos_binary": "/tmp/argos",
  "input": {
    "project": "mall-api",
    "phase": "implementation",
    "task": "tune cache ttl for product list",
    "query": "cache ttl product list",
    "files": ["internal/catalog/products.go"],
    "limit": 5
  },
  "prompt": "You are validating one Argos Discovery case..."
}
```

Markdown form is optimized for copying into an AI session. It contains the same
data and explicitly says not to inspect `cases.json`, source files, SQLite/FTS
tables, vector tables, or Markdown knowledge files unless the case validates a
fallback path.

The packet must never include `expected`.

## Report Shape

The first slice supports the current Markdown report template and an optional
JSON report shape.

Required Markdown sections:

- `## Inputs`
- `## Tool Transcript Summary`
- `## Observed Results`
- `## Guards`
- `## Result`

Required observed fields:

- actual support;
- usage guidance;
- search status;
- discovered IDs;
- read IDs;
- cited IDs;
- missing needs;
- Argos-backed vs general reasoning;
- next steps.

Required guards:

- progressive reading;
- weak/none no-overclaim;
- citation accountability;
- cited IDs subset of read-and-used IDs;
- missing needs not cited;
- attribution boundary;
- no Discovery-triggered upload/capture;
- usage guidance followed;
- context contamination.

The parser may be deliberately strict about section names and guard labels. If a
report is too free-form, evaluation should return `review-needed` with missing
fields rather than guessing.

## Evaluation Model

Evaluator output has three top-level results:

- `pass`: observed report satisfies the golden expectation and guard rules.
- `fail`: observed report clearly violates a golden expectation or guard rule.
- `review-needed`: the report is incomplete, ambiguous, or outside parser scope.

The evaluator checks:

- actual support level against expected `support` or `support_level`;
- required IDs are present in discovered/read/cited sets as appropriate;
- forbidden IDs are absent;
- list/find did not expose full bodies when `no_bodies` is true;
- weak and none cases did not read or cite;
- cited IDs are a subset of read-and-used IDs;
- missing needs are not cited as knowledge;
- expected missing-need sources are present;
- inventory cases avoid read and cite;
- CLI/MCP/interface cases report the expected rejection or command behavior.

The evaluator should not require perfect prose. It should require enough
structured fields to ground the result.

## Multi-Step Cases

Some golden cases imply multiple checks. The packet generator should add
operation-specific notes:

- inventory cases: list only; do not read or cite;
- deprecated visibility cases: run default list and, when relevant, a second
  list with `--include-deprecated`;
- workflow cases: find, read selected IDs, cite only read-and-used IDs;
- interface cases: include the exact CLI or JSON-RPC command pattern needed for
  the case;
- adapter cases: run `argos install-adapters` before checking generated files.

These notes are instructions only. Golden expected values remain hidden.

## Internal Design

Add a small package for dogfood harness logic. Suggested location:

```text
internal/dogfood
```

Responsibilities:

- load cases from `internal/discoverytest`;
- sanitize a case by removing `Expected`;
- render runner packet Markdown and JSON;
- parse runner reports into a structured `Report`;
- evaluate a `Report` against one `discoverytest.Case`;
- format evaluator summaries.

Keep CLI code thin. `internal/cli` should parse flags, call `internal/dogfood`,
and print results.

## Data Types

Core types:

```go
type Packet struct {
	CaseID      string
	Operation   string
	Fixture     string
	Workspace   string
	ArgosBinary string
	Input       discoverytest.Input
	Prompt      string
}

type Report struct {
	CaseID        string
	ActualSupport string
	Usage         UsageObservation
	SearchStatus  string
	DiscoveredIDs []string
	ReadIDs       []string
	CitedIDs      []string
	MissingNeeds  []MissingNeedObservation
	Guards        map[string]string
	Result        string
}

type Evaluation struct {
	CaseID   string
	Result   string
	Findings []Finding
}
```

Use existing golden structs for expectations. Do not duplicate expectation
schema unless needed for output formatting.

## Error Handling

- Unknown case: exit code `2`, message names the missing case.
- Missing workspace or binary for packet generation: exit code `2`.
- Report file cannot be read: exit code `1`.
- Report cannot be parsed enough to evaluate: evaluator result
  `review-needed`, command exit code `0`.
- Clear expectation violation: evaluator result `fail`, command exit code `0`.
- Internal parsing or JSON formatting error: exit code `1`.

The evaluator result is product output, not necessarily process failure.

## Testing

Unit tests:

- packet generation excludes expected values;
- packet generation includes correct CLI commands and repeated flag guidance;
- packet generation adds operation-specific notes for inventory, deprecated,
  workflow, MCP schema, CLI read/cite, and adapter cases;
- Markdown report parser accepts the current template;
- parser returns missing-field findings for incomplete reports;
- evaluator returns pass for representative strong, partial, weak, none,
  inventory, and interface reports;
- evaluator returns fail for unread citations, forbidden read on weak/none,
  missing required IDs, and forbidden IDs;
- evaluator returns review-needed for report shape drift.

CLI tests:

- `argos dogfood cases --json` returns IDs and inputs without expected values;
- `argos dogfood packet --case ...` returns packet content without expected
  values;
- `argos dogfood evaluate --case ... --report ... --json` returns the expected
  evaluation result.

Regression tests:

- retired shared-knowledge terms remain absent from active dogfood packet and
  evaluator wording;
- `argos_cite_knowledge` remains allowed as the current cite tool name.

## Compatibility

Existing dogfood checklist, report template, and historical reports remain
valid. The harness consumes the current template but does not rewrite old
reports.

`testdata/discovery-golden/cases.json` remains the source of case definitions
and expectations.

## Open Design Choices Resolved For V1

- Output both Markdown and JSON packets, with Markdown as default.
- Keep report parsing strict enough to produce `review-needed` rather than
  guessing.
- Keep AI execution manual.
- Use golden expectations only in evaluator code, never in runner packets.
- Add CLI surface now instead of keeping helpers test-only, because the human
  dogfood workflow is a product workflow.
