# Argos Context Entrypoint Tightening Design

Date: 2026-05-02

## Purpose

Make `argos_context` and `argos context` a reliable first step for agents before
substantial project work.

Argos already has working retrieval primitives: inventory, find, standards,
read, and cite. The current context endpoint is thinner than the rest of the
system: it returns general next-step guidance, but it does not preserve all
task inputs across CLI and MCP surfaces or make the next Argos calls concrete
enough for agents to carry forward consistently.

This slice turns context into a sharper entrypoint without changing ranking,
semantic search, capture, or knowledge storage.

## Background

The current product direction is agent-first:

- humans express knowledge intent in natural language;
- host workflow systems decide how work proceeds;
- Argos supplies durable project knowledge through MCP, CLI JSON, adapters, and
  Markdown fallback.

Discovery and dogfood validation are now stable enough to support product work
above the retrieval core. Round 8 validated the productized dogfood harness and
left the next likely product slice as context handling: help agents carry
`project`, `phase`, `task`, and `files` through the start of a work session.

## Goals

- Align CLI and MCP context inputs.
- Require enough input for context to be task-specific.
- Preserve `project`, `phase`, `task`, and `files` in the context response.
- Return concrete next steps that include reusable request arguments for
  follow-up Argos calls.
- Keep context usable without an index, while becoming more useful when an
  index-backed service is available.
- Add validation coverage and dogfood coverage for context-driven workflows.

## Non-Goals

- Do not add semantic search or embedding providers.
- Do not change the ranking formula for `argos_find_knowledge`.
- Do not change knowledge item or package storage format.
- Do not implement capture, inbox creation, or `argos new`.
- Do not make `argos_context` replace host workflow gates.
- Do not automatically read or cite knowledge from context alone.

## Current State

MCP `argos_context` requires `project`, `phase`, and `task`, and accepts
`files`. CLI `argos context` accepts `project`, defaults `phase` to `planning`,
accepts `task`, and does not currently accept `files`.

The service response contains:

```go
type ContextResponse struct {
    Project              string     `json:"project"`
    Phase                string     `json:"phase"`
    RecommendedNextSteps []NextStep `json:"recommended_next_steps"`
}
```

The returned steps are callable tool names, but they are generic and do not
include request arguments. Agents must reconstruct the same project/task/files
context manually before calling `argos_find_knowledge` or `argos_standards`.

## Design

### Input Contract

`argos_context` and `argos context` should both require:

- `project`
- `phase`
- `task`

Both surfaces should accept repeated or array-style `files`.

CLI examples:

```bash
argos context --json --project mall-api --phase implementation --task "add refresh token endpoint" --files internal/auth/session.go
argos context --json --project mall-api --phase review --task "review refresh token endpoint" --files internal/auth/session.go --files internal/auth/session_test.go
```

MCP schema remains strict and should keep `additionalProperties: false`.

### Response Contract

Extend `ContextResponse` to echo normalized request context:

```go
type ContextResponse struct {
    Project              string            `json:"project"`
    Phase                string            `json:"phase"`
    Task                 string            `json:"task"`
    Files                []string          `json:"files,omitempty"`
    RecommendedNextSteps []ContextNextStep `json:"recommended_next_steps"`
}
```

`ContextNextStep` should remain simple and JSON-friendly:

```go
type ContextNextStep struct {
    Tool      string         `json:"tool"`
    Reason    string         `json:"reason"`
    Arguments map[string]any `json:"arguments,omitempty"`
}
```

The existing `NextStep` type can still be used by find results if that keeps
the implementation smaller. Context steps need arguments because their main job
is to pass the initial task context to the next call.

### Recommended Steps

Context should recommend only implemented tools:

- `argos_list_knowledge`
- `argos_find_knowledge`
- `argos_standards`

It must not recommend read or cite directly. Reading and citing should remain
downstream of `find` or `standards` results.

Default ordering:

1. `argos_find_knowledge`
2. `argos_standards`

Add `argos_list_knowledge` before those only for broad orientation work:

- phase is `planning`;
- task contains terms such as `understand`, `explore`, `orient`, or `map`;
- no files were provided and the task appears broad.

### Step Arguments

Each returned step should include the arguments needed to call the next tool
without re-inventing context.

For `argos_find_knowledge`:

```json
{
  "project": "mall-api",
  "phase": "implementation",
  "task": "add refresh token endpoint",
  "files": ["internal/auth/session.go"]
}
```

For `argos_standards`:

```json
{
  "project": "mall-api",
  "task_type": "implementation",
  "files": ["internal/auth/session.go"]
}
```

For `argos_list_knowledge`:

```json
{
  "project": "mall-api"
}
```

The arguments should be omitted only when there is no useful argument payload.

### Index Availability

Context should remain callable without an index. It should not fail just
because retrieval is not yet indexed.

When no index is available, context still returns the same recommended call
sequence. The follow-up index-backed tools already return explicit
`index not available: run argos index first` errors.

If context later gains index-aware summaries, that should be a separate slice.

### Adapter Guidance

Generated adapters should continue to say that agents call `argos_context`
before substantial project work. They should also encourage agents to preserve
the returned arguments when making follow-up `find` and `standards` calls.

This is wording only. Adapters should not add unimplemented tools or imply that
context itself loads knowledge bodies.

## CLI Changes

`argos context` should:

- require `--json`;
- require `--project`;
- require `--phase`;
- require `--task`;
- accept repeated `--files`;
- print explicit validation errors that mirror the style of `knowledge find`.

This makes the CLI contract match MCP and prevents empty task/query context from
leaking into later dogfood workflows.

## MCP Changes

`argos_context` should keep its strict schema and required
`project/phase/task` fields. It should return the extended response with task,
files, and step arguments.

Unknown arguments should continue to return tool errors. Missing required
arguments should continue to return tool errors.

## Tests

Add or update tests for:

- CLI context rejects missing `--json`;
- CLI context rejects missing project, phase, and task;
- CLI context accepts repeated `--files`;
- CLI and MCP context responses include `project`, `phase`, `task`, `files`,
  and step arguments;
- context recommendations include only implemented tools;
- context does not recommend read or cite directly;
- planning/broad tasks include list before find/standards;
- narrow implementation tasks do not include list by default;
- context still works without an index.

Existing discovery, CLI, MCP, adapter, and dogfood tests should continue to
pass.

## Dogfood

Add a narrow dogfood case or fresh runner packet that validates the entrypoint
workflow:

1. Call `argos context` or `argos_context` with project, phase, task, and files.
2. Use the returned arguments to call `knowledge find` or
   `argos_find_knowledge`.
3. Read and cite only after find permits it.
4. Report whether the context response prevented argument drift.

The runner should not receive expected IDs or support labels.

## Success Criteria

- Agents can start with context and carry the same task fields into follow-up
  calls without manually reconstructing arguments.
- CLI and MCP context behavior match.
- Context remains a routing step, not a body-loading or citation step.
- Existing Discovery support/usage semantics remain unchanged.
- The dogfood harness can validate a context-driven workflow with fresh runner
  sessions.

## Risks

- Making CLI context stricter may break ad hoc manual calls that omitted task or
  phase. This is acceptable because the command is agent/internal and MCP is
  already strict.
- Adding `Arguments map[string]any` introduces a slightly more flexible response
  shape. Keep it constrained to known tools and small primitive values.
- If context grows too much, it could duplicate `find`. This slice avoids that
  by returning call routing and arguments only, not search results or read
  recommendations.
