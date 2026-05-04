package provenance

import (
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
			DraftBound:         draftTargetEvidence(root, loaded.Record),
			LatestCheck:        latestCheckEvidence(loaded.Record),
			DesignDecision:     decisionEvidence(decisions, StageDesign),
			DraftWriteDecision: decisionEvidence(decisions, StageDraftWrite),
			PublishDecision:    decisionEvidence(decisions, StagePublish),
			OfficialTarget:     officialTargetEvidence(root, loaded.Record),
		},
	}
	latestCheckPath := latestCheckRecordPath(loaded.Dir, loaded.Record)
	for _, finding := range verify.Findings {
		applyVerifyFindingEvidence(&result.Evidence, loaded.Record, latestCheckPath, finding)
		statusFinding, ok := statusFindingForVerifyFinding(finding, latestCheckPath)
		if ok {
			result.Findings = append(result.Findings, statusFinding)
		}
	}
	addMissingDraftFinding(&result, loaded.Record)
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

func latestCheckRecordPath(recordDir string, record Record) string {
	if record.LatestCheck == nil || strings.TrimSpace(record.LatestCheck.Path) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(recordDir, record.LatestCheck.Path))
}

func draftTargetEvidence(root string, record Record) string {
	if record.State != StateDraft {
		return "pass"
	}
	if strings.TrimSpace(record.Subject.DraftPath) == "" {
		return "missing"
	}
	_, ok, err := existingPathInsideRoot(root, record.Subject.DraftPath)
	if err != nil || !ok {
		return "missing"
	}
	return "pass"
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
	_, ok, err := existingPathInsideRoot(root, record.Subject.OfficialPath)
	if err != nil {
		return "unsafe"
	}
	if !ok {
		return "missing"
	}
	return "pass"
}

func applyVerifyFindingEvidence(evidence *StatusEvidence, record Record, latestCheckPath string, message string) {
	switch {
	case strings.Contains(message, "design decision hashes do not match current record"):
		evidence.DesignDecision = "changed"
	case strings.Contains(message, "draft_write decision hashes do not match current record") ||
		strings.Contains(message, "draft-write decision hashes do not match current record"):
		evidence.DraftWriteDecision = "changed"
	case strings.Contains(message, "publish decision hashes do not match current record"):
		evidence.PublishDecision = "changed"
	case strings.Contains(message, "design hash is required"):
		evidence.DesignBound = "missing"
	case strings.Contains(message, "design hash changed"):
		evidence.DesignBound = "changed"
	case strings.Contains(message, "draft tree hash is required"):
		evidence.DraftBound = "missing"
	case strings.Contains(message, "draft tree hash changed"):
		evidence.DraftBound = "changed"
	case strings.Contains(message, "latest check hash is required"):
		evidence.LatestCheck = "missing"
	case strings.Contains(message, "latest check hash changed") ||
		strings.Contains(message, "latest check result does not match stored check"):
		evidence.LatestCheck = "changed"
	case strings.Contains(message, "latest check is required") ||
		strings.Contains(message, "latest check path is required"):
		evidence.LatestCheck = "missing"
	case isLatestCheckArtifactFinding(message, latestCheckPath):
		evidence.LatestCheck = "missing"
	case strings.Contains(message, "latest check"):
		evidence.LatestCheck = "failed"
	case containsNonEmpty(message, record.Subject.DesignPath):
		evidence.DesignBound = "missing"
	case containsNonEmpty(message, record.Subject.DraftPath):
		evidence.DraftBound = "missing"
	}
}

func containsNonEmpty(s string, substr string) bool {
	return strings.TrimSpace(substr) != "" && strings.Contains(s, substr)
}

func isLatestCheckArtifactFinding(message string, latestCheckPath string) bool {
	return containsNonEmpty(message, latestCheckPath)
}

func statusFindingForVerifyFinding(message string, latestCheckPath string) (StatusFinding, bool) {
	if isMissingApprovalDecisionFinding(message) {
		return StatusFinding{}, false
	}
	category := "evidence_mismatch"
	if strings.Contains(message, "design hash changed") {
		category = "design_changed"
	}
	if strings.Contains(message, "draft tree hash changed") {
		category = "draft_changed"
	}
	if strings.Contains(message, "latest check") || isLatestCheckArtifactFinding(message, latestCheckPath) {
		category = "needs_check"
	}
	if strings.Contains(message, "decision") {
		category = "decision_mismatch"
	}
	return StatusFinding{Severity: "blocked", Category: category, Message: message}, true
}

func isMissingApprovalDecisionFinding(message string) bool {
	return strings.HasSuffix(message, " approval decision is required")
}

func addMissingDraftFinding(result *StatusResult, record Record) {
	if record.State != StateDraft || result.Evidence.DraftBound != "missing" {
		return
	}
	result.Findings = append(result.Findings, StatusFinding{
		Severity: "blocked",
		Category: "draft_missing",
		Message:  "draft target is missing",
	})
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
