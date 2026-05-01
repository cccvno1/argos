package dogfood

import (
	"encoding/json"
	"fmt"
	"strings"

	"argos/internal/discoverytest"
)

type PacketOptions struct {
	CaseID      string
	Workspace   string
	ArgosBinary string
}

type Packet struct {
	CaseID      string              `json:"case_id"`
	Operation   string              `json:"operation"`
	Fixture     string              `json:"fixture"`
	Workspace   string              `json:"workspace"`
	ArgosBinary string              `json:"argos_binary"`
	Input       discoverytest.Input `json:"input"`
	Markdown    string              `json:"markdown"`
}

func BuildPacket(cases []discoverytest.Case, options PacketOptions) (Packet, error) {
	tc, index, err := lookupCase(cases, options.CaseID)
	if err != nil {
		return Packet{}, fmt.Errorf("build dogfood packet: %w", err)
	}
	publicID := publicCaseID(index)
	workspace := strings.TrimSpace(options.Workspace)
	if workspace == "" {
		return Packet{}, fmt.Errorf("build dogfood packet: workspace is required")
	}
	argosBinary := strings.TrimSpace(options.ArgosBinary)
	if argosBinary == "" {
		return Packet{}, fmt.Errorf("build dogfood packet: argos binary is required")
	}
	notes, err := operationNotes(tc, publicID)
	if err != nil {
		return Packet{}, err
	}

	packet := Packet{
		CaseID:      publicID,
		Operation:   tc.Operation,
		Fixture:     fixtureName(tc.Fixture),
		Workspace:   workspace,
		ArgosBinary: argosBinary,
		Input:       tc.Input,
	}
	packet.Markdown = renderMarkdown(packet, notes)
	return packet, nil
}

func renderMarkdown(packet Packet, notes []string) string {
	input, err := json.MarshalIndent(packet.Input, "", "  ")
	if err != nil {
		input = []byte("{}")
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "# Argos Dogfood Runner Packet\n\n")
	fmt.Fprintf(&builder, "Case: `%s`\n", packet.CaseID)
	fmt.Fprintf(&builder, "Operation: `%s`\n", packet.Operation)
	fmt.Fprintf(&builder, "Fixture: `%s`\n", packet.Fixture)
	fmt.Fprintf(&builder, "Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Runner Instructions\n\n")
	fmt.Fprintf(&builder, "- Use only the workspace and Argos binary listed above.\n")
	fmt.Fprintf(&builder, "- Do not inspect `cases.json`, source code, generated indexes, databases, or raw knowledge files.\n")
	fmt.Fprintf(&builder, "- Run commands as a runner would; do not rely on fixture expectations or hidden oracle values.\n")
	fmt.Fprintf(&builder, "- Read knowledge before citing it, and cite only knowledge that was actually used.\n\n")

	fmt.Fprintf(&builder, "## Context Isolation\n\n")
	fmt.Fprintf(&builder, "- Start from a clean context for this case.\n")
	fmt.Fprintf(&builder, "- Treat the input below as the whole task request.\n")
	fmt.Fprintf(&builder, "- Keep Argos-backed findings separate from general reasoning when reporting uncertainty.\n\n")

	fmt.Fprintf(&builder, "## CLI Equivalents\n\n")
	fmt.Fprintf(&builder, "Use current CLI vocabulary. MCP tool names are conceptual only; do not call MCP-style names directly as CLI subcommands.\n\n")
	fmt.Fprintf(&builder, "```bash\n")
	fmt.Fprintf(&builder, "%s context --json --project <project> --phase <phase> --task \"<task>\"\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge list --json --project <project> [--domain <domain>] [--types <type>] [--include-deprecated]\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge find --json --project <project> [--phase <phase>] [--task \"<task>\"] [--query \"<query>\"] [--files <path>] [--types <type>] [--tags <tag>] [--domains <domain>] [--status <status>] [--include-deprecated] [--limit <n>]\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge read --json <id>\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge cite --json <id>...\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "```\n\n")
	fmt.Fprintf(&builder, "Repeat multi-value flags once per value.\n\n")

	fmt.Fprintf(&builder, "## Input JSON\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", input)

	fmt.Fprintf(&builder, "## Operation Notes\n\n")
	for _, note := range notes {
		fmt.Fprintf(&builder, "- %s\n", note)
	}
	fmt.Fprintf(&builder, "\n")

	fmt.Fprintf(&builder, "## Required Report Shape\n\n")
	fmt.Fprintf(&builder, "Include these sections: `## Inputs`, `## Tool Transcript Summary`, `## Observed Results`, `## Guards`, and `## Result`.\n")
	return builder.String()
}

func operationNotes(tc discoverytest.Case, publicID string) ([]string, error) {
	var notes []string
	switch tc.Operation {
	case "list", "list-empty", "cli-list":
		notes = append(notes, "Inventory case: run list only; do not read or cite.")
	case "find", "cli-find":
		notes = append(notes, "Find case: run find, read selected IDs when useful, and cite only read-and-used IDs.")
	case "workflow":
		notes = append(notes, "Workflow case: find, read selected IDs, and cite only read-and-used IDs.")
	case "mcp-schema":
		notes = append(notes, "MCP schema case: use JSON-RPC commands through `argos mcp`.")
	case "adapter":
		notes = append(notes, "Adapter case: run `argos install-adapters` before checking generated files.")
	case "cli-read":
		notes = append(notes, "CLI read case: first discover and select an ID from public command output, then run `argos knowledge read --json <id>` for that selected ID.")
	case "cli-cite":
		notes = append(notes, "CLI citation case: first discover and select read-and-used IDs from public command output, then run `argos knowledge cite --json <id>...` for those selected IDs.")
	default:
		return nil, fmt.Errorf("build dogfood packet: unknown operation %q for case %q", tc.Operation, publicID)
	}
	if tc.ID == "list_hides_deprecated_by_default" {
		notes = append(notes, "Deprecated visibility case: run default list, then run list with `--include-deprecated`.")
	}
	return notes, nil
}
