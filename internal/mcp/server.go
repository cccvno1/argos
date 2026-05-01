package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"argos/internal/index"
	"argos/internal/query"
)

const maxFrameSize = 1024 * 1024

var errFrameTooLarge = errors.New("mcp frame too large")

type Server struct {
	service *query.Service
	store   *index.Store
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

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

type resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type resourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type prompt struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func NewServer(service *query.Service) *Server {
	return &Server{service: service}
}

func NewServerWithStore(store *index.Store) *Server {
	return &Server{
		service: query.New(store),
		store:   store,
	}
}

func (s *Server) Serve(stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	for {
		line, err := readLineBounded(reader, maxFrameSize)
		if errors.Is(err, errFrameTooLarge) {
			if writeErr := writeError(stdout, nil, -32600, "Invalid Request"); writeErr != nil {
				return writeErr
			}
			continue
		}
		if len(line) > 0 {
			if err := s.HandleLine(line, stdout); err != nil {
				return err
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func readLineBounded(reader *bufio.Reader, maxSize int) ([]byte, error) {
	var buf bytes.Buffer

	for {
		fragment, isPrefix, err := reader.ReadLine()
		if len(fragment) > 0 {
			if buf.Len()+len(fragment) > maxSize {
				discardLineRemainder(reader, isPrefix)
				return nil, errFrameTooLarge
			}
			buf.Write(fragment)
		}
		if err != nil {
			if err == io.EOF && buf.Len() > 0 {
				return buf.Bytes(), nil
			}
			return nil, err
		}
		if !isPrefix {
			return buf.Bytes(), nil
		}
	}
}

func discardLineRemainder(reader *bufio.Reader, hasMore bool) {
	for hasMore {
		_, isPrefix, err := reader.ReadLine()
		if err != nil {
			return
		}
		hasMore = isPrefix
	}
}

func (s *Server) HandleLine(line []byte, stdout io.Writer) error {
	if len(bytes.TrimSpace(line)) == 0 {
		return nil
	}

	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return writeError(stdout, nil, -32700, "Parse error")
	}

	if req.JSONRPC != "2.0" || req.Method == "" {
		return writeError(stdout, req.ID, -32600, "Invalid Request")
	}

	result, rpcErr, ok := s.result(req.Method, req.Params)
	if !ok {
		if len(req.ID) == 0 {
			return nil
		}
		return writeError(stdout, req.ID, -32601, "Method not found")
	}
	if len(req.ID) == 0 {
		return nil
	}
	if rpcErr != nil {
		return writeError(stdout, req.ID, rpcErr.Code, rpcErr.Message)
	}

	resp := response{JSONRPC: "2.0", ID: req.ID, Result: result}
	encoded, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC response: %w", err)
	}
	_, err = fmt.Fprintln(stdout, string(encoded))
	return err
}

func (s *Server) result(method string, params json.RawMessage) (any, *rpcError, bool) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2025-03-26",
			"serverInfo": map[string]any{
				"name":    "argos",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
				"prompts":   map[string]any{},
			},
		}, nil, true
	case "tools/list":
		return map[string]any{"tools": tools()}, nil, true
	case "tools/call":
		result, rpcErr, err := s.callTool(params)
		if rpcErr != nil {
			return nil, rpcErr, true
		}
		if err != nil {
			return textToolError(err.Error()), nil, true
		}
		return result, nil, true
	case "resources/list":
		return map[string]any{"resources": resources()}, nil, true
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": resourceTemplates()}, nil, true
	case "prompts/list":
		return map[string]any{"prompts": prompts()}, nil, true
	default:
		return nil, nil, false
	}
}

func (s *Server) callTool(data json.RawMessage) (toolCallResult, *rpcError, error) {
	var params callToolParams
	if err := decodeStrict(data, &params); err != nil {
		return toolCallResult{}, invalidParams("invalid tools/call params: " + err.Error()), nil
	}
	if params.Name == "" {
		return toolCallResult{}, invalidParams("missing tool name"), nil
	}
	if err := requireObjectArguments(params.Arguments); err != nil {
		return toolCallResult{}, invalidParams("invalid tools/call params: " + err.Error()), nil
	}

	switch params.Name {
	case "argos_context":
		var req query.ContextRequest
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_context: " + err.Error()), nil, nil
		}
		if err := requireStringFields(map[string]string{
			"project": req.Project,
			"phase":   req.Phase,
			"task":    req.Task,
		}, "project", "phase", "task"); err != nil {
			return textToolError("invalid arguments for argos_context: " + err.Error()), nil, nil
		}
		result, err := textResult(s.service.Context(req))
		return result, nil, err
	case "argos_standards":
		var req query.StandardsRequest
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_standards: " + err.Error()), nil, nil
		}
		if err := requireStringFields(map[string]string{"project": req.Project}, "project"); err != nil {
			return textToolError("invalid arguments for argos_standards: " + err.Error()), nil, nil
		}
		limitProvided, err := hasArgument(params.Arguments, "limit")
		if err != nil {
			return textToolError("invalid arguments for argos_standards: " + err.Error()), nil, nil
		}
		if limitProvided && (req.Limit < 1 || req.Limit > 5) {
			return textToolError("invalid arguments for argos_standards: limit must be between 1 and 5"), nil, nil
		}
		if s.store == nil {
			return textToolError("index not available: run argos index first"), nil, nil
		}
		resp, err := s.service.Standards(req)
		if err != nil {
			return textToolError("query standards: " + err.Error()), nil, nil
		}
		result, err := textResult(resp)
		return result, nil, err
	case "argos_find_knowledge":
		var req query.FindKnowledgeRequest
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_find_knowledge: " + err.Error()), nil, nil
		}
		if err := requireStringFields(map[string]string{"project": req.Project}, "project"); err != nil {
			return textToolError("invalid arguments for argos_find_knowledge: " + err.Error()), nil, nil
		}
		if strings.TrimSpace(req.Task) == "" && strings.TrimSpace(req.Query) == "" {
			return textToolError("invalid arguments for argos_find_knowledge: task or query is required"), nil, nil
		}
		limitProvided, err := hasArgument(params.Arguments, "limit")
		if err != nil {
			return textToolError("invalid arguments for argos_find_knowledge: " + err.Error()), nil, nil
		}
		if limitProvided && (req.Limit < 1 || req.Limit > 20) {
			return textToolError("invalid arguments for argos_find_knowledge: limit must be between 1 and 20"), nil, nil
		}
		if s.store == nil {
			return textToolError("index not available: run argos index first"), nil, nil
		}
		resp, err := s.service.FindKnowledge(req)
		if err != nil {
			return textToolError("find knowledge: " + err.Error()), nil, nil
		}
		result, err := textResult(resp)
		return result, nil, err
	case "argos_list_knowledge":
		var req query.ListKnowledgeRequest
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_list_knowledge: " + err.Error()), nil, nil
		}
		if err := requireStringFields(map[string]string{"project": req.Project}, "project"); err != nil {
			return textToolError("invalid arguments for argos_list_knowledge: " + err.Error()), nil, nil
		}
		if s.store == nil {
			return textToolError("index not available: run argos index first"), nil, nil
		}
		resp, err := s.service.ListKnowledge(req)
		if err != nil {
			return textToolError("list knowledge: " + err.Error()), nil, nil
		}
		result, err := textResult(resp)
		return result, nil, err
	case "argos_read_knowledge":
		var req struct {
			ID string `json:"id"`
		}
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_read_knowledge: " + err.Error()), nil, nil
		}
		req.ID = strings.TrimSpace(req.ID)
		if req.ID == "" {
			return textToolError("invalid arguments for argos_read_knowledge: id is required"), nil, nil
		}
		if s.store == nil {
			return textToolError("index not available: run argos index first"), nil, nil
		}
		item, err := s.service.ReadKnowledge(req.ID)
		if err != nil {
			return textToolError("read knowledge: " + err.Error()), nil, nil
		}
		result, err := textResult(item)
		return result, nil, err
	case "argos_cite_knowledge":
		var req struct {
			IDs []string `json:"ids"`
		}
		if err := decodeArgs(params.Arguments, &req); err != nil {
			return textToolError("invalid arguments for argos_cite_knowledge: " + err.Error()), nil, nil
		}
		if len(req.IDs) == 0 {
			return textToolError("invalid arguments for argos_cite_knowledge: ids is required"), nil, nil
		}
		if s.store == nil {
			return textToolError("index not available: run argos index first"), nil, nil
		}
		result, err := textResult(s.service.CiteKnowledge(req.IDs))
		return result, nil, err
	default:
		return toolCallResult{}, invalidParams("unknown tool: " + params.Name), nil
	}
}

func decodeArgs(data json.RawMessage, out any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		data = json.RawMessage("{}")
	}
	if err := decodeStrict(data, out); err != nil {
		return err
	}
	return nil
}

func requireObjectArguments(data json.RawMessage) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}

	var args map[string]json.RawMessage
	if err := json.Unmarshal(data, &args); err != nil {
		return fmt.Errorf("arguments must be an object")
	}
	if args == nil {
		return fmt.Errorf("arguments must be an object")
	}
	return nil
}

func decodeStrict(data json.RawMessage, out any) error {
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

func hasArgument(data json.RawMessage, name string) (bool, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return false, nil
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(data, &args); err != nil {
		return false, err
	}
	_, ok := args[name]
	return ok, nil
}

func requireStringFields(values map[string]string, names ...string) error {
	for _, name := range names {
		if strings.TrimSpace(values[name]) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

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

func invalidParams(message string) *rpcError {
	return &rpcError{Code: -32602, Message: message}
}

func tools() []tool {
	return []tool{
		{
			Name:        "argos_context",
			Description: "Get workflow context and recommended next steps.",
			InputSchema: objectSchema(map[string]any{
				"project": stringProperty("Project identifier."),
				"phase":   stringProperty("Workflow phase, such as implementation, review, debugging, or deployment."),
				"task":    stringProperty("Current task description."),
				"files":   stringArrayProperty("Files relevant to the current task."),
			}, []string{"project", "phase", "task"}),
		},
		{
			Name:        "argos_standards",
			Description: "Find active standards for project work.",
			InputSchema: objectSchema(map[string]any{
				"project":   stringProperty("Project identifier."),
				"task_type": stringProperty("Type of work being performed."),
				"files":     stringArrayProperty("Files relevant to the current task."),
				"limit":     integerProperty("Maximum number of standards to return.", 1, 5),
			}, []string{"project"}),
		},
		{
			Name:        "argos_find_knowledge",
			Description: "Find relevant knowledge for current work.",
			InputSchema: findKnowledgeInputSchema(),
		},
		{
			Name:        "argos_list_knowledge",
			Description: "List available knowledge inventory for a project.",
			InputSchema: objectSchema(map[string]any{
				"project":            stringProperty("Project identifier."),
				"domain":             stringProperty("Domain filter."),
				"types":              stringArrayProperty("Knowledge item types to include."),
				"include_deprecated": booleanProperty("Include deprecated knowledge items."),
			}, []string{"project"}),
		},
		{
			Name:        "argos_read_knowledge",
			Description: "Read a knowledge item by id.",
			InputSchema: objectSchema(map[string]any{
				"id": stringProperty("Knowledge item id."),
			}, []string{"id"}),
		},
		{
			Name:        "argos_cite_knowledge",
			Description: "Create citations for knowledge items.",
			InputSchema: objectSchema(map[string]any{
				"ids": stringArrayProperty("Knowledge item ids to cite."),
			}, []string{"ids"}),
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func findKnowledgeInputSchema() map[string]any {
	schema := objectSchema(map[string]any{
		"project":            stringProperty("Project identifier."),
		"phase":              stringProperty("Workflow phase."),
		"task":               stringProperty("Current task description."),
		"query":              stringProperty("Search query."),
		"files":              stringArrayProperty("Files relevant to the current task."),
		"types":              stringArrayProperty("Knowledge item types to include."),
		"tags":               stringArrayProperty("Tags to include."),
		"domains":            stringArrayProperty("Domains to include."),
		"status":             stringArrayProperty("Statuses to include."),
		"include_deprecated": booleanProperty("Include deprecated knowledge items."),
		"limit":              integerProperty("Maximum number of knowledge results to return.", 1, 20),
	}, []string{"project"})
	schema["anyOf"] = []map[string]any{
		{"required": []string{"task"}},
		{"required": []string{"query"}},
	}
	return schema
}

func stringProperty(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

func stringArrayProperty(description string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": "string",
		},
	}
}

func integerProperty(description string, minimum, maximum int) map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": description,
		"minimum":     minimum,
		"maximum":     maximum,
	}
}

func booleanProperty(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}

func resources() []resource {
	return []resource{}
}

func resourceTemplates() []resourceTemplate {
	return []resourceTemplate{
		{URITemplate: "argos://project/{project}/brief", Name: "project_brief", Description: "Project overview and current context."},
		{URITemplate: "argos://project/{project}/rules", Name: "project_rules", Description: "Project-specific rules and standards."},
		{URITemplate: "argos://domain/{domain}/map", Name: "domain_map", Description: "Domain knowledge map."},
		{URITemplate: "argos://knowledge/{id}", Name: "knowledge_item", Description: "Knowledge item by id."},
	}
}

func prompts() []prompt {
	return []prompt{
		{Name: "prepare_feature_work", Description: "Prepare context before feature implementation."},
		{Name: "debug_with_lessons", Description: "Debug using known lessons and risks."},
		{Name: "review_with_standards", Description: "Review changes against active standards."},
		{Name: "prepare_deployment", Description: "Prepare deployment with operational guidance."},
	}
}

func writeError(stdout io.Writer, id json.RawMessage, code int, message string) error {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	resp := response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
		},
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC error response: %w", err)
	}
	_, err = fmt.Fprintln(stdout, string(encoded))
	return err
}
