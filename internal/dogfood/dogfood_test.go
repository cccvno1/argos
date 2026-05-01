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
		{name: "mcp", id: "interface_mcp_strict_schema", want: "MCP schema case: use JSON-RPC commands through `argos mcp`."},
		{name: "adapter", id: "adapter_flow_recommendations", want: "Adapter case: run `argos install-adapters` before checking generated files."},
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
