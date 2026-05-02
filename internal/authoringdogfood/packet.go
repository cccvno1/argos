package authoringdogfood

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PacketOptions struct {
	CaseID      string
	Workspace   string
	ArgosBinary string
}

type Packet struct {
	CaseID      string               `json:"case_id"`
	Kind        string               `json:"kind"`
	Fixture     string               `json:"fixture"`
	Workspace   string               `json:"workspace"`
	ArgosBinary string               `json:"argos_binary"`
	Input       Input                `json:"input"`
	HumanReview HumanReviewDecisions `json:"human_review"`
	Markdown    string               `json:"markdown"`
}

func BuildPacket(cases []Case, options PacketOptions) (Packet, error) {
	tc, index, err := lookupAuthoringCase(cases, options.CaseID)
	if err != nil {
		return Packet{}, fmt.Errorf("build authoring dogfood packet: %w", err)
	}

	workspace := strings.TrimSpace(options.Workspace)
	if workspace == "" {
		return Packet{}, fmt.Errorf("build authoring dogfood packet: workspace is required")
	}
	argosBinary := strings.TrimSpace(options.ArgosBinary)
	if argosBinary == "" {
		return Packet{}, fmt.Errorf("build authoring dogfood packet: argos binary is required")
	}

	packet := Packet{
		CaseID:      publicCaseID(index),
		Kind:        publicKind(tc.Kind),
		Fixture:     fixtureName(tc.Fixture),
		Workspace:   workspace,
		ArgosBinary: argosBinary,
		Input:       tc.Input,
		HumanReview: HumanReviewDecisions{
			ProposalApproved:           tc.Approval.ProposalApproved,
			CandidateWriteApproved:     tc.Approval.CandidateWriteApproved,
			PriorityMustAuthorized:     tc.Approval.PriorityMustAuthorized,
			OfficialMutationAuthorized: tc.Approval.OfficialMutationAuthorized,
			PromoteAuthorized:          tc.Approval.PromoteAuthorized,
		},
	}
	packet.Markdown = renderAuthoringPacketMarkdown(packet)
	return packet, nil
}

func renderAuthoringPacketMarkdown(packet Packet) string {
	input, err := json.MarshalIndent(packet.Input, "", "  ")
	if err != nil {
		input = []byte("{}")
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "# Argos Authoring Dogfood Runner Packet\n\n")
	fmt.Fprintf(&builder, "Case: `%s`\n", packet.CaseID)
	fmt.Fprintf(&builder, "Kind: `%s`\n", packet.Kind)
	fmt.Fprintf(&builder, "Fixture: `%s`\n", packet.Fixture)
	fmt.Fprintf(&builder, "Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Runner Instructions\n\n")
	fmt.Fprintf(&builder, "- Use only the workspace and Argos binary listed above.\n")
	fmt.Fprintf(&builder, "- Treat the natural request as the user's full write-side authoring task.\n")
	fmt.Fprintf(&builder, "- Do not inspect case files, golden data, source test code, or private expectation data.\n")
	fmt.Fprintf(&builder, "- Keep proposal review separate from candidate writing; write candidates only when the approval boundary below allows it.\n")
	fmt.Fprintf(&builder, "- Keep all generated artifacts under the workspace using relative paths.\n\n")

	fmt.Fprintf(&builder, "## Natural User Request\n\n")
	fmt.Fprintf(&builder, "%s\n\n", packet.Input.UserRequest)

	fmt.Fprintf(&builder, "## Workspace\n\n")
	fmt.Fprintf(&builder, "- Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "- Argos binary: `%s`\n\n", packet.ArgosBinary)

	fmt.Fprintf(&builder, "## Authoring CLI Equivalents\n\n")
	fmt.Fprintf(&builder, "Use current CLI vocabulary. File writes are regular workspace edits guarded by the human review boundary.\n\n")
	fmt.Fprintf(&builder, "```bash\n")
	fmt.Fprintf(&builder, "cd %s\n", shellQuote(packet.Workspace))
	fmt.Fprintf(&builder, "%s author inspect --json --project %q --goal %q\n", packet.ArgosBinary, packet.Input.Project, packet.Input.UserRequest)
	fmt.Fprintf(&builder, "# Write proposal JSON to a workspace-relative proposal path after proposal review.\n")
	fmt.Fprintf(&builder, "# Write candidate knowledge only when candidate writing is approved.\n")
	fmt.Fprintf(&builder, "%s author verify --json --proposal <proposal-path> --path <candidate-path>\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "```\n\n")

	fmt.Fprintf(&builder, "## Public Input JSON\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", input)

	fmt.Fprintf(&builder, "## Human Review Boundary\n\n")
	fmt.Fprintf(&builder, "- Proposal approved: `%t`\n", packet.HumanReview.ProposalApproved)
	fmt.Fprintf(&builder, "- Candidate write approved: `%t`\n", packet.HumanReview.CandidateWriteApproved)
	fmt.Fprintf(&builder, "- Priority must authorized: `%t`\n", packet.HumanReview.PriorityMustAuthorized)
	fmt.Fprintf(&builder, "- Official mutation authorized: `%t`\n", packet.HumanReview.OfficialMutationAuthorized)
	fmt.Fprintf(&builder, "- Promote authorized: `%t`\n\n", packet.HumanReview.PromoteAuthorized)

	fmt.Fprintf(&builder, "## Required Report Shape\n\n")
	fmt.Fprintf(&builder, "Include these sections: `## Inputs`, `## Tool Transcript Summary`, `## Artifacts`, `## Human Review Decisions`, `## Guards`, and `## Result`.\n\n")
	fmt.Fprintf(&builder, "Required artifact fields: `Proposal path`, `Candidate path`, and `Author Verify result`.\n")
	fmt.Fprintf(&builder, "Use `none` for an intentionally absent candidate path and `not-run` when verification is intentionally skipped.\n\n")
	fmt.Fprintf(&builder, "Required human review fields: `Proposal approved`, `Candidate write approved`, `Priority must authorized`, `Official mutation authorized`, and `Promote authorized`.\n\n")
	fmt.Fprintf(&builder, "Required guard bullets:\n")
	for _, guard := range requiredAuthoringReportGuards {
		fmt.Fprintf(&builder, "- %s: pass | fail | review-needed | not-applicable | not-run\n", strings.Title(guard))
	}
	fmt.Fprintf(&builder, "\n")

	return builder.String()
}

func lookupAuthoringCase(cases []Case, id string) (Case, int, error) {
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
		return Case{}, 0, fmt.Errorf("unknown authoring case %q", id)
	}
	if matches > 1 {
		return Case{}, 0, fmt.Errorf("ambiguous authoring case handle %q", id)
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
