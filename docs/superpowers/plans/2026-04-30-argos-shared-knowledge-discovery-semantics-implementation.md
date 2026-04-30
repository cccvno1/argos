# Argos Shared Knowledge Discovery Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fully migrate Discovery from capture-oriented `gap_candidates` semantics to shared-knowledge `coverage_gaps` semantics.

**Architecture:** Replace the query response field and Go type in one breaking migration, then update golden, CLI, MCP, adapters, templates, and dogfood. Keep `recall.semantic` and the existing progressive-disclosure/citation-accountability behavior intact. Treat coverage gaps as attribution boundaries, not upload or capture proposals.

**Tech Stack:** Go, JSON response structs, existing query/CLI/MCP/adapters packages, golden discovery fixtures, Markdown dogfood reports.

---

## Scope

This plan implements `docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md`.

This is a development-stage breaking migration:

- Remove production JSON field `gap_candidates`.
- Remove Go type `GapCandidate`.
- Remove `capture_candidate`, `candidate_only`, and `proposal_required` from Discovery responses and adapter guidance.
- Add production JSON field `coverage_gaps`.
- Add Go type `CoverageGap`.
- Change partial claim policy to `must_separate_argos_backed_and_general_reasoning`.
- Keep `coverage.missing_knowledge_hints` for now as historical/internal compatibility; remove or rename it in the later global naming audit.

## File Structure

- Modify `internal/query/query.go`: response fields, `CoverageGap` type, helper functions, action policy wording.
- Modify `internal/query/query_test.go`: focused TDD tests for coverage gaps and absence of `gap_candidates`.
- Modify `internal/discoverytest/golden.go`: expected coverage gap fields.
- Modify `testdata/discovery-golden/cases.json`: expected `coverage_gap_sources`.
- Modify `internal/query/discovery_golden_test.go`: assert `coverage_gaps`.
- Modify `internal/cli/discovery_golden_test.go`: assert CLI JSON uses `coverage_gaps`.
- Modify `internal/mcp/server_test.go`: assert MCP JSON uses `coverage_gaps`.
- Modify `internal/adapters/adapters.go`: shared uploaded knowledge and attribution wording.
- Modify `internal/adapters/adapters_test.go`: lock adapter wording and forbid old gap/capture wording.
- Modify `docs/superpowers/templates/argos-discovery-dogfood-report.md`: coverage gap and attribution guards.
- Add `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-5.md`: fresh runner dogfood result.

## Task 1: Replace Query Gap Candidates With Coverage Gaps

**Files:**
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/query.go`

- [ ] **Step 1: Add failing query tests**

In `internal/query/query_test.go`, add `encoding/json` to imports:

```go
import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)
```

Replace `TestDiscoverGapCandidatesForNoneCoverage`, `TestDiscoverStrongCoverageOmitsGapCandidates`, and `assertGapCandidateKinds` with these tests and helper:

```go
func TestDiscoverCoverageGapsForNoneCoverage(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"unmatched_intent"})
	for _, gap := range result.CoverageGaps {
		if gap.Need == "" {
			t.Fatalf("expected coverage gap need, got %#v", gap)
		}
		if gap.Reason == "" {
			t.Fatalf("expected coverage gap reason, got %#v", gap)
		}
		if gap.Severity != "blocking" {
			t.Fatalf("expected blocking severity for none coverage, got %#v", gap)
		}
		if gap.ArgosBacked {
			t.Fatalf("coverage gaps must not be Argos-backed: %#v", gap)
		}
	}
}

func TestDiscoverWeakCoverageGapsAreNotArgosBacked(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/generic-token.md",
		ID:              "rule:backend.generic-token.v1",
		Title:           "Generic token rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Token guidance applies to platform work.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "weak" {
		t.Fatalf("expected weak coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"weak_match"})
	for _, gap := range result.CoverageGaps {
		if gap.ArgosBacked {
			t.Fatalf("coverage gaps must not be Argos-backed: %#v", gap)
		}
	}
}

func TestDiscoverPartialCoverageUsesAttributionClaim(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "partial" {
		t.Fatalf("expected partial coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"partial_match"})
	if result.ActionPolicy.Claim != "must_separate_argos_backed_and_general_reasoning" {
		t.Fatalf("expected attribution claim policy, got %#v", result.ActionPolicy)
	}
}

func TestDiscoverStrongCoverageOmitsCoverageGaps(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "strong" {
		t.Fatalf("expected strong coverage, got %#v", result.Coverage)
	}
	if len(result.CoverageGaps) != 0 {
		t.Fatalf("strong coverage should not produce coverage gaps: %#v", result.CoverageGaps)
	}
}

func TestDiscoverJSONDoesNotExposeGapCandidates(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal discovery response: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"coverage_gaps"`) {
		t.Fatalf("expected coverage_gaps in JSON: %s", body)
	}
	if strings.Contains(body, "gap_candidates") {
		t.Fatalf("did not expect gap_candidates in JSON: %s", body)
	}
	if strings.Contains(body, "capture_candidate") || strings.Contains(body, "candidate_only") || strings.Contains(body, "proposal_required") {
		t.Fatalf("did not expect capture-oriented gap semantics in JSON: %s", body)
	}
}

func assertCoverageGapSources(t *testing.T, got []CoverageGap, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected coverage gap sources %v, got %#v", want, got)
	}
	seen := map[string]bool{}
	for _, gap := range got {
		seen[gap.Source] = true
		if gap.Need == "" {
			t.Fatalf("expected need for %#v", gap)
		}
		if gap.Reason == "" {
			t.Fatalf("expected reason for %#v", gap)
		}
		if gap.Severity == "" {
			t.Fatalf("expected severity for %#v", gap)
		}
		if gap.ArgosBacked {
			t.Fatalf("coverage gap must not be Argos-backed: %#v", gap)
		}
	}
	for _, source := range want {
		if !seen[source] {
			t.Fatalf("expected coverage gap source %q in %#v", source, got)
		}
	}
}
```

- [ ] **Step 2: Run query tests to verify red**

Run:

```bash
go test ./internal/query -run 'TestDiscoverCoverageGapsForNoneCoverage|TestDiscoverWeakCoverageGapsAreNotArgosBacked|TestDiscoverPartialCoverageUsesAttributionClaim|TestDiscoverStrongCoverageOmitsCoverageGaps|TestDiscoverJSONDoesNotExposeGapCandidates' -count=1
```

Expected: FAIL to compile because `CoverageGaps` and `CoverageGap` do not exist.

- [ ] **Step 3: Replace response field and type**

In `internal/query/query.go`, replace the `DiscoveryResponse` gap field:

```go
type DiscoveryResponse struct {
	Project      string                      `json:"project"`
	Phase        string                      `json:"phase"`
	Query        string                      `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Coverage     Coverage                    `json:"coverage"`
	ActionPolicy ActionPolicy                `json:"action_policy"`
	Recall       RecallState                 `json:"recall"`
	CoverageGaps []CoverageGap               `json:"coverage_gaps,omitempty"`
	Items        []DiscoveryItem             `json:"items"`
	NextCalls    []RecommendedCall           `json:"next_calls"`
}
```

Replace `type GapCandidate` with:

```go
type CoverageGap struct {
	Need        string `json:"need"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	Severity    string `json:"severity"`
	ArgosBacked bool   `json:"argos_backed"`
}
```

- [ ] **Step 4: Replace gap helper implementation**

In `internal/query/query.go`, replace `gapCandidatesForCoverage` and `titleFromGapHint` with:

```go
func coverageGapsForCoverage(coverage Coverage, req DiscoverRequest, intent string) []CoverageGap {
	if coverage.Status == "strong" || len(coverage.MissingKnowledgeHints) == 0 {
		return nil
	}
	need := coverageGapNeed(req, intent)
	if need == "" {
		return nil
	}
	source := coverageGapSource(coverage, req)
	severity := coverageGapSeverity(coverage, source)
	return []CoverageGap{{
		Need:        need,
		Reason:      coverageGapReason(coverage, source, need),
		Source:      source,
		Severity:    severity,
		ArgosBacked: false,
	}}
}

func coverageGapNeed(req DiscoverRequest, intent string) string {
	task := strings.TrimSpace(req.Task)
	query := strings.TrimSpace(req.Query)
	if task == "" {
		return query
	}
	if query == "" {
		return task
	}
	taskLower := strings.ToLower(task)
	queryLower := strings.ToLower(query)
	if strings.Contains(taskLower, queryLower) {
		return task
	}
	if strings.Contains(queryLower, taskLower) {
		return query
	}
	return strings.TrimSpace(task + " / " + query)
}

func coverageGapSource(coverage Coverage, req DiscoverRequest) string {
	if hasExplicitDiscoveryFilters(req) && coverage.Status == "none" {
		return "filter_excluded"
	}
	switch coverage.Status {
	case "partial":
		return "partial_match"
	case "weak":
		return "weak_match"
	default:
		return "unmatched_intent"
	}
}

func hasExplicitDiscoveryFilters(req DiscoverRequest) bool {
	return len(req.Types) > 0 || len(req.Tags) > 0 || len(req.Domains) > 0 || len(req.Status) > 0 || req.IncludeDeprecated
}

func coverageGapSeverity(coverage Coverage, source string) string {
	if coverage.Status == "none" {
		return "blocking"
	}
	if source == "weak_match" || source == "partial_match" || source == "filter_excluded" {
		return "important"
	}
	return "informational"
}

func coverageGapReason(coverage Coverage, source string, need string) string {
	switch source {
	case "filter_excluded":
		return "Explicit discovery filters excluded shared knowledge that might otherwise match: " + need
	case "partial_match":
		return "Some shared knowledge matched, but it does not fully cover this task need: " + need
	case "weak_match":
		return "Only weak shared knowledge matched, so this need is not Argos-backed: " + need
	default:
		return "No sufficiently relevant shared knowledge matched this task need: " + need
	}
}
```

- [ ] **Step 5: Populate response and update partial policy**

In `Service.Discover`, replace:

```go
GapCandidates: gapCandidatesForCoverage(coverage),
```

with:

```go
CoverageGaps: coverageGapsForCoverage(coverage, req, intent),
```

In `discoveryActionPolicy`, replace the `partial` case with:

```go
case "partial":
	return ActionPolicy{
		Authority: "partial",
		Load:      "allowed",
		Cite:      "after_loaded_and_used",
		Claim:     "must_separate_argos_backed_and_general_reasoning",
		Reason:    "Partial Argos coverage; load only relevant shared knowledge and separate Argos-backed claims from general reasoning.",
	}
```

- [ ] **Step 6: Run query tests to verify green**

Run:

```bash
gofmt -w internal/query/query.go internal/query/query_test.go
go test ./internal/query -run 'TestDiscoverCoverageGapsForNoneCoverage|TestDiscoverWeakCoverageGapsAreNotArgosBacked|TestDiscoverPartialCoverageUsesAttributionClaim|TestDiscoverStrongCoverageOmitsCoverageGaps|TestDiscoverJSONDoesNotExposeGapCandidates' -count=1
go test ./internal/query -count=1
```

Expected: all tests pass.

- [ ] **Step 7: Commit query migration**

Run:

```bash
git add internal/query/query.go internal/query/query_test.go
git commit -m "feat: rename discovery gaps to coverage gaps"
```

## Task 2: Update Golden, CLI, And MCP Interface Tests

**Files:**
- Modify: `internal/discoverytest/golden.go`
- Modify: `testdata/discovery-golden/cases.json`
- Modify: `internal/query/discovery_golden_test.go`
- Modify: `internal/cli/discovery_golden_test.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Update golden expected fields**

In `internal/discoverytest/golden.go`, replace:

```go
GapCandidateKinds []string `json:"gap_candidate_kinds"`
```

with:

```go
CoverageGapSources []string `json:"coverage_gap_sources"`
```

- [ ] **Step 2: Update golden cases JSON**

In `testdata/discovery-golden/cases.json`:

Replace all:

```json
"action_claim": "must_mention_gap"
```

with:

```json
"action_claim": "must_separate_argos_backed_and_general_reasoning"
```

For each partial case, replace:

```json
"gap_candidate_kinds": ["standard", "decision", "lesson"]
```

with:

```json
"coverage_gap_sources": ["partial_match"]
```

For each weak case, replace:

```json
"gap_candidate_kinds": ["standard", "decision", "lesson"]
```

with:

```json
"coverage_gap_sources": ["weak_match"]
```

For `none_payment_webhook` and `none_project_scope_mismatch`, replace:

```json
"gap_candidate_kinds": ["standard", "decision", "lesson"]
```

with:

```json
"coverage_gap_sources": ["unmatched_intent"]
```

For `none_explicit_filter_excludes_match`, replace:

```json
"gap_candidate_kinds": ["standard", "decision", "lesson"]
```

with:

```json
"coverage_gap_sources": ["filter_excluded"]
```

- [ ] **Step 3: Update query golden assertions**

In `internal/query/discovery_golden_test.go`, replace:

```go
assertGapCandidatesMatchExpected(t, result.GapCandidates, tc.Expected.GapCandidateKinds)
```

with:

```go
assertCoverageGapsMatchExpected(t, result.CoverageGaps, tc.Expected.CoverageGapSources)
```

Replace `assertGapCandidatesMatchExpected` with:

```go
func assertCoverageGapsMatchExpected(t *testing.T, got []CoverageGap, want []string) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("expected no coverage gaps, got %#v", got)
		}
		return
	}
	assertCoverageGapSources(t, got, want)
}
```

- [ ] **Step 4: Update CLI tests**

In `internal/cli/discovery_golden_test.go`, replace test `TestGoldenCLIDiscoverNoneIncludesGapCandidates` with:

```go
func TestGoldenCLIDiscoverNoneIncludesCoverageGaps(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "none_payment_webhook")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"discover", "--json",
		"--project", tc.Input.Project,
		"--phase", tc.Input.Phase,
		"--task", tc.Input.Task,
		"--query", tc.Input.Query,
		"--limit", strconv.Itoa(tc.Input.Limit),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.DiscoveryResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse discover JSON: %v\n%s", err, stdout.String())
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCLICoverageGapSources(t, result.CoverageGaps, tc.Expected.CoverageGapSources)
	for _, gap := range result.CoverageGaps {
		if gap.ArgosBacked {
			t.Fatalf("coverage gaps must not be Argos-backed, got %#v", gap)
		}
	}
	if strings.Contains(stdout.String(), "gap_candidates") {
		t.Fatalf("did not expect gap_candidates in CLI JSON: %s", stdout.String())
	}
}
```

Replace `assertCLIGapCandidateKinds` with:

```go
func assertCLICoverageGapSources(t *testing.T, got []query.CoverageGap, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected coverage gap sources %v, got %#v", want, got)
	}
	seen := map[string]bool{}
	for _, gap := range got {
		seen[gap.Source] = true
	}
	for _, source := range want {
		if !seen[source] {
			t.Fatalf("expected coverage gap source %q in %#v", source, got)
		}
	}
}
```

- [ ] **Step 5: Update MCP tests**

In `internal/mcp/server_test.go`, replace `TestToolCallArgosDiscoverNoneReturnsGapCandidates` with:

```go
func TestToolCallArgosDiscoverNoneReturnsCoverageGaps(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","phase":"implementation","task":"add payment webhook signature verification","query":"payment webhook signature","limit":5}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	for _, fragment := range []string{
		`"coverage"`,
		`"status": "none"`,
		`"coverage_gaps"`,
		`"source": "unmatched_intent"`,
		`"argos_backed": false`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected %q in discover response: %s", fragment, text)
		}
	}
	if strings.Contains(text, `"gap_candidates"`) {
		t.Fatalf("did not expect gap_candidates in discover response: %s", text)
	}
	if strings.Contains(text, `"body"`) {
		t.Fatalf("discover should not return full body: %s", text)
	}
}
```

- [ ] **Step 6: Run interface tests**

Run:

```bash
gofmt -w internal/discoverytest/golden.go internal/query/discovery_golden_test.go internal/cli/discovery_golden_test.go internal/mcp/server_test.go
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp -count=1
```

Expected: all tests pass.

- [ ] **Step 7: Commit interface migration**

Run:

```bash
git add internal/discoverytest/golden.go internal/query/discovery_golden_test.go internal/cli/discovery_golden_test.go internal/mcp/server_test.go testdata/discovery-golden/cases.json
git commit -m "test: assert discovery coverage gap interfaces"
```

## Task 3: Update Adapter Guidance And Dogfood Template

**Files:**
- Modify: `internal/adapters/adapters_test.go`
- Modify: `internal/adapters/adapters.go`
- Modify: `docs/superpowers/templates/argos-discovery-dogfood-report.md`

- [ ] **Step 1: Update adapter contract test**

In `internal/adapters/adapters_test.go`, replace expected strings:

```go
"Argos is a project knowledge layer.",
"Use argos_discover to route current work to relevant Argos knowledge.",
"Cite Argos knowledge IDs used in final responses.",
"Mention Argos coverage gaps when action_policy.claim is must_mention_gap.",
"Treat gap_candidates as candidate prompts, not official knowledge.",
"Do not cite gap_candidates.",
"Start capture-knowledge only with user approval.",
"Argos validation does not replace tests, builds, linting, or review.",
```

with:

```go
"Argos is a shared knowledge layer built from uploaded reusable knowledge.",
"Use argos_discover to route current work to relevant shared knowledge.",
"Cite Argos knowledge IDs used in final responses only after loading and applying them.",
"When coverage_gaps are present, separate Argos-backed claims from general reasoning.",
"Do not cite coverage_gaps; they are coverage boundaries, not knowledge items.",
"Do not start upload, capture, or inbox creation from Discovery alone.",
"Semantic recall never overrides action_policy.",
"Argos validation does not replace tests, builds, linting, or review.",
```

Add a forbidden wording assertion to the same test body after the expected-string loop:

```go
for _, forbidden := range []string{
	"gap_candidates",
	"capture_candidate",
	"candidate_only",
	"proposal_required",
	"Start capture-knowledge only with user approval.",
	"official knowledge",
} {
	if strings.Contains(tt.body, forbidden) {
		t.Fatalf("did not expect %q in %s adapter:\n%s", forbidden, tt.name, tt.body)
	}
}
```

- [ ] **Step 2: Run adapter test to verify red**

Run:

```bash
go test ./internal/adapters -run TestRenderedAdaptersIncludeStableKnowledgeContract -count=1
```

Expected: FAIL because old adapter wording is still rendered.

- [ ] **Step 3: Update adapter renderer**

In `internal/adapters/adapters.go`, replace:

```go
Argos is a project knowledge layer. Use it to retrieve, apply, validate, and cite durable project knowledge.
```

with:

```go
Argos is a shared knowledge layer built from uploaded reusable knowledge. Use it to retrieve, apply, validate, and cite shared knowledge with clear attribution.
```

Replace the Work Protocol block with:

```markdown
1. Before substantial project work, call or emulate argos_context.
2. Use argos_map for broad orientation before unfamiliar project work.
3. Use argos_discover to route current work to relevant shared knowledge.
4. Before implementation or review, call or emulate argos_standards.
5. Follow recommended calls returned by Argos.
6. Load full knowledge items only through get_knowledge_item when routed to specific IDs or paths.
7. Cite Argos knowledge IDs used in final responses only after loading and applying them.
8. Do not cite IDs returned only by argos_map or argos_discover.
9. Cite only knowledge IDs whose full item was loaded with get_knowledge_item and actually applied.
10. Follow action_policy.load before loading full knowledge items.
11. Follow action_policy.cite before calling cite_knowledge.
12. When coverage_gaps are present, separate Argos-backed claims from general reasoning.
13. Do not cite coverage_gaps; they are coverage boundaries, not knowledge items.
14. Do not start upload, capture, or inbox creation from Discovery alone.
15. Semantic recall never overrides action_policy.
```

Replace the Boundaries block with:

```markdown
Argos validation does not replace tests, builds, linting, or review.
Shared knowledge should not be changed or uploaded without the user's explicit request.
```

- [ ] **Step 4: Update dogfood template**

In `docs/superpowers/templates/argos-discovery-dogfood-report.md`, replace:

```markdown
- Missing knowledge hints:
- Gap candidates:
```

with:

```markdown
- Missing knowledge hints:
- Coverage gaps:
- Argos-backed vs general reasoning:
```

Replace guard lines:

```markdown
- Gap candidates not cited: `pass|fail|not-applicable`
- Capture permission gate: `pass|fail|not-applicable`
```

with:

```markdown
- Coverage gaps not cited: `pass|fail|not-applicable`
- Attribution boundary: `pass|fail|not-applicable`
- No Discovery-triggered upload/capture: `pass|fail|not-applicable`
```

- [ ] **Step 5: Run adapter tests and wording scan**

Run:

```bash
gofmt -w internal/adapters/adapters.go internal/adapters/adapters_test.go
go test ./internal/adapters -count=1
rg -n "gap_candidates|capture_candidate|candidate_only|proposal_required|official knowledge|Start capture-knowledge" internal/adapters docs/superpowers/templates/argos-discovery-dogfood-report.md
```

Expected: adapter tests pass; `rg` returns no matches.

- [ ] **Step 6: Commit adapter and template migration**

Run:

```bash
git add internal/adapters/adapters.go internal/adapters/adapters_test.go docs/superpowers/templates/argos-discovery-dogfood-report.md
git commit -m "docs: teach adapters shared knowledge coverage gaps"
```

## Task 4: Update Historical Round 4 Report With Supersession Note

**Files:**
- Modify: `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md`

- [ ] **Step 1: Add supersession notice**

At the top of `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md`, after the title, add:

```markdown
> Superseded terminology note: this report used the earlier `gap_candidates`
> wording. The active Discovery direction after
> `2026-04-30-argos-shared-knowledge-discovery-semantics-design.md` is
> `coverage_gaps`, meaning uncovered shared-knowledge needs rather than capture
> or upload candidates.
```

- [ ] **Step 2: Commit report note**

Run:

```bash
git add docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md
git commit -m "docs: mark round 4 gap wording superseded"
```

## Task 5: Run Round 5 Shared Knowledge Dogfood

**Files:**
- Add: `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-5.md`

- [ ] **Step 1: Build dogfood binary and workspace**

Run:

```bash
go build -o /tmp/argos-dogfood ./cmd/argos
full=$(mktemp -d /tmp/argos-round5-full-XXXXXX)
cp -R testdata/discovery-golden/knowledge "$full/knowledge"
(cd "$full" && /tmp/argos-dogfood index)
printf 'FULL=%s\n' "$full"
```

Expected: command exits 0, prints `indexed 9 knowledge item(s)`, and prints a concrete temporary workspace path with the `FULL=` prefix.

- [ ] **Step 2: Run fresh dogfood runners**

Use fresh runner sessions with `fork_context:false`. Each runner must:

- use only `/tmp/argos-dogfood` and the temporary workspace;
- not read `cases.json`, SQLite files, vector stores, Markdown knowledge source, or old reports;
- inspect JSON output only;
- follow `action_policy`;
- never cite `coverage_gaps`;
- not start upload/capture from Discovery alone.

Run these cases:

1. `strong_shared_knowledge_no_coverage_gaps`
   - project `mall-api`
   - phase `implementation`
   - task `add refresh token endpoint`
   - query `refresh token session renewal`
   - files `internal/auth/session.go`
   - expect strong coverage, `coverage_gaps` absent, semantic disabled, load recommendation present.

2. `partial_shared_knowledge_requires_attribution_boundary`
   - project `mall-api`
   - phase `debugging`
   - task `debug session renewal test failure`
   - query `session renewal tests fail logs`
   - expect partial coverage, `claim=must_separate_argos_backed_and_general_reasoning`, `coverage_gaps` with `source=partial_match`, targeted load allowed.

3. `weak_shared_knowledge_not_argos_backed`
   - project `mall-api`
   - phase `implementation`
   - task `add warehouse barcode scanner`
   - query `barcode scanner token`
   - expect weak coverage, load/cite forbidden, `coverage_gaps` with `source=weak_match`, no Argos-backed claim.

4. `none_shared_knowledge_can_continue_without_argos_claim`
   - project `mall-api`
   - phase `implementation`
   - task `add payment webhook signature verification`
   - query `payment webhook signature`
   - expect none coverage, no items, no next calls, `coverage_gaps` with `source=unmatched_intent`, agent says normal work may continue without Argos-backed attribution.

5. `filter_excluded_shared_knowledge_boundary`
   - project `mall-api`
   - phase `implementation`
   - task `add refresh token endpoint`
   - query `refresh token`
   - tags `payments`
   - expect `coverage_gaps` with `source=filter_excluded` and no bypass of user filter.

- [ ] **Step 3: Record Round 5 report**

Create `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-5.md` with:

```markdown
# Argos Discovery Dogfood Round 5

Date: 2026-04-30
Workspace: write the concrete `FULL=` path printed by the build step.
Binary: `/tmp/argos-dogfood`

## Goal

Validate shared-knowledge Discovery semantics after migrating from
`gap_candidates` to `coverage_gaps`.

## Cases

### strong_shared_knowledge_no_coverage_gaps

- Actual coverage: write the observed status and confidence.
- Action policy: write observed authority, load, cite, and claim values.
- Recall: write observed semantic status and reason.
- Coverage gaps: write observed gap count and source values.
- Discovered IDs: write observed IDs.
- Loaded IDs: write observed IDs or `none`.
- Cited IDs: write observed IDs or `none`.
- Next calls: write observed tools and IDs.
- Progressive disclosure: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Attribution boundary: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Coverage gaps not cited: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- No Discovery-triggered upload/capture: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Result: write `pass`, `fail`, or `review-needed`.

### partial_shared_knowledge_requires_attribution_boundary

- Actual coverage: write the observed status and confidence.
- Action policy: write observed authority, load, cite, and claim values.
- Recall: write observed semantic status and reason.
- Coverage gaps: write observed gap count and source values.
- Discovered IDs: write observed IDs.
- Loaded IDs: write observed IDs or `none`.
- Cited IDs: write observed IDs or `none`.
- Next calls: write observed tools and IDs.
- Progressive disclosure: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Attribution boundary: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Coverage gaps not cited: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- No Discovery-triggered upload/capture: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Result: write `pass`, `fail`, or `review-needed`.

### weak_shared_knowledge_not_argos_backed

- Actual coverage: write the observed status and confidence.
- Action policy: write observed authority, load, cite, and claim values.
- Recall: write observed semantic status and reason.
- Coverage gaps: write observed gap count and source values.
- Discovered IDs: write observed IDs.
- Loaded IDs: write observed IDs or `none`.
- Cited IDs: write observed IDs or `none`.
- Next calls: write observed tools and IDs.
- Progressive disclosure: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Attribution boundary: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Coverage gaps not cited: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- No Discovery-triggered upload/capture: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Result: write `pass`, `fail`, or `review-needed`.

### none_shared_knowledge_can_continue_without_argos_claim

- Actual coverage: write the observed status and confidence.
- Action policy: write observed authority, load, cite, and claim values.
- Recall: write observed semantic status and reason.
- Coverage gaps: write observed gap count and source values.
- Discovered IDs: write observed IDs.
- Loaded IDs: write observed IDs or `none`.
- Cited IDs: write observed IDs or `none`.
- Next calls: write observed tools and IDs.
- Progressive disclosure: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Attribution boundary: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Coverage gaps not cited: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- No Discovery-triggered upload/capture: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Result: write `pass`, `fail`, or `review-needed`.

### filter_excluded_shared_knowledge_boundary

- Actual coverage: write the observed status and confidence.
- Action policy: write observed authority, load, cite, and claim values.
- Recall: write observed semantic status and reason.
- Coverage gaps: write observed gap count and source values.
- Discovered IDs: write observed IDs.
- Loaded IDs: write observed IDs or `none`.
- Cited IDs: write observed IDs or `none`.
- Next calls: write observed tools and IDs.
- Progressive disclosure: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Attribution boundary: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Coverage gaps not cited: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- No Discovery-triggered upload/capture: write `pass`, `fail`, or `not-applicable` with one sentence of evidence.
- Result: write `pass`, `fail`, or `review-needed`.

## Summary

- Pass count: write the observed number.
- Review-needed count: write the observed number.
- Fail count: write the observed number.
- Follow-up: write concrete follow-up items or `none`.
```

Fill every field with observed runner output. Do not leave the instructional text from this template in the final report.

- [ ] **Step 4: Commit Round 5 report**

Run:

```bash
git add docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-5.md
git commit -m "docs: record discovery dogfood round 5"
```

## Task 6: Final Verification And Residual Old-Term Scan

**Files:**
- Read-only verification.

- [ ] **Step 1: Run full tests**

Run:

```bash
go test ./... -count=1
```

Expected: all packages pass.

- [ ] **Step 2: Scan production code and active templates for old semantics**

Run:

```bash
rg -n "gap_candidates|GapCandidate|capture_candidate|candidate_only|proposal_required" internal testdata docs/superpowers/templates
```

Expected: no matches.

Then run:

```bash
rg -n "gap_candidates|GapCandidate|capture_candidate|candidate_only|proposal_required" docs/superpowers/specs docs/superpowers/plans docs/superpowers/reports
```

Expected: matches only in historical specs/plans/reports or supersession notes, not in the new active spec or Round 5 report.

- [ ] **Step 3: Check worktree**

Run:

```bash
git status --short
```

Expected: clean worktree.

- [ ] **Step 4: Summarize completion**

Final response should include:

- commits made;
- test command and result;
- old-term scan result;
- Round 5 dogfood outcome;
- next recommended step: global naming audit plan.
