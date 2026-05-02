package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"argos/internal/index"
	"argos/internal/knowledge"
)

type Service struct {
	store *index.Store
}

type StandardsRequest struct {
	Project  string   `json:"project"`
	TaskType string   `json:"task_type"`
	Files    []string `json:"files"`
	Limit    int      `json:"limit"`
}

type ContextRequest struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Files   []string `json:"files"`
}

type FindKnowledgeRequest struct {
	Project           string   `json:"project"`
	Phase             string   `json:"phase"`
	Task              string   `json:"task"`
	Query             string   `json:"query"`
	Files             []string `json:"files"`
	Types             []string `json:"types"`
	Tags              []string `json:"tags"`
	Domains           []string `json:"domains"`
	Status            []string `json:"status"`
	IncludeDeprecated bool     `json:"include_deprecated"`
	Limit             int      `json:"limit"`
}

type ListKnowledgeRequest struct {
	Project           string   `json:"project"`
	Domain            string   `json:"domain"`
	Types             []string `json:"types"`
	IncludeDeprecated bool     `json:"include_deprecated"`
}

type ContextResponse struct {
	Project              string            `json:"project"`
	Phase                string            `json:"phase"`
	Task                 string            `json:"task"`
	Files                []string          `json:"files,omitempty"`
	RecommendedNextSteps []ContextNextStep `json:"recommended_next_steps"`
}

type ContextNextStep struct {
	Tool      string         `json:"tool"`
	Reason    string         `json:"reason"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type FindKnowledgeResponse struct {
	Project      string                      `json:"project"`
	Phase        string                      `json:"phase"`
	Query        string                      `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Support      Support                     `json:"support"`
	Usage        UsageGuidance               `json:"usage"`
	SearchStatus SearchStatus                `json:"search_status"`
	MissingNeeds []MissingNeed               `json:"missing_needs,omitempty"`
	Items        []KnowledgeSummary          `json:"items"`
	NextSteps    []NextStep                  `json:"next_steps"`
}

type ListKnowledgeResponse struct {
	Project   string        `json:"project"`
	Usage     UsageGuidance `json:"usage"`
	Inventory Inventory     `json:"inventory"`
	Groups    []ListGroup   `json:"groups"`
}

type Support struct {
	Level                 string   `json:"level"`
	Confidence            float64  `json:"confidence"`
	Reason                string   `json:"reason"`
	Recommendation        string   `json:"recommendation"`
	MissingKnowledgeHints []string `json:"missing_knowledge_hints,omitempty"`
}

type UsageGuidance struct {
	Read   string `json:"read"`
	Cite   string `json:"cite"`
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}

type SearchStatus struct {
	Semantic SemanticSearchStatus `json:"semantic"`
}

type SemanticSearchStatus struct {
	Status   string `json:"status"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type MissingNeed struct {
	Need        string `json:"need"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	Severity    string `json:"severity"`
	ArgosBacked bool   `json:"argos_backed"`
}

type Inventory struct {
	Types    map[string]int     `json:"types"`
	Domains  []string           `json:"domains"`
	Tags     []string           `json:"tags"`
	Packages []KnowledgeSummary `json:"packages"`
}

type ListGroup struct {
	Key   string             `json:"key"`
	Title string             `json:"title"`
	Items []KnowledgeSummary `json:"items"`
}

type Response struct {
	Items                []ResultItem `json:"items"`
	Conflicts            []ResultItem `json:"conflicts"`
	RecommendedNextSteps []NextStep   `json:"recommended_next_steps"`
}

type ResultItem struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	WhyMatched string `json:"why_matched"`
}

type KnowledgeItemResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Path     string `json:"path"`
	Body     string `json:"body"`
}

type CitationResult struct {
	Citations []Citation `json:"citations"`
	Missing   []string   `json:"missing"`
}

type Citation struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type KnowledgeSummary struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Status          string          `json:"status"`
	Priority        string          `json:"priority"`
	Path            string          `json:"path"`
	Score           float64         `json:"score"`
	ScoreComponents ScoreComponents `json:"score_components"`
	WhyMatched      []string        `json:"why_matched"`
	MatchedSections []string        `json:"matched_sections"`
	ReadStatus      ReadStatus      `json:"read_status"`
	RecommendedStep string          `json:"recommended_step"`
	Body            string          `json:"-"`
}

type ScoreComponents struct {
	Project   float64 `json:"project"`
	FileScope float64 `json:"file_scope"`
	TypePhase float64 `json:"type_phase"`
	Priority  float64 `json:"priority"`
	Status    float64 `json:"status"`
	TagDomain float64 `json:"tag_domain"`
	Lexical   float64 `json:"lexical"`
	Semantic  float64 `json:"semantic"`
}

type ReadStatus struct {
	Level             string `json:"level"`
	FullBodyAvailable bool   `json:"full_body_available"`
	ReadTool          string `json:"read_tool"`
}

type NextStep struct {
	Tool   string   `json:"tool"`
	Reason string   `json:"reason"`
	IDs    []string `json:"ids,omitempty"`
}

type match struct {
	whyMatched string
	fileScoped bool
}

type candidate struct {
	item  knowledge.Item
	match match
}

func New(store *index.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Context(req ContextRequest) ContextResponse {
	project := strings.TrimSpace(req.Project)
	phase := strings.TrimSpace(req.Phase)
	task := strings.TrimSpace(req.Task)
	files := normalizedContextFiles(req.Files)

	reason := "standards are useful before code changes"
	switch phase {
	case "planning":
		reason = "planning should start from active project standards"
	case "implementation":
		reason = "implementation should follow active coding and architecture standards"
	case "review":
		reason = "review should check changes against active standards"
	case "debugging":
		reason = "debugging should account for active standards before changing behavior"
	case "operations", "deployment":
		reason = "operational work should respect active project standards"
	}

	calls := []ContextNextStep{
		{
			Tool:      "argos_find_knowledge",
			Reason:    "find task-relevant Argos knowledge without reading full bodies",
			Arguments: contextFindArguments(project, phase, task, files),
		},
		{
			Tool:      "argos_standards",
			Reason:    reason,
			Arguments: contextStandardsArguments(project, phase, files),
		},
	}
	if contextNeedsInventory(phase, task, files) {
		calls = append([]ContextNextStep{{
			Tool:      "argos_list_knowledge",
			Reason:    "inventory available project knowledge before broad work",
			Arguments: map[string]any{"project": project},
		}}, calls...)
	}

	return ContextResponse{
		Project:              project,
		Phase:                phase,
		Task:                 task,
		Files:                files,
		RecommendedNextSteps: calls,
	}
}

func normalizedContextFiles(files []string) []string {
	var normalized []string
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file != "" {
			normalized = append(normalized, file)
		}
	}
	return normalized
}

func contextFindArguments(project string, phase string, task string, files []string) map[string]any {
	args := map[string]any{
		"project": project,
		"phase":   phase,
		"task":    task,
	}
	if len(files) > 0 {
		args["files"] = files
	}
	return args
}

func contextStandardsArguments(project string, phase string, files []string) map[string]any {
	args := map[string]any{
		"project":   project,
		"task_type": phase,
	}
	if len(files) > 0 {
		args["files"] = files
	}
	return args
}

func contextNeedsInventory(phase string, task string, files []string) bool {
	if len(files) > 0 {
		return false
	}
	task = strings.ToLower(task)
	if phase == "planning" {
		return true
	}
	for _, term := range []string{"understand", "explore", "orient", "map"} {
		if strings.Contains(task, term) {
			return true
		}
	}
	return false
}

func (s *Service) Standards(req StandardsRequest) (Response, error) {
	items, err := s.store.ListItems()
	if err != nil {
		return Response{}, err
	}

	limit := req.Limit
	if limit <= 0 || limit > 5 {
		limit = 5
	}

	var candidates []candidate
	for _, item := range items {
		if item.Type != "rule" || item.Status == "deprecated" {
			continue
		}
		match, ok, err := matchReason(item, req)
		if err != nil {
			return Response{}, err
		}
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{item: item, match: match})
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]

		if priorityRank(left.item.Priority) != priorityRank(right.item.Priority) {
			return priorityRank(left.item.Priority) < priorityRank(right.item.Priority)
		}
		if left.match.fileScoped != right.match.fileScoped {
			return left.match.fileScoped
		}
		return left.item.ID < right.item.ID
	})

	var response Response
	for _, candidate := range candidates {
		if len(response.Items) >= limit {
			break
		}
		item := candidate.item
		response.Items = append(response.Items, ResultItem{
			ID:         item.ID,
			Type:       item.Type,
			Title:      item.Title,
			Summary:    firstSentence(item.Body),
			Priority:   item.Priority,
			Status:     item.Status,
			WhyMatched: candidate.match.whyMatched,
		})
	}

	return response, nil
}

func (s *Service) FindKnowledge(req FindKnowledgeRequest) (FindKnowledgeResponse, error) {
	caps, err := s.store.DiscoveryCapabilities()
	if err != nil {
		return FindKnowledgeResponse{}, err
	}
	items, err := s.store.ListItems()
	if err != nil {
		return FindKnowledgeResponse{}, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	intent := strings.TrimSpace(strings.Join([]string{req.Task, req.Query}, " "))
	textMatches, err := s.store.SearchText(intent, 50)
	if err != nil {
		return FindKnowledgeResponse{}, err
	}
	lexical := lexicalScores(textMatches)
	sections := matchedSections(textMatches)

	var results []KnowledgeSummary
	for _, item := range items {
		if !findCandidateAllowed(item, req) {
			continue
		}
		result, err := knowledgeSummary(item, req, lexical[item.ID], sections[item.ID], intent)
		if err != nil {
			return FindKnowledgeResponse{}, err
		}
		if item.Type == "package" && result.ScoreComponents.Lexical < 0.25 && result.ScoreComponents.FileScope < 1 && result.ScoreComponents.TagDomain <= 0.3 && !contains(req.Types, "package") {
			continue
		}
		if !hasDiscoverySignal(result.ScoreComponents, req) || result.Score <= 0.25 {
			continue
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if priorityRank(results[i].Priority) != priorityRank(results[j].Priority) {
			return priorityRank(results[i].Priority) < priorityRank(results[j].Priority)
		}
		return results[i].ID < results[j].ID
	})
	if len(results) > limit {
		results = results[:limit]
	}

	support := findSupport(results, intent, req)
	nextSteps := findNextSteps(results, support, req.Phase)

	return FindKnowledgeResponse{
		Project:      req.Project,
		Phase:        req.Phase,
		Query:        intent,
		Capabilities: caps,
		Support:      support,
		Usage:        usageGuidance(support),
		SearchStatus: defaultSearchStatus(),
		MissingNeeds: missingNeedsForSupport(support, req, intent, items, lexical),
		Items:        results,
		NextSteps:    nextSteps,
	}, nil
}

func (s *Service) ListKnowledge(req ListKnowledgeRequest) (ListKnowledgeResponse, error) {
	items, err := s.store.ListItems()
	if err != nil {
		return ListKnowledgeResponse{}, err
	}
	inventory := Inventory{
		Types: map[string]int{},
	}
	grouped := map[string][]KnowledgeSummary{}
	domainSet := map[string]bool{}
	tagSet := map[string]bool{}

	for _, item := range items {
		if !listCandidateAllowed(item, req) {
			continue
		}
		inventory.Types[item.Type]++
		for _, domain := range append(append([]string{}, item.TechDomains...), item.BusinessDomains...) {
			domainSet[domain] = true
		}
		for _, tag := range item.Tags {
			tagSet[tag] = true
		}
		summary := knowledgeSummaryFromKnowledge(item)
		if item.Type == "package" {
			inventory.Packages = append(inventory.Packages, summary)
		}
		key := listGroupKey(item)
		grouped[key] = append(grouped[key], summary)
	}

	inventory.Domains = sortedKeys(domainSet)
	inventory.Tags = sortedKeys(tagSet)
	sort.Slice(inventory.Packages, func(i, j int) bool { return inventory.Packages[i].ID < inventory.Packages[j].ID })

	var groups []ListGroup
	for key, groupItems := range grouped {
		sort.Slice(groupItems, func(i, j int) bool { return groupItems[i].ID < groupItems[j].ID })
		groups = append(groups, ListGroup{Key: key, Title: titleFromKey(key), Items: groupItems})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })

	return ListKnowledgeResponse{Project: req.Project, Usage: listUsageGuidance(), Inventory: inventory, Groups: groups}, nil
}

func (s *Service) ReadKnowledge(id string) (KnowledgeItemResult, error) {
	item, err := s.store.GetItem(id)
	if err != nil {
		return KnowledgeItemResult{}, err
	}
	return knowledgeItemResult(item), nil
}

func (s *Service) CiteKnowledge(ids []string) CitationResult {
	var result CitationResult
	for _, id := range ids {
		item, err := s.store.GetItem(id)
		if err != nil {
			result.Missing = append(result.Missing, id)
			continue
		}
		result.Citations = append(result.Citations, Citation{
			ID:     item.ID,
			Title:  item.Title,
			Path:   item.Path,
			Status: item.Status,
		})
	}
	return result
}

func knowledgeItemResult(item knowledge.Item) KnowledgeItemResult {
	return KnowledgeItemResult{
		ID:       item.ID,
		Title:    item.Title,
		Type:     item.Type,
		Status:   item.Status,
		Priority: item.Priority,
		Path:     item.Path,
		Body:     item.Body,
	}
}

func matchReason(item knowledge.Item, req StandardsRequest) (match, bool, error) {
	if !contains(item.Projects, req.Project) {
		return match{}, false, nil
	}

	if len(item.AppliesTo.Files) == 0 {
		return match{whyMatched: fmt.Sprintf("project %s matched", req.Project)}, true, nil
	}

	for _, file := range req.Files {
		for _, pattern := range item.AppliesTo.Files {
			matched, err := doublestar.PathMatch(pattern, file)
			if err != nil {
				return match{}, false, fmt.Errorf("%s: match file scope %q: %w", item.ID, pattern, err)
			}
			if matched {
				return match{
					whyMatched: fmt.Sprintf("project %s and file %s matched %s", req.Project, file, pattern),
					fileScoped: true,
				}, true, nil
			}
		}
	}

	return match{}, false, nil
}

func priorityRank(priority string) int {
	switch priority {
	case "must":
		return 0
	case "should":
		return 1
	case "may":
		return 2
	default:
		return 3
	}
}

func firstSentence(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for i, r := range line {
			if r == '.' || r == '!' || r == '?' {
				return strings.TrimSpace(line[:i+1])
			}
		}
		return line
	}
	return ""
}

func knowledgeSummary(item knowledge.Item, req FindKnowledgeRequest, lexical float64, sections []string, intent string) (KnowledgeSummary, error) {
	fileScope, err := fileScopeScore(item, req.Files)
	if err != nil {
		return KnowledgeSummary{}, fmt.Errorf("%s: match file scope: %w", item.ID, err)
	}
	lexical = minFloat(lexical, lexicalTermScore(item, intent))
	components := ScoreComponents{
		Project:   projectScore(item, req.Project),
		FileScope: fileScope,
		TypePhase: typePhaseScore(item.Type, req.Phase),
		Priority:  priorityScore(item.Priority),
		Status:    statusScore(item.Status),
		TagDomain: tagDomainScore(item, req.Tags, req.Domains),
		Lexical:   lexical,
		Semantic:  0,
	}
	score := weightedScore(components)
	result := knowledgeSummaryFromKnowledge(item)
	result.Score = score
	result.ScoreComponents = components
	result.WhyMatched = whyMatched(item, req, components)
	result.MatchedSections = sections
	result.RecommendedStep = recommendedStep(item, score, req.Phase, relevanceScore(components, req))
	return result, nil
}

func knowledgeSummaryFromKnowledge(item knowledge.Item) KnowledgeSummary {
	return KnowledgeSummary{
		ID:       item.ID,
		Type:     item.Type,
		Title:    item.Title,
		Summary:  firstSentence(item.Body),
		Status:   item.Status,
		Priority: item.Priority,
		Path:     item.Path,
		ReadStatus: ReadStatus{
			Level:             "summary",
			FullBodyAvailable: true,
			ReadTool:          "argos_read_knowledge",
		},
		RecommendedStep: "skim_summary_only",
	}
}

func weightedScore(c ScoreComponents) float64 {
	total := c.Project*0.18 + c.FileScope*0.18 + c.TypePhase*0.14 + c.Priority*0.12 + c.Status*0.08 + c.TagDomain*0.12 + c.Lexical*0.18
	if total > 1 {
		return 1
	}
	return total
}

func boolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func projectScore(item knowledge.Item, project string) float64 {
	if projectMatches(item, project) {
		return 1
	}
	return 0
}

func findCandidateAllowed(item knowledge.Item, req FindKnowledgeRequest) bool {
	if item.Status == "deprecated" && !req.IncludeDeprecated {
		return false
	}
	if !projectMatches(item, req.Project) {
		return false
	}
	if len(req.Types) > 0 && !contains(req.Types, item.Type) {
		return false
	}
	if len(req.Status) > 0 && !contains(req.Status, item.Status) {
		return false
	}
	if len(req.Tags) > 0 && !containsAny(item.Tags, req.Tags) {
		return false
	}
	if len(req.Domains) > 0 && !containsAnyDomain(item, req.Domains) {
		return false
	}
	return true
}

func listCandidateAllowed(item knowledge.Item, req ListKnowledgeRequest) bool {
	if item.Status == "deprecated" && !req.IncludeDeprecated {
		return false
	}
	if !projectMatches(item, req.Project) {
		return false
	}
	if req.Domain != "" && !contains(item.TechDomains, req.Domain) && !contains(item.BusinessDomains, req.Domain) {
		return false
	}
	if len(req.Types) > 0 && !contains(req.Types, item.Type) {
		return false
	}
	return true
}

func fileScopeScore(item knowledge.Item, files []string) (float64, error) {
	if len(item.AppliesTo.Files) == 0 {
		return 0.4, nil
	}
	for _, file := range files {
		for _, pattern := range item.AppliesTo.Files {
			matched, err := doublestar.PathMatch(pattern, file)
			if err != nil {
				return 0, fmt.Errorf("%q: %w", pattern, err)
			}
			if matched {
				return 1, nil
			}
		}
	}
	return 0, nil
}

func typePhaseScore(itemType string, phase string) float64 {
	preferences := map[string][]string{
		"planning":       {"decision", "guide", "package", "reference"},
		"implementation": {"rule", "package", "runbook", "decision"},
		"review":         {"rule", "decision", "lesson"},
		"debugging":      {"lesson", "runbook", "decision"},
		"operations":     {"runbook", "decision", "rule"},
		"deployment":     {"runbook", "decision", "rule"},
	}
	for i, preferred := range preferences[phase] {
		if preferred == itemType {
			return 1 - float64(i)*0.15
		}
	}
	if phase == "" {
		return 0.5
	}
	return 0.2
}

func priorityScore(priority string) float64 {
	switch priority {
	case "must":
		return 1
	case "should":
		return 0.75
	case "may":
		return 0.45
	default:
		return 0.25
	}
}

func statusScore(status string) float64 {
	switch status {
	case "active":
		return 1
	case "draft":
		return 0.65
	default:
		return 0
	}
}

func tagDomainScore(item knowledge.Item, tags []string, domains []string) float64 {
	if len(tags) == 0 && len(domains) == 0 {
		return 0.3
	}
	matches := 0
	total := len(tags) + len(domains)
	for _, tag := range tags {
		if contains(item.Tags, tag) {
			matches++
		}
	}
	for _, domain := range domains {
		if contains(item.TechDomains, domain) || contains(item.BusinessDomains, domain) {
			matches++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(matches) / float64(total)
}

func lexicalScores(matches []index.TextMatch) map[string]float64 {
	scores := map[string]float64{}
	for _, match := range matches {
		if match.Score > scores[match.ItemID] {
			scores[match.ItemID] = match.Score
		}
	}
	return scores
}

func matchedSections(matches []index.TextMatch) map[string][]string {
	seen := map[string]map[string]bool{}
	for _, match := range matches {
		if match.Section == "" {
			continue
		}
		if seen[match.ItemID] == nil {
			seen[match.ItemID] = map[string]bool{}
		}
		seen[match.ItemID][match.Section] = true
	}
	result := map[string][]string{}
	for id, sections := range seen {
		result[id] = sortedKeys(sections)
	}
	return result
}

func whyMatched(item knowledge.Item, req FindKnowledgeRequest, c ScoreComponents) []string {
	var reasons []string
	if c.Project > 0 {
		reasons = append(reasons, fmt.Sprintf("project %s matched", req.Project))
	}
	if c.FileScope >= 1 {
		reasons = append(reasons, "file scope matched applies_to.files")
	}
	if c.TypePhase >= 0.7 {
		reasons = append(reasons, fmt.Sprintf("%s phase prefers %s knowledge", req.Phase, item.Type))
	}
	if c.TagDomain > 0.3 {
		reasons = append(reasons, "requested tags or domains matched")
	}
	if c.Lexical > 0 {
		reasons = append(reasons, "task or query text matched indexed knowledge")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "general project knowledge matched")
	}
	return reasons
}

func recommendedStep(item knowledge.Item, score float64, phase string, relevance float64) string {
	if score < 0.45 || relevance < 0.5 {
		return "skim_summary_only"
	}
	switch phase {
	case "implementation":
		if item.Priority == "must" || item.Type == "package" {
			return "read_full_before_implementation"
		}
	case "review":
		if item.Priority == "must" || item.Type == "decision" {
			return "read_full_before_review"
		}
	case "debugging":
		if item.Type == "lesson" || item.Type == "runbook" {
			return "read_full_before_debugging"
		}
	case "planning":
		if item.Type == "decision" || item.Type == "package" {
			return "read_full_before_planning"
		}
	}
	return "cite_after_read_and_used"
}

func findSupport(items []KnowledgeSummary, intent string, req FindKnowledgeRequest) Support {
	if len(items) == 0 {
		return Support{
			Level:                 "none",
			Confidence:            0,
			Reason:                "No active Argos knowledge matched this request strongly.",
			Recommendation:        "Proceed without Argos-specific claims and do not cite Argos knowledge for this task.",
			MissingKnowledgeHints: missingKnowledgeHints(intent),
		}
	}
	topItem := items[0]
	top := topItem.Score
	if topItem.Type == "lesson" && top >= 0.7 {
		return Support{Level: "partial", Confidence: top, Reason: "Found related Argos knowledge, but task-specific support has missing needs.", Recommendation: "Read only high-confidence IDs and mention missing needs when relevant.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
	if topItem.Type == "reference" && (top >= 0.6 || topItem.ScoreComponents.Lexical >= 0.3) {
		return Support{Level: "partial", Confidence: top, Reason: "Found related Argos knowledge, but task-specific support has missing needs.", Recommendation: "Read only high-confidence IDs and mention missing needs when relevant.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
	if topItem.ScoreComponents.Lexical > 0 && topItem.ScoreComponents.Lexical < 0.5 && topItem.ScoreComponents.FileScope < 1 && topItem.ScoreComponents.TagDomain > 0.3 {
		return Support{Level: "weak", Confidence: top, Reason: "Only broad or low-confidence Argos knowledge matched.", Recommendation: "Skim summaries or inspect the knowledge list; do not treat results as authoritative.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
	if topItem.ScoreComponents.Lexical > 0 && topItem.ScoreComponents.Lexical < 0.3 && topItem.ScoreComponents.FileScope < 1 && topItem.ScoreComponents.TagDomain <= 0.3 {
		return Support{Level: "weak", Confidence: top, Reason: "Only broad or low-confidence Argos knowledge matched.", Recommendation: "Skim summaries or inspect the knowledge list; do not treat results as authoritative.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
	if top < 0.6 && topItem.ScoreComponents.Lexical > 0 && topItem.ScoreComponents.Lexical < 0.5 && topItem.ScoreComponents.FileScope < 1 && topItem.ScoreComponents.TagDomain <= 0.3 {
		return Support{Level: "weak", Confidence: top, Reason: "Only broad or low-confidence Argos knowledge matched.", Recommendation: "Skim summaries or inspect the knowledge list; do not treat results as authoritative.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
	switch {
	case top >= 0.75:
		return Support{Level: "strong", Confidence: top, Reason: "Found active project knowledge matching this request.", Recommendation: "Read high-priority matched knowledge before work."}
	case top >= 0.7 && (topItem.Type != "package" || !packageOnlyRequest(req)) && (topItem.ScoreComponents.Lexical >= 0.4 || topItem.ScoreComponents.FileScope >= 1):
		return Support{Level: "strong", Confidence: top, Reason: "Found active project knowledge matching this request.", Recommendation: "Read high-priority matched knowledge before work."}
	case top >= 0.6:
		return Support{Level: "partial", Confidence: top, Reason: "Found related Argos knowledge, but task-specific support has missing needs.", Recommendation: "Read only high-confidence IDs and mention missing needs when relevant.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	case top >= 0.5:
		return Support{Level: "partial", Confidence: top, Reason: "Found related Argos knowledge, but task-specific support has missing needs.", Recommendation: "Read only high-confidence IDs and mention missing needs when relevant.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	default:
		return Support{Level: "weak", Confidence: top, Reason: "Only broad or low-confidence Argos knowledge matched.", Recommendation: "Skim summaries or inspect the knowledge list; do not treat results as authoritative.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
}

func packageOnlyRequest(req FindKnowledgeRequest) bool {
	return len(req.Types) == 1 && req.Types[0] == "package"
}

func usageGuidance(coverage Support) UsageGuidance {
	switch coverage.Level {
	case "strong":
		return UsageGuidance{
			Read:   "recommended",
			Cite:   "after_read_and_used",
			Claim:  "allowed",
			Reason: "Strong Argos support; read selected items before applying and cite only read knowledge actually used.",
		}
	case "partial":
		return UsageGuidance{
			Read:   "allowed",
			Cite:   "after_read_and_used",
			Claim:  "must_separate_argos_backed_and_general_reasoning",
			Reason: "Partial Argos support; read only relevant shared knowledge and separate Argos-backed claims from general reasoning.",
		}
	case "weak":
		return UsageGuidance{
			Read:   "forbidden",
			Cite:   "forbidden",
			Claim:  "forbidden",
			Reason: "Weak Argos support; inspect summaries only and do not make Argos-backed claims.",
		}
	default:
		return UsageGuidance{
			Read:   "forbidden",
			Cite:   "forbidden",
			Claim:  "forbidden",
			Reason: "No Argos support; use missing knowledge hints as missing needs only and do not cite Argos knowledge.",
		}
	}
}

func listUsageGuidance() UsageGuidance {
	return UsageGuidance{
		Read:   "forbidden",
		Cite:   "forbidden",
		Claim:  "forbidden",
		Reason: "List inventory is for orientation only; do not read, cite, or make task claims from inventory alone.",
	}
}

func findNextSteps(items []KnowledgeSummary, coverage Support, phase string) []NextStep {
	if coverage.Level == "none" || coverage.Level == "weak" {
		return nil
	}
	var ids []string
	for _, item := range items {
		if strings.HasPrefix(item.RecommendedStep, "read_full") {
			ids = append(ids, item.ID)
		}
	}
	var calls []NextStep
	if len(ids) > 0 {
		calls = append(calls, NextStep{Tool: "argos_read_knowledge", Reason: "Read selected knowledge before applying it.", IDs: ids})
	}
	calls = append(calls, NextStep{Tool: "argos_cite_knowledge", Reason: "Cite Argos knowledge IDs actually used in the final response."})
	return calls
}

func missingKnowledgeHints(intent string) []string {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return nil
	}
	return []string{intent + " standard", intent + " decision", intent + " lesson"}
}

func defaultSearchStatus() SearchStatus {
	return SearchStatus{
		Semantic: SemanticSearchStatus{
			Status: "disabled",
			Reason: "semantic provider is not configured",
		},
	}
}

func missingNeedsForSupport(coverage Support, req FindKnowledgeRequest, intent string, items []knowledge.Item, lexical map[string]float64) []MissingNeed {
	if coverage.Level == "strong" || len(coverage.MissingKnowledgeHints) == 0 {
		return nil
	}
	need := missingNeedText(req, intent)
	if need == "" {
		return nil
	}
	source := missingNeedSource(coverage, req, intent, items, lexical)
	severity := missingNeedSeverity(coverage, source)
	return []MissingNeed{{
		Need:        need,
		Reason:      missingNeedReason(coverage, source, need),
		Source:      source,
		Severity:    severity,
		ArgosBacked: false,
	}}
}

func missingNeedText(req FindKnowledgeRequest, intent string) string {
	task := strings.TrimSpace(req.Task)
	query := strings.TrimSpace(req.Query)
	if task == "" {
		return query
	}
	if query == "" {
		return task
	}
	taskLower := strings.ToLower(task)
	queryLower := strings.ToLower(query)
	if strings.Contains(taskLower, queryLower) {
		return task
	}
	if strings.Contains(queryLower, taskLower) {
		return query
	}
	return strings.TrimSpace(task + " / " + query)
}

func missingNeedSource(coverage Support, req FindKnowledgeRequest, intent string, items []knowledge.Item, lexical map[string]float64) string {
	if (coverage.Level == "none" || coverage.Level == "partial") && hasRestrictiveDiscoveryFilters(req) && restrictiveFiltersExcludedRelevantKnowledge(req, intent, items, lexical) {
		return "filtered_out"
	}
	if coverage.Level == "none" && crossDomainMismatchKnowledge(req, intent, items, lexical) {
		return "wrong_scope"
	}
	switch coverage.Level {
	case "partial":
		return "partial_match"
	case "weak":
		return "weak_match"
	default:
		return "not_found"
	}
}

func hasRestrictiveDiscoveryFilters(req FindKnowledgeRequest) bool {
	return len(req.Types) > 0 || len(req.Tags) > 0 || len(req.Domains) > 0 || len(req.Status) > 0
}

func restrictiveFiltersExcludedRelevantKnowledge(req FindKnowledgeRequest, intent string, items []knowledge.Item, lexical map[string]float64) bool {
	unfiltered := req
	unfiltered.Types = nil
	unfiltered.Tags = nil
	unfiltered.Domains = nil
	unfiltered.Status = nil

	for _, item := range items {
		if !findCandidateAllowed(item, unfiltered) || findCandidateAllowed(item, req) {
			continue
		}
		if lexical[item.ID] > 0 || lexicalTermScore(item, intent) > 0 {
			unfilteredResult, err := knowledgeSummary(item, unfiltered, lexical[item.ID], nil, intent)
			if err != nil {
				continue
			}
			unfilteredSupport := findSupport([]KnowledgeSummary{unfilteredResult}, intent, unfiltered)
			if unfilteredSupport.Level == "strong" || unfilteredSupport.Level == "partial" {
				return true
			}
		}
	}
	return false
}

func crossDomainMismatchKnowledge(req FindKnowledgeRequest, intent string, items []knowledge.Item, lexical map[string]float64) bool {
	if req.Project == "" {
		return false
	}
	for _, item := range items {
		if projectMatches(item, req.Project) {
			continue
		}
		if item.Status == "deprecated" && !req.IncludeDeprecated {
			continue
		}
		if maxFloat(lexical[item.ID], lexicalTermScore(item, intent)) >= 0.2 {
			return true
		}
	}
	return false
}

func missingNeedSeverity(coverage Support, source string) string {
	if coverage.Level == "none" {
		return "blocking"
	}
	if source == "weak_match" || source == "partial_match" || source == "filtered_out" {
		return "important"
	}
	return "informational"
}

func missingNeedReason(coverage Support, source string, need string) string {
	switch source {
	case "filtered_out":
		return "Explicit discovery filters excluded shared knowledge that might otherwise match: " + need
	case "partial_match":
		return "Some shared knowledge matched, but it does not fully cover this task need: " + need
	case "weak_match":
		return "Only weak shared knowledge matched, so this need is not Argos-backed: " + need
	case "wrong_scope":
		return "Similar shared knowledge exists, but its project or domain scope does not match this task need: " + need
	default:
		return "No sufficiently relevant shared knowledge matched this task need: " + need
	}
}

func listGroupKey(item knowledge.Item) string {
	if len(item.TechDomains) > 0 {
		if len(item.Tags) > 0 {
			return item.TechDomains[0] + "/" + item.Tags[0]
		}
		return item.TechDomains[0]
	}
	if len(item.BusinessDomains) > 0 {
		return item.BusinessDomains[0]
	}
	return item.Type
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hasDiscoverySignal(c ScoreComponents, req FindKnowledgeRequest) bool {
	if c.Lexical > 0 || c.FileScope >= 1 {
		return true
	}
	if (len(req.Tags) > 0 || len(req.Domains) > 0) && c.TagDomain > 0 {
		return true
	}
	return false
}

func relevanceScore(c ScoreComponents, req FindKnowledgeRequest) float64 {
	score := maxFloat(c.Lexical, c.FileScope)
	if len(req.Tags) > 0 || len(req.Domains) > 0 {
		score = maxFloat(score, c.TagDomain)
	}
	return score
}

func lexicalTermScore(item knowledge.Item, intent string) float64 {
	terms := uniqueTerms(intent)
	if len(terms) == 0 {
		return 0
	}
	text := searchableItemText(item)
	matches := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			matches++
		}
	}
	return float64(matches) / float64(len(terms))
}

func searchableItemText(item knowledge.Item) string {
	parts := []string{
		item.ID,
		item.Title,
		item.Body,
		strings.Join(item.Tags, " "),
		strings.Join(item.TechDomains, " "),
		strings.Join(item.BusinessDomains, " "),
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func uniqueTerms(text string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, term := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_'
	}) {
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
	}
	return terms
}

func projectMatches(item knowledge.Item, project string) bool {
	return project == "" || len(item.Projects) == 0 || contains(item.Projects, project)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func titleFromKey(key string) string {
	words := strings.Fields(strings.ReplaceAll(key, "/", " "))
	for i, word := range words {
		words[i] = titleWord(word)
	}
	return strings.Join(words, " ")
}

func titleWord(word string) string {
	if word == "" {
		return ""
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAny(values []string, requested []string) bool {
	for _, value := range requested {
		if contains(values, value) {
			return true
		}
	}
	return false
}

func containsAnyDomain(item knowledge.Item, domains []string) bool {
	for _, domain := range domains {
		if contains(item.TechDomains, domain) || contains(item.BusinessDomains, domain) {
			return true
		}
	}
	return false
}
