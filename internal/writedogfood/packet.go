package writedogfood

import (
	"encoding/json"
	"fmt"
	"strings"

	"argos/internal/knowledgewrite"
)

type PacketOptions struct {
	CaseID      string
	Workspace   string
	ArgosBinary string
}

type Packet struct {
	CaseID                  string                         `json:"case_id"`
	Kind                    string                         `json:"kind"`
	Fixture                 string                         `json:"fixture"`
	Workspace               string                         `json:"workspace"`
	ArgosBinary             string                         `json:"argos_binary"`
	Input                   Input                          `json:"input"`
	Review                  ReviewDecisions                `json:"review"`
	WriteGuidance           knowledgewrite.WriteGuidance   `json:"write_guidance"`
	KnowledgeDesignTemplate knowledgewrite.KnowledgeDesign `json:"knowledge_design_template"`
	Markdown                string                         `json:"markdown"`
}

func BuildPacket(cases []Case, options PacketOptions) (Packet, error) {
	tc, index, err := lookupWriteCase(cases, options.CaseID)
	if err != nil {
		return Packet{}, fmt.Errorf("build write dogfood packet: %w", err)
	}

	workspace := strings.TrimSpace(options.Workspace)
	if workspace == "" {
		return Packet{}, fmt.Errorf("build write dogfood packet: workspace is required")
	}
	argosBinary := strings.TrimSpace(options.ArgosBinary)
	if argosBinary == "" {
		return Packet{}, fmt.Errorf("build write dogfood packet: argos binary is required")
	}

	design, err := knowledgewrite.Design(workspace, designRequest(tc.Input))
	if err != nil {
		return Packet{}, fmt.Errorf("build write dogfood packet: design guidance: %w", err)
	}

	packet := Packet{
		CaseID:      publicCaseID(index),
		Kind:        publicKind(tc.Kind),
		Fixture:     fixtureName(tc.Fixture),
		Workspace:   workspace,
		ArgosBinary: argosBinary,
		Input:       tc.Input,
		Review: ReviewDecisions{
			DesignApproved:        tc.Approval.DesignApproved,
			DraftWriteApproved:    tc.Approval.DraftWriteApproved,
			PriorityMustApproved:  tc.Approval.PriorityMustApproved,
			OfficialWriteApproved: tc.Approval.OfficialWriteApproved,
			PublishApproved:       tc.Approval.PublishApproved,
		},
		WriteGuidance:           design.WriteGuidance,
		KnowledgeDesignTemplate: design.KnowledgeDesignTemplate,
	}
	packet.Markdown = renderWritePacketMarkdown(packet)
	return packet, nil
}

func renderWritePacketMarkdown(packet Packet) string {
	input := mustMarshalJSON(packet.Input)
	guidance := mustMarshalJSON(packet.WriteGuidance)
	design := mustMarshalJSON(packet.KnowledgeDesignTemplate)

	var builder strings.Builder
	fmt.Fprintf(&builder, "# Argos Write Dogfood Runner Packet\n\n")
	fmt.Fprintf(&builder, "Case: `%s`\n", packet.CaseID)
	fmt.Fprintf(&builder, "Kind: `%s`\n", packet.Kind)
	fmt.Fprintf(&builder, "Fixture: `%s`\n", packet.Fixture)
	fmt.Fprintf(&builder, "Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Runner Instructions\n\n")
	fmt.Fprintf(&builder, "- Use only the workspace, Argos binary, generated packet, and report template path listed in this packet.\n")
	fmt.Fprintf(&builder, "- Treat the natural request as the user's full knowledge write task.\n")
	fmt.Fprintf(&builder, "- Do not inspect case files, golden data, source test code, or private expectation data.\n")
	fmt.Fprintf(&builder, "- Write the design artifact first; write draft knowledge only when the review boundary below allows it.\n")
	fmt.Fprintf(&builder, "- Keep design and draft artifacts under the workspace using relative paths.\n")
	fmt.Fprintf(&builder, "- Use the write dogfood report template at `docs/superpowers/templates/argos-write-dogfood-report.md`.\n")
	fmt.Fprintf(&builder, "- If the coordinator provides a report path, save the completed report there.\n\n")

	fmt.Fprintf(&builder, "## Natural User Request\n\n")
	fmt.Fprintf(&builder, "%s\n\n", packet.Input.UserRequest)

	fmt.Fprintf(&builder, "## Workspace\n\n")
	fmt.Fprintf(&builder, "- Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "- Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Write Guidance\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", guidance)

	fmt.Fprintf(&builder, "## Knowledge Design Template\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", design)

	fmt.Fprintf(&builder, "## CLI Equivalent\n\n")
	fmt.Fprintf(&builder, "```bash\n")
	fmt.Fprintf(&builder, "cd %s\n", shellQuote(packet.Workspace))
	fmt.Fprintf(&builder, "argos knowledge design --json --project <project> --intent <intent>\n")
	fmt.Fprintf(&builder, "%s knowledge design --json --project %q --intent %q\n", packet.ArgosBinary, packet.Input.Project, packet.Input.UserRequest)
	fmt.Fprintf(&builder, "# Write the knowledge design JSON to the design path from write guidance.\n")
	fmt.Fprintf(&builder, "# Write draft knowledge only when draft writing is approved.\n")
	fmt.Fprintf(&builder, "argos knowledge check --json --design <design-path> --draft <draft-path>\n")
	fmt.Fprintf(&builder, "%s knowledge check --json --design <design-path> --draft <draft-path>\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "```\n\n")

	fmt.Fprintf(&builder, "## Public Input JSON\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", input)

	fmt.Fprintf(&builder, "## Review Boundary\n\n")
	fmt.Fprintf(&builder, "- Design approved: `%t`\n", packet.Review.DesignApproved)
	fmt.Fprintf(&builder, "- Draft write approved: `%t`\n", packet.Review.DraftWriteApproved)
	fmt.Fprintf(&builder, "- Priority must approved: `%t`\n", packet.Review.PriorityMustApproved)
	fmt.Fprintf(&builder, "- Official write approved: `%t`\n", packet.Review.OfficialWriteApproved)
	fmt.Fprintf(&builder, "- Publish approved: `%t`\n\n", packet.Review.PublishApproved)

	fmt.Fprintf(&builder, "## Required Report Shape\n\n")
	fmt.Fprintf(&builder, "Use the write dogfood report template at `docs/superpowers/templates/argos-write-dogfood-report.md`.\n\n")
	fmt.Fprintf(&builder, "Include these sections: `## Inputs`, `## Write Guidance`, `## Artifacts`, `## Review Decisions`, `## Guards`, and `## Result`.\n\n")
	fmt.Fprintf(&builder, "Required write guidance fields: `State`, `Next action`, `Design path`, `Draft path`, `Draft allowed`, `Design only`, and `Check result`.\n")
	fmt.Fprintf(&builder, "Report `Design path` and `Draft path` as workspace-relative paths, never absolute filesystem paths.\n")
	fmt.Fprintf(&builder, "Use `none` for an intentionally absent draft path and `not-run` when check is intentionally skipped.\n\n")
	fmt.Fprintf(&builder, "Required review fields: `Design approved`, `Draft write approved`, `Priority must approved`, `Official write approved`, and `Publish approved`.\n\n")
	fmt.Fprintf(&builder, "Required guard bullets:\n")
	for _, guard := range requiredWriteReportGuards {
		fmt.Fprintf(&builder, "- %s: pass | fail | review-needed | not-applicable | not-run\n", reportLabel(guard))
	}
	fmt.Fprintf(&builder, "\n")
	fmt.Fprintf(&builder, "Result semantics:\n")
	fmt.Fprintf(&builder, "- `pass`: draft knowledge is ready for review within the current approval boundary.\n")
	fmt.Fprintf(&builder, "- `review-needed`: approval, source state, design state, findability, or substantive content still needs a human decision.\n")
	fmt.Fprintf(&builder, "- `fail`: workflow violated a boundary or an artifact cannot be inspected.\n\n")

	return builder.String()
}

func mustMarshalJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func reportLabel(guard string) string {
	if guard == "" {
		return ""
	}
	return strings.ToUpper(guard[:1]) + guard[1:]
}

func lookupWriteCase(cases []Case, id string) (Case, int, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Case{}, 0, fmt.Errorf("case id is required")
	}

	var matched Case
	matchedIndex := 0
	matches := 0
	for i, tc := range cases {
		if tc.ID == id || publicCaseID(i) == id {
			matched = tc
			matchedIndex = i
			matches++
		}
	}
	if matches == 0 {
		return Case{}, 0, fmt.Errorf("unknown write case %q", id)
	}
	if matches > 1 {
		return Case{}, 0, fmt.Errorf("ambiguous write case handle %q", id)
	}
	return matched, matchedIndex, nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/._-:", r))
	}) < 0 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
