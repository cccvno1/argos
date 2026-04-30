# Argos Discovery Recall And Gap Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add structured recall state and gap candidates to Discovery so agents can distinguish semantic recall capability from real knowledge gaps without treating gaps as official knowledge.

**Architecture:** Keep the first implementation slice light: add response structs in `internal/query`, default semantic recall to disabled, derive `gap_candidates` from existing missing knowledge hints, and preserve the old `coverage.missing_knowledge_hints` field for compatibility. CLI and MCP inherit JSON serialization, adapters teach agents how to treat gap candidates, and dogfood verifies that agents do not load/cite gap candidates or start capture without approval.

**Tech Stack:** Go, JSON response structs, existing query/CLI/MCP/adapters packages, golden discovery fixtures, Markdown dogfood reports.

---

## Scope

This plan implements the first slice recommended by `docs/superpowers/specs/2026-04-30-argos-discovery-recall-and-gap-workflow-design.md`:

- structured `RecallState` with semantic disabled default;
- structured `GapCandidate` values generated from existing missing hints;
- CLI/MCP/golden/adapter/dogfood coverage;
- no concrete semantic provider;
- no new CLI command;
- no automatic inbox writes or promotion.

Provider interface, stale/unavailable provider behavior, and concrete semantic implementations are intentionally left for a later plan.

## File Structure

- Modify `internal/query/query.go`: add `RecallState`, `SemanticRecallState`, `GapCandidate`, response fields, and helper functions.
- Modify `internal/query/query_test.go`: add focused tests for semantic disabled default, gap candidates, and strong no-gap behavior.
- Modify `internal/discoverytest/golden.go`: add expected gap candidate fields to golden cases.
- Modify `testdata/discovery-golden/cases.json`: add representative gap expectations.
- Modify `internal/query/discovery_golden_test.go`: assert golden recall/gap behavior.
- Modify `internal/cli/discovery_golden_test.go`: assert CLI JSON includes recall and gap candidates.
- Modify `internal/mcp/server_test.go`: assert MCP JSON includes recall and gap candidates.
- Modify `internal/adapters/adapters_test.go`: assert gap candidate adapter guidance.
- Modify `internal/adapters/adapters.go`: render gap candidate guidance.
- Modify `docs/superpowers/templates/argos-discovery-dogfood-report.md`: add recall and gap candidate report fields.
- Add `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md`: record dogfood results.

## Task 1: Add Recall State And Gap Candidates To Query Responses

**Files:**
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/query.go`

- [ ] **Step 1: Write failing tests for recall and gap behavior**

Add these tests near the action policy tests in `internal/query/query_test.go`:

```go
func TestDiscoverRecallStateDefaultsSemanticDisabled(t *testing.T) {
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
	if result.Recall.Semantic.Status != "disabled" {
		t.Fatalf("expected semantic recall disabled, got %#v", result.Recall)
	}
	if result.Recall.Semantic.Reason == "" {
		t.Fatalf("expected semantic recall reason, got %#v", result.Recall)
	}
}
```

Add this test:

```go
func TestDiscoverGapCandidatesForNoneCoverage(t *testing.T) {
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
	assertGapCandidateKinds(t, result.GapCandidates, []string{"standard", "decision", "lesson"})
	for _, candidate := range result.GapCandidates {
		if candidate.NextAction != "capture_candidate" {
			t.Fatalf("expected capture_candidate next action, got %#v", candidate)
		}
		if candidate.CaptureMode != "proposal_required" {
			t.Fatalf("expected proposal_required capture mode, got %#v", candidate)
		}
		if candidate.Authority != "candidate_only" {
			t.Fatalf("expected candidate_only authority, got %#v", candidate)
		}
	}
}
```

Add this test:

```go
func TestDiscoverStrongCoverageOmitsGapCandidates(t *testing.T) {
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
	if len(result.GapCandidates) != 0 {
		t.Fatalf("strong coverage should not produce gap candidates: %#v", result.GapCandidates)
	}
}
```

Add this helper:

```go
func assertGapCandidateKinds(t *testing.T, got []GapCandidate, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected gap candidate kinds %v, got %#v", want, got)
	}
	seen := map[string]bool{}
	for _, candidate := range got {
		seen[candidate.Kind] = true
		if candidate.SuggestedTitle == "" {
			t.Fatalf("expected suggested title for %#v", candidate)
		}
		if candidate.Reason == "" {
			t.Fatalf("expected reason for %#v", candidate)
		}
	}
	for _, kind := range want {
		if !seen[kind] {
			t.Fatalf("expected gap candidate kind %q in %#v", kind, got)
		}
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/query -run 'TestDiscoverRecallStateDefaultsSemanticDisabled|TestDiscoverGapCandidatesForNoneCoverage|TestDiscoverStrongCoverageOmitsGapCandidates' -count=1
```

Expected: FAIL to compile because `Recall`, `GapCandidates`, and `GapCandidate` do not exist.

- [ ] **Step 3: Add response structs and fields**

In `internal/query/query.go`, add these structs after `ActionPolicy`:

```go
type RecallState struct {
	Semantic SemanticRecallState `json:"semantic"`
}

type SemanticRecallState struct {
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type GapCandidate struct {
	Kind           string `json:"kind"`
	SuggestedTitle string `json:"suggested_title"`
	Reason         string `json:"reason"`
	Source         string `json:"source"`
	NextAction     string `json:"next_action"`
	CaptureMode    string `json:"capture_mode"`
	Authority      string `json:"authority"`
}
```

Add fields to `DiscoveryResponse`:

```go
	Recall        RecallState       `json:"recall"`
	GapCandidates []GapCandidate    `json:"gap_candidates,omitempty"`
```

Place them after `ActionPolicy` so the response order reads as policy, recall, gaps, items.

- [ ] **Step 4: Add helper functions**

Add these helpers near `missingKnowledgeHints` in `internal/query/query.go`:

```go
func defaultRecallState() RecallState {
	return RecallState{
		Semantic: SemanticRecallState{
			Status: "disabled",
			Reason: "semantic provider is not configured",
		},
	}
}

func gapCandidatesForCoverage(coverage Coverage) []GapCandidate {
	if coverage.Status == "strong" || len(coverage.MissingKnowledgeHints) == 0 {
		return nil
	}
	kinds := []string{"standard", "decision", "lesson"}
	candidates := make([]GapCandidate, 0, len(kinds))
	for i, hint := range coverage.MissingKnowledgeHints {
		kind := "reference"
		if i < len(kinds) {
			kind = kinds[i]
		}
		candidates = append(candidates, GapCandidate{
			Kind:           kind,
			SuggestedTitle: titleFromGapHint(hint, kind),
			Reason:         "No official Argos " + kind + " knowledge matched: " + hint,
			Source:         "missing_knowledge_hint",
			NextAction:     "capture_candidate",
			CaptureMode:    "proposal_required",
			Authority:      "candidate_only",
		})
	}
	return candidates
}

func titleFromGapHint(hint string, kind string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return "Missing " + kind + " knowledge"
	}
	return strings.TrimSpace(strings.TrimSuffix(hint, " "+kind)) + " " + kind
}
```

- [ ] **Step 5: Populate response fields**

In `Discover`, after computing `coverage` and `nextCalls`, return:

```go
return DiscoveryResponse{
	Project:       req.Project,
	Phase:         req.Phase,
	Query:         intent,
	Capabilities:  caps,
	Coverage:      coverage,
	ActionPolicy:  discoveryActionPolicy(coverage),
	Recall:        defaultRecallState(),
	GapCandidates: gapCandidatesForCoverage(coverage),
	Items:         results,
	NextCalls:     nextCalls,
}, nil
```

- [ ] **Step 6: Run focused query tests and commit**

Run:

```bash
gofmt -w internal/query/query.go internal/query/query_test.go
go test ./internal/query -run 'TestDiscoverRecallStateDefaultsSemanticDisabled|TestDiscoverGapCandidatesForNoneCoverage|TestDiscoverStrongCoverageOmitsGapCandidates' -count=1
go test ./internal/query -count=1
```

Expected: PASS.

Commit:

```bash
git add internal/query/query.go internal/query/query_test.go
git commit -m "feat: add discovery recall and gap candidates"
```

## Task 2: Extend Golden, CLI, And MCP Coverage

**Files:**
- Modify: `internal/discoverytest/golden.go`
- Modify: `testdata/discovery-golden/cases.json`
- Modify: `internal/query/discovery_golden_test.go`
- Modify: `internal/cli/discovery_golden_test.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Add expected gap fields**

In `internal/discoverytest/golden.go`, add these fields to `Expected`:

```go
	RecallSemanticStatus string   `json:"recall_semantic_status"`
	GapCandidateKinds    []string `json:"gap_candidate_kinds"`
```

In `testdata/discovery-golden/cases.json`, add:

```json
"recall_semantic_status": "disabled"
```

to every discover/workflow/cli-discover case expected block.

Add representative gap expectations:

- partial cases: `"gap_candidate_kinds": ["standard", "decision", "lesson"]`
- weak cases: `"gap_candidate_kinds": ["standard", "decision", "lesson"]`
- none cases: `"gap_candidate_kinds": ["standard", "decision", "lesson"]`
- strong cases: omit `gap_candidate_kinds`

- [ ] **Step 2: Assert golden recall and gap behavior**

In `internal/query/discovery_golden_test.go`, after action policy assertion for discovery cases, add:

```go
assertRecallMatchesExpected(t, result.Recall, tc.Expected)
assertGapCandidatesMatchExpected(t, result.GapCandidates, tc.Expected.GapCandidateKinds)
```

Add helpers:

```go
func assertRecallMatchesExpected(t *testing.T, got RecallState, expected discoverytest.Expected) {
	t.Helper()
	if expected.RecallSemanticStatus == "" {
		return
	}
	if got.Semantic.Status != expected.RecallSemanticStatus {
		t.Fatalf("expected semantic recall status %q, got %#v", expected.RecallSemanticStatus, got)
	}
}

func assertGapCandidatesMatchExpected(t *testing.T, got []GapCandidate, want []string) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("expected no gap candidates, got %#v", got)
		}
		return
	}
	assertGapCandidateKinds(t, got, want)
}
```

- [ ] **Step 3: Run golden tests**

Run:

```bash
gofmt -w internal/discoverytest/golden.go internal/query/discovery_golden_test.go
go test ./internal/discoverytest ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 4: Assert CLI JSON includes recall and gaps**

In `internal/cli/discovery_golden_test.go`, add to `TestGoldenCLIDiscoverMatchesQueryBehavior`:

```go
if result.Recall.Semantic.Status != tc.Expected.RecallSemanticStatus {
	t.Fatalf("expected semantic recall status %q, got %#v", tc.Expected.RecallSemanticStatus, result.Recall)
}
```

Add a new CLI test:

```go
func TestGoldenCLIDiscoverNoneIncludesGapCandidates(t *testing.T) {
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
	if len(result.GapCandidates) != len(tc.Expected.GapCandidateKinds) {
		t.Fatalf("expected gap candidate kinds %v, got %#v", tc.Expected.GapCandidateKinds, result.GapCandidates)
	}
}
```

Run:

```bash
gofmt -w internal/cli/discovery_golden_test.go
go test ./internal/cli -run 'TestGoldenCLIDiscoverMatchesQueryBehavior|TestGoldenCLIDiscoverNoneIncludesGapCandidates' -count=1
```

Expected: PASS.

- [ ] **Step 5: Assert MCP JSON includes recall and gaps**

In `internal/mcp/server_test.go`, update `TestToolCallArgosDiscoverReturnsRoutes`:

```go
if !strings.Contains(text, `"recall"`) || !strings.Contains(text, `"semantic"`) {
	t.Fatalf("expected recall state in discover response: %s", text)
}
```

Add a new MCP test:

```go
func TestToolCallArgosDiscoverNoneReturnsGapCandidates(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","phase":"implementation","task":"add payment webhook signature verification","query":"payment webhook signature","limit":5}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
	if !strings.Contains(text, `"gap_candidates"`) || !strings.Contains(text, `"authority": "candidate_only"`) {
		t.Fatalf("expected gap candidates in discover response: %s", text)
	}
}
```

Run:

```bash
gofmt -w internal/mcp/server_test.go
go test ./internal/mcp -run 'TestToolCallArgosDiscoverReturnsRoutes|TestToolCallArgosDiscoverNoneReturnsGapCandidates' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run interface tests and commit**

Run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp -count=1
```

Expected: PASS.

Commit:

```bash
git add internal/discoverytest/golden.go testdata/discovery-golden/cases.json internal/query/discovery_golden_test.go internal/cli/discovery_golden_test.go internal/mcp/server_test.go
git commit -m "test: assert discovery recall and gap interfaces"
```

## Task 3: Update Adapter And Dogfood Guidance

**Files:**
- Modify: `internal/adapters/adapters_test.go`
- Modify: `internal/adapters/adapters.go`
- Modify: `docs/superpowers/templates/argos-discovery-dogfood-report.md`

- [ ] **Step 1: Write failing adapter assertions**

In `internal/adapters/adapters_test.go`, add these expected strings to `TestRenderedAdaptersIncludeStableKnowledgeContract`:

```go
"Treat gap_candidates as candidate prompts, not official knowledge.",
"Do not cite gap_candidates.",
"Start capture-knowledge only with user approval.",
"Semantic recall never overrides action_policy.",
```

Run:

```bash
go test ./internal/adapters -run TestRenderedAdaptersIncludeStableKnowledgeContract -count=1
```

Expected: FAIL because rendered adapters do not contain these strings yet.

- [ ] **Step 2: Render adapter guidance**

In `internal/adapters/adapters.go`, add these lines under `## Work Protocol`:

```text
13. Treat gap_candidates as candidate prompts, not official knowledge.
14. Do not cite gap_candidates.
15. Start capture-knowledge only with user approval.
16. Semantic recall never overrides action_policy.
```

- [ ] **Step 3: Update dogfood report template**

In `docs/superpowers/templates/argos-discovery-dogfood-report.md`, add to `## Observed Results`:

```md
- Recall:
- Gap candidates:
```

Add to `## Guards`:

```md
- Gap candidates not cited: `pass|fail|not-applicable`
- Capture permission gate: `pass|fail|not-applicable`
```

- [ ] **Step 4: Run adapter tests and commit**

Run:

```bash
gofmt -w internal/adapters/adapters.go internal/adapters/adapters_test.go
go test ./internal/adapters -count=1
rg -n "gap_candidates|capture-knowledge|Semantic recall|Gap candidates" internal/adapters docs/superpowers/templates/argos-discovery-dogfood-report.md
```

Expected: PASS, and `rg` finds the new guidance.

Commit:

```bash
git add internal/adapters/adapters.go internal/adapters/adapters_test.go docs/superpowers/templates/argos-discovery-dogfood-report.md
git commit -m "docs: teach adapters discovery gap workflow"
```

## Task 4: Run Round 4 Dogfood And Record Results

**Files:**
- Add: `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md`

- [ ] **Step 1: Build dogfood binary and fixture workspace**

Run:

```bash
go build -o /tmp/argos-dogfood ./cmd/argos
full=$(mktemp -d /tmp/argos-round4-full-XXXXXX)
cp -R testdata/discovery-golden/knowledge "$full/knowledge"
(cd "$full" && /tmp/argos-dogfood index)
printf 'FULL=%s\n' "$full"
```

Expected: `indexed 9 knowledge item(s)`.

- [ ] **Step 2: Run fresh runner cases**

Run fresh runner sessions with `fork_context:false`, one per case:

- `semantic_disabled_strong_still_works`: use `strong_auth_refresh_full_signal`
- `semantic_disabled_none_has_gap_candidates`: use `none_payment_webhook`
- `weak_gap_does_not_load_or_cite`: use `weak_single_generic_term`
- `partial_gap_mentions_gap_before_capture`: use `partial_lesson_without_rule`
- `filter_excluded_match_produces_filter_gap`: use `none_explicit_filter_excludes_match`
- `agent_requests_capture_permission`: use `none_payment_webhook`, with instruction to decide whether capture should be offered

Each runner receives only:

- case ID
- fixture workspace path
- `/tmp/argos-dogfood`
- case input
- allowed flow
- requirement to follow `action_policy`, `recall`, and `gap_candidates`
- report template path

Each runner must not receive golden expected values, previous transcripts, or dogfood reports.

- [ ] **Step 3: Evaluate dogfood results**

Use these pass criteria:

- semantic disabled is treated as a capability fact, not an error;
- strong still loads and cites normally;
- gap candidates are not loaded or cited;
- weak/none still obey `action_policy`;
- agents do not invent official knowledge from gaps;
- agents ask before invoking `capture-knowledge`;
- filter-excluded matches do not bypass filters.

- [ ] **Step 4: Write Round 4 report**

Create `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md`:

```md
# Argos Discovery Dogfood Round 4

Date: 2026-04-30

## Scope

Recall and gap workflow validation.

## Result

Overall: `pass|fail|review-needed`

## Findings

### Semantic Disabled

### Gap Candidates

### Weak/None Safety

### Capture Permission Gate

### Filter Respect

## Decision

State whether recall/gap first slice is ready and whether concrete semantic provider design can be considered next.
```

- [ ] **Step 5: Run verification and commit**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

Commit:

```bash
git add docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-4.md
git commit -m "docs: record discovery dogfood round 4"
```

## Task 5: Final Verification

**Files:**
- No new file changes expected.

- [ ] **Step 1: Run full verification**

Run:

```bash
go test ./... -count=1
git status --short
```

Expected:

- all packages pass;
- `git status --short` is empty.

- [ ] **Step 2: Summarize final state**

Report:

- commits created;
- tests run;
- Round 4 result;
- whether the first recall/gap slice is ready;
- what remains before a concrete semantic provider is worth designing.

## Self-Review

Spec coverage:

- Structured recall state is implemented in Task 1.
- Gap candidates are implemented in Tasks 1 and 2.
- Compatibility with `missing_knowledge_hints` is preserved by deriving gaps from the existing field without removing it.
- CLI/MCP serialization is covered in Task 2.
- Adapter guidance is covered in Task 3.
- Dogfood is covered in Task 4.
- Concrete semantic provider implementation is intentionally excluded from this first slice.

Placeholder scan: no unfinished placeholders are intentionally left in this plan.

Type consistency: the plan uses `RecallState`, `SemanticRecallState`, `GapCandidate`, JSON `recall`, and JSON `gap_candidates` consistently across query, CLI, MCP, adapters, and dogfood.
