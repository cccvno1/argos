package author

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

func TestInspectReportsRegistryPolicyAndNoIndex(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		Mode:       ModeSynthesized,
		FutureTask: "implement product list cache",
		Phase:      "implementation",
		Files:      []string{"internal/catalog/products.go"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Project != "mall-api" || result.Goal != "create product-list cache knowledge" || result.Mode != ModeSynthesized {
		t.Fatalf("unexpected inspect identity: %#v", result)
	}
	if result.Capabilities.Filesystem != "enabled" || result.Capabilities.Index != "unavailable" {
		t.Fatalf("unexpected capabilities: %#v", result.Capabilities)
	}
	if !result.Registry.ProjectKnown {
		t.Fatalf("expected known project: %#v", result.Registry)
	}
	if !reflect.DeepEqual(result.Registry.TechDomains, []string{"backend", "database"}) {
		t.Fatalf("unexpected tech domains: %#v", result.Registry.TechDomains)
	}
	if !reflect.DeepEqual(result.Registry.BusinessDomains, []string{"catalog"}) {
		t.Fatalf("unexpected business domains: %#v", result.Registry.BusinessDomains)
	}
	if result.Policy.Write != "after_proposal_approval" ||
		result.Policy.OfficialMutation != "requires_explicit_review_path" ||
		result.Policy.Promote != "requires_explicit_approval" ||
		result.Policy.PriorityMust != "requires_explicit_authorization" ||
		result.Policy.SynthesizedClaims != "must_mark_assumptions" {
		t.Fatalf("unexpected policy: %#v", result.Policy)
	}
	wantRequirements := []string{
		"schema_version:authoring.proposal.v2",
		"user_request",
		"future_agent_audience",
		"source_profile",
		"future_use",
		"applicability",
		"overlap_decision",
		"delivery",
		"candidate_files",
		"verification_plan",
		"human_review",
	}
	if !reflect.DeepEqual(result.ProposalRequirements, wantRequirements) {
		t.Fatalf("unexpected proposal requirements: %#v", result.ProposalRequirements)
	}
	if len(result.RecommendedNextSteps) != 1 ||
		result.RecommendedNextSteps[0].Step != "write_knowledge_design_proposal" ||
		result.RecommendedNextSteps[0].Reason != "Human review is required before durable writes." {
		t.Fatalf("unexpected next steps: %#v", result.RecommendedNextSteps)
	}
}

func TestInspectIncludesValidProposalV2Scaffold(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Create reusable Redis draft practices for future agents.",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.SchemaVersion != ProposalV2SchemaVersion {
		t.Fatalf("schema = %q, want %q", proposal.SchemaVersion, ProposalV2SchemaVersion)
	}
	if proposal.UserRequest != "Create reusable Redis draft practices for future agents." {
		t.Fatalf("user_request = %q", proposal.UserRequest)
	}
	if proposal.KnowledgeGoal == "" {
		t.Fatalf("knowledge_goal is empty in scaffold: %#v", proposal)
	}
	if proposal.Project != "mall-api" {
		t.Fatalf("project = %q, want mall-api", proposal.Project)
	}
	if proposal.ProposedShape.Kind != "package" {
		t.Fatalf("kind = %q, want package", proposal.ProposedShape.Kind)
	}
	if proposal.ProposedShape.ArtifactState != "candidate" {
		t.Fatalf("artifact_state = %q, want candidate", proposal.ProposedShape.ArtifactState)
	}
	if proposal.VerificationPlan.ValidatePath != proposal.ProposedShape.Path {
		t.Fatalf("validate_path = %q, want proposed path %q", proposal.VerificationPlan.ValidatePath, proposal.ProposedShape.Path)
	}
	if hasFinding(ValidateProposalV2(proposal), "fail", "") {
		t.Fatalf("scaffold has failing findings: %#v", ValidateProposalV2(proposal))
	}

	data, err := json.Marshal(proposal)
	if err != nil {
		t.Fatalf("marshal scaffold: %v", err)
	}
	for _, want := range []string{
		`"proposal_approved":false`,
		`"candidate_write_approved":false`,
		`"priority_must_authorized":false`,
		`"official_mutation_authorized":false`,
		`"promote_authorized":false`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("scaffold JSON missing human review boundary %s: %s", want, string(data))
		}
	}
}

func TestInspectIncludesAuthoringPacketForProposalPath(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "Create reusable Redis draft practices for future agents.",
		FutureTask: "design Redis cache practices",
		Phase:      "planning",
		Query:      "redis cache practices",
		Tags:       []string{"redis", "cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	packet := result.AuthoringPacket
	if packet.State != "ready_for_proposal" {
		t.Fatalf("state = %q, want ready_for_proposal: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_proposal" {
		t.Fatalf("recommended_action = %q, want write_proposal: %#v", packet.RecommendedAction, packet)
	}
	if packet.ProposalPath != "knowledge/.inbox/proposals/mall-api/create-reusable-redis-draft-practices-for-future-agents/proposal.json" {
		t.Fatalf("proposal_path = %q", packet.ProposalPath)
	}
	if packet.CandidatePath != result.ProposalScaffold.ProposedShape.Path {
		t.Fatalf("candidate_path = %q, want scaffold path %q", packet.CandidatePath, result.ProposalScaffold.ProposedShape.Path)
	}
	if packet.CandidateAllowed {
		t.Fatalf("inspect must not authorize candidate writing: %#v", packet)
	}
	if packet.ReviewOnly {
		t.Fatalf("normal packet should not be review-only: %#v", packet)
	}
	if !containsText(packet.StopConditions, "Do not write candidate files until human_review.candidate_write_approved is true.") {
		t.Fatalf("packet missing candidate stop condition: %#v", packet.StopConditions)
	}
	if !containsText(packet.ProposalFocus, "Separate user-stated intent, observed facts, imported facts, synthesized guidance, assumptions, and open questions.") {
		t.Fatalf("packet missing source-separation focus: %#v", packet.ProposalFocus)
	}
	if packet.Commands.VerifyCandidate == "" || !strings.Contains(packet.Commands.VerifyCandidate, "--proposal "+packet.ProposalPath) || !strings.Contains(packet.Commands.VerifyCandidate, "--path "+packet.CandidatePath) {
		t.Fatalf("verify command does not reference proposal and candidate paths: %#v", packet.Commands)
	}
	if len(packet.HumanReviewQuestions) == 0 {
		t.Fatalf("packet should include human review questions: %#v", packet)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal inspect response: %v", err)
	}
	if !strings.Contains(string(data), `"authoring_packet"`) {
		t.Fatalf("inspect JSON missing authoring_packet: %s", string(data))
	}
}

func TestInspectInfersConsumerAudienceForAPIConsumerKnowledge(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "This project is called by many people. I want future agents to understand the business points and interfaces so they can help other developers use it.",
		Files:   []string{"internal/api/README.md"},
		Tags:    []string{"api", "consumer", "business-capability"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.Audience.Primary != "consumer_agent" {
		t.Fatalf("audience.primary = %q, want consumer_agent", proposal.Audience.Primary)
	}
	if !containsText(proposal.Audience.AgentActionsSupported, "help developers use or integrate with the documented project interface") {
		t.Fatalf("consumer scaffold missing consumer action guidance: %#v", proposal.Audience.AgentActionsSupported)
	}
}

func TestInspectKeepsImplementerAudienceForInternalAPISourceWithoutConsumerIntent(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Refactor internal API handlers for implementation.",
		Files:   []string{"internal/api/handler.go"},
		Tags:    []string{"backend"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.Audience.Primary != "implementer_agent" {
		t.Fatalf("audience.primary = %q, want implementer_agent", proposal.Audience.Primary)
	}
}

func TestInspectProposalScaffoldUsesReviewOnlyForUnresolvedOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Cache TTL Rule"))

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Create another cache TTL rule that may overlap existing cache knowledge.",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.Kind != "review" {
		t.Fatalf("kind = %q, want review", proposal.ProposedShape.Kind)
	}
	if proposal.ProposedShape.Type != "review" {
		t.Fatalf("type = %q, want review", proposal.ProposedShape.Type)
	}
	if proposal.ProposedShape.ArtifactState != "review_only" {
		t.Fatalf("artifact_state = %q, want review_only", proposal.ProposedShape.ArtifactState)
	}
	if proposal.OverlapDecision.Decision != "unresolved" {
		t.Fatalf("overlap decision = %q, want unresolved", proposal.OverlapDecision.Decision)
	}
	if !proposal.OverlapDecision.HumanChoiceRequired {
		t.Fatalf("human choice should be required for review-only scaffold: %#v", proposal.OverlapDecision)
	}
	if len(proposal.CandidateFiles) != 0 {
		t.Fatalf("candidate files = %#v, want none", proposal.CandidateFiles)
	}
	if proposal.VerificationPlan.ValidatePath != "" {
		t.Fatalf("validate path = %q, want empty", proposal.VerificationPlan.ValidatePath)
	}
	if hasFinding(ValidateProposalV2(proposal), "fail", "") {
		t.Fatalf("review-only scaffold has failing findings: %#v", ValidateProposalV2(proposal))
	}

	packet := result.AuthoringPacket
	if packet.State != "review_only" {
		t.Fatalf("packet state = %q, want review_only: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_review_only_proposal" {
		t.Fatalf("packet recommended_action = %q, want write_review_only_proposal: %#v", packet.RecommendedAction, packet)
	}
	if !packet.ReviewOnly {
		t.Fatalf("packet should be review-only: %#v", packet)
	}
	if packet.CandidatePath != "" {
		t.Fatalf("review-only packet candidate_path = %q, want empty", packet.CandidatePath)
	}
	if packet.Commands.VerifyCandidate != "" {
		t.Fatalf("review-only packet verify command = %q, want empty", packet.Commands.VerifyCandidate)
	}
	if packet.ProposalPath != proposal.ProposedShape.Path {
		t.Fatalf("proposal_path = %q, want review-only scaffold path %q", packet.ProposalPath, proposal.ProposedShape.Path)
	}
	if !containsText(packet.HumanReviewQuestions, "Is the proposed knowledge goal correct and specific enough?") {
		t.Fatalf("packet should preserve scaffold review questions: %#v", packet.HumanReviewQuestions)
	}
}

func TestInspectUsesReviewOnlyPacketWhenPersonalConventionContentIsMissing(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "I have a personal convention for this project. Preserve it for future agents without making it global truth.",
		Tags:    []string{"personal", "project-convention"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.ArtifactState != "review_only" {
		t.Fatalf("artifact_state = %q, want review_only: %#v", proposal.ProposedShape.ArtifactState, proposal.ProposedShape)
	}
	if proposal.Scope.Distribution != "personal" {
		t.Fatalf("distribution = %q, want personal", proposal.Scope.Distribution)
	}
	if proposal.HumanReview.CandidateWriteApproved {
		t.Fatalf("missing-content scaffold must not approve candidate writing: %#v", proposal.HumanReview)
	}
	if len(proposal.CandidateFiles) != 0 {
		t.Fatalf("missing-content scaffold should not contain candidate files: %#v", proposal.CandidateFiles)
	}

	packet := result.AuthoringPacket
	if packet.State != "review_only" {
		t.Fatalf("packet state = %q, want review_only: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_review_only_proposal" {
		t.Fatalf("packet action = %q, want write_review_only_proposal: %#v", packet.RecommendedAction, packet)
	}
	if !containsText(packet.HumanReviewQuestions, "What exact convention should future agents preserve?") {
		t.Fatalf("packet should ask for exact convention content: %#v", packet.HumanReviewQuestions)
	}
	if !containsText(packet.StopConditions, "Do not write candidate files until human_review.candidate_write_approved is true.") {
		t.Fatalf("packet missing candidate stop condition: %#v", packet.StopConditions)
	}
}

func TestInspectUsesReviewOnlyWhenConventionLabelHasNoContent(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Personal convention: Preserve it for future agents.",
		Tags:    []string{"personal", "project-convention"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.ArtifactState != "review_only" {
		t.Fatalf("artifact_state = %q, want review_only: %#v", proposal.ProposedShape.ArtifactState, proposal.ProposedShape)
	}

	packet := result.AuthoringPacket
	if packet.State != "review_only" {
		t.Fatalf("packet state = %q, want review_only: %#v", packet.State, packet)
	}
	if !containsText(packet.HumanReviewQuestions, "What exact convention should future agents preserve?") {
		t.Fatalf("packet should ask for exact convention content: %#v", packet.HumanReviewQuestions)
	}
}

func TestInspectKeepsConcretePersonalConventionCandidate(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Personal convention: put generated SQL under internal/db/query. Preserve it for future agents.",
		Tags:    []string{"personal", "project-convention"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.ArtifactState != "candidate" {
		t.Fatalf("artifact_state = %q, want candidate: %#v", proposal.ProposedShape.ArtifactState, proposal.ProposedShape)
	}

	packet := result.AuthoringPacket
	if packet.State != "ready_for_proposal" {
		t.Fatalf("packet state = %q, want ready_for_proposal: %#v", packet.State, packet)
	}
	if packet.ReviewOnly {
		t.Fatalf("concrete convention packet should not be review-only: %#v", packet)
	}
	if packet.CandidatePath == "" {
		t.Fatalf("concrete convention packet should include candidate_path: %#v", packet)
	}
}

func TestInspectKeepsNormalAuthoringWithPreserveVocabularyCandidate(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "Create Go template authoring guidance that preserves source trust boundaries for future agents.",
		FutureTask: "document template authoring practices",
		Query:      "template authoring source trust",
		Tags:       []string{"templates", "source-trust"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.ArtifactState != "candidate" {
		t.Fatalf("artifact_state = %q, want candidate: %#v", proposal.ProposedShape.ArtifactState, proposal.ProposedShape)
	}

	packet := result.AuthoringPacket
	if packet.State != "ready_for_proposal" {
		t.Fatalf("packet state = %q, want ready_for_proposal: %#v", packet.State, packet)
	}
	if packet.ReviewOnly {
		t.Fatalf("normal authoring packet should not be review-only: %#v", packet)
	}
	if packet.CandidatePath == "" {
		t.Fatalf("normal authoring packet should include candidate_path: %#v", packet)
	}
}

func TestInspectFindsOfficialAndInboxOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))
	writeAuthorFile(t, root, "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md", authorPackage("package:backend.product-list-cache.v1", "draft", "Product List Cache Package"))

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		FutureTask: "product list cache ttl",
		Query:      "product list cache",
		Tags:       []string{"cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(result.Overlap.Official) == 0 {
		t.Fatalf("expected official overlap: %#v", result.Overlap)
	}
	if len(result.Overlap.Inbox) == 0 {
		t.Fatalf("expected inbox overlap: %#v", result.Overlap)
	}
	if result.Overlap.Official[0].Kind != "official" || result.Overlap.Official[0].ID != "rule:backend.cache.v1" {
		t.Fatalf("unexpected official overlap: %#v", result.Overlap.Official)
	}
	if result.Overlap.Inbox[0].Kind != "inbox" || result.Overlap.Inbox[0].ID != "package:backend.product-list-cache.v1" {
		t.Fatalf("unexpected inbox overlap: %#v", result.Overlap.Inbox)
	}
	if len(result.Overlap.Official[0].Reasons) == 0 || len(result.Overlap.Inbox[0].Reasons) == 0 {
		t.Fatalf("expected overlap reasons: %#v", result.Overlap)
	}
}

func TestInspectIgnoresGenericAuthoringTermsForOverlap(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))

	generic, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "Create reusable knowledge for this project so future agents can help other developers understand it.",
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if hasOverlap(generic.Overlap.Official, "official", "rule:backend.cache.v1") {
		t.Fatalf("generic authoring request should not overlap cache item: %#v", generic.Overlap.Official)
	}

	redisCache, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "Design safe Redis cache draft practices for future agents.",
		FutureTask: "design Redis cache practices",
		Query:      "redis cache",
		Tags:       []string{"redis", "cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if !hasOverlap(redisCache.Overlap.Official, "official", "rule:backend.cache.v1") {
		t.Fatalf("redis cache request should overlap cache item: %#v", redisCache.Overlap.Official)
	}
}

func TestInspectFindsIndexOverlapReadOnly(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)
	writeAuthorFile(t, root, "knowledge/items/backend/cache.md", authorItem("rule:backend.cache.v1", "active", "Product list cache TTL rule"))
	items, err := knowledge.LoadOfficial(root)
	if err != nil {
		t.Fatalf("LoadOfficial returned error: %v", err)
	}
	if err := index.Rebuild(filepath.Join(root, "argos", "index.db"), items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "create product-list cache knowledge",
		FutureTask: "product list cache ttl",
		Query:      "product list cache",
		Tags:       []string{"cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Capabilities.Index != "enabled" {
		t.Fatalf("expected enabled index capability, got %#v", result.Capabilities)
	}
	if !hasOverlap(result.Overlap.Index, "index", "rule:backend.cache.v1") {
		t.Fatalf("expected indexed product-list cache overlap, got %#v", result.Overlap.Index)
	}
}

func TestInspectReportsCandidatePathRisk(t *testing.T) {
	tests := []struct {
		name           string
		candidatePath  string
		wantStatus     string
		wantReviewOnly bool
	}{
		{
			name:           "unsafe escaping path",
			candidatePath:  "../outside",
			wantStatus:     "unsafe",
			wantReviewOnly: true,
		},
		{
			name:          "standard inbox item path",
			candidatePath: "knowledge/.inbox/items/backend/cache.md",
			wantStatus:    "allowed",
		},
		{
			name:          "standard inbox package path",
			candidatePath: "knowledge/.inbox/packages/backend/product-list-cache/KNOWLEDGE.md",
			wantStatus:    "allowed",
		},
		{
			name:           "official item path",
			candidatePath:  "knowledge/items/backend/cache.md",
			wantStatus:     "official_review_required",
			wantReviewOnly: true,
		},
		{
			name:           "official package path",
			candidatePath:  "knowledge/packages/backend/product-list-cache/KNOWLEDGE.md",
			wantStatus:     "official_review_required",
			wantReviewOnly: true,
		},
		{
			name:           "nonstandard authoring path",
			candidatePath:  "tmp/backend/cache.md",
			wantStatus:     "review-needed",
			wantReviewOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeAuthorRegistry(t, root)

			result, err := Inspect(root, InspectRequest{
				Project:       "mall-api",
				Goal:          "create product-list cache knowledge",
				CandidatePath: tt.candidatePath,
			})
			if err != nil {
				t.Fatalf("Inspect returned error: %v", err)
			}
			if result.PathRisk.Status != tt.wantStatus {
				t.Fatalf("expected path risk status %q, got %#v", tt.wantStatus, result.PathRisk)
			}
			if tt.wantReviewOnly {
				if result.ProposalScaffold.ProposedShape.Kind != "review" ||
					result.ProposalScaffold.ProposedShape.Type != "review" ||
					result.ProposalScaffold.ProposedShape.ArtifactState != "review_only" {
					t.Fatalf("expected review-only scaffold for path risk, got %#v", result.ProposalScaffold.ProposedShape)
				}
				if len(result.ProposalScaffold.CandidateFiles) != 0 {
					t.Fatalf("expected no candidate files for review-only path risk, got %#v", result.ProposalScaffold.CandidateFiles)
				}
				if result.ProposalScaffold.VerificationPlan.ValidatePath != "" {
					t.Fatalf("expected empty validate path for review-only path risk, got %q", result.ProposalScaffold.VerificationPlan.ValidatePath)
				}
				if !result.AuthoringPacket.ReviewOnly ||
					result.AuthoringPacket.CandidatePath != "" ||
					result.AuthoringPacket.RecommendedAction != "write_review_only_proposal" {
					t.Fatalf("expected review-only authoring packet for path risk, got %#v", result.AuthoringPacket)
				}
				if result.AuthoringPacket.ProposalPath != result.ProposalScaffold.ProposedShape.Path {
					t.Fatalf("packet proposal path = %q, want scaffold path %q", result.AuthoringPacket.ProposalPath, result.ProposalScaffold.ProposedShape.Path)
				}
				return
			}
			if result.ProposalScaffold.ProposedShape.ArtifactState != "candidate" {
				t.Fatalf("expected candidate scaffold for allowed path, got %#v", result.ProposalScaffold.ProposedShape)
			}
			if result.AuthoringPacket.ReviewOnly {
				t.Fatalf("expected non-review-only authoring packet for allowed path, got %#v", result.AuthoringPacket)
			}
		})
	}
}

func writeAuthorRegistry(t *testing.T, root string) {
	t.Helper()
	writeAuthorFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend, database]\nbusiness_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeAuthorFile(t, root, "knowledge/types.yaml", "types: [rule, decision, lesson, runbook, reference, package]\n")
}

func writeAuthorFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func authorItem(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: rule
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache]
applies_to:
  files: [internal/catalog/**]
---
Product list cache TTL guidance for backend work.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}

func authorPackage(id string, status string, title string) string {
	body := `---
id: ITEM_ID
title: ITEM_TITLE
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: ITEM_STATUS
priority: should
updated_at: 2026-05-02
tags: [cache, redis]
applies_to:
  files: [internal/catalog/**]
---
## Purpose

Product list cache package.

## When To Use

Use when implementing product list cache behavior.

## Start Here

Read this entrypoint before implementation.

## Load On Demand

Load supporting references only when needed.
`
	body = strings.ReplaceAll(body, "ITEM_ID", id)
	body = strings.ReplaceAll(body, "ITEM_TITLE", title)
	body = strings.ReplaceAll(body, "ITEM_STATUS", status)
	return body
}

func hasOverlap(matches []OverlapMatch, kind string, id string) bool {
	for _, match := range matches {
		if match.Kind == kind && match.ID == id {
			return true
		}
	}
	return false
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
