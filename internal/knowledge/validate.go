package knowledge

import (
	"fmt"
	"strings"

	"argos/internal/registry"
)

func ValidateItem(item Item, reg registry.Registry) []error {
	var errs []error

	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", item.Path, fmt.Sprintf(format, args...)))
	}

	if strings.TrimSpace(item.ID) == "" {
		addErr("missing id")
	}
	if strings.TrimSpace(item.Title) == "" {
		addErr("missing title")
	}
	if strings.TrimSpace(item.Type) == "" {
		addErr("missing type")
	} else if !contains(reg.Types, item.Type) {
		addErr("unknown type: %s", item.Type)
	}

	for _, domain := range item.TechDomains {
		if !contains(reg.TechDomains, domain) {
			addErr("unknown tech domain: %s", domain)
		}
	}
	for _, domain := range item.BusinessDomains {
		if !contains(reg.BusinessDomains, domain) {
			addErr("unknown business domain: %s", domain)
		}
	}
	projectIDs := make(map[string]struct{}, len(reg.Projects))
	for _, project := range reg.Projects {
		projectIDs[project.ID] = struct{}{}
	}
	for _, project := range item.Projects {
		if _, ok := projectIDs[project]; !ok {
			addErr("unknown project: %s", project)
		}
	}

	if strings.TrimSpace(item.Status) == "" {
		addErr("missing status")
	} else if !contains([]string{"draft", "active", "deprecated"}, item.Status) {
		addErr("invalid status: %s", item.Status)
	}
	if strings.TrimSpace(item.Priority) == "" {
		addErr("missing priority")
	} else if !contains([]string{"must", "should", "may"}, item.Priority) {
		addErr("invalid priority: %s", item.Priority)
	}
	if strings.TrimSpace(item.UpdatedAt) == "" {
		addErr("missing updated_at")
	}
	if strings.TrimSpace(item.Body) == "" {
		addErr("empty body")
	}

	return errs
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
