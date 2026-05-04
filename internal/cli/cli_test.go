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
	"argos/internal/knowledgewrite"
	"argos/internal/registry"
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
	if strings.Contains(out.String(), "\n  new\n") {
		t.Fatalf("usage should not list unimplemented new command, got %q", out.String())
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

func TestRunProjectAddCreatesProject(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--tech-domain", "database",
		"--business-domain", "account",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if strings.TrimSpace(stdout.String()) != "added project mall-api" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	reg, err := registry.Load(root)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if len(reg.Projects) != 1 || reg.Projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects: %#v", reg.Projects)
	}
	project := reg.Projects[0]
	if project.Name != "Mall API" {
		t.Fatalf("unexpected project name: %#v", project)
	}
	if project.Path != "services/mall-api" {
		t.Fatalf("unexpected project path: %#v", project)
	}
	if strings.Join(project.TechDomains, ",") != "backend,database" {
		t.Fatalf("unexpected tech domains: %#v", project.TechDomains)
	}
	if strings.Join(project.BusinessDomains, ",") != "account" {
		t.Fatalf("unexpected business domains: %#v", project.BusinessDomains)
	}
}

func TestRunProjectListReturnsJSON(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})
	runOK(t, root, []string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--business-domain", "account",
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "list", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var result struct {
		Projects []registry.Project `json:"projects"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(result.Projects) != 1 || result.Projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects JSON: %s", stdout.String())
	}
	project := result.Projects[0]
	if project.Name != "Mall API" {
		t.Fatalf("unexpected decoded project name: %#v", project)
	}
	if project.Path != "services/mall-api" {
		t.Fatalf("unexpected decoded project path: %#v", project)
	}
	if strings.Join(project.TechDomains, ",") != "backend" {
		t.Fatalf("unexpected decoded tech domains: %#v", project.TechDomains)
	}
	if strings.Join(project.BusinessDomains, ",") != "account" {
		t.Fatalf("unexpected decoded business domains: %#v", project.BusinessDomains)
	}
	if !strings.Contains(stdout.String(), `"tech_domains"`) || !strings.Contains(stdout.String(), `"business_domains"`) {
		t.Fatalf("project list should use snake_case JSON fields, got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), `"ID"`) || strings.Contains(stdout.String(), `"TechDomains"`) {
		t.Fatalf("project list should use snake_case JSON fields, got: %s", stdout.String())
	}
}

func TestRunProjectAddRequiresFields(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "add", "--id", "mall-api"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "project add: --name is required") {
		t.Fatalf("expected missing name error, got %q", stderr.String())
	}
}

func TestRunProjectListRequiresJSON(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"project", "list"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "project list: --json is required") {
		t.Fatalf("expected missing json error, got %q", stderr.String())
	}
}

func TestRunProjectAddRejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})
	runOK(t, root, []string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API 2",
		"--path", "services/mall-api-2",
	}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "project already exists: mall-api") {
		t.Fatalf("expected duplicate project error, got %q", stderr.String())
	}
}

func TestRunProjectAddRejectsUnknownDomain(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "mobile",
	}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown tech domain: mobile") {
		t.Fatalf("expected unknown domain error, got %q", stderr.String())
	}
}

func TestRunProjectAddRejectsUnsafePath(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "../mall-api",
	}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "project path must stay inside workspace") {
		t.Fatalf("expected unsafe path error, got %q", stderr.String())
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
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIDraftPackage("package:backend.redis.best-practices.v1"))
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
	writeCLIFile(t, root, "knowledge/.inbox/packages/backend/redis/best-practices/KNOWLEDGE.md", validCLIDraftPackage("package:backend.redis.best-practices.v1"))
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

func TestRunDogfoodCasesReturnsInputsWithoutExpectedValues(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "cases", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var cases []struct {
		ID    string `json:"id"`
		Input struct {
			Task string `json:"task"`
		} `json:"input"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &cases); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(cases) == 0 || cases[0].ID != "case-001" {
		t.Fatalf("expected public case handles, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "add refresh token endpoint") {
		t.Fatalf("expected case input task in JSON: %s", stdout.String())
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
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("dogfood cases leaked %q in output: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDogfoodPacketReturnsMarkdownWithoutExpectedValues(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "packet",
		"--case", "case-009",
		"--workspace", "/tmp/argos-fixture",
		"--argos-binary", "/usr/local/bin/argos",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"# Argos Dogfood Runner Packet",
		"Case: `case-009`",
		"Workspace: `/tmp/argos-fixture`",
		"Argos binary: `/usr/local/bin/argos`",
		"/usr/local/bin/argos knowledge find --json",
		"tune cache ttl for product list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected packet markdown to contain %q, got: %s", want, output)
		}
	}
	for _, forbidden := range []string{
		"expected",
		"support_level",
		"include_ids",
		"reference:backend.cache-policy.v1",
		"partial_domain_without_task_detail",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("dogfood packet leaked %q in output: %s", forbidden, output)
		}
	}
}

func TestRunDogfoodPacketRejectsMissingCase(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "packet", "--workspace", "/tmp/ws", "--argos-binary", "/tmp/argos"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "dogfood packet: --case is required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunDogfoodEvaluateReturnsJSONEvaluation(t *testing.T) {
	root := repoRootForCLITest(t)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	writeCLIFile(t, filepath.Dir(reportPath), filepath.Base(reportPath), sampleCLIDogfoodReport())
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "evaluate", "--case", "case-009", "--report", reportPath, "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var evaluation struct {
		CaseID   string `json:"case_id"`
		Result   string `json:"result"`
		Findings []struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &evaluation); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if evaluation.CaseID != "case-009" || evaluation.Result != "pass" || len(evaluation.Findings) != 0 {
		t.Fatalf("unexpected evaluation: %s", stdout.String())
	}
}

func sampleCLIDogfoodReport() string {
	return `Case: case-009

## Inputs

Runner used the packet input.

## Tool Transcript Summary

Ran find, read, and cite commands.

## Observed Results

- Actual support: partial
- Usage guidance: must separate Argos-backed and general reasoning
- Search status: disabled
- Discovered IDs: reference:backend.cache-policy.v1
- Read IDs: reference:backend.cache-policy.v1
- Cited IDs: reference:backend.cache-policy.v1
- Missing needs: product-list TTL implementation detail
- Argos-backed vs general reasoning: separated
- Next steps: none

## Guards

- Progressive reading: pass
- Weak/none no-overclaim: pass
- Citation accountability: pass
- Cited IDs subset of read-and-used IDs: pass
- Missing needs not cited: pass
- Attribution boundary: pass
- No discovery-triggered upload/capture: pass
- Usage guidance followed: pass
- Context contamination: pass

## Result

Result: pass
`
}

func TestRunDogfoodEvaluateMismatchDoesNotLeakGoldenCaseID(t *testing.T) {
	root := repoRootForCLITest(t)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	writeCLIFile(t, filepath.Dir(reportPath), filepath.Base(reportPath), `Case: case-008

## Inputs

Runner used the packet input.

## Tool Transcript Summary

Ran find, read, and cite commands.

## Observed Results

- Actual support: partial
- Usage guidance: must separate Argos-backed and general reasoning
- Search status: disabled
- Discovered IDs: reference:backend.cache-policy.v1
- Read IDs: reference:backend.cache-policy.v1
- Cited IDs: reference:backend.cache-policy.v1
- Missing needs: product-list TTL implementation detail
- Argos-backed vs general reasoning: separated
- Next steps: none

## Guards

- Progressive reading: pass
- Weak/none no-overclaim: pass
- Citation accountability: pass
- Cited IDs subset of read-and-used IDs: pass
- Missing needs not cited: pass
- Attribution boundary: pass
- No discovery-triggered upload/capture: pass
- Usage guidance followed: pass
- Context contamination: pass

## Result

Result: pass
`)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "evaluate", "--case", "case-009", "--report", reportPath, "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	for _, forbidden := range []string{"partial_domain_without_task_detail", "strong_auth_refresh_full_signal", "weak_single_generic_term", "none_payment_webhook"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("dogfood evaluate leaked %q in output: %s", forbidden, stdout.String())
		}
	}
	if !strings.Contains(stdout.String(), `"result": "fail"`) {
		t.Fatalf("expected mismatch to fail without leaking golden ID: %s", stdout.String())
	}
}

func TestRunDogfoodEvaluateMismatchDoesNotEchoHiddenReportCaseID(t *testing.T) {
	root := repoRootForCLITest(t)
	reportPath := filepath.Join(t.TempDir(), "report.md")
	writeCLIFile(t, filepath.Dir(reportPath), filepath.Base(reportPath), strings.Replace(sampleCLIDogfoodReport(), "Case: case-009", "Case: strong_auth_refresh_full_signal", 1))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "evaluate", "--case", "case-009", "--report", reportPath, "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if strings.Contains(stdout.String(), "strong_auth_refresh_full_signal") || strings.Contains(stdout.String(), "partial_domain_without_task_detail") {
		t.Fatalf("dogfood evaluate echoed hidden case ID in output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "report case id does not match requested case") {
		t.Fatalf("expected case mismatch finding: %s", stdout.String())
	}
}

func TestRunDogfoodWriteCasesReturnsNaturalPublicInput(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "write", "cases", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var cases []struct {
		ID    string `json:"id"`
		Input struct {
			UserRequest string `json:"user_request"`
		} `json:"input"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &cases); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(cases) == 0 || cases[0].ID != "case-001" {
		t.Fatalf("expected public case handles, got: %s", stdout.String())
	}
	if !strings.Contains(cases[0].Input.UserRequest, "I designed a Go service template") {
		t.Fatalf("expected natural write request, got: %s", stdout.String())
	}
	for _, forbidden := range hiddenWriteDogfoodTokens() {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("write cases leaked %q in output: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDogfoodWritePacketReturnsMarkdownWithoutHiddenData(t *testing.T) {
	chdir(t, repoRootForCLITest(t))
	workspace := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "write", "packet",
		"--case", "case-001",
		"--workspace", workspace,
		"--argos-binary", "/tmp/argos",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"# Argos Write Dogfood Runner Packet",
		"## Write Guidance",
		"## Knowledge Design Template",
		"argos knowledge design --json --project <project> --intent <intent>",
		"argos knowledge check --json --design <design-path> --draft <draft-path>",
		"I designed a Go service template",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected packet markdown to contain %q, got: %s", want, output)
		}
	}
	assertNoRemovedWriteTerms(t, output)
	for _, forbidden := range hiddenWriteDogfoodTokens() {
		if strings.Contains(output, forbidden) {
			t.Fatalf("write packet leaked %q in output: %s", forbidden, output)
		}
	}
}

func TestRunDogfoodWritePacketSeedsFixtureWorkspace(t *testing.T) {
	chdir(t, repoRootForCLITest(t))
	workspace := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "write", "packet",
		"--case", "case-003",
		"--workspace", workspace,
		"--argos-binary", "/tmp/argos",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	for _, rel := range []string{
		"knowledge/domains.yaml",
		"knowledge/projects.yaml",
		"internal/api/README.md",
	} {
		if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("packet command did not seed fixture file %s: %v", rel, err)
		}
	}
}

func TestRunDogfoodWriteEvaluateRequiresWorkspaceBackedArtifacts(t *testing.T) {
	workspace := t.TempDir()
	reportPath := filepath.Join(t.TempDir(), "report.md")
	writeCLIFile(t, filepath.Dir(reportPath), filepath.Base(reportPath), sampleCLIWriteMissingDesignReport())
	chdir(t, repoRootForCLITest(t))

	var packetStdout bytes.Buffer
	var packetStderr bytes.Buffer
	packetCode := Run([]string{
		"dogfood", "write", "packet",
		"--case", "case-001",
		"--workspace", workspace,
		"--argos-binary", "/tmp/argos",
	}, &packetStdout, &packetStderr)
	if packetCode != 0 {
		t.Fatalf("seed packet failed with code %d stderr=%q stdout=%q", packetCode, packetStderr.String(), packetStdout.String())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "write", "evaluate",
		"--case", "case-001",
		"--report", reportPath,
		"--workspace", workspace,
		"--json",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var evaluation struct {
		CaseID   string `json:"case_id"`
		Result   string `json:"result"`
		Findings []struct {
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &evaluation); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if evaluation.CaseID != "case-001" || evaluation.Result != "fail" {
		t.Fatalf("expected case-001 failure, got: %s", stdout.String())
	}
	foundMissingDesign := false
	for _, finding := range evaluation.Findings {
		if finding.Severity == "fail" && strings.Contains(finding.Message, "design artifact missing") {
			foundMissingDesign = true
		}
	}
	if !foundMissingDesign {
		t.Fatalf("expected missing design artifact failure, got: %s", stdout.String())
	}
}

func TestRunDogfoodWriteRequiresFlags(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "cases json", args: []string{"dogfood", "write", "cases"}, want: "dogfood write cases: --json is required"},
		{name: "packet case", args: []string{"dogfood", "write", "packet", "--workspace", "/tmp/ws", "--argos-binary", "/tmp/argos"}, want: "dogfood write packet: --case is required"},
		{name: "evaluate json", args: []string{"dogfood", "write", "evaluate", "--case", "case-001", "--report", "report.md", "--workspace", "/tmp/ws"}, want: "dogfood write evaluate: --json is required"},
		{name: "evaluate case", args: []string{"dogfood", "write", "evaluate", "--json", "--report", "report.md", "--workspace", "/tmp/ws"}, want: "dogfood write evaluate: --case is required"},
		{name: "evaluate report", args: []string{"dogfood", "write", "evaluate", "--json", "--case", "case-001", "--workspace", "/tmp/ws"}, want: "dogfood write evaluate: --report is required"},
		{name: "evaluate workspace", args: []string{"dogfood", "write", "evaluate", "--json", "--case", "case-001", "--report", "report.md"}, want: "dogfood write evaluate: --workspace is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tt.want, stderr.String())
			}
		})
	}
}

func hiddenWriteDogfoodTokens() []string {
	return []string{
		"oracle",
		"expected_result",
		"required_guards",
		"required_design_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"go_template_standard",
		"redis_best_practices",
		"api_consumer_knowledge",
		"observed_repo_lesson",
		"overlap_requires_choice",
		"draft_not_findable",
		"unauthorized_scope",
		"personal_project_convention",
	}
}

func sampleCLIWriteMissingDesignReport() string {
	return `# Argos Write Dogfood Runner Report

Case: case-001

## Inputs

- Workspace: /tmp/argos-write
- Argos binary: /tmp/argos
- User request: I designed a Go service template.

## Write Guidance

- State: design_only
- Next action: write_knowledge_design
- Design path: knowledge/.inbox/designs/mall-api/i-designed-a-go-service-template-turn-it-into-reusable-knowledge/design.json
- Draft path: none
- Draft allowed: false
- Design only: true
- Check result: not-run

## Artifacts

- Design path: knowledge/.inbox/designs/mall-api/i-designed-a-go-service-template-turn-it-into-reusable-knowledge/design.json
- Draft path: none
- Check result: not-run

## Review Decisions

- Design approved: true
- Draft write approved: false
- Priority must approved: false
- Official write approved: false
- Publish approved: false

## Guards

- Design reviewed before draft write: pass
- Sources and scope documented: pass
- Future use documented: pass
- Draft stayed in approved area: not-applicable
- Official knowledge unchanged: pass
- Publish not run: pass
- Check run: not-run

## Result

Result: pass
`
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

func TestRunKnowledgeDesignReturnsWriteGuidance(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	initWorkspace(t, root)
	code := runInDir(t, root, []string{
		"knowledge", "design", "--json",
		"--project", "mall-api",
		"--intent", "Create Redis cache best practices for future backend agents.",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var result struct {
		WriteGuidance struct {
			State        string `json:"state"`
			NextAction   string `json:"next_action"`
			DraftAllowed bool   `json:"draft_allowed"`
		} `json:"write_guidance"`
		KnowledgeDesignTemplate struct {
			SchemaVersion string `json:"schema_version"`
			Sources       any    `json:"sources"`
		} `json:"knowledge_design_template"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, stdout.String())
	}
	if result.WriteGuidance.State != "ready_to_design" {
		t.Fatalf("unexpected write guidance: %s", stdout.String())
	}
	if result.WriteGuidance.DraftAllowed {
		t.Fatalf("design must not approve draft writing: %s", stdout.String())
	}
	if result.KnowledgeDesignTemplate.SchemaVersion != "knowledge.design.v1" {
		t.Fatalf("unexpected schema: %s", stdout.String())
	}
	assertNoRemovedWriteTerms(t, stdout.String())
}

func TestRunKnowledgeCheckRequiresDesignAndDraft(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "check", "--json", "--design", "design.json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "knowledge check: --draft is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunKnowledgeCheckReturnsJSONStatus(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "check", "--json", "--design", designPath, "--draft", draftPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var result struct {
		Result      string `json:"result"`
		Design      any    `json:"design"`
		Draft       any    `json:"draft"`
		Policy      any    `json:"policy"`
		Findability any    `json:"findability"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, stdout.String())
	}
	if result.Result == "" || result.Design == nil || result.Draft == nil || result.Policy == nil || result.Findability == nil {
		t.Fatalf("missing check status fields: %s", stdout.String())
	}
}

func TestRunProvenanceStartRecordCheckAndVerify(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)

	startOutput := runOK(t, root, []string{
		"provenance", "start", "--json",
		"--design", designPath,
		"--draft", draftPath,
		"--created-by", "codex",
	})
	var startResult struct {
		ProvenanceID string `json:"provenance_id"`
	}
	if err := json.Unmarshal([]byte(startOutput), &startResult); err != nil {
		t.Fatalf("parse start JSON: %v\n%s", err, startOutput)
	}
	if startResult.ProvenanceID == "" {
		t.Fatalf("start JSON missing provenance_id: %s", startOutput)
	}

	for _, stage := range []string{"design", "draft_write"} {
		runOK(t, root, []string{
			"provenance", "record-decision", "--json",
			"--provenance", startResult.ProvenanceID,
			"--stage", stage,
			"--decision", "approved",
			"--decided-by", "chenchi",
			"--role", "knowledge_owner",
			"--source", "conversation",
			"--reason", stage + " approved.",
			"--recorded-by", "codex",
		})
	}

	runOK(t, root, []string{
		"provenance", "record-check", "--json",
		"--provenance", startResult.ProvenanceID,
	})

	runOK(t, root, []string{
		"provenance", "record-decision", "--json",
		"--provenance", startResult.ProvenanceID,
		"--stage", "publish",
		"--decision", "approved",
		"--decided-by", "chenchi",
		"--role", "knowledge_owner",
		"--source", "conversation",
		"--reason", "publish approved.",
		"--recorded-by", "codex",
	})

	verifyOutput := runOK(t, root, []string{
		"provenance", "verify", "--json",
		"--provenance", startResult.ProvenanceID,
	})
	var verifyResult struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(verifyOutput), &verifyResult); err != nil {
		t.Fatalf("parse verify JSON: %v\n%s", err, verifyOutput)
	}
	if verifyResult.Result != "pass" {
		t.Fatalf("expected verify pass, got: %s", verifyOutput)
	}
}

func TestRunProvenanceRequiresJSONForMachineReadableCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"provenance", "start", "--design", "design.json", "--draft", "knowledge/.inbox/packages/x"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "provenance start: --json is required") {
		t.Fatalf("expected missing json error, got %q", stderr.String())
	}
}

func TestKnowledgeWritePublishAndFindbackFlow(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	runOK(t, root, []string{"init"})
	runOK(t, root, []string{
		"project", "add",
		"--id", "mall-api",
		"--name", "Mall API",
		"--path", "services/mall-api",
		"--tech-domain", "backend",
		"--business-domain", "account",
	})

	designOutput := runOK(t, root, []string{
		"knowledge", "design", "--json",
		"--project", "mall-api",
		"--intent", "Create Redis cache best practices for future backend agents.",
	})
	var designResult struct {
		WriteGuidance struct {
			DesignPath string `json:"design_path"`
		} `json:"write_guidance"`
		KnowledgeDesignTemplate struct {
			SchemaVersion string `json:"schema_version"`
		} `json:"knowledge_design_template"`
	}
	if err := json.Unmarshal([]byte(designOutput), &designResult); err != nil {
		t.Fatalf("parse design JSON: %v\n%s", err, designOutput)
	}
	if designResult.KnowledgeDesignTemplate.SchemaVersion != knowledgewrite.KnowledgeDesignSchemaVersion {
		t.Fatalf("unexpected design template schema: %s", designOutput)
	}
	if designResult.WriteGuidance.DesignPath == "" {
		t.Fatalf("design output missing write_guidance.design_path: %s", designOutput)
	}
	if !strings.HasPrefix(designResult.WriteGuidance.DesignPath, "knowledge/.inbox/designs/mall-api/") ||
		!strings.HasSuffix(designResult.WriteGuidance.DesignPath, "/design.json") {
		t.Fatalf("unexpected design path: %q", designResult.WriteGuidance.DesignPath)
	}
	designPath := writeJSONFixture(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validKnowledgeDesignForCLI(t))
	draftPath := "knowledge/.inbox/packages/mall-api/redis-cache"
	writeDraftPackageForCLI(t, root, draftPath)
	if _, err := os.Stat(filepath.Join(root, draftPath, "references/redis-cache.md")); err != nil {
		t.Fatalf("expected referenced draft file: %v", err)
	}

	checkOutput := runOK(t, root, []string{"knowledge", "check", "--json", "--design", designPath, "--draft", draftPath})
	var checkResult struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(checkOutput), &checkResult); err != nil {
		t.Fatalf("parse check JSON: %v\n%s", err, checkOutput)
	}
	if checkResult.Result != "pass" && checkResult.Result != "review-needed" {
		t.Fatalf("expected check pass or review-needed, got: %s", checkOutput)
	}
	runOK(t, root, []string{"knowledge", "publish", "--design", designPath, "--path", draftPath})
	runOK(t, root, []string{"index"})

	findOutput := runOK(t, root, []string{
		"knowledge", "find", "--json",
		"--project", "mall-api",
		"--query", "redis cache best practices",
	})
	if !strings.Contains(findOutput, "package:mall-api.redis-cache.v1") {
		t.Fatalf("expected published package findback, got: %s", findOutput)
	}
	var findResult struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(findOutput), &findResult); err != nil {
		t.Fatalf("parse find JSON: %v\n%s", err, findOutput)
	}
	if len(findResult.Items) == 0 || findResult.Items[0].Status != "active" {
		t.Fatalf("expected published package findback to be active, got: %s", findOutput)
	}
}

func TestRunAuthorCommandIsRemovedFromPublicCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"author", "inspect", "--json", "--project", "mall-api", "--goal", "x"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command: author") {
		t.Fatalf("expected unknown command, got stderr = %s", stderr.String())
	}
}

func TestUsageUsesWriteVocabulary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d", code)
	}
	body := stderr.String()
	for _, want := range []string{
		"argos project add --id <project> --name <name> --path <path>",
		"argos project list --json",
		"argos knowledge design --json --project <project> --intent <intent>",
		"argos knowledge check --json --design <design.json> --draft <draft>",
		"argos knowledge publish --design <design.json> --path <draft>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("usage missing %q:\n%s", want, body)
		}
	}
	assertNoRemovedWriteTerms(t, body)
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

func TestRunKnowledgePublishMovesInboxPackageToOfficialPackages(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	targetPath := "knowledge/packages/backend/redis/best-practices"
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, targetPath, "KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected published package: %v", err)
	}
	published, err := os.ReadFile(filepath.Join(root, targetPath, "KNOWLEDGE.md"))
	if err != nil {
		t.Fatalf("read published package: %v", err)
	}
	if !strings.Contains(string(published), "status: active") {
		t.Fatalf("expected published package to be active, got:\n%s", string(published))
	}
	if strings.Contains(string(published), "status: draft") {
		t.Fatalf("expected published package to not remain draft, got:\n%s", string(published))
	}
	if _, err := os.Stat(filepath.Join(root, draftPath)); !os.IsNotExist(err) {
		t.Fatalf("expected inbox package removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "published "+targetPath) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunKnowledgePublishRejectsPackageEntrypointPath(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftFile := draftPath + "/KNOWLEDGE.md"
	draftID := "package:backend.redis.best-practices.v1"
	design := validCLIKnowledgeDesign(draftFile, draftID)
	design.DraftFiles = []knowledgewrite.DraftFile{{
		Path:    draftFile,
		Purpose: "Package entrypoint.",
		Load:    "read_before_implementation",
	}}
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", design)
	targetPath := "knowledge/packages/backend/redis/best-practices"
	writeCLIFile(t, root, draftFile, validCLICheckDraftPackage(draftID))
	writeCLIFile(t, root, draftPath+"/references/redis-cache.md", "Redis cache reference must move with the package.\n")
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftFile}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "package publish path must be the package directory, not KNOWLEDGE.md") {
		t.Fatalf("expected package entrypoint path error, got %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, draftFile)); err != nil {
		t.Fatalf("expected draft entrypoint to remain in inbox: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, draftPath, "references/redis-cache.md")); err != nil {
		t.Fatalf("expected draft reference to remain in inbox: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, targetPath)); !os.IsNotExist(err) {
		t.Fatalf("expected official package not to exist, stat err=%v", err)
	}
}

func TestRunKnowledgePublishRejectsNonInboxPath(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	design := validCLIKnowledgeDesign(draftPath, draftID)
	design.WriteBoundary.Path = "official_review"
	design.Review.OfficialWriteApproved = true
	design.DraftOutput.Status = "active"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", design)
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", strings.Replace(validCLICheckDraftPackage(draftID), "status: draft", "status: active", 1))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "draft must be under knowledge/.inbox/items or knowledge/.inbox/packages") {
		t.Fatalf("expected non-inbox publish error, got %q", stderr.String())
	}
}

func TestRunKnowledgePublishRequiresDesign(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--path", draftPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "knowledge publish: --design is required") {
		t.Fatalf("expected missing design error, got %q", stderr.String())
	}
}

func TestRunKnowledgePublishRequiresPublishApproval(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	design := validCLIKnowledgeDesign(draftPath, draftID)
	design.Review.PublishApproved = false
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", design)
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, draftPath, "KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected draft to remain in inbox: %v", err)
	}
	if !strings.Contains(stderr.String(), "review.publish_approved") {
		t.Fatalf("expected publish approval error, got %q", stderr.String())
	}
}

func TestRunKnowledgePublishLeavesDraftInInboxWhenOfficialValidationFails(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	duplicateID := "package:backend.redis.best-practices.v1"
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, duplicateID))
	targetPath := "knowledge/packages/backend/redis/best-practices"
	writeCLIFile(t, root, "knowledge/packages/backend/redis/existing/KNOWLEDGE.md", validCLIPackage(duplicateID))
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(duplicateID))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, draftPath, "KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected draft to remain in inbox: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, targetPath)); !os.IsNotExist(err) {
		t.Fatalf("expected official target not to remain, stat err=%v", err)
	}
	if !strings.Contains(stderr.String(), "duplicate id "+duplicateID) {
		t.Fatalf("expected duplicate id validation error, got %q", stderr.String())
	}
}

func TestRunKnowledgePublishRejectsExistingOfficialDraftStatus(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	writeCLIFile(t, root, "knowledge/items/backend/stale-draft.md", `---
id: rule:backend.stale-draft.v1
title: Stale Official Draft
type: rule
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
updated_at: 2026-05-04
---
Official roots must not carry draft status.
`)
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--design", designPath, "--path", draftPath}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, draftPath, "KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected draft to remain in inbox: %v", err)
	}
	if !strings.Contains(stderr.String(), "official knowledge must not use status: draft") {
		t.Fatalf("expected official draft status error, got %q", stderr.String())
	}
}

func TestRunContextPrintsWorkflowContractJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"context",
		"--json",
		"--project", "mall-api",
		"--phase", "planning",
		"--task", "understand auth refresh token flow",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		Project              string `json:"project"`
		Phase                string `json:"phase"`
		Task                 string `json:"task"`
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
	if result.Phase != "planning" {
		t.Fatalf("unexpected phase: %s", result.Phase)
	}
	if result.Task != "understand auth refresh token flow" {
		t.Fatalf("unexpected task: %s", result.Task)
	}
	if len(result.RecommendedNextSteps) == 0 {
		t.Fatal("expected recommended next steps")
	}
}

func TestRunContextRequiresJSONAndTaskContext(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing json", args: []string{"context", "--project", "mall-api", "--phase", "implementation", "--task", "add refresh token endpoint"}, want: "context: --json is required"},
		{name: "missing project", args: []string{"context", "--json", "--phase", "implementation", "--task", "add refresh token endpoint"}, want: "context: --project is required"},
		{name: "missing phase", args: []string{"context", "--json", "--project", "mall-api", "--task", "add refresh token endpoint"}, want: "context: --phase is required"},
		{name: "missing task", args: []string{"context", "--json", "--project", "mall-api", "--phase", "implementation"}, want: "context: --task is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestRunContextAcceptsRepeatedFilesAndReturnsStepArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"context",
		"--json",
		"--project", "mall-api",
		"--phase", "implementation",
		"--task", "add refresh token endpoint",
		"--files", " internal/auth/session.go ",
		"--files", "",
		"--files", "internal/auth/session_test.go",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		Project string   `json:"project"`
		Phase   string   `json:"phase"`
		Task    string   `json:"task"`
		Files   []string `json:"files"`
		Steps   []struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		} `json:"recommended_next_steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON output, got error %v and output %q", err, stdout.String())
	}
	if result.Project != "mall-api" || result.Phase != "implementation" || result.Task != "add refresh token endpoint" {
		t.Fatalf("context echo mismatch: %#v", result)
	}
	if got, want := result.Files, []string{"internal/auth/session.go", "internal/auth/session_test.go"}; !sameCLIStrings(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
	if len(result.Steps) == 0 || result.Steps[0].Tool != "argos_find_knowledge" {
		t.Fatalf("expected find as first step, got %#v", result.Steps)
	}
	if result.Steps[0].Arguments["project"] != "mall-api" || result.Steps[0].Arguments["phase"] != "implementation" || result.Steps[0].Arguments["task"] != "add refresh token endpoint" {
		t.Fatalf("find arguments did not preserve context: %#v", result.Steps[0].Arguments)
	}
}

func sameCLIStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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

func TestRunValidateRejectsOfficialDraftStatus(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/items/backend/draft.md", `---
id: rule:backend.draft.v1
title: Draft Official Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: draft
priority: should
updated_at: 2026-05-04
---
Official roots must not carry draft status.
`)
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "official knowledge must not use status: draft") {
		t.Fatalf("expected official draft error, got %q", stderr.String())
	}
}

func TestRunValidateInboxRejectsActiveStatus(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/items/backend/active.md", `---
id: rule:backend.active-draft.v1
title: Active Inbox Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: should
updated_at: 2026-05-04
---
Inbox roots must carry draft status.
`)
	chdir(t, root)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", "--inbox"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "inbox knowledge must use status: draft; set the draft back to status: draft before check or publish") {
		t.Fatalf("expected inbox status error, got %q", stderr.String())
	}
}

func TestRunValidatePathDerivesStorageScope(t *testing.T) {
	root := t.TempDir()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/.inbox/items/backend/active.md", `---
id: rule:backend.active-draft.v1
title: Active Inbox Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: active
priority: should
updated_at: 2026-05-04
---
Inbox roots must carry draft status.
`)
	writeCLIFile(t, root, "knowledge/items/backend/draft.md", `---
id: rule:backend.official-draft.v1
title: Draft Official Rule
type: rule
tech_domains: [backend]
business_domains: [account]
projects: []
status: draft
priority: should
updated_at: 2026-05-04
---
Official roots must not carry draft status.
`)
	chdir(t, root)

	var inboxStdout, inboxStderr bytes.Buffer
	inboxCode := Run([]string{"validate", "--path", "knowledge/.inbox/items/backend/active.md"}, &inboxStdout, &inboxStderr)
	if inboxCode != 1 || !strings.Contains(inboxStderr.String(), "inbox knowledge must use status: draft; set the draft back to status: draft before check or publish") {
		t.Fatalf("expected inbox path status error, code=%d stderr=%q", inboxCode, inboxStderr.String())
	}

	var officialStdout, officialStderr bytes.Buffer
	officialCode := Run([]string{"validate", "--path", "knowledge/items/backend/draft.md"}, &officialStdout, &officialStderr)
	if officialCode != 1 || !strings.Contains(officialStderr.String(), "official knowledge must not use status: draft") {
		t.Fatalf("expected official path status error, code=%d stderr=%q", officialCode, officialStderr.String())
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

func initWorkspace(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		"knowledge/.inbox/items",
		"knowledge/.inbox/packages",
		"knowledge/.inbox/designs",
		"knowledge/items",
		"knowledge/packages",
		"argos/generated",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeCLIFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, database]
business_domains: [catalog]
`)
	writeCLIFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [catalog]
`)
	writeCLIFile(t, root, "knowledge/types.yaml", "types: [rule, decision, lesson, runbook, reference, package]\n")
}

func writeCLIKnowledgeDesign(t *testing.T, root string, rel string, design knowledgewrite.KnowledgeDesign) string {
	t.Helper()
	data, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	writeCLIFile(t, root, rel, string(data)+"\n")
	return rel
}

func validKnowledgeDesignForCLI(t *testing.T) knowledgewrite.KnowledgeDesign {
	t.Helper()
	return knowledgewrite.KnowledgeDesign{
		SchemaVersion: knowledgewrite.KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices for future backend agents.",
		KnowledgeGoal: "Create reusable Redis cache best practices for future backend agents.",
		Project:       "mall-api",
		Audience: knowledgewrite.Audience{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"apply Redis cache best practices during backend implementation"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{"mall-api"},
			Stability:    "draft",
			Distribution: "project",
			TechDomains:  []string{"backend"},
			FileGlobs:    []string{"**/*"},
		},
		Sources: knowledgewrite.Sources{
			UserInput:   []string{"User requested Redis cache best practices for future backend agents."},
			AISuggested: []string{"Prefer explicit TTLs, cache keys with stable namespaces, and safe invalidation paths."},
			Assumptions: []string{"The team will review the package before treating it as official guidance."},
			Claims: []knowledgewrite.SourceClaim{
				{
					Claim:  "Redis cache guidance should cover keys, TTLs, invalidation, and fallback behavior.",
					Kind:   "recommendation",
					Source: []string{"user request"},
					Trust:  "user_input",
				},
				{
					Claim:          "Future backend agents should read the package before implementing Redis-backed caching.",
					Kind:           "recommendation",
					Source:         []string{"AI synthesis"},
					Trust:          "ai_suggested",
					RequiresReview: true,
				},
			},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             "package:mall-api.redis-cache.v1",
			Path:           "knowledge/.inbox/packages/mall-api/redis-cache",
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package gives future agents one entrypoint for Redis cache implementation guidance.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: knowledgewrite.FutureUse{
			TriggerRequests:  []string{"implement Redis cache behavior"},
			NegativeTriggers: []string{"unrelated authentication work"},
			QueryPhrases:     []string{"redis cache best practices"},
			ExpectedUse:      "Future agents should read this before implementing Redis cache behavior.",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: knowledgewrite.Applicability{
			WhenToUse:    []string{"When implementing or changing Redis-backed caching in Mall API."},
			WhenNotToUse: []string{"When work does not touch cache behavior or Redis integration."},
		},
		ExistingKnowledge: knowledgewrite.ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing Mall API knowledge package covers Redis cache best practices.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path:                        "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:        true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path:    "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
			Purpose: "Package entrypoint.",
			Load:    "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: "knowledge/.inbox/packages/mall-api/redis-cache",
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{
			Questions:             []string{"Should this Redis cache package be published for future backend agents?"},
			DesignApproved:        true,
			DraftWriteApproved:    true,
			PublishApproved:       true,
			OfficialWriteApproved: true,
		},
	}
}

func writeDraftPackageForCLI(t *testing.T, root string, rel string) {
	t.Helper()
	writeCLIFile(t, root, rel+"/KNOWLEDGE.md", `---
id: package:mall-api.redis-cache.v1
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-05-03
applies_to:
  files: ["**/*"]
---
## Purpose

Document Redis cache best practices for future backend agents.

## When To Use

Use when implementing Redis cache behavior for Mall API services.

## Start Here

Read this package before changing cache keys, TTLs, invalidation, or fallback behavior.

## Load On Demand

- references/redis-cache.md when deeper Redis cache tradeoffs are needed.

Redis cache best practices: choose stable key namespaces, set explicit TTLs, keep invalidation close to writes, and preserve safe fallback behavior when Redis is unavailable.
`)
	writeCLIFile(t, root, rel+"/references/redis-cache.md", `# Redis Cache Reference

Use stable key namespaces, explicit TTLs, bounded payloads, and safe fallback behavior when Redis is unavailable.
`)
}

func runOK(t *testing.T, root string, args []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runInDir(t, root, args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) code = %d, stdout = %q, stderr = %q", args, code, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("Run(%v) stderr = %q", args, stderr.String())
	}
	return stdout.String()
}

func writeJSONFixture(t *testing.T, root string, rel string, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture %s: %v", rel, err)
	}
	writeCLIFile(t, root, rel, string(data)+"\n")
	return rel
}

func validCLIKnowledgeDesign(draftPath string, draftID string) knowledgewrite.KnowledgeDesign {
	return knowledgewrite.KnowledgeDesign{
		SchemaVersion: knowledgewrite.KnowledgeDesignSchemaVersion,
		UserRequest:   "Create Redis cache best practices for future backend agents.",
		KnowledgeGoal: "Document Redis cache best practices.",
		Project:       "mall-api",
		Audience: knowledgewrite.Audience{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"implement Redis cache behavior"},
		},
		Scope: knowledgewrite.Scope{
			Projects:     []string{"mall-api"},
			Stability:    "reviewed",
			Distribution: "project",
			TechDomains:  []string{"backend"},
		},
		Sources: knowledgewrite.Sources{
			UserInput: []string{"User requested Redis cache best practices."},
			Claims: []knowledgewrite.SourceClaim{{
				Claim:  "Redis cache best practices are needed for backend agents.",
				Kind:   "fact",
				Source: []string{"user request"},
				Trust:  "user_input",
			}},
		},
		DraftOutput: knowledgewrite.DraftOutput{
			Kind:           "package",
			Type:           "package",
			Title:          "Redis Cache Best Practices",
			ID:             draftID,
			Path:           draftPath,
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package can hold cache guidance plus future references.",
			EntrypointLoad: "read_before_implementation",
			DraftState:     "draft",
		},
		FutureUse: knowledgewrite.FutureUse{
			TriggerRequests:  []string{"implement Redis cache behavior"},
			NegativeTriggers: []string{"unrelated authentication work"},
			QueryPhrases:     []string{"redis cache best practices"},
			ExpectedUse:      "Future agents should read this before implementing Redis cache behavior.",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: knowledgewrite.Applicability{
			WhenToUse:    []string{"When implementing Redis-backed caching."},
			WhenNotToUse: []string{"When changing unrelated auth behavior."},
		},
		ExistingKnowledge: knowledgewrite.ExistingKnowledgeDecision{
			Decision: "create_new",
			Reason:   "No existing knowledge covers this exact Redis cache guidance.",
		},
		WriteBoundary: knowledgewrite.WriteBoundary{
			Path:                        "inbox",
			WriteRequiresReviewApproval: true,
			ReviewPacketRequired:        true,
		},
		DraftFiles: []knowledgewrite.DraftFile{{
			Path:    draftPath + "/KNOWLEDGE.md",
			Purpose: "Package entrypoint.",
			Load:    "read_before_implementation",
		}},
		CheckPlan: knowledgewrite.CheckPlan{
			ValidatePath: draftPath,
			FindabilityChecks: []knowledgewrite.FindabilityCheckScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement Redis cache behavior",
				Query:   "redis cache best practices",
			}},
		},
		Review: knowledgewrite.Review{
			Questions:          []string{"Is draft writing and publishing approved?"},
			DesignApproved:     true,
			DraftWriteApproved: true,
			PublishApproved:    true,
		},
	}
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
	return validCLIPackageWithStatus(id, "active")
}

func validCLIDraftPackage(id string) string {
	return validCLIPackageWithStatus(id, "draft")
}

func validCLIPackageWithStatus(id string, status string) string {
	return `---
id: ` + id + `
title: Redis Best Practices
type: package
tech_domains: [backend]
business_domains: []
projects: []
status: ` + status + `
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

func validCLICheckDraftPackage(id string) string {
	return `---
id: ` + id + `
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-05-04
---
## Purpose

Document Redis cache best practices for backend agents.

## When To Use

Use when implementing Redis cache behavior for Mall API.

## Start Here

Read this before implementing Redis cache behavior.

## Load On Demand

- references/redis-cache.md when deeper cache tradeoffs are needed.
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

func runInDir(t *testing.T, dir string, args []string, stdout io.Writer, stderr io.Writer) int {
	t.Helper()
	chdir(t, dir)
	return Run(args, stdout, stderr)
}

func assertNoRemovedWriteTerms(t *testing.T, body string) {
	t.Helper()
	for _, term := range []string{
		removedWriteTerm("authoring", "_", "packet"),
		removedWriteTerm("proposal", "_", "scaffold"),
		removedWriteTerm("source", "_", "profile"),
		removedWriteTerm("proposed", "_", "shape"),
		removedWriteTerm("overlap", "_", "decision"),
		removedWriteTerm("verification", "_", "plan"),
		removedWriteTerm("human", "_", "review"),
		removedWriteTerm("artifact", "_", "state"),
		removedWriteTerm("author", " ", "inspect"),
		removedWriteTerm("author", " ", "verify"),
	} {
		if strings.Contains(body, term) {
			t.Fatalf("body contains removed write term %q:\n%s", term, body)
		}
	}
}

func removedWriteTerm(parts ...string) string {
	return strings.Join(parts, "")
}

func legacyDogfoodWriteCommand(subcommand string) string {
	return removedWriteTerm("author", " ", subcommand, " --json")
}

func repoRootForCLITest(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("expected repository root at %s: %v", root, err)
	}
	return root
}

func runWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return run(args, stdin, stdout, stderr)
}
