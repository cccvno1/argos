# Argos MCP Tool Calls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement MCP `tools/call` for the first usable Argos tool set: `argos_context`, `argos_standards`, `get_knowledge_item`, and `cite_knowledge`.

**Architecture:** Extend the existing MCP server with a tool dispatcher that wraps query/index results in MCP content responses. The CLI will open `argos/index.db` when starting `argos mcp`, while discovery and `argos_context` remain usable without an index.

**Tech Stack:** Go 1.23, existing `internal/mcp`, `internal/query`, `internal/index`, `encoding/json`, line-delimited JSON-RPC over stdio.

---

## File Structure

- Modify `internal/mcp/server.go`: add `tools/call` handling, tool argument decoding, result wrappers, tool schemas, and index-unavailable behavior.
- Modify `internal/mcp/server_test.go`: add structured tests for tool calls and schema discovery.
- Modify `internal/cli/cli.go`: open `argos/index.db` for `argos mcp` and pass index-backed services into the server.
- Modify `internal/cli/cli_test.go`: add CLI-level MCP behavior for missing and present index.
- Modify `internal/query/query.go`: add item and citation helper methods.
- Modify `internal/query/query_test.go`: add tests for item/citation helper methods.

## Task 1: MCP Tool Response Infrastructure

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing tests for tool response shapes**

Add tests to `internal/mcp/server_test.go`:

```go
func TestToolCallUnknownToolReturnsToolError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"missing_tool","arguments":{}}}`), &out)
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
	if !strings.Contains(text, "unknown tool: missing_tool") {
		t.Fatalf("unexpected tool error text: %s", text)
	}
}

func TestToolCallMalformedParamsReturnsToolError(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":123,"arguments":{}}}`), &out)
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
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/mcp
```

Expected: FAIL because `tools/call` is not implemented.

- [ ] **Step 3: Implement MCP tool result wrappers**

In `internal/mcp/server.go`, add:

```go
type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
```

Add helpers:

```go
func textResult(value any) (toolCallResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return toolCallResult{}, err
	}
	return toolCallResult{
		Content: []toolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

func textToolError(message string) toolCallResult {
	return toolCallResult{
		Content: []toolContent{{Type: "text", Text: message}},
		IsError: true,
	}
}
```

Add `tools/call` to `result` dispatch by calling a new method:

```go
case "tools/call":
	result, err := s.callTool(req.Params)
	if err != nil {
		return map[string]any{
			"content": []toolContent{{Type: "text", Text: err.Error()}},
			"isError": true,
		}, true
	}
	return result, true
```

Implement `callTool` for now with unknown-tool errors. It should decode `callToolParams`, require `name`, default empty `arguments` to `{}`, and return `textToolError("unknown tool: <name>")` for all names until Task 2.

- [ ] **Step 4: Run tests**

Run:

```bash
gofmt -w internal/mcp/server.go internal/mcp/server_test.go
go test ./internal/mcp
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat: add mcp tool call response handling"
```

## Task 2: Implement `argos_context`

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing `argos_context` test**

Add to `internal/mcp/server_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/mcp -run TestToolCallArgosContextWorksWithoutIndex -count=1
```

Expected: FAIL because `argos_context` is unknown.

- [ ] **Step 3: Implement `argos_context`**

In `callTool`, add case:

```go
case "argos_context":
	var req query.ContextRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_context: " + err.Error()), nil
	}
	return textResult(s.service.Context(req))
```

Add helper:

```go
func decodeArgs(data json.RawMessage, out any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		data = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
gofmt -w internal/mcp/server.go internal/mcp/server_test.go
go test ./internal/mcp
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat: implement argos context tool call"
```

## Task 3: Index-Backed MCP Server Wiring

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI test for index-backed MCP**

Add to `internal/cli/cli_test.go`:

```go
func TestRunMCPStartsWithoutIndexAndDiscoversTools(t *testing.T) {
	root := t.TempDir()
	withWorkingDir(t, root, func() {
		var stdout, stderr bytes.Buffer
		input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
			`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n")

		code := runWithIO([]string{"mcp"}, input, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "argos_context") {
			t.Fatalf("expected tools discovery, got %s", stdout.String())
		}
	})
}
```

Add this helper in `internal/cli/cli.go` so tests can provide stdin while the public `Run` signature stays stable:

```go
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	return run(args, os.Stdin, stdout, stderr)
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	// Move the current Run body here and replace os.Stdin usage with stdin.
}
```

In `internal/cli/cli_test.go`, add:

```go
func runWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return run(args, stdin, stdout, stderr)
}
```

- [ ] **Step 2: Run test**

Run:

```bash
go test ./internal/cli -run TestRunMCPStartsWithoutIndexAndDiscoversTools -count=1
```

Expected: FAIL until CLI can inject stdin or mcp wiring supports tests.

- [ ] **Step 3: Add index-aware MCP constructor**

In `internal/mcp/server.go`, extend `Server`:

```go
type Server struct {
	service *query.Service
	store   *index.Store
}

func NewServer(service *query.Service) *Server {
	return &Server{service: service}
}

func NewServerWithStore(store *index.Store) *Server {
	return &Server{service: query.New(store), store: store}
}
```

- [ ] **Step 4: Wire CLI `mcp` to open index if present**

In `internal/cli/cli.go`, update mcp command:

```go
root, err := os.Getwd()
if err != nil {
	fmt.Fprintf(stderr, "get current directory: %v\n", err)
	return 1
}
store, err := index.Open(filepath.Join(root, "argos/index.db"))
if err != nil {
	server := mcp.NewServer(query.New(nil))
	if err := server.Serve(stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "serve mcp: %v\n", err)
		return 1
	}
	return 0
}
defer store.Close()
server := mcp.NewServerWithStore(store)
if err := server.Serve(stdin, stdout); err != nil {
	fmt.Fprintf(stderr, "serve mcp: %v\n", err)
	return 1
}
return 0
```

Opening a missing index should not fail startup. Treat any open error as unavailable index for now.

- [ ] **Step 5: Run tests**

Run:

```bash
gofmt -w internal/mcp/server.go internal/cli/cli.go internal/cli/cli_test.go
go test ./internal/cli ./internal/mcp
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/server.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: wire mcp server to local index"
```

## Task 4: Implement `argos_standards`

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing tests**

Add tests:

```go
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
```

Add `buildMCPTestStore` helper using `index.Rebuild` with one rule item.

- [ ] **Step 2: Run tests**

Run:

```bash
go test ./internal/mcp -run 'TestToolCallArgosStandards' -count=1
```

Expected: FAIL because standards tool is not implemented.

- [ ] **Step 3: Implement `argos_standards`**

In `callTool`, add:

```go
case "argos_standards":
	if s.store == nil {
		return textToolError("index not available: run argos index first"), nil
	}
	var req query.StandardsRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_standards: " + err.Error()), nil
	}
	resp, err := s.service.Standards(req)
	if err != nil {
		return textToolError("query standards: " + err.Error()), nil
	}
	return textResult(resp)
```

- [ ] **Step 4: Run tests**

Run:

```bash
gofmt -w internal/mcp/server.go internal/mcp/server_test.go
go test ./internal/mcp
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat: implement argos standards tool call"
```

## Task 5: Implement Item And Citation Tool Calls

**Files:**
- Modify: `internal/query/query.go`
- Modify: `internal/query/query_test.go`
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Write failing query tests**

Add to `internal/query/query_test.go`:

```go
func TestGetKnowledgeItemReturnsFullBody(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	item, err := service.GetKnowledgeItem("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("GetKnowledgeItem returned error: %v", err)
	}
	if item.Body == "" {
		t.Fatal("expected full body")
	}
}

func TestCiteKnowledgeReportsMissingIDs(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	result := service.CiteKnowledge([]string{"rule:backend.auth.v1", "missing.v1"})
	if len(result.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(result.Citations))
	}
	if len(result.Missing) != 1 || result.Missing[0] != "missing.v1" {
		t.Fatalf("expected missing id, got %#v", result.Missing)
	}
}
```

- [ ] **Step 2: Implement query helper types**

In `internal/query/query.go`, add:

```go
type KnowledgeItemResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Path     string `json:"path"`
	Body     string `json:"body"`
}

type CitationResult struct {
	Citations []Citation `json:"citations"`
	Missing   []string   `json:"missing"`
}

type Citation struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Status string `json:"status"`
}
```

Implement:

```go
func (s *Service) GetKnowledgeItem(id string) (KnowledgeItemResult, error)
func (s *Service) CiteKnowledge(ids []string) CitationResult
```

`GetKnowledgeItem` uses `store.GetItem`. `CiteKnowledge` calls `store.GetItem` for each ID and treats errors as missing.

- [ ] **Step 3: Add MCP tests**

Add tests for `get_knowledge_item` and `cite_knowledge` in `internal/mcp/server_test.go`, using `buildMCPTestStore`.

Assertions:

- `get_knowledge_item` returns `"body"` and the full rule body.
- `cite_knowledge` returns `"citations"` and `"missing"`.
- both return tool errors without index.

- [ ] **Step 4: Implement MCP tool cases**

In `callTool`, add cases:

```go
case "get_knowledge_item":
	if s.store == nil { return textToolError("index not available: run argos index first"), nil }
	var req struct { ID string `json:"id"` }
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for get_knowledge_item: " + err.Error()), nil
	}
	if strings.TrimSpace(req.ID) == "" {
		return textToolError("invalid arguments for get_knowledge_item: id is required"), nil
	}
	item, err := s.service.GetKnowledgeItem(req.ID)
	if err != nil { return textToolError("get knowledge item: " + err.Error()), nil }
	return textResult(item)

case "cite_knowledge":
	if s.store == nil { return textToolError("index not available: run argos index first"), nil }
	var req struct { IDs []string `json:"ids"` }
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for cite_knowledge: " + err.Error()), nil
	}
	return textResult(s.service.CiteKnowledge(req.IDs))
```

- [ ] **Step 5: Run tests**

Run:

```bash
gofmt -w internal/query/query.go internal/query/query_test.go internal/mcp/server.go internal/mcp/server_test.go
go test ./internal/query ./internal/mcp
go test ./...
go vet ./...
go mod tidy -diff
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/query/query.go internal/query/query_test.go internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat: implement knowledge item and citation tool calls"
```

## Task 6: Tool Schemas And Final Verification

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/server_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write failing schema test**

Add to `internal/mcp/server_test.go`:

```go
func TestToolsListIncludesConcreteSchemasForImplementedTools(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	result := resultMap(t, decodeRPCResponse(t, out.Bytes()))
	toolsValue, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools array: %#v", result["tools"])
	}
	assertToolSchemaHasProperty(t, toolsValue, "argos_context", "project")
	assertToolSchemaHasProperty(t, toolsValue, "argos_standards", "files")
	assertToolSchemaHasProperty(t, toolsValue, "get_knowledge_item", "id")
	assertToolSchemaHasProperty(t, toolsValue, "cite_knowledge", "ids")
}
```

Add helper `assertToolSchemaHasProperty`.

- [ ] **Step 2: Implement concrete input schemas**

Replace the shared empty schema in `tools()` with per-tool schemas.

Minimum helpers:

```go
func objectSchema(properties map[string]any, required []string) map[string]any
func stringProperty(description string) map[string]any
func stringArrayProperty(description string) map[string]any
func integerProperty(description string, minimum int, maximum int) map[string]any
```

Implemented tool schemas:

- `argos_context`: project, phase, task, files
- `argos_standards`: project, task_type, files, limit
- `get_knowledge_item`: id
- `cite_knowledge`: ids

Descriptions for unimplemented tools should include `Not implemented in this MVP phase`.

- [ ] **Step 3: Update README**

Add a short MCP section:

```md
## MCP

Start the server from a workspace root:

```bash
argos mcp
```

Implemented tool calls:

- `argos_context`
- `argos_standards`
- `get_knowledge_item`
- `cite_knowledge`
```

- [ ] **Step 4: Run full verification**

Run:

```bash
gofmt -w internal/mcp/server.go internal/mcp/server_test.go
go test ./...
go vet ./...
go test -race ./...
go build -o /tmp/argos-mcp-tool-calls ./cmd/argos
git diff --check
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go README.md
git commit -m "docs: document mcp tool calls"
```

## Self-Review

Spec coverage:

- `tools/call` dispatcher: Task 1.
- `argos_context`: Task 2.
- CLI index-backed MCP runtime: Task 3.
- `argos_standards`: Task 4.
- `get_knowledge_item` and `cite_knowledge`: Task 5.
- concrete schemas and README: Task 6.
- index-absent behavior: Tasks 3, 4, and 5.
- malformed tool arguments and unknown tools: Task 1.

Out of scope:

- full requirements/risks/operations implementations.
- MCP resource reads.
- MCP prompt gets.
- semantic search.
