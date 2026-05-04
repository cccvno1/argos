package registry

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

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
	if err := validateProjectPath(project.Path); err != nil {
		return err
	}

	unlock, err := lockProjectsRegistry(root)
	if err != nil {
		return err
	}
	defer unlock()

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
	return writeFileAtomically(path, body.Bytes())
}

func projectsPath(root string) string {
	return filepath.Join(root, "knowledge", "projects.yaml")
}

func projectsLockPath(root string) string {
	return projectsPath(root) + ".lock"
}

func lockProjectsRegistry(root string) (func(), error) {
	lockPath := projectsLockPath(root)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create projects registry lock parent: %w", err)
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("projects registry is locked: %s", lockPath)
		}
		return nil, fmt.Errorf("lock projects registry: %w", err)
	}
	return func() {
		_ = lock.Close()
		_ = os.Remove(lockPath)
	}, nil
}

func normalizeProject(project Project) Project {
	project.ID = strings.TrimSpace(project.ID)
	project.Name = strings.TrimSpace(project.Name)
	project.Path = cleanProjectPath(project.Path)
	project.TechDomains = uniqueRegistryValues(project.TechDomains)
	project.BusinessDomains = uniqueRegistryValues(project.BusinessDomains)
	return project
}

func cleanProjectPath(projectPath string) string {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return ""
	}
	projectPath = strings.ReplaceAll(projectPath, `\`, "/")
	projectPath = path.Clean(projectPath)
	return projectPath
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

func validateProjectPath(projectPath string) error {
	if path.IsAbs(projectPath) || hasWindowsVolumeName(projectPath) {
		return fmt.Errorf("project path must be relative")
	}
	if projectPath == ".." || strings.HasPrefix(projectPath, "../") {
		return fmt.Errorf("project path must stay inside workspace")
	}
	return nil
}

func hasWindowsVolumeName(projectPath string) bool {
	if len(projectPath) < 2 || projectPath[1] != ':' {
		return false
	}
	drive := projectPath[0]
	return ('A' <= drive && drive <= 'Z') || ('a' <= drive && drive <= 'z')
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

func writeFileAtomically(target string, data []byte) error {
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".projects-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp projects registry: %w", err)
	}

	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write %s: %w", tempName, err)
	}
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod %s: %w", tempName, err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync %s: %w", tempName, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tempName, err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tempName, target, err)
	}
	removeTemp = false
	if err := syncParentDirectory(target); err != nil {
		return err
	}
	return nil
}

func syncParentDirectory(target string) error {
	dirPath := filepath.Dir(target)
	dir, err := os.Open(dirPath)
	if err != nil {
		return fmt.Errorf("open projects registry parent: %w", err)
	}
	defer dir.Close()

	if err := dir.Sync(); err != nil {
		// Some platforms/filesystems do not support fsync on directories and
		// report EINVAL. Linux supports this path and should sync successfully.
		if errors.Is(err, syscall.EINVAL) {
			return nil
		}
		return fmt.Errorf("sync projects registry parent: %w", err)
	}
	return nil
}
