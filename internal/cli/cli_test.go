package cli

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestRunValidatePathValidatesSingleInboxPackage(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "validated 1 knowledge item(s)") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunValidateInboxValidatesInboxOnly(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	writeCLIFile(t, root, "knowledge/packages/backend/broken/KNOWLEDGE.md", `---
id: package:backend.broken.v1
title: Broken
type: package
status: active
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate", "--inbox"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "validated 1 knowledge item(s)") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunValidateDefaultIgnoresInboxCandidates(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", validCLIKnowledgeItem("rule:backend.auth.v1", "rule"))
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/broken/KNOWLEDGE.md", `---
id: package:backend.broken.v1
title: Broken
type: package
status: draft
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "validated 1 knowledge item(s)" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
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

func TestRunIndexIncludesOfficialPackages(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"index"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}

	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	item, err := store.GetItem("package:backend.redis.best-practices.v1")
	if err != nil {
		t.Fatalf("expected indexed package: %v", err)
	}
	if item.Type != "package" {
		t.Fatalf("expected package type, got %q", item.Type)
	}
	if !strings.Contains(item.Body, "## Load On Demand") {
		t.Fatalf("expected package body, got %q", item.Body)
	}
}

func TestRunKnowledgeFindReturnsJSONResults(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "find", "--json", "--project", "mall-api", "--phase", "implementation", "--task", "add refresh token endpoint", "--query", "refresh token", "--files", "internal/auth/session.go"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Support struct {
			Level string `json:"level"`
		} `json:"support"`
		Items []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Support.Level != "strong" {
		t.Fatalf("expected strong coverage: %s", stdout.String())
	}
	if len(result.Items) == 0 || result.Items[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Full body implementation detail") {
		t.Fatalf("knowledge find should not print full body: %s", stdout.String())
	}
}

func TestRunKnowledgeFindAcceptsRepeatedFiles(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "find", "--json", "--project", "mall-api", "--phase", "implementation", "--task", "add auth middleware", "--query", "auth", "--files", "   ", "--files", " internal/auth/session.go "}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Items []struct {
			ID              string `json:"id"`
			ScoreComponents struct {
				FileScope float64 `json:"file_scope"`
			} `json:"score_components"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Items) == 0 || result.Items[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule from repeated --files flags: %s", stdout.String())
	}
	if result.Items[0].ScoreComponents.FileScope != 1 {
		t.Fatalf("expected trimmed repeated --files to match file scope: %s", stdout.String())
	}
}

func TestRunKnowledgeFindAcceptsDiscoveryFiltersAndLimit(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"knowledge", "find", "--json",
		"--project", "mall-api",
		"--query", "auth",
		"--types", "rule",
		"--tags", "missing",
		"--tags", "auth",
		"--domains", "missing",
		"--domains", "security",
		"--status", "active",
		"--include-deprecated",
		"--limit", "1",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Items) != 1 || result.Items[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected one filtered auth rule from CLI flags, got %s", stdout.String())
	}
}

func TestRunKnowledgeFindRejectsMissingProject(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "find", "--json", "--query", "auth"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "knowledge find: --project is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunKnowledgeFindRejectsMissingTaskAndQuery(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "find", "--json", "--project", "mall-api"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "knowledge find: --task or --query is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunKnowledgeFindRejectsExplicitLimitOutOfRange(t *testing.T) {
	for _, limit := range []string{"0", "21"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run([]string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "auth", "--limit", limit}, &stdout, &stderr)

		if code != 2 {
			t.Fatalf("expected exit code 2 for limit %s, got %d", limit, code)
		}
		if !strings.Contains(stderr.String(), "knowledge find: --limit must be between 1 and 20") {
			t.Fatalf("unexpected stderr for limit %s: %q", limit, stderr.String())
		}
	}
}

func TestRunKnowledgeListReturnsJSONInventory(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "list", "--json", "--project", "mall-api", "--domain", "backend"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"packages"`) || !strings.Contains(stdout.String(), `"rule"`) {
		t.Fatalf("expected inventory JSON: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Full body implementation detail") {
		t.Fatalf("knowledge list should not print full body: %s", stdout.String())
	}
}

func TestRunKnowledgeListAcceptsTypesAndIncludeDeprecated(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "list", "--json", "--project", "mall-api", "--domain", "backend", "--types", "package", "--include-deprecated"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"package": 1`) {
		t.Fatalf("expected knowledge list to honor --types package: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"rule"`) {
		t.Fatalf("expected knowledge list to filter out rules: %s", stdout.String())
	}
}

func TestRunKnowledgeListRejectsMissingProject(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "list", "--json", "--domain", "backend"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "knowledge list: --project is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunKnowledgeReadReturnsFullBody(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "read", "--json", "rule:backend.auth.v1"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		ID   string `json:"id"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.ID != "rule:backend.auth.v1" || !strings.Contains(result.Body, "Full body implementation detail") {
		t.Fatalf("expected full body for auth rule: %s", stdout.String())
	}
}

func TestRunKnowledgeCiteReturnsCitations(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "cite", "--json", "rule:backend.auth.v1"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Citations []struct {
			ID string `json:"id"`
		} `json:"citations"`
		Missing []string `json:"missing"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Citations) != 1 || result.Citations[0].ID != "rule:backend.auth.v1" || len(result.Missing) != 0 {
		t.Fatalf("expected auth rule citation: %s", stdout.String())
	}
}

func TestRunKnowledgeReadAndCiteValidateRequiredFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "read missing json", args: []string{"knowledge", "read", "rule:backend.auth.v1"}, want: "knowledge read: --json is required"},
		{name: "read missing id", args: []string{"knowledge", "read", "--json"}, want: "knowledge read: id is required"},
		{name: "read extra id", args: []string{"knowledge", "read", "--json", "rule:backend.auth.v1", "rule:backend.extra.v1"}, want: "knowledge read: id is required"},
		{name: "cite missing json", args: []string{"knowledge", "cite", "rule:backend.auth.v1"}, want: "knowledge cite: --json is required"},
		{name: "cite missing ids", args: []string{"knowledge", "cite", "--json"}, want: "knowledge cite: at least one id is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("expected exit code 2, got %d", code)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.want, stderr.String())
			}
		})
	}
}

func TestRunKnowledgeRequiresSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "knowledge: subcommand is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunKnowledgeRejectsUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "missing"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), `knowledge: unknown subcommand "missing"`) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunKnowledgeFindRequiresIndex(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "auth"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "index not available: run argos index first") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
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

func TestRunPromoteMovesInboxPackageToOfficialPackages(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected promoted package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/.inbox/packages/backend/redis/best-practices")); !os.IsNotExist(err) {
		t.Fatalf("expected inbox package removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "promoted knowledge/packages/backend/redis/best-practices") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "run argos index") {
		t.Fatalf("expected index recommendation, got %q", stdout.String())
	}
}

func TestRunPromoteRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.best-practices.v1"))
	writeCLIFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIPackage("package:backend.redis.existing.v1"))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/best-practices"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "target already exists") {
		t.Fatalf("expected overwrite error, got %q", stderr.String())
	}
}

func TestRunPromoteRejectsInvalidCandidate(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/broken/KNOWLEDGE.md", `---
id: package:backend.redis.broken.v1
title: Broken
type: package
status: draft
priority: should
updated_at: 2026-04-29
---
broken
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"promote", "--path", "knowledge/.inbox/packages/backend/redis/broken"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/.inbox/packages/backend/redis/broken/KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected invalid candidate to remain in inbox: %v", err)
	}
	if !strings.Contains(stderr.String(), "validation failed") {
		t.Fatalf("expected validation error, got %q", stderr.String())
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
		RecommendedNextSteps []struct {
			Tool   string `json:"tool"`
			Reason string `json:"reason"`
		} `json:"recommended_next_steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON output, got error %v and output %q", err, stdout.String())
	}
	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", result.Project)
	}
	if len(result.RecommendedNextSteps) == 0 {
		t.Fatal("expected recommended next steps")
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

func TestRunMCPStartsWithoutIndexAndDiscoversTools(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n")

	code := runWithIO([]string{"mcp"}, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "argos_context") {
		t.Fatalf("expected tools discovery, got %s", stdout.String())
	}
}

func TestOpenMCPServerClassifiesMissingIndexAsUnavailable(t *testing.T) {
	root := t.TempDir()

	_, closeServer, available := openMCPServer(root)
	defer closeServer()

	if available {
		t.Fatal("expected missing index to be unavailable")
	}
}

func TestRunMCPWithInvalidIndexFallsBackAndDiscoversTools(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "argos/index.db", "not a sqlite database")
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n")

	code := runWithIO([]string{"mcp"}, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "argos_context") {
		t.Fatalf("expected tools discovery, got %s", stdout.String())
	}

	_, closeServer, available := openMCPServer(root)
	defer closeServer()
	if available {
		t.Fatal("expected invalid index to be unavailable")
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

func writeCLIRegistry(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, database]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule, package]\n")
}

func writeCLIDiscoveryWorkspace(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, security]
business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend, security]
    business_domains: [account]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule, package]\n")
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: rule:backend.auth.v1
title: Refresh token auth rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  files: ["internal/auth/**"]
updated_at: 2026-04-29
tags: [auth, refresh-token]
---
Refresh token auth guidance.

Full body implementation detail: refresh token endpoints must rotate tokens and require auth middleware.
`)
	writeCLIFile(t, root, "knowledge/packages/backend/auth-refresh/KNOWLEDGE.md", validCLIPackage("package:backend.auth-refresh.v1"))
}

func validCLIPackage(id string) string {
	return `---
id: ` + id + `
title: Redis Best Practices
type: package
tech_domains: [backend]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis]
updated_at: 2026-04-29
---
## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read the short rules first.

## Load On Demand

- references/key-design.md when designing keys.
`
}

func validCLIKnowledgeItem(id string, itemType string) string {
	return `---
id: ` + id + `
title: Auth rule
type: ` + itemType + `
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: must
updated_at: 2026-04-29
---
Use explicit auth middleware.
`
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

func runWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return run(args, stdin, stdout, stderr)
}
