# Argos Agent Knowledge Authoring Protocol Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a workflow-neutral authoring protocol so agents can inspect, propose, verify, and review AI-authored engineering knowledge before it enters trusted Argos knowledge.

**Architecture:** Add a new `internal/author` package that owns proposal contracts, inspection facts, policy checks, and future retrieval simulation. Keep CLI wiring thin in `internal/cli`, reuse existing `knowledge`, `registry`, `index`, and `query` packages, and leave MCP mutation out of this slice.

**Tech Stack:** Go, standard `flag` CLI parsing, JSON serialization, existing Argos Markdown/YAML knowledge parser, SQLite-backed `internal/index`, `internal/query`, existing CLI test helpers, Markdown docs, and local skill files.

---

## File Structure

- Create `internal/author/proposal.go`
  - Define the canonical `authoring.proposal.v1` JSON contract.
  - Validate proposal completeness, policy-critical fields, authoring modes, delivery decisions, and future retrieval scenarios.
- Create `internal/author/proposal_test.go`
  - Lock required fields, mode-specific validation, and `priority: must` authorization behavior.
- Create `internal/author/inspect.go`
  - Implement `Inspect(root, InspectRequest)` for registry facts, official/inbox overlap, index availability, filesystem fallback overlap, candidate path risk, policy, and next steps.
- Create `internal/author/inspect_test.go`
  - Cover registry reporting, official overlap, inbox overlap, no-index behavior, and unsafe candidate paths.
- Create `internal/author/verify.go`
  - Implement `Verify(root, VerifyRequest)` for proposal loading, candidate validation, policy checks, metadata alignment, and future retrieval simulation in a temporary index.
- Create `internal/author/verify_test.go`
  - Cover pass, fail, and review-needed outcomes without mutating `argos/index.db`.
- Modify `internal/cli/cli.go`
  - Add top-level `author` command with `inspect` and `verify` subcommands.
  - Parse repeated flags with existing `multiValueFlag`.
  - Keep `--json` mandatory.
- Modify `internal/cli/cli_test.go`
  - Cover `author inspect --json`, required flag errors, `author verify --json`, and verification result JSON.
- Modify `internal/workspace/workspace.go`
  - Initialize `knowledge/.inbox/items`, `knowledge/.inbox/packages`, and `knowledge/.inbox/proposals`.
- Modify `internal/workspace/workspace_test.go`
  - Assert new inbox subdirectories are created by `argos init`.
- Modify `README.md`
  - Reframe write-side UX as agent-operated authoring with inspect, proposal, verify, review, and explicit promote.
- Modify `skills/capture-knowledge/SKILL.md`
  - Keep compatibility name but change the workflow vocabulary from capture to agent knowledge authoring.
- Create `testdata/authoring-golden/cases.json`
  - Add initial authoring harness cases for user-specified, observed, synthesized, overlap, findability, and `priority: must` guardrails.
- Create `internal/authoringtest/cases.go`
  - Load authoring golden cases and expose public summaries without hidden expected values.
- Create `internal/authoringtest/cases_test.go`
  - Ensure all required case types exist and public summaries do not leak expected IDs, ideal proposal contents, or pass/fail oracles.

---

### Task 1: Proposal Contract And Validation

**Files:**
- Create: `internal/author/proposal.go`
- Create: `internal/author/proposal_test.go`

- [ ] **Step 1: Write failing proposal validation tests**

Create `internal/author/proposal_test.go`:

```go
package author

import (
	"strings"
	"testing"
)

func TestValidateProposalAcceptsCompleteSynthesizedDraft(t *testing.T) {
	proposal := validProposal()
	findings := ValidateProposal(proposal)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestValidateProposalRequiresFutureRetrievalContract(t *testing.T) {
	proposal := validProposal()
	proposal.FutureRetrievalContract.Tasks = nil
	proposal.FutureRetrievalContract.QueryPhrases = nil
	proposal.FutureRetrievalContract.Projects = nil

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "future_retrieval_contract") {
		t.Fatalf("expected future retrieval failure, got %#v", findings)
	}
}

func TestValidateProposalRequiresSourceAndTrust(t *testing.T) {
	proposal := validProposal()
	proposal.SourceAndTrust.Assumptions = nil

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "review-needed", "source_and_trust.assumptions") {
		t.Fatalf("expected synthesized assumptions review finding, got %#v", findings)
	}
}

func TestValidateProposalRejectsPriorityMustWithoutAuthorization(t *testing.T) {
	proposal := validProposal()
	proposal.ProposedShape.Priority = "must"
	proposal.Delivery.PriorityMustAuthorized = false

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "priority: must") {
		t.Fatalf("expected must authorization failure, got %#v", findings)
	}
}

func TestValidateProposalRejectsUnknownMode(t *testing.T) {
	proposal := validProposal()
	proposal.AuthoringMode = "memo"

	findings := ValidateProposal(proposal)
	if !hasFinding(findings, "fail", "authoring_mode") {
		t.Fatalf("expected authoring mode failure, got %#v", findings)
	}
}

func validProposal() Proposal {
	return Proposal{
		SchemaVersion: "authoring.proposal.v1",
		KnowledgeGoal: "Create product-list cache engineering knowledge for future backend agents.",
		AuthoringMode: ModeSynthesized,
		Project: "mall-api",
		ProposedShape: ProposedShape{
			Kind: "package",
			Type: "package",
			Title: "Product List Cache Engineering Knowledge",
			ID: "package:backend.product-list-cache.v1",
			Path: "knowledge/.inbox/packages/backend/product-list-cache",
			Status: "draft",
			Priority: "should",
		},
		FutureRetrievalContract: FutureRetrievalContract{
			Tasks: []string{"implement product list cache"},
			Phases: []string{"implementation", "review"},
			Files: []string{"internal/catalog/products.go"},
			FileGlobs: []string{"internal/catalog/**"},
			QueryPhrases: []string{"product list cache", "cache ttl product list"},
			Projects: []string{"mall-api"},
			TechDomains: []string{"backend", "database"},
			BusinessDomains: []string{"catalog"},
			Tags: []string{"cache", "redis"},
			ExpectedUse: "read_before_implementation",
			CiteAfterUse: true,
		},
		SourceAndTrust: SourceAndTrust{
			Synthesized: []string{"General cache design practice adapted to the project context."},
			Assumptions: []string{"The project uses Redis for product-list cache storage."},
		},
		Applicability: Applicability{
			WhenToUse: []string{"When implementing product-list cache behavior."},
			WhenNotToUse: []string{"When changing unrelated catalog reads."},
			Tradeoffs: []string{"Caching improves latency but can serve stale data."},
		},
		OverlapDecision: OverlapDecision{
			Decision: "create_new",
			Reason: "No existing product-list cache knowledge covers this future task.",
		},
		Delivery: Delivery{
			Path: "inbox",
			RequiresHumanApproval: true,
			PriorityMustAuthorized: false,
			PromoteAuthorized: false,
		},
		CandidateFiles: []CandidateFile{
			{Path: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", Purpose: "entrypoint", Load: "start_here"},
		},
		VerificationPlan: VerificationPlan{
			ValidatePath: "knowledge/.inbox/packages/backend/product-list-cache",
			FindabilityScenarios: []FindabilityScenario{
				{
					Project: "mall-api",
					Phase: "implementation",
					Task: "implement product list cache",
					Query: "cache ttl product list",
					Files: []string{"internal/catalog/products.go"},
				},
			},
		},
	}
}

func hasFinding(findings []Finding, severity string, text string) bool {
	for _, finding := range findings {
		if finding.Severity == severity && strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/author -run 'TestValidateProposal' -count=1
```

Expected: FAIL because `internal/author` and the proposal types do not exist yet.

- [ ] **Step 3: Implement the proposal contract**

Create `internal/author/proposal.go`:

```go
package author

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const ProposalSchemaVersion = "authoring.proposal.v1"

const (
	ModeObserved      = "observed"
	ModeUserSpecified = "user_specified"
	ModeSynthesized   = "synthesized"
	ModeImported      = "imported"
	ModeTemplate      = "template"
)

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Proposal struct {
	SchemaVersion          string                  `json:"schema_version"`
	KnowledgeGoal          string                  `json:"knowledge_goal"`
	AuthoringMode          string                  `json:"authoring_mode"`
	Project                string                  `json:"project"`
	ProposedShape          ProposedShape           `json:"proposed_shape"`
	FutureRetrievalContract FutureRetrievalContract `json:"future_retrieval_contract"`
	SourceAndTrust         SourceAndTrust          `json:"source_and_trust"`
	Applicability          Applicability           `json:"applicability"`
	OverlapDecision        OverlapDecision         `json:"overlap_decision"`
	Delivery               Delivery                `json:"delivery"`
	CandidateFiles         []CandidateFile         `json:"candidate_files"`
	VerificationPlan       VerificationPlan        `json:"verification_plan"`
}

type ProposedShape struct {
	Kind     string `json:"kind"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	ID       string `json:"id"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

type FutureRetrievalContract struct {
	Tasks           []string `json:"tasks"`
	Phases          []string `json:"phases"`
	Files           []string `json:"files"`
	FileGlobs       []string `json:"file_globs"`
	QueryPhrases    []string `json:"query_phrases"`
	Projects        []string `json:"projects"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
	Tags            []string `json:"tags"`
	ExpectedUse     string   `json:"expected_use"`
	CiteAfterUse    bool     `json:"cite_after_use"`
}

type SourceAndTrust struct {
	UserProvided  []string `json:"user_provided"`
	Observed      []string `json:"observed"`
	Imported      []string `json:"imported"`
	Synthesized   []string `json:"synthesized"`
	Assumptions   []string `json:"assumptions"`
	OpenQuestions []string `json:"open_questions"`
}

type Applicability struct {
	WhenToUse    []string `json:"when_to_use"`
	WhenNotToUse []string `json:"when_not_to_use"`
	Tradeoffs    []string `json:"tradeoffs"`
}

type OverlapDecision struct {
	OfficialOverlap []string `json:"official_overlap"`
	InboxOverlap    []string `json:"inbox_overlap"`
	Decision        string   `json:"decision"`
	Reason          string   `json:"reason"`
}

type Delivery struct {
	Path                  string `json:"path"`
	RequiresHumanApproval bool   `json:"requires_human_approval"`
	PriorityMustAuthorized bool   `json:"priority_must_authorized"`
	PromoteAuthorized     bool   `json:"promote_authorized"`
}

type CandidateFile struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Load    string `json:"load"`
}

type VerificationPlan struct {
	ValidatePath          string                `json:"validate_path"`
	FindabilityScenarios []FindabilityScenario `json:"findability_scenarios"`
}

type FindabilityScenario struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Query   string   `json:"query"`
	Files   []string `json:"files"`
}

func LoadProposal(path string) (Proposal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Proposal{}, fmt.Errorf("read proposal: %w", err)
	}
	var proposal Proposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return Proposal{}, fmt.Errorf("parse proposal JSON: %w", err)
	}
	return proposal, nil
}

func ValidateProposal(proposal Proposal) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}

	if strings.TrimSpace(proposal.SchemaVersion) != ProposalSchemaVersion {
		addFail("schema_version must be authoring.proposal.v1")
	}
	if strings.TrimSpace(proposal.KnowledgeGoal) == "" {
		addFail("knowledge_goal is required")
	}
	if !validMode(proposal.AuthoringMode) {
		addFail("authoring_mode must be observed, user_specified, synthesized, imported, or template")
	}
	if strings.TrimSpace(proposal.Project) == "" {
		addFail("project is required")
	}
	if strings.TrimSpace(proposal.ProposedShape.Kind) == "" || strings.TrimSpace(proposal.ProposedShape.Type) == "" || strings.TrimSpace(proposal.ProposedShape.ID) == "" || strings.TrimSpace(proposal.ProposedShape.Path) == "" {
		addFail("proposed_shape must include kind, type, id, and path")
	}
	if strings.TrimSpace(proposal.ProposedShape.Status) == "" || strings.TrimSpace(proposal.ProposedShape.Priority) == "" {
		addFail("proposed_shape must include status and priority")
	}
	if strings.TrimSpace(proposal.ProposedShape.Priority) == "must" && !proposal.Delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	if len(nonEmpty(proposal.FutureRetrievalContract.Tasks)) == 0 && len(nonEmpty(proposal.FutureRetrievalContract.QueryPhrases)) == 0 {
		addFail("future_retrieval_contract must include tasks or query_phrases")
	}
	if len(nonEmpty(proposal.FutureRetrievalContract.Projects)) == 0 {
		addFail("future_retrieval_contract.projects is required")
	}
	if strings.TrimSpace(proposal.FutureRetrievalContract.ExpectedUse) == "" {
		addFail("future_retrieval_contract.expected_use is required")
	}
	if proposal.AuthoringMode == ModeSynthesized && len(nonEmpty(proposal.SourceAndTrust.Assumptions)) == 0 {
		addReview("source_and_trust.assumptions is required for synthesized authoring")
	}
	if !hasAnySource(proposal.SourceAndTrust) {
		addReview("source_and_trust must include at least one source bucket")
	}
	if len(nonEmpty(proposal.Applicability.WhenToUse)) == 0 || len(nonEmpty(proposal.Applicability.WhenNotToUse)) == 0 {
		addReview("applicability must include when_to_use and when_not_to_use")
	}
	if !validOverlapDecision(proposal.OverlapDecision.Decision) {
		addFail("overlap_decision.decision must be create_new, update_existing, stop, or unresolved")
	}
	if strings.TrimSpace(proposal.OverlapDecision.Reason) == "" {
		addReview("overlap_decision.reason is required")
	}
	if proposal.Delivery.Path != "inbox" && proposal.Delivery.Path != "official_review" {
		addFail("delivery.path must be inbox or official_review")
	}
	if !proposal.Delivery.RequiresHumanApproval {
		addFail("delivery.requires_human_approval must be true")
	}
	if len(proposal.CandidateFiles) == 0 {
		addFail("candidate_files must include at least one file")
	}
	if strings.TrimSpace(proposal.VerificationPlan.ValidatePath) == "" {
		addFail("verification_plan.validate_path is required")
	}
	if len(proposal.VerificationPlan.FindabilityScenarios) == 0 {
		addReview("verification_plan.findability_scenarios should include at least one scenario")
	}
	return findings
}

func validMode(mode string) bool {
	switch mode {
	case ModeObserved, ModeUserSpecified, ModeSynthesized, ModeImported, ModeTemplate:
		return true
	default:
		return false
	}
}

func validOverlapDecision(decision string) bool {
	switch decision {
	case "create_new", "update_existing", "stop", "unresolved":
		return true
	default:
		return false
	}
}

func hasAnySource(source SourceAndTrust) bool {
	return len(nonEmpty(source.UserProvided)) > 0 || len(nonEmpty(source.Observed)) > 0 || len(nonEmpty(source.Imported)) > 0 || len(nonEmpty(source.Synthesized)) > 0
}

func nonEmpty(values []string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}
```

- [ ] **Step 4: Run proposal tests**

Run:

```bash
go test ./internal/author -run 'TestValidateProposal' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit proposal contract**

Run:

```bash
git add internal/author/proposal.go internal/author/proposal_test.go
git commit -m "feat: add authoring proposal contract"
```

---

### Task 2: Author Inspect Service

**Files:**
- Create: `internal/author/inspect.go`
- Create: `internal/author/inspect_test.go`

- [ ] **Step 1: Write failing inspect tests**

Create `internal/author/inspect_test.go`:

```go
package author

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectReportsRegistryPolicyAndNoIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal: "create product-list cache knowledge",
		Mode: ModeSynthesized,
		FutureTask: "implement product list cache",
		Phase: "implementation",
		Files: []string{"internal/catalog/products.go"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Project != "mall-api" || result.Goal == "" || result.Mode != ModeSynthesized {
		t.Fatalf("unexpected inspect identity: %#v", result)
	}
	if result.Capabilities.Filesystem != "enabled" || result.Capabilities.Index != "unavailable" {
		t.Fatalf("unexpected capabilities: %#v", result.Capabilities)
	}
	if !result.Registry.ProjectKnown {
		t.Fatalf("expected known project: %#v", result.Registry)
	}
	if result.Policy.PriorityMust != "requires_explicit_authorization" {
		t.Fatalf("unexpected policy: %#v", result.Policy)
	}
	if len(result.ProposalRequirements) == 0 || result.RecommendedNextSteps[0].Step != "write_knowledge_design_proposal" {
		t.Fatalf("expected proposal next step: %#v", result)
	}
}

func TestInspectFindsOfficialAndInboxOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "draft", "Product List Cache Package"))

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal: "create product-list cache knowledge",
		FutureTask: "product list cache ttl",
		Query: "product list cache",
		Tags: []string{"cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(result.Overlap.Official) == 0 {
		t.Fatalf("expected official overlap: %#v", result.Overlap)
	}
	if len(result.Overlap.Inbox) == 0 {
		t.Fatalf("expected inbox overlap: %#v", result.Overlap)
	}
	if result.Overlap.Official[0].Kind != "official" || result.Overlap.Inbox[0].Kind != "inbox" {
		t.Fatalf("unexpected overlap kinds: %#v", result.Overlap)
	}
}

func TestInspectReportsCandidatePathRisk(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal: "create product-list cache knowledge",
		CandidatePath: "../outside",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.PathRisk.Status != "unsafe" {
		t.Fatalf("expected unsafe path risk, got %#v", result.PathRisk)
	}
}

func writeAuthorRegistry(t *testing.T, root string) {
	t.Helper()
	writeAuthorFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend, database]\nbusiness_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/types.yaml", "types: [rule, decision, lesson, runbook, reference, package]\n")
}

func writeAuthorFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func authorItem(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: rule
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache]
applies_to:
  files: [internal/catalog/**]
---
Product list cache TTL guidance for backend work.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}

func authorPackage(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache, redis]
applies_to:
  files: [internal/catalog/**]
---
## Purpose

Product list cache package.

## When To Use

Use when implementing product list cache behavior.

## Start Here

Read this entrypoint before implementation.

## Load On Demand

Load supporting references only when needed.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}
```

- [ ] **Step 2: Run inspect tests and verify they fail**

Run:

```bash
go test ./internal/author -run 'TestInspect' -count=1
```

Expected: FAIL because `Inspect`, `InspectRequest`, and response types do not exist.

- [ ] **Step 3: Implement inspect response types and core flow**

Create `internal/author/inspect.go`:

```go
package author

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/registry"
)

type InspectRequest struct {
	Project       string   `json:"project"`
	Goal          string   `json:"goal"`
	Mode          string   `json:"mode,omitempty"`
	FutureTask    string   `json:"future_task,omitempty"`
	Phase         string   `json:"phase,omitempty"`
	Query         string   `json:"query,omitempty"`
	Files         []string `json:"files,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CandidatePath string   `json:"candidate_path,omitempty"`
}

type InspectResponse struct {
	Project              string                 `json:"project"`
	Goal                 string                 `json:"goal"`
	Mode                 string                 `json:"mode,omitempty"`
	Capabilities         InspectCapabilities    `json:"capabilities"`
	Registry             InspectRegistry        `json:"registry"`
	Overlap              InspectOverlap         `json:"overlap"`
	PathRisk             PathRisk               `json:"path_risk"`
	Policy               InspectPolicy          `json:"policy"`
	ProposalRequirements []string               `json:"proposal_requirements"`
	RecommendedNextSteps []InspectNextStep      `json:"recommended_next_steps"`
}

type InspectCapabilities struct {
	Filesystem string `json:"filesystem"`
	Index      string `json:"index"`
}

type InspectRegistry struct {
	ProjectKnown    bool     `json:"project_known"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
}

type InspectOverlap struct {
	Official []OverlapMatch `json:"official"`
	Inbox    []OverlapMatch `json:"inbox"`
	Index    []OverlapMatch `json:"index"`
}

type OverlapMatch struct {
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Path   string   `json:"path"`
	Reason []string `json:"reason"`
}

type PathRisk struct {
	CandidatePath string `json:"candidate_path"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
}

type InspectPolicy struct {
	Write             string `json:"write"`
	OfficialMutation  string `json:"official_mutation"`
	Promote           string `json:"promote"`
	PriorityMust      string `json:"priority_must"`
	SynthesizedClaims string `json:"synthesized_claims"`
}

type InspectNextStep struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

func Inspect(root string, req InspectRequest) (InspectResponse, error) {
	reg, err := registry.Load(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load registry: %w", err)
	}
	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load official knowledge: %w", err)
	}
	inbox, err := knowledge.LoadInbox(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load inbox knowledge: %w", err)
	}

	response := InspectResponse{
		Project: strings.TrimSpace(req.Project),
		Goal: strings.TrimSpace(req.Goal),
		Mode: strings.TrimSpace(req.Mode),
		Capabilities: InspectCapabilities{Filesystem: "enabled", Index: indexStatus(root)},
		Registry: InspectRegistry{
			ProjectKnown: projectKnown(reg, req.Project),
			TechDomains: append([]string{}, reg.TechDomains...),
			BusinessDomains: append([]string{}, reg.BusinessDomains...),
		},
		Overlap: InspectOverlap{
			Official: overlapMatches("official", official, req),
			Inbox: overlapMatches("inbox", inbox, req),
		},
		PathRisk: inspectPathRisk(req.CandidatePath),
		Policy: InspectPolicy{
			Write: "after_proposal_approval",
			OfficialMutation: "requires_explicit_review_path",
			Promote: "requires_explicit_approval",
			PriorityMust: "requires_explicit_authorization",
			SynthesizedClaims: "must_mark_assumptions",
		},
		ProposalRequirements: []string{
			"future_retrieval_contract",
			"source_and_trust",
			"applicability",
			"overlap_decision",
			"verification_plan",
		},
		RecommendedNextSteps: []InspectNextStep{
			{Step: "write_knowledge_design_proposal", Reason: "Human review is required before durable writes."},
		},
	}
	response.Overlap.Index = indexOverlap(root, req)
	return response, nil
}
```

- [ ] **Step 4: Add inspect helpers**

Append these helpers to `internal/author/inspect.go`:

```go
func projectKnown(reg registry.Registry, project string) bool {
	for _, candidate := range reg.Projects {
		if candidate.ID == strings.TrimSpace(project) {
			return true
		}
	}
	return false
}

func indexStatus(root string) string {
	info, err := os.Stat(filepath.Join(root, "argos", "index.db"))
	if err != nil || !info.Mode().IsRegular() {
		return "unavailable"
	}
	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		return "unavailable"
	}
	defer store.Close()
	if err := store.CheckSchema(); err != nil {
		return "unavailable"
	}
	return "enabled"
}

func inspectPathRisk(path string) PathRisk {
	path = strings.TrimSpace(path)
	if path == "" {
		return PathRisk{Status: "not_checked"}
	}
	if filepath.IsAbs(path) {
		return PathRisk{CandidatePath: path, Status: "unsafe", Reason: "candidate path must be relative"}
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return PathRisk{CandidatePath: path, Status: "unsafe", Reason: "candidate path must stay inside workspace"}
	}
	slash := filepath.ToSlash(clean)
	if strings.HasPrefix(slash, "knowledge/.inbox/items/") || strings.HasPrefix(slash, "knowledge/.inbox/packages/") {
		return PathRisk{CandidatePath: clean, Status: "allowed"}
	}
	if strings.HasPrefix(slash, "knowledge/items/") || strings.HasPrefix(slash, "knowledge/packages/") {
		return PathRisk{CandidatePath: clean, Status: "official_review_required", Reason: "official mutation requires explicit review path"}
	}
	return PathRisk{CandidatePath: clean, Status: "review-needed", Reason: "candidate path is outside standard authoring inbox locations"}
}

func overlapMatches(kind string, items []knowledge.Item, req InspectRequest) []OverlapMatch {
	terms := overlapTerms(req)
	var matches []OverlapMatch
	for _, item := range items {
		reasons := overlapReasons(item, req, terms)
		if len(reasons) == 0 {
			continue
		}
		matches = append(matches, OverlapMatch{
			Kind: kind,
			ID: item.ID,
			Title: item.Title,
			Path: item.Path,
			Reason: reasons,
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i].Reason) != len(matches[j].Reason) {
			return len(matches[i].Reason) > len(matches[j].Reason)
		}
		return matches[i].ID < matches[j].ID
	})
	return matches
}

func overlapTerms(req InspectRequest) []string {
	text := strings.Join([]string{req.Goal, req.FutureTask, req.Query}, " ")
	text = strings.ToLower(text)
	var terms []string
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/' || r == ':' || r == ',' || r == '.'
	}) {
		field = strings.TrimSpace(field)
		if len(field) >= 4 {
			terms = append(terms, field)
		}
	}
	terms = append(terms, req.Tags...)
	terms = append(terms, req.Domains...)
	return uniqueNonEmpty(terms)
}

func overlapReasons(item knowledge.Item, req InspectRequest, terms []string) []string {
	var reasons []string
	if containsStringValue(item.Projects, strings.TrimSpace(req.Project)) {
		reasons = append(reasons, "project")
	}
	for _, tag := range req.Tags {
		if containsStringValue(item.Tags, tag) {
			reasons = append(reasons, "tag:"+tag)
		}
	}
	body := strings.ToLower(strings.Join([]string{item.ID, item.Title, item.Body, strings.Join(item.Tags, " ")}, " "))
	for _, term := range terms {
		if strings.Contains(body, strings.ToLower(term)) {
			reasons = append(reasons, "term:"+term)
		}
	}
	return uniqueNonEmpty(reasons)
}

func indexOverlap(root string, req InspectRequest) []OverlapMatch {
	if indexStatus(root) != "enabled" {
		return nil
	}
	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		return nil
	}
	defer store.Close()
	matches, err := store.SearchText(strings.Join([]string{req.Goal, req.FutureTask, req.Query}, " "), 10)
	if err != nil {
		return nil
	}
	var out []OverlapMatch
	seen := map[string]bool{}
	for _, match := range matches {
		if seen[match.ItemID] {
			continue
		}
		seen[match.ItemID] = true
		item, err := store.GetItem(match.ItemID)
		if err != nil {
			continue
		}
		out = append(out, OverlapMatch{Kind: "index", ID: item.ID, Title: item.Title, Path: item.Path, Reason: []string{"index_text_match"}})
	}
	return out
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want && strings.TrimSpace(want) != "" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run inspect tests**

Run:

```bash
go test ./internal/author -run 'TestInspect' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit inspect service**

Run:

```bash
git add internal/author/inspect.go internal/author/inspect_test.go
git commit -m "feat: add authoring inspect service"
```

---

### Task 3: Author Verify Service

**Files:**
- Create: `internal/author/verify.go`
- Create: `internal/author/verify_test.go`

- [ ] **Step 1: Write failing verification tests**

Create `internal/author/verify_test.go`:

```go
package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyPassesValidInboxCandidateAndDoesNotMutateWorkspaceIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	proposal := validProposal()
	writeProposal(t, root, "knowledge/.inbox/proposals/product-list-cache/proposal.json", proposal)
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "draft", "Product List Cache Engineering Knowledge"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath: "knowledge/.inbox/proposals/product-list-cache/proposal.json",
		CandidatePath: "knowledge/.inbox/packages/backend/product-list-cache",
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "pass" {
		t.Fatalf("expected pass, got %#v", result)
	}
	if result.Candidate.Validation != "pass" || result.Findability.Result != "pass" {
		t.Fatalf("expected candidate and findability pass, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "argos", "index.db")); !os.IsNotExist(err) {
		t.Fatalf("verify must not create workspace index.db, stat error=%v", err)
	}
}

func TestVerifyFailsCandidateOutsideApprovedBoundary(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	proposal := validProposal()
	proposal.VerificationPlan.ValidatePath = "knowledge/packages/backend/product-list-cache"
	proposal.ProposedShape.Path = "knowledge/packages/backend/product-list-cache"
	writeProposal(t, root, "knowledge/.inbox/proposals/product-list-cache/proposal.json", proposal)
	writeAuthorFile(t, root, "knowledge/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "active", "Product List Cache Engineering Knowledge"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath: "knowledge/.inbox/proposals/product-list-cache/proposal.json",
		CandidatePath: "knowledge/packages/backend/product-list-cache",
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" || !hasFinding(result.Findings, "fail", "approved delivery boundary") {
		t.Fatalf("expected boundary failure, got %#v", result)
	}
}

func TestVerifyReturnsReviewNeededWhenCandidateIsNotFindable(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	proposal := validProposal()
	proposal.VerificationPlan.FindabilityScenarios[0].Task = "payments webhook retry strategy"
	proposal.VerificationPlan.FindabilityScenarios[0].Query = "payments webhook retry"
	writeProposal(t, root, "knowledge/.inbox/proposals/product-list-cache/proposal.json", proposal)
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "draft", "Product List Cache Engineering Knowledge"))

	result, err := Verify(root, VerifyRequest{
		ProposalPath: "knowledge/.inbox/proposals/product-list-cache/proposal.json",
		CandidatePath: "knowledge/.inbox/packages/backend/product-list-cache",
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "review-needed" || result.Findability.Result != "review-needed" {
		t.Fatalf("expected review-needed findability, got %#v", result)
	}
}

func writeProposal(t *testing.T, root string, rel string, proposal Proposal) {
	t.Helper()
	data, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	writeAuthorFile(t, root, rel, string(data))
}
```

- [ ] **Step 2: Run verification tests and verify they fail**

Run:

```bash
go test ./internal/author -run 'TestVerify' -count=1
```

Expected: FAIL because `Verify` and response types do not exist.

- [ ] **Step 3: Implement verification response types and main flow**

Create `internal/author/verify.go`:

```go
package author

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/query"
	"argos/internal/registry"
)

type VerifyRequest struct {
	ProposalPath  string `json:"proposal_path"`
	CandidatePath string `json:"candidate_path"`
}

type VerifyResponse struct {
	Result      string             `json:"result"`
	Proposal    ProposalStatus     `json:"proposal"`
	Candidate   CandidateStatus    `json:"candidate"`
	Policy      PolicyStatus       `json:"policy"`
	Findability FindabilityStatus  `json:"findability"`
	Findings    []Finding          `json:"findings,omitempty"`
}

type ProposalStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

type CandidateStatus struct {
	Path       string `json:"path"`
	Validation string `json:"validation"`
	ID         string `json:"id,omitempty"`
}

type PolicyStatus struct {
	Result   string    `json:"result"`
	Findings []Finding `json:"findings,omitempty"`
}

type FindabilityStatus struct {
	Result    string                `json:"result"`
	Scenarios []FindabilityCheck    `json:"scenarios"`
}

type FindabilityCheck struct {
	Task             string `json:"task"`
	Query            string `json:"query"`
	MatchedCandidate bool   `json:"matched_candidate"`
	Support          string `json:"support"`
}

func Verify(root string, req VerifyRequest) (VerifyResponse, error) {
	proposalPath, err := cleanWorkspacePath(req.ProposalPath)
	if err != nil {
		return VerifyResponse{}, err
	}
	candidatePath, err := cleanWorkspacePath(req.CandidatePath)
	if err != nil {
		return VerifyResponse{}, err
	}

	proposal, err := LoadProposal(filepath.Join(root, proposalPath))
	if err != nil {
		return VerifyResponse{}, err
	}
	response := VerifyResponse{
		Result: "pass",
		Proposal: ProposalStatus{Path: proposalPath, Status: "valid"},
		Candidate: CandidateStatus{Path: candidatePath, Validation: "pass"},
		Policy: PolicyStatus{Result: "pass"},
		Findability: FindabilityStatus{Result: "pass"},
	}
	for _, finding := range ValidateProposal(proposal) {
		response.addFinding(finding)
	}

	reg, err := registry.Load(root)
	if err != nil {
		return VerifyResponse{}, fmt.Errorf("load registry: %w", err)
	}
	candidateItems, err := knowledge.LoadPath(root, candidatePath)
	if err != nil {
		response.Candidate.Validation = "fail"
		response.addFinding(Finding{Severity: "fail", Message: fmt.Sprintf("candidate validation failed: %v", err)})
		return response, nil
	}
	for _, validationErr := range knowledge.ValidateItems(candidateItems, reg) {
		response.Candidate.Validation = "fail"
		response.addFinding(Finding{Severity: "fail", Message: validationErr.Error()})
	}
	if len(candidateItems) > 0 {
		response.Candidate.ID = candidateItems[0].ID
	}

	for _, finding := range policyFindings(proposal, candidatePath, candidateItems) {
		response.Policy.Findings = append(response.Policy.Findings, finding)
		response.addFinding(finding)
	}
	response.Policy.Result = resultFromFindings(response.Policy.Findings)

	if response.Result != "fail" && len(candidateItems) > 0 {
		findability, err := simulateFindability(root, proposal, candidateItems[0])
		if err != nil {
			response.addFinding(Finding{Severity: "review-needed", Message: fmt.Sprintf("findability simulation unavailable: %v", err)})
			response.Findability.Result = "review-needed"
		} else {
			response.Findability = findability
			for _, check := range findability.Scenarios {
				if !check.MatchedCandidate || check.Support == "none" {
					response.addFinding(Finding{Severity: "review-needed", Message: "candidate was not findable for scenario: " + check.Task})
				}
			}
		}
	}
	response.Result = resultFromFindings(response.Findings)
	return response, nil
}
```

- [ ] **Step 4: Add verification helpers**

Append these helpers to `internal/author/verify.go`:

```go
func (r *VerifyResponse) addFinding(finding Finding) {
	r.Findings = append(r.Findings, finding)
	if finding.Severity == "fail" {
		r.Result = "fail"
		return
	}
	if finding.Severity == "review-needed" && r.Result == "pass" {
		r.Result = "review-needed"
	}
}

func resultFromFindings(findings []Finding) string {
	result := "pass"
	for _, finding := range findings {
		if finding.Severity == "fail" {
			return "fail"
		}
		if finding.Severity == "review-needed" {
			result = "review-needed"
		}
	}
	return result
}

func cleanWorkspacePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("%s: path must be relative", path)
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: path must stay inside workspace", path)
	}
	return clean, nil
}

func policyFindings(proposal Proposal, candidatePath string, candidateItems []knowledge.Item) []Finding {
	var findings []Finding
	addFail := func(message string) {
		findings = append(findings, Finding{Severity: "fail", Message: message})
	}
	addReview := func(message string) {
		findings = append(findings, Finding{Severity: "review-needed", Message: message})
	}
	slash := filepath.ToSlash(candidatePath)
	if proposal.Delivery.Path == "inbox" && !strings.HasPrefix(slash, "knowledge/.inbox/items/") && !strings.HasPrefix(slash, "knowledge/.inbox/packages/") {
		addFail("candidate path is outside approved delivery boundary")
	}
	if proposal.VerificationPlan.ValidatePath != "" && filepath.Clean(proposal.VerificationPlan.ValidatePath) != candidatePath {
		addFail("candidate path must match verification_plan.validate_path")
	}
	if proposal.ProposedShape.Path != "" && filepath.Clean(proposal.ProposedShape.Path) != candidatePath {
		addFail("candidate path must match proposed_shape.path")
	}
	if proposal.ProposedShape.Priority == "must" && !proposal.Delivery.PriorityMustAuthorized {
		addFail("priority: must requires explicit authorization")
	}
	for _, item := range candidateItems {
		if proposal.Delivery.Path == "inbox" && item.Status != "draft" {
			addFail("inbox candidates must use status: draft")
		}
		if proposal.ProposedShape.ID != "" && item.ID != proposal.ProposedShape.ID {
			addFail("candidate id must match proposed_shape.id")
		}
		if proposal.ProposedShape.Type != "" && item.Type != proposal.ProposedShape.Type {
			addFail("candidate type must match proposed_shape.type")
		}
		for _, project := range proposal.FutureRetrievalContract.Projects {
			if !containsStringValue(item.Projects, project) {
				addReview("candidate projects do not include future retrieval project: " + project)
			}
		}
		for _, tag := range proposal.FutureRetrievalContract.Tags {
			if !containsStringValue(item.Tags, tag) {
				addReview("candidate tags do not include future retrieval tag: " + tag)
			}
		}
	}
	return findings
}

func simulateFindability(root string, proposal Proposal, candidate knowledge.Item) (FindabilityStatus, error) {
	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return FindabilityStatus{}, err
	}
	items := append(append([]knowledge.Item{}, official...), candidate)
	tempDir, err := os.MkdirTemp("", "argos-author-verify-*")
	if err != nil {
		return FindabilityStatus{}, err
	}
	defer os.RemoveAll(tempDir)
	dbPath := filepath.Join(tempDir, "index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		return FindabilityStatus{}, err
	}
	store, err := index.Open(dbPath)
	if err != nil {
		return FindabilityStatus{}, err
	}
	defer store.Close()

	service := query.New(store)
	status := FindabilityStatus{Result: "pass"}
	for _, scenario := range proposal.VerificationPlan.FindabilityScenarios {
		result, err := service.FindKnowledge(query.FindKnowledgeRequest{
			Project: scenario.Project,
			Phase: scenario.Phase,
			Task: scenario.Task,
			Query: scenario.Query,
			Files: scenario.Files,
			Limit: 8,
		})
		if err != nil {
			return FindabilityStatus{}, err
		}
		check := FindabilityCheck{
			Task: scenario.Task,
			Query: scenario.Query,
			Support: result.Support.Level,
		}
		for _, item := range result.Items {
			if item.ID == candidate.ID {
				check.MatchedCandidate = true
				break
			}
		}
		if !check.MatchedCandidate || check.Support == "none" {
			status.Result = "review-needed"
		}
		status.Scenarios = append(status.Scenarios, check)
	}
	return status, nil
}
```

- [ ] **Step 5: Run verification tests**

Run:

```bash
go test ./internal/author -run 'TestVerify' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run the full author package tests**

Run:

```bash
go test ./internal/author -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit verification service**

Run:

```bash
git add internal/author/verify.go internal/author/verify_test.go
git commit -m "feat: verify authoring proposals and candidates"
```

---

### Task 4: CLI Author Commands

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add these tests to `internal/cli/cli_test.go` near the existing validate and knowledge command tests:

```go
func TestRunAuthorInspectReturnsJSON(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"author", "inspect", "--json",
		"--project", "mall-api",
		"--goal", "create product-list cache engineering knowledge",
		"--mode", "synthesized",
		"--future-task", "implement product list cache",
		"--phase", "implementation",
		"--files", "internal/catalog/products.go",
		"--tags", "cache",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Project string `json:"project"`
		Policy struct {
			PriorityMust string `json:"priority_must"`
		} `json:"policy"`
		RecommendedNextSteps []struct {
			Step string `json:"step"`
		} `json:"recommended_next_steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Project != "mall-api" || result.Policy.PriorityMust != "requires_explicit_authorization" {
		t.Fatalf("unexpected inspect result: %s", stdout.String())
	}
	if len(result.RecommendedNextSteps) == 0 || result.RecommendedNextSteps[0].Step != "write_knowledge_design_proposal" {
		t.Fatalf("missing author next step: %s", stdout.String())
	}
}

func TestRunAuthorInspectRequiresJSONProjectAndGoal(t *testing.T) {
	for _, args := range [][]string{
		{"author", "inspect", "--project", "mall-api", "--goal", "create knowledge"},
		{"author", "inspect", "--json", "--goal", "create knowledge"},
		{"author", "inspect", "--json", "--project", "mall-api"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run(args, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("expected exit code 2 for %#v, got %d", args, code)
		}
	}
}

func TestRunAuthorVerifyReturnsJSON(t *testing.T) {
	root := t.TempDir()
	writeCLIAuthorRegistry(t, root)
	proposal := cliAuthorProposal()
	writeCLIProposal(t, root, "knowledge/.inbox/proposals/product-list-cache/proposal.json", proposal)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", validCLIAuthorPackage())
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"author", "verify", "--json",
		"--proposal", "knowledge/.inbox/proposals/product-list-cache/proposal.json",
		"--path", "knowledge/.inbox/packages/backend/product-list-cache",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var result struct {
		Result string `json:"result"`
		Candidate struct {
			Validation string `json:"validation"`
		} `json:"candidate"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Result != "pass" || result.Candidate.Validation != "pass" {
		t.Fatalf("unexpected verify result: %s", stdout.String())
	}
}
```

Add test helpers below existing CLI helpers:

```go
func writeCLIAuthorRegistry(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend, database]\nbusiness_domains: [catalog]\n")
	writeCLIFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule, decision, lesson, runbook, reference, package]\n")
}

func cliAuthorProposal() map[string]any {
	return map[string]any{
		"schema_version": "authoring.proposal.v1",
		"knowledge_goal": "Create product-list cache engineering knowledge.",
		"authoring_mode": "synthesized",
		"project": "mall-api",
		"proposed_shape": map[string]any{
			"kind": "package",
			"type": "package",
			"title": "Product List Cache Engineering Knowledge",
			"id": "package:backend.product-list-cache.v1",
			"path": "knowledge/.inbox/packages/backend/product-list-cache",
			"status": "draft",
			"priority": "should",
		},
		"future_retrieval_contract": map[string]any{
			"tasks": []string{"implement product list cache"},
			"phases": []string{"implementation"},
			"files": []string{"internal/catalog/products.go"},
			"file_globs": []string{"internal/catalog/**"},
			"query_phrases": []string{"product list cache", "cache ttl product list"},
			"projects": []string{"mall-api"},
			"tech_domains": []string{"backend"},
			"business_domains": []string{"catalog"},
			"tags": []string{"cache", "redis"},
			"expected_use": "read_before_implementation",
			"cite_after_use": true,
		},
		"source_and_trust": map[string]any{
			"synthesized": []string{"General cache practice adapted to project context."},
			"assumptions": []string{"Redis is available for product list caching."},
		},
		"applicability": map[string]any{
			"when_to_use": []string{"When implementing product list cache behavior."},
			"when_not_to_use": []string{"When changing unrelated catalog reads."},
			"tradeoffs": []string{"Cache freshness must be managed."},
		},
		"overlap_decision": map[string]any{
			"decision": "create_new",
			"reason": "No existing knowledge covers this task.",
		},
		"delivery": map[string]any{
			"path": "inbox",
			"requires_human_approval": true,
			"priority_must_authorized": false,
			"promote_authorized": false,
		},
		"candidate_files": []map[string]any{
			{"path": "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", "purpose": "entrypoint", "load": "start_here"},
		},
		"verification_plan": map[string]any{
			"validate_path": "knowledge/.inbox/packages/backend/product-list-cache",
			"findability_scenarios": []map[string]any{
				{"project": "mall-api", "phase": "implementation", "task": "implement product list cache", "query": "cache ttl product list", "files": []string{"internal/catalog/products.go"}},
			},
		},
	}
}

func validCLIAuthorPackage() string {
	return `---
id: package:backend.product-list-cache.v1
title: Product List Cache Engineering Knowledge
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
tags: [cache, redis]
updated_at: 2026-05-02
applies_to:
  files: [internal/catalog/**]
---
## Purpose

Product list cache engineering knowledge for future backend agents.

## When To Use

Use when implementing product list cache behavior.

## Start Here

Read before implementing cache ttl behavior for product list reads.

## Load On Demand

Load references when tuning Redis key design.
`
}

func writeCLIProposal(t *testing.T, root string, rel string, value map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	writeCLIFile(t, root, rel, string(data))
}
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunAuthor' -count=1
```

Expected: FAIL because `author` is not a recognized top-level command.

- [ ] **Step 3: Wire the author command in CLI**

In `internal/cli/cli.go`, add the import:

```go
	"argos/internal/author"
```

Add this case to the top-level `switch` in `run`:

```go
	case "author":
		return runAuthor(args[1:], stdout, stderr)
```

Add these functions near `runKnowledge`:

```go
func runAuthor(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "author: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "inspect":
		return runAuthorInspect(args[1:], stdout, stderr)
	case "verify":
		return runAuthorVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "author: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runAuthorInspect(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("author inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	goal := flags.String("goal", "", "knowledge authoring goal")
	mode := flags.String("mode", "", "authoring mode")
	futureTask := flags.String("future-task", "", "intended future task")
	phase := flags.String("phase", "", "workflow phase")
	queryText := flags.String("query", "", "search query")
	candidatePath := flags.String("candidate-path", "", "candidate item or package path")
	var files multiValueFlag
	var domains multiValueFlag
	var tags multiValueFlag
	flags.Var(&files, "files", "future file path; may be repeated")
	flags.Var(&domains, "domains", "domain; may be repeated")
	flags.Var(&tags, "tags", "tag; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "author inspect: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "author inspect: --project is required")
		return 2
	}
	if strings.TrimSpace(*goal) == "" {
		fmt.Fprintln(stderr, "author inspect: --goal is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get current directory: %v\n", err)
		return 1
	}
	result, err := author.Inspect(root, author.InspectRequest{
		Project: *project,
		Goal: *goal,
		Mode: *mode,
		FutureTask: *futureTask,
		Phase: *phase,
		Query: *queryText,
		Files: files,
		Domains: domains,
		Tags: tags,
		CandidatePath: *candidatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "author inspect: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runAuthorVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("author verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	proposal := flags.String("proposal", "", "proposal JSON path")
	path := flags.String("path", "", "candidate item or package path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "author verify: --json is required")
		return 2
	}
	if strings.TrimSpace(*proposal) == "" {
		fmt.Fprintln(stderr, "author verify: --proposal is required")
		return 2
	}
	if strings.TrimSpace(*path) == "" {
		fmt.Fprintln(stderr, "author verify: --path is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get current directory: %v\n", err)
		return 1
	}
	result, err := author.Verify(root, author.VerifyRequest{ProposalPath: *proposal, CandidatePath: *path})
	if err != nil {
		fmt.Fprintf(stderr, "author verify: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}
```

Update `printUsage` command list:

```go
	fmt.Fprintln(w, "  author")
```

Add examples:

```go
	fmt.Fprintln(w, "  argos author inspect --json --project mall-api --goal \"create product-list cache knowledge\"")
	fmt.Fprintln(w, "  argos author verify --json --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json --path knowledge/.inbox/packages/backend/product-list-cache")
```

- [ ] **Step 4: Run CLI author tests**

Run:

```bash
go test ./internal/cli -run 'TestRunAuthor' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit CLI commands**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: expose authoring inspect and verify CLI"
```

---

### Task 5: Workspace, README, And Capture Skill Reframe

**Files:**
- Modify: `internal/workspace/workspace.go`
- Modify: `internal/workspace/workspace_test.go`
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`

- [ ] **Step 1: Write failing workspace init test**

Add to `internal/workspace/workspace_test.go`:

```go
func TestInitCreatesAuthoringInboxDirectories(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	for _, rel := range []string{
		"knowledge/.inbox/items",
		"knowledge/.inbox/packages",
		"knowledge/.inbox/proposals",
	} {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", rel)
		}
	}
}
```

- [ ] **Step 2: Run workspace test and verify it fails**

Run:

```bash
go test ./internal/workspace -run TestInitCreatesAuthoringInboxDirectories -count=1
```

Expected: FAIL because `knowledge/.inbox/items`, `knowledge/.inbox/packages`, and `knowledge/.inbox/proposals` are not all created.

- [ ] **Step 3: Update workspace init directories**

In `internal/workspace/workspace.go`, replace the `dirs` value with:

```go
	dirs := []string{
		"knowledge/.inbox/items",
		"knowledge/.inbox/packages",
		"knowledge/.inbox/proposals",
		"knowledge/items",
		"knowledge/packages",
		"argos/generated",
	}
```

- [ ] **Step 4: Run workspace test**

Run:

```bash
go test ./internal/workspace -run TestInitCreatesAuthoringInboxDirectories -count=1
```

Expected: PASS.

- [ ] **Step 5: Update README authoring section**

In `README.md`, add this section after the existing discovery/query usage section:

```markdown
## Agent Knowledge Authoring

Argos write-side UX is agent-operated. Humans describe the engineering knowledge
they want future agents to have, then review a proposal and verification packet.
Agents run the commands.

The durable flow is:

```bash
argos author inspect --json \
  --project mall-api \
  --goal "create product-list cache engineering knowledge" \
  --mode synthesized \
  --future-task "implement product list cache" \
  --phase implementation \
  --files internal/catalog/products.go
```

The agent turns that inspection output into a `Knowledge Design Proposal` under
`knowledge/.inbox/proposals/`. After human approval, the agent writes candidate
knowledge under `knowledge/.inbox/items/` or `knowledge/.inbox/packages/` and
runs:

```bash
argos author verify --json \
  --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json \
  --path knowledge/.inbox/packages/backend/product-list-cache
```

Verification checks the proposal contract, candidate validation, policy gates,
and whether the candidate is discoverable through the normal query path. It does
not promote candidates or mutate the official index. Promotion remains explicit:

```bash
argos promote --path knowledge/.inbox/packages/backend/product-list-cache
argos index
```
```

- [ ] **Step 6: Reframe capture skill overview and workflow headings**

In `skills/capture-knowledge/SKILL.md`, keep the file path and frontmatter name for compatibility, but replace the title and overview with:

```markdown
# Agent Knowledge Authoring

## Overview

Use this skill to turn natural user intent into deliberate Argos engineering
knowledge for future agents. This is not a memo workflow. The user should not
need to know Argos paths or commands. Your job is to help design the knowledge,
inspect existing Argos facts, produce a Knowledge Design Proposal, ask for the
user's decisions, write inbox candidates only after approval, verify the
candidate, and present a review packet.

Argos is an agent-operated knowledge layer. Humans express knowledge intent and
review knowledge design; the agent operates Argos in the background.
```

Rename workflow headings in the same file:

```markdown
### 1. Confirm Authoring Intent
### 2. Gather Context
### 3. Inspect Existing Knowledge
### 4. Choose Item Or Package Shape
### 5. Refine Precision Knowledge
### 6. Present The Knowledge Design Proposal
### 7. Require A Delivery Path
```

Add this command guidance after the existing hard gates:

```markdown
Before presenting the proposal, run or emulate:

```bash
argos author inspect --json --project mall-api --goal "create product-list cache engineering knowledge"
```

After approved candidate files are written, run:

```bash
argos author verify --json \
  --proposal knowledge/.inbox/proposals/product-list-cache/proposal.json \
  --path knowledge/.inbox/packages/backend/product-list-cache
```

Do not use `author inspect` output as permission to write. Do not use `author
verify` output as permission to promote. Both still require human review.
```

- [ ] **Step 7: Run focused docs-adjacent checks**

Run:

```bash
go test ./internal/workspace -count=1
rg -n "Capture Knowledge|capture workflow|capture intent" skills/capture-knowledge/SKILL.md README.md
```

Expected: workspace tests PASS. The `rg` command may still show compatibility wording in the skill description, but it must not show old section headings that present the primary workflow as memo-style capture.

- [ ] **Step 8: Commit workspace and docs changes**

Run:

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go README.md skills/capture-knowledge/SKILL.md
git commit -m "docs: reframe knowledge capture as authoring"
```

---

### Task 6: Authoring Golden Harness Seed

**Files:**
- Create: `testdata/authoring-golden/cases.json`
- Create: `internal/authoringtest/cases.go`
- Create: `internal/authoringtest/cases_test.go`

- [ ] **Step 1: Add the authoring golden cases fixture**

Create `testdata/authoring-golden/cases.json`:

```json
{
  "cases": [
    {
      "id": "purpose_built_rule_user_specified",
      "kind": "user_specified",
      "input": {
        "project": "mall-api",
        "goal": "Create a backend rule for how future agents should design product-list cache invalidation.",
        "mode": "user_specified"
      },
      "required_guards": [
        "proposal_before_write",
        "source_and_trust",
        "future_retrieval_contract",
        "no_unauthorized_priority_must"
      ],
      "expected": {
        "result": "pass"
      }
    },
    {
      "id": "observed_lesson_from_repo",
      "kind": "observed",
      "input": {
        "project": "mall-api",
        "goal": "Turn the observed retry handling pattern into reusable agent knowledge.",
        "mode": "observed"
      },
      "required_guards": [
        "evidence_paths_recorded",
        "assumptions_separated",
        "candidate_inbox_only"
      ],
      "expected": {
        "result": "pass"
      }
    },
    {
      "id": "synthesized_package_requires_assumptions",
      "kind": "synthesized",
      "input": {
        "project": "mall-api",
        "goal": "Create a package that helps future agents design product-list cache behavior.",
        "mode": "synthesized"
      },
      "required_guards": [
        "assumptions_explicit",
        "status_draft",
        "priority_should_or_may"
      ],
      "expected": {
        "result": "pass"
      }
    },
    {
      "id": "overlap_requires_user_choice",
      "kind": "overlap",
      "input": {
        "project": "mall-api",
        "goal": "Create another cache TTL rule that may overlap existing cache knowledge.",
        "mode": "user_specified"
      },
      "required_guards": [
        "overlap_checked",
        "overlap_decision_present",
        "no_write_before_decision"
      ],
      "expected": {
        "result": "review-needed"
      }
    },
    {
      "id": "candidate_validates_but_not_findable",
      "kind": "findability",
      "input": {
        "project": "mall-api",
        "goal": "Create knowledge whose metadata is too weak to be found for the intended task.",
        "mode": "synthesized"
      },
      "required_guards": [
        "author_verify_run",
        "findability_reported",
        "review_needed_when_not_findable"
      ],
      "expected": {
        "result": "review-needed"
      }
    },
    {
      "id": "priority_must_requires_authorization",
      "kind": "policy",
      "input": {
        "project": "mall-api",
        "goal": "Create a must-follow engineering rule for future agents.",
        "mode": "user_specified"
      },
      "required_guards": [
        "priority_must_authorization_checked",
        "human_authorization_required"
      ],
      "expected": {
        "result": "review-needed"
      }
    }
  ]
}
```

- [ ] **Step 2: Write failing harness tests**

Create `internal/authoringtest/cases_test.go`:

```go
package authoringtest

import (
	"strings"
	"testing"
)

func TestLoadCasesIncludesRequiredAuthoringKinds(t *testing.T) {
	cases, err := LoadCases("../../testdata/authoring-golden/cases.json")
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"user_specified": false,
		"observed": false,
		"synthesized": false,
		"overlap": false,
		"findability": false,
		"policy": false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing authoring case kind %s in %#v", kind, cases)
		}
	}
}

func TestSummariesHideExpectedOracles(t *testing.T) {
	cases, err := LoadCases("../../testdata/authoring-golden/cases.json")
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	text := stringifySummaries(summaries)
	for _, forbidden := range []string{
		"expected",
		"result",
		"purpose_built_rule_user_specified",
		"candidate_validates_but_not_findable",
		"priority_must_requires_authorization",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "case-001") || !strings.Contains(text, "Create a backend rule") {
		t.Fatalf("summary should expose public handles and inputs: %s", text)
	}
}
```

- [ ] **Step 3: Run harness tests and verify they fail**

Run:

```bash
go test ./internal/authoringtest -count=1
```

Expected: FAIL because `internal/authoringtest` does not exist.

- [ ] **Step 4: Implement authoring case loader and public summaries**

Create `internal/authoringtest/cases.go`:

```go
package authoringtest

import (
	"encoding/json"
	"fmt"
	"os"
)

type CaseFile struct {
	Cases []Case `json:"cases"`
}

type Case struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Input          Input    `json:"input"`
	RequiredGuards []string `json:"required_guards"`
	Expected       Expected `json:"expected"`
}

type Input struct {
	Project string `json:"project"`
	Goal    string `json:"goal"`
	Mode    string `json:"mode"`
}

type Expected struct {
	Result string `json:"result"`
}

type Summary struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Input Input  `json:"input"`
}

func LoadCases(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read authoring cases %q: %w", path, err)
	}
	var file CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse authoring cases %q: %w", path, err)
	}
	return file.Cases, nil
}

func Summaries(cases []Case) []Summary {
	summaries := make([]Summary, 0, len(cases))
	for i, tc := range cases {
		summaries = append(summaries, Summary{
			ID: fmt.Sprintf("case-%03d", i+1),
			Kind: tc.Kind,
			Input: tc.Input,
		})
	}
	return summaries
}

func stringifySummaries(summaries []Summary) string {
	data, err := json.Marshal(summaries)
	if err != nil {
		return ""
	}
	return string(data)
}
```

- [ ] **Step 5: Run authoring harness tests**

Run:

```bash
go test ./internal/authoringtest -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit harness seed**

Run:

```bash
git add testdata/authoring-golden/cases.json internal/authoringtest/cases.go internal/authoringtest/cases_test.go
git commit -m "test: seed authoring golden harness"
```

---

### Task 7: Full Verification And Integration Sweep

**Files:**
- All files changed by Tasks 1-6.

- [ ] **Step 1: Run focused package tests**

Run:

```bash
go test ./internal/author ./internal/cli ./internal/workspace ./internal/authoringtest -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full repository tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run CLI smoke checks in a temporary workspace**

Run:

```bash
tmpdir=$(mktemp -d)
go build -o "$tmpdir/argos" ./cmd/argos
(cd "$tmpdir" && "$tmpdir/argos" init)
```

Expected stdout:

```text
initialized Argos workspace
```

Then run:

```bash
(cd "$tmpdir" && "$tmpdir/argos" author inspect --json --project mall-api --goal "create product-list cache engineering knowledge")
```

Expected: exit code 0 with JSON containing `"policy"` and `"proposal_requirements"`. The project may be unknown in a blank initialized workspace, and that is acceptable because registry project configuration is user-owned.

- [ ] **Step 4: Check docs and skill for stale primary vocabulary**

Run:

```bash
rg -n "memo|note-taking|Capture Knowledge|capture workflow|capture intent" README.md skills/capture-knowledge/SKILL.md docs/superpowers/specs/2026-05-02-argos-agent-knowledge-authoring-protocol-design.md
```

Expected: no stale primary workflow headings. Mentions that explicitly contrast authoring with memo/note-taking are acceptable.

- [ ] **Step 5: Check git diff hygiene**

Run:

```bash
git diff --check
git status --short --branch
```

Expected: `git diff --check` exits 0. `git status --short --branch` shows only intentional branch-ahead state and no unstaged changes after the final commit.

- [ ] **Step 6: Commit final integration fixes if needed**

If any command above required small integration fixes, commit them with:

```bash
git add internal README.md skills testdata docs
git commit -m "test: verify authoring protocol integration"
```

If no fixes were needed, do not create an empty commit.

---

## Spec Coverage Checklist

- Agent knowledge authoring, not memo capture: Tasks 5 and 6 update docs/skill and harness vocabulary.
- Workflow-neutral design: Tasks 1-4 validate artifacts and commands without naming host workflow tools.
- Purpose-built engineering knowledge: Task 6 includes `user_specified` and `synthesized` cases.
- AI-facing commands: Task 4 adds `author inspect` and `author verify` with mandatory JSON.
- Proposal artifact contract: Task 1 defines and validates `authoring.proposal.v1`.
- Deterministic inspection: Task 2 implements registry, overlap, policy, and path-risk facts.
- Deterministic verification: Task 3 implements schema, candidate, policy, and findability checks.
- Human approval before durable trust: Tasks 1, 3, and 5 keep approval and promotion explicit.
- Future retrieval simulation: Task 3 builds a temporary index with official knowledge plus candidate.
- No workspace index mutation during verify: Task 3 tests that `argos/index.db` is not created.
- Authoring harness seed: Task 6 adds hidden-oracle cases and public summaries.
- MCP deferred: No task adds MCP authoring tools in this slice.
- Candidate scaffold deferred: No task adds `author scaffold` in this slice.

---

## Execution Notes

- Keep commits task-sized. The plan intentionally commits after each component so failures are easy to bisect.
- Do not promote or move any real knowledge candidates while executing this plan.
- Do not add `author propose`, `author write`, or MCP mutation tools during this implementation.
- Treat `review-needed` as a valid verification result, not as a process crash.
