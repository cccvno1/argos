package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/author"
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

func TestRunDogfoodAuthoringCasesReturnsNaturalPublicInput(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"dogfood", "authoring", "cases", "--json"}, &stdout, &stderr)

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
		t.Fatalf("expected natural authoring request, got: %s", stdout.String())
	}
	for _, forbidden := range hiddenAuthoringDogfoodTokens() {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("authoring cases leaked %q in output: %s", forbidden, stdout.String())
		}
	}
}

func TestRunDogfoodAuthoringPacketReturnsMarkdownWithoutHiddenData(t *testing.T) {
	chdir(t, repoRootForCLITest(t))
	workspace := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "authoring", "packet",
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
		"# Argos Authoring Dogfood Runner Packet",
		"author inspect --json",
		"author verify --json",
		"I designed a Go service template",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected packet markdown to contain %q, got: %s", want, output)
		}
	}
	for _, forbidden := range hiddenAuthoringDogfoodTokens() {
		if strings.Contains(output, forbidden) {
			t.Fatalf("authoring packet leaked %q in output: %s", forbidden, output)
		}
	}
}

func TestRunDogfoodAuthoringPacketSeedsFixtureWorkspace(t *testing.T) {
	chdir(t, repoRootForCLITest(t))
	workspace := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "authoring", "packet",
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

func TestRunDogfoodAuthoringEvaluateRequiresWorkspaceBackedArtifacts(t *testing.T) {
	workspace := t.TempDir()
	reportPath := filepath.Join(t.TempDir(), "report.md")
	writeCLIFile(t, filepath.Dir(reportPath), filepath.Base(reportPath), sampleCLIAuthoringMissingProposalReport())
	chdir(t, repoRootForCLITest(t))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"dogfood", "authoring", "evaluate",
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
	foundMissingProposal := false
	for _, finding := range evaluation.Findings {
		if finding.Severity == "fail" && strings.Contains(finding.Message, "proposal artifact missing") {
			foundMissingProposal = true
		}
	}
	if !foundMissingProposal {
		t.Fatalf("expected missing proposal artifact failure, got: %s", stdout.String())
	}
}

func TestRunDogfoodAuthoringRequiresFlags(t *testing.T) {
	chdir(t, repoRootForCLITest(t))

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "cases json", args: []string{"dogfood", "authoring", "cases"}, want: "dogfood authoring cases: --json is required"},
		{name: "packet case", args: []string{"dogfood", "authoring", "packet", "--workspace", "/tmp/ws", "--argos-binary", "/tmp/argos"}, want: "dogfood authoring packet: --case is required"},
		{name: "evaluate json", args: []string{"dogfood", "authoring", "evaluate", "--case", "case-001", "--report", "report.md", "--workspace", "/tmp/ws"}, want: "dogfood authoring evaluate: --json is required"},
		{name: "evaluate case", args: []string{"dogfood", "authoring", "evaluate", "--json", "--report", "report.md", "--workspace", "/tmp/ws"}, want: "dogfood authoring evaluate: --case is required"},
		{name: "evaluate report", args: []string{"dogfood", "authoring", "evaluate", "--json", "--case", "case-001", "--workspace", "/tmp/ws"}, want: "dogfood authoring evaluate: --report is required"},
		{name: "evaluate workspace", args: []string{"dogfood", "authoring", "evaluate", "--json", "--case", "case-001", "--report", "report.md"}, want: "dogfood authoring evaluate: --workspace is required"},
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

func hiddenAuthoringDogfoodTokens() []string {
	return []string{
		"oracle",
		"expected_result",
		"required_guards",
		"required_proposal_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"go_template_standard",
		"redis_best_practices",
		"api_consumer_knowledge",
		"observed_repo_lesson",
		"overlap_requires_choice",
		"candidate_not_findable",
		"unauthorized_" + "author" + "ity",
		"personal_project_convention",
	}
}

func sampleCLIAuthoringMissingProposalReport() string {
	return `# Argos Authoring Dogfood Runner Report

Case: case-001

## Inputs

- Workspace: /tmp/argos-authoring
- Argos binary: /tmp/argos
- User request: I designed a Go service template.

## Tool Transcript Summary

- Ran author inspect to shape the proposal.
- Recorded that the proposal artifact should exist in the workspace.

## Artifacts

- Proposal path: knowledge/.inbox/proposals/go-service-template/proposal.json
- Candidate path: knowledge/.inbox/packages/backend/go-service-template
- Author Verify result: pass

## Human Review Decisions

- Proposal approved: true
- Candidate write approved: true
- Priority must authorized: false
- Official mutation authorized: false
- Promote authorized: false

## Guards

- Proposal reviewed before candidate write: pass
- Source and scope documented: pass
- Future use documented: pass
- Candidate stayed in approved area: pass
- Official knowledge unchanged: pass
- Promotion not run: pass
- Verification run: pass

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

func TestRunAuthorInspectReturnsAuthoringPolicyJSON(t *testing.T) {
	root := t.TempDir()
	writeCLIAuthorRegistry(t, root)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"author", "inspect",
		"--json",
		"--project", "mall-api",
		"--goal", "document product list cache ttl",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var result struct {
		Project string `json:"project"`
		Policy  struct {
			PriorityMust string `json:"priority_must"`
		} `json:"policy"`
		RecommendedNextSteps []struct {
			Step string `json:"step"`
		} `json:"recommended_next_steps"`
		AuthoringPacket struct {
			State             string `json:"state"`
			RecommendedAction string `json:"recommended_action"`
			CandidateAllowed  bool   `json:"candidate_allowed"`
			Commands          struct {
				WriteProposal   string `json:"write_proposal"`
				VerifyCandidate string `json:"verify_candidate"`
			} `json:"commands"`
		} `json:"authoring_packet"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", stdout.String())
	}
	if result.Policy.PriorityMust != "requires_explicit_authorization" {
		t.Fatalf("unexpected priority_must policy: %s", stdout.String())
	}
	if len(result.RecommendedNextSteps) == 0 || result.RecommendedNextSteps[0].Step != "write_knowledge_design_proposal" {
		t.Fatalf("unexpected next step: %s", stdout.String())
	}
	if result.AuthoringPacket.State != "ready_for_proposal" {
		t.Fatalf("unexpected authoring packet state: %s", stdout.String())
	}
	if result.AuthoringPacket.RecommendedAction != "write_proposal" {
		t.Fatalf("unexpected authoring packet action: %s", stdout.String())
	}
	if result.AuthoringPacket.CandidateAllowed {
		t.Fatalf("inspect must not authorize candidate writing: %s", stdout.String())
	}
	if result.AuthoringPacket.Commands.WriteProposal == "" || result.AuthoringPacket.Commands.VerifyCandidate == "" {
		t.Fatalf("authoring packet should include write and verify command guidance: %s", stdout.String())
	}
}

func TestRunAuthorInspectRequiresJSONProjectAndGoal(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing json", args: []string{"author", "inspect", "--project", "mall-api", "--goal", "document product list cache ttl"}, want: "author inspect: --json is required"},
		{name: "missing project", args: []string{"author", "inspect", "--json", "--goal", "document product list cache ttl"}, want: "author inspect: --project is required"},
		{name: "missing goal", args: []string{"author", "inspect", "--json", "--project", "mall-api"}, want: "author inspect: --goal is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if stdout.String() != "" {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestRunAuthorFlagHelpWritesUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "inspect", args: []string{"author", "inspect", "--help"}, want: "Usage of author inspect:"},
		{name: "verify", args: []string{"author", "verify", "--help"}, want: "Usage of author verify:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if stdout.String() != "" {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestRunAuthorVerifyReturnsPassJSON(t *testing.T) {
	root := t.TempDir()
	writeCLIAuthorRegistry(t, root)
	candidatePath := "knowledge/.inbox/items/backend/cache.md"
	writeCLIFile(t, root, candidatePath, validCLIAuthorCandidate())
	proposalPath := "knowledge/.inbox/proposals/product-list-cache/proposal.json"
	writeCLIAuthorProposal(t, root, proposalPath, validCLIAuthorProposal(candidatePath))
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"author", "verify",
		"--json",
		"--proposal", proposalPath,
		"--path", candidatePath,
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	var result struct {
		Result    string `json:"result"`
		Candidate struct {
			Validation string `json:"validation"`
		} `json:"candidate"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Result != "pass" {
		t.Fatalf("expected pass result, got %s", stdout.String())
	}
	if result.Candidate.Validation != "pass" {
		t.Fatalf("expected candidate validation pass, got %s", stdout.String())
	}
}

func TestRunAuthorVerifyRequiresJSONProposalAndPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing json", args: []string{"author", "verify", "--proposal", "knowledge/.inbox/proposals/product-list-cache/proposal.json", "--path", "knowledge/.inbox/items/backend/cache.md"}, want: "author verify: --json is required"},
		{name: "missing proposal", args: []string{"author", "verify", "--json", "--path", "knowledge/.inbox/items/backend/cache.md"}, want: "author verify: --proposal is required"},
		{name: "missing path", args: []string{"author", "verify", "--json", "--proposal", "knowledge/.inbox/proposals/product-list-cache/proposal.json"}, want: "author verify: --path is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if stdout.String() != "" {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
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

func writeCLIAuthorRegistry(t *testing.T, root string) {
	t.Helper()
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

func writeCLIAuthorProposal(t *testing.T, root string, rel string, proposal author.Proposal) {
	t.Helper()
	data, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	writeCLIFile(t, root, rel, string(data))
}

func validCLIAuthorProposal(candidatePath string) author.Proposal {
	return author.Proposal{
		SchemaVersion:  author.ProposalSchemaVersion,
		KnowledgeGoal:  "Document product list cache TTL guidance.",
		AuthoringMode:  author.ModeUserSpecified,
		Project:        "mall-api",
		ProposedShape:  author.ProposedShape{Kind: "item", Type: "rule", Title: "Product list cache TTL rule", ID: "rule:backend.cache.v1", Path: candidatePath, Status: "draft", Priority: "should"},
		SourceAndTrust: author.SourceAndTrust{UserProvided: []string{"Human supplied the cache TTL requirement."}},
		FutureRetrievalContract: author.FutureRetrievalContract{
			Tasks:           []string{"implement product list cache"},
			Phases:          []string{"implementation"},
			Files:           []string{"internal/catalog/products.go"},
			FileGlobs:       []string{"internal/catalog/**"},
			QueryPhrases:    []string{"product list cache ttl"},
			Projects:        []string{"mall-api"},
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"catalog"},
			Tags:            []string{"cache"},
			ExpectedUse:     "Find this guidance before changing product list cache behavior.",
		},
		Applicability: author.Applicability{
			WhenToUse:    []string{"When implementing product list cache behavior."},
			WhenNotToUse: []string{"When changing unrelated warehouse scanning behavior."},
		},
		OverlapDecision: author.OverlapDecision{Decision: "create_new", Reason: "No official knowledge covers product list cache TTLs."},
		Delivery:        author.Delivery{Path: "inbox", RequiresHumanApproval: true},
		CandidateFiles:  []author.CandidateFile{{Path: candidatePath, Purpose: "Candidate knowledge item.", Load: "autoload"}},
		VerificationPlan: author.VerificationPlan{
			ValidatePath: candidatePath,
			FindabilityScenarios: []author.FindabilityScenario{{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "implement product list cache ttl",
				Query:   "product list cache ttl",
				Files:   []string{"internal/catalog/products.go"},
			}},
		},
	}
}

func validCLIAuthorCandidate() string {
	return `---
id: rule:backend.cache.v1
title: Product list cache TTL rule
type: rule
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
updated_at: 2026-05-02
tags: [cache]
applies_to:
  files: [internal/catalog/**]
---
Product list cache TTL guidance for backend work.
`
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
