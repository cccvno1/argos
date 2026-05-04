package audit

import (
	"strings"

	"argos/internal/knowledge"
	"argos/internal/provenance"
)

type Request struct {
	Project          string
	IncludePublished bool
}

type Result struct {
	Result  string  `json:"result"`
	Summary Summary `json:"summary"`
	Items   []Item  `json:"items"`
}

type Summary struct {
	Open                      int `json:"open"`
	ReadyToPublish            int `json:"ready_to_publish"`
	Blocked                   int `json:"blocked"`
	Problems                  int `json:"problems"`
	Published                 int `json:"published"`
	OfficialMissingProvenance int `json:"official_missing_provenance"`
}

type Item struct {
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	ProvenanceID string `json:"provenance_id,omitempty"`
	Project      string `json:"project,omitempty"`
	KnowledgeID  string `json:"knowledge_id,omitempty"`
	Path         string `json:"path,omitempty"`
	Action       string `json:"action,omitempty"`
}

func Knowledge(root string, req Request) (Result, error) {
	result := Result{Result: "pass"}
	list, err := provenance.List(root, provenance.ListFilter{Project: strings.TrimSpace(req.Project)})
	if err != nil {
		return Result{}, err
	}

	publishedByKnowledgeID := map[string]bool{}
	for _, record := range list.Records {
		if record.State == provenance.StatePublished {
			publishedByKnowledgeID[record.KnowledgeID] = true
		}
		status, err := provenance.Status(root, record.Path)
		if err != nil {
			return Result{}, err
		}
		addStatusItems(&result, status, req.IncludePublished)
	}

	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return Result{}, err
	}
	for _, item := range official {
		if !matchesProject(item, req.Project) || publishedByKnowledgeID[item.ID] {
			continue
		}
		result.Summary.Open++
		result.Summary.Problems++
		result.Summary.OfficialMissingProvenance++
		result.Items = append(result.Items, Item{
			Category:    "official_missing_provenance",
			Severity:    "problem",
			Project:     itemProject(item, req.Project),
			KnowledgeID: item.ID,
			Path:        item.Path,
			Action:      "inspect official knowledge and add provenance through the next change",
		})
	}

	result.Result = resultStatus(result.Summary)
	return result, nil
}

func addStatusItems(result *Result, status provenance.StatusResult, includePublished bool) {
	if status.State == provenance.StatePublished && status.Result == "pass" {
		result.Summary.Published++
		if includePublished {
			result.Items = append(result.Items, Item{
				Category:     "published",
				Severity:     "info",
				ProvenanceID: status.ProvenanceID,
				Project:      status.Subject.Project,
				KnowledgeID:  status.Subject.KnowledgeID,
				Path:         status.Path,
			})
		}
		return
	}

	action := statusAction(status)
	for _, finding := range status.Findings {
		result.Summary.Open++
		switch finding.Severity {
		case "problem":
			result.Summary.Problems++
		case "blocked":
			result.Summary.Blocked++
		}
		result.Items = append(result.Items, Item{
			Category:     finding.Category,
			Severity:     finding.Severity,
			ProvenanceID: status.ProvenanceID,
			Project:      status.Subject.Project,
			KnowledgeID:  status.Subject.KnowledgeID,
			Path:         status.Path,
			Action:       action,
		})
	}

	if status.ReadyToPublish {
		result.Summary.Open++
		result.Summary.ReadyToPublish++
		result.Items = append(result.Items, Item{
			Category:     "ready_to_publish",
			Severity:     "warning",
			ProvenanceID: status.ProvenanceID,
			Project:      status.Subject.Project,
			KnowledgeID:  status.Subject.KnowledgeID,
			Path:         status.Path,
			Action:       action,
		})
	}
}

func statusAction(status provenance.StatusResult) string {
	action := strings.Join(status.Actions, "; ")
	if action == "" {
		return "inspect provenance status"
	}
	return action
}

func resultStatus(summary Summary) string {
	switch {
	case summary.Problems > 0:
		return "problem"
	case summary.Blocked > 0:
		return "blocked"
	case summary.Open > 0 || summary.ReadyToPublish > 0:
		return "warning"
	default:
		return "pass"
	}
}

func matchesProject(item knowledge.Item, project string) bool {
	project = strings.TrimSpace(project)
	if project == "" {
		return true
	}
	for _, candidate := range item.Projects {
		if candidate == project {
			return true
		}
	}
	return false
}

func itemProject(item knowledge.Item, project string) string {
	project = strings.TrimSpace(project)
	if project != "" {
		return project
	}
	if len(item.Projects) == 1 {
		return item.Projects[0]
	}
	return ""
}
