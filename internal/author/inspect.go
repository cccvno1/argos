package author

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/registry"
)

type InspectRequest struct {
	Project       string   `json:"project"`
	Goal          string   `json:"goal"`
	Mode          string   `json:"mode,omitempty"`
	FutureTask    string   `json:"future_task,omitempty"`
	Phase         string   `json:"phase,omitempty"`
	Query         string   `json:"query,omitempty"`
	Files         []string `json:"files,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CandidatePath string   `json:"candidate_path,omitempty"`
}

type InspectResponse struct {
	Project              string              `json:"project"`
	Goal                 string              `json:"goal"`
	Mode                 string              `json:"mode,omitempty"`
	Capabilities         InspectCapabilities `json:"capabilities"`
	Registry             InspectRegistry     `json:"registry"`
	Overlap              InspectOverlap      `json:"overlap"`
	PathRisk             PathRisk            `json:"path_risk"`
	Policy               InspectPolicy       `json:"policy"`
	ProposalRequirements []string            `json:"proposal_requirements"`
	ProposalScaffold     ProposalV2          `json:"proposal_scaffold"`
	AuthoringPacket      AuthoringPacket     `json:"authoring_packet"`
	RecommendedNextSteps []InspectNextStep   `json:"recommended_next_steps"`
}

type InspectCapabilities struct {
	Filesystem string `json:"filesystem"`
	Index      string `json:"index"`
}

type InspectRegistry struct {
	ProjectKnown    bool     `json:"project_known"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
}

type InspectOverlap struct {
	Official []OverlapMatch `json:"official"`
	Inbox    []OverlapMatch `json:"inbox"`
	Index    []OverlapMatch `json:"index"`
}

type OverlapMatch struct {
	Kind    string   `json:"kind"`
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Reasons []string `json:"reasons"`
}

type PathRisk struct {
	CandidatePath string `json:"candidate_path"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
}

type InspectPolicy struct {
	Write             string `json:"write"`
	OfficialMutation  string `json:"official_mutation"`
	Promote           string `json:"promote"`
	PriorityMust      string `json:"priority_must"`
	SynthesizedClaims string `json:"synthesized_claims"`
}

type InspectNextStep struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

func Inspect(root string, req InspectRequest) (InspectResponse, error) {
	reg, err := registry.Load(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load registry: %w", err)
	}
	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load official knowledge: %w", err)
	}
	inbox, err := knowledge.LoadInbox(root)
	if err != nil {
		return InspectResponse{}, fmt.Errorf("load inbox knowledge: %w", err)
	}

	indexCapability := indexStatus(root)
	response := InspectResponse{
		Project: strings.TrimSpace(req.Project),
		Goal:    strings.TrimSpace(req.Goal),
		Mode:    strings.TrimSpace(req.Mode),
		Capabilities: InspectCapabilities{
			Filesystem: "enabled",
			Index:      indexCapability,
		},
		Registry: InspectRegistry{
			ProjectKnown:    projectKnown(reg, req.Project),
			TechDomains:     append([]string{}, reg.TechDomains...),
			BusinessDomains: append([]string{}, reg.BusinessDomains...),
		},
		Overlap: InspectOverlap{
			Official: overlapMatches("official", official, req),
			Inbox:    overlapMatches("inbox", inbox, req),
		},
		PathRisk: inspectPathRisk(req.CandidatePath),
		Policy: InspectPolicy{
			Write:             "after_proposal_approval",
			OfficialMutation:  "requires_explicit_review_path",
			Promote:           "requires_explicit_approval",
			PriorityMust:      "requires_explicit_authorization",
			SynthesizedClaims: "must_mark_assumptions",
		},
		ProposalRequirements: []string{
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
		},
		RecommendedNextSteps: []InspectNextStep{
			{Step: "write_knowledge_design_proposal", Reason: "Human review is required before durable writes."},
		},
	}
	if indexCapability == "enabled" {
		response.Overlap.Index = indexOverlap(root, req)
	}
	response.ProposalScaffold = buildProposalScaffold(response, req)
	response.AuthoringPacket = buildAuthoringPacket(response, req)
	return response, nil
}

func buildProposalScaffold(response InspectResponse, req InspectRequest) ProposalV2 {
	project := strings.TrimSpace(response.Project)
	if project == "" {
		project = "project"
	}
	goal := strings.TrimSpace(response.Goal)
	if goal == "" {
		goal = "Design reusable knowledge for future agents."
	}
	slug := slugForAuthoring(goal)
	candidatePath := filepath.ToSlash(filepath.Join("knowledge/.inbox/packages", project, slug))
	reviewPath := filepath.ToSlash(filepath.Join("knowledge/.inbox/proposals", project, slug, "proposal.json"))
	overlapIDs := overlapIDs(response.Overlap.Official, response.Overlap.Inbox, response.Overlap.Index)
	pathRiskBlocks := pathRiskBlocksCandidate(response.PathRisk)
	missingContent := requestLooksMissingSubstantiveContent(req)
	reviewOnly := len(overlapIDs) > 0 || pathRiskBlocks || missingContent

	source := SourceProfileV2{
		UserConfirmed: []string{"User request: " + goal},
		Synthesized:   []string{"Initial proposal scaffold generated by argos author inspect."},
		Assumptions: []string{
			"The runner must replace scaffold wording with reviewed, source-backed knowledge before promotion.",
		},
		Claims: []SourceClaimV2{
			{
				Claim: "The user requested durable reusable knowledge for future agents.",
				Kind:  "fact",
				Trust: "user_stated",
				Source: []string{
					"author inspect request goal",
				},
			},
			{
				Claim:          "The scaffold is draft guidance and requires human review before durable use.",
				Kind:           "recommendation",
				Trust:          "synthesized",
				Source:         []string{"argos author inspect"},
				RequiresReview: true,
			},
		},
	}
	if len(overlapIDs) > 0 {
		source.Observed = []string{"author inspect found existing related knowledge: " + strings.Join(overlapIDs, ", ")}
		source.Claims = append(source.Claims, SourceClaimV2{
			Claim:  "Existing related knowledge must be reviewed before candidate writing.",
			Kind:   "fact",
			Trust:  "observed",
			Source: []string{"argos author inspect overlap"},
		})
	}
	if pathRiskBlocks {
		source.Observed = append(source.Observed, "author inspect blocked candidate path: "+pathRiskReviewReason(response.PathRisk))
		source.Claims = append(source.Claims, SourceClaimV2{
			Claim:  "The supplied candidate path requires review before candidate writing.",
			Kind:   "fact",
			Trust:  "observed",
			Source: []string{"argos author inspect path risk"},
		})
	}
	if missingContent {
		source.Assumptions = []string{"The exact convention content was not provided in the inspect request."}
		source.OpenQuestions = []string{"What exact convention should future agents preserve?"}
		source.Claims = append(source.Claims, SourceClaimV2{
			Claim:          "The exact actionable convention content is missing and must be reviewed before candidate writing.",
			Kind:           "recommendation",
			Trust:          "synthesized",
			Source:         []string{"argos author inspect"},
			RequiresReview: true,
		})
	}

	proposal := ProposalV2{
		SchemaVersion: ProposalV2SchemaVersion,
		UserRequest:   goal,
		KnowledgeGoal: "Design reusable knowledge for future agents: " + goal,
		Project:       project,
		Audience: AudienceV2{
			Primary: "implementer_agent",
			AgentActionsSupported: []string{
				"find and apply the reviewed knowledge during future development work",
				"respect source trust, applicability, and human review boundaries",
			},
		},
		Scope: ScopeV2{
			Projects:       []string{project},
			Stability:      "draft",
			Distribution:   "project",
			SubjectDomains: append([]string{}, response.Registry.BusinessDomains...),
			TechDomains:    append([]string{}, response.Registry.TechDomains...),
			FileGlobs:      []string{"**/*"},
			OutOfScope: []string{
				"official knowledge mutation without explicit authorization",
				"promotion without explicit authorization",
				"priority must without explicit authorization",
			},
		},
		SourceProfile: source,
		ProposedShape: ProposedShapeV2{
			Kind:           "package",
			Type:           "package",
			Title:          titleFromAuthoringGoal(goal),
			ID:             "package:" + project + "." + slug + ".v1",
			Path:           candidatePath,
			Status:         "draft",
			Priority:       "should",
			Rationale:      "A package gives future agents an entrypoint plus room for references, examples, and checklists.",
			EntrypointLoad: "read_before_implementation",
			ArtifactState:  "candidate",
		},
		FutureUse: FutureUseV2{
			TriggerRequests:  []string{goal},
			NegativeTriggers: []string{"Do not use outside the reviewed project scope or before resolving open review questions."},
			QueryPhrases:     []string{goal},
			ExpectedUse:      "Future agents should read this knowledge before implementing the matching task.",
			CitationPolicy:   "cite_after_use",
		},
		Applicability: Applicability{
			WhenToUse:    []string{"When future development work matches the reviewed knowledge goal."},
			WhenNotToUse: []string{"When the task is outside the reviewed scope or relies on unresolved assumptions."},
			Tradeoffs:    []string{"Draft knowledge improves reuse but must keep source trust and assumptions visible."},
		},
		OverlapDecision: OverlapDecisionV2{
			OfficialOverlap:     overlapIDsFromMatches(response.Overlap.Official),
			InboxOverlap:        overlapIDsFromMatches(response.Overlap.Inbox),
			PossibleOverlap:     overlapIDsFromMatches(response.Overlap.Index),
			Decision:            "create_new",
			Reason:              "No blocking overlap was selected by the scaffold.",
			HumanChoiceRequired: false,
		},
		Delivery: DeliveryV2{
			Path:                       "inbox",
			WriteRequiresHumanApproval: true,
			ReviewPacketRequired:       true,
		},
		CandidateFiles: []CandidateFile{
			{
				Path:    filepath.ToSlash(filepath.Join(candidatePath, "KNOWLEDGE.md")),
				Purpose: "package entrypoint",
				Load:    "read_before_implementation",
			},
		},
		VerificationPlan: VerificationPlan{
			ValidatePath: candidatePath,
			FindabilityScenarios: []FindabilityScenario{
				{Project: project, Task: goal, Query: goal},
			},
		},
		HumanReview: HumanReviewV2{
			ReviewQuestions: []string{
				"Is the proposed knowledge goal correct and specific enough?",
				"Are source trust, assumptions, and applicability boundaries accurate?",
			},
		},
	}

	if requestLooksConsumerFacing(req) {
		proposal.Audience.Primary = "consumer_agent"
		proposal.Audience.AgentActionsSupported = []string{
			"help developers use or integrate with the documented project interface",
			"separate observed interface facts from user interpretation before advising consumers",
			"respect source trust, applicability, and human review boundaries",
		}
	}

	if missingContent {
		proposal.Scope.Distribution = "personal"
		proposal.FutureUse.MissingNeeds = []string{"Exact convention content."}
		proposal.HumanReview.ReviewQuestions = []string{"What exact convention should future agents preserve?"}
	}

	if reviewOnly {
		proposal.ProposedShape.Kind = "review"
		proposal.ProposedShape.Type = "review"
		proposal.ProposedShape.ID = "review:" + project + "." + slug
		proposal.ProposedShape.Path = reviewPath
		proposal.ProposedShape.Status = "review"
		proposal.ProposedShape.Priority = "may"
		proposal.ProposedShape.Rationale = "Existing related knowledge requires a human decision before candidate writing."
		if missingContent && len(overlapIDs) > 0 && pathRiskBlocks {
			proposal.ProposedShape.Rationale = "Existing related knowledge, the supplied candidate path, and missing convention content require human review before candidate writing."
		} else if missingContent && len(overlapIDs) > 0 {
			proposal.ProposedShape.Rationale = "Existing related knowledge and missing convention content require human review before candidate writing."
		} else if missingContent && pathRiskBlocks {
			proposal.ProposedShape.Rationale = "The supplied candidate path and missing convention content require human review before candidate writing."
		} else if missingContent {
			proposal.ProposedShape.Rationale = "The exact convention content must be reviewed before candidate writing."
		} else if len(overlapIDs) > 0 && pathRiskBlocks {
			proposal.ProposedShape.Rationale = "Existing related knowledge and the supplied candidate path require human review before candidate writing."
		} else if pathRiskBlocks {
			proposal.ProposedShape.Rationale = "The supplied candidate path requires human review before candidate writing."
		}
		proposal.ProposedShape.EntrypointLoad = "reference_only"
		proposal.ProposedShape.ArtifactState = "review_only"
		proposal.OverlapDecision.Decision = "unresolved"
		proposal.OverlapDecision.Reason = "author inspect found existing related knowledge: " + strings.Join(overlapIDs, ", ")
		if missingContent && len(overlapIDs) > 0 && pathRiskBlocks {
			proposal.OverlapDecision.Reason += "; blocked candidate path: " + pathRiskReviewReason(response.PathRisk) + "; exact convention content is missing"
		} else if missingContent && len(overlapIDs) > 0 {
			proposal.OverlapDecision.Reason += "; exact convention content is missing"
		} else if missingContent && pathRiskBlocks {
			proposal.OverlapDecision.Reason = "author inspect blocked candidate path: " + pathRiskReviewReason(response.PathRisk) + "; exact convention content is missing"
		} else if missingContent {
			proposal.OverlapDecision.Reason = "author inspect needs exact convention content before candidate writing."
		} else if len(overlapIDs) > 0 && pathRiskBlocks {
			proposal.OverlapDecision.Reason += "; blocked candidate path: " + pathRiskReviewReason(response.PathRisk)
		} else if pathRiskBlocks {
			proposal.OverlapDecision.Reason = "author inspect blocked candidate path: " + pathRiskReviewReason(response.PathRisk)
		}
		proposal.OverlapDecision.HumanChoiceRequired = true
		proposal.CandidateFiles = nil
		proposal.VerificationPlan.ValidatePath = ""
		proposal.VerificationPlan.FindabilityScenarios = nil
		proposal.HumanReview.UnresolvedBlockers = nil
		if len(overlapIDs) > 0 {
			proposal.HumanReview.UnresolvedBlockers = append(proposal.HumanReview.UnresolvedBlockers, "Choose create-new, update-existing, merge, or stop before candidate writing.")
		}
		if pathRiskBlocks {
			proposal.HumanReview.UnresolvedBlockers = append(proposal.HumanReview.UnresolvedBlockers, "Provide an allowed inbox candidate path before candidate writing.")
		}
		if missingContent {
			proposal.HumanReview.UnresolvedBlockers = append(proposal.HumanReview.UnresolvedBlockers, "Provide the exact convention content before candidate writing.")
		}
	}

	return proposal
}

func requestLooksConsumerFacing(req InspectRequest) bool {
	text := strings.ToLower(strings.Join(append([]string{
		req.Goal,
		req.FutureTask,
		req.Query,
	}, append(append([]string{}, req.Tags...), req.Files...)...), " "))
	if text == "" {
		return false
	}
	hasConsumerIntent := strings.Contains(text, "interface") ||
		strings.Contains(text, "interfaces") ||
		strings.Contains(text, "consumer") ||
		strings.Contains(text, "called by") ||
		strings.Contains(text, "callers") ||
		strings.Contains(text, "developers use") ||
		strings.Contains(text, "integrate") ||
		strings.Contains(text, "integration") ||
		strings.Contains(text, "business-capability") ||
		strings.Contains(text, "business points")
	hasSourceSignal := strings.Contains(text, "internal/api") ||
		strings.Contains(text, "api") ||
		strings.Contains(text, "consumer") ||
		strings.Contains(text, "interface")
	return hasConsumerIntent && hasSourceSignal
}

func requestLooksMissingSubstantiveContent(req InspectRequest) bool {
	text := strings.ToLower(strings.Join(append([]string{
		req.Goal,
		req.FutureTask,
		req.Query,
	}, append(append([]string{}, req.Tags...), req.Files...)...), " "))
	if text == "" {
		return false
	}
	hasConventionMarker := strings.Contains(text, "personal convention") ||
		strings.Contains(text, "project convention") ||
		strings.Contains(text, "project-convention")
	hasConventionIntent := hasConventionMarker || strings.Contains(text, "preserve it for future agents")
	if !hasConventionIntent || requestHasConcreteConventionDetails(text) {
		return false
	}
	hasVaguePersonalConvention := strings.Contains(text, "i have a personal convention") ||
		strings.Contains(text, "i have a project convention") ||
		strings.Contains(text, "my personal convention") ||
		strings.Contains(text, "my project convention")
	hasVagueConventionReference := hasConventionMarker &&
		(strings.Contains(text, "preserve it") || strings.Contains(text, "future agents"))
	hasGlobalTruthBoundary := strings.Contains(text, "without making it global truth")
	return hasVaguePersonalConvention || hasVagueConventionReference || hasGlobalTruthBoundary
}

func requestHasConcreteConventionDetails(text string) bool {
	for _, marker := range []string{"put", "use", "under", "path"} {
		if hasConcreteConventionMarker(text, marker) {
			return true
		}
	}
	return false
}

func hasConcreteConventionMarker(text string, marker string) bool {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/' || r == ':' || r == ',' || r == '.'
	})
	for i, word := range words {
		if word != marker || i+1 >= len(words) {
			continue
		}
		if !isVagueConventionPronoun(words[i+1]) {
			return true
		}
	}
	return false
}

func isVagueConventionPronoun(word string) bool {
	switch strings.ToLower(strings.TrimSpace(word)) {
	case "it", "this", "that", "them", "these", "those":
		return true
	default:
		return false
	}
}

func overlapIDs(overlap ...[]OverlapMatch) []string {
	var ids []string
	for _, matches := range overlap {
		ids = append(ids, overlapIDsFromMatches(matches)...)
	}
	return uniqueNonEmpty(ids)
}

func overlapIDsFromMatches(matches []OverlapMatch) []string {
	var ids []string
	for _, match := range matches {
		ids = append(ids, strings.TrimSpace(match.ID))
	}
	return uniqueNonEmpty(ids)
}

func slugForAuthoring(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "knowledge-design"
	}
	if len(slug) > 64 {
		slug = strings.Trim(slug[:64], "-")
	}
	if slug == "" {
		return "knowledge-design"
	}
	return slug
}

func titleFromAuthoringGoal(goal string) string {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "Knowledge Design Draft"
	}
	if len(goal) > 80 {
		goal = strings.TrimSpace(goal[:80])
	}
	return strings.TrimRight(goal, ".")
}

func projectKnown(reg registry.Registry, project string) bool {
	project = strings.TrimSpace(project)
	for _, candidate := range reg.Projects {
		if candidate.ID == project {
			return true
		}
	}
	return false
}

func indexStatus(root string) string {
	dbPath := filepath.Join(root, "argos", "index.db")
	info, err := os.Stat(dbPath)
	if err != nil || !info.Mode().IsRegular() {
		return "unavailable"
	}
	store, err := index.OpenReadOnly(dbPath)
	if err != nil {
		return "unavailable"
	}
	defer store.Close()
	if err := store.CheckSchema(); err != nil {
		return "unavailable"
	}
	return "enabled"
}

func inspectPathRisk(path string) PathRisk {
	path = strings.TrimSpace(path)
	if path == "" {
		return PathRisk{Status: "not_checked"}
	}
	if filepath.IsAbs(path) {
		return PathRisk{CandidatePath: path, Status: "unsafe", Reason: "candidate path must be relative"}
	}
	if hasParentSegment(path) {
		return PathRisk{CandidatePath: path, Status: "unsafe", Reason: "candidate path must stay inside workspace"}
	}

	clean := filepath.Clean(path)
	slash := filepath.ToSlash(clean)
	if strings.HasPrefix(slash, "knowledge/.inbox/items/") || strings.HasPrefix(slash, "knowledge/.inbox/packages/") {
		return PathRisk{CandidatePath: clean, Status: "allowed"}
	}
	if strings.HasPrefix(slash, "knowledge/items/") || strings.HasPrefix(slash, "knowledge/packages/") {
		return PathRisk{CandidatePath: clean, Status: "official_review_required", Reason: "official mutation requires explicit review path"}
	}
	return PathRisk{CandidatePath: clean, Status: "review-needed", Reason: "candidate path is outside standard authoring inbox locations"}
}

func pathRiskBlocksCandidate(risk PathRisk) bool {
	switch risk.Status {
	case "", "not_checked", "allowed":
		return false
	default:
		return true
	}
}

func pathRiskReviewReason(risk PathRisk) string {
	if risk.Reason == "" {
		return risk.Status
	}
	return risk.Status + ": " + risk.Reason
}

func hasParentSegment(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func overlapMatches(kind string, items []knowledge.Item, req InspectRequest) []OverlapMatch {
	terms := overlapTerms(req)
	var matches []OverlapMatch
	for _, item := range items {
		reasons := overlapReasons(item, req, terms)
		if len(reasons) == 0 {
			continue
		}
		matches = append(matches, OverlapMatch{
			Kind:    kind,
			ID:      item.ID,
			Title:   item.Title,
			Path:    item.Path,
			Reasons: reasons,
		})
	}
	sortOverlap(matches)
	return matches
}

func overlapTerms(req InspectRequest) []string {
	text := strings.ToLower(strings.Join([]string{req.Goal, req.FutureTask, req.Query}, " "))
	var terms []string
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/' || r == ':' || r == ',' || r == '.'
	}) {
		field = strings.TrimSpace(field)
		if len(field) >= 4 && !isAuthoringOverlapStopword(field) {
			terms = append(terms, field)
		}
	}
	for _, tag := range req.Tags {
		terms = append(terms, strings.ToLower(tag))
	}
	for _, domain := range req.Domains {
		terms = append(terms, strings.ToLower(domain))
	}
	return uniqueNonEmpty(terms)
}

func isAuthoringOverlapStopword(term string) bool {
	switch strings.ToLower(strings.TrimSpace(term)) {
	case "knowledge",
		"future",
		"agents",
		"agent",
		"project",
		"reusable",
		"create",
		"turn",
		"this",
		"other",
		"help",
		"understand",
		"into":
		return true
	default:
		return false
	}
}

func overlapReasons(item knowledge.Item, req InspectRequest, terms []string) []string {
	var reasons []string
	project := strings.TrimSpace(req.Project)
	projectReason := ""
	if containsStringValue(item.Projects, project) {
		projectReason = "project:" + project
	}
	for _, tag := range req.Tags {
		tag = strings.TrimSpace(tag)
		if containsStringValue(item.Tags, tag) {
			reasons = append(reasons, "tag:"+tag)
		}
	}
	for _, domain := range req.Domains {
		domain = strings.TrimSpace(domain)
		if containsStringValue(item.TechDomains, domain) || containsStringValue(item.BusinessDomains, domain) {
			reasons = append(reasons, "domain:"+domain)
		}
	}

	searchText := strings.ToLower(strings.Join([]string{
		item.ID,
		item.Title,
		item.Path,
		strings.Join(item.Tags, " "),
		strings.Join(item.TechDomains, " "),
		strings.Join(item.BusinessDomains, " "),
		item.Body,
	}, " "))
	for _, term := range terms {
		if strings.Contains(searchText, strings.ToLower(term)) {
			reasons = append(reasons, "term:"+term)
		}
	}
	if len(reasons) > 0 && projectReason != "" {
		reasons = append([]string{projectReason}, reasons...)
	}
	return uniqueNonEmpty(reasons)
}

func indexOverlap(root string, req InspectRequest) []OverlapMatch {
	store, err := index.OpenReadOnly(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		return nil
	}
	defer store.Close()
	if err := store.CheckSchema(); err != nil {
		return nil
	}

	matches, err := store.SearchText(strings.Join([]string{req.Goal, req.FutureTask, req.Query}, " "), 10)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []OverlapMatch
	for _, match := range matches {
		if seen[match.ItemID] {
			continue
		}
		seen[match.ItemID] = true
		item, err := store.GetItem(match.ItemID)
		if err != nil {
			continue
		}
		out = append(out, OverlapMatch{
			Kind:    "index",
			ID:      item.ID,
			Title:   item.Title,
			Path:    item.Path,
			Reasons: []string{"index_text_match"},
		})
	}
	sortOverlap(out)
	return out
}

func sortOverlap(matches []OverlapMatch) {
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i].Reasons) != len(matches[j].Reasons) {
			return len(matches[i].Reasons) > len(matches[j].Reasons)
		}
		return matches[i].ID < matches[j].ID
	})
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func containsStringValue(values []string, want string) bool {
	if strings.TrimSpace(want) == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
