# Argos MCP Tool Calls Design

Date: 2026-04-29

## Purpose

This phase turns `argos mcp` from a discoverable MCP server into a usable MCP server. AI clients should be able to call a small set of Argos tools and receive progressively disclosed knowledge from the local index.

The scope is intentionally narrow. It should validate the end-to-end AI tool integration path without finishing every future knowledge workflow.

## Scope

Implement MCP `tools/call` for these tools:

- `argos_context`
- `argos_standards`
- `get_knowledge_item`
- `cite_knowledge`

Keep these tools discoverable but not yet implemented:

- `argos_requirements`
- `argos_risks`
- `argos_operations`

Unimplemented tools should return a JSON-RPC tool error with a clear message, not crash the server.

## Runtime Model

`argos mcp` runs from a workspace root.

On startup, the CLI should open the local index at:

```text
argos/index.db
```

Then it should pass an index-backed `query.Service` into the MCP server.

If the index does not exist or cannot be opened, `argos mcp` should still start when possible, but tool calls that require the index should return clear tool errors. This keeps discovery and initialization usable and lets the AI explain that `argos index` must be run.

## MCP Call Shape

The server should handle:

```text
tools/call
```

Expected request params:

```json
{
  "name": "argos_standards",
  "arguments": {
    "project": "mall-api",
    "task_type": "implementation",
    "files": ["internal/auth/session.go"],
    "limit": 5
  }
}
```

Successful result shape:

```json
{
  "content": [
    {
      "type": "text",
      "text": "{...pretty JSON...}"
    }
  ],
  "isError": false
}
```

Tool-level failures should return a successful JSON-RPC response with:

```json
{
  "content": [
    {
      "type": "text",
      "text": "index not available: run argos index first"
    }
  ],
  "isError": true
}
```

Malformed JSON-RPC, missing method, and unknown JSON-RPC method should continue using JSON-RPC errors. Invalid tool names or invalid tool arguments should be tool errors inside `tools/call`.

## Tool Behavior

### `argos_context`

Arguments:

```json
{
  "project": "mall-api",
  "phase": "implementation",
  "task": "add refresh token endpoint",
  "files": ["internal/auth/session.go"]
}
```

Behavior:

- Calls `query.Service.Context`.
- Does not require the index.
- Returns pretty JSON.

### `argos_standards`

Arguments:

```json
{
  "project": "mall-api",
  "task_type": "implementation",
  "files": ["internal/auth/session.go"],
  "limit": 5
}
```

Behavior:

- Requires an index-backed query service.
- Calls `query.Service.Standards`.
- Returns summaries, not full Markdown bodies.
- Preserves current progressive disclosure behavior: priority-ranked, file-scoped matches first, max limit 5.

### `get_knowledge_item`

Arguments:

```json
{
  "id": "rule:backend.auth.v1"
}
```

Behavior:

- Requires an index-backed store.
- Fetches the item by ID.
- Returns full Markdown body plus metadata.
- This is the explicit second-stage disclosure step.

Suggested result:

```json
{
  "id": "rule:backend.auth.v1",
  "title": "Auth middleware rule",
  "type": "rule",
  "status": "active",
  "priority": "must",
  "path": "knowledge/items/backend/auth.md",
  "body": "Require explicit auth middleware for account endpoints."
}
```

### `cite_knowledge`

Arguments:

```json
{
  "ids": ["rule:backend.auth.v1"]
}
```

Behavior:

- Requires an index-backed store.
- Fetches each item by ID.
- Returns citation metadata only, not full body.
- Missing IDs should be reported in the result rather than failing the whole call.

Suggested result:

```json
{
  "citations": [
    {
      "id": "rule:backend.auth.v1",
      "title": "Auth middleware rule",
      "path": "knowledge/items/backend/auth.md",
      "status": "active"
    }
  ],
  "missing": []
}
```

## Input Schemas

`tools/list` should expose concrete JSON schemas for implemented tools. The current empty schemas are not enough for clients to guide model calls.

Minimum schemas:

- `argos_context`: `project`, `phase`, `task`, `files`
- `argos_standards`: `project`, `task_type`, `files`, `limit`
- `get_knowledge_item`: `id`
- `cite_knowledge`: `ids`

Unimplemented tools may keep permissive schemas, but their descriptions must make the current status clear.

## CLI Behavior

`argos mcp` should:

1. Determine the current working directory.
2. Try to open `argos/index.db`.
3. Create an MCP server with either:
   - an index-backed query service and store, or
   - a nil/unavailable index state that returns tool errors for index-backed tools.
4. Serve JSON-RPC over stdio.

Discovery methods and `initialize` must keep working even when the index is absent.

## Testing

Add tests for:

- `tools/list` includes real schemas for implemented tools.
- `tools/call` `argos_context` returns context JSON without an index.
- `tools/call` `argos_standards` returns matching rule summaries from a fixture index.
- `tools/call` `get_knowledge_item` returns full item body.
- `tools/call` `cite_knowledge` returns citation metadata and missing IDs.
- index-backed tools return tool errors when no index is available.
- unknown tool names return tool errors.
- malformed `tools/call` arguments return tool errors.
- JSON-RPC protocol errors still use JSON-RPC errors.

## Out Of Scope

This phase does not implement:

- full `argos_requirements`
- full `argos_risks`
- full `argos_operations`
- MCP resource read handlers
- MCP prompt get handlers
- embeddings or semantic search
- web UI

Those should follow after real tool-call integration works in at least one AI coding tool.
