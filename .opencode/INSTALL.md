# Installing Argos for OpenCode

## Prerequisites

- [OpenCode.ai](https://opencode.ai) installed
- Argos CLI installed and on PATH

## Install Argos CLI

Build and install the argos binary:

```bash
go install argos@latest
```

Or from source:

```bash
go build -ldflags "-X argos/internal/version.Version=v0.1.0" -o ~/.local/bin/argos ./cmd/argos
```

Verify:

```bash
argos --version
```

## Install Argos Plugin

Add argos to the `plugin` array in your `opencode.json` (global or project-level):

```json
{
  "plugin": ["argos@git+https://github.com/cccvno1/argos.git"]
}
```

Restart OpenCode. The plugin registers the `capture-knowledge` skill automatically.

Verify by asking: "What argos skills do you have?"

## Usage

Use OpenCode's native `skill` tool to load the argos skill, or simply tell your agent:

- "Remember this for future agents"
- "Preserve this decision"
- "Check project knowledge before changing this"

The agent will use the `capture-knowledge` skill automatically.