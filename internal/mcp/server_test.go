package mcp

import (
	"bytes"
	"strings"
	"testing"

	"argos/internal/query"
)

func TestServerHandlesToolsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}
	if !strings.Contains(out.String(), "argos_context") {
		t.Fatalf("expected argos_context tool in response: %s", out.String())
	}
}

func TestServerHandlesResourcesAndPromptsList(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(query.New(nil))

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine resources/list returned error: %v", err)
	}
	if !strings.Contains(out.String(), "argos://project/{project}/brief") {
		t.Fatalf("expected project brief resource in response: %s", out.String())
	}

	out.Reset()
	err = server.HandleLine([]byte(`{"jsonrpc":"2.0","id":3,"method":"prompts/list","params":{}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine prompts/list returned error: %v", err)
	}
	if !strings.Contains(out.String(), "prepare_feature_work") {
		t.Fatalf("expected prepare_feature_work prompt in response: %s", out.String())
	}
}
