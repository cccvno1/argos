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

type ContextResponse struct {
	Project              string            `json:"project"`
	Phase                string            `json:"phase"`
	RecommendedNextCalls []RecommendedCall `json:"recommended_next_calls"`
}

type Response struct {
	Items                []ResultItem      `json:"items"`
	Conflicts            []ResultItem      `json:"conflicts"`
	RecommendedNextCalls []RecommendedCall `json:"recommended_next_calls"`
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

type RecommendedCall struct {
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
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
	calls := []RecommendedCall{{
		Tool:   "argos_requirements",
		Reason: "workflow start should collect constraints",
	}}

	switch req.Phase {
	case "implementation", "review":
		calls = append(calls, RecommendedCall{
			Tool:   "argos_standards",
			Reason: "implementation and review require active rules",
		})
	case "debugging":
		calls = append(calls, RecommendedCall{
			Tool:   "argos_risks",
			Reason: "debugging should check lessons and incident history",
		})
	case "operations", "deployment":
		calls = append(calls, RecommendedCall{
			Tool:   "argos_operations",
			Reason: "operations should use runbooks",
		})
	default:
		calls = append(calls, RecommendedCall{
			Tool:   "argos_standards",
			Reason: "standards are useful before code changes",
		})
	}

	return ContextResponse{
		Project:              req.Project,
		Phase:                req.Phase,
		RecommendedNextCalls: calls,
	}
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

func (s *Service) GetKnowledgeItem(id string) (KnowledgeItemResult, error) {
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
