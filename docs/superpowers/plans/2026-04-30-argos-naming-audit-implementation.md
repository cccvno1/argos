# Argos Naming Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename Argos shared-knowledge discovery surfaces to plain professional language: list/find/read/cite, support, missing_needs, usage, search_status, read_status, and next_steps.

**Architecture:** Keep the existing search, ranking, support-level, and citation behavior unchanged. This is a naming migration across query response types, MCP tools, CLI commands, golden fixtures, adapters, and active guidance documents. Because Argos is still in development, old development-stage names are removed from active public surfaces instead of preserved as long-term aliases.

**Tech Stack:** Go, MCP JSON-RPC over stdio, Cobra-free internal CLI, JSON golden fixtures, Markdown specs/templates/checklists.

---

## File Structure

- Modify `internal/query/query.go`: rename response types, JSON fields, helper functions, returned tool names, and user-facing reason strings.
- Modify `internal/query/query_test.go`: update unit tests to the new names and add active-surface negative checks.
- Modify `internal/query/discovery_golden_test.go`: update golden assertions from coverage/action policy/recall/disclosure naming to support/usage/search/read naming.
- Modify `internal/discoverytest/golden.go`: rename expected fixture fields from coverage-oriented names to support-oriented names.
- Modify `internal/discoverytest/golden_test.go`: update fixture sanity tests.
- Modify `testdata/discovery-golden/cases.json`: rename operations and expected fields.
- Modify `internal/mcp/server.go`: expose new MCP tool names and error messages.
- Modify `internal/mcp/server_test.go`: update schema/call/error tests and add rejection tests for old tool names.
- Modify `internal/cli/cli.go`: rename CLI commands to `knowledge list/find/read/cite` or, if keeping the current flat command style for one slice, at minimum rename `discover` to `find` and `map` to `list`.
- Modify `internal/cli/cli_test.go` and `internal/cli/discovery_golden_test.go`: update CLI tests.
- Modify `cmd/argos/e2e_test.go`: update any CLI help or command expectations.
- Modify `internal/adapters/adapters.go`: teach list/find/read/cite and the new JSON field names.
- Modify `internal/adapters/adapters_test.go`: update adapter output assertions and retired-name scans.
- Modify `docs/superpowers/templates/argos-discovery-dogfood-report.md`: rename report labels.
- Modify `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`: rename dogfood workflow.
- Modify active specs:
  - `docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md`
  - `docs/superpowers/specs/2026-04-30-argos-naming-audit-design.md`

---

### Task 1: Update Golden Fixture Vocabulary

**Files:**
- Modify: `testdata/discovery-golden/cases.json`
- Modify: `internal/discoverytest/golden.go`
- Modify: `internal/discoverytest/golden_test.go`

- [ ] **Step 1: Rename fixture expectation fields**

In `testdata/discovery-golden/cases.json`, rename only test metadata fields, not knowledge IDs or prose unless the prose is active guidance:

```json
"coverage": "strong"
```

becomes:

```json
"support": "strong"
```

```json
"coverage_gap_sources": ["partial_match"]
```

becomes:

```json
"missing_need_sources": ["partial_match"]
```

```json
"recall_semantic_status": "disabled"
```

becomes:

```json
"search_semantic_status": "disabled"
```

Rename operation values:

```json
"operation": "map"
```

to:

```json
"operation": "list"
```

```json
"operation": "discover"
```

to:

```json
"operation": "find"
```

```json
"operation": "cli-map"
```

to:

```json
"operation": "cli-list"
```

```json
"operation": "cli-discover"
```

to:

```json
"operation": "cli-find"
```

Keep case IDs unchanged in this first task unless they contain old tool names. For case IDs that do contain old names, rename:

```text
map_inventory_normal -> list_inventory_normal
map_inventory_empty -> list_inventory_empty
map_hides_deprecated_by_default -> list_hides_deprecated_by_default
map_global_knowledge_visible -> list_global_knowledge_visible
interface_cli_discover_matches_query -> interface_cli_find_matches_query
interface_cli_map_matches_query -> interface_cli_list_matches_query
```

- [ ] **Step 2: Update the fixture loader struct**

In `internal/discoverytest/golden.go`, change:

```go
type Expected struct {
	Coverage                         string         `json:"coverage"`
	TopID                            string         `json:"top_id"`
	IncludeIDs                       []string       `json:"include_ids"`
	ExcludeIDs                       []string       `json:"exclude_ids"`
	RequireNextCallTools             []string       `json:"require_next_call_tools"`
	ForbidNextCallTools              []string       `json:"forbid_next_call_tools"`
	RequireMissingHints              bool           `json:"require_missing_hints"`
	RequireActionClaim               string         `json:"require_action_claim"`
	RecallSemanticStatus             string         `json:"recall_semantic_status"`
	CoverageGapSources               []string       `json:"coverage_gap_sources"`
	NoBodies                         bool           `json:"no_bodies"`
	CiteIDs                          []string       `json:"cite_ids"`
	InventoryTypesMin                map[string]int `json:"inventory_types_min"`
	IncludeDeprecated                bool           `json:"include_deprecated"`
	RequireWhyContains               []string       `json:"require_why_contains"`
	RequireRecommendedActions        []string       `json:"require_recommended_actions"`
	RequireAdapterRecommendations    []string       `json:"require_adapter_recommendations"`
	ForbidAdapterRecommendations     []string       `json:"forbid_adapter_recommendations"`
}
```

to:

```go
type Expected struct {
	Support                         string         `json:"support"`
	TopID                           string         `json:"top_id"`
	IncludeIDs                      []string       `json:"include_ids"`
	ExcludeIDs                      []string       `json:"exclude_ids"`
	RequireNextStepTools            []string       `json:"require_next_step_tools"`
	ForbidNextStepTools             []string       `json:"forbid_next_step_tools"`
	RequireMissingHints             bool           `json:"require_missing_hints"`
	RequireUsageClaim               string         `json:"require_usage_claim"`
	SearchSemanticStatus            string         `json:"search_semantic_status"`
	MissingNeedSources              []string       `json:"missing_need_sources"`
	NoBodies                        bool           `json:"no_bodies"`
	CiteIDs                         []string       `json:"cite_ids"`
	InventoryTypesMin               map[string]int `json:"inventory_types_min"`
	IncludeDeprecated               bool           `json:"include_deprecated"`
	RequireWhyContains              []string       `json:"require_why_contains"`
	RequireRecommendedSteps         []string       `json:"require_recommended_steps"`
	RequireAdapterRecommendations   []string       `json:"require_adapter_recommendations"`
	ForbidAdapterRecommendations    []string       `json:"forbid_adapter_recommendations"`
}
```

- [ ] **Step 3: Update fixture sanity tests**

In `internal/discoverytest/golden_test.go`, update case IDs and fields:

```go
func TestLoadCases(t *testing.T) {
	cases := LoadCases(t)
	if len(cases) == 0 {
		t.Fatal("expected discovery golden cases")
	}
	if CaseByID(t, cases, "strong_auth_refresh_full_signal").Expected.Support != "strong" {
		t.Fatalf("expected strong_auth_refresh_full_signal to expect strong support")
	}
	if CaseByID(t, cases, "list_inventory_empty").Fixture != "empty" {
		t.Fatalf("expected list_inventory_empty to use empty fixture")
	}
}
```

- [ ] **Step 4: Run fixture loader tests and verify expected failure/pass**

Run:

```bash
go test ./internal/discoverytest -count=1
```

Expected: PASS after all fixture field references are updated.

- [ ] **Step 5: Commit**

```bash
git add testdata/discovery-golden/cases.json internal/discoverytest/golden.go internal/discoverytest/golden_test.go
git commit -m "test: rename discovery golden vocabulary"
```

---

### Task 2: Rename Query Response Model

**Files:**
- Modify: `internal/query/query.go`
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/discovery_golden_test.go`

- [ ] **Step 1: Update response and helper type names**

In `internal/query/query.go`, rename these public types and JSON fields:

```go
type DiscoveryResponse struct {
	Project      string                   `json:"project"`
	Phase        string                   `json:"phase"`
	Query        string                   `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Coverage     Coverage                 `json:"coverage"`
	ActionPolicy ActionPolicy             `json:"action_policy"`
	Recall       RecallState              `json:"recall"`
	CoverageGaps []CoverageGap            `json:"coverage_gaps,omitempty"`
	Items        []DiscoveryItem          `json:"items"`
	NextCalls    []RecommendedCall        `json:"next_calls"`
}
```

to:

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
```

Rename:

```go
type MapResponse struct
```

to:

```go
type ListKnowledgeResponse struct
```

Rename:

```go
type Coverage struct
```

to:

```go
type Support struct {
	Level                 string   `json:"level"`
	Confidence            float64  `json:"confidence"`
	Reason                string   `json:"reason"`
	Recommendation        string   `json:"recommendation"`
	MissingKnowledgeHints  []string `json:"missing_knowledge_hints,omitempty"`
}
```

Rename:

```go
type ActionPolicy struct
```

to:

```go
type UsageGuidance struct {
	Read   string `json:"read"`
	Cite   string `json:"cite"`
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}
```

Remove the old `Authority` field. Use `Support.Level` as the support signal.

Rename:

```go
type RecallState struct
type SemanticRecallState struct
type CoverageGap struct
type DiscoveryItem struct
type Disclosure struct
type RecommendedCall struct
```

to:

```go
type SearchStatus struct
type SemanticSearchStatus struct
type MissingNeed struct
type KnowledgeSummary struct
type ReadStatus struct
type NextStep struct
```

Use:

```go
type ReadStatus struct {
	Level             string `json:"level"`
	FullBodyAvailable bool   `json:"full_body_available"`
	ReadTool          string `json:"read_tool"`
}
```

- [ ] **Step 2: Rename service methods while preserving behavior**

Rename:

```go
func (s *Service) Discover(req DiscoverRequest) (DiscoveryResponse, error)
func (s *Service) Map(req MapRequest) (MapResponse, error)
```

to:

```go
func (s *Service) FindKnowledge(req FindKnowledgeRequest) (FindKnowledgeResponse, error)
func (s *Service) ListKnowledge(req ListKnowledgeRequest) (ListKnowledgeResponse, error)
```

Rename request types:

```go
type DiscoverRequest -> type FindKnowledgeRequest
type MapRequest -> type ListKnowledgeRequest
```

Do not change request JSON fields except where tool names appear in descriptions. Keep `project`, `phase`, `task`, `query`, `files`, `types`, `tags`, `domains`, `status`, `include_deprecated`, and `limit`.

- [ ] **Step 3: Rename helper functions and source values**

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

Rename missing need source values:

```go
"unmatched_intent" -> "not_found"
"filter_excluded" -> "filtered_out"
"conflicting_shared_knowledge" -> "conflict"
"cross_domain_mismatch" -> "wrong_scope"
```

Keep these source values unchanged:

```go
"weak_match"
"partial_match"
"low_confidence"
```

- [ ] **Step 4: Update query tests to compile against new names**

In `internal/query/discovery_golden_test.go`, change assertions:

```go
assertCoverage(t, result.Coverage, tc.Expected.Coverage, result.Items)
assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
assertRecallMatchesExpected(t, result.Recall, tc.Expected)
assertCoverageGapsMatchExpected(t, result.CoverageGaps, tc.Expected.CoverageGapSources)
assertNoDiscoveryBodies(t, result.Items, tc.Expected.NoBodies)
```

to:

```go
assertSupport(t, result.Support, tc.Expected.Support, result.Items)
assertUsageMatchesExpected(t, result.Usage, tc.Expected)
assertSearchStatusMatchesExpected(t, result.SearchStatus, tc.Expected)
assertMissingNeedsMatchExpected(t, result.MissingNeeds, tc.Expected.MissingNeedSources)
assertNoKnowledgeBodies(t, result.Items, tc.Expected.NoBodies)
```

Update helper signatures:

```go
func assertSupport(t *testing.T, got Support, want string, items []KnowledgeSummary)
func assertUsageMatchesExpected(t *testing.T, got UsageGuidance, expected discoverytest.Expected)
func assertSearchStatusMatchesExpected(t *testing.T, got SearchStatus, expected discoverytest.Expected)
func assertMissingNeedsMatchExpected(t *testing.T, got []MissingNeed, want []string)
```

- [ ] **Step 5: Run query tests**

Run:

```bash
go test ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/query/query.go internal/query/query_test.go internal/query/discovery_golden_test.go
git commit -m "refactor: rename query discovery model"
```

---

### Task 3: Rename MCP Tool Surface

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Update MCP tool handlers**

In `internal/mcp/server.go`, replace handler cases:

```go
case "argos_discover":
```

with:

```go
case "argos_find_knowledge":
```

and call:

```go
result, err := s.queries.FindKnowledge(req)
```

Replace:

```go
case "argos_map":
```

with:

```go
case "argos_list_knowledge":
```

and call:

```go
result, err := s.queries.ListKnowledge(req)
```

Replace:

```go
case "get_knowledge_item":
```

with:

```go
case "argos_read_knowledge":
```

Replace:

```go
case "cite_knowledge":
```

with:

```go
case "argos_cite_knowledge":
```

Update error prefixes to the new tool names:

```text
invalid arguments for argos_find_knowledge
invalid arguments for argos_list_knowledge
invalid arguments for argos_read_knowledge
invalid arguments for argos_cite_knowledge
```

- [ ] **Step 2: Update `tools/list` schemas**

In `tools()`, expose:

```go
{
	Name:        "argos_find_knowledge",
	Description: "Find shared knowledge for current work.",
	InputSchema: findKnowledgeInputSchema(),
},
{
	Name:        "argos_list_knowledge",
	Description: "List available shared knowledge for a project.",
	InputSchema: objectSchema(...),
},
{
	Name:        "argos_read_knowledge",
	Description: "Read a knowledge item by id.",
	InputSchema: objectSchema(...),
},
{
	Name:        "argos_cite_knowledge",
	Description: "Create citations for knowledge items actually used.",
	InputSchema: objectSchema(...),
},
```

Rename:

```go
func discoverInputSchema()
```

to:

```go
func findKnowledgeInputSchema()
```

Change the limit description to:

```go
"Maximum number of knowledge items to return."
```

- [ ] **Step 3: Update context and next-step tool names**

In `internal/query/query.go`, every `NextStep.Tool` and context recommendation should use:

```text
argos_list_knowledge
argos_find_knowledge
argos_read_knowledge
argos_cite_knowledge
```

Do not return old tool names in any `next_steps` response.

- [ ] **Step 4: Update MCP tests**

In `internal/mcp/server_test.go`, update the expected tools list:

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

Update tool-call JSON examples from:

```json
{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth"}}
```

to:

```json
{"name":"argos_find_knowledge","arguments":{"project":"mall-api","query":"auth"}}
```

Update decoded response type to:

```go
var decoded query.FindKnowledgeResponse
```

Assert:

```go
if decoded.Support.Level != "none" {
	t.Fatalf("expected none support, got %q", decoded.Support.Level)
}
if len(decoded.MissingNeeds) == 0 {
	t.Fatalf("expected missing needs, got none: %#v", decoded)
}
```

- [ ] **Step 5: Add old tool rejection tests**

Add a table test to `internal/mcp/server_test.go`:

```go
func TestRenamedMCPToolsRejectOldNames(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	server := NewServer(store)

	oldNames := []string{
		"argos_discover",
		"argos_map",
		"get_knowledge_item",
		"cite_knowledge",
	}
	for _, oldName := range oldNames {
		t.Run(oldName, func(t *testing.T) {
			var out bytes.Buffer
			line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + oldName + `","arguments":{"project":"mall-api","query":"auth","id":"rule:backend.auth.v1","ids":["rule:backend.auth.v1"]}}}`)
			err := server.HandleLine(line, &out)
			if err != nil {
				t.Fatalf("handle line: %v", err)
			}
			assertToolErrorContains(t, out.Bytes(), "unknown tool")
		})
	}
}
```

- [ ] **Step 6: Run MCP tests**

Run:

```bash
go test ./internal/mcp -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go internal/query/query.go internal/query/query_test.go
git commit -m "feat: rename mcp knowledge tools"
```

---

### Task 4: Rename CLI Commands And Golden CLI Cases

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/discovery_golden_test.go`
- Modify: `cmd/argos/e2e_test.go`
- Modify: `testdata/discovery-golden/cases.json`

- [ ] **Step 1: Choose the CLI shape for this migration**

Use the grouped command shape from the approved spec:

```text
argos knowledge list
argos knowledge find
argos knowledge read <id>
argos knowledge cite <id>...
```

Keep `context` and `standards` as existing top-level commands.

- [ ] **Step 2: Update CLI dispatch**

In `internal/cli/cli.go`, replace the top-level `discover` and `map` cases with a `knowledge` case:

```go
case "knowledge":
	if len(args) < 2 {
		fmt.Fprintln(stderr, "knowledge: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[1] {
	case "find":
		return runKnowledgeFind(args[2:], stdout, stderr)
	case "list":
		return runKnowledgeList(args[2:], stdout, stderr)
	case "read":
		return runKnowledgeRead(args[2:], stdout, stderr)
	case "cite":
		return runKnowledgeCite(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "knowledge: unknown subcommand %q\n", args[1])
		printUsage(stderr)
		return 2
	}
```

Move the current `discover` case body into:

```go
func runKnowledgeFind(args []string, stdout io.Writer, stderr io.Writer) int
```

Move the current `map` case body into:

```go
func runKnowledgeList(args []string, stdout io.Writer, stderr io.Writer) int
```

Use new error prefixes:

```text
knowledge find: --json is required
knowledge find: --project is required
knowledge find: --task or --query is required
knowledge find: --limit must be between 1 and 20
knowledge list: --json is required
knowledge list: --project is required
```

- [ ] **Step 3: Add read and cite CLI subcommands**

The current CLI has no read/cite commands. Add:

```go
func runKnowledgeRead(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge read", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOutput {
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
	service := query.New(store)
	result, err := service.GetKnowledgeItem(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "knowledge read: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}
```

Add:

```go
func runKnowledgeCite(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge cite", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOutput {
		fmt.Fprintln(stderr, "knowledge cite: --json is required")
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "knowledge cite: ids are required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	service := query.New(store)
	result := service.CiteKnowledge(flags.Args())
	return printJSON(stdout, stderr, result)
}
```

- [ ] **Step 4: Update CLI help**

In `printUsage`, replace:

```text
discover
map
```

with:

```text
knowledge list
knowledge find
knowledge read
knowledge cite
```

- [ ] **Step 5: Update CLI tests**

Rename tests:

```text
TestRunDiscoverReturnsJSONRoutes -> TestRunKnowledgeFindReturnsJSONItems
TestRunDiscoverAcceptsRepeatedFiles -> TestRunKnowledgeFindAcceptsRepeatedFiles
TestRunDiscoverAcceptsDiscoveryFiltersAndLimit -> TestRunKnowledgeFindAcceptsFiltersAndLimit
TestRunMapReturnsJSONInventory -> TestRunKnowledgeListReturnsJSONInventory
TestRunMapAcceptsTypesAndIncludeDeprecated -> TestRunKnowledgeListAcceptsTypesAndIncludeDeprecated
```

Update calls:

```go
Run([]string{"discover", "--json", "--project", "mall-api", "--query", "auth"}, &stdout, &stderr)
```

to:

```go
Run([]string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "auth"}, &stdout, &stderr)
```

Update:

```go
Run([]string{"map", "--json", "--project", "mall-api"}, &stdout, &stderr)
```

to:

```go
Run([]string{"knowledge", "list", "--json", "--project", "mall-api"}, &stdout, &stderr)
```

Update JSON assertions to `support.level`.

- [ ] **Step 6: Run CLI tests**

Run:

```bash
go test ./internal/cli ./cmd/argos -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go internal/cli/discovery_golden_test.go cmd/argos/e2e_test.go testdata/discovery-golden/cases.json
git commit -m "feat: rename cli knowledge commands"
```

---

### Task 5: Update Adapters And Active Guidance Docs

**Files:**
- Modify: `internal/adapters/adapters.go`
- Modify: `internal/adapters/adapters_test.go`
- Modify: `docs/superpowers/templates/argos-discovery-dogfood-report.md`
- Modify: `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- Modify: active specs listed in File Structure.

- [ ] **Step 1: Update adapter instructions**

In `internal/adapters/adapters.go`, replace the workflow block with:

```text
1. Use argos_context to understand the current workflow.
2. Use argos_list_knowledge for broad orientation before unfamiliar project work.
3. Use argos_find_knowledge to find shared knowledge for the current task.
4. Use argos_read_knowledge only for selected knowledge IDs you plan to apply.
5. Use argos_cite_knowledge only for knowledge IDs you actually read and used.
6. Do not query SQLite, vector tables, or generated files as the primary path when MCP is available.
7. If MCP is unavailable, use CLI JSON next: argos knowledge list/find/read/cite.
8. If both MCP and CLI are unavailable, use generated adapter files and Markdown as fallbacks.
9. Do not cite IDs returned only by list/find; cite only read and used knowledge.
10. Weak or none support means Argos did not provide task support; continue with normal reasoning without Argos-backed claims.
11. When missing_needs are present, separate shared-knowledge-backed claims from general reasoning.
12. Do not cite missing_needs; they are task needs, not knowledge items.
```

- [ ] **Step 2: Update adapter tests**

In `internal/adapters/adapters_test.go`, update expected snippets:

```go
"Use argos_find_knowledge to find shared knowledge for the current task.",
"Use argos_list_knowledge for broad orientation before unfamiliar project work.",
"Use argos_read_knowledge only for selected knowledge IDs you plan to apply.",
"Use argos_cite_knowledge only for knowledge IDs you actually read and used.",
"When missing_needs are present, separate shared-knowledge-backed claims from general reasoning.",
"Do not cite missing_needs; they are task needs, not knowledge items.",
```

Update retired terms helper to reject old active names:

```go
func retiredUserVisibleTerms() []string {
	return []string{
		"argos_discover",
		"argos_map",
		"get_knowledge_item",
		"coverage_gaps",
		"action_policy",
		"Authority",
		"RecallState",
		"Disclosure",
	}
}
```

Do not reject `argos_cite_knowledge`.

- [ ] **Step 3: Update dogfood checklist**

In `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`, replace the runner workflow with:

```text
1. Start from a fresh context.
2. Call `argos_list_knowledge` if you need inventory awareness.
3. Call `argos_find_knowledge` with the case input.
4. Inspect `support`, `missing_needs`, `usage`, `search_status`, and `next_steps`.
5. Call `argos_read_knowledge` only for selected IDs.
6. Call `argos_cite_knowledge` only for IDs actually read and used.
7. Produce the report using `docs/superpowers/templates/argos-discovery-dogfood-report.md`.
```

Replace:

```text
coverage -> support
coverage gaps -> missing needs
map/discover -> list/find
loaded IDs -> read IDs
```

- [ ] **Step 4: Update dogfood report template**

In `docs/superpowers/templates/argos-discovery-dogfood-report.md`, use:

```markdown
- `argos_list_knowledge`:
- `argos_find_knowledge`:
- `argos_read_knowledge`:
- `argos_cite_knowledge`:

- Actual support:
- Search status:
- Missing needs:
- Read status: `pass|fail`
- Missing needs not cited: `pass|fail|not-applicable`
```

- [ ] **Step 5: Update active specs**

Update active specs to refer to the new public vocabulary. Add a supersession note at the top of older active discovery specs:

```markdown
> Naming note: the approved public vocabulary is now list/find/read/cite,
> support, missing_needs, usage, search_status, read_status, and next_steps.
> Older terms in historical examples should be read as pre-audit vocabulary.
```

For the current active guidance sections, replace old names directly instead of relying only on the note.

- [ ] **Step 6: Run adapter tests and docs scan**

Run:

```bash
go test ./internal/adapters -count=1
```

Expected: PASS.

Run:

```bash
rg -n "argos_discover|argos_map|get_knowledge_item|coverage_gaps|action_policy|RecallState|Disclosure" internal testdata docs/superpowers/templates docs/superpowers/checklists
```

Expected: no matches. If a match is in a negative test helper that builds old terms from parts, keep it only if it does not contain the literal full old term.

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/adapters.go internal/adapters/adapters_test.go docs/superpowers/templates/argos-discovery-dogfood-report.md docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md docs/superpowers/specs
git commit -m "docs: update adapters and guidance naming"
```

---

### Task 6: Final Verification And Retired Name Guard

**Files:**
- Modify: tests only if final scan exposes missing guards.

- [ ] **Step 1: Run the full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run active-surface retired-name scan**

Run:

```bash
rg -n "argos_discover|argos_map|get_knowledge_item|\\bcoverage\\b|coverage_gaps|CoverageGap|action_policy|ActionPolicy|\\bauthority\\b|\\brecall\\b|RecallState|\\bdisclosure\\b|Disclosure" internal testdata docs/superpowers/templates docs/superpowers/checklists
```

Expected: no matches except:

- `coverage` may appear in historical comments only if the comment explicitly explains old vocabulary; prefer removing it.
- `authority` may appear only in a negative test helper built from parts, not as a literal user-visible term.
- `cite_knowledge` may appear as part of the valid new tool name `argos_cite_knowledge`.

If the scan reports valid new `argos_cite_knowledge` matches because of the substring `cite_knowledge`, do not add `cite_knowledge` to this regex. Reject only old standalone tool names in tests.

- [ ] **Step 3: Run active specs scan**

Run:

```bash
rg -n "argos_discover|argos_map|get_knowledge_item|coverage_gaps|CoverageGap|action_policy|ActionPolicy|\\bauthority\\b|\\brecall\\b|RecallState|\\bdisclosure\\b|Disclosure" docs/superpowers/specs
```

Expected: only historical/supersession notes, not current recommendations or examples. If old names appear in current examples, update the spec.

- [ ] **Step 4: Confirm no old tool names in MCP tools/list**

Run:

```bash
go test ./internal/mcp -run 'TestToolsList|TestRenamedMCPToolsRejectOldNames' -count=1
```

Expected: PASS.

- [ ] **Step 5: Confirm JSON shape in one CLI path**

Run:

```bash
go test ./internal/cli -run TestRunKnowledgeFindReturnsJSONItems -count=1
```

Expected: PASS and the test asserts `support.level`, `usage`, `search_status`, and `next_steps`.

- [ ] **Step 6: Commit final guards if changed**

If this task required test/doc guard changes, commit:

```bash
git add internal testdata docs/superpowers/templates docs/superpowers/checklists docs/superpowers/specs
git commit -m "test: guard renamed knowledge surfaces"
```

If no files changed, do not create an empty commit.

---

## Self-Review

### Spec Coverage

- Public tool names `argos_list_knowledge`, `argos_find_knowledge`, `argos_read_knowledge`, `argos_cite_knowledge`: Tasks 3, 4, and 5.
- Response names `support`, `missing_needs`, `usage`, `search_status`, `next_steps`, `read_status`: Tasks 1, 2, 3, and 4.
- Internal Go names: Task 2.
- CLI names: Task 4.
- Adapter language: Task 5.
- Active guidance docs: Task 5.
- Retired-name scans: Tasks 5 and 6.
- Behavior unchanged: every task uses existing tests plus renamed golden expectations; full suite runs in Task 6.

### Type Consistency

The plan uses these final names consistently:

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

### Scope Boundaries

This plan does not add vector search, upload, capture, ranking changes, or storage changes. It is a naming migration plus verification guards.
