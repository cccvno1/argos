# Argos

Argos gives AI coding agents durable project memory.

## Install

Tell your OpenCode agent:

> Fetch and follow instructions from https://raw.githubusercontent.com/cccvno1/argos/refs/heads/main/.opencode/INSTALL.md

That's it. After installation, your agent knows how to use Argos.

## How You Use It

You express intent in natural language:

- Remember this for future agents.
- Preserve this decision.
- Use project knowledge before changing this.
- Audit Argos knowledge before release.

The agent decides when to query, read, write, validate, publish, or cite
Argos knowledge. When writing or publishing durable knowledge, the agent
asks for your approval before changing trusted knowledge.

## What Argos Does

Argos stores project knowledge as local repository files, builds a local
index, and lets future agents find, review, publish, and cite that
knowledge.

Use Argos for durable knowledge such as:

- project standards
- decisions
- lessons
- examples
- runbooks
- references
- knowledge packages

## How It Fits

Workflow systems like Superpowers and OpenSpec decide how work proceeds:
design, planning, testing, debugging, review, and branch completion.

Argos supplies the durable project knowledge those workflows can use.

## Development

Build the CLI:

```bash
go build -ldflags "-X argos/internal/version.Version=$(git describe --tags 2>/dev/null || echo dev)" ./cmd/argos
```

Run the test suite:

```bash
go test ./...
```

## License

MIT
