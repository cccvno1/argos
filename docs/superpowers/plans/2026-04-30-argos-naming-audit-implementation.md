# Argos Naming Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete one coherent naming migration for Argos shared-knowledge discovery so public surfaces read as list/find/read/cite, support, missing_needs, usage, search_status, read_status, and next_steps.

**Architecture:** Keep storage, ranking, validation, indexing, and citation behavior unchanged. Start by adding an active-surface retired-name guard, then migrate query/golden models, MCP tools, CLI commands, adapters, README, active specs, templates, and checklists. Old development-stage names are removed from active public surfaces; historical plans and reports may keep them as record.

**Tech Stack:** Go, JSON golden fixtures, MCP JSON-RPC over stdio, internal CLI, Markdown docs/templates/checklists, `rg`-backed retired-name verification.

---

## File Structure

- Modify `internal/query/query.go`: rename request/response types, JSON fields, helper functions, returned tool names, read status, missing-need source values, and user-facing reason strings.
- Modify `internal/query/query_test.go`: update unit tests to new names and add JSON negative assertions.
- Modify `internal/query/discovery_golden_test.go`: update golden assertions to support/usage/search/read naming.
- Modify `internal/discoverytest/golden.go`: rename expected fixture fields from coverage/action/recall/next-call terms to support/usage/search/next-step terms.
- Modify `internal/discoverytest/golden_test.go`: update fixture sanity tests and renamed case IDs.
- Modify `testdata/discovery-golden/cases.json`: rename operations, expected fields, old tool names, and old missing-need source values.
- Modify `internal/mcp/server.go`: expose new MCP tools and reject old tool names.
- Modify `internal/mcp/server_test.go`: update schemas/calls/errors and add old-tool rejection tests.
- Modify `internal/cli/cli.go`: replace `discover` and `map` with `knowledge find/list/read/cite`.
- Modify `internal/cli/cli_test.go` and `internal/cli/discovery_golden_test.go`: update command tests and JSON shape assertions.
- Modify `cmd/argos/e2e_test.go`: update end-to-end command expectations.
- Modify `internal/adapters/adapters.go`: teach list/find/read/cite and new JSON names.
- Modify `internal/adapters/adapters_test.go`: assert new adapter text and retired-name absence.
- Modify `README.md`: update Discovery, MCP, and Agent/Internal Commands sections to the new vocabulary.
- Modify `docs/superpowers/templates/argos-discovery-dogfood-report.md`: rename report labels to support/missing-needs/search/read language.
- Modify `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`: rename dogfood workflow and tool names.
- Modify active specs:
  - `docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-naming-audit-design.md`

Historical implementation plans and reports under `docs/superpowers/plans/` and `docs/superpowers/reports/` may retain old terminology except this plan and active templates/checklists/specs.

---

### Task 0: Add Retired-Name Guard First

**Files:**
- Modify: `internal/query/query_test.go`

- [ ] **Step 1: Add an active-surface retired-name test**

Add this test to `internal/query/query_test.go`:

```go
func TestActiveSurfacesDoNotUseRetiredSharedKnowledgeNames(t *testing.T) {
	retired := []string{
		"argos_map",
		"argos_discover",
		"get_knowledge_item",
		"coverage_gaps",
		"CoverageGap",
		"action_policy",
		"ActionPolicy",
		"authority",
		"recall",
		"RecallState",
		"disclosure",
		"Disclosure",
		"next_calls",
		"RecommendedCall",
	}
	root := repoRootForActiveSurfaceTest(t)
	activeRoots := []string{
		"README.md",
		"internal",
		"testdata",
		"docs/superpowers/templates",
		"docs/superpowers/checklists",
		"docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md",
		"docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md",
		"docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md",
	}

	for _, rel := range activeRoots {
		body := readActiveSurface(t, filepath.Join(root, rel))
		for _, term := range retired {
			if strings.Contains(body, term) {
				t.Fatalf("active surface %s contains retired term %q", rel, term)
			}
		}
	}
}

func repoRootForActiveSurfaceTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readActiveSurface(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat active surface %s: %v", path, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read active surface %s: %v", path, err)
		}
		return string(data)
	}
	var builder strings.Builder
	err = filepath.WalkDir(path, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(file, ".md") && !strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, ".json") {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		builder.WriteString("\n")
		builder.Write(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk active surface %s: %v", path, err)
	}
	return builder.String()
}
```

Also add imports used by the test:

```go
import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"argos/internal/discoverytest"
)
```

- [ ] **Step 2: Run the guard and confirm it fails before migration**

Run:

```bash
go test ./internal/query -run TestActiveSurfacesDoNotUseRetiredSharedKnowledgeNames -count=1
```

Expected: FAIL with at least one retired term such as `argos_discover`, `coverage_gaps`, or `action_policy`.

- [ ] **Step 3: Commit the failing guard**

```bash
git add internal/query/query_test.go
git commit -m "test: guard retired shared knowledge names"
```

---

### Task 1: Rename Query Model And Golden Vocabulary

**Files:**
- Modify: `internal/query/query.go`
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/discovery_golden_test.go`
- Modify: `internal/discoverytest/golden.go`
- Modify: `internal/discoverytest/golden_test.go`
- Modify: `testdata/discovery-golden/cases.json`

- [ ] **Step 1: Rename golden fixture fields and operations**

In `testdata/discovery-golden/cases.json`, apply these JSON key/value migrations:

```text
"operation": "map" -> "operation": "list"
"operation": "map-empty" -> "operation": "list-empty"
"operation": "discover" -> "operation": "find"
"operation": "cli-map" -> "operation": "cli-list"
"operation": "cli-discover" -> "operation": "cli-find"
"coverage" -> "support"
"action_authority" -> "support_level"
"action_load" -> "usage_read"
"action_cite" -> "usage_cite"
"action_claim" -> "usage_claim"
"recall_semantic_status" -> "search_semantic_status"
"coverage_gap_sources" -> "missing_need_sources"
"require_next_call_tools" -> "require_next_step_tools"
"forbid_next_call_tools" -> "forbid_next_step_tools"
```

Rename case IDs that contain old operation names:

```text
map_inventory_normal -> list_inventory_normal
map_inventory_empty -> list_inventory_empty
map_hides_deprecated_by_default -> list_hides_deprecated_by_default
map_global_knowledge_visible -> list_global_knowledge_visible
interface_cli_discover_matches_query -> interface_cli_find_matches_query
interface_cli_map_matches_query -> interface_cli_list_matches_query
```

Replace expected tool names in fixture arrays:

```text
get_knowledge_item -> argos_read_knowledge
cite_knowledge -> argos_cite_knowledge
```

Replace missing-need source values:

```text
unmatched_intent -> not_found
filter_excluded -> filtered_out
conflicting_shared_knowledge -> conflict
cross_domain_mismatch -> wrong_scope
```

- [ ] **Step 2: Update fixture structs**

In `internal/discoverytest/golden.go`, update `Expected` to:

```go
type Expected struct {
	Support                         string         `json:"support"`
	SupportLevel                    string         `json:"support_level"`
	UsageRead                       string         `json:"usage_read"`
	UsageCite                       string         `json:"usage_cite"`
	UsageClaim                      string         `json:"usage_claim"`
	SearchSemanticStatus            string         `json:"search_semantic_status"`
	MissingNeedSources              []string       `json:"missing_need_sources"`
	TopID                           string         `json:"top_id"`
	IncludeIDs                      []string       `json:"include_ids"`
	ExcludeIDs                      []string       `json:"exclude_ids"`
	LoadIDs                         []string       `json:"load_ids"`
	CiteIDs                         []string       `json:"cite_ids"`
	IncludeDomains                  []string       `json:"include_domains"`
	IncludeTags                     []string       `json:"include_tags"`
	IncludeText                     []string       `json:"include_text"`
	ExcludeText                     []string       `json:"exclude_text"`
	InventoryTypesMin               map[string]int `json:"inventory_types_min"`
	IncludeDeprecatedIDWhenRequested string        `json:"include_deprecated_id_when_requested"`
	RequireNextStepTools            []string       `json:"require_next_step_tools"`
	ForbidNextStepTools             []string       `json:"forbid_next_step_tools"`
	WhyContains                     []string       `json:"why_contains"`
	NoBodies                        bool           `json:"no_bodies"`
	GroupsEmpty                     bool           `json:"groups_empty"`
	ItemsEmpty                      bool           `json:"items_empty"`
	RequireMissingHints             bool           `json:"require_missing_hints"`
	RejectUnknownArguments          bool           `json:"reject_unknown_arguments"`
	RejectMissingTaskAndQuery       bool           `json:"reject_missing_task_and_query"`
	RejectOutOfRangeLimit           bool           `json:"reject_out_of_range_limit"`
}
```

- [ ] **Step 3: Rename query public types and JSON fields**

In `internal/query/query.go`, rename:

```text
DiscoverRequest -> FindKnowledgeRequest
MapRequest -> ListKnowledgeRequest
DiscoveryResponse -> FindKnowledgeResponse
MapResponse -> ListKnowledgeResponse
Coverage -> Support
ActionPolicy -> UsageGuidance
RecallState -> SearchStatus
SemanticRecallState -> SemanticSearchStatus
CoverageGap -> MissingNeed
DiscoveryItem -> KnowledgeSummary
Disclosure -> ReadStatus
RecommendedCall -> NextStep
```

Use these final response shapes:

```go
type FindKnowledgeResponse struct {
	Project      string                      `json:"project"`
	Phase        string                      `json:"phase"`
	Query        string                      `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Support      Support                     `json:"support"`
	Usage        UsageGuidance               `json:"usage"`
	SearchStatus SearchStatus                `json:"search_status"`
	MissingNeeds []MissingNeed               `json:"missing_needs,omitempty"`
	Items        []KnowledgeSummary          `json:"items"`
	NextSteps    []NextStep                  `json:"next_steps"`
}

type ListKnowledgeResponse struct {
	Project   string        `json:"project"`
	Usage     UsageGuidance `json:"usage"`
	Inventory Inventory     `json:"inventory"`
	Groups    []ListGroup   `json:"groups"`
}

type Support struct {
	Level                 string   `json:"level"`
	Confidence            float64  `json:"confidence"`
	Reason                string   `json:"reason"`
	Recommendation        string   `json:"recommendation"`
	MissingKnowledgeHints  []string `json:"missing_knowledge_hints,omitempty"`
}

type UsageGuidance struct {
	Read   string `json:"read"`
	Cite   string `json:"cite"`
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}

type SearchStatus struct {
	Semantic SemanticSearchStatus `json:"semantic"`
}

type SemanticSearchStatus struct {
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type MissingNeed struct {
	Need        string `json:"need"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	Severity    string `json:"severity"`
	ArgosBacked bool   `json:"argos_backed"`
}

type KnowledgeSummary struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Status          string          `json:"status"`
	Priority        string          `json:"priority"`
	Path            string          `json:"path"`
	Score           float64         `json:"score"`
	ScoreComponents ScoreComponents `json:"score_components"`
	WhyMatched      []string        `json:"why_matched"`
	MatchedSections []string        `json:"matched_sections"`
	ReadStatus      ReadStatus      `json:"read_status"`
	RecommendedStep string          `json:"recommended_step"`
	Body            string          `json:"-"`
}

type ReadStatus struct {
	Level             string `json:"level"`
	FullBodyAvailable bool   `json:"full_body_available"`
	ReadTool          string `json:"read_tool"`
}

type NextStep struct {
	Tool   string   `json:"tool"`
	Reason string   `json:"reason"`
	IDs    []string `json:"ids,omitempty"`
}
```

Remove the old `Authority` field. Use `Support.Level` and fixture `SupportLevel` for the same behavioral signal.

- [ ] **Step 4: Rename query service methods and helpers**

Rename public methods:

```text
Discover -> FindKnowledge
Map -> ListKnowledge
GetKnowledgeItem -> ReadKnowledge
CiteKnowledge -> CiteKnowledge
```

Rename helpers mechanically:

```text
discoveryCoverage -> findSupport
discoveryActionPolicy -> usageGuidance
defaultRecallState -> defaultSearchStatus
coverageGapsForCoverage -> missingNeedsForSupport
coverageGapNeed -> missingNeedText
coverageGapSource -> missingNeedSource
coverageGapSeverity -> missingNeedSeverity
coverageGapReason -> missingNeedReason
discoveryNextCalls -> findNextSteps
discoveryItem -> knowledgeSummary
discoveryItemFromKnowledge -> knowledgeSummaryFromKnowledge
discoverCandidateAllowed -> findCandidateAllowed
mapCandidateAllowed -> listCandidateAllowed
mapActionPolicy -> listUsageGuidance
mapGroupKey -> listGroupKey
```

Every returned `NextStep.Tool` must use:

```text
argos_list_knowledge
argos_find_knowledge
argos_read_knowledge
argos_cite_knowledge
```

- [ ] **Step 5: Update golden query assertions**

In `internal/query/discovery_golden_test.go`, use:

```go
result, err := service.FindKnowledge(FindKnowledgeRequest{...})
assertSupport(t, result.Support, tc.Expected.Support, result.Items)
assertUsageMatchesExpected(t, result.Usage, tc.Expected)
assertSearchStatusMatchesExpected(t, result.SearchStatus, tc.Expected)
assertMissingNeedsMatchExpected(t, result.MissingNeeds, tc.Expected.MissingNeedSources)
assertKnowledgeIDs(t, result.Items, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
assertNoKnowledgeBodies(t, result.Items, tc.Expected.NoBodies)
assertNextSteps(t, result.NextSteps, tc.Expected.RequireNextStepTools, tc.Expected.ForbidNextStepTools)
```

Use `service.ListKnowledge(ListKnowledgeRequest{...})` for inventory cases and `service.ReadKnowledge(id)` for full-body reads.

- [ ] **Step 6: Run query and fixture tests**

Run:

```bash
go test ./internal/discoverytest ./internal/query -count=1
```

Expected: PASS except the Task 0 guard may still fail because MCP, CLI, adapters, and docs are not migrated yet. If only the retired-name guard fails, continue to Task 2.

- [ ] **Step 7: Commit query and golden migration**

```bash
git add internal/query internal/discoverytest testdata/discovery-golden/cases.json
git commit -m "refactor: rename shared knowledge query model"
```

---

### Task 2: Rename MCP Tool Surface

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/query/query.go`

- [ ] **Step 1: Rename MCP handler cases**

In `internal/mcp/server.go`, replace handler cases:

```text
argos_discover -> argos_find_knowledge
argos_map -> argos_list_knowledge
get_knowledge_item -> argos_read_knowledge
cite_knowledge -> argos_cite_knowledge
```

Call the renamed query methods:

```go
result, err := s.service.FindKnowledge(req)
result, err := s.service.ListKnowledge(req)
result, err := s.service.ReadKnowledge(req.ID)
result := s.service.CiteKnowledge(req.IDs)
```

Use new error prefixes:

```text
invalid arguments for argos_find_knowledge
invalid arguments for argos_list_knowledge
invalid arguments for argos_read_knowledge
invalid arguments for argos_cite_knowledge
```

- [ ] **Step 2: Rename MCP schemas**

In `tools()`, expose these names:

```go
"argos_context"
"argos_standards"
"argos_find_knowledge"
"argos_list_knowledge"
"argos_read_knowledge"
"argos_cite_knowledge"
```

Rename `discoverInputSchema()` to `findKnowledgeInputSchema()`.

- [ ] **Step 3: Update MCP tests**

In `internal/mcp/server_test.go`, update `tools/list` expectations to:

```go
for _, name := range []string{
	"argos_context",
	"argos_standards",
	"argos_find_knowledge",
	"argos_list_knowledge",
	"argos_read_knowledge",
	"argos_cite_knowledge",
} {
	assertToolListed(t, result.Tools, name)
}
```

Update all `tools/call` JSON examples to the new names and decode find responses into `query.FindKnowledgeResponse`.

- [ ] **Step 4: Add old tool rejection tests**

Add:

```go
func TestRenamedMCPToolsRejectOldNames(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	server := NewServerWithStore(store)

	oldNames := []string{"argos_discover", "argos_map", "get_knowledge_item", "cite_knowledge"}
	for _, oldName := range oldNames {
		t.Run(oldName, func(t *testing.T) {
			var out bytes.Buffer
			line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + oldName + `","arguments":{"project":"mall-api","query":"auth","id":"rule:backend.auth.v1","ids":["rule:backend.auth.v1"]}}}`)
			if err := server.HandleLine(line, &out); err != nil {
				t.Fatalf("handle line: %v", err)
			}
			assertToolErrorContains(t, out.Bytes(), "unknown tool")
		})
	}
}
```

- [ ] **Step 5: Run MCP tests**

Run:

```bash
go test ./internal/mcp -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit MCP migration**

```bash
git add internal/mcp internal/query/query.go
git commit -m "feat: rename mcp shared knowledge tools"
```

---

### Task 3: Rename CLI Commands

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/discovery_golden_test.go`
- Modify: `cmd/argos/e2e_test.go`
- Modify: `testdata/discovery-golden/cases.json`

- [ ] **Step 1: Replace top-level discovery commands with grouped commands**

In `internal/cli/cli.go`, remove top-level `discover` and `map` dispatches. Add:

```go
case "knowledge":
	return runKnowledge(args[1:], stdout, stderr)
```

Implement:

```go
func runKnowledge(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "knowledge: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "find":
		return runKnowledgeFind(args[1:], stdout, stderr)
	case "list":
		return runKnowledgeList(args[1:], stdout, stderr)
	case "read":
		return runKnowledgeRead(args[1:], stdout, stderr)
	case "cite":
		return runKnowledgeCite(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "knowledge: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}
```

Move existing `discover` behavior into `runKnowledgeFind`. Move existing `map` behavior into `runKnowledgeList`.

- [ ] **Step 2: Add `knowledge read` and `knowledge cite`**

Implement `knowledge read`:

```go
func runKnowledgeRead(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge read", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge read: --json is required")
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "knowledge read: id is required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).ReadKnowledge(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "knowledge read: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}
```

Implement `knowledge cite`:

```go
func runKnowledgeCite(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge cite", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge cite: --json is required")
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "knowledge cite: at least one id is required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result := query.New(store).CiteKnowledge(flags.Args())
	return printJSON(stdout, stderr, result)
}
```

- [ ] **Step 3: Update CLI errors and usage text**

Use these error prefixes:

```text
knowledge find: --json is required
knowledge find: --project is required
knowledge find: --task or --query is required
knowledge find: --limit must be between 1 and 20
knowledge list: --json is required
knowledge list: --project is required
knowledge read: --json is required
knowledge read: id is required
knowledge cite: --json is required
knowledge cite: at least one id is required
```

Update usage examples to:

```text
argos knowledge list --json --project <project>
argos knowledge find --json --project <project> --task <task>
argos knowledge read --json <id>
argos knowledge cite --json <id>...
```

- [ ] **Step 4: Update CLI tests and golden CLI cases**

In `internal/cli/discovery_golden_test.go`, rename tests:

```text
TestGoldenCLIDiscoverMatchesQueryBehavior -> TestGoldenCLIKnowledgeFindMatchesQueryBehavior
TestGoldenCLIDiscoverNoneIncludesCoverageGaps -> TestGoldenCLIKnowledgeFindNoneIncludesMissingNeeds
TestGoldenCLIMapMatchesQueryBehavior -> TestGoldenCLIKnowledgeListMatchesQueryBehavior
```

Use command args:

```go
[]string{"knowledge", "find", "--json", "--project", ...}
[]string{"knowledge", "list", "--json", "--project", ...}
[]string{"knowledge", "read", "--json", "rule:backend.auth-refresh.v1"}
[]string{"knowledge", "cite", "--json", "rule:backend.auth-refresh.v1"}
```

Decode find output into `query.FindKnowledgeResponse`, list output into `query.ListKnowledgeResponse`, and assert `Support.Level`, `Usage.Read`, `SearchStatus.Semantic.Status`, `MissingNeeds`, and `NextSteps`.

- [ ] **Step 5: Run CLI and E2E tests**

Run:

```bash
go test ./internal/cli ./cmd/argos -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit CLI migration**

```bash
git add internal/cli cmd/argos testdata/discovery-golden/cases.json
git commit -m "feat: group shared knowledge cli commands"
```

---

### Task 4: Update Adapters And Active Guidance Docs

**Files:**
- Modify: `internal/adapters/adapters.go`
- Modify: `internal/adapters/adapters_test.go`
- Modify: `README.md`
- Modify: `docs/superpowers/templates/argos-discovery-dogfood-report.md`
- Modify: `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- Modify: active specs listed in File Structure

- [ ] **Step 1: Update adapter protocol text**

In `internal/adapters/adapters.go`, replace the Work Protocol with:

```text
1. Before substantial project work, call argos_context when available; otherwise follow equivalent adapter or Markdown guidance.
2. Use argos_list_knowledge for broad orientation before unfamiliar project work.
3. Use argos_find_knowledge to find shared knowledge for current work.
4. Before implementation or review, call argos_standards when available; otherwise follow equivalent adapter or Markdown guidance.
5. Follow next_steps returned by Argos.
6. Read full knowledge items only through argos_read_knowledge when routed to specific IDs or paths.
7. Cite Argos knowledge IDs used in final responses only after reading and applying them.
8. Do not cite IDs returned only by argos_list_knowledge or argos_find_knowledge.
9. Cite only knowledge IDs whose full item was read with argos_read_knowledge and actually applied.
10. Follow usage.read before reading full knowledge items.
11. Follow usage.cite before calling argos_cite_knowledge.
12. When missing_needs are present, separate Argos-backed claims from general reasoning.
13. Do not cite missing_needs; they are unsupported needs, not knowledge items.
14. Do not start upload, capture, or inbox creation from find results alone.
15. Semantic search status never overrides usage guidance.
```

- [ ] **Step 2: Update README commands and explanation**

In `README.md`, update examples to:

```bash
argos knowledge find --json --project <project> --phase <phase> --task "<task>" --query "<query>"
argos knowledge list --json --project <project> --domain <domain>
argos knowledge read --json <id>
argos knowledge cite --json <id>...
```

Update MCP tool list to:

```text
argos_find_knowledge
argos_list_knowledge
argos_read_knowledge
argos_cite_knowledge
```

Describe JSON fields as `support`, `usage`, `search_status`, `missing_needs`, and `next_steps`.

- [ ] **Step 3: Update active specs/templates/checklists**

In active docs, replace active guidance vocabulary:

```text
argos_map -> argos_list_knowledge
argos_discover -> argos_find_knowledge
get_knowledge_item -> argos_read_knowledge
cite_knowledge -> argos_cite_knowledge
coverage -> support
coverage_gaps -> missing_needs
action_policy -> usage
recall -> search_status
disclosure -> read_status
next_calls -> next_steps
recommended_action -> recommended_step
```

When docs mention source values, also replace:

```text
unmatched_intent -> not_found
filter_excluded -> filtered_out
conflicting_shared_knowledge -> conflict
cross_domain_mismatch -> wrong_scope
```

Do not edit historical dogfood reports except to add a superseded note if an active doc points readers there.

- [ ] **Step 4: Update adapter tests**

In `internal/adapters/adapters_test.go`, assert the new recommendations:

```go
required := []string{
	"Use argos_list_knowledge for broad orientation before unfamiliar project work.",
	"Use argos_find_knowledge to find shared knowledge for current work.",
	"Follow next_steps returned by Argos.",
	"Read full knowledge items only through argos_read_knowledge when routed to specific IDs or paths.",
	"Do not cite IDs returned only by argos_list_knowledge or argos_find_knowledge.",
	"Follow usage.read before reading full knowledge items.",
	"Follow usage.cite before calling argos_cite_knowledge.",
	"When missing_needs are present, separate Argos-backed claims from general reasoning.",
	"Do not cite missing_needs; they are unsupported needs, not knowledge items.",
	"Semantic search status never overrides usage guidance.",
}
```

Add retired-name negative assertions for adapter output:

```go
retired := []string{
	"argos_map",
	"argos_discover",
	"get_knowledge_item",
	"coverage_gaps",
	"action_policy",
	"Semantic recall",
}
```

- [ ] **Step 5: Run adapter tests and docs guard**

Run:

```bash
go test ./internal/adapters ./internal/query -run 'TestRenderedAdapters|TestActiveSurfacesDoNotUseRetiredSharedKnowledgeNames' -count=1
```

Expected: PASS after all active docs and adapter text are migrated.

- [ ] **Step 6: Commit docs and adapter migration**

```bash
git add internal/adapters README.md docs/superpowers/templates docs/superpowers/checklists docs/superpowers/specs
git commit -m "docs: teach renamed shared knowledge workflow"
```

---

### Task 5: Final Verification And Retired-Name Scan

**Files:**
- Modify if needed: `internal/query/query_test.go`
- Modify if needed: any active surface still failing the scan

- [ ] **Step 1: Run the full targeted suite**

Run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp ./internal/adapters ./cmd/argos -count=1
```

Expected: PASS.

- [ ] **Step 2: Run the complete repository suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run explicit retired-name scan**

Run:

```bash
rg -n "argos_map|argos_discover|get_knowledge_item|coverage_gaps|CoverageGap|action_policy|ActionPolicy|authority|recall|RecallState|disclosure|Disclosure|next_calls|RecommendedCall" README.md internal testdata docs/superpowers/templates docs/superpowers/checklists docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md
```

Expected: no matches.

- [ ] **Step 4: Verify new public vocabulary exists**

Run:

```bash
rg -n "argos_list_knowledge|argos_find_knowledge|argos_read_knowledge|argos_cite_knowledge|support|missing_needs|usage|search_status|read_status|next_steps" README.md internal testdata docs/superpowers/templates docs/superpowers/checklists docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md
```

Expected: matches in README, internal code/tests, fixtures, templates, checklists, and active specs.

- [ ] **Step 5: Commit final verification guards if changed**

If verification required additional guard or wording fixes:

```bash
git add README.md internal testdata docs/superpowers/templates docs/superpowers/checklists docs/superpowers/specs
git commit -m "test: verify renamed shared knowledge surfaces"
```

If no files changed, do not create an empty commit.

---

## Self-Review

### Spec Coverage

- Public MCP names `argos_list_knowledge`, `argos_find_knowledge`, `argos_read_knowledge`, `argos_cite_knowledge`: Tasks 1, 2, 4, and 5.
- CLI names `argos knowledge list/find/read/cite`: Task 3 and README updates in Task 4.
- JSON response names `support`, `missing_needs`, `usage`, `search_status`, `read_status`, `next_steps`: Tasks 1, 2, 3, 4, and 5.
- Internal Go names: Task 1.
- Adapter language: Task 4.
- Active guidance docs: Task 4.
- Retired-name guard before implementation and final scan after implementation: Tasks 0 and 5.
- Behavior unchanged: query, golden, CLI, MCP, adapter, E2E, and full repository tests run before completion.

### Type Consistency

The final names used by the plan are:

- `FindKnowledgeRequest`
- `FindKnowledgeResponse`
- `ListKnowledgeRequest`
- `ListKnowledgeResponse`
- `Support`
- `UsageGuidance`
- `SearchStatus`
- `SemanticSearchStatus`
- `MissingNeed`
- `KnowledgeSummary`
- `ReadStatus`
- `NextStep`
- `ListGroup`

The public tool and CLI names are:

- MCP: `argos_list_knowledge`, `argos_find_knowledge`, `argos_read_knowledge`, `argos_cite_knowledge`
- CLI: `argos knowledge list`, `argos knowledge find`, `argos knowledge read`, `argos knowledge cite`

### Scope Boundaries

This plan intentionally does not add vector search, semantic provider wiring, upload flows, capture automation, storage changes, ranking changes, or new validation rules unrelated to naming. It is a complete public-language migration for the existing shared-knowledge discovery path.
