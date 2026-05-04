package registry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type projectsFile struct {
	Projects []Project `yaml:"projects"`
}

func ListProjects(root string) ([]Project, error) {
	file, err := loadProjectsFile(root)
	if err != nil {
		return nil, err
	}
	return copyProjects(file.Projects), nil
}

func AddProject(root string, project Project) error {
	project = normalizeProject(project)
	if err := validateProjectRequired(project); err != nil {
		return err
	}

	reg, err := Load(root)
	if err != nil {
		return err
	}
	for _, existing := range reg.Projects {
		if strings.TrimSpace(existing.ID) == project.ID {
			return fmt.Errorf("project already exists: %s", project.ID)
		}
	}
	if err := validateProjectDomains(reg, project); err != nil {
		return err
	}

	file, err := loadProjectsFile(root)
	if err != nil {
		return err
	}
	file.Projects = append(file.Projects, project)
	return writeProjectsFile(root, file)
}

func loadProjectsFile(root string) (projectsFile, error) {
	var file projectsFile
	if err := readYAML(projectsPath(root), &file); err != nil {
		return projectsFile{}, err
	}
	return file, nil
}

func writeProjectsFile(root string, file projectsFile) error {
	var body bytes.Buffer
	encoder := yaml.NewEncoder(&body)
	encoder.SetIndent(2)
	if err := encoder.Encode(file); err != nil {
		return fmt.Errorf("encode projects registry: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("encode projects registry: %w", err)
	}

	path := projectsPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create projects registry parent: %w", err)
	}
	if err := os.WriteFile(path, body.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func projectsPath(root string) string {
	return filepath.Join(root, "knowledge", "projects.yaml")
}

func normalizeProject(project Project) Project {
	project.ID = strings.TrimSpace(project.ID)
	project.Name = strings.TrimSpace(project.Name)
	project.Path = cleanProjectPath(project.Path)
	project.TechDomains = uniqueRegistryValues(project.TechDomains)
	project.BusinessDomains = uniqueRegistryValues(project.BusinessDomains)
	return project
}

func cleanProjectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return ""
	}
	return path
}

func validateProjectRequired(project Project) error {
	var failures []string
	if project.ID == "" {
		failures = append(failures, "project id is required")
	}
	if project.Name == "" {
		failures = append(failures, "project name is required")
	}
	if project.Path == "" {
		failures = append(failures, "project path is required")
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func validateProjectDomains(reg Registry, project Project) error {
	var failures []string
	for _, domain := range project.TechDomains {
		if !containsRegistryValue(reg.TechDomains, domain) {
			failures = append(failures, fmt.Sprintf("unknown tech domain: %s", domain))
		}
	}
	for _, domain := range project.BusinessDomains {
		if !containsRegistryValue(reg.BusinessDomains, domain) {
			failures = append(failures, fmt.Sprintf("unknown business domain: %s", domain))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func uniqueRegistryValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsRegistryValue(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func copyProjects(projects []Project) []Project {
	out := make([]Project, len(projects))
	for i, project := range projects {
		out[i] = project
		out[i].TechDomains = append([]string{}, project.TechDomains...)
		out[i].BusinessDomains = append([]string{}, project.BusinessDomains...)
	}
	return out
}
