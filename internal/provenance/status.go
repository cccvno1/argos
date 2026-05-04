package provenance

import (
	"os"
	"path/filepath"
	"strings"
)

func Status(root string, idOrPath string) (StatusResult, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return StatusResult{}, err
	}
	verify, err := Verify(root, idOrPath)
	if err != nil {
		return StatusResult{}, err
	}
	decisions, err := LoadDecisions(root, idOrPath)
	if err != nil {
		return StatusResult{}, err
	}

	result := StatusResult{
		Result:       "pass",
		ProvenanceID: loaded.Record.ProvenanceID,
		State:        loaded.Record.State,
		Path:         loaded.Dir,
		Subject:      loaded.Record.Subject,
		Evidence: StatusEvidence{
			DesignBound:        "pass",
			DraftBound:         "pass",
			LatestCheck:        latestCheckEvidence(loaded.Record),
			DesignDecision:     decisionEvidence(decisions, StageDesign),
			DraftWriteDecision: decisionEvidence(decisions, StageDraftWrite),
			PublishDecision:    decisionEvidence(decisions, StagePublish),
			OfficialTarget:     officialTargetEvidence(root, loaded.Record),
		},
	}
	for _, finding := range verify.Findings {
		result.Findings = append(result.Findings, statusFindingForVerifyFinding(finding))
	}
	addMissingDecisionFinding(&result, result.Evidence.DesignDecision, "needs_design_decision", "design decision is missing")
	addMissingDecisionFinding(&result, result.Evidence.DraftWriteDecision, "needs_draft_write_decision", "draft-write decision is missing")
	addMissingDecisionFinding(&result, result.Evidence.PublishDecision, "needs_publish_decision", "publish decision is missing")

	if loaded.Record.State == StateDraft && len(result.Findings) == 0 {
		result.ReadyToPublish = true
	}
	if loaded.Record.State == StatePublished && result.Evidence.OfficialTarget != "pass" {
		result.Findings = append(result.Findings, StatusFinding{
			Severity: "problem", Category: "published_inconsistent",
			Message: "published provenance does not point to an existing official target",
		})
	}
	result.Result = statusResultFromFindings(result.Findings)
	result.Actions = statusActions(result)
	return result, nil
}

func latestCheckEvidence(record Record) string {
	if record.LatestCheck == nil {
		return "missing"
	}
	if record.LatestCheck.Result == "pass" {
		return "pass"
	}
	return "failed"
}

func decisionEvidence(decisions []Decision, stage string) string {
	for i := len(decisions) - 1; i >= 0; i-- {
		if decisions[i].Stage != stage {
			continue
		}
		if decisions[i].Decision == DecisionApproved {
			return "pass"
		}
		return decisions[i].Decision
	}
	return "missing"
}

func officialTargetEvidence(root string, record Record) string {
	if record.State != StatePublished {
		return "not_published"
	}
	if strings.TrimSpace(record.Subject.OfficialPath) == "" {
		return "missing"
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(record.Subject.OfficialPath))); err != nil {
		return "missing"
	}
	return "pass"
}

func statusFindingForVerifyFinding(message string) StatusFinding {
	category := "evidence_mismatch"
	if strings.Contains(message, "design hash changed") {
		category = "design_changed"
	}
	if strings.Contains(message, "draft tree hash changed") {
		category = "draft_changed"
	}
	if strings.Contains(message, "latest check") {
		category = "needs_check"
	}
	if strings.Contains(message, "decision") {
		category = "decision_mismatch"
	}
	return StatusFinding{Severity: "blocked", Category: category, Message: message}
}

func addMissingDecisionFinding(result *StatusResult, evidence string, category string, message string) {
	if evidence != "missing" {
		return
	}
	result.Findings = append(result.Findings, StatusFinding{
		Severity: "blocked",
		Category: category,
		Message:  message,
	})
}

func statusResultFromFindings(findings []StatusFinding) string {
	status := "pass"
	for _, finding := range findings {
		if finding.Severity == "problem" {
			return "problem"
		}
		if finding.Severity == "blocked" {
			status = "blocked"
		}
	}
	return status
}

func statusActions(result StatusResult) []string {
	if result.ReadyToPublish {
		return []string{"review evidence and run argos knowledge publish --provenance " + result.ProvenanceID}
	}
	var actions []string
	if result.Evidence.DesignDecision == "missing" {
		actions = append(actions, "record design decision")
	}
	if result.Evidence.DraftWriteDecision == "missing" {
		actions = append(actions, "record draft-write decision")
	}
	if result.Evidence.LatestCheck == "missing" || result.Evidence.LatestCheck == "failed" {
		actions = append(actions, "run argos provenance record-check --json --provenance "+result.ProvenanceID)
	}
	if result.Evidence.PublishDecision == "missing" {
		actions = append(actions, "record publish decision before publishing")
	}
	return actions
}
