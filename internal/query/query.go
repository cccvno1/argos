package query

import (
	"fmt"
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

type RecommendedCall struct {
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
}

func New(store *index.Store) *Service {
	return &Service{store: store}
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

	var response Response
	for _, item := range items {
		if len(response.Items) >= limit {
			break
		}
		if item.Type != "rule" || item.Status != "active" {
			continue
		}
		whyMatched, ok, err := matchReason(item, req)
		if err != nil {
			return Response{}, err
		}
		if !ok {
			continue
		}
		response.Items = append(response.Items, ResultItem{
			ID:         item.ID,
			Type:       item.Type,
			Title:      item.Title,
			Summary:    firstSentence(item.Body),
			Priority:   item.Priority,
			Status:     item.Status,
			WhyMatched: whyMatched,
		})
	}

	return response, nil
}

func matchReason(item knowledge.Item, req StandardsRequest) (string, bool, error) {
	if !contains(item.Projects, req.Project) {
		return "", false, nil
	}

	if len(item.AppliesTo.Files) == 0 {
		return fmt.Sprintf("project %s matched", req.Project), true, nil
	}

	for _, file := range req.Files {
		for _, pattern := range item.AppliesTo.Files {
			matched, err := doublestar.PathMatch(pattern, file)
			if err != nil {
				return "", false, fmt.Errorf("%s: match file scope %q: %w", item.ID, pattern, err)
			}
			if matched {
				return fmt.Sprintf("project %s and file %s matched %s", req.Project, file, pattern), true, nil
			}
		}
	}

	return "", false, nil
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
