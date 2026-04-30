package mcp

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"argos/internal/discoverytest"
	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/query"
)

type testResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *testError      `json:"error"`
}

type testError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestServerHandlesToolsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertSuccessID(t, resp, "1")

	var result struct {
		Tools []struct {
			Name        string         `json:"name"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	decodeResult(t, resp, &result)

	for _, name := range []string{"argos_context", "argos_standards", "argos_discover", "argos_map", "get_knowledge_item", "cite_knowledge"} {
		tool := findTool(result.Tools, name)
		if tool == nil {
			t.Fatalf("expected %s tool in response: %s", name, out.String())
		}
		if tool.InputSchema == nil {
			t.Fatalf("expected %s tool to include inputSchema: %s", name, out.String())
		}
	}
	for _, name := range []string{"argos_requirements", "argos_risks", "argos_operations"} {
		if tool := findTool(result.Tools, name); tool != nil {
			t.Fatalf("did not expect unimplemented %s tool in response: %s", name, out.String())
		}
	}
}

func TestToolsListIncludesConcreteSchemasForImplementedTools(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	var result struct {
		Tools []struct {
			Name        string         `json:"name"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	decodeResult(t, resp, &result)

	assertToolSchemaHasProperties(t, result.Tools, "argos_context", []string{"project", "phase", "task", "files"})
	assertToolSchemaRequired(t, result.Tools, "argos_context", []string{"project", "phase", "task"})
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "argos_context")

	assertToolSchemaHasProperties(t, result.Tools, "argos_standards", []string{"project", "task_type", "files", "limit"})
	assertToolSchemaRequired(t, result.Tools, "argos_standards", []string{"project"})
	assertToolSchemaPropertyBounds(t, result.Tools, "argos_standards", "limit", 1, 5)
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "argos_standards")

	assertToolSchemaHasProperties(t, result.Tools, "argos_discover", []string{"project", "phase", "task", "query", "files", "types", "tags", "domains", "status", "include_deprecated", "limit"})
	assertToolSchemaLacksProperty(t, result.Tools, "argos_discover", "include_inbox")
	assertToolSchemaRequired(t, result.Tools, "argos_discover", []string{"project"})
	assertToolSchemaAnyOfRequiresOneOf(t, result.Tools, "argos_discover", []string{"task", "query"})
	assertToolSchemaPropertyBounds(t, result.Tools, "argos_discover", "limit", 1, 20)
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "argos_discover")

	assertToolSchemaHasProperties(t, result.Tools, "argos_map", []string{"project", "domain", "types", "include_deprecated"})
	assertToolSchemaLacksProperty(t, result.Tools, "argos_map", "include_inbox")
	assertToolSchemaRequired(t, result.Tools, "argos_map", []string{"project"})
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "argos_map")

	assertToolSchemaHasProperties(t, result.Tools, "get_knowledge_item", []string{"id"})
	assertToolSchemaRequired(t, result.Tools, "get_knowledge_item", []string{"id"})
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "get_knowledge_item")

	assertToolSchemaHasProperties(t, result.Tools, "cite_knowledge", []string{"ids"})
	assertToolSchemaRequired(t, result.Tools, "cite_knowledge", []string{"ids"})
	assertToolSchemaDisallowsAdditionalProperties(t, result.Tools, "cite_knowledge")
}

func TestToolCallUnknownToolReturnsInvalidParams(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"missing_tool","arguments":{}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertError(t, decodeRPCResponse(t, out.Bytes()), "1", -32602)
}

func TestToolCallMalformedParamsReturnsInvalidParams(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":123,"arguments":{}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertError(t, decodeRPCResponse(t, out.Bytes()), "1", -32602)
}

func TestToolCallMissingNameReturnsInvalidParams(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertError(t, decodeRPCResponse(t, out.Bytes()), "1", -32602)
}

func TestToolCallNonObjectArgumentsReturnInvalidParams(t *testing.T) {
	server := NewServer(query.New(nil))

	for _, tc := range []struct {
		name      string
		arguments string
	}{
		{name: "string", arguments: `"not an object"`},
		{name: "array", arguments: `[]`},
		{name: "null", arguments: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_context","arguments":` + tc.arguments + `}}`)

			err := server.HandleLine(line, &out)
			if err != nil {
				t.Fatalf("HandleLine returned error: %v", err)
			}

			assertError(t, decodeRPCResponse(t, out.Bytes()), "1", -32602)
		})
	}
}

func TestToolCallArgosContextWorksWithoutIndex(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_context","arguments":{"project":"mall-api","phase":"implementation","task":"add refresh token endpoint","files":["internal/auth/session.go"]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeRPCResponse(t, out.Bytes())
	result := resultMap(t, resp)
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}

	text := firstContentText(t, result)
	if !strings.Contains(text, `"project": "mall-api"`) {
		t.Fatalf("expected project in context response: %s", text)
	}
	if !strings.Contains(text, "argos_standards") {
		t.Fatalf("expected next call in context response: %s", text)
	}
}

func TestToolCallArgosContextInvalidArgsReturnsToolError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_context","arguments":{"project":"mall-api","unknown":true}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeRPCResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("expected tool error result, got rpc error: %#v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["isError"] != true {
		t.Fatalf("expected isError true, got %#v", result["isError"])
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, "invalid arguments for argos_context") {
		t.Fatalf("unexpected tool error text: %s", text)
	}
}

func TestToolCallArgosContextMissingRequiredArgsReturnsToolError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_context","arguments":{"project":"mall-api","phase":"implementation"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_context: task is required")
}

func TestToolCallArgosStandardsReturnsRuleSummaries(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_standards","arguments":{"project":"mall-api","files":["internal/auth/session.go"],"limit":5}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
	if !strings.Contains(text, `"id": "rule:backend.auth.v1"`) {
		t.Fatalf("expected auth rule summary: %s", text)
	}
	if strings.Contains(text, "Require explicit auth middleware for account endpoints.") && strings.Contains(text, "body") {
		t.Fatalf("standards should not return full body: %s", text)
	}
}

func TestToolCallArgosStandardsWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_standards","arguments":{"project":"mall-api"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] != true {
		t.Fatalf("expected tool error: %#v", result)
	}
}

func TestToolCallArgosStandardsMissingRequiredArgsReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_standards","arguments":{"files":["internal/auth/session.go"]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_standards: project is required")
}

func TestToolCallArgosStandardsExplicitLimitOutOfRangeReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	for _, limit := range []int{0, 6} {
		var out bytes.Buffer
		line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_standards","arguments":{"project":"mall-api","limit":` + strconv.Itoa(limit) + `}}}`)
		err := server.HandleLine(line, &out)
		if err != nil {
			t.Fatalf("HandleLine returned error for limit %d: %v", limit, err)
		}

		assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_standards: limit must be between 1 and 5")
	}
}

func TestToolCallArgosDiscoverReturnsRoutes(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","phase":"implementation","task":"add refresh token endpoint","query":"auth middleware refresh token","files":["internal/auth/session.go"],"limit":5}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"coverage"`) || !strings.Contains(text, `"items"`) {
		t.Fatalf("expected coverage and items in discover response: %s", text)
	}
	if !strings.Contains(text, `"action_policy"`) || !strings.Contains(text, `"authority": "strong"`) {
		t.Fatalf("expected action policy in discover response: %s", text)
	}
	if strings.Contains(text, `"body"`) {
		t.Fatalf("discover should not return full body: %s", text)
	}
}

func TestToolCallArgosMapReturnsInventory(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api","domain":"backend"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"inventory"`) {
		t.Fatalf("expected inventory in map response: %s", text)
	}
	if !strings.Contains(text, `"action_policy"`) || !strings.Contains(text, `"authority": "inventory"`) {
		t.Fatalf("expected inventory action policy in map response: %s", text)
	}
	if strings.Contains(text, `"body"`) {
		t.Fatalf("map should not return full body: %s", text)
	}
}

func TestToolCallArgosDiscoverWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","task":"add refresh token endpoint"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "index not available: run argos index first")
}

func TestToolCallArgosDiscoverMissingTaskAndQueryReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_discover: task or query is required")
}

func TestToolCallArgosDiscoverExplicitLimitOutOfRangeReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	for _, limit := range []int{0, 21} {
		var out bytes.Buffer
		line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","task":"add refresh token endpoint","limit":` + strconv.Itoa(limit) + `}}}`)
		err := server.HandleLine(line, &out)
		if err != nil {
			t.Fatalf("HandleLine returned error for limit %d: %v", limit, err)
		}

		assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_discover: limit must be between 1 and 20")
	}
}

func TestToolCallArgosDiscoverRejectsIncludeInboxArgument(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth","include_inbox":true}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_discover")
}

func TestToolCallArgosMapWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "index not available: run argos index first")
}

func TestToolCallArgosMapRejectsIncludeInboxArgument(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api","include_inbox":true}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for argos_map")
}

func TestGoldenMCPDiscoveryStrictSchema(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	server := NewServerWithStore(store)

	for _, tc := range []struct {
		name string
		line string
		want string
	}{
		{
			name: "discover unknown argument",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth","include_inbox":true}}}`,
			want: "unknown field",
		},
		{
			name: "discover missing task and query",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api"}}}`,
			want: "task or query is required",
		},
		{
			name: "discover bad limit",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth","limit":99}}}`,
			want: "limit must be between 1 and 20",
		},
		{
			name: "map unknown argument",
			line: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api","include_inbox":true}}}`,
			want: "unknown field",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := server.HandleLine([]byte(tc.line), &out); err != nil {
				t.Fatalf("HandleLine returned error: %v", err)
			}
			text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
			if !strings.Contains(text, tc.want) {
				t.Fatalf("expected %q in %s", tc.want, text)
			}
		})
	}
}

func TestToolCallGetKnowledgeItemReturnsFullBody(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_knowledge_item","arguments":{"id":"rule:backend.auth.v1"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"body"`) {
		t.Fatalf("expected body in item response: %s", text)
	}
	if !strings.Contains(text, `Require explicit auth middleware for account endpoints.\nThis is the full rule body.`) {
		t.Fatalf("expected full rule body: %s", text)
	}
}

func TestToolCallGetKnowledgeItemReturnsPackageEntrypoint(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_knowledge_item","arguments":{"id":"package:backend.redis.best-practices.v1"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"type": "package"`) || !strings.Contains(text, "## Load On Demand") {
		t.Fatalf("expected package entrypoint body: %s", text)
	}
}

func TestToolCallGetKnowledgeItemMissingRequiredArgsReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_knowledge_item","arguments":{}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for get_knowledge_item: id is required")
}

func TestToolCallCiteKnowledgeReturnsCitationsAndMissing(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cite_knowledge","arguments":{"ids":["rule:backend.auth.v1","missing.v1"]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] == true {
		t.Fatalf("expected success result: %#v", result)
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, `"citations"`) || !strings.Contains(text, `"missing"`) {
		t.Fatalf("expected citations and missing in response: %s", text)
	}
	if !strings.Contains(text, `"id": "rule:backend.auth.v1"`) {
		t.Fatalf("expected auth rule citation: %s", text)
	}
	if !strings.Contains(text, `"missing.v1"`) {
		t.Fatalf("expected missing id: %s", text)
	}
}

func TestToolCallCiteKnowledgeMissingRequiredArgsReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cite_knowledge","arguments":{}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for cite_knowledge: ids is required")
}

func TestToolCallCiteKnowledgeEmptyIDsReturnsToolError(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cite_knowledge","arguments":{"ids":[]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "invalid arguments for cite_knowledge: ids is required")
}

func TestToolCallGetKnowledgeItemWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_knowledge_item","arguments":{"id":"rule:backend.auth.v1"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] != true {
		t.Fatalf("expected tool error: %#v", result)
	}
}

func TestToolCallCiteKnowledgeWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cite_knowledge","arguments":{"ids":["rule:backend.auth.v1"]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	if result["isError"] != true {
		t.Fatalf("expected tool error: %#v", result)
	}
}

func TestServerHandlesResourcesTemplatesAndPromptsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":2,"method":"resources/templates/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine resources/templates/list returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertSuccessID(t, resp, "2")

	var templatesResult struct {
		ResourceTemplates []struct {
			URITemplate string `json:"uriTemplate"`
		} `json:"resourceTemplates"`
	}
	decodeResult(t, resp, &templatesResult)
	for _, uriTemplate := range []string{
		"argos://project/{project}/brief",
		"argos://project/{project}/rules",
		"argos://domain/{domain}/map",
		"argos://knowledge/{id}",
	} {
		if !containsTemplate(templatesResult.ResourceTemplates, uriTemplate) {
			t.Fatalf("expected resource template %s in response: %s", uriTemplate, out.String())
		}
	}

	out.Reset()
	err = server.HandleLine([]byte(`{"jsonrpc":"2.0","id":3,"method":"resources/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine resources/list returned error: %v", err)
	}
	resp = decodeResponse(t, out.Bytes())
	assertSuccessID(t, resp, "3")
	var resourcesResult struct {
		Resources []struct{} `json:"resources"`
	}
	decodeResult(t, resp, &resourcesResult)
	if resourcesResult.Resources == nil {
		t.Fatalf("expected resources/list to include resources array: %s", out.String())
	}

	out.Reset()
	err = server.HandleLine([]byte(`{"jsonrpc":"2.0","id":4,"method":"prompts/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine prompts/list returned error: %v", err)
	}
	resp = decodeResponse(t, out.Bytes())
	assertSuccessID(t, resp, "4")
	var promptsResult struct {
		Prompts []struct {
			Name string `json:"name"`
		} `json:"prompts"`
	}
	decodeResult(t, resp, &promptsResult)
	for _, name := range []string{"prepare_feature_work", "debug_with_lessons", "review_with_standards", "prepare_deployment"} {
		if !containsPrompt(promptsResult.Prompts, name) {
			t.Fatalf("expected prompt %s in response: %s", name, out.String())
		}
	}
}

func TestServerDoesNotRespondToNotifications(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected notification to be silent, got %q", out.String())
	}
}

func TestServerInitializeThenNotificationThenLists(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))
	input := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
			`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n" +
			`{"jsonrpc":"2.0","id":3,"method":"resources/templates/list","params":{}}` + "\n" +
			`{"jsonrpc":"2.0","id":4,"method":"prompts/list","params":{}}` + "\n")

	err := server.Serve(input, &out)
	if err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected four responses, got %d: %q", len(lines), out.String())
	}

	initResp := decodeResponse(t, []byte(lines[0]))
	assertSuccessID(t, initResp, "1")
	var initResult struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		Capabilities struct {
			Tools     map[string]any `json:"tools"`
			Resources map[string]any `json:"resources"`
			Prompts   map[string]any `json:"prompts"`
		} `json:"capabilities"`
	}
	decodeResult(t, initResp, &initResult)
	if initResult.ProtocolVersion != "2025-03-26" {
		t.Fatalf("unexpected protocolVersion: %q", initResult.ProtocolVersion)
	}
	if initResult.ServerInfo.Name != "argos" || initResult.ServerInfo.Version != "0.1.0" {
		t.Fatalf("unexpected serverInfo: %+v", initResult.ServerInfo)
	}
	if initResult.Capabilities.Tools == nil || initResult.Capabilities.Resources == nil || initResult.Capabilities.Prompts == nil {
		t.Fatalf("expected tools, resources, and prompts capabilities: %+v", initResult.Capabilities)
	}

	assertSuccessID(t, decodeResponse(t, []byte(lines[1])), "2")
	assertSuccessID(t, decodeResponse(t, []byte(lines[2])), "3")
	assertSuccessID(t, decodeResponse(t, []byte(lines[3])), "4")
}

func TestServerWritesJSONRPCErrorForParseError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertError(t, resp, "null", -32700)
}

func TestServerWritesJSONRPCErrorForInvalidRequest(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":5}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertError(t, resp, "5", -32600)
}

func TestServerWritesJSONRPCErrorForUnknownMethod(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":6,"method":"unknown/method"}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertError(t, resp, "6", -32601)
}

func TestServerContinuesAfterClientRequestError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))
	input := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"unknown/method"}` + "\n" +
		`{"jsonrpc":"2.0","id":8,"method":"tools/list","params":{}}` + "\n")

	err := server.Serve(input, &out)
	if err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two responses, got %d: %q", len(lines), out.String())
	}
	assertError(t, decodeResponse(t, []byte(lines[0])), "7", -32601)
	assertSuccessID(t, decodeResponse(t, []byte(lines[1])), "8")
}

func TestServerHandlesOversizedValidFrame(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))
	longTask := strings.Repeat("x", 70*1024)
	line := `{"jsonrpc":"2.0","id":9,"method":"tools/list","params":{"task":"` + longTask + `"}}` + "\n"

	err := server.Serve(strings.NewReader(line), &out)
	if err != nil {
		t.Fatalf("Serve returned error for oversized valid frame: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertSuccessID(t, resp, "9")
}

func TestServerReportsFrameTooLargeThenContinues(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))
	input := strings.NewReader(strings.Repeat("x", maxFrameSize+1) + "\n" +
		`{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}` + "\n")

	err := server.Serve(input, &out)
	if err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected frame error and next response, got %d: %q", len(lines), out.String())
	}
	assertError(t, decodeResponse(t, []byte(lines[0])), "null", -32600)
	assertSuccessID(t, decodeResponse(t, []byte(lines[1])), "10")
}

func TestServerReportsFrameTooLargeAtEOF(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.Serve(strings.NewReader(strings.Repeat("x", maxFrameSize+1)), &out)
	if err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	resp := decodeResponse(t, out.Bytes())
	assertError(t, resp, "null", -32600)
}

func decodeResponse(t *testing.T, body []byte) testResponse {
	t.Helper()

	var resp testResponse
	if err := json.Unmarshal(bytes.TrimSpace(body), &resp); err != nil {
		t.Fatalf("decode JSON-RPC response: %v; body=%q", err, string(body))
	}
	return resp
}

func decodeRPCResponse(t *testing.T, body []byte) testResponse {
	t.Helper()

	return decodeResponse(t, body)
}

func decodeResult(t *testing.T, resp testResponse, target any) {
	t.Helper()

	if len(resp.Result) == 0 {
		t.Fatalf("expected result, got response: %+v", resp)
	}
	if err := json.Unmarshal(resp.Result, target); err != nil {
		t.Fatalf("decode result: %v; result=%s", err, string(resp.Result))
	}
}

func resultMap(t *testing.T, resp testResponse) map[string]any {
	t.Helper()

	var result map[string]any
	decodeResult(t, resp, &result)
	return result
}

func firstContentText(t *testing.T, result map[string]any) string {
	t.Helper()

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content array, got %#v", result["content"])
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first content item to be an object, got %#v", content[0])
	}
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("expected first content text, got %#v", first["text"])
	}
	return text
}

func assertToolErrorContains(t *testing.T, body []byte, want string) {
	t.Helper()

	resp := decodeRPCResponse(t, body)
	if resp.Error != nil {
		t.Fatalf("expected tool error result, got rpc error: %#v", resp.Error)
	}
	result := resultMap(t, resp)
	if result["isError"] != true {
		t.Fatalf("expected isError true, got %#v", result["isError"])
	}
	text := firstContentText(t, result)
	if !strings.Contains(text, want) {
		t.Fatalf("expected tool error text to contain %q, got %q", want, text)
	}
}

func assertSuccessID(t *testing.T, resp testResponse, id string) {
	t.Helper()

	if resp.JSONRPC != "2.0" {
		t.Fatalf("unexpected jsonrpc version: %q", resp.JSONRPC)
	}
	if string(resp.ID) != id {
		t.Fatalf("expected id %s, got %s", id, string(resp.ID))
	}
	if resp.Error != nil {
		t.Fatalf("expected success response, got error: %+v", resp.Error)
	}
	if len(resp.Result) == 0 {
		t.Fatalf("expected result in response: %+v", resp)
	}
}

func assertError(t *testing.T, resp testResponse, id string, code int) {
	t.Helper()

	if resp.JSONRPC != "2.0" {
		t.Fatalf("unexpected jsonrpc version: %q", resp.JSONRPC)
	}
	if string(resp.ID) != id {
		t.Fatalf("expected id %s, got %s", id, string(resp.ID))
	}
	if resp.Error == nil {
		t.Fatalf("expected error response, got result: %s", string(resp.Result))
	}
	if resp.Error.Code != code {
		t.Fatalf("expected error code %d, got %+v", code, resp.Error)
	}
	if len(resp.Result) != 0 {
		t.Fatalf("expected no result in error response, got %s", string(resp.Result))
	}
}

func findTool(tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, name string) *struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
} {
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i]
		}
	}
	return nil
}

func assertToolSchemaHasProperty(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName, propertyName string) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s inputSchema properties object, got %#v", toolName, tool.InputSchema["properties"])
	}
	if _, ok := properties[propertyName]; !ok {
		t.Fatalf("expected %s inputSchema to include property %s, got %#v", toolName, propertyName, properties)
	}
}

func assertToolSchemaHasProperties(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName string, propertyNames []string) {
	t.Helper()

	for _, propertyName := range propertyNames {
		assertToolSchemaHasProperty(t, tools, toolName, propertyName)
	}
}

func assertToolSchemaLacksProperty(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName string, propertyName string) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s inputSchema properties object, got %#v", toolName, tool.InputSchema["properties"])
	}
	if _, ok := properties[propertyName]; ok {
		t.Fatalf("expected %s inputSchema to omit property %s, got %#v", toolName, propertyName, properties)
	}
}

func assertToolSchemaRequired(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName string, required []string) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	values, ok := tool.InputSchema["required"].([]any)
	if !ok {
		t.Fatalf("expected %s inputSchema required array, got %#v", toolName, tool.InputSchema["required"])
	}
	if len(values) != len(required) {
		t.Fatalf("expected %s required fields %v, got %#v", toolName, required, values)
	}
	for i, want := range required {
		got, ok := values[i].(string)
		if !ok || got != want {
			t.Fatalf("expected %s required fields %v, got %#v", toolName, required, values)
		}
	}
}

func assertToolSchemaPropertyBounds(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName, propertyName string, minimum, maximum int) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s inputSchema properties object, got %#v", toolName, tool.InputSchema["properties"])
	}
	property, ok := properties[propertyName].(map[string]any)
	if !ok {
		t.Fatalf("expected %s inputSchema property %s object, got %#v", toolName, propertyName, properties[propertyName])
	}
	if got := property["minimum"]; got != float64(minimum) {
		t.Fatalf("expected %s.%s minimum %d, got %#v", toolName, propertyName, minimum, got)
	}
	if got := property["maximum"]; got != float64(maximum) {
		t.Fatalf("expected %s.%s maximum %d, got %#v", toolName, propertyName, maximum, got)
	}
}

func assertToolSchemaAnyOfRequiresOneOf(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName string, propertyNames []string) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	anyOf, ok := tool.InputSchema["anyOf"].([]any)
	if !ok {
		t.Fatalf("expected %s inputSchema anyOf array, got %#v", toolName, tool.InputSchema["anyOf"])
	}
	if len(anyOf) != len(propertyNames) {
		t.Fatalf("expected %s anyOf to contain %v, got %#v", toolName, propertyNames, anyOf)
	}
	for i, propertyName := range propertyNames {
		option, ok := anyOf[i].(map[string]any)
		if !ok {
			t.Fatalf("expected %s anyOf option %d to be an object, got %#v", toolName, i, anyOf[i])
		}
		required, ok := option["required"].([]any)
		if !ok || len(required) != 1 {
			t.Fatalf("expected %s anyOf option %d required array, got %#v", toolName, i, option["required"])
		}
		got, ok := required[0].(string)
		if !ok || got != propertyName {
			t.Fatalf("expected %s anyOf option %d to require %q, got %#v", toolName, i, propertyName, required)
		}
	}
}

func assertToolSchemaDisallowsAdditionalProperties(t *testing.T, tools []struct {
	Name        string         `json:"name"`
	InputSchema map[string]any `json:"inputSchema"`
}, toolName string) {
	t.Helper()

	tool := findTool(tools, toolName)
	if tool == nil {
		t.Fatalf("expected %s tool", toolName)
	}
	if got := tool.InputSchema["additionalProperties"]; got != false {
		t.Fatalf("expected %s inputSchema additionalProperties false, got %#v", toolName, got)
	}
}

func containsTemplate(templates []struct {
	URITemplate string `json:"uriTemplate"`
}, uriTemplate string) bool {
	for _, template := range templates {
		if template.URITemplate == uriTemplate {
			return true
		}
	}
	return false
}

func containsPrompt(prompts []struct {
	Name string `json:"name"`
}, name string) bool {
	for _, prompt := range prompts {
		if prompt.Name == name {
			return true
		}
	}
	return false
}

func buildMCPTestStore(t *testing.T) *index.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/auth.md",
			ID:              "rule:backend.auth.v1",
			Title:           "Auth middleware",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
			UpdatedAt:       "2026-04-29",
			Body:            "Require explicit auth middleware for account endpoints.\nThis is the full rule body.",
		},
		{
			Path:        "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md",
			ID:          "package:backend.redis.best-practices.v1",
			Title:       "Redis Best Practices",
			Type:        "package",
			TechDomains: []string{"backend"},
			Status:      "active",
			Priority:    "should",
			UpdatedAt:   "2026-04-29",
			Body: `## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read this first.

## Load On Demand

- references/key-design.md
`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
