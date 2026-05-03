package knowledgewrite

import (
	"fmt"
	"strings"
)

func buildWriteGuidance(response DesignResponse, req DesignRequest) WriteGuidance {
	design := response.KnowledgeDesignTemplate
	designOnly := design.DraftOutput.DraftState == "design_only" ||
		design.DraftOutput.Path == "" ||
		len(design.DraftFiles) == 0
	designPath := designPath(response.Project, response.Intent)
	draftPath := design.DraftOutput.Path
	state := "ready_to_design"
	nextAction := "write_knowledge_design"
	reason := "No blocking existing knowledge was detected; write a knowledge design for review before any draft knowledge."

	if designOnly {
		state = "design_only"
		reason = "A design-only review is required before draft knowledge writing."
		if pathRiskBlocksDraft(response.DraftPathRisk) {
			reason = "The requested draft path requires review before draft writing: " + draftPathRiskReviewReason(response.DraftPathRisk) + "."
		}
		if requestLooksMissingSubstantiveContent(req) {
			reason = "Substantive convention content is missing; write a design-only review and ask for the exact convention before draft writing."
		}
		draftPath = ""
	}

	guidance := WriteGuidance{
		State:        state,
		NextAction:   nextAction,
		Reason:       reason,
		DesignPath:   designPath,
		DraftPath:    draftPath,
		DraftAllowed: false,
		DesignOnly:   designOnly,
		StopConditions: []string{
			"Do not write draft knowledge until review.draft_write_approved is true.",
			"Do not write official knowledge unless official writing is explicitly approved.",
			"Do not publish drafts unless publishing is explicitly approved.",
			"Do not use priority must unless review.priority_must_approved is true.",
		},
		DesignFocus: []string{
			"Clarify the future agent task this knowledge supports.",
			"Separate user input, observed facts, imported facts, AI-suggested guidance, assumptions, and open questions.",
			"Record applicability boundaries and negative triggers.",
			"Record existing knowledge decision before draft writing.",
		},
		SourceWork: SourceWorkGuidance{
			UserInput:     []string{"Preserve the user's request as user input."},
			Observed:      []string{"Add observed repo facts only after inspecting files or commands."},
			Imported:      []string{"Use imported only for external material supplied by the user."},
			AISuggested:   []string{"Mark AI recommendations as AI-suggested and review-needed."},
			Assumptions:   []string{"Keep missing details visible as assumptions or open questions."},
			OpenQuestions: []string{"Ask only for decisions needed to make the design accurate and usable."},
		},
		ReviewQuestions: append([]string{}, design.Review.Questions...),
		Commands: WriteCommands{
			WriteDesign: "write JSON to " + designPath,
		},
	}
	if draftPath != "" {
		guidance.Commands.CheckDraft = fmt.Sprintf("argos knowledge check --json --design %s --draft %s", designPath, draftPath)
	}
	if requestLooksConsumerFacing(req) {
		guidance.DesignFocus = append(guidance.DesignFocus, "For consumer-facing knowledge, separate observed interface facts from user interpretation before advising consumers.")
	}
	if len(guidance.ReviewQuestions) == 0 {
		guidance.ReviewQuestions = []string{
			"Is the knowledge goal correct and specific enough?",
			"Is draft writing approved after reviewing the design?",
		}
	}
	return guidance
}

func isInboxDraftPath(path string) bool {
	path = strings.TrimSpace(filepathSlash(path))
	return strings.HasPrefix(path, "knowledge/.inbox/items/") ||
		strings.HasPrefix(path, "knowledge/.inbox/packages/")
}

func isOfficialDraftPath(path string) bool {
	path = strings.TrimSpace(filepathSlash(path))
	return strings.HasPrefix(path, "knowledge/items/") ||
		strings.HasPrefix(path, "knowledge/packages/")
}

func filepathSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
