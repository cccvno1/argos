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
	return response, nil
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
