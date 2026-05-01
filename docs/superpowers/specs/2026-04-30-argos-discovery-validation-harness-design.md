# Argos Discovery Validation Harness Design

Date: 2026-04-30

## Purpose

Argos Discovery needs a repeatable validation harness so future changes can be
judged by behavior, not intuition.

The harness verifies the full agent knowledge flow:

```text
argos_context -> argos_list_knowledge -> argos_find_knowledge -> argos_read_knowledge -> argos_cite_knowledge
```

It must answer four product questions:

1. Can the agent know what the knowledge base contains?
2. Can the agent connect current work to the right knowledge?
3. Can the agent retrieve only the knowledge it needs?
4. When knowledge is missing, does the agent avoid oversearching, reading too
   much into context, or making Argos-backed claims?

This harness is not a replacement for unit tests. It is a golden workflow layer
that combines deterministic regression checks with AI-executed dogfood reports.

## Design Principles

### AI Does The Work

Humans should not need to write code or hand-maintain long checklists to run
the validation flow.

The harness should provide:

- reusable fixtures
- machine-readable golden cases
- automated CLI and query checks
- an AI dogfood checklist
- a structured report template

The human role is to review reports, judge product fit, and approve changes to
the golden expectations when the intended behavior changes.

### Validate The Workflow, Not Just Search

The harness must test Discovery as a governed retrieval workflow.

A passing result means more than "some item was found." It means:

- inventory is visible through `argos_list_knowledge`
- matches are justified through `argos_find_knowledge`
- full bodies are read only through `argos_read_knowledge`
- weak and none support do not recommend citation
- final citations refer only to knowledge that was actually used

### Absence And Uncertainty Are Success Cases

The harness must treat weak, partial, and none results as first-class outcomes.

Discovery should not inflate broad lexical matches into Argos-backed guidance.
The validation suite should fail if weak or none cases recommend full reading,
recommend citation, or imply Argos-backed knowledge that does not exist.

### Keep V1 Lightweight

The validation harness should run on ordinary developer machines.

V1 uses the existing local index, CLI, MCP server tests, and Go test suite. It
does not require embeddings, Ollama, remote model calls, browser automation, or
background services.

## Architecture

The validation system has two layers:

```text
Golden Corpus + Golden Cases -> Automated Harness -> AI Dogfood Checklist
```

### Golden Corpus

The golden corpus is a small fixed knowledge base used by validation tests.

It should include:

- project-scoped knowledge
- global knowledge
- active, draft, and deprecated status examples
- rule, decision, lesson, runbook, reference, and package types
- file-scoped and generic knowledge
- tags and domains that intentionally overlap
- package entrypoints with internal sections

The corpus should live under a test-only path such as:

```text
testdata/discovery-golden/
```

The source Markdown remains the durable fixture. Tests should build the index
from the fixture instead of hand-constructing all knowledge items in Go when the
case is meant to validate real workflow behavior.

### Golden Cases

Golden cases describe expected workflow behavior for representative tasks.

Each case should include:

- case ID
- purpose
- project
- phase
- task
- query
- files
- filters
- expected support
- expected included IDs
- expected excluded IDs
- expected next-call behavior
- progressive reading requirements
- citation requirements

Golden cases should be represented as JSON so Go tests can load them without a
new parser dependency. The format should be simple enough for both Go tests and
AI dogfood instructions to consume.

### Automated Harness

The automated harness runs deterministic checks against the golden corpus.

It should cover:

- `query.Service.ListKnowledge`
- `query.Service.FindKnowledge`
- CLI JSON output for `argos knowledge list --json`
- CLI JSON output for `argos knowledge find --json`
- MCP schema and strict argument behavior where relevant
- progressive reading invariants
- citation guardrails

The automated harness is the regression layer. It should fail on behavioral
drift that would make agents overtrust, overfetch, or miss relevant knowledge.

### Context Isolation

AI dogfood runs must not rely on the current design conversation or any previous
case transcript. Otherwise the runner can be contaminated by known case names,
expected IDs, earlier failures, or the author's intent.

Each dogfood case should run in a fresh, minimal context:

- one case per independent AI session
- no previous case transcripts
- no golden expected IDs or expected support
- no full design spec unless the case explicitly validates documentation use
- no hints about which knowledge should be discovered
- only the current case input, allowed tools, workspace path, checklist, and
  report template

The harness should separate runner and evaluator roles:

- the runner receives only the task input and performs the workflow
- the runner produces a transcript and structured report
- the evaluator receives the report, transcript, and golden expectations
- the evaluator decides pass, fail, or review-needed

The runner must ground every claim in tool output. A report should fail if it
mentions, reads, or cites an ID that did not appear in the runner's discovery or
full reading transcript.

At least one dogfood case should act as a contamination probe. For example, a
case may use an unrelated task after auth-refresh cases have been discussed.
If the runner mentions auth-refresh knowledge without tool evidence, the
dogfood run is contaminated and should fail.

### AI Dogfood Checklist

The AI dogfood checklist is a written procedure for an AI agent to execute.

For each selected case, the agent should:

1. read the case input
2. call `argos_context` when the case models a real workflow entrypoint
3. call `argos_list_knowledge` when inventory awareness is part of the case
4. call `argos_find_knowledge` with the case task, query, files, and filters
5. decide which IDs to read
6. call `argos_read_knowledge` only for selected IDs
7. call `argos_cite_knowledge` only for IDs actually used
8. produce a structured report

The checklist should be written so a human can ask an AI agent to run it without
writing code.

### Validation Report

Each dogfood report should use a stable shape:

```text
Case: strong_auth_refresh_full_signal
Expected support: strong
Actual support: strong
Expected IDs: rule:backend.auth.v1, decision:backend.session.v1
Actual IDs: rule:backend.auth.v1, decision:backend.session.v1
Read IDs: rule:backend.auth.v1
Cited IDs: rule:backend.auth.v1
progressive reading: pass
No-overclaim guard: pass
Result: pass
Notes: Read only the rule because the implementation task did not need the decision body.
```

Reports should distinguish:

- functional failures
- ranking concerns
- missing knowledge
- overfetching
- overclaiming
- expectation changes caused by intentional product decisions

## Scenario Matrix

The first version should cover eight scenario groups.

### 1. Inventory

These cases verify that an agent can know what the knowledge base contains.

Required support:

- normal project inventory includes types, domains, tags, and packages
- empty inventory is explicit and not treated as a tool failure
- global knowledge with no project list is visible to project work
- deprecated knowledge is hidden by default
- deprecated knowledge appears only when explicitly requested
- package entrypoints appear as packages
- package assets and internal files do not appear as standalone items

### 2. Strong Matching

These cases verify that clear tasks find the right knowledge.

Required support:

- task, query, and files all match the same domain
- query-only discovery can find relevant knowledge
- task-only discovery can find relevant knowledge
- file-scoped knowledge ranks above generic knowledge when files match
- must or should priority ranks above lower priority when relevance is similar
- matched IDs include the required rule, decision, runbook, or package

### 3. Partial Matching

These cases verify that Discovery can say "some relevant knowledge exists, but
support is incomplete."

Required support:

- domain-level knowledge exists but no task-specific rule exists
- task-related knowledge exists but file scope does not match
- a lesson exists but no rule or runbook exists
- a package entrypoint exists but detailed supporting knowledge is absent
- response includes missing knowledge hints
- next steps read only high-confidence IDs

### 4. Weak Matching

These cases verify that broad matches do not become Argos-backed guidance.

Required support:

- a single generic term matches, such as `token`
- phase or type matches but task content does not
- broad tags match but task intent is unrelated
- weak support does not recommend `argos_read_knowledge`
- weak support does not recommend `argos_cite_knowledge`

### 5. None

These cases verify that missing knowledge is a clean result.

Required support:

- no item matches the task
- explicit filters exclude otherwise relevant knowledge
- project does not match and no global knowledge exists
- empty index returns none or an inventory-empty result as appropriate
- none support returns no items
- none support does not recommend citation

### 6. Progressive Reading

These cases verify that Discovery does not overload context.

Required support:

- `argos_list_knowledge` does not return full Markdown bodies
- `argos_find_knowledge` does not return full Markdown bodies
- discovery results return IDs, summaries, reasons, matched sections, read_status
  metadata, and next steps
- `argos_read_knowledge` is the only tool that returns full body text
- dogfood runs read only the top one to three needed IDs

### 7. Citation Accountability

These cases verify that final claims are traceable.

Required support:

- strong and partial cases cite only IDs that were actually read and used
- weak and none cases do not cite Argos knowledge
- IDs returned by discovery but never read do not count as used knowledge
- invalid or absent knowledge IDs are rejected by citation checks
- dogfood reports separate "discovered", "read", and "cited" IDs

### 8. Interface Consistency

These cases verify that real entrypoints behave consistently.

Required support:

- `argos knowledge find --json` matches query-service behavior
- `argos knowledge list --json` matches query-service behavior
- MCP `argos_find_knowledge` schema rejects unknown fields
- MCP `argos_list_knowledge` schema rejects unknown fields
- generated adapters recommend MCP first, CLI JSON second, generated files
  third, and Markdown fallback last
- missing index errors are explicit

## Initial Golden Cases

V1 should start with these twenty-one cases. Together they cover the scenario
matrix while staying small enough to run on every development machine.

### `list_inventory_normal`

Purpose: verify that a populated project inventory exposes the knowledge shape.

Expected behavior:

- `argos_list_knowledge` returns nonzero counts for rules and packages
- domains include backend and security
- tags include auth and refresh-token
- package entrypoints appear in `packages`
- full bodies are absent

### `list_inventory_empty`

Purpose: verify that an empty knowledge base is understandable.

Expected behavior:

- `argos_list_knowledge` returns zero counts
- groups are empty
- full bodies are absent
- the result is not confused with a command or schema failure

### `list_hides_deprecated_by_default`

Purpose: verify that stale guidance does not appear unless requested.

Expected behavior:

- default list excludes deprecated items
- `include_deprecated` includes the deprecated item
- deprecated status is visible when included

### `list_global_knowledge_visible`

Purpose: verify that project work can see global knowledge.

Expected behavior:

- an item with no project list appears for `mall-api`
- inventory counts include the global item
- project-scoped unrelated items remain excluded

### `strong_auth_refresh_full_signal`

Purpose: verify the best-case workflow for a concrete implementation task.

Input shape:

- project: `mall-api`
- phase: `implementation`
- task: `add refresh token endpoint`
- query: `refresh token session renewal`
- files: `internal/auth/session.go`

Expected behavior:

- support is strong
- top results include auth rule and session decision
- file-scoped auth rule ranks above generic backend knowledge
- next steps recommend full reading high-priority IDs
- full bodies are absent from discovery

### `strong_auth_refresh_query_only`

Purpose: verify discovery when the agent has search intent but no file context.

Expected behavior:

- support is strong
- results include refresh-token knowledge
- file-scoped ranking does not dominate without files
- no unrelated cache or payment knowledge is returned

### `strong_auth_refresh_task_only`

Purpose: verify discovery when the agent has a natural-language task but no
separate query string.

Expected behavior:

- support is strong
- results include refresh-token knowledge
- the request does not require files to produce a useful match
- no unrelated cache or payment knowledge is returned

### `strong_file_scope_beats_generic`

Purpose: verify file scope as a ranking signal.

Expected behavior:

- both generic backend and file-scoped auth knowledge may match
- file-scoped auth knowledge ranks first when the file is under
  `internal/auth/**`
- why-matched explains file scope

### `partial_domain_without_task_detail`

Purpose: verify partial support when only domain guidance exists.

Expected behavior:

- support is partial
- relevant domain rule or reference appears
- missing knowledge hints mention absent task-specific guidance
- next steps do not overrecommend broad unrelated IDs

### `partial_lesson_without_rule`

Purpose: verify partial support when only experience knowledge exists.

Expected behavior:

- support is partial
- the lesson appears as a relevant match
- missing knowledge hints mention absent rule or runbook guidance
- next steps recommend full reading the lesson and no absent rule or runbook IDs

### `partial_package_entrypoint_without_detail`

Purpose: verify package-aware partial matching.

Expected behavior:

- package entrypoint appears as a relevant match
- support is partial if package summary matches but detailed rule is absent
- matched sections identify package entrypoint sections
- package assets are not standalone results

### `weak_single_generic_term`

Purpose: verify that one broad lexical overlap is not enough.

Expected behavior:

- support is weak
- no `argos_read_knowledge` next step
- no `argos_cite_knowledge` next step
- report recommends inspecting the list or proceeding without Argos-backed guidance

### `weak_broad_tag_only`

Purpose: verify that broad metadata overlap is not treated as task support.

Expected behavior:

- support is weak
- broad backend or platform tags do not produce Argos-backed guidance
- no `argos_read_knowledge` next step
- no `argos_cite_knowledge` next step

### `none_payment_webhook`

Purpose: verify absence behavior for a task outside the corpus.

Expected behavior:

- support is none
- items are empty
- no citation is recommended
- dogfood report says no Argos-backed guidance exists

### `none_project_scope_mismatch`

Purpose: verify that project boundaries are respected.

Expected behavior:

- an item scoped to another project is not returned for `mall-api`
- support is none when no global item matches
- response does not recommend citation

### `none_explicit_filter_excludes_match`

Purpose: verify that explicit user filters are respected.

Expected behavior:

- an otherwise relevant auth item is excluded by nonmatching tags or domains
- support is none
- response does not silently ignore filters to recover a match

### `progressive_read_status_and_citation_guard`

Purpose: verify the end-to-end full reading and citation protocol.

Expected behavior:

- list and find expose no full bodies
- dogfood flow reads only selected IDs
- citation uses only read and applied IDs
- weak and none subcases produce no Argos citation

### `interface_cli_find_matches_query`

Purpose: verify that CLI discovery is not a separate behavioral path.

Expected behavior:

- `argos knowledge find --json` returns the same core support and item IDs as
  `query.Service.FindKnowledge`
- CLI output is valid JSON
- missing or invalid required arguments produce explicit errors

### `interface_cli_list_matches_query`

Purpose: verify that CLI inventory is not a separate behavioral path.

Expected behavior:

- `argos knowledge list --json` returns the same core inventory as `query.Service.ListKnowledge`
- CLI output is valid JSON
- missing project produces an explicit error

### `interface_mcp_strict_schema`

Purpose: verify that MCP tools are governed entrypoints.

Expected behavior:

- `argos_find_knowledge` rejects unknown arguments
- `argos_list_knowledge` rejects unknown arguments
- `argos_find_knowledge` requires task or query
- out-of-range limits are rejected

### `adapter_flow_recommendations`

Purpose: verify that generated adapters keep agents on the governed flow.

Expected behavior:

- adapters recommend MCP first
- adapters recommend CLI JSON second
- adapters preserve generated files and Markdown as fallbacks
- adapters do not tell agents to query SQLite or vector tables directly

## Pass And Fail Criteria

### Pass

A case passes when all of its expected behavior is met and no forbidden behavior
appears.

### Fail

A case fails if any of these happen:

- relevant expected IDs are absent in strong cases
- weak or none cases recommend full reading or citation
- list or find returns full Markdown bodies
- explicit filters are ignored
- deprecated knowledge appears by default
- package internals appear as standalone knowledge items
- dogfood reports cite knowledge that was not read and used
- CLI, MCP, and query service disagree on core behavior

### Review Needed

A case should be marked review-needed, not pass or fail, when:

- the product intentionally changes ranking behavior
- a fixture is updated to represent a new knowledge shape
- a new tool changes the recommended flow
- semantic search is later added as an optional enhancement

## Out Of Scope For V1

V1 does not validate:

- semantic embedding quality
- remote model behavior
- inbox candidate discovery
- package asset full reading beyond entrypoint grouping
- human-authored acceptance narratives
- browser-based workflows

These may be added later only after the local deterministic harness is stable.

## Implementation Notes

The implementation plan should create small, reviewable slices:

1. add reusable golden fixture files
2. add a machine-readable golden case format
3. add query-service golden tests
4. add CLI golden tests
5. add MCP/interface guard tests where needed
6. add the AI dogfood checklist and report template
7. update README or developer docs with how to run validation

The harness should prefer existing project patterns and avoid a new test runner
unless Go tests become too awkward for the golden case format.
