package author

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AuthoringPacket struct {
	State                string              `json:"state"`
	RecommendedAction    string              `json:"recommended_action"`
	ActionReason         string              `json:"action_reason"`
	ProposalPath         string              `json:"proposal_path"`
	CandidatePath        string              `json:"candidate_path,omitempty"`
	CandidateAllowed     bool                `json:"candidate_allowed"`
	ReviewOnly           bool                `json:"review_only"`
	StopConditions       []string            `json:"stop_conditions"`
	ProposalFocus        []string            `json:"proposal_focus"`
	SourceWork           AuthoringSourceWork `json:"source_work"`
	HumanReviewQuestions []string            "json:\"human_\x72eview_questions\""
	Commands             AuthoringCommands   `json:"commands"`
}

type AuthoringSourceWork struct {
	UserConfirmed []string `json:"user_confirmed,omitempty"`
	Observed      []string `json:"observed,omitempty"`
	Imported      []string `json:"imported,omitempty"`
	Synthesized   []string `json:"synthesized,omitempty"`
	Templates     []string `json:"templates,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	Assumptions   []string `json:"assumptions,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
}

type AuthoringCommands struct {
	WriteProposal   string `json:"write_proposal"`
	VerifyCandidate string `json:"verify_candidate,omitempty"`
}

func buildAuthoringPacket(response InspectResponse, req InspectRequest) AuthoringPacket {
	proposal := response.ProposalScaffold
	reviewOnly := proposal.ProposedShape.ArtifactState == "review_only"
	proposalPath := inspectProposalPath(response.Project, response.Goal)
	candidatePath := proposal.ProposedShape.Path
	state := "ready_for_proposal"
	action := "write_proposal"
	reason := "No blocking overlap was detected; write a proposal for human review before any candidate files."

	if reviewOnly {
		state = "review_only"
		action = "write_review_only_proposal"
		reason = "A review-only proposal is required before candidate writing."
		if pathRiskBlocksCandidate(response.PathRisk) {
			reason = "The requested candidate path requires review before candidate writing: " + pathRiskReviewReason(response.PathRisk) + "."
		}
		if requestLooksMissingSubstantiveContent(req) {
			reason = "Substantive convention content is missing; write a review-only proposal and ask for the exact convention before candidate writing."
		}
		proposalPath = proposal.ProposedShape.Path
		candidatePath = ""
	}

	packet := AuthoringPacket{
		State:             state,
		RecommendedAction: action,
		ActionReason:      reason,
		ProposalPath:      proposalPath,
		CandidatePath:     candidatePath,
		CandidateAllowed:  false,
		ReviewOnly:        reviewOnly,
		StopConditions: []string{
			"Do not write candidate files until human_\x72eview.candidate_write_approved is true.",
			"Do not mutate official knowledge unless official mutation is explicitly authorized.",
			"Do not promote candidates unless promotion is explicitly authorized.",
			"Do not use priority must unless priority_must_authorized is true.",
		},
		ProposalFocus: []string{
			"Clarify the future agent task this knowledge supports.",
			"Separate user-stated intent, observed facts, imported facts, synthesized guidance, assumptions, and open questions.",
			"Record applicability boundaries and negative triggers.",
			"Record overlap decision before candidate writing.",
		},
		SourceWork: AuthoringSourceWork{
			UserConfirmed: []string{"Preserve the user's request as user-stated intent."},
			Observed:      []string{"Add observed repo facts only after inspecting files or commands."},
			Imported:      []string{"Use imported only for external material supplied by the user."},
			Synthesized:   []string{"Mark agent recommendations as synthesized and review-needed."},
			Assumptions:   []string{"Keep missing details visible as assumptions or open questions."},
			OpenQuestions: []string{"Ask only for decisions needed to make the proposal accurate and usable."},
		},
		HumanReviewQuestions: append([]string{}, proposal.HumanReview.ReviewQuestions...),
		Commands: AuthoringCommands{
			WriteProposal: "write JSON to " + proposalPath,
		},
	}
	if candidatePath != "" {
		packet.Commands.VerifyCandidate = fmt.Sprintf("argos author \x76erify --json --proposal %s --path %s", proposalPath, candidatePath)
	}
	if requestLooksConsumerFacing(req) {
		packet.ProposalFocus = append(packet.ProposalFocus, "For consumer-facing knowledge, separate observed interface facts from user interpretation before advising consumers.")
	}
	if len(packet.HumanReviewQuestions) == 0 {
		packet.HumanReviewQuestions = []string{
			"Is the knowledge goal correct and specific enough?",
			"Is candidate writing approved after reviewing the proposal?",
		}
	}
	return packet
}

func inspectProposalPath(project string, goal string) string {
	project = strings.TrimSpace(project)
	if project == "" {
		project = "project"
	}
	return filepath.ToSlash(filepath.Join("knowledge/.inbox/proposals", project, slugForAuthoring(goal), "proposal.json"))
}
