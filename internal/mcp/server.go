package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"argos/internal/query"
)

const maxFrameSize = 1024 * 1024

type Server struct {
	service *query.Service
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

func (s *Server) Serve(stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if len(line) > maxFrameSize {
				if writeErr := writeError(stdout, nil, -32600, "Invalid Request"); writeErr != nil {
					return writeErr
				}
			} else if err := s.HandleLine(line, stdout); err != nil {
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

	result, ok := s.result(req.Method)
	if !ok {
		if len(req.ID) == 0 {
			return nil
		}
		return writeError(stdout, req.ID, -32601, "Method not found")
	}
	if len(req.ID) == 0 {
		return nil
	}

	resp := response{JSONRPC: "2.0", ID: req.ID, Result: result}
	encoded, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC response: %w", err)
	}
	_, err = fmt.Fprintln(stdout, string(encoded))
	return err
}

func (s *Server) result(method string) (map[string]any, bool) {
	switch method {
	case "tools/list":
		return map[string]any{"tools": tools()}, true
	case "resources/list":
		return map[string]any{"resources": resources()}, true
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": resourceTemplates()}, true
	case "prompts/list":
		return map[string]any{"prompts": prompts()}, true
	default:
		return nil, false
	}
}

func tools() []tool {
	inputSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	return []tool{
		{Name: "argos_context", Description: "Get workflow context and recommended next calls.", InputSchema: inputSchema},
		{Name: "argos_requirements", Description: "Find relevant requirements for a task.", InputSchema: inputSchema},
		{Name: "argos_standards", Description: "Find active standards for project work.", InputSchema: inputSchema},
		{Name: "argos_risks", Description: "Find relevant risks, lessons, and incident history.", InputSchema: inputSchema},
		{Name: "argos_operations", Description: "Find operational runbooks and deployment guidance.", InputSchema: inputSchema},
		{Name: "get_knowledge_item", Description: "Fetch a knowledge item by id.", InputSchema: inputSchema},
		{Name: "cite_knowledge", Description: "Create a citation for a knowledge item.", InputSchema: inputSchema},
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
