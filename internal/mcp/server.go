package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"argos/internal/query"
)

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
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result"`
}

type tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type resource struct {
	URI         string `json:"uri"`
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
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		if err := s.HandleLine(scanner.Bytes(), stdout); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) HandleLine(line []byte, stdout io.Writer) error {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return fmt.Errorf("parse JSON-RPC request: %w", err)
	}

	resp := response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  s.result(req.Method),
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal JSON-RPC response: %w", err)
	}
	_, err = fmt.Fprintln(stdout, string(encoded))
	return err
}

func (s *Server) result(method string) map[string]any {
	switch method {
	case "tools/list":
		return map[string]any{"tools": tools()}
	case "resources/list":
		return map[string]any{"resources": resources()}
	case "prompts/list":
		return map[string]any{"prompts": prompts()}
	default:
		return map[string]any{
			"message": "method not implemented",
			"method":  method,
		}
	}
}

func tools() []tool {
	return []tool{
		{Name: "argos_context", Description: "Get workflow context and recommended next calls."},
		{Name: "argos_requirements", Description: "Find relevant requirements for a task."},
		{Name: "argos_standards", Description: "Find active standards for project work."},
		{Name: "argos_risks", Description: "Find relevant risks, lessons, and incident history."},
		{Name: "argos_operations", Description: "Find operational runbooks and deployment guidance."},
		{Name: "get_knowledge_item", Description: "Fetch a knowledge item by id."},
		{Name: "cite_knowledge", Description: "Create a citation for a knowledge item."},
	}
}

func resources() []resource {
	return []resource{
		{URI: "argos://project/{project}/brief", Name: "project_brief", Description: "Project overview and current context."},
		{URI: "argos://project/{project}/rules", Name: "project_rules", Description: "Project-specific rules and standards."},
		{URI: "argos://domain/{domain}/map", Name: "domain_map", Description: "Domain knowledge map."},
		{URI: "argos://knowledge/{id}", Name: "knowledge_item", Description: "Knowledge item by id."},
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
