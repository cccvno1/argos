# Argos Knowledge Audit And Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build CLI-first provenance list/status and knowledge audit commands that summarize write evidence for personal and team review without replacing human or PR review.

**Architecture:** Keep the evidence model in `internal/provenance`, add repository-level aggregation in a new `internal/audit` package, and keep `internal/cli` as thin flag parsing plus JSON output. Status/list do not mutate files and do not require the index.

**Tech Stack:** Go standard library, existing Argos CLI patterns, existing `internal/provenance`, `internal/knowledge`, `internal/knowledgewrite`, Git-tracked Markdown/JSON/YAML files.

---

## File Structure

- Create `internal/provenance/list.go`: provenance record discovery and filtering.
- Create `internal/provenance/status.go`: one-record evidence status built on existing verification semantics.
- Modify `internal/provenance/model.go`: add JSON response types for list/status.
- Create `internal/provenance/list_test.go`: focused list service tests.
- Create `internal/provenance/status_test.go`: focused status service tests.
- Create `internal/audit/audit.go`: repository-level audit aggregation.
- Create `internal/audit/audit_test.go`: audit service tests.
- Modify `internal/cli/cli.go`: add `provenance list`, `provenance status`, and `knowledge audit`.
- Modify `internal/cli/cli_test.go`: public CLI JSON and end-to-end audit tests.
- Modify `README.md`: add audit/status commands and review-assistance explanation.
- Modify `skills/capture-knowledge/SKILL.md`: tell agents to use status/audit before publishing or asking for review.

Keep implementation focused on CLI JSON. Do not add MCP tools in this round.

---

### Task 1: Provenance List Service

**Files:**
- Modify: `internal/provenance/model.go`
- Create: `internal/provenance/list.go`
- Create: `internal/provenance/list_test.go`

- [ ] **Step 1: Write failing list tests**

Create `internal/provenance/list_test.go`:

```go
package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestListReturnsDraftAndPublishedRecords(t *testing.T) {
	root := t.TempDir()
	writeListRecord(t, root, "knowledge/.inbox/provenance/prov-draft/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-draft",
		State:         StateDraft,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
		CreatedAt: "2026-05-04T00:00:00Z",
		CreatedBy: "codex",
	})
	writeListRecord(t, root, "knowledge/provenance/package_mall-api.redis-cache.v1/prov-published/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-published",
		State:         StatePublished,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
		LatestCheck: &LatestCheck{Result: "pass"},
		CreatedAt:   "2026-05-04T00:00:00Z",
		CreatedBy:   "codex",
		PublishedAt: "2026-05-04T00:10:00Z",
	})

	result, err := List(root, ListFilter{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected 2 records, got %#v", result.Records)
	}
	if result.Records[0].ProvenanceID != "prov-draft" || result.Records[1].ProvenanceID != "prov-published" {
		t.Fatalf("records should be sorted by path: %#v", result.Records)
	}
}

func TestListFiltersByStateProjectAndKnowledgeID(t *testing.T) {
	root := t.TempDir()
	writeListRecord(t, root, "knowledge/.inbox/provenance/prov-mall/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-mall",
		State:         StateDraft,
		Subject:       Subject{Project: "mall-api", KnowledgeID: "package:mall-api.redis-cache.v1"},
	})
	writeListRecord(t, root, "knowledge/provenance/package_billing.redis.v1/prov-billing/provenance.json", Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  "prov-billing",
		State:         StatePublished,
		Subject:       Subject{Project: "billing-api", KnowledgeID: "package:billing.redis.v1"},
	})

	result, err := List(root, ListFilter{
		State:       StateDraft,
		Project:     "mall-api",
		KnowledgeID: "package:mall-api.redis-cache.v1",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result.Records) != 1 || result.Records[0].ProvenanceID != "prov-mall" {
		t.Fatalf("unexpected filtered records: %#v", result.Records)
	}
}

func TestListRejectsInvalidState(t *testing.T) {
	_, err := List(t.TempDir(), ListFilter{State: "ready"})
	if err == nil || err.Error() != "state must be draft, published, or all" {
		t.Fatalf("expected invalid state error, got %v", err)
	}
}

func writeListRecord(t *testing.T, root string, rel string, record Record) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir record parent: %v", err)
	}
	if err := writeRecord(abs, record); err != nil {
		t.Fatalf("write record: %v", err)
	}
}
```

- [ ] **Step 2: Run list tests to verify failure**

Run:

```bash
go test ./internal/provenance -run 'TestList' -count=1
```

Expected: FAIL with compiler errors for `List` and `ListFilter`.

- [ ] **Step 3: Add list response types**

Add to `internal/provenance/model.go` after `VerifyResult`:

```go
type ListFilter struct {
	State       string
	Project     string
	KnowledgeID string
}

type ListResponse struct {
	Records []ListRecord `json:"records"`
}

type ListRecord struct {
	ProvenanceID      string `json:"provenance_id"`
	State             string `json:"state"`
	Path              string `json:"path"`
	Project           string `json:"project,omitempty"`
	KnowledgeID       string `json:"knowledge_id,omitempty"`
	Kind              string `json:"kind,omitempty"`
	DesignPath        string `json:"design_path,omitempty"`
	DraftPath         string `json:"draft_path,omitempty"`
	OfficialPath      string `json:"official_path,omitempty"`
	LatestCheckResult string `json:"latest_check_result,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	PublishedAt       string `json:"published_at,omitempty"`
}
```

- [ ] **Step 4: Implement provenance list**

Create `internal/provenance/list.go`:

```go
package provenance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func List(root string, filter ListFilter) (ListResponse, error) {
	state := strings.TrimSpace(filter.State)
	if state == "" {
		state = "all"
	}
	if state != "all" && state != StateDraft && state != StatePublished {
		return ListResponse{}, fmt.Errorf("state must be draft, published, or all")
	}

	var loaded []Loaded
	for _, relRoot := range []string{
		filepath.Join("knowledge", ".inbox", "provenance"),
		filepath.Join("knowledge", "provenance"),
	} {
		records, err := listRecordsUnder(root, relRoot)
		if err != nil {
			return ListResponse{}, err
		}
		loaded = append(loaded, records...)
	}
	sort.Slice(loaded, func(i int, j int) bool {
		return loaded[i].Path < loaded[j].Path
	})

	project := strings.TrimSpace(filter.Project)
	knowledgeID := strings.TrimSpace(filter.KnowledgeID)
	var records []ListRecord
	for _, item := range loaded {
		record := item.Record
		if state != "all" && record.State != state {
			continue
		}
		if project != "" && record.Subject.Project != project {
			continue
		}
		if knowledgeID != "" && record.Subject.KnowledgeID != knowledgeID {
			continue
		}
		records = append(records, listRecordFromLoaded(item))
	}
	return ListResponse{Records: records}, nil
}

func listRecordsUnder(root string, relRoot string) ([]Loaded, error) {
	absRoot := filepath.Join(root, relRoot)
	info, err := os.Stat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", filepath.ToSlash(relRoot), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s exists but is not a directory", filepath.ToSlash(relRoot))
	}

	var records []Loaded
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "provenance.json" {
			return nil
		}
		loaded, err := readRecord(root, path)
		if err != nil {
			return err
		}
		records = append(records, loaded)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func listRecordFromLoaded(loaded Loaded) ListRecord {
	record := loaded.Record
	latest := ""
	if record.LatestCheck != nil {
		latest = record.LatestCheck.Result
	}
	return ListRecord{
		ProvenanceID:      record.ProvenanceID,
		State:             record.State,
		Path:              loaded.Dir,
		Project:           record.Subject.Project,
		KnowledgeID:       record.Subject.KnowledgeID,
		Kind:              record.Subject.Kind,
		DesignPath:        record.Subject.DesignPath,
		DraftPath:         record.Subject.DraftPath,
		OfficialPath:      record.Subject.OfficialPath,
		LatestCheckResult: latest,
		CreatedAt:         record.CreatedAt,
		PublishedAt:       record.PublishedAt,
	}
}
```

- [ ] **Step 5: Run list tests**

Run:

```bash
go test ./internal/provenance -run 'TestList' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit list service**

```bash
git add internal/provenance/model.go internal/provenance/list.go internal/provenance/list_test.go
git commit -m "feat: list provenance records"
```

---

### Task 2: Provenance Status Service

**Files:**
- Modify: `internal/provenance/model.go`
- Create: `internal/provenance/status.go`
- Create: `internal/provenance/status_test.go`

- [ ] **Step 1: Write failing status tests**

Create `internal/provenance/status_test.go`:

```go
package provenance

import (
	"encoding/json"
	"testing"

	"argos/internal/knowledgewrite"
)

func TestStatusReportsNeedsPublishDecision(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "blocked" {
		t.Fatalf("expected blocked, got %#v", status)
	}
	if status.Evidence.PublishDecision != "missing" {
		t.Fatalf("expected missing publish decision, got %#v", status.Evidence)
	}
	if !statusHasCategory(status.Findings, "needs_publish_decision") {
		t.Fatalf("missing needs_publish_decision finding: %#v", status.Findings)
	}
}

func TestStatusReportsReadyToPublish(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	if _, err := RecordDecision(root, id, DecisionInput{
		Stage: StagePublish, Decision: DecisionApproved, DecidedBy: "chenchi",
		Role: "knowledge_owner", Source: "conversation", Reason: "publish approved",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "pass" || !status.ReadyToPublish {
		t.Fatalf("expected ready pass, got %#v", status)
	}
}

func TestStatusBlocksWhenDraftChangesAfterPublishDecision(t *testing.T) {
	root, id := createStatusWorkspaceThroughCheck(t)
	if _, err := RecordDecision(root, id, DecisionInput{
		Stage: StagePublish, Decision: DecisionApproved, DecidedBy: "chenchi",
		Role: "knowledge_owner", Source: "conversation", Reason: "publish approved",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed\n")

	status, err := Status(root, id)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Result != "blocked" {
		t.Fatalf("expected blocked after draft change, got %#v", status)
	}
	if !statusHasCategory(status.Findings, "draft_changed") {
		t.Fatalf("expected draft_changed finding, got %#v", status.Findings)
	}
}

func createStatusWorkspaceThroughCheck(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	designPath := "knowledge/.inbox/designs/mall-api/redis-cache/design.json"
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeStatusDesign(t, root, designPath, draftPath)
	writeTestFile(t, root, draftPath+"/KNOWLEDGE.md", validProvenanceDraftPackage())
	writeTestFile(t, root, draftPath+"/references/redis.md", "Redis reference.\n")

	record, err := Start(root, StartRequest{DesignPath: designPath, DraftPath: draftPath, CreatedBy: "codex"})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	for _, stage := range []string{StageDesign, StageDraftWrite} {
		if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
			Stage: stage, Decision: DecisionApproved, DecidedBy: "chenchi",
			Role: "knowledge_owner", Source: "conversation", Reason: stage + " approved",
			RecordedBy: "codex",
		}); err != nil {
			t.Fatalf("record %s decision: %v", stage, err)
		}
	}
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	return root, record.ProvenanceID
}

func writeStatusDesign(t *testing.T, root string, rel string, draftPath string) {
	t.Helper()
	design := knowledgewrite.KnowledgeDesign{
		SchemaVersion: knowledgewrite.KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices.",
		KnowledgeGoal: "Document Redis cache best practices.",
		Project:       "mall-api",
		Audience: knowledgewrite.Audience{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"implement Redis cache behavior"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			TechDomains:  []string{"backend"},
			FileGlobs:    []string{"**/*"},
		},
		Sources: knowledgewrite.Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			Claims: []knowledgewrite.SourceClaim{{
				Claim: "Redis cache guidance is needed.", Kind: "fact",
				Source: []string{"user request"}, Trust: "user_input",
			}},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind: "package", Type: "package", Title: "Redis Cache Best Practices",
			ID: "package:mall-api.redis-cache.v1", Path: draftPath, Status: "draft",
			Priority: "should", Rationale: "Package entrypoint for cache knowledge.",
			EntrypointLoad: "read_before_implementation", DraftState: "draft",
		},
		FutureUse: knowledgewrite.FutureUse{
			TriggerRequests: []string{"implement Redis cache behavior"},
			NegativeTriggers: []string{"unrelated auth work"},
			QueryPhrases: []string{"redis cache best practices"},
			ExpectedUse: "Read before implementing Redis cache behavior.",
			CitationPolicy: "cite_after_use",
		},
		Applicability: knowledgewrite.Applicability{
			WhenToUse: []string{"When implementing Redis cache behavior."},
			WhenNotToUse: []string{"When work is unrelated to Redis."},
		},
		ExistingKnowledge: knowledgewrite.ExistingKnowledgeDecision{
			Decision: "create_new", Reason: "No existing knowledge covers this.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path: "inbox", WriteRequiresReviewApproval: true, ReviewPacketRequired: true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path: draftPath + "/KNOWLEDGE.md", Purpose: "Package entrypoint.",
			Load: "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: draftPath,
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: "mall-api", Phase: "implementation",
				Task: "implement Redis cache behavior", Query: "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{Questions: []string{"Is this ready?"}},
	}
	data, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	writeTestFile(t, root, rel, string(data)+"\n")
}

func statusHasCategory(findings []StatusFinding, category string) bool {
	for _, finding := range findings {
		if finding.Category == category {
			return true
		}
	}
	return false
}

```

- [ ] **Step 2: Run status tests to verify failure**

Run:

```bash
go test ./internal/provenance -run 'TestStatus' -count=1
```

Expected: FAIL with compiler errors for `Status`, `StatusFinding`, or `StatusResult`.

- [ ] **Step 3: Add status response types**

Add to `internal/provenance/model.go`:

```go
type StatusResult struct {
	Result         string          `json:"result"`
	ProvenanceID   string          `json:"provenance_id"`
	State          string          `json:"state"`
	Path           string          `json:"path"`
	Subject        Subject         `json:"subject"`
	Evidence       StatusEvidence  `json:"evidence"`
	ReadyToPublish bool            `json:"ready_to_publish"`
	Actions        []string        `json:"actions,omitempty"`
	Findings       []StatusFinding `json:"findings,omitempty"`
}

type StatusEvidence struct {
	DesignBound        string `json:"design_bound"`
	DraftBound         string `json:"draft_bound"`
	LatestCheck        string `json:"latest_check"`
	DesignDecision     string `json:"design_decision"`
	DraftWriteDecision string `json:"draft_write_decision"`
	PublishDecision    string `json:"publish_decision"`
	OfficialTarget     string `json:"official_target"`
}

type StatusFinding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
}
```

- [ ] **Step 4: Implement status**

Create `internal/provenance/status.go`:

```go
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
```

- [ ] **Step 5: Run status tests**

Run:

```bash
go test ./internal/provenance -run 'TestStatus' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run full provenance package tests**

Run:

```bash
go test ./internal/provenance -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit status service**

```bash
git add internal/provenance/model.go internal/provenance/status.go internal/provenance/status_test.go
git commit -m "feat: summarize provenance status"
```

---

### Task 3: Provenance List And Status CLI

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add these tests near existing provenance CLI tests in `internal/cli/cli_test.go`:

```go
func TestRunProvenanceListReturnsJSONRecords(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	designPath := writeJSONFixture(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validCLIKnowledgeDesign("knowledge/.inbox/packages/mall-api/redis-cache", "package:mall-api.redis-cache.v1"))
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackageForCLI(t, root, draftPath)
	provenanceID := createCLIProvenanceThroughCheck(t, root, designPath, draftPath)

	output := runOK(t, root, []string{"provenance", "list", "--json", "--state", "draft", "--project", "mall-api"})
	if !strings.Contains(output, provenanceID) || !strings.Contains(output, `"latest_check_result": "pass"`) {
		t.Fatalf("expected provenance list record, got %s", output)
	}
}

func TestRunProvenanceStatusShowsMissingPublishDecision(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	designPath := writeJSONFixture(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validCLIKnowledgeDesign("knowledge/.inbox/packages/mall-api/redis-cache", "package:mall-api.redis-cache.v1"))
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackageForCLI(t, root, draftPath)
	provenanceID := createCLIProvenanceThroughCheck(t, root, designPath, draftPath)

	output := runOK(t, root, []string{"provenance", "status", "--json", "--provenance", provenanceID})
	if !strings.Contains(output, `"category": "needs_publish_decision"`) {
		t.Fatalf("expected missing publish decision status, got %s", output)
	}
}

func TestRunProvenanceListRequiresJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"provenance", "list"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "provenance list: --json is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -run 'TestRunProvenance(List|Status)' -count=1
```

Expected: FAIL with unknown provenance subcommand.

- [ ] **Step 3: Wire CLI subcommands**

In `internal/cli/cli.go`, update `runProvenance`:

```go
	case "list":
		return runProvenanceList(args[1:], stdout, stderr)
	case "status":
		return runProvenanceStatus(args[1:], stdout, stderr)
```

Add functions after `runProvenanceVerify`:

```go
func runProvenanceList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	state := flags.String("state", "all", "record state: draft, published, or all")
	project := flags.String("project", "", "project id")
	knowledgeID := flags.String("knowledge-id", "", "knowledge id")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance list: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance list: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.List(root, provenance.ListFilter{
		State: *state, Project: *project, KnowledgeID: *knowledgeID,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance list: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runProvenanceStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	provenanceID := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance status: --json is required")
		return 2
	}
	if strings.TrimSpace(*provenanceID) == "" {
		fmt.Fprintln(stderr, "provenance status: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance status: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.Status(root, *provenanceID)
	if err != nil {
		fmt.Fprintf(stderr, "provenance status: %v\n", err)
		return 1
	}
	if code := printJSON(stdout, stderr, result); code != 0 {
		return code
	}
	if result.Result == "problem" {
		return 1
	}
	return 0
}
```

Update `printUsage` to include:

```go
fmt.Fprintln(w, "  argos provenance list --json")
fmt.Fprintln(w, "  argos provenance status --json --provenance <id>")
```

- [ ] **Step 4: Run CLI provenance tests**

Run:

```bash
go test ./internal/cli -run 'TestRunProvenance(List|Status)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit provenance CLI**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: expose provenance status commands"
```

---

### Task 4: Knowledge Audit Service And CLI

**Files:**
- Create: `internal/audit/audit.go`
- Create: `internal/audit/audit_test.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing audit service tests**

Create `internal/audit/audit_test.go`:

```go
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"argos/internal/provenance"
)

func TestKnowledgeAuditEmptyWorkspacePasses(t *testing.T) {
	result, err := Knowledge(t.TempDir(), Request{})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "pass" || len(result.Items) != 0 {
		t.Fatalf("expected empty pass, got %#v", result)
	}
}

func TestKnowledgeAuditShowsNeedsPublishDecision(t *testing.T) {
	root := t.TempDir()
	writeAuditRecord(t, root, "knowledge/.inbox/provenance/prov-needs-publish/provenance.json", provenance.Record{
		SchemaVersion: provenance.SchemaVersion,
		ProvenanceID:  "prov-needs-publish",
		State:         provenance.StateDraft,
		Subject: provenance.Subject{
			Project: "mall-api", KnowledgeID: "package:mall-api.redis-cache.v1",
			Kind: "package", DraftPath: "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
		LatestCheck: &provenance.LatestCheck{Result: "pass"},
	})

	result, err := Knowledge(root, Request{Project: "mall-api"})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "blocked" {
		t.Fatalf("expected blocked, got %#v", result)
	}
	if !hasAuditCategory(result.Items, "needs_publish_decision") {
		t.Fatalf("missing needs_publish_decision item: %#v", result.Items)
	}
}

func TestKnowledgeAuditReportsOfficialMissingProvenance(t *testing.T) {
	root := t.TempDir()
	writeAuditFile(t, root, "knowledge/packages/mall-api/redis-cache/KNOWLEDGE.md", `---
id: package:mall-api.redis-cache.v1
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-05-04
---
## Purpose

Document Redis cache guidance.

## When To Use

Use when Redis cache behavior changes.

## Start Here

Read this before implementation.

## Load On Demand

No extra files.
`)

	result, err := Knowledge(root, Request{})
	if err != nil {
		t.Fatalf("Knowledge returned error: %v", err)
	}
	if result.Result != "problem" {
		t.Fatalf("expected problem for missing provenance, got %#v", result)
	}
	if !hasAuditCategory(result.Items, "official_missing_provenance") {
		t.Fatalf("missing official_missing_provenance item: %#v", result.Items)
	}
}

func writeAuditRecord(t *testing.T, root string, rel string, record provenance.Record) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir record parent: %v", err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(abs, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}
}

func writeAuditFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir file parent: %v", err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func hasAuditCategory(items []Item, category string) bool {
	for _, item := range items {
		if item.Category == category {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run audit service tests to verify failure**

Run:

```bash
go test ./internal/audit -count=1
```

Expected: FAIL because package or functions do not exist.

- [ ] **Step 3: Implement audit service**

Create `internal/audit/audit.go`:

```go
package audit

import (
	"argos/internal/knowledge"
	"argos/internal/provenance"
)

type Request struct {
	Project          string
	IncludePublished bool
}

type Result struct {
	Result  string  `json:"result"`
	Summary Summary `json:"summary"`
	Items   []Item  `json:"items"`
}

type Summary struct {
	Open                      int `json:"open"`
	ReadyToPublish            int `json:"ready_to_publish"`
	Blocked                   int `json:"blocked"`
	Problems                  int `json:"problems"`
	Published                 int `json:"published"`
	OfficialMissingProvenance int `json:"official_missing_provenance"`
}

type Item struct {
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	ProvenanceID string `json:"provenance_id,omitempty"`
	Project      string `json:"project,omitempty"`
	KnowledgeID  string `json:"knowledge_id,omitempty"`
	Path         string `json:"path,omitempty"`
	Action       string `json:"action"`
}

func Knowledge(root string, req Request) (Result, error) {
	list, err := provenance.List(root, provenance.ListFilter{Project: req.Project})
	if err != nil {
		return Result{}, err
	}
	var result Result
	publishedByID := map[string]bool{}
	for _, record := range list.Records {
		if record.State == provenance.StatePublished {
			publishedByID[record.KnowledgeID] = true
		}
		status, err := provenance.Status(root, record.ProvenanceID)
		if err != nil {
			return Result{}, err
		}
		addStatusItem(&result, record, status, req.IncludePublished)
	}
	if err := addOfficialMissingProvenance(root, req.Project, publishedByID, &result); err != nil {
		return Result{}, err
	}
	result.Result = resultStatus(result.Summary)
	return result, nil
}

func addStatusItem(result *Result, record provenance.ListRecord, status provenance.StatusResult, includePublished bool) {
	if status.ReadyToPublish {
		result.Items = append(result.Items, Item{
			Category: "ready_to_publish", Severity: "warning",
			ProvenanceID: record.ProvenanceID, Project: record.Project,
			KnowledgeID: record.KnowledgeID, Path: record.Path,
			Action: "review evidence and run argos knowledge publish --provenance " + record.ProvenanceID,
		})
		result.Summary.ReadyToPublish++
		result.Summary.Open++
		return
	}
	if record.State == provenance.StatePublished && status.Result == "pass" {
		result.Summary.Published++
		if includePublished {
			result.Items = append(result.Items, Item{
				Category: "published", Severity: "pass",
				ProvenanceID: record.ProvenanceID, Project: record.Project,
				KnowledgeID: record.KnowledgeID, Path: record.Path,
				Action: "no action required",
			})
		}
		return
	}
	for _, finding := range status.Findings {
		severity := finding.Severity
		if severity == "" {
			severity = "blocked"
		}
		result.Items = append(result.Items, Item{
			Category: finding.Category, Severity: severity,
			ProvenanceID: record.ProvenanceID, Project: record.Project,
			KnowledgeID: record.KnowledgeID, Path: record.Path,
			Action: firstAction(status.Actions),
		})
		if severity == "problem" {
			result.Summary.Problems++
		} else {
			result.Summary.Blocked++
		}
		result.Summary.Open++
	}
}

func addOfficialMissingProvenance(root string, project string, publishedByID map[string]bool, result *Result) error {
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		return err
	}
	for _, item := range items {
		if project != "" && !itemAppliesToProject(item, project) {
			continue
		}
		if publishedByID[item.ID] {
			continue
		}
		result.Items = append(result.Items, Item{
			Category: "official_missing_provenance", Severity: "problem",
			Project: projectForAuditItem(item, project), KnowledgeID: item.ID,
			Path: item.Path, Action: "inspect official knowledge and add provenance through the next change",
		})
		result.Summary.OfficialMissingProvenance++
		result.Summary.Problems++
	}
	return nil
}

func firstAction(actions []string) string {
	if len(actions) == 0 {
		return "inspect provenance status"
	}
	return actions[0]
}

func resultStatus(summary Summary) string {
	if summary.Problems > 0 {
		return "problem"
	}
	if summary.Blocked > 0 {
		return "blocked"
	}
	if summary.Open > 0 || summary.ReadyToPublish > 0 {
		return "warning"
	}
	return "pass"
}

func projectForAuditItem(item knowledge.Item, requested string) string {
	if requested != "" {
		return requested
	}
	if len(item.Projects) > 0 {
		return item.Projects[0]
	}
	return ""
}

func itemAppliesToProject(item knowledge.Item, project string) bool {
	for _, candidate := range item.Projects {
		if candidate == project {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run audit service tests**

Run:

```bash
go test ./internal/audit -count=1
```

Expected: PASS.

- [ ] **Step 5: Write failing audit CLI tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunKnowledgeAuditReturnsReviewQueue(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	designPath := writeJSONFixture(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validCLIKnowledgeDesign("knowledge/.inbox/packages/mall-api/redis-cache", "package:mall-api.redis-cache.v1"))
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackageForCLI(t, root, draftPath)
	provenanceID := createCLIProvenanceThroughCheck(t, root, designPath, draftPath)

	output := runOK(t, root, []string{"knowledge", "audit", "--json", "--project", "mall-api"})
	if !strings.Contains(output, provenanceID) || !strings.Contains(output, `"category": "needs_publish_decision"`) {
		t.Fatalf("expected audit queue item, got %s", output)
	}
}

func TestRunKnowledgeAuditRequiresJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "audit"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "knowledge audit: --json is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
```

- [ ] **Step 6: Wire `knowledge audit` CLI**

Modify imports in `internal/cli/cli.go`:

```go
"argos/internal/audit"
```

Update `runKnowledge` switch:

```go
	case "audit":
		return runKnowledgeAudit(args[1:], stdout, stderr)
```

Add:

```go
func runKnowledgeAudit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge audit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	includePublished := flags.Bool("include-published", false, "include healthy published provenance records")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge audit: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge audit: get current directory: %v\n", err)
		return 1
	}
	result, err := audit.Knowledge(root, audit.Request{
		Project: *project, IncludePublished: *includePublished,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge audit: %v\n", err)
		return 1
	}
	if code := printJSON(stdout, stderr, result); code != 0 {
		return code
	}
	if result.Result == "problem" {
		return 1
	}
	return 0
}
```

Update `printUsage`:

```go
fmt.Fprintln(w, "  argos knowledge audit --json")
```

- [ ] **Step 7: Run audit CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgeAudit' -count=1
```

Expected: PASS.

- [ ] **Step 8: Run audit and CLI package tests**

Run:

```bash
go test ./internal/audit ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit audit service and CLI**

```bash
git add internal/audit internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: audit knowledge provenance"
```

---

### Task 5: Documentation And Final Verification

**Files:**
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`

- [ ] **Step 1: Update README commands**

In `README.md`, add these lines to the agent/internal commands block after provenance verify:

```bash
argos provenance list --json
argos provenance status --json --provenance <id>
argos knowledge audit --json
```

Add this paragraph under `### Provenance Contract`:

```markdown
Use `argos provenance status --json --provenance <id>` to inspect one write
attempt before publishing. Use `argos knowledge audit --json` to summarize open
evidence work across the repository. These commands organize evidence for
personal review and PR review; they do not grant approval or replace the
repository review process.
```

- [ ] **Step 2: Update capture skill**

In `skills/capture-knowledge/SKILL.md`, add to the publish section before the final publish command:

```markdown
Before publishing, run `argos provenance status --json --provenance PROVENANCE_ID`.
If the user asks for a broader review queue, run `argos knowledge audit --json`
and summarize categories, findings, and actions. Treat audit/status as evidence
organization only; the user or PR reviewer decides whether to publish or merge.
```

- [ ] **Step 3: Scan live docs for forbidden approval wording**

Run:

```bash
rg -n "Argos approved|Argos review passed|PR approved|merge allowed|review\\.draft_write_approved|review\\.design_approved|review\\.publish_approved" README.md skills/capture-knowledge/SKILL.md internal
```

Expected: no live guidance matches except tests that intentionally assert absence.

- [ ] **Step 4: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS for all packages.

- [ ] **Step 5: Run diff check**

Run:

```bash
git diff --check
```

Expected: no output and exit code 0.

- [ ] **Step 6: Commit docs**

```bash
git add README.md skills/capture-knowledge/SKILL.md
git commit -m "docs: document knowledge audit workflow"
```

---

## Final Review Gate

After Task 5:

1. Request code review with `superpowers:requesting-code-review`.
2. Fix Critical and Important findings.
3. Re-run:

```bash
go test ./... -count=1
git diff --check
```

4. Use `superpowers:finishing-a-development-branch` to choose merge, PR, keep branch, or discard.
