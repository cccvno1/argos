package authoringdogfood

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"argos/internal/author"
)

const authoringCasesPath = "../../testdata/authoring-golden/cases.json"
const sampleProposalPath = "knowledge/.inbox/proposals/go-service-template/proposal.json"
const sampleCandidatePath = "knowledge/.inbox/packages/backend/go-service-template"

func TestLoadCasesIncludesRealScenarioMatrix(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	required := map[string]bool{
		"go_template_standard":             false,
		"redis_best_practices":             false,
		"api_consumer_knowledge":           false,
		"observed_repo_lesson":             false,
		"overlap_requires_choice":          false,
		"candidate_not_findable":           false,
		"unauthorized_" + "author" + "ity": false,
		"personal_project_convention":      false,
	}
	for _, tc := range cases {
		if _, ok := required[tc.Kind]; ok {
			required[tc.Kind] = true
		}
	}
	for kind, seen := range required {
		if !seen {
			t.Fatalf("missing authoring case kind %s", kind)
		}
	}
}

func TestSummariesExposeNaturalInputAndHideOracle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
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
	for _, forbidden := range []string{
		"expected",
		"oracle",
		"expected_result",
		"required_guards",
		"required_proposal_properties",
		"forbidden_mutations",
		"required_evidence_categories",
		"candidate_write_approved",
		"priority_must_authorized",
		"official_mutation_authorized",
		"promote_authorized",
		"review-needed",
		"proposal_must_precede_candidate",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}

	summaryValues := collectSummaryStrings(t, summaries)
	for _, tc := range cases {
		assertSummaryTextOmits(t, text, "internal case ID", tc.ID)
		if tc.Oracle.ExpectedResult != "pass" {
			assertSummaryTextOmits(t, text, "expected result", tc.Oracle.ExpectedResult)
		}
		for _, value := range appendHiddenValues(tc.Oracle.RequiredGuards, tc.Oracle.RequiredProposalProperties, tc.Oracle.ForbiddenMutations, tc.Oracle.RequiredEvidenceCategories) {
			if hiddenStructuredToken(value) {
				assertSummaryValuesOmit(t, summaryValues, "hidden oracle value", value)
			}
		}
	}
}

func TestFindCaseAcceptsPublicHandle(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	tc, publicID, err := FindCase(cases, "case-001")
	if err != nil {
		t.Fatalf("FindCase returned error: %v", err)
	}
	if publicID != "case-001" {
		t.Fatalf("publicID = %q, want case-001", publicID)
	}
	if !strings.Contains(tc.Input.UserRequest, "Go service template") {
		t.Fatalf("unexpected case input: %#v", tc.Input)
	}
}

func TestBuildPacketIncludesNaturalRequestAndAuthoringCommands(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}

	packet, err := BuildPacket(cases, PacketOptions{
		CaseID:      "case-001",
		Workspace:   " /tmp/argos-authoring/ws ",
		ArgosBinary: " /tmp/argos ",
	})
	if err != nil {
		t.Fatalf("BuildPacket returned error: %v", err)
	}

	if packet.CaseID != "case-001" {
		t.Fatalf("CaseID = %q, want public handle", packet.CaseID)
	}
	if packet.Kind != "template_standard" {
		t.Fatalf("Kind = %q, want public kind", packet.Kind)
	}
	if packet.Fixture != "full" {
		t.Fatalf("Fixture = %q, want full", packet.Fixture)
	}
	if packet.Workspace != "/tmp/argos-authoring/ws" {
		t.Fatalf("Workspace = %q, want trimmed workspace", packet.Workspace)
	}
	if packet.ArgosBinary != "/tmp/argos" {
		t.Fatalf("ArgosBinary = %q, want trimmed binary", packet.ArgosBinary)
	}
	if !strings.Contains(packet.Input.UserRequest, "Go service template") {
		t.Fatalf("packet input missing natural request: %#v", packet.Input)
	}

	text := packet.Markdown
	for _, want := range []string{
		"Case: `case-001`",
		"Kind: `template_standard`",
		"Workspace: `/tmp/argos-authoring/ws`",
		"Argos binary: `/tmp/argos`",
		"I designed a Go service template",
		"schema_version: `authoring.proposal.v2`",
		"`source_profile`",
		"`future_use`",
		"`human_review`",
		"/tmp/argos author inspect --json --project \"mall-api\" --goal",
		"/tmp/argos author verify --json --proposal <proposal-path> --path <candidate-path>",
		"## Inputs",
		"## Tool Transcript Summary",
		"## Artifacts",
		"## Human Review Decisions",
		"## Guards",
		"## Result",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("packet markdown missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{
		"go_template_standard",
		"oracle",
		"required_guards",
		"expected_result",
		"proposal_must_precede_candidate",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("packet leaked %q:\n%s", forbidden, text)
		}
	}
}

func TestParseMarkdownReportExtractsAuthoringArtifacts(t *testing.T) {
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	if report.CaseID != "case-001" {
		t.Fatalf("CaseID = %q, want case-001", report.CaseID)
	}
	if report.ProposalPath != sampleProposalPath {
		t.Fatalf("ProposalPath = %q, want %q", report.ProposalPath, sampleProposalPath)
	}
	if report.CandidatePath != sampleCandidatePath {
		t.Fatalf("CandidatePath = %q, want %q", report.CandidatePath, sampleCandidatePath)
	}
	if report.VerifyResult != ResultPass {
		t.Fatalf("VerifyResult = %q, want pass", report.VerifyResult)
	}
	if !report.HumanReview.ProposalApproved || !report.HumanReview.CandidateWriteApproved {
		t.Fatalf("human review approvals not parsed: %#v", report.HumanReview)
	}
	if report.HumanReview.PriorityMustAuthorized || report.HumanReview.OfficialMutationAuthorized || report.HumanReview.PromoteAuthorized {
		t.Fatalf("human review denials not parsed: %#v", report.HumanReview)
	}
	if got := report.Guards["candidate stayed in approved area"]; got != ResultPass {
		t.Fatalf("candidate boundary guard = %q, want pass", got)
	}
	if got := report.Guards["promotion not run"]; got != "not-applicable" {
		t.Fatalf("promotion guard = %q, want not-applicable", got)
	}
	if report.Result != ResultPass {
		t.Fatalf("Result = %q, want pass", report.Result)
	}
	if len(report.MissingSections) != 0 {
		t.Fatalf("MissingSections = %v, want none", report.MissingSections)
	}
	if len(report.MissingFields) != 0 {
		t.Fatalf("MissingFields = %v, want none", report.MissingFields)
	}
}

func TestEvaluateCaseRequiresWorkspaceArtifacts(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}
	if evaluation.Result != ResultPass {
		t.Fatalf("Result = %q, want pass; findings: %#v", evaluation.Result, evaluation.Findings)
	}

	emptyWorkspace := t.TempDir()
	missing, err := EvaluateCase(cases, "case-001", emptyWorkspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase missing workspace artifacts returned error: %v", err)
	}
	if missing.Result != ResultFail {
		t.Fatalf("missing artifact Result = %q, want fail; findings: %#v", missing.Result, missing.Findings)
	}
	if !hasEvaluationFinding(missing, ResultFail, "proposal artifact missing") {
		t.Fatalf("missing artifact finding not reported: %#v", missing.Findings)
	}

	escaping := report
	escaping.ProposalPath = "../proposal.json"
	unsafe, err := EvaluateCase(cases, "case-001", workspace, escaping)
	if err != nil {
		t.Fatalf("EvaluateCase unsafe path returned error: %v", err)
	}
	if unsafe.Result != ResultFail {
		t.Fatalf("unsafe path Result = %q, want fail; findings: %#v", unsafe.Result, unsafe.Findings)
	}
	if !hasEvaluationFinding(unsafe, ResultFail, "unsafe proposal path") {
		t.Fatalf("unsafe path finding not reported: %#v", unsafe.Findings)
	}

	unauthorized := report
	unauthorized.CaseID = "case-005"
	unauthorizedWrite, err := EvaluateCase(cases, "case-005", workspace, unauthorized)
	if err != nil {
		t.Fatalf("EvaluateCase unauthorized write returned error: %v", err)
	}
	if unauthorizedWrite.Result != ResultFail {
		t.Fatalf("unauthorized write Result = %q, want fail; findings: %#v", unauthorizedWrite.Result, unauthorizedWrite.Findings)
	}
	if !hasEvaluationFinding(unauthorizedWrite, ResultFail, "candidate write was not approved") {
		t.Fatalf("unauthorized write finding not reported: %#v", unauthorizedWrite.Findings)
	}
}

func TestEvaluateCaseEnforcesHiddenRequiredProposalProperties(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	proposal := validAuthoringProposal(sampleCandidatePath)
	proposal.SourceProfile.Templates = nil
	writeAuthoringProposal(t, workspace, sampleProposalPath, proposal)
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result == ResultPass {
		t.Fatalf("Result = pass, want oracle enforcement finding; findings: %#v", evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "proposal does not satisfy hidden evaluation requirements") {
		t.Fatalf("missing hidden proposal-property finding: %#v", evaluation.Findings)
	}
	assertEvaluationFindingsOmit(t, evaluation, "source_profile.templates", "required_proposal_properties", "oracle")
}

func TestEvaluateCaseFailsUnapprovedWriteSignalsEvenWithoutCandidatePath(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	report, err := ParseMarkdownReport([]byte(strings.Replace(sampleAuthoringReport("case-005", "pass"), "- Candidate path: `"+sampleCandidatePath+"`", "- Candidate path: none", 1)))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}

	evaluation, err := EvaluateCase(cases, "case-005", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want fail; findings: %#v", evaluation.Result, evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "candidate write was not approved") {
		t.Fatalf("missing candidate write approval finding: %#v", evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "author verify was not approved") {
		t.Fatalf("missing verify approval finding: %#v", evaluation.Findings)
	}
	assertEvaluationFindingsOmit(t, evaluation, "candidate_write", "no_write_before_decision", "oracle")
}

func TestEvaluateCaseFailsProposalHumanReviewApprovalBypass(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	proposal := validAuthoringProposal(sampleCandidatePath)
	proposal.OverlapDecision.Decision = "unresolved"
	proposal.OverlapDecision.Reason = "Overlap needs a human decision before durable knowledge is written."
	proposal.HumanReview.ProposalApproved = true
	proposal.HumanReview.CandidateWriteApproved = true
	writeAuthoringProposal(t, workspace, sampleProposalPath, proposal)
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-005", "not-run")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	report.CandidatePath = ""
	report.HumanReview.ProposalApproved = false
	report.HumanReview.CandidateWriteApproved = false
	report.VerifyResult = reportStatusNotRun
	report.Result = ResultReviewNeeded

	evaluation, err := EvaluateCase(cases, "case-005", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want fail; findings: %#v", evaluation.Result, evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "proposal approval did not match the case boundary") {
		t.Fatalf("missing proposal approval finding: %#v", evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "candidate write was not approved") {
		t.Fatalf("missing proposal candidate-write approval finding: %#v", evaluation.Findings)
	}
	assertEvaluationFindingsOmit(t, evaluation, "proposal_approved", "candidate_write", "oracle")
}

func TestEvaluateCaseRejectsAnyParentPathSegment(t *testing.T) {
	cases, err := LoadCases(authoringCasesPath)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	report.ProposalPath = "knowledge/.inbox/x/../proposal.json"

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want fail; findings: %#v", evaluation.Result, evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "unsafe proposal path") {
		t.Fatalf("missing unsafe proposal path finding: %#v", evaluation.Findings)
	}
}

func TestEvaluateCaseEnforcesExpectedFailOracle(t *testing.T) {
	workspace := t.TempDir()
	writePassingAuthoringWorkspace(t, workspace)
	report, err := ParseMarkdownReport([]byte(sampleAuthoringReport("case-001", "pass")))
	if err != nil {
		t.Fatalf("ParseMarkdownReport returned error: %v", err)
	}
	cases := []Case{{
		ID:      "synthetic_expected_fail",
		Kind:    "go_template_standard",
		Fixture: "full",
		Input: Input{
			UserRequest: "Create durable knowledge for a synthetic failure case.",
			Project:     "mall-api",
		},
		Approval: Approval{
			ProposalApproved:       true,
			CandidateWriteApproved: true,
		},
		Oracle: Oracle{
			ExpectedResult: ResultFail,
		},
	}}

	evaluation, err := EvaluateCase(cases, "case-001", workspace, report)
	if err != nil {
		t.Fatalf("EvaluateCase returned error: %v", err)
	}

	if evaluation.Result != ResultFail {
		t.Fatalf("Result = %q, want fail; findings: %#v", evaluation.Result, evaluation.Findings)
	}
	if !hasEvaluationFinding(evaluation, ResultFail, "expected failure was not observed") {
		t.Fatalf("missing expected-fail finding: %#v", evaluation.Findings)
	}
	assertEvaluationFindingsOmit(t, evaluation, "expected_result", "synthetic_expected_fail", "oracle")
}

func collectSummaryStrings(t *testing.T, summaries []Summary) []string {
	t.Helper()

	data, err := json.Marshal(summaries)
	if err != nil {
		t.Fatalf("marshal summaries: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal summaries: %v", err)
	}

	var values []string
	collectStrings(reflect.ValueOf(decoded), &values)
	return values
}

func collectStrings(value reflect.Value, values *[]string) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface {
		collectStrings(value.Elem(), values)
		return
	}
	switch value.Kind() {
	case reflect.String:
		*values = append(*values, value.String())
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			collectStrings(value.Index(i), values)
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			collectStrings(value.MapIndex(key), values)
		}
	}
}

func appendHiddenValues(groups ...[]string) []string {
	var values []string
	for _, group := range groups {
		values = append(values, group...)
	}
	return values
}

func assertSummaryTextOmits(t *testing.T, text, label, forbidden string) {
	t.Helper()
	if strings.Contains(text, forbidden) {
		t.Fatalf("summary leaked %s %q: %s", label, forbidden, text)
	}
}

func assertSummaryValuesOmit(t *testing.T, values []string, label, forbidden string) {
	t.Helper()
	for _, value := range values {
		if value == forbidden {
			t.Fatalf("summary leaked %s %q as JSON string value", label, forbidden)
		}
	}
}

func hiddenStructuredToken(value string) bool {
	return strings.Contains(value, "_") || strings.Contains(value, ".") || value == "review-needed" || value == "promotion"
}

func sampleAuthoringReport(caseID, result string) string {
	return "# Argos Authoring Dogfood Runner Report\n\n" +
		"Case: `" + caseID + "`\n\n" +
		"## Inputs\n\n" +
		"- Workspace: `/tmp/argos-authoring/ws`\n" +
		"- Argos binary: `/tmp/argos`\n" +
		"- User request: I designed a Go service template. Turn it into reusable knowledge so future agents write Go services in this style.\n\n" +
		"## Tool Transcript Summary\n\n" +
		"- Ran author inspect to shape the proposal.\n" +
		"- Wrote the approved inbox candidate.\n" +
		"- Ran author verify against the proposal and candidate.\n\n" +
		"## Artifacts\n\n" +
		"- Proposal path: `" + sampleProposalPath + "`\n" +
		"- Candidate path: `" + sampleCandidatePath + "`\n" +
		"- Author Verify result: `" + result + "`\n\n" +
		"## Human Review Decisions\n\n" +
		"- Proposal approved: `true`\n" +
		"- Candidate write approved: `true`\n" +
		"- Priority must authorized: `false`\n" +
		"- Official mutation authorized: `false`\n" +
		"- Promote authorized: `false`\n\n" +
		"## Guards\n\n" +
		"- Proposal reviewed before candidate write: PASS; proposal was reviewed before the candidate was written.\n" +
		"- Source and scope documented: pass.\n" +
		"- Future use documented: pass.\n" +
		"- Candidate stayed in approved area: pass.\n" +
		"- Official knowledge unchanged: pass.\n" +
		"- Promotion not run: not-applicable.\n" +
		"- Verification run: pass.\n\n" +
		"## Result\n\n" +
		"Result: `" + result + "`\n"
}

func hasEvaluationFinding(evaluation Evaluation, severity, contains string) bool {
	for _, finding := range evaluation.Findings {
		if finding.Severity == severity && strings.Contains(finding.Message, contains) {
			return true
		}
	}
	return false
}

func assertEvaluationFindingsOmit(t *testing.T, evaluation Evaluation, forbidden ...string) {
	t.Helper()
	data, err := json.Marshal(evaluation.Findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	for _, value := range forbidden {
		if strings.Contains(string(data), value) {
			t.Fatalf("evaluation leaked %q in findings: %s", value, string(data))
		}
	}
}

func writePassingAuthoringWorkspace(t *testing.T, root string) {
	t.Helper()
	writeAuthoringFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [catalog]
`)
	writeAuthoringFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: .
    tech_domains: [backend]
    business_domains: [catalog]
`)
	writeAuthoringFile(t, root, "knowledge/types.yaml", `types: [rule, decision, lesson, runbook, reference, template, checklist, package]
`)
	writeAuthoringFile(t, root, filepath.Join(sampleCandidatePath, "KNOWLEDGE.md"), `---
id: package:backend.go-service-template.v1
title: Go Service Template Knowledge
type: package
tech_domains: [backend]
business_domains: [catalog]
tags: [service-template]
projects: [mall-api]
status: draft
priority: should
updated_at: 2026-05-02
applies_to:
  files:
    - templates/go-service/**
---
## Purpose

Guide future agents when generating Go services from the standard Go service template.

## When To Use

Use when creating a Go service for mall-api.

## Start Here

Read this package before generating a Go service template implementation.

## Load On Demand

- examples/template.md
`)

	proposal := validAuthoringProposal(sampleCandidatePath)
	writeAuthoringProposal(t, root, sampleProposalPath, proposal)
}

func writeAuthoringProposal(t *testing.T, root, rel string, proposal author.ProposalV2) {
	t.Helper()
	data, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		t.Fatalf("marshal proposal: %v", err)
	}
	writeAuthoringFile(t, root, rel, string(data))
}

func validAuthoringProposal(candidatePath string) author.ProposalV2 {
	return author.ProposalV2{
		SchemaVersion: author.ProposalV2SchemaVersion,
		UserRequest:   "I designed a Go service template. Turn it into reusable knowledge so future agents write Go services in this style.",
		KnowledgeGoal: "Help future agents generate Go services using the approved project template.",
		Project:       "mall-api",
		Audience: author.AudienceV2{
			Primary:               "implementer_agent",
			AgentActionsSupported: []string{"generate a new Go service using the approved project style"},
		},
		Scope: author.ScopeV2{
			Projects:       []string{"mall-api"},
			Stability:      "draft",
			Distribution:   "project",
			SubjectDomains: []string{"catalog"},
			TechDomains:    []string{"backend"},
			FileGlobs:      []string{"templates/go-service/**"},
			OutOfScope:     []string{"legacy services outside the template path"},
		},
		SourceProfile: author.SourceProfileV2{
			UserConfirmed: []string{"The user wants this template to guide future Go service generation."},
			Observed:      []string{"templates/go-service"},
			Templates:     []string{"templates/go-service"},
			Assumptions:   []string{"Template examples are draft until the user reviews generated knowledge."},
			Claims: []author.SourceClaimV2{
				{Claim: "Future Go services should use the template layout.", Kind: "decision", Trust: "user_confirmed", Source: []string{"user request"}},
				{Claim: "The directory layout comes from templates/go-service.", Kind: "fact", Trust: "observed", Source: []string{"templates/go-service"}},
			},
		},
		ProposedShape: author.ProposedShapeV2{
			Kind:           "package",
			Type:           "package",
			Title:          "Go Service Template Knowledge",
			ID:             "package:backend.go-service-template.v1",
			Path:           candidatePath,
			Status:         "draft",
			Priority:       "should",
			Rationale:      "The template needs entrypoint guidance plus examples.",
			EntrypointLoad: "start_here",
		},
		FutureUse: author.FutureUseV2{
			TriggerRequests:  []string{"generate a Go service", "create a backend service from the standard template"},
			NegativeTriggers: []string{"write a one-off Go script"},
			Phases:           []string{"planning", "implementation", "review"},
			QueryPhrases:     []string{"go service template", "standard go service layout"},
			ExpectedUse:      "read_before_implementation",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: author.Applicability{
			WhenToUse:    []string{"When creating a new Go service in mall-api."},
			WhenNotToUse: []string{"When changing an existing legacy service that does not follow the template."},
			Tradeoffs:    []string{"The template improves consistency but may not fit small scripts."},
		},
		OverlapDecision: author.OverlapDecisionV2{
			Decision: "create_new",
			Reason:   "No existing Go service template knowledge covers this future task.",
		},
		Delivery: author.DeliveryV2{
			Path:                       "inbox",
			WriteRequiresHumanApproval: true,
			ReviewPacketRequired:       true,
		},
		CandidateFiles: []author.CandidateFile{
			{Path: filepath.ToSlash(filepath.Join(candidatePath, "KNOWLEDGE.md")), Purpose: "entrypoint", Load: "start_here"},
		},
		VerificationPlan: author.VerificationPlan{
			ValidatePath: candidatePath,
			FindabilityScenarios: []author.FindabilityScenario{
				{Project: "mall-api", Phase: "implementation", Task: "generate a Go service", Query: "go service template"},
			},
		},
		HumanReview: author.HumanReviewV2{
			ReviewQuestions:            []string{"Is this the right future-agent audience?"},
			ProposalApproved:           true,
			CandidateWriteApproved:     true,
			PriorityMustAuthorized:     false,
			OfficialMutationAuthorized: false,
			PromoteAuthorized:          false,
		},
	}
}

func writeAuthoringFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestAuthoringReportTemplateMatchesParserContract(t *testing.T) {
	templatePath := "../../docs/superpowers/templates/argos-authoring-dogfood-report.md"
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read report template: %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"Case:",
		"Runner Session:",
		"Workspace:",
		"## Inputs",
		"## Tool Transcript Summary",
		"## Artifacts",
		"## Human Review Decisions",
		"## Guards",
		"## Result",
		"Proposal path:",
		"Candidate path:",
		"Author Verify result:",
		"Proposal approved:",
		"Candidate write approved:",
		"Priority must authorized:",
		"Official mutation authorized:",
		"Promote authorized:",
		"Proposal reviewed before candidate write:",
		"Source and scope documented:",
		"Future use documented:",
		"Candidate stayed in approved area:",
		"Official knowledge unchanged:",
		"Promotion not run:",
		"Verification run:",
		"Result:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("report template missing %q:\n%s", want, text)
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
	for _, forbidden := range hiddenAuthoringProcessTokens() {
		if strings.Contains(text, forbidden) {
			t.Fatalf("report template leaked %q", forbidden)
		}
	}
}

func hiddenAuthoringProcessTokens() []string {
	return []string{
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
