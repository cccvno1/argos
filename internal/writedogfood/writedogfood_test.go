package writedogfood

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"argos/internal/knowledgewrite"
)

const writeCasesPath = "../../testdata/write-golden/cases.json"

func TestLoadCasesIncludesWriteScenarioMatrix(t *testing.T) {
	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"go_template_standard":        false,
		"redis_best_practices":        false,
		"api_consumer_knowledge":      false,
		"observed_repo_lesson":        false,
		"overlap_requires_choice":     false,
		"draft_not_findable":          false,
		"unauthorized_scope":          false,
		"personal_project_convention": false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing write case kind %s", kind)
		}
	}
}

func TestSummariesExposeNaturalInputAndHideOracle(t *testing.T) {
	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	summaries := Summaries(cases)
	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	text := string(data)
	for _, want := range []string{"case-001", "user_request", "I designed a Go service template"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary missing %q: %s", want, text)
		}
	}
	for _, forbidden := range append(removedAuthoringTermsForTests(),
		"expected_result",
		"required_guards",
		"required_design_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"draft_write_approved",
		"priority_must_approved",
		"official_write_approved",
		"publish_approved",
		"review-needed",
	) {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}
}

func TestBuildPacketIncludesWriteGuidanceAndCommands(t *testing.T) {
	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	if err := SeedFixtureWorkspace("../../testdata/write-golden/fixtures", "full", workspace); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      "case-001",
		Workspace:   workspace,
		ArgosBinary: "/tmp/argos",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}

	for _, want := range []string{
		"# Argos Write Dogfood Runner Packet",
		"## Write Guidance",
		"## Knowledge Design Template",
		"## CLI Equivalent",
		"argos knowledge design --json --project <project> --intent <intent>",
		"knowledge check --json --design <design-path> --draft <draft-path>",
		"I designed a Go service template",
		"docs/superpowers/templates/argos-write-dogfood-report.md",
	} {
		if !strings.Contains(packet.Markdown, want) {
			t.Fatalf("packet missing %q:\n%s", want, packet.Markdown)
		}
	}
	if packet.WriteGuidance.DesignPath == "" {
		t.Fatalf("packet missing computed write guidance: %#v", packet.WriteGuidance)
	}
	if packet.KnowledgeDesignTemplate.SchemaVersion != knowledgewrite.KnowledgeDesignSchemaVersion {
		t.Fatalf("packet missing knowledge design template: %#v", packet.KnowledgeDesignTemplate)
	}
	assertNoRemovedTerms(t, packet.Markdown)
}

func TestParseMarkdownReportExtractsWriteContract(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(sampleWriteReport("case-001", "pass", "not-run")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	if report.CaseID != "case-001" {
		t.Fatalf("CaseID = %q, want case-001", report.CaseID)
	}
	if report.State != "design_only" || report.NextAction != "write_knowledge_design" {
		t.Fatalf("write guidance not parsed: %#v", report)
	}
	if report.DesignPath != sampleDesignPath() {
		t.Fatalf("DesignPath = %q, want %q", report.DesignPath, sampleDesignPath())
	}
	if report.DraftPath != "" {
		t.Fatalf("DraftPath = %q, want empty", report.DraftPath)
	}
	if report.DraftAllowed || !report.DesignOnly {
		t.Fatalf("unexpected write booleans: %#v", report)
	}
	if !report.Review.DesignApproved || report.Review.DraftWriteApproved {
		t.Fatalf("review decisions not parsed: %#v", report.Review)
	}
	if got := report.Guards["design reviewed before draft write"]; got != ResultPass {
		t.Fatalf("design guard = %q, want pass", got)
	}
}

func TestEvaluateCaseRequiresWorkspaceBackedDesign(t *testing.T) {
	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	if err := SeedFixtureWorkspace("../../testdata/write-golden/fixtures", "full", workspace); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	report, err := ParseMarkdownReport([]byte(sampleWriteReport("case-001", "pass", "not-run")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "design artifact missing") {
		t.Fatalf("missing design artifact finding: %#v", evaluation.Findings)
	}
}

func TestEvaluateCaseEnforcesGuidanceAndReviewBoundary(t *testing.T) {
	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	if err := SeedFixtureWorkspace("../../testdata/write-golden/fixtures", "full", workspace); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	writeComputedDesign(t, workspace, cases[0])
	draftPath := "knowledge/.inbox/packages/mall-api/go-service-template"
	writeDraftPackage(t, workspace, draftPath)
	report, err := ParseMarkdownReport([]byte(sampleWriteReportWithDraft("case-001", draftPath)))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "draft write was not approved") {
		t.Fatalf("missing draft boundary finding: %#v", evaluation.Findings)
	}
}

func TestWriteDogfoodReportTemplateMatchesParserContract(t *testing.T) {
	data, err := os.ReadFile("../../docs/superpowers/templates/argos-write-dogfood-report.md")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	body := string(data)
	for _, want := range []string{
		"## Write Guidance",
		"State:",
		"Next action:",
		"Design path:",
		"Draft path:",
		"Draft allowed:",
		"Design only:",
		"Check result:",
		"## Review Decisions",
		"Design approved:",
		"Draft write approved:",
		"Priority must approved:",
		"Official write approved:",
		"Publish approved:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("template missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range removedAuthoringTermsForTests() {
		if strings.Contains(body, forbidden) {
			t.Fatalf("template contains removed term %q:\n%s", forbidden, body)
		}
	}
	report, err := ParseMarkdownReport(data)
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	if len(report.MissingSections) != 0 {
		t.Fatalf("template missing parser sections: %#v", report.MissingSections)
	}
	if len(report.MissingFields) != 0 {
		t.Fatalf("template missing parser fields: %#v", report.MissingFields)
	}
}

func TestWriteDogfoodChecklistUsesCurrentWorkflow(t *testing.T) {
	data, err := os.ReadFile("../../docs/superpowers/checklists/2026-05-03-argos-write-dogfood-checklist.md")
	if err != nil {
		t.Fatalf("read checklist: %v", err)
	}
	body := string(data)
	for _, want := range []string{
		"dogfood write cases --json",
		"dogfood write packet --case case-001",
		"dogfood write evaluate --case case-001",
		"argos knowledge design --json --project <project> --intent <intent>",
		"argos knowledge check --json --design <design-path> --draft <draft-path>",
		"ROUND_ROOT=$(mktemp -d /tmp/argos-write-dogfood.XXXXXX)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("checklist missing %q:\n%s", want, body)
		}
	}
	assertNoRemovedTerms(t, body)
}

func TestWriteFixtureSeedSupportsPublicCases(t *testing.T) {
	fixtureRoot := "../../testdata/write-golden/fixtures/full"
	for _, rel := range []string{
		"knowledge/domains.yaml",
		"knowledge/projects.yaml",
		"knowledge/types.yaml",
		"knowledge/items/backend/cache-ttl.md",
	} {
		if _, err := os.Stat(filepath.Join(fixtureRoot, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("fixture missing %s: %v", rel, err)
		}
	}

	cases, err := LoadCases(writeCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	for _, tc := range cases {
		for _, source := range tc.Input.AvailableSources {
			if strings.TrimSpace(source.Path) == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(fixtureRoot, filepath.FromSlash(source.Path))); err != nil {
				t.Fatalf("fixture source path %s missing: %v", source.Path, err)
			}
		}
	}
}

func sampleWriteReport(caseID, result, checkResult string) string {
	return "# Argos Write Dogfood Runner Report\n\n" +
		"Case: `" + caseID + "`\n\n" +
		"## Inputs\n\n" +
		"- Workspace: `/tmp/argos-write/ws`\n" +
		"- User request: I designed a Go service template.\n\n" +
		"## Write Guidance\n\n" +
		"- State: `design_only`\n" +
		"- Next action: `write_knowledge_design`\n" +
		"- Design path: `" + sampleDesignPath() + "`\n" +
		"- Draft path: `none`\n" +
		"- Draft allowed: `false`\n" +
		"- Design only: `true`\n" +
		"- Check result: `" + checkResult + "`\n\n" +
		"## Artifacts\n\n" +
		"- Design path: `" + sampleDesignPath() + "`\n" +
		"- Draft path: `none`\n" +
		"- Check result: `" + checkResult + "`\n\n" +
		"## Review Decisions\n\n" +
		"- Design approved: `true`\n" +
		"- Draft write approved: `false`\n" +
		"- Priority must approved: `false`\n" +
		"- Official write approved: `false`\n" +
		"- Publish approved: `false`\n\n" +
		"## Guards\n\n" +
		"- Design reviewed before draft write: pass\n" +
		"- Sources and scope documented: pass\n" +
		"- Future use documented: pass\n" +
		"- Draft stayed in approved area: not-applicable\n" +
		"- Official knowledge unchanged: pass\n" +
		"- Publish not run: pass\n" +
		"- Check run: not-run\n\n" +
		"## Result\n\n" +
		"Result: `" + result + "`\n"
}

func sampleWriteReportWithDraft(caseID, draftPath string) string {
	return strings.Replace(sampleWriteReport(caseID, "pass", "pass"), "- Draft path: `none`", "- Draft path: `"+draftPath+"`", 2)
}

func sampleDesignPath() string {
	return "knowledge/.inbox/designs/mall-api/i-designed-a-go-service-template-turn-it-into-reusable-knowledge/design.json"
}

func writeComputedDesign(t *testing.T, root string, tc Case) knowledgewrite.KnowledgeDesign {
	t.Helper()
	designResponse, err := knowledgewrite.Design(root, designRequest(tc.Input))
	if err != nil {
		t.Fatalf("Design returned error: %v", err)
	}
	design := designResponse.KnowledgeDesignTemplate
	design.Sources.Templates = []string{"templates/go-service"}
	design.Review.DesignApproved = true
	design.Review.DraftWriteApproved = true
	designPath := filepath.Join(root, filepath.FromSlash(designResponse.WriteGuidance.DesignPath))
	if err := os.MkdirAll(filepath.Dir(designPath), 0o755); err != nil {
		t.Fatalf("mkdir design parent: %v", err)
	}
	data, err := json.MarshalIndent(design, "", "  ")
	if err != nil {
		t.Fatalf("marshal design: %v", err)
	}
	if err := os.WriteFile(designPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write design: %v", err)
	}
	return design
}

func writeDraftPackage(t *testing.T, root string, rel string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir draft: %v", err)
	}
	body := `---
id: package:mall-api.go-service-template.v1
type: package
title: Go Service Template
status: draft
priority: should
projects: [mall-api]
tech_domains: [backend]
tags: [go, service]
updated_at: 2026-05-03
applies_to:
  files: ["**/*"]
---

# Go Service Template

## Purpose

Use reviewed Go service template guidance for future backend implementation.

## When To Use

When implementing Go services in Mall API.

## Start Here

Read this package before creating a service.
`
	if err := os.WriteFile(filepath.Join(dir, "KNOWLEDGE.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write draft package: %v", err)
	}
}

func hasEvaluationFinding(evaluation Evaluation, severity string, message string) bool {
	for _, finding := range evaluation.Findings {
		if finding.Severity == severity && strings.Contains(finding.Message, message) {
			return true
		}
	}
	return false
}

func assertNoRemovedTerms(t *testing.T, body string) {
	t.Helper()
	for _, term := range removedAuthoringTermsForTests() {
		if strings.Contains(body, term) {
			t.Fatalf("body contains removed term %q:\n%s", term, body)
		}
	}
}

func removedAuthoringTermsForTests() []string {
	terms := []string{
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
		removedWriteTerm("authoring", ".", "proposal", ".", "v2"),
	}
	sort.Strings(terms)
	return terms
}

func removedWriteTerm(parts ...string) string {
	return strings.Join(parts, "")
}
