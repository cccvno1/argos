package knowledge

import (
	"fmt"
	"strings"

	"argos/internal/registry"
)

func ValidateItem(item Item, reg registry.Registry) []error {
	var errs []error

	if item.ID == "" {
		errs = append(errs, fmt.Errorf("missing id"))
	}
	if item.Title == "" {
		errs = append(errs, fmt.Errorf("missing title"))
	}
	if item.Type == "" {
		errs = append(errs, fmt.Errorf("missing type"))
	} else if !contains(reg.Types, item.Type) {
		errs = append(errs, fmt.Errorf("unknown type: %s", item.Type))
	}

	for _, domain := range item.TechDomains {
		if !contains(reg.TechDomains, domain) {
			errs = append(errs, fmt.Errorf("unknown tech domain: %s", domain))
		}
	}
	for _, domain := range item.BusinessDomains {
		if !contains(reg.BusinessDomains, domain) {
			errs = append(errs, fmt.Errorf("unknown business domain: %s", domain))
		}
	}
	projectIDs := make(map[string]struct{}, len(reg.Projects))
	for _, project := range reg.Projects {
		projectIDs[project.ID] = struct{}{}
	}
	for _, project := range item.Projects {
		if _, ok := projectIDs[project]; !ok {
			errs = append(errs, fmt.Errorf("unknown project: %s", project))
		}
	}

	if item.Status == "" {
		errs = append(errs, fmt.Errorf("missing status"))
	} else if !contains([]string{"draft", "active", "deprecated"}, item.Status) {
		errs = append(errs, fmt.Errorf("invalid status: %s", item.Status))
	}
	if item.Priority == "" {
		errs = append(errs, fmt.Errorf("missing priority"))
	} else if !contains([]string{"must", "should", "may"}, item.Priority) {
		errs = append(errs, fmt.Errorf("invalid priority: %s", item.Priority))
	}
	if item.UpdatedAt == "" {
		errs = append(errs, fmt.Errorf("missing updated_at"))
	}
	if strings.TrimSpace(item.Body) == "" {
		errs = append(errs, fmt.Errorf("empty body"))
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
