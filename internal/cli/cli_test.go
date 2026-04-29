package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/index"
)

func TestRunPrintsHelpWithoutCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{}, &out, &out)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage: argos <command>") {
		t.Fatalf("expected usage output, got %q", out.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"unknown"}, &out, &out)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(out.String(), "unknown command: unknown") {
		t.Fatalf("expected unknown command error, got %q", out.String())
	}
}

func TestRunValidateReportsValidKnowledgeItems(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "validated 1 knowledge item(s)" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunIndexRebuildsKnowledgeIndex(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"index"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "indexed 1 knowledge item(s)" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	dbPath := filepath.Join(root, "argos", "index.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected index database at %s: %v", dbPath, err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("backend.auth.jwt-refresh-token.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "JWT refresh token handling convention" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}

func TestRunIndexRejectsInvalidKnowledgeWithoutReplacingExistingIndex(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"index"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected initial index exit code 0, got %d; stderr: %q", code, stderr.String())
	}

	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: backend.auth.jwt-refresh-token.v1
title: Invalid replacement
type: guide
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
This item should not be indexed.
`)
	stdout.Reset()
	stderr.Reset()

	code = Run([]string{"index"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected invalid index exit code 1, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "knowledge/items/backend/auth.md: unknown type: guide") {
		t.Fatalf("expected validation error in stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "validation failed with 1 error(s)") {
		t.Fatalf("expected validation summary in stderr, got %q", stderr.String())
	}

	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()
	got, err := store.GetItem("backend.auth.jwt-refresh-token.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if got.Title != "JWT refresh token handling convention" {
		t.Fatalf("expected existing index to remain unchanged, got title %q", got.Title)
	}
}

func TestRunInstallAdaptersGeneratesProjectFiles(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"install-adapters"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "installed adapters for 1 project(s)" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	path := filepath.Join(root, "argos", "generated", "mall-api", "AGENTS.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected generated AGENTS.md at %s: %v", path, err)
	}
}

func TestRunContextPrintsWorkflowContractJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"context", "--json", "--project", "mall-api"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		Project              string `json:"project"`
		RecommendedNextCalls []struct {
			Tool   string `json:"tool"`
			Reason string `json:"reason"`
		} `json:"recommended_next_calls"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON output, got error %v and output %q", err, stdout.String())
	}
	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", result.Project)
	}
	if len(result.RecommendedNextCalls) == 0 {
		t.Fatal("expected recommended next calls")
	}
}

func TestRunMCPHandlesToolsList(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	_, err = write.WriteString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n")
	if err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := write.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}
	previousStdin := os.Stdin
	os.Stdin = read
	t.Cleanup(func() {
		os.Stdin = previousStdin
		if err := read.Close(); err != nil {
			t.Fatalf("close stdin reader: %v", err)
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"mcp"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "argos_context") {
		t.Fatalf("expected MCP tools/list response, got %q", stdout.String())
	}
}

func TestRunMCPWritesJSONRPCErrorForMalformedInput(t *testing.T) {
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	_, err = write.WriteString(`{"jsonrpc":"2.0","id":1,"method":` + "\n")
	if err != nil {
		t.Fatalf("write stdin pipe: %v", err)
	}
	if err := write.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}
	previousStdin := os.Stdin
	os.Stdin = read
	t.Cleanup(func() {
		os.Stdin = previousStdin
		if err := read.Close(); err != nil {
			t.Fatalf("close stdin reader: %v", err)
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"mcp"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":-32700`) {
		t.Fatalf("expected JSON-RPC parse error response, got %q", stdout.String())
	}
}

func TestRunValidatePrintsValidationErrors(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: guide
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "knowledge/items/backend/auth.md: unknown type: guide") {
		t.Fatalf("expected validation error in stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "validation failed with 1 error(s)") {
		t.Fatalf("expected validation summary in stderr, got %q", stderr.String())
	}
}

func writeCLIFile(t *testing.T, root, rel, body string) {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory to %s: %v", previous, err)
		}
	})
}
