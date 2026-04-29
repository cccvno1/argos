# Argos

Argos is a local-first knowledge substrate for AI development workflows.

## MVP

- Markdown knowledge items with YAML frontmatter
- Controlled registries for domains, projects, and types
- Rebuildable local SQLite index
- Static adapters for AI coding tools
- MCP server for progressive knowledge disclosure
- Workflow contracts for agent and skill systems

## Commands

```bash
argos init
argos validate
argos index
argos install-adapters
argos context --json --project <project>
argos mcp
```

## MCP

Run the local MCP server over stdio:

```bash
argos mcp
```

The server supports tool discovery with `tools/list` and implements these
`tools/call` entries:

- `argos_context`: returns workflow context and recommended next calls. Arguments:
  `project`, `phase`, `task`, `files`.
- `argos_standards`: returns active standards for project work from the local
  index. Arguments: `project`, `task_type`, `files`, `limit`.
- `get_knowledge_item`: fetches one indexed knowledge item including its full
  body. Arguments: `id`.
- `cite_knowledge`: returns citation metadata for indexed knowledge items and
  reports missing ids. Arguments: `ids`.

Run `argos index` before calling index-backed tools:
`argos_standards`, `get_knowledge_item`, and `cite_knowledge`.
