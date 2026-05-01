# Argos Dogfood Harness Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a repeatable local dogfood harness that generates runner packets, parses structured reports, and evaluates reports against golden discovery cases.

**Architecture:** Add a focused `internal/dogfood` package for case loading, packet rendering, report parsing, and evaluation. Keep `internal/cli` thin by adding a `dogfood` command group that delegates to that package. Preserve human-in-the-loop AI execution: Argos creates packets and evaluates saved reports, but does not invoke AI.

**Tech Stack:** Go standard library, existing `internal/discoverytest` golden structs, Markdown/JSON output, existing CLI test style.

---

## File Structure

- Create `internal/dogfood/cases.go`: load and summarize golden cases without exposing expectations.
- Create `internal/dogfood/packet.go`: render Markdown and JSON runner packets.
- Create `internal/dogfood/report.go`: parse structured Markdown runner reports.
- Create `internal/dogfood/evaluate.go`: compare parsed reports against golden expectations.
- Create `internal/dogfood/dogfood_test.go`: unit tests for packet rendering, parsing, and evaluation.
- Modify `internal/cli/cli.go`: add `dogfood cases`, `dogfood packet`, and `dogfood evaluate`.
- Modify `internal/cli/cli_test.go`: add CLI coverage for dogfood commands.
- Modify `README.md`: document the dogfood harness command flow.
- Modify `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`: point human coordinators to generated packets.

The implementation intentionally does not modify query, ranking, indexing, MCP behavior, or golden expectations.

---

### Task 1: Case Catalog And Runner Packet Rendering

**Files:**
- Create: `internal/dogfood/cases.go`
- Create: `internal/dogfood/packet.go`
- Create: `internal/dogfood/dogfood_test.go`

- [ ] **Step 1: Write failing tests for case summaries and packet rendering**

Add this to `internal/dogfood/dogfood_test.go`:

```go
package dogfood

import (
	"encoding/json"
	"strings"
	"testing"
)

const goldenCasesPath = "../../testdata/discovery-golden/cases.json"

func TestListCasesHidesExpectedValues(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	body, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "strong_auth_refresh_full_signal") {
		t.Fatalf("expected case id in summaries: %s", text)
	}
	for _, forbidden := range []string{"expected", "support_level", "include_ids", "rule:backend.auth-refresh.v1"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("case summaries leaked %q: %s", forbidden, text)
		}
	}
}

func TestPacketMarkdownOmitsExpectedAndIncludesCLIEquivalents(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      "partial_domain_without_task_detail",
		Workspace:   "/tmp/argos-dogfood/full",
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	text := packet.Markdown
	for _, want := range []string{
		"Case: `partial_domain_without_task_detail`",
		"Workspace: `/tmp/argos-dogfood/full`",
		"Argos binary: `/tmp/argos`",
		"/tmp/argos knowledge find --json --project <project>",
		"/tmp/argos knowledge read --json <id>",
		"/tmp/argos knowledge cite --json <id>...",
		"Do not inspect `cases.json`",
		"\"task\": \"tune cache ttl for product list\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("packet missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"\"expected\"", "reference:backend.cache-policy.v1", "support_level"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("packet leaked %q:\n%s", forbidden, text)
		}
	}
}

func TestPacketAddsOperationSpecificNotes(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	for _, tc := range []struct {
		name string
		id   string
		want string
	}{
		{name: "inventory", id: "list_inventory_normal", want: "Inventory case: run list only; do not read or cite."},
		{name: "deprecated", id: "list_hides_deprecated_by_default", want: "Deprecated visibility case: run default list, then run list with `--include-deprecated`."},
		{name: "workflow", id: "progressive_read_status_and_citation_guard", want: "Workflow case: find, read selected IDs, and cite only read-and-used IDs."},
		{name: "mcp", id: "interface_mcp_strict_schema", want: "MCP schema case: use JSON-RPC commands through `argos mcp`."},
		{name: "adapter", id: "adapter_flow_recommendations", want: "Adapter case: run `argos install-adapters` before checking generated files."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			packet, err := BuildPacket(cases, PacketOptions{CaseID: tc.id, Workspace: "/tmp/ws", ArgosBinary: "/tmp/argos"})
			if err != nil {
				t.Fatalf("BuildPacket returned error: %v", err)
			}
			if !strings.Contains(packet.Markdown, tc.want) {
				t.Fatalf("packet missing note %q:\n%s", tc.want, packet.Markdown)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/dogfood -run 'TestListCasesHidesExpectedValues|TestPacketMarkdownOmitsExpectedAndIncludesCLIEquivalents|TestPacketAddsOperationSpecificNotes' -count=1
```

Expected: FAIL because `internal/dogfood` does not exist.

- [ ] **Step 3: Implement case loading and summaries**

Create `internal/dogfood/cases.go`:

```go
package dogfood

import (
	"encoding/json"
	"fmt"
	"os"

	"argos/internal/discoverytest"
)

type CaseSummary struct {
	ID        string              `json:"id"`
	Fixture   string              `json:"fixture,omitempty"`
	Operation string              `json:"operation"`
	Input     discoverytest.Input `json:"input"`
}

func LoadCases(path string) ([]discoverytest.Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dogfood cases: %w", err)
	}
	var file discoverytest.CaseFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse dogfood cases: %w", err)
	}
	return file.Cases, nil
}

func Summaries(cases []discoverytest.Case) []CaseSummary {
	summaries := make([]CaseSummary, 0, len(cases))
	for _, tc := range cases {
		summaries = append(summaries, CaseSummary{
			ID:        tc.ID,
			Fixture:   fixtureName(tc.Fixture),
			Operation: tc.Operation,
			Input:     tc.Input,
		})
	}
	return summaries
}

func FindCase(cases []discoverytest.Case, id string) (discoverytest.Case, bool) {
	for _, tc := range cases {
		if tc.ID == id {
			return tc, true
		}
	}
	return discoverytest.Case{}, false
}

func fixtureName(value string) string {
	if value == "" {
		return "full"
	}
	return value
}
```

- [ ] **Step 4: Implement packet rendering**

Create `internal/dogfood/packet.go`:

```go
package dogfood

import (
	"encoding/json"
	"fmt"
	"strings"

	"argos/internal/discoverytest"
)

type PacketOptions struct {
	CaseID      string
	Workspace   string
	ArgosBinary string
}

type Packet struct {
	CaseID      string              `json:"case_id"`
	Operation   string              `json:"operation"`
	Fixture     string              `json:"fixture"`
	Workspace   string              `json:"workspace"`
	ArgosBinary string              `json:"argos_binary"`
	Input       discoverytest.Input `json:"input"`
	Markdown    string              `json:"markdown"`
}

func BuildPacket(cases []discoverytest.Case, options PacketOptions) (Packet, error) {
	tc, ok := FindCase(cases, options.CaseID)
	if !ok {
		return Packet{}, fmt.Errorf("unknown dogfood case: %s", options.CaseID)
	}
	if strings.TrimSpace(options.Workspace) == "" {
		return Packet{}, fmt.Errorf("dogfood packet: --workspace is required")
	}
	if strings.TrimSpace(options.ArgosBinary) == "" {
		return Packet{}, fmt.Errorf("dogfood packet: --argos-binary is required")
	}
	packet := Packet{
		CaseID:      tc.ID,
		Operation:   tc.Operation,
		Fixture:     fixtureName(tc.Fixture),
		Workspace:   options.Workspace,
		ArgosBinary: options.ArgosBinary,
		Input:       tc.Input,
	}
	packet.Markdown = renderPacketMarkdown(packet, tc)
	return packet, nil
}

func renderPacketMarkdown(packet Packet, tc discoverytest.Case) string {
	input, err := json.MarshalIndent(packet.Input, "", "  ")
	if err != nil {
		input = []byte("{}")
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Argos Discovery Dogfood Runner Packet\n\n")
	fmt.Fprintf(&builder, "Case: `%s`\n", packet.CaseID)
	fmt.Fprintf(&builder, "Operation: `%s`\n", packet.Operation)
	fmt.Fprintf(&builder, "Fixture: `%s`\n", packet.Fixture)
	fmt.Fprintf(&builder, "Workspace: `%s`\n", packet.Workspace)
	fmt.Fprintf(&builder, "Argos binary: `%s`\n\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "## Context Isolation\n\n")
	fmt.Fprintf(&builder, "- Run this case in a fresh AI session.\n")
	fmt.Fprintf(&builder, "- Do not inspect `cases.json`, source files, SQLite/FTS tables, vector tables, or Markdown knowledge files directly.\n")
	fmt.Fprintf(&builder, "- Do not use expected IDs or expected support values.\n")
	fmt.Fprintf(&builder, "- Do not cite IDs that were not read.\n")
	fmt.Fprintf(&builder, "- Do not treat weak or none support as Argos-backed guidance.\n\n")
	fmt.Fprintf(&builder, "## CLI Equivalents\n\n")
	fmt.Fprintf(&builder, "MCP tool names are conceptual when MCP is unavailable. Use these CLI forms from the fixture workspace:\n\n")
	fmt.Fprintf(&builder, "```bash\n")
	fmt.Fprintf(&builder, "%s context --json --project <project> --phase <phase> --task \"<task>\"\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge list --json --project <project> [--domain <domain>] [--types <type>] [--include-deprecated]\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge find --json --project <project> [--phase <phase>] [--task \"<task>\"] [--query \"<query>\"] [--files <path>] [--types <type>] [--tags <tag>] [--domains <domain>] [--status <status>] [--include-deprecated] [--limit <n>]\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge read --json <id>\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "%s knowledge cite --json <id>...\n", packet.ArgosBinary)
	fmt.Fprintf(&builder, "```\n\n")
	fmt.Fprintf(&builder, "Repeat flags once per value. Do not call MCP-style names directly as CLI subcommands.\n\n")
	fmt.Fprintf(&builder, "## Case Input\n\n")
	fmt.Fprintf(&builder, "```json\n%s\n```\n\n", string(input))
	fmt.Fprintf(&builder, "## Case Notes\n\n")
	for _, note := range operationNotes(tc) {
		fmt.Fprintf(&builder, "- %s\n", note)
	}
	fmt.Fprintf(&builder, "\n## Required Report Shape\n\n")
	fmt.Fprintf(&builder, "Use sections `## Inputs`, `## Tool Transcript Summary`, `## Observed Results`, `## Guards`, and `## Result`.\n")
	return builder.String()
}

func operationNotes(tc discoverytest.Case) []string {
	notes := []string{}
	switch tc.Operation {
	case "list", "list-empty", "cli-list":
		notes = append(notes, "Inventory case: run list only; do not read or cite.")
	case "workflow":
		notes = append(notes, "Workflow case: find, read selected IDs, and cite only read-and-used IDs.")
	case "mcp-schema":
		notes = append(notes, "MCP schema case: use JSON-RPC commands through `argos mcp`.")
	case "cli-read":
		notes = append(notes, "CLI read case: use `argos knowledge read --json <id>` and verify the body is present.")
	case "cli-cite":
		notes = append(notes, "CLI cite case: use `argos knowledge cite --json <id>` and verify citation metadata is present.")
	case "adapter":
		notes = append(notes, "Adapter case: run `argos install-adapters` before checking generated files.")
	default:
		notes = append(notes, "Find case: run find with the case input, read only selected IDs, and cite only read-and-used IDs.")
	}
	if tc.Expected.IncludeDeprecatedIDWhenRequested != "" {
		notes = append(notes, "Deprecated visibility case: run default list, then run list with `--include-deprecated`.")
	}
	return notes
}
```

- [ ] **Step 5: Run packet tests**

Run:

```bash
go test ./internal/dogfood -run 'TestListCasesHidesExpectedValues|TestPacketMarkdownOmitsExpectedAndIncludesCLIEquivalents|TestPacketAddsOperationSpecificNotes' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 1**

Run:

```bash
git add internal/dogfood/cases.go internal/dogfood/packet.go internal/dogfood/dogfood_test.go
git commit -m "feat: generate dogfood runner packets"
```

---

### Task 2: Structured Report Parser

**Files:**
- Modify: `internal/dogfood/report.go`
- Modify: `internal/dogfood/dogfood_test.go`

- [ ] **Step 1: Add parser tests**

Append to `internal/dogfood/dogfood_test.go`:

```go
func TestParseMarkdownReportExtractsStructuredFields(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(sampleStrongReport()))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	if report.CaseID != "progressive_read_status_and_citation_guard" {
		t.Fatalf("unexpected case id: %#v", report)
	}
	if report.ActualSupport != "strong" {
		t.Fatalf("unexpected support: %#v", report)
	}
	assertStringSetContains(t, report.DiscoveredIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.ReadIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.CitedIDs, "rule:backend.auth-refresh.v1")
	if report.Guards["progressive reading"] != "pass" {
		t.Fatalf("expected progressive reading pass: %#v", report.Guards)
	}
}

func TestParseMarkdownReportRecordsMissingSections(t *testing.T) {
	report, err := ParseMarkdownReport([]byte("Case: `weak_single_generic_term`\n\n## Result\n\nResult: `pass`\n"))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	for _, want := range []string{"Inputs", "Tool Transcript Summary", "Observed Results", "Guards"} {
		assertStringSetContains(t, report.MissingSections, want)
	}
}

func sampleStrongReport() string {
	return "# Argos Discovery Dogfood Report\n\n" +
		"Case: `progressive_read_status_and_citation_guard`\n\n" +
		"## Inputs\n\n" +
		"- Project: `mall-api`\n\n" +
		"## Tool Transcript Summary\n\n" +
		"- `argos_find_knowledge`: returned strong support.\n" +
		"- `argos_read_knowledge`: read `rule:backend.auth-refresh.v1`.\n" +
		"- `argos_cite_knowledge`: cited `rule:backend.auth-refresh.v1`.\n\n" +
		"## Observed Results\n\n" +
		"- Actual support: `strong`\n" +
		"- Usage guidance: read recommended; cite after read and used.\n" +
		"- Search status: semantic disabled.\n" +
		"- Discovered IDs: `rule:backend.auth-refresh.v1`, `decision:backend.session-renewal.v1`\n" +
		"- Read IDs: `rule:backend.auth-refresh.v1`\n" +
		"- Cited IDs: `rule:backend.auth-refresh.v1`\n" +
		"- Missing needs: none\n" +
		"- Argos-backed vs general reasoning: Argos-backed claim used the read rule.\n" +
		"- Next steps: none\n\n" +
		"## Guards\n\n" +
		"- progressive reading: `pass`\n" +
		"- Weak/none no-overclaim: `not-applicable`\n" +
		"- Citation accountability: `pass`\n" +
		"- Cited IDs subset of read-and-used IDs: `pass`\n" +
		"- Missing needs not cited: `pass`\n" +
		"- Attribution boundary: `pass`\n" +
		"- No Discovery-triggered upload/capture: `pass`\n" +
		"- Usage guidance followed: `pass`\n" +
		"- Context contamination: `pass`\n\n" +
		"## Result\n\n" +
		"Result: `pass`\n"
}

func assertStringSetContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %#v", want, got)
}
```

- [ ] **Step 2: Run parser tests to verify they fail**

Run:

```bash
go test ./internal/dogfood -run 'TestParseMarkdownReportExtractsStructuredFields|TestParseMarkdownReportRecordsMissingSections' -count=1
```

Expected: FAIL because `ParseMarkdownReport` is not defined.

- [ ] **Step 3: Implement Markdown report parser**

Create `internal/dogfood/report.go`:

```go
package dogfood

import (
	"regexp"
	"strings"
)

type Report struct {
	CaseID          string            `json:"case_id"`
	ActualSupport   string            `json:"actual_support"`
	UsageGuidance   string            `json:"usage_guidance"`
	SearchStatus    string            `json:"search_status"`
	DiscoveredIDs   []string          `json:"discovered_ids"`
	ReadIDs         []string          `json:"read_ids"`
	CitedIDs        []string          `json:"cited_ids"`
	MissingNeeds    []string          `json:"missing_needs"`
	NextSteps       string            `json:"next_steps"`
	Guards          map[string]string `json:"guards"`
	Result          string            `json:"result"`
	MissingSections []string          `json:"missing_sections"`
	MissingFields   []string          `json:"missing_fields"`
}

var (
	backtickValuePattern = regexp.MustCompile("`([^`]+)`")
	idPattern            = regexp.MustCompile(`[a-z]+:[a-z0-9._-]+`)
)

func ParseMarkdownReport(data []byte) (Report, error) {
	text := string(data)
	report := Report{Guards: map[string]string{}}
	report.CaseID = firstBacktickAfterPrefix(text, "Case:")
	report.Result = normalizeStatus(firstBacktickAfterPrefix(text, "Result:"))
	report.MissingSections = missingSections(text, []string{"Inputs", "Tool Transcript Summary", "Observed Results", "Guards", "Result"})
	observed := section(text, "Observed Results")
	report.ActualSupport = normalizeStatus(valueForBullet(observed, "Actual support"))
	report.UsageGuidance = valueForBullet(observed, "Usage guidance")
	report.SearchStatus = valueForBullet(observed, "Search status")
	report.DiscoveredIDs = idsFromValue(valueForBullet(observed, "Discovered IDs"))
	report.ReadIDs = idsFromValue(valueForBullet(observed, "Read IDs"))
	report.CitedIDs = idsFromValue(valueForBullet(observed, "Cited IDs"))
	report.MissingNeeds = valuesFromList(valueForBullet(observed, "Missing needs"))
	report.NextSteps = valueForBullet(observed, "Next steps")
	guards := section(text, "Guards")
	for _, label := range []string{
		"progressive reading",
		"Weak/none no-overclaim",
		"Citation accountability",
		"Cited IDs subset of read-and-used IDs",
		"Missing needs not cited",
		"Attribution boundary",
		"No Discovery-triggered upload/capture",
		"Usage guidance followed",
		"Context contamination",
	} {
		if value := valueForBullet(guards, label); value != "" {
			report.Guards[strings.ToLower(label)] = normalizeStatus(value)
		}
	}
	report.MissingFields = missingReportFields(report)
	return report, nil
}

func missingSections(text string, names []string) []string {
	var missing []string
	for _, name := range names {
		if !strings.Contains(text, "## "+name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func section(text string, name string) string {
	marker := "## " + name
	start := strings.Index(text, marker)
	if start < 0 {
		return ""
	}
	rest := text[start+len(marker):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		return rest[:next]
	}
	return rest
}

func firstBacktickAfterPrefix(text string, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			matches := backtickValuePattern.FindStringSubmatch(line)
			if len(matches) == 2 {
				return matches[1]
			}
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), prefix))
		}
	}
	return ""
}

func valueForBullet(text string, label string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "- ")
		if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(label)+":") {
			return strings.TrimSpace(trimmed[len(label)+1:])
		}
	}
	return ""
}

func idsFromValue(value string) []string {
	if strings.EqualFold(strings.Trim(value, "` ."), "none") {
		return nil
	}
	return uniqueStrings(idPattern.FindAllString(value, -1))
}

func valuesFromList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(strings.Trim(value, "` ."), "none") {
		return nil
	}
	return []string{strings.Trim(value, "` ")}
}

func normalizeStatus(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "` .")
	return strings.ToLower(value)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func missingReportFields(report Report) []string {
	var missing []string
	if report.CaseID == "" {
		missing = append(missing, "case")
	}
	if report.ActualSupport == "" {
		missing = append(missing, "actual support")
	}
	for _, guard := range []string{
		"progressive reading",
		"citation accountability",
		"usage guidance followed",
		"context contamination",
	} {
		if report.Guards[guard] == "" {
			missing = append(missing, "guard: "+guard)
		}
	}
	return missing
}
```

- [ ] **Step 4: Run parser tests**

Run:

```bash
go test ./internal/dogfood -run 'TestParseMarkdownReportExtractsStructuredFields|TestParseMarkdownReportRecordsMissingSections' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

Run:

```bash
git add internal/dogfood/report.go internal/dogfood/dogfood_test.go
git commit -m "feat: parse dogfood runner reports"
```

---

### Task 3: Golden Evaluator

**Files:**
- Create: `internal/dogfood/evaluate.go`
- Modify: `internal/dogfood/dogfood_test.go`

- [ ] **Step 1: Add evaluator tests**

Append to `internal/dogfood/dogfood_test.go`:

```go
func TestEvaluateReportPassesStrongWorkflow(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, ok := FindCase(cases, "progressive_read_status_and_citation_guard")
	if !ok {
		t.Fatal("missing golden case")
	}
	report, err := ParseMarkdownReport([]byte(sampleStrongReport()))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	evaluation := Evaluate(tc, report)
	if evaluation.Result != ResultPass {
		t.Fatalf("expected pass, got %#v", evaluation)
	}
}

func TestEvaluateReportFailsUnreadCitation(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, ok := FindCase(cases, "progressive_read_status_and_citation_guard")
	if !ok {
		t.Fatal("missing golden case")
	}
	report, err := ParseMarkdownReport([]byte(strings.ReplaceAll(sampleStrongReport(), "- Read IDs: `rule:backend.auth-refresh.v1`", "- Read IDs: none")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	evaluation := Evaluate(tc, report)
	if evaluation.Result != ResultFail {
		t.Fatalf("expected fail, got %#v", evaluation)
	}
	if !strings.Contains(evaluation.Findings[0].Message, "cited ID was not read") {
		t.Fatalf("expected unread citation finding, got %#v", evaluation.Findings)
	}
}

func TestEvaluateReportReviewNeededForShapeDrift(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, ok := FindCase(cases, "weak_single_generic_term")
	if !ok {
		t.Fatal("missing golden case")
	}
	report, err := ParseMarkdownReport([]byte("Case: `weak_single_generic_term`\n\n## Result\n\nResult: `pass`\n"))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	evaluation := Evaluate(tc, report)
	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("expected review-needed, got %#v", evaluation)
	}
}
```

- [ ] **Step 2: Run evaluator tests to verify they fail**

Run:

```bash
go test ./internal/dogfood -run 'TestEvaluateReportPassesStrongWorkflow|TestEvaluateReportFailsUnreadCitation|TestEvaluateReportReviewNeededForShapeDrift' -count=1
```

Expected: FAIL because `Evaluate` is not defined.

- [ ] **Step 3: Implement evaluator**

Create `internal/dogfood/evaluate.go`:

```go
package dogfood

import "argos/internal/discoverytest"

const (
	ResultPass         = "pass"
	ResultFail         = "fail"
	ResultReviewNeeded = "review-needed"
)

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Evaluation struct {
	CaseID   string    `json:"case_id"`
	Result   string    `json:"result"`
	Findings []Finding `json:"findings,omitempty"`
}

func Evaluate(tc discoverytest.Case, report Report) Evaluation {
	evaluation := Evaluation{CaseID: tc.ID, Result: ResultPass}
	if report.CaseID != "" && report.CaseID != tc.ID {
		evaluation.addFail("report case id does not match requested case")
	}
	for _, section := range report.MissingSections {
		evaluation.addReview("missing section: " + section)
	}
	for _, field := range report.MissingFields {
		evaluation.addReview("missing field: " + field)
	}
	expectedSupport := tc.Expected.Support
	if expectedSupport == "" {
		expectedSupport = tc.Expected.SupportLevel
	}
	if expectedSupport != "" && report.ActualSupport != "" && report.ActualSupport != expectedSupport {
		evaluation.addFail("actual support does not match expected support")
	}
	for _, id := range tc.Expected.IncludeIDs {
		if !containsString(report.DiscoveredIDs, id) && !containsString(report.ReadIDs, id) && !containsString(report.CitedIDs, id) {
			evaluation.addFail("required ID missing from report: " + id)
		}
	}
	for _, id := range tc.Expected.ExcludeIDs {
		if containsString(report.DiscoveredIDs, id) || containsString(report.ReadIDs, id) || containsString(report.CitedIDs, id) {
			evaluation.addFail("forbidden ID appeared in report: " + id)
		}
	}
	for _, id := range tc.Expected.LoadIDs {
		if !containsString(report.ReadIDs, id) {
			evaluation.addFail("expected read ID missing from report: " + id)
		}
	}
	for _, id := range tc.Expected.CiteIDs {
		if !containsString(report.CitedIDs, id) {
			evaluation.addFail("expected cited ID missing from report: " + id)
		}
	}
	for _, id := range report.CitedIDs {
		if !containsString(report.ReadIDs, id) {
			evaluation.addFail("cited ID was not read: " + id)
		}
	}
	if tc.Expected.UsageRead == "forbidden" && len(report.ReadIDs) > 0 {
		evaluation.addFail("read IDs present when usage.read is forbidden")
	}
	if tc.Expected.UsageCite == "forbidden" && len(report.CitedIDs) > 0 {
		evaluation.addFail("cited IDs present when usage.cite is forbidden")
	}
	for _, guard := range []string{"progressive reading", "citation accountability", "usage guidance followed", "context contamination"} {
		if report.Guards[guard] == "fail" {
			evaluation.addFail("guard failed: " + guard)
		}
	}
	return evaluation
}

func (evaluation *Evaluation) addFail(message string) {
	evaluation.Findings = append(evaluation.Findings, Finding{Severity: "fail", Message: message})
	evaluation.Result = ResultFail
}

func (evaluation *Evaluation) addReview(message string) {
	evaluation.Findings = append(evaluation.Findings, Finding{Severity: "review-needed", Message: message})
	if evaluation.Result == ResultPass {
		evaluation.Result = ResultReviewNeeded
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run evaluator tests**

Run:

```bash
go test ./internal/dogfood -run 'TestEvaluateReportPassesStrongWorkflow|TestEvaluateReportFailsUnreadCitation|TestEvaluateReportReviewNeededForShapeDrift' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 3**

Run:

```bash
git add internal/dogfood/evaluate.go internal/dogfood/dogfood_test.go
git commit -m "feat: evaluate dogfood reports"
```

---

### Task 4: Dogfood CLI Commands

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add CLI tests**

Append to `internal/cli/cli_test.go`:

```go
func TestRunDogfoodCasesReturnsInputsWithoutExpectedValues(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "cases", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "strong_auth_refresh_full_signal") {
		t.Fatalf("expected case id in output: %s", stdout.String())
	}
	for _, forbidden := range []string{"expected", "support_level", "include_ids", "rule:backend.auth-refresh.v1"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("dogfood cases leaked %q: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDogfoodPacketReturnsMarkdownWithoutExpectedValues(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "packet",
		"--case", "partial_domain_without_task_detail",
		"--workspace", "/tmp/argos-dogfood/full",
		"--argos-binary", "/tmp/argos",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Case: `partial_domain_without_task_detail`") {
		t.Fatalf("expected packet markdown: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "support_level") || strings.Contains(stdout.String(), "reference:backend.cache-policy.v1") {
		t.Fatalf("packet leaked expected values: %s", stdout.String())
	}
}

func TestRunDogfoodEvaluateReturnsJSONEvaluation(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(reportPath, []byte(sampleCLIDogfoodReport()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "evaluate", "--json", "--case", "progressive_read_status_and_citation_guard", "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"result": "pass"`) {
		t.Fatalf("expected pass JSON evaluation: %s", stdout.String())
	}
}

func sampleCLIDogfoodReport() string {
	return "# Argos Discovery Dogfood Report\n\n" +
		"Case: `progressive_read_status_and_citation_guard`\n\n" +
		"## Inputs\n\n- Project: `mall-api`\n\n" +
		"## Tool Transcript Summary\n\n- `argos_find_knowledge`: strong.\n\n" +
		"## Observed Results\n\n" +
		"- Actual support: `strong`\n" +
		"- Usage guidance: read recommended; cite after read and used.\n" +
		"- Search status: semantic disabled.\n" +
		"- Discovered IDs: `rule:backend.auth-refresh.v1`\n" +
		"- Read IDs: `rule:backend.auth-refresh.v1`\n" +
		"- Cited IDs: `rule:backend.auth-refresh.v1`\n" +
		"- Missing needs: none\n" +
		"- Argos-backed vs general reasoning: read rule only.\n" +
		"- Next steps: none\n\n" +
		"## Guards\n\n" +
		"- progressive reading: `pass`\n" +
		"- Weak/none no-overclaim: `not-applicable`\n" +
		"- Citation accountability: `pass`\n" +
		"- Cited IDs subset of read-and-used IDs: `pass`\n" +
		"- Missing needs not cited: `pass`\n" +
		"- Attribution boundary: `pass`\n" +
		"- No Discovery-triggered upload/capture: `pass`\n" +
		"- Usage guidance followed: `pass`\n" +
		"- Context contamination: `pass`\n\n" +
		"## Result\n\nResult: `pass`\n"
}
```

- [ ] **Step 2: Run CLI dogfood tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunDogfoodCasesReturnsInputsWithoutExpectedValues|TestRunDogfoodPacketReturnsMarkdownWithoutExpectedValues|TestRunDogfoodEvaluateReturnsJSONEvaluation' -count=1
```

Expected: FAIL because `dogfood` command is unknown.

- [ ] **Step 3: Add dogfood command dispatch**

In `internal/cli/cli.go`, add the import:

```go
	"argos/internal/dogfood"
```

Add a new switch case in `run`:

```go
	case "dogfood":
		return runDogfood(args[1:], stdout, stderr)
```

Add `dogfood` to `printUsage` commands:

```go
	fmt.Fprintln(w, "  dogfood")
```

- [ ] **Step 4: Add dogfood CLI handlers**

Append these functions to `internal/cli/cli.go` near `runKnowledge`:

```go
const defaultDogfoodCasesPath = "testdata/discovery-golden/cases.json"

func runDogfood(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "cases":
		return runDogfoodCases(args[1:], stdout, stderr)
	case "packet":
		return runDogfoodPacket(args[1:], stdout, stderr)
	case "evaluate":
		return runDogfoodEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dogfood: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfoodCases(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood cases: --json is required")
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood cases: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, dogfood.Summaries(cases))
}

func runDogfoodPacket(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood packet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "dogfood case id")
	workspacePath := flags.String("workspace", "", "fixture workspace path")
	argosBinary := flags.String("argos-binary", "", "argos binary path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood packet: %v\n", err)
		return 1
	}
	packet, err := dogfood.BuildPacket(cases, dogfood.PacketOptions{CaseID: *caseID, Workspace: *workspacePath, ArgosBinary: *argosBinary})
	if err != nil {
		fmt.Fprintf(stderr, "dogfood packet: %v\n", err)
		return 2
	}
	if *jsonOut {
		return printJSON(stdout, stderr, packet)
	}
	fmt.Fprint(stdout, packet.Markdown)
	return 0
}

func runDogfoodEvaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "dogfood case id")
	reportPath := flags.String("report", "", "runner report path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: %v\n", err)
		return 1
	}
	tc, ok := dogfood.FindCase(cases, *caseID)
	if !ok {
		fmt.Fprintf(stderr, "dogfood evaluate: unknown dogfood case: %s\n", *caseID)
		return 2
	}
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(stderr, "dogfood evaluate: --report is required")
		return 2
	}
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: read report: %v\n", err)
		return 1
	}
	report, err := dogfood.ParseMarkdownReport(data)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: parse report: %v\n", err)
		return 1
	}
	evaluation := dogfood.Evaluate(tc, report)
	if *jsonOut {
		return printJSON(stdout, stderr, evaluation)
	}
	fmt.Fprintf(stdout, "%s: %s\n", evaluation.CaseID, evaluation.Result)
	for _, finding := range evaluation.Findings {
		fmt.Fprintf(stdout, "- %s: %s\n", finding.Severity, finding.Message)
	}
	return 0
}
```

- [ ] **Step 5: Run CLI dogfood tests**

Run:

```bash
go test ./internal/cli -run 'TestRunDogfoodCasesReturnsInputsWithoutExpectedValues|TestRunDogfoodPacketReturnsMarkdownWithoutExpectedValues|TestRunDogfoodEvaluateReturnsJSONEvaluation' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: add dogfood harness cli"
```

---

### Task 5: Docs, Guard Scan, And Final Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`

- [ ] **Step 1: Update README dogfood validation section**

In `README.md`, under `### Discovery Validation`, replace the AI dogfood validation paragraph with:

```markdown
AI dogfood validation uses:

- `argos dogfood cases --json`
- `argos dogfood packet --case <case-id> --workspace <fixture> --argos-binary <argos>`
- `argos dogfood evaluate --case <case-id> --report <report.md> --json`
- `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`
- `docs/superpowers/templates/argos-discovery-dogfood-report.md`

Dogfood runners must use fresh minimal context per case. Generate runner packets
instead of hand-copying case inputs. Do not give runner agents expected IDs,
expected support, prior transcripts, or design history. Evaluate saved reports
separately against `testdata/discovery-golden/cases.json`.
```

- [ ] **Step 2: Update checklist to prefer generated packets**

In `docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md`, add this paragraph after `## Context Isolation Rules`:

```markdown
Prefer generated runner packets:

```bash
argos dogfood cases --json
argos dogfood packet --case <case-id> --workspace <fixture workspace path> --argos-binary <argos binary path>
argos dogfood evaluate --case <case-id> --report <runner report path> --json
```

The manual template below is a fallback when the productized harness is not
available.
```
```

- [ ] **Step 3: Run targeted tests**

Run:

```bash
go test ./internal/dogfood ./internal/cli ./internal/discoverytest ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 4: Run full repository tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Run active wording scan**

Run:

```bash
go test ./internal/query -run TestActiveSurfacesDoNotUseRetiredSharedKnowledgeNames -count=1
rg -n -i "dogfood.*old shared knowledge vocabulary|MCP-style names directly as CLI subcommands" README.md internal/dogfood internal/cli docs/superpowers/checklists docs/superpowers/specs/2026-05-02-argos-dogfood-harness-productization-design.md docs/superpowers/plans/2026-05-02-argos-dogfood-harness-productization-implementation.md
```

Expected: the guard test passes, and the `rg` command finds no accidental prose that contradicts the current list/find/read/cite vocabulary.

- [ ] **Step 6: Commit Task 5**

Run:

```bash
git add README.md docs/superpowers/checklists/2026-04-30-argos-discovery-dogfood-checklist.md
git commit -m "docs: document dogfood harness workflow"
```

---

## Final Verification

- [ ] **Step 1: Verify log and status**

Run:

```bash
git status --short --branch
git log --oneline -6
```

Expected: clean worktree, implementation commits on `main` or the current feature branch.

- [ ] **Step 2: Verify command examples manually**

Run:

```bash
argos dogfood cases --json
argos dogfood packet --case progressive_read_status_and_citation_guard --workspace /tmp/argos-dogfood/full --argos-binary /tmp/argos
```

Expected: cases output contains no `expected`; packet output contains no expected IDs except IDs present in the input itself.

- [ ] **Step 3: Request final code review**

Use the mandatory review step from subagent-driven development or executing-plans. The reviewer should check:

- runner packets never include expected values;
- evaluator distinguishes `pass`, `fail`, and `review-needed`;
- CLI command output is useful for humans;
- no runtime search/ranking/citation behavior changed.
