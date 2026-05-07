package dogfood

import (
	"encoding/json"
	"strings"
	"testing"

	"argos/internal/discoverytest"
)

const goldenCasesPath = "../../testdata/discovery-golden/cases.json"

func TestListCasesHidesExpectedValues(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	data, err := json.Marshal(Summaries(cases))
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "case-001") {
		t.Fatalf("summaries missing public case handle: %s", text)
	}
	for _, forbidden := range []string{
		"expected",
		"support_level",
		"include_ids",
		"rule:backend.auth-refresh.v1",
		"strong_auth_refresh_full_signal",
		"partial_domain_without_task_detail",
		"weak_single_generic_term",
		"none_payment_webhook",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summaries leaked %q: %s", forbidden, text)
		}
	}
}

func TestPacketMarkdownOmitsExpectedAndIncludesCLIEquivalents(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	publicID, ok := PublicCaseID(cases, "partial_domain_without_task_detail")
	if !ok {
		t.Fatal("missing public case handle")
	}

	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      "partial_domain_without_task_detail",
		Workspace:   " /tmp/argos-dogfood/full ",
		ArgosBinary: " /tmp/argos ",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}

	text := packet.Markdown
	for _, want := range []string{
		"Case: `" + publicID + "`",
		"Workspace: `/tmp/argos-dogfood/full`",
		"Argos binary: `/tmp/argos`",
		"/tmp/argos context --json --project <project> --phase <phase> --task \"<task>\" [--files <path>]",
		"/tmp/argos knowledge find --json --project <project>",
		"/tmp/argos knowledge read --json <id>",
		"/tmp/argos knowledge cite --json <id>...",
		"Do not inspect `cases.json`",
		"\"task\": \"tune cache ttl for product list\"",
		"Include these sections: `## Inputs`, `## Tool Transcript Summary`, `## Observed Results`, `## Guards`, and `## Result`.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("packet missing %q:\n%s", want, text)
		}
	}
	if packet.CaseID != publicID {
		t.Fatalf("packet case ID = %q, want %q", packet.CaseID, publicID)
	}
	for _, forbidden := range []string{"\"expected\"", "reference:backend.cache-policy.v1", "support_level", "partial_domain_without_task_detail"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("packet leaked %q:\n%s", forbidden, text)
		}
	}
}

func TestPublishedCaseHandlesStayStable(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	tests := []struct {
		goldenID string
		want     string
	}{
		{goldenID: "interface_cli_read_returns_body", want: "case-020"},
		{goldenID: "interface_cli_cite_returns_citation", want: "case-021"},
		{goldenID: "interface_mcp_strict_schema", want: "case-022"},
	}
	for _, tt := range tests {
		t.Run(tt.goldenID, func(t *testing.T) {
			got, ok := PublicCaseID(cases, tt.goldenID)
			if !ok {
				t.Fatalf("missing public case handle for %s", tt.goldenID)
			}
			if got != tt.want {
				t.Fatalf("PublicCaseID(%s) = %q, want %q", tt.goldenID, got, tt.want)
			}
		})
	}
}

func TestFindCaseAcceptsPublicCaseHandle(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	publicID, ok := PublicCaseID(cases, "strong_auth_refresh_full_signal")
	if !ok {
		t.Fatal("missing public case handle")
	}
	tc, ok := FindCase(cases, publicID)
	if !ok {
		t.Fatalf("FindCase did not accept public handle %q", publicID)
	}
	if tc.ID != "strong_auth_refresh_full_signal" {
		t.Fatalf("FindCase returned %q, want golden case", tc.ID)
	}
	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      publicID,
		Workspace:   "/tmp/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket did not accept public handle %q: %v", publicID, err)
	}
	if packet.CaseID != publicID || !strings.Contains(packet.Markdown, "Case: `"+publicID+"`") {
		t.Fatalf("BuildPacket did not render public handle %q:\n%s", publicID, packet.Markdown)
	}
}

func TestPacketAddsOperationSpecificNotes(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "inventory", id: "list_inventory_normal", want: "Inventory case: run list only; do not read or cite."},
		{name: "empty inventory", id: "list_inventory_empty", want: "Inventory case: run list only; do not read or cite."},
		{name: "deprecated", id: "list_hides_deprecated_by_default", want: "Deprecated visibility case: run default list, then run list with `--include-deprecated`."},
		{name: "workflow", id: "progressive_read_status_and_citation_guard", want: "Workflow case: find, read selected IDs, and cite only read-and-used IDs."},
		{name: "cli find", id: "interface_cli_find_matches_query", want: "Find case: run find, read selected IDs when useful, and cite only read-and-used IDs."},
		{name: "cli list", id: "interface_cli_list_matches_query", want: "Inventory case: run list only; do not read or cite."},
		{name: "cli read", id: "interface_cli_read_returns_body", want: "CLI read case: first discover and select an ID from public command output, then run `argos knowledge read --json <id>` for that selected ID."},
		{name: "cli cite", id: "interface_cli_cite_returns_citation", want: "CLI citation case: first discover and select read-and-used IDs from public command output, then run `argos knowledge cite --json <id>...` for those selected IDs."},
		{name: "context workflow", id: "context_entrypoint_carries_task_arguments", want: "Context workflow case: call context first, then use returned arguments for find before read/cite."},
		{name: "mcp", id: "interface_mcp_strict_schema", want: "MCP schema case: use JSON-RPC commands through `argos mcp`."},
		{name: "default find", id: "partial_domain_without_task_detail", want: "Find case: run find, read selected IDs when useful, and cite only read-and-used IDs."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := BuildPacket(cases, PacketOptions{CaseID: tt.id, Workspace: "/tmp/ws", ArgosBinary: "/tmp/argos"})
			if err != nil {
				t.Fatalf("BuildPacket returned error: %v", err)
			}
			if !strings.Contains(packet.Markdown, tt.want) {
				t.Fatalf("packet missing note %q:\n%s", tt.want, packet.Markdown)
			}
		})
	}
}

func TestCLIReadAndCitePacketsIncludeTaskOrQueryForFreshDiscovery(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	for _, id := range []string{"interface_cli_read_returns_body", "interface_cli_cite_returns_citation"} {
		t.Run(id, func(t *testing.T) {
			packet, err := BuildPacket(cases, PacketOptions{CaseID: id, Workspace: "/tmp/ws", ArgosBinary: "/tmp/argos"})
			if err != nil {
				t.Fatalf("BuildPacket returned error: %v", err)
			}
			if packet.Input.Task == "" && packet.Input.Query == "" {
				t.Fatalf("packet %s has no task/query for fresh public discovery:\n%s", id, packet.Markdown)
			}
		})
	}
}

func TestOperationNotesDoNotDependOnExpectedFields(t *testing.T) {
	deprecatedCase := discoverytest.Case{
		ID:        "list_hides_deprecated_by_default",
		Operation: "list",
		Input:     discoverytest.Input{Project: "mall-api"},
	}
	packet, err := BuildPacket([]discoverytest.Case{deprecatedCase}, PacketOptions{
		CaseID:      "list_hides_deprecated_by_default",
		Workspace:   "/tmp/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	if !strings.Contains(packet.Markdown, "Deprecated visibility case: run default list, then run list with `--include-deprecated`.") {
		t.Fatalf("deprecated note missing without expected fields:\n%s", packet.Markdown)
	}

	oracularCase := discoverytest.Case{
		ID:        "list_inventory_normal",
		Operation: "list",
		Input:     discoverytest.Input{Project: "mall-api"},
		Expected:  discoverytest.Expected{IncludeDeprecatedIDWhenRequested: "rule:backend.deprecated-auth.v1"},
	}
	packet, err = BuildPacket([]discoverytest.Case{oracularCase}, PacketOptions{
		CaseID:      "list_inventory_normal",
		Workspace:   "/tmp/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}
	if strings.Contains(packet.Markdown, "Deprecated visibility case") {
		t.Fatalf("deprecated note depended on expected fields:\n%s", packet.Markdown)
	}
}

func TestBuildPacketRejectsUnknownOperation(t *testing.T) {
	_, err := BuildPacket([]discoverytest.Case{{
		ID:        "unknown_operation_case",
		Operation: "surprise",
		Input:     discoverytest.Input{Project: "mall-api"},
	}}, PacketOptions{
		CaseID:      "unknown_operation_case",
		Workspace:   "/tmp/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err == nil {
		t.Fatal("BuildPacket returned nil error for unknown operation")
	}
	if strings.Contains(err.Error(), "unknown_operation_case") {
		t.Fatalf("unknown-operation error leaked golden ID: %v", err)
	}
}

func TestBuildPacketRejectsAmbiguousPublicHandleCollision(t *testing.T) {
	cases := []discoverytest.Case{
		{ID: "first_case", Operation: "find", Input: discoverytest.Input{Project: "mall-api"}},
		{ID: "case-001", Operation: "list", Input: discoverytest.Input{Project: "mall-api"}},
	}

	_, err := BuildPacket(cases, PacketOptions{
		CaseID:      "case-001",
		Workspace:   "/tmp/ws",
		ArgosBinary: "/tmp/argos",
	})
	if err == nil {
		t.Fatal("BuildPacket returned nil error for ambiguous case handle")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("BuildPacket returned non-ambiguity error: %v", err)
	}
}

func TestParseMarkdownReportExtractsStructuredFields(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(sampleStrongReport()))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	if report.CaseID != "progressive_read_status_and_citation_guard" {
		t.Fatalf("CaseID = %q, want legacy evaluator-compatible case ID", report.CaseID)
	}
	if report.ActualSupport != "strong" {
		t.Fatalf("ActualSupport = %q, want strong", report.ActualSupport)
	}
	if report.SearchStatus != "found exact backend auth refresh guidance" {
		t.Fatalf("SearchStatus = %q, want normalized status", report.SearchStatus)
	}
	assertStringSetContains(t, report.DiscoveredIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.ReadIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.CitedIDs, "rule:backend.auth-refresh.v1")
	if got := report.Guards["progressive reading"]; got != "pass" {
		t.Fatalf("progressive reading guard = %q, want pass", got)
	}
	if got := report.Guards["weak/none no-overclaim"]; got != "pass" {
		t.Fatalf("weak/none no-overclaim guard = %q, want pass", got)
	}
	if got := report.Guards["citation accountability"]; got != "pass" {
		t.Fatalf("citation accountability guard = %q, want pass", got)
	}
	if got := report.Guards["cited ids subset of read-and-used ids"]; got != "pass" {
		t.Fatalf("cited ids subset guard = %q, want pass", got)
	}
	if report.Result != "pass" {
		t.Fatalf("Result = %q, want pass", report.Result)
	}
	if len(report.MissingSections) != 0 {
		t.Fatalf("MissingSections = %v, want none", report.MissingSections)
	}
	if len(report.MissingFields) != 0 {
		t.Fatalf("MissingFields = %v, want none", report.MissingFields)
	}
}

func TestParseMarkdownReportAcceptsFreshRunnerStyleReport(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(`# Case: case-001

## Inputs

- Packet: /tmp/packet.md

## Tool Transcript Summary

- Ran list only.

## Observed Results

- Actual support: Argos returned inventory for project mall-api.
- Usage guidance: Argos reported read: forbidden, cite: forbidden, and claim: forbidden.
- Search status: none; this was a list-only inventory case.
- Discovered IDs: package:backend.auth-refresh.v1, rule:backend.auth-refresh.v1.
- Read IDs: none; reads were forbidden for this case.
- Cited IDs: none; citations were forbidden for this case.
- Missing needs: none; no task/query was provided beyond inventory listing.
- Argos-backed vs general reasoning: Argos-backed facts are limited to command output.
- Next steps: none for this case.

## Guards

- Progressive reading: pass; no knowledge body was read because the packet required list only.
- Weak/none no-overclaim: followed; no task claim was made from inventory alone.
- Citation accountability: yes; no citations were emitted.
- Cited IDs subset of read-and-used IDs: none; citation set is empty.
- Missing needs not cited: pass; no missing needs were cited.
- Attribution boundary: pass; observed facts are attributed to Argos command output.
- No Discovery-triggered upload/capture: pass; no upload or capture command was run.
- Usage guidance followed: pass; read, cite, and claim restrictions were followed.
- Context contamination: pass; no forbidden sources were inspected.

## Result

pass
`))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	if report.CaseID != "case-001" {
		t.Fatalf("CaseID = %q, want case-001", report.CaseID)
	}
	if report.ActualSupport != "inventory" {
		t.Fatalf("ActualSupport = %q, want inventory", report.ActualSupport)
	}
	if len(report.ReadIDs) != 0 {
		t.Fatalf("ReadIDs = %v, want none", report.ReadIDs)
	}
	if len(report.CitedIDs) != 0 {
		t.Fatalf("CitedIDs = %v, want none", report.CitedIDs)
	}
	if got := report.Guards["progressive reading"]; got != "pass" {
		t.Fatalf("progressive reading guard = %q, want pass", got)
	}
	if got := report.Guards["weak/none no-overclaim"]; got != "pass" {
		t.Fatalf("weak/none no-overclaim guard = %q, want pass", got)
	}
	if got := report.Guards["citation accountability"]; got != "pass" {
		t.Fatalf("citation accountability guard = %q, want pass", got)
	}
	if got := report.Guards["cited ids subset of read-and-used ids"]; got != "pass" {
		t.Fatalf("cited ids subset guard = %q, want pass", got)
	}
	if report.Result != "pass" {
		t.Fatalf("Result = %q, want pass", report.Result)
	}
	if len(report.MissingFields) != 0 {
		t.Fatalf("MissingFields = %v, want none", report.MissingFields)
	}
}

func TestParseMarkdownReportRecordsMissingSections(t *testing.T) {
	report, err := ParseMarkdownReport([]byte("Case: `case-001`\n\n## Result\n\nResult: pass\n"))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	for _, want := range []string{"Inputs", "Tool Transcript Summary", "Observed Results", "Guards"} {
		assertStringSetContains(t, report.MissingSections, want)
	}
	for _, want := range []string{"actual support", "progressive reading", "citation accountability", "usage guidance followed", "context contamination"} {
		assertStringSetContains(t, report.MissingFields, want)
	}
}

func TestParseMarkdownReportExtractsNestedIDBullets(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(`# Report

Case: case-001

## Inputs

## Tool Transcript Summary

## Observed Results

- Actual support: strong
- Discovered IDs:
  - ` + "`rule:backend.auth-refresh.v1`" + `
  - ` + "`decision:backend.session-renewal.v1`" + `
- Read IDs included:
  - ` + "`rule:backend.auth-refresh.v1`" + `
- Cited IDs:
  - ` + "`decision:backend.session-renewal.v1`" + `
- Missing needs:
  - follow-up owner

## Guards

- Progressive reading: pass
- Citation accountability: pass
- Usage guidance followed: pass
- Context contamination: pass

## Result

Result: pass
`))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	assertStringSetContains(t, report.DiscoveredIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.DiscoveredIDs, "decision:backend.session-renewal.v1")
	assertStringSetContains(t, report.ReadIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.CitedIDs, "decision:backend.session-renewal.v1")
	assertStringSetContains(t, report.MissingNeeds, "follow-up owner")
}

func TestParseMarkdownReportStopsNestedIDListAtUnlabeledTopLevelBullet(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(`# Report

Case: case-001

## Observed Results

- Discovered IDs:
  - ` + "`rule:backend.auth-refresh.v1`" + `
- Note from runner
  - ` + "`decision:backend.session-renewal.v1`" + `
`))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	if got, want := len(report.DiscoveredIDs), 1; got != want {
		t.Fatalf("len(DiscoveredIDs) = %d, want %d: %v", got, want, report.DiscoveredIDs)
	}
	assertStringSetContains(t, report.DiscoveredIDs, "rule:backend.auth-refresh.v1")
}

func TestParseMarkdownReportDedupesCommaSeparatedIDs(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(`# Report

Case: case-001

## Observed Results

- Discovered IDs: ` + "`rule:backend.auth-refresh.v1`" + `, rule:backend.auth-refresh.v1, decision:backend.session-renewal.v1
`))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	if got, want := len(report.DiscoveredIDs), 2; got != want {
		t.Fatalf("len(DiscoveredIDs) = %d, want %d: %v", got, want, report.DiscoveredIDs)
	}
	assertStringSetContains(t, report.DiscoveredIDs, "rule:backend.auth-refresh.v1")
	assertStringSetContains(t, report.DiscoveredIDs, "decision:backend.session-renewal.v1")
}

func TestParseMarkdownReportRecordsBlankResultField(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(`# Report

Case: case-001

## Inputs

## Tool Transcript Summary

## Observed Results

- Actual support: strong

## Guards

- Progressive reading: pass
- Citation accountability: pass
- Usage guidance followed: pass
- Context contamination: pass

## Result

Result:
`))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	assertStringSetContains(t, report.MissingFields, "result")
}

func TestEvaluateReportPassesStrongWorkflow(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	report, err := ParseMarkdownReport([]byte(sampleStrongReport()))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.CaseID != tc.ID {
		t.Fatalf("CaseID = %q, want %q", evaluation.CaseID, tc.ID)
	}
	if evaluation.Result != ResultPass {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultPass, evaluation.Findings)
	}
	if len(evaluation.Findings) != 0 {
		t.Fatalf("Findings = %#v, want none", evaluation.Findings)
	}
}

func TestEvaluateReportFailsUnreadCitation(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	report, err := ParseMarkdownReport([]byte(strings.Replace(sampleStrongReport(), "- Read IDs: `rule:backend.auth-refresh.v1`", "- Read IDs: none", 1)))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultFail, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "cited ID was not read")
}

func TestEvaluateReportReviewNeededForShapeDrift(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "weak_single_generic_term")
	report, err := ParseMarkdownReport([]byte("Case: weak_single_generic_term\n\n## Result\n\nResult: pass\n"))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultReviewNeeded, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "missing section")
	assertFindingContains(t, evaluation.Findings, "missing field")
}

func TestEvaluateCaseAcceptsPublicCaseHandle(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	publicID, ok := PublicCaseID(cases, "progressive_read_status_and_citation_guard")
	if !ok {
		t.Fatal("missing public case handle")
	}
	report, err := ParseMarkdownReport([]byte(strings.Replace(sampleStrongReport(), "progressive_read_status_and_citation_guard", publicID, 1)))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, publicID, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.CaseID != publicID {
		t.Fatalf("CaseID = %q, want public handle %q", evaluation.CaseID, publicID)
	}
	if evaluation.Result != ResultPass {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultPass, evaluation.Findings)
	}
}

func TestEvaluateReportReviewNeededForMalformedStrongReport(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	report, err := ParseMarkdownReport([]byte("Case: progressive_read_status_and_citation_guard\n\n## Result\n\nResult: pass\n"))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultReviewNeeded, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "missing section")
	for _, finding := range evaluation.Findings {
		if strings.Contains(finding.Message, "expected ID was not reported") {
			t.Fatalf("shape drift should not fail from absent ID evidence: %#v", evaluation.Findings)
		}
	}
}

func TestEvaluateReportReviewNeededForInvalidStatuses(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	text := strings.Replace(sampleStrongReport(), "Result: `pass`", "Result: `maybe`", 1)
	text = strings.Replace(text, "- Progressive reading: PASS.", "- Progressive reading: maybe", 1)
	report, err := ParseMarkdownReport([]byte(text))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultReviewNeeded {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultReviewNeeded, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "unknown result status")
	assertFindingContains(t, evaluation.Findings, "unknown guard status")
}

func TestEvaluateReportFailDominatesReviewNeeded(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	text := strings.Replace(sampleStrongReport(), "- Read IDs: `rule:backend.auth-refresh.v1`", "- Read IDs: none", 1)
	text = strings.Replace(text, "- Next steps: none\n", "", 1)
	report, err := ParseMarkdownReport([]byte(text))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultFail, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "missing field: next steps")
	assertFindingContains(t, evaluation.Findings, "cited ID was not read")
}

func TestEvaluateReportFailsAnyGuardFailure(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	text := strings.Replace(sampleStrongReport(), "- Weak/none no-overclaim: pass", "- Weak/none no-overclaim: fail", 1)
	report, err := ParseMarkdownReport([]byte(text))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultFail, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "guard failed: weak/none no-overclaim")
}

func TestEvaluateReportFailsWhenRunnerReportsFail(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	text := strings.Replace(sampleStrongReport(), "Result: `pass`", "Result: `fail`", 1)
	report, err := ParseMarkdownReport([]byte(text))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultFail, evaluation.Findings)
	}
	assertFindingContains(t, evaluation.Findings, "runner reported result: fail")
}

func TestEvaluateFindingsDoNotLeakExpectedValues(t *testing.T) {
	cases, err := LoadCases(goldenCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc := discoverytest.CaseByID(t, cases, "progressive_read_status_and_citation_guard")
	text := strings.Replace(sampleStrongReport(), "- Actual support: `strong`", "- Actual support: `weak`", 1)
	text = strings.Replace(text, "- Discovered IDs: `rule:backend.auth-refresh.v1`", "- Discovered IDs: none", 1)
	text = strings.Replace(text, "- Read IDs: `rule:backend.auth-refresh.v1`", "- Read IDs: none", 1)
	text = strings.Replace(text, "- Cited IDs: `rule:backend.auth-refresh.v1`", "- Cited IDs: none", 1)
	report, err := ParseMarkdownReport([]byte(text))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation := Evaluate(tc, report)

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want %q; findings: %#v", evaluation.Result, ResultFail, evaluation.Findings)
	}
	body, err := json.Marshal(evaluation.Findings)
	if err != nil {
		t.Fatalf("Marshal findings returned error: %v", err)
	}
	for _, forbidden := range []string{"strong", "expected support", "expected ID", "rule:backend.auth-refresh.v1", "decision:backend.session-renewal.v1"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("findings leaked %q: %s", forbidden, string(body))
		}
	}
}

func sampleStrongReport() string {
	return `# Argos Dogfood Runner Report

Case: ` + "`progressive_read_status_and_citation_guard`" + `

## Inputs

- Project: mall-api
- Phase: implementation
- Task: refresh backend auth tokens safely

## Tool Transcript Summary

- Ran ` + "`argos knowledge find --json --project mall-api --task \"refresh backend auth tokens safely\"`" + `
- Ran ` + "`argos knowledge read --json rule:backend.auth-refresh.v1`" + `
- Ran ` + "`argos knowledge cite --json rule:backend.auth-refresh.v1`" + `

## Observed Results

- Actual support: ` + "`strong`" + `
- Usage guidance: Followed the auth-refresh runbook and kept findings scoped to the cited rule.
- Search status: ` + "`FOUND exact backend auth refresh guidance.`" + `
- Discovered IDs: ` + "`rule:backend.auth-refresh.v1`" + `
- Read IDs: ` + "`rule:backend.auth-refresh.v1`" + `
- Cited IDs: ` + "`rule:backend.auth-refresh.v1`" + `
- Missing needs: none
- Argos-backed vs general reasoning: Argos-backed claim used only the cited rule; general reasoning stayed separate.
- Next steps: none

## Guards

- Progressive reading: PASS.
- Weak/none no-overclaim: pass
- Citation accountability: ` + "`PASS`" + `
- Cited IDs subset of read-and-used IDs: pass
- Missing needs not cited: pass
- Attribution boundary: pass
- No Discovery-triggered upload/capture: pass
- Usage guidance followed: pass
- Context contamination: pass

## Result

Result: ` + "`pass`" + `
`
}

func assertStringSetContains(t *testing.T, got []string, want string) {
	t.Helper()
	for _, value := range got {
		if value == want {
			return
		}
	}
	t.Fatalf("%v does not contain %q", got, want)
}

func assertFindingContains(t *testing.T, findings []Finding, want string) {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding.Message, want) {
			return
		}
	}
	t.Fatalf("%#v does not contain finding message %q", findings, want)
}
