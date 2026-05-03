# Argos Write Module Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the old write-side authoring surface with a first-release knowledge write module using `design -> draft -> check -> publish` vocabulary across code, CLI, MCP, adapters, dogfood, and tests.

**Architecture:** Build a new `internal/knowledgewrite` package that owns knowledge design, write guidance, and draft checking. Public entrypoints move to `argos knowledge design/check/publish` and MCP tools `argos_design_knowledge` / `argos_check_knowledge`. Dogfood is renamed to the write vocabulary and evaluates whether runners followed generated write guidance, not just whether final artifacts look valid.

**Tech Stack:** Go standard library, existing `internal/knowledge`, `internal/query`, `internal/index`, `internal/registry`, `internal/cli`, `internal/mcp`, Markdown dogfood assets, `go test`.

---

## Scope Notes

This plan implements the module-level closure from:

`docs/superpowers/specs/2026-05-03-argos-authoring-module-productization-design.md`

The spec covers several surfaces, so the plan is split into five working tasks.
Each task has its own tests and commit point. Historical specs, plans, and
reports may mention removed vocabulary as history. Live write-side code,
current docs, templates, adapters, CLI help, MCP schemas, dogfood assets, and
tests must move to the new vocabulary.

## File Map

Create:

- `internal/knowledgewrite/model.go` - public write-side data model: `KnowledgeDesign`, `WriteGuidance`, `DesignRequest`, `DesignResponse`, `CheckRequest`, `CheckResponse`.
- `internal/knowledgewrite/design.go` - design/inspect logic, adapted from old inspect behavior with new names.
- `internal/knowledgewrite/guidance.go` - write guidance builder.
- `internal/knowledgewrite/check.go` - draft checking logic, adapted from old verify behavior with new names.
- `internal/knowledgewrite/design_test.go` - design response and design-only tests.
- `internal/knowledgewrite/check_test.go` - schema, boundary, and findability checks.
- `internal/knowledgewrite/vocabulary_test.go` - live write-side removed-term scan.
- `internal/writedogfood/` - renamed dogfood harness package.
- `docs/superpowers/templates/argos-write-dogfood-report.md` - new write dogfood report template.
- `docs/superpowers/checklists/2026-05-03-argos-write-dogfood-checklist.md` - new write dogfood checklist.
- `docs/superpowers/reports/2026-05-03-argos-write-dogfood-round-0.md` - write dogfood round record.
- `testdata/write-golden/` - renamed write-side dogfood cases and fixtures.

Modify:

- `internal/cli/cli.go` - remove public `author` command, add `knowledge design/check/publish`, update dogfood command from `authoring` to `write`.
- `internal/cli/cli_test.go` - update CLI tests to new commands and removed old help text.
- `internal/mcp/server.go` - add write tools and server root support.
- `internal/mcp/server_test.go` - strict schemas and tool call tests.
- `internal/adapters/adapters.go` - add write protocol to generated adapter text.
- `internal/adapters/adapters_test.go` - assert write protocol and removed direct write from query results.
- `README.md` - update write module docs to design/draft/check/publish.

Delete after replacement:

- `internal/author/`
- `internal/authoringdogfood/`
- `docs/superpowers/templates/argos-authoring-dogfood-report.md`
- `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
- `testdata/authoring-golden/`

---

### Task 1: Core Knowledge Write Model, Design, And Check

**Files:**
- Create: `internal/knowledgewrite/model.go`
- Create: `internal/knowledgewrite/design.go`
- Create: `internal/knowledgewrite/guidance.go`
- Create: `internal/knowledgewrite/check.go`
- Create: `internal/knowledgewrite/design_test.go`
- Create: `internal/knowledgewrite/check_test.go`
- Create: `internal/knowledgewrite/vocabulary_test.go`
- Delete after migration: `internal/author/`

- [ ] **Step 1: Write failing tests for design response vocabulary**

Add `internal/knowledgewrite/design_test.go`:

```go
package knowledgewrite

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/workspace"
)

func TestDesignKnowledgeReturnsWriteGuidanceAndTemplate(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Create Redis cache best practices for future backend agents.",
		Tags:    []string{"cache", "redis"},
		Domains: []string{"backend", "redis"},
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}

	if result.Intent == "" {
		t.Fatalf("intent missing from result: %#v", result)
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("state = %q, want ready_to_design: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if result.WriteGuidance.NextAction != "write_knowledge_design" {
		t.Fatalf("next_action = %q", result.WriteGuidance.NextAction)
	}
	if result.WriteGuidance.DraftAllowed {
		t.Fatalf("design response must not approve draft writing: %#v", result.WriteGuidance)
	}
	if result.KnowledgeDesignTemplate.SchemaVersion != KnowledgeDesignSchemaVersion {
		t.Fatalf("schema_version = %q", result.KnowledgeDesignTemplate.SchemaVersion)
	}
	if !strings.HasPrefix(result.WriteGuidance.DesignPath, "knowledge/.inbox/designs/mall-api/") {
		t.Fatalf("design path should be in inbox designs, got %q", result.WriteGuidance.DesignPath)
	}
	if !strings.HasPrefix(result.WriteGuidance.DraftPath, "knowledge/.inbox/packages/mall-api/") {
		t.Fatalf("draft path should be in inbox packages, got %q", result.WriteGuidance.DraftPath)
	}
}

func TestDesignKnowledgeJSONUsesNewVocabulary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{Project: "mall-api", Intent: "Document product list cache TTL."})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal design response: %v", err)
	}
	text := string(body)
	required := []string{
		`"write_guidance"`,
		`"knowledge_design_template"`,
		`"sources"`,
		`"draft_output"`,
		`"existing_knowledge"`,
		`"write_boundary"`,
		`"draft_files"`,
		`"check_plan"`,
		`"review"`,
	}
	for _, want := range required {
		if !strings.Contains(text, want) {
			t.Fatalf("design JSON missing %s: %s", want, text)
		}
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("design JSON contains removed term %q: %s", forbidden, text)
		}
	}
}

func TestDesignKnowledgeMissingContentIsDesignOnly(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{
		Project: "mall-api",
		Intent:  "Remember our project convention.",
	})
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	if result.WriteGuidance.State != "design_only" {
		t.Fatalf("state = %q, want design_only: %#v", result.WriteGuidance.State, result.WriteGuidance)
	}
	if !result.WriteGuidance.DesignOnly {
		t.Fatalf("design_only flag should be true: %#v", result.WriteGuidance)
	}
	if result.WriteGuidance.DraftPath != "" {
		t.Fatalf("design-only response should not include draft path: %#v", result.WriteGuidance)
	}
	if len(result.KnowledgeDesignTemplate.Review.UnresolvedBlockers) == 0 {
		t.Fatalf("design-only template should include unresolved blocker: %#v", result.KnowledgeDesignTemplate.Review)
	}
}

func TestDesignKnowledgeRejectsUnsafeDraftPath(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	result, err := Design(root, DesignRequest{
		Project:   "mall-api",
		Intent:    "Create Redis guidance.",
		DraftPath: filepath.Join("..", "outside"),
	})
	if err != nil {
		t.Fatalf("unsafe draft path should produce guidance, not system error: %v", err)
	}
	if result.WriteGuidance.State != "design_only" {
		t.Fatalf("unsafe draft path should block draft writing: %#v", result.WriteGuidance)
	}
	if !containsText(result.WriteGuidance.StopConditions, "Do not write draft knowledge") {
		t.Fatalf("missing draft stop condition: %#v", result.WriteGuidance.StopConditions)
	}
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
```

Add `internal/knowledgewrite/vocabulary_test.go` with the shared removed-term helper used by Task 1 tests:

```go
package knowledgewrite

func removedAuthoringTermsForTests() []string {
	return []string{
		"authoring_packet",
		"proposal_scaffold",
		"source_profile",
		"proposed_shape",
		"overlap_decision",
		"verification_plan",
		"human_review",
		"artifact_state",
		"author inspect",
		"author verify",
		"authoring.proposal.v2",
	}
}
```

- [ ] **Step 2: Write failing tests for check response vocabulary and draft boundary**

Add `internal/knowledgewrite/check_test.go`:

```go
package knowledgewrite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/workspace"
)

func TestCheckKnowledgeReturnsNewVocabulary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	designPath := writeDesignFile(t, root, validKnowledgeDesign())
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackage(t, root, draftPath)

	result, err := Check(root, CheckRequest{DesignPath: designPath, DraftPath: draftPath})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Result != "pass" && result.Result != "review-needed" {
		t.Fatalf("unexpected check result: %#v", result)
	}
	if result.Design.Path != designPath {
		t.Fatalf("design path mismatch: %#v", result.Design)
	}
	if result.Draft.Path != draftPath {
		t.Fatalf("draft path mismatch: %#v", result.Draft)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal check result: %v", err)
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("check JSON contains removed term %q: %s", forbidden, string(body))
		}
	}
}

func TestCheckKnowledgeRejectsDraftOutsideWriteBoundary(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	design := validKnowledgeDesign()
	design.DraftOutput.Path = "knowledge/.inbox/packages/mall-api/redis-cache"
	design.CheckPlan.ValidatePath = "knowledge/.inbox/packages/mall-api/redis-cache"
	designPath := writeDesignFile(t, root, design)
	draftPath := "knowledge/packages/mall-api/redis-cache"
	writeDraftPackage(t, root, draftPath)

	result, err := Check(root, CheckRequest{DesignPath: designPath, DraftPath: draftPath})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected fail for official draft outside inbox boundary, got %#v", result)
	}
	if !hasFinding(result.Findings, "draft path is outside approved write boundary") {
		t.Fatalf("missing boundary finding: %#v", result.Findings)
	}
}

func validKnowledgeDesign() KnowledgeDesign {
	return KnowledgeDesign{
		SchemaVersion: KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices for future backend agents.",
		KnowledgeGoal: "Design reusable knowledge for future agents: Create Redis cache best practices.",
		Project:       "mall-api",
		Audience: Audience{
			Primary: "implementer_agent",
			AgentActionsSupported: []string{
				"apply Redis cache guidance during implementation",
			},
		},
		Scope: Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			FileGlobs:    []string{"**/*"},
		},
		Sources: Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			AISuggested: []string{
				"Use reviewed Redis recommendations as draft guidance.",
			},
			Assumptions: []string{"Team will review before publishing."},
			Claims: []SourceClaim{{
				Claim:          "Redis cache best practices are being designed for future agents.",
				Kind:           "recommendation",
				Trust:          "ai_suggested",
				Source:         []string{"argos knowledge design"},
				RequiresReview: true,
			}},
		},
		DraftOutput: DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             "package:mall-api.redis-cache.v1",
			Path:           "knowledge/.inbox/packages/mall-api/redis-cache",
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package gives future agents one entrypoint for Redis cache guidance.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: FutureUse{
			TriggerRequests: []string{"implement Redis cache behavior"},
			QueryPhrases:    []string{"redis cache best practices"},
			ExpectedUse:     "Future agents should read this before implementing Redis cache behavior.",
			CitationPolicy:  "cite_after_use",
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When implementing Redis-backed caching."},
			WhenNotToUse: []string{"When changing unrelated storage behavior."},
		},
		ExistingKnowledge: ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing knowledge covers this exact guidance.",
		},
		WriteBoundary: WriteBoundary{
			Path:                       "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:       true,
		},
		DraftFiles: []DraftFile{{
			Path:    "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
			Purpose: "entrypoint",
			Load:    "read_before_implementation",
		}},
		CheckPlan: CheckPlan{
			ValidatePath: "knowledge/.inbox/packages/mall-api/redis-cache",
			FindabilityChecks: []FindabilityCheckScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: Review{
			Questions:          []string{"Is draft writing approved?"},
			DesignApproved:     true,
			DraftWriteApproved: true,
		},
	}
}

func writeDesignFile(t *testing.T, root string, design KnowledgeDesign) string {
	t.Helper()
	rel := "knowledge/.inbox/designs/mall-api/redis-cache/design.json"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir design parent: %v", err)
	}
	body, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	if err := os.WriteFile(abs, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write design: %v", err)
	}
	return rel
}

func writeDraftPackage(t *testing.T, root string, rel string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir draft: %v", err)
	}
	body := `---
id: package:mall-api.redis-cache.v1
type: package
title: Redis Cache Best Practices
status: draft
priority: should
projects: [mall-api]
tech_domains: [redis]
tags: [cache]
applies_to:
  files: ["**/*"]
---

# Redis Cache Best Practices

Use reviewed Redis cache guidance for future backend implementation.
`
	if err := os.WriteFile(filepath.Join(dir, "KNOWLEDGE.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write draft package: %v", err)
	}
}

func hasFinding(findings []Finding, text string) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Message, text) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Run core tests to verify they fail**

Run:

```bash
go test ./internal/knowledgewrite -run 'TestDesignKnowledge|TestCheckKnowledge' -count=1
```

Expected: FAIL because package `internal/knowledgewrite` and the tested symbols do not exist.

- [ ] **Step 4: Implement `internal/knowledgewrite/model.go`**

Create `internal/knowledgewrite/model.go` with these public types and JSON names:

```go
package knowledgewrite

const KnowledgeDesignSchemaVersion = "knowledge.design.v1"

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type DesignRequest struct {
	Project    string   `json:"project"`
	Intent     string   `json:"intent"`
	FutureTask string   `json:"future_task,omitempty"`
	Phase      string   `json:"phase,omitempty"`
	Query      string   `json:"query,omitempty"`
	Files      []string `json:"files,omitempty"`
	Domains    []string `json:"domains,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	DraftPath  string   `json:"draft_path,omitempty"`
}

type DesignResponse struct {
	Project                 string          `json:"project"`
	Intent                  string          `json:"intent"`
	Capabilities            Capabilities    `json:"capabilities"`
	Registry                RegistryFacts   `json:"registry"`
	ExistingKnowledge       ExistingMatches `json:"existing_knowledge"`
	DraftPathRisk           DraftPathRisk   `json:"draft_path_risk"`
	WritePolicy             WritePolicy     `json:"write_policy"`
	WriteGuidance           WriteGuidance   `json:"write_guidance"`
	KnowledgeDesignTemplate KnowledgeDesign `json:"knowledge_design_template"`
	NextSteps               []NextStep      `json:"next_steps"`
}

type Capabilities struct {
	Filesystem string `json:"filesystem"`
	Index      string `json:"index"`
}

type RegistryFacts struct {
	ProjectKnown    bool     `json:"project_known"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
}

type ExistingMatches struct {
	Official []ExistingMatch `json:"official"`
	Inbox    []ExistingMatch `json:"inbox"`
	Index    []ExistingMatch `json:"index"`
}

type ExistingMatch struct {
	Kind    string   `json:"kind"`
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Reasons []string `json:"reasons"`
}

type DraftPathRisk struct {
	DraftPath string `json:"draft_path"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

type WritePolicy struct {
	Write        string `json:"write"`
	OfficialWrite string `json:"official_write"`
	Publish      string `json:"publish"`
	PriorityMust string `json:"priority_must"`
	AISuggestions string `json:"ai_suggestions"`
}

type NextStep struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

type WriteGuidance struct {
	State           string            `json:"state"`
	NextAction      string            `json:"next_action"`
	Reason          string            `json:"reason"`
	DesignPath      string            `json:"design_path"`
	DraftPath       string            `json:"draft_path,omitempty"`
	DraftAllowed    bool              `json:"draft_allowed"`
	DesignOnly      bool              `json:"design_only"`
	StopConditions  []string          `json:"stop_conditions"`
	DesignFocus     []string          `json:"design_focus"`
	SourceWork      SourceWorkGuidance `json:"source_work"`
	ReviewQuestions []string          `json:"review_questions"`
	Commands        WriteCommands     `json:"commands"`
}

type SourceWorkGuidance struct {
	UserInput     []string `json:"user_input,omitempty"`
	Observed      []string `json:"observed,omitempty"`
	Imported      []string `json:"imported,omitempty"`
	AISuggested   []string `json:"ai_suggested,omitempty"`
	Templates     []string `json:"templates,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	Assumptions   []string `json:"assumptions,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
}

type WriteCommands struct {
	WriteDesign string `json:"write_design"`
	CheckDraft  string `json:"check_draft,omitempty"`
}

type KnowledgeDesign struct {
	SchemaVersion     string                    `json:"schema_version"`
	UserRequest       string                    `json:"user_request"`
	KnowledgeGoal     string                    `json:"knowledge_goal"`
	Project           string                    `json:"project"`
	Audience          Audience                  `json:"audience"`
	Scope             Scope                     `json:"scope"`
	Sources           Sources                   `json:"sources"`
	DraftOutput       DraftOutput               `json:"draft_output"`
	FutureUse         FutureUse                 `json:"future_use"`
	Applicability     Applicability             `json:"applicability"`
	ExistingKnowledge ExistingKnowledgeDecision `json:"existing_knowledge"`
	WriteBoundary     WriteBoundary             `json:"write_boundary"`
	DraftFiles        []DraftFile               `json:"draft_files"`
	CheckPlan         CheckPlan                 `json:"check_plan"`
	Review            Review                    `json:"review"`
}

type Audience struct {
	Primary               string   `json:"primary"`
	Secondary             []string `json:"secondary,omitempty"`
	Reviewers             []string `json:"reviewers,omitempty"`
	AgentActionsSupported []string `json:"agent_actions_supported"`
}

type Scope struct {
	Projects       []string `json:"projects"`
	Stability      string   `json:"stability"`
	Distribution   string   `json:"distribution"`
	SubjectDomains []string `json:"subject_domains"`
	TechDomains    []string `json:"tech_domains"`
	Files          []string `json:"files"`
	FileGlobs      []string `json:"file_globs"`
	OutOfScope     []string `json:"out_of_scope"`
}

type Sources struct {
	UserInput     []string      `json:"user_input"`
	Observed      []string      `json:"observed"`
	Imported      []string      `json:"imported"`
	AISuggested   []string      `json:"ai_suggested"`
	Templates     []string      `json:"templates"`
	Examples      []string      `json:"examples"`
	Assumptions   []string      `json:"assumptions"`
	OpenQuestions []string      `json:"open_questions"`
	Claims        []SourceClaim `json:"claims"`
}

type SourceClaim struct {
	Claim          string   `json:"claim"`
	Kind           string   `json:"kind"`
	Source         []string `json:"source"`
	Trust          string   `json:"trust"`
	RequiresReview bool     `json:"requires_review"`
}

type DraftOutput struct {
	Kind           string `json:"kind"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	ID             string `json:"id"`
	Path           string `json:"path"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	Rationale      string `json:"rationale"`
	EntrypointLoad string `json:"entrypoint_load"`
	DraftState     string `json:"draft_state"`
}

type FutureUse struct {
	TriggerRequests  []string `json:"trigger_requests"`
	NegativeTriggers []string `json:"negative_triggers"`
	Phases           []string `json:"phases"`
	QueryPhrases     []string `json:"query_phrases"`
	ExpectedUse      string   `json:"expected_use"`
	CitationPolicy   string   `json:"citation_policy"`
	MissingNeeds     []string `json:"missing_needs"`
}

type Applicability struct {
	WhenToUse    []string `json:"when_to_use"`
	WhenNotToUse []string `json:"when_not_to_use"`
	Tradeoffs    []string `json:"tradeoffs"`
}

type ExistingKnowledgeDecision struct {
	OfficialMatches     []string `json:"official_matches"`
	InboxMatches        []string `json:"inbox_matches"`
	PossibleMatches     []string `json:"possible_matches"`
	Decision            string   `json:"decision"`
	Reason              string   `json:"reason"`
	ReviewChoiceRequired bool    `json:"review_choice_required"`
}

type WriteBoundary struct {
	Path                        string `json:"path"`
	WriteRequiresReviewApproval bool   `json:"write_requires_review_approval"`
	PriorityMustApproved        bool   `json:"priority_must_approved"`
	PublishApproved             bool   `json:"publish_approved"`
	OfficialWriteApproved       bool   `json:"official_write_approved"`
	ReviewPacketRequired        bool   `json:"review_packet_required"`
}

type DraftFile struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Load    string `json:"load"`
}

type CheckPlan struct {
	ValidatePath      string                     `json:"validate_path"`
	FindabilityChecks []FindabilityCheckScenario `json:"findability_checks"`
}

type FindabilityCheckScenario struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Query   string   `json:"query"`
	Files   []string `json:"files"`
}

type Review struct {
	Questions             []string `json:"questions"`
	DesignApproved        bool     `json:"design_approved"`
	DraftWriteApproved    bool     `json:"draft_write_approved"`
	PriorityMustApproved  bool     `json:"priority_must_approved"`
	OfficialWriteApproved bool     `json:"official_write_approved"`
	PublishApproved       bool     `json:"publish_approved"`
	RequestedEdits        []string `json:"requested_edits,omitempty"`
	UnresolvedBlockers    []string `json:"unresolved_blockers,omitempty"`
}

type CheckRequest struct {
	DesignPath string `json:"design"`
	DraftPath  string `json:"draft"`
}

type CheckResponse struct {
	Result      string            `json:"result"`
	Design      DesignStatus      `json:"design"`
	Draft       DraftStatus       `json:"draft"`
	Policy      PolicyStatus      `json:"policy"`
	Findability FindabilityStatus `json:"findability"`
	Findings    []Finding         `json:"findings,omitempty"`
}

type DesignStatus struct {
	Path       string `json:"path"`
	Validation string `json:"validation"`
}

type DraftStatus struct {
	Path       string   `json:"path"`
	Validation string   `json:"validation"`
	IDs        []string `json:"ids,omitempty"`
}

type PolicyStatus struct {
	Result string `json:"result"`
}

type FindabilityStatus struct {
	Result string             `json:"result"`
	Checks []FindabilityCheck `json:"checks,omitempty"`
}

type FindabilityCheck struct {
	Scenario      FindabilityCheckScenario `json:"scenario"`
	Result        string                   `json:"result"`
	DraftID       string                   `json:"draft_id,omitempty"`
	MatchedDraft  bool                     `json:"matched_draft"`
	MatchedIDs    []string                 `json:"matched_ids,omitempty"`
	Support       string                   `json:"support"`
}

func LoadDesign(path string) (KnowledgeDesign, error)
func ValidateDesign(design KnowledgeDesign) []Finding
```

- [ ] **Step 5: Implement design and guidance behavior**

Create `internal/knowledgewrite/design.go` and `internal/knowledgewrite/guidance.go` by porting existing inspect behavior with new names. The exported function must be:

```go
func Design(root string, req DesignRequest) (DesignResponse, error)
```

Required behavior:

- Load registry, official knowledge, inbox knowledge, and optional index overlap.
- Use `Intent` instead of `Goal`.
- Use `DraftPath` instead of the removed path field.
- Build `KnowledgeDesignTemplate` with `schema_version: knowledge.design.v1`.
- Use `sources.user_input`, `sources.observed`, and `sources.ai_suggested`.
- Use `draft_output.path` and `check_plan.validate_path`.
- Use `write_boundary.path`.
- Set `write_guidance.draft_allowed=false` for every design response.
- For overlap, unsafe draft path, or missing substantive content, set `write_guidance.state=design_only`, `write_guidance.design_only=true`, no draft path, empty `draft_files`, and `review.draft_write_approved=false`.

Use this helper signature set so later tasks can call the package consistently:

```go
func designPath(project string, intent string) string
func defaultDraftPath(project string, intent string) string
func buildKnowledgeDesignTemplate(response DesignResponse, req DesignRequest) KnowledgeDesign
func buildWriteGuidance(response DesignResponse, req DesignRequest) WriteGuidance
func requestLooksMissingSubstantiveContent(req DesignRequest) bool
func requestLooksConsumerFacing(req DesignRequest) bool
```

- [ ] **Step 6: Implement check behavior**

Create `internal/knowledgewrite/check.go` by porting existing verify behavior with new names. The exported function must be:

```go
func LoadDesign(path string) (KnowledgeDesign, error)
func ValidateDesign(design KnowledgeDesign) []Finding
func Check(root string, req CheckRequest) (CheckResponse, error)
```

Required behavior:

- Load a `KnowledgeDesign` from `req.DesignPath`.
- Validate `KnowledgeDesignSchemaVersion`.
- Validate `sources`, `draft_output`, `existing_knowledge`, `write_boundary`, `draft_files`, `check_plan`, and `review`.
- Load draft knowledge from `req.DraftPath`.
- Validate draft items through `knowledge.ValidateItems`.
- Enforce draft path equals `draft_output.path` and `check_plan.validate_path`.
- Enforce inbox boundary unless `write_boundary.path` allows official review and `review.official_write_approved=true`.
- Enforce `priority: must` only when `review.priority_must_approved=true`.
- Run findability checks through `query.FindKnowledge` with a temporary index containing official knowledge plus draft items.
- Return only new JSON field names: `design`, `draft`, `policy`, `findability`, `findings`.

- [ ] **Step 7: Run package tests**

Run:

```bash
go test ./internal/knowledgewrite -count=1
```

Expected: PASS.

- [ ] **Step 8: Keep the old package only until CLI and dogfood imports move**

Run:

```bash
rg -n '"argos/internal/author"|author\\.' internal/cli internal/authoringdogfood
```

Expected: output still exists in Task 1 because CLI and dogfood have not moved.
Do not delete `internal/author/` in this task. The deletion step is mandatory in
Task 4 after CLI, MCP, and dogfood all import `internal/knowledgewrite`.

- [ ] **Step 9: Commit Task 1**

```bash
git add internal/knowledgewrite internal/author
git commit -m "write: add knowledge design and check core"
```

---

### Task 2: Public CLI Replacement

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `README.md`
- Delete when imports are gone: `internal/author/`

- [ ] **Step 1: Write failing CLI tests**

In `internal/cli/cli_test.go`, add tests near the current write-side CLI tests:

```go
func TestRunKnowledgeDesignReturnsWriteGuidance(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	var stdout, stderr bytes.Buffer
	code := runInDir(t, root, []string{
		"knowledge", "design", "--json",
		"--project", "mall-api",
		"--intent", "Create Redis cache best practices for future backend agents.",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var result struct {
		WriteGuidance struct {
			State        string `json:"state"`
			NextAction   string `json:"next_action"`
			DraftAllowed bool   `json:"draft_allowed"`
		} `json:"write_guidance"`
		KnowledgeDesignTemplate struct {
			SchemaVersion string `json:"schema_version"`
			Sources       any    `json:"sources"`
		} `json:"knowledge_design_template"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, stdout.String())
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("unexpected write guidance: %s", stdout.String())
	}
	if result.WriteGuidance.DraftAllowed {
		t.Fatalf("design must not approve draft writing: %s", stdout.String())
	}
	if result.KnowledgeDesignTemplate.SchemaVersion != "knowledge.design.v1" {
		t.Fatalf("unexpected schema: %s", stdout.String())
	}
	assertNoRemovedWriteTerms(t, stdout.String())
}

func TestRunKnowledgeCheckRequiresDesignAndDraft(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "check", "--json", "--design", "design.json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "knowledge check: --draft is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunAuthorCommandIsRemovedFromPublicCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"author", "inspect", "--json", "--project", "mall-api", "--goal", "x"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command: author") {
		t.Fatalf("expected unknown command, got stderr = %s", stderr.String())
	}
}

func TestUsageUsesWriteVocabulary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	body := stderr.String()
	for _, want := range []string{
		"argos knowledge design --json --project <project> --intent <intent>",
		"argos knowledge check --json --design <design.json> --draft <draft>",
		"argos knowledge publish --path <draft>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("usage missing %q:\n%s", want, body)
		}
	}
	assertNoRemovedWriteTerms(t, body)
}
```

Add test helper:

```go
func assertNoRemovedWriteTerms(t *testing.T, body string) {
	t.Helper()
	for _, term := range []string{
		"authoring_packet",
		"proposal_scaffold",
		"source_profile",
		"proposed_shape",
		"overlap_decision",
		"verification_plan",
		"human_review",
		"artifact_state",
		"author inspect",
		"author verify",
	} {
		if strings.Contains(body, term) {
			t.Fatalf("body contains removed write term %q:\n%s", term, body)
		}
	}
}
```

- [ ] **Step 2: Run CLI tests and verify failure**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgeDesignReturnsWriteGuidance|TestRunKnowledgeCheckRequiresDesignAndDraft|TestRunAuthorCommandIsRemovedFromPublicCLI|TestUsageUsesWriteVocabulary' -count=1
```

Expected: FAIL because CLI commands still use the removed public surface.

- [ ] **Step 3: Implement CLI command changes**

In `internal/cli/cli.go`:

- Remove the top-level `case "author"`.
- Add `knowledge design`, `knowledge check`, and `knowledge publish` in `runKnowledge`.
- Import `argos/internal/knowledgewrite`.
- Rename publish helpers:
  - `promoteCandidate` -> `publishDraft`
  - `promotionTarget` -> `publishTarget`
  - error text `candidate` -> `draft`

Use this command shape:

```go
func runKnowledgeDesign(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge design", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	intent := flags.String("intent", "", "knowledge intent")
	futureTask := flags.String("future-task", "", "future task this knowledge should support")
	phase := flags.String("phase", "", "workflow phase")
	queryText := flags.String("query", "", "search query")
	draftPath := flags.String("draft-path", "", "draft knowledge path")
	var files multiValueFlag
	var domains multiValueFlag
	var tags multiValueFlag
	flags.Var(&files, "files", "file path relevant to the knowledge intent; may be repeated")
	flags.Var(&domains, "domains", "domain relevant to the knowledge intent; may be repeated")
	flags.Var(&tags, "tags", "tag relevant to the knowledge intent; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge design: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "knowledge design: --project is required")
		return 2
	}
	if strings.TrimSpace(*intent) == "" {
		fmt.Fprintln(stderr, "knowledge design: --intent is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge design: get current directory: %v\n", err)
		return 1
	}
	result, err := knowledgewrite.Design(root, knowledgewrite.DesignRequest{
		Project: *project, Intent: *intent, FutureTask: *futureTask,
		Phase: *phase, Query: *queryText, Files: files, Domains: domains,
		Tags: tags, DraftPath: *draftPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge design: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}
```

Implement `runKnowledgeCheck` with flags `--json`, `--design`, and `--draft`, calling `knowledgewrite.Check`. Implement `runKnowledgePublish` with `--path`, calling `publishDraft`.

- [ ] **Step 4: Update README write module docs**

Replace the current write-side section with the public flow:

```markdown
### Designing And Publishing Knowledge

When the user explicitly asks to create durable knowledge, use the write flow:

1. Run `argos knowledge design --json --project <project> --intent <intent>`.
2. Write the returned `knowledge_design_template` to `write_guidance.design_path`.
3. Ask the human to review the design and set `review.draft_write_approved`.
4. Write draft knowledge only after review approves draft writing.
5. Run `argos knowledge check --json --design <design.json> --draft <draft-path>`.
6. Publish only after explicit authorization with `argos knowledge publish --path <draft-path>`.
7. Run `argos index`.
8. Confirm the new knowledge is discoverable with `argos knowledge find --json`.
```

Do not mention removed write-side commands or old schema names in README.

- [ ] **Step 5: Run CLI and package tests**

Run:

```bash
go test ./internal/knowledgewrite ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 6: Remove `internal/author` when no imports remain**

Run:

```bash
rg -n '"argos/internal/author"|internal/author|author\\.' internal
```

Expected: no live imports after CLI and dogfood are migrated. If Task 3 or 4 still needs migration, delete `internal/author` at the end of Task 4.

- [ ] **Step 7: Commit Task 2**

```bash
git add internal/cli README.md internal/knowledgewrite internal/author
git commit -m "cli: replace author commands with knowledge write flow"
```

---

### Task 3: MCP Tools And Adapter Write Protocol

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/adapters/adapters.go`
- Modify: `internal/adapters/adapters_test.go`

- [ ] **Step 1: Write failing MCP tool list and schema tests**

In `internal/mcp/server_test.go`, add:

```go
func TestToolListIncludesKnowledgeWriteTools(t *testing.T) {
	server := NewServerWithRoot(t.TempDir(), nil)
	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("parse response: %v\n%s", err, out.String())
	}
	names := map[string]bool{}
	for _, tool := range decoded.Result.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"argos_design_knowledge", "argos_check_knowledge"} {
		if !names[want] {
			t.Fatalf("missing tool %s in %#v", want, names)
		}
	}
}

func TestToolCallArgosDesignKnowledgeReturnsWriteGuidance(t *testing.T) {
	root := t.TempDir()
	if err := workspace.Init(root); err != nil {
		t.Fatalf("init workspace: %v", err)
	}
	server := NewServerWithRoot(root, nil)
	var out bytes.Buffer
	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_design_knowledge","arguments":{"project":"mall-api","intent":"Create Redis cache best practices for future backend agents."}}}`)
	if err := server.HandleLine(line, &out); err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	text := toolTextResult(t, out.Bytes())
	if !strings.Contains(text, `"write_guidance"`) || !strings.Contains(text, `"knowledge_design_template"`) {
		t.Fatalf("missing write response fields: %s", text)
	}
	assertNoRemovedWriteTerms(t, text)
}

func TestToolCallArgosDesignKnowledgeRejectsUnknownFields(t *testing.T) {
	server := NewServerWithRoot(t.TempDir(), nil)
	var out bytes.Buffer
	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_design_knowledge","arguments":{"project":"mall-api","intent":"x","proposal_scaffold":true}}}`)
	if err := server.HandleLine(line, &out); err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_design_knowledge")
}
```

Add this helper if `server_test.go` does not already contain an equivalent:

```go
func toolTextResult(t *testing.T, body []byte) string {
	t.Helper()
	resp := decodeRPCResponse(t, body)
	if resp.Error != nil {
		t.Fatalf("unexpected rpc error: %#v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["isError"] == true {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	return firstContentText(t, result)
}
```

Add imports for `argos/internal/workspace` and `argos/internal/knowledgewrite`
when the new tests and tool calls require them.

- [ ] **Step 2: Write failing adapter tests**

In `internal/adapters/adapters_test.go`, add assertions to the existing render tests:

```go
func TestRenderAdaptersIncludeKnowledgeWriteProtocol(t *testing.T) {
	project := registry.Project{ID: "mall-api", Name: "Mall API"}
	body := RenderAGENTS(project)
	for _, want := range []string{
		"Use argos_design_knowledge when the user explicitly asks to create durable knowledge.",
		"Do not write draft knowledge from query results alone.",
		"Run argos_check_knowledge after writing draft knowledge.",
		"Publish only after explicit user authorization.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("adapter missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{
		"authoring_packet",
		"proposal_scaffold",
		"argos author inspect",
		"argos author verify",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("adapter contains removed write term %q:\n%s", forbidden, body)
		}
	}
}
```

- [ ] **Step 3: Run targeted tests and verify failure**

Run:

```bash
go test ./internal/mcp ./internal/adapters -run 'TestToolListIncludesKnowledgeWriteTools|TestToolCallArgosDesignKnowledge|TestRenderAdaptersIncludeKnowledgeWriteProtocol' -count=1
```

Expected: FAIL because tools and adapter text are not implemented.

- [ ] **Step 4: Add root-aware MCP server construction**

In `internal/mcp/server.go`, add `root string` to `Server` and replace constructors with:

```go
func NewServer(service *query.Service) *Server {
	return &Server{service: service}
}

func NewServerWithRoot(root string, store *index.Store) *Server {
	return &Server{
		root:    strings.TrimSpace(root),
		service: query.New(store),
		store:   store,
	}
}

func NewServerWithStore(store *index.Store) *Server {
	return NewServerWithRoot("", store)
}
```

Update `internal/cli/cli.go` `openMCPServer` to call `mcp.NewServerWithRoot(root, store)` when a store exists and `mcp.NewServerWithRoot(root, nil)` when the index is absent.

- [ ] **Step 5: Add MCP tool calls**

In `callTool`, add cases:

```go
case "argos_design_knowledge":
	var req knowledgewrite.DesignRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_design_knowledge: " + err.Error()), nil, nil
	}
	if err := requireStringFields(map[string]string{"project": req.Project, "intent": req.Intent}, "project", "intent"); err != nil {
		return textToolError("invalid arguments for argos_design_knowledge: " + err.Error()), nil, nil
	}
	if strings.TrimSpace(s.root) == "" {
		return textToolError("workspace root not available"), nil, nil
	}
	resp, err := knowledgewrite.Design(s.root, req)
	if err != nil {
		return textToolError("design knowledge: " + err.Error()), nil, nil
	}
	result, err := textResult(resp)
	return result, nil, err
case "argos_check_knowledge":
	var req knowledgewrite.CheckRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_check_knowledge: " + err.Error()), nil, nil
	}
	if err := requireStringFields(map[string]string{"design": req.DesignPath, "draft": req.DraftPath}, "design", "draft"); err != nil {
		return textToolError("invalid arguments for argos_check_knowledge: " + err.Error()), nil, nil
	}
	if strings.TrimSpace(s.root) == "" {
		return textToolError("workspace root not available"), nil, nil
	}
	resp, err := knowledgewrite.Check(s.root, req)
	if err != nil {
		return textToolError("check knowledge: " + err.Error()), nil, nil
	}
	result, err := textResult(resp)
	return result, nil, err
```

Add strict schemas with properties from the spec. Required fields:

- `argos_design_knowledge`: `project`, `intent`
- `argos_check_knowledge`: `design`, `draft`

- [ ] **Step 6: Update adapter text**

In `internal/adapters/adapters.go`, add write protocol bullets after discovery rules:

```text
17. Use argos_design_knowledge when the user explicitly asks to create durable knowledge.
18. Do not write draft knowledge from query results alone.
19. Write a knowledge design before draft knowledge.
20. Separate user input, observed facts, imported material, AI suggestions, assumptions, and open questions.
21. Write draft knowledge only after review approves draft writing.
22. Run argos_check_knowledge after writing draft knowledge.
23. Publish only after explicit user authorization.
24. After publishing, rebuild the index and confirm the knowledge is findable.
```

- [ ] **Step 7: Run MCP and adapter tests**

Run:

```bash
go test ./internal/mcp ./internal/adapters -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

```bash
git add internal/mcp internal/adapters internal/cli
git commit -m "mcp: add knowledge design and check tools"
```

---

### Task 4: Write Dogfood Harness Contract

**Files:**
- Create: `internal/writedogfood/`
- Delete: `internal/authoringdogfood/`
- Create: `docs/superpowers/templates/argos-write-dogfood-report.md`
- Delete: `docs/superpowers/templates/argos-authoring-dogfood-report.md`
- Create: `docs/superpowers/checklists/2026-05-03-argos-write-dogfood-checklist.md`
- Delete: `docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md`
- Create: `testdata/write-golden/`
- Delete: `testdata/authoring-golden/`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Rename dogfood package and fixtures mechanically**

Run:

```bash
mkdir -p internal/writedogfood
git mv internal/authoringdogfood/*.go internal/writedogfood/
git mv testdata/authoring-golden testdata/write-golden
git mv docs/superpowers/templates/argos-authoring-dogfood-report.md docs/superpowers/templates/argos-write-dogfood-report.md
git mv docs/superpowers/checklists/2026-05-03-argos-authoring-dogfood-checklist.md docs/superpowers/checklists/2026-05-03-argos-write-dogfood-checklist.md
```

Then change package declarations from `authoringdogfood` to `writedogfood`.

- [ ] **Step 2: Write failing report parser tests for new fields**

In `internal/writedogfood/writedogfood_test.go` or the existing renamed test file, add:

```go
func TestWriteDogfoodReportTemplateMatchesParserContract(t *testing.T) {
	data, err := os.ReadFile("../../docs/superpowers/templates/argos-write-dogfood-report.md")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	body := string(data)
	for _, want := range []string{
		"## Write Guidance",
		"State:",
		"Next action:",
		"Design path:",
		"Draft path:",
		"Draft allowed:",
		"Design only:",
		"Check result:",
		"## Review Decisions",
		"Design approved:",
		"Draft write approved:",
		"Priority must approved:",
		"Official write approved:",
		"Publish approved:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("template missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(body, forbidden) {
			t.Fatalf("template contains removed term %q:\n%s", forbidden, body)
		}
	}
}
```

- [ ] **Step 3: Update report parser model**

In `internal/writedogfood/report.go`, rename report fields:

```go
type ReviewDecisions struct {
	DesignApproved        bool `json:"design_approved"`
	DraftWriteApproved    bool `json:"draft_write_approved"`
	PriorityMustApproved  bool `json:"priority_must_approved"`
	OfficialWriteApproved bool `json:"official_write_approved"`
	PublishApproved       bool `json:"publish_approved"`
}

type Report struct {
	CaseID          string          `json:"case_id"`
	State           string          `json:"state"`
	NextAction      string          `json:"next_action"`
	DesignPath      string          `json:"design_path"`
	DraftPath       string          `json:"draft_path"`
	DraftAllowed    bool            `json:"draft_allowed"`
	DesignOnly      bool            `json:"design_only"`
	CheckResult     string          `json:"check_result"`
	Review          ReviewDecisions `json:"review"`
	Guards          map[string]string `json:"guards"`
	Result          string          `json:"result"`
	MissingSections []string        `json:"missing_sections"`
	MissingFields   []string        `json:"missing_fields"`
	fieldPresence   map[string]bool
}
```

Required sections:

- `Inputs`
- `Write Guidance`
- `Artifacts`
- `Review Decisions`
- `Guards`
- `Result`

Required guards:

- `Design reviewed before draft write`
- `Sources and scope documented`
- `Future use documented`
- `Draft stayed in approved area`
- `Official knowledge unchanged`
- `Publish not run`
- `Check run`

- [ ] **Step 4: Make packet generation consume real write guidance**

In `internal/writedogfood/packet.go`:

- Import `argos/internal/knowledgewrite`.
- Extend `Packet` with `WriteGuidance knowledgewrite.WriteGuidance`.
- Seed fixture workspace before calling `BuildPacket` in CLI, or make `BuildPacket` call `knowledgewrite.Design` after the workspace exists.
- Render packet sections:
  - `## Write Guidance`
  - `## Knowledge Design Template`
  - `## CLI Equivalent`
  - `argos knowledge design --json --project <project> --intent <intent>`
  - `argos knowledge check --json --design <design-path> --draft <draft-path>`

Do not render removed write terms.

- [ ] **Step 5: Update evaluator to enforce guidance**

In `internal/writedogfood/evaluate.go`:

- Load the expected public case.
- Recompute write guidance using `knowledgewrite.Design`.
- Compare report `state`, `next_action`, `design_path`, `draft_path`, `draft_allowed`, and `design_only`.
- Load `KnowledgeDesign` through `knowledgewrite.LoadDesign`.
- If `DraftAllowed=false` or report review blocks draft writing, fail on non-empty draft path.
- If draft path exists and review permits draft writing, run `knowledgewrite.Check`.
- Enforce hidden oracle through new design field names.

Use public messages with new vocabulary:

- `draft write was not approved`
- `design artifact missing`
- `check result did not match workspace check`
- `workflow guards do not satisfy hidden evaluation requirements`

- [ ] **Step 6: Update CLI dogfood command**

In `internal/cli/cli.go`:

- Replace `dogfood authoring` with `dogfood write`.
- Update defaults:
  - `defaultWriteDogfoodCasesPath = "testdata/write-golden/cases.json"`
  - `defaultWriteDogfoodFixtures = "testdata/write-golden/fixtures"`
- Update error prefixes to `dogfood write`, `dogfood write cases`,
  `dogfood write packet`, and `dogfood write evaluate`.

- [ ] **Step 7: Run dogfood and CLI tests**

Run:

```bash
go test ./internal/writedogfood ./internal/cli -run 'Write|Dogfood|TestRunDogfood' -count=1
go test ./internal/writedogfood ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Task 4**

```bash
git add internal/writedogfood internal/authoringdogfood testdata/write-golden testdata/authoring-golden docs/superpowers/templates docs/superpowers/checklists internal/cli
git commit -m "dogfood: enforce knowledge write guidance"
```

---

### Task 5: End-To-End Closure And Removed Vocabulary Scan

**Files:**
- Create: `internal/knowledgewrite/vocabulary_test.go`
- Modify: `README.md`
- Modify: `docs/superpowers/reports/2026-05-03-argos-write-dogfood-round-0.md`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add live vocabulary scan test**

Replace the Task 1 helper-only `internal/knowledgewrite/vocabulary_test.go` with this full scan file:

```go
package knowledgewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func removedAuthoringTermsForTests() []string {
	return []string{
		"authoring_packet",
		"proposal_scaffold",
		"source_profile",
		"proposed_shape",
		"overlap_decision",
		"verification_plan",
		"human_review",
		"artifact_state",
		"author inspect",
		"author verify",
		"authoring.proposal.v2",
	}
}

func TestRemovedWriteVocabularyAbsentFromLiveSurface(t *testing.T) {
	root := repositoryRoot(t)
	allowed := []string{
		filepath.FromSlash("docs/superpowers/specs/"),
		filepath.FromSlash("docs/superpowers/plans/"),
		filepath.FromSlash("docs/superpowers/reports/"),
		filepath.FromSlash(".git/"),
		filepath.FromSlash(".worktrees/"),
	}
	liveExtensions := map[string]bool{
		".go": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
	}
	var failures []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if entry.IsDir() {
			for _, prefix := range allowed {
				if rel == strings.TrimSuffix(prefix, string(filepath.Separator)) || strings.HasPrefix(rel+string(filepath.Separator), prefix) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !liveExtensions[filepath.Ext(path)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(data)
		for _, term := range removedAuthoringTermsForTests() {
			if strings.Contains(body, term) {
				failures = append(failures, rel+": "+term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}
	if len(failures) > 0 {
		t.Fatalf("removed write vocabulary found in live surface:\n%s", strings.Join(failures, "\n"))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repository root not found from %s", dir)
		}
		dir = parent
	}
}
```

- [ ] **Step 2: Add an end-to-end CLI test for design/check/publish/findback**

In `internal/cli/cli_test.go`, add:

```go
func TestKnowledgeWritePublishAndFindbackFlow(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	runOK(t, root, []string{"knowledge", "design", "--json", "--project", "mall-api", "--intent", "Create Redis cache best practices for future backend agents."})

	design := validKnowledgeDesignForCLI(t)
	design.Review.DesignApproved = true
	design.Review.DraftWriteApproved = true
	design.Review.PublishApproved = true
	designPath := writeJSONFixture(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", design)
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackageForCLI(t, root, draftPath)

	checkOut := runOK(t, root, []string{"knowledge", "check", "--json", "--design", designPath, "--draft", draftPath})
	if !strings.Contains(checkOut, `"result": "pass"`) && !strings.Contains(checkOut, `"result": "review-needed"`) {
		t.Fatalf("unexpected check output: %s", checkOut)
	}

	runOK(t, root, []string{"knowledge", "publish", "--path", draftPath})
	runOK(t, root, []string{"index"})
	found := runOK(t, root, []string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "redis cache best practices"})
	if !strings.Contains(found, "package:mall-api.redis-cache.v1") {
		t.Fatalf("published knowledge was not findable: %s", found)
	}
}
```

Use existing CLI helpers `chdir`, `writeCLIFile`, and `Run`. Add these helpers
near the other CLI helpers:

```go
func validKnowledgeDesignForCLI(t *testing.T) knowledgewrite.KnowledgeDesign {
	t.Helper()
	return knowledgewrite.KnowledgeDesign{
		SchemaVersion: "knowledge.design.v1",
		UserRequest:   "Create Redis cache best practices for future backend agents.",
		KnowledgeGoal: "Design reusable knowledge for future agents: Redis cache best practices.",
		Project:       "mall-api",
		Audience: knowledgewrite.Audience{
			Primary: "implementer_agent",
			AgentActionsSupported: []string{"apply Redis cache guidance during implementation"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			TechDomains:  []string{"backend"},
			FileGlobs:    []string{"**/*"},
		},
		Sources: knowledgewrite.Sources{
			UserInput:   []string{"User requested Redis cache best practices."},
			AISuggested: []string{"Use reviewed Redis recommendations as draft guidance."},
			Assumptions: []string{"Team will review before publishing."},
			Claims: []knowledgewrite.SourceClaim{{
				Claim:          "Redis cache best practices are being designed for future agents.",
				Kind:           "recommendation",
				Trust:          "ai_suggested",
				Source:         []string{"argos knowledge design"},
				RequiresReview: true,
			}},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             "package:mall-api.redis-cache.v1",
			Path:           "knowledge/.inbox/packages/mall-api/redis-cache",
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package gives future agents one entrypoint.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: knowledgewrite.FutureUse{
			TriggerRequests: []string{"implement Redis cache behavior"},
			QueryPhrases:    []string{"redis cache best practices"},
			ExpectedUse:     "Future agents should read this before implementing Redis cache behavior.",
			CitationPolicy:  "cite_after_use",
		},
		Applicability: knowledgewrite.Applicability{
			WhenToUse:    []string{"When implementing Redis-backed caching."},
			WhenNotToUse: []string{"When changing unrelated storage behavior."},
		},
		ExistingKnowledge: knowledgewrite.ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing knowledge covers this exact guidance.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path:                        "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:        true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path:    "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
			Purpose: "entrypoint",
			Load:    "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: "knowledge/.inbox/packages/mall-api/redis-cache",
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{
			Questions:          []string{"Is draft writing approved?"},
			DesignApproved:     true,
			DraftWriteApproved: true,
			PublishApproved:    true,
		},
	}
}

func writeDraftPackageForCLI(t *testing.T, root string, rel string) {
	t.Helper()
	writeCLIFile(t, root, filepath.ToSlash(filepath.Join(rel, "KNOWLEDGE.md")), `---
id: package:mall-api.redis-cache.v1
type: package
title: Redis Cache Best Practices
status: draft
priority: should
projects: [mall-api]
tech_domains: [backend]
tags: [cache, redis]
applies_to:
  files: ["**/*"]
---

# Redis Cache Best Practices

Use reviewed Redis cache guidance for future backend implementation.
`)
}

func runOK(t *testing.T, root string, args []string) string {
	t.Helper()
	chdir(t, root)
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) code = %d, stderr = %s", args, code, stderr.String())
	}
	return stdout.String()
}

func writeJSONFixture(t *testing.T, root string, rel string, value any) string {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	writeCLIFile(t, root, rel, string(body)+"\n")
	return rel
}
```

- [ ] **Step 3: Update round report**

Create or update `docs/superpowers/reports/2026-05-03-argos-write-dogfood-round-0.md` with:

```markdown
# Argos Write Dogfood Round 0

Date: 2026-05-03

## Purpose

Validate the first-release write module flow:

intent -> knowledge_design -> draft_knowledge -> check -> publish -> query findback

## Result

Status: not-run

## Required Cases

- Go template standard
- Redis best practices
- Business interface knowledge
- Missing content design-only
- Existing knowledge decision
- Draft check and query findback

## Next Action

Run fresh write dogfood sessions after the implementation passes full tests.
```

Update it with actual results if fresh dogfood runs happen during execution.

- [ ] **Step 4: Run full removed vocabulary scan**

Run:

```bash
go test ./internal/knowledgewrite -run TestRemovedWriteVocabularyAbsentFromLiveSurface -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Final live grep sanity check**

Run:

```bash
rg -n "authoring_packet|proposal_scaffold|source_profile|proposed_shape|overlap_decision|verification_plan|human_review|artifact_state|author inspect|author verify|authoring\\.proposal\\.v2" \
  --glob '!docs/superpowers/specs/**' \
  --glob '!docs/superpowers/plans/**' \
  --glob '!docs/superpowers/reports/**' \
  --glob '!.worktrees/**'
```

Expected: no output.

- [ ] **Step 7: Commit Task 5**

```bash
git add internal/knowledgewrite internal/cli README.md docs/superpowers/reports
git commit -m "test: validate knowledge write release flow"
```

---

## Final Verification

Run:

```bash
go test ./... -count=1
git status --short --branch
```

Expected:

- all packages pass;
- branch contains only intentional commits;
- no live write-side removed terms outside historical specs, plans, and reports.
