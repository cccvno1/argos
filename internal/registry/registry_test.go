package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMergesRegistryFiles(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains:
  - backend
  - security
business_domains:
  - account
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", `types:
  - rule
  - guide
`)

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.TechDomains) != 2 || reg.TechDomains[1] != "security" {
		t.Fatalf("unexpected tech domains: %#v", reg.TechDomains)
	}
	if len(reg.BusinessDomains) != 1 || reg.BusinessDomains[0] != "account" {
		t.Fatalf("unexpected business domains: %#v", reg.BusinessDomains)
	}
	if len(reg.Projects) != 1 || reg.Projects[0].ID != "mall-api" || reg.Projects[0].Path != "services/mall-api" {
		t.Fatalf("unexpected projects: %#v", reg.Projects)
	}
	if len(reg.Types) != 2 || reg.Types[0] != "rule" {
		t.Fatalf("unexpected types: %#v", reg.Types)
	}
}

func TestLoadMissingFileErrorIncludesRegistryPath(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	_, err := Load(root)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "knowledge", "domains.yaml")) {
		t.Fatalf("expected registry path in error, got: %v", err)
	}
}

func TestLoadMalformedYAMLErrorIncludesRegistryPath(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend\n")
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	_, err := Load(root)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "knowledge", "domains.yaml")) {
		t.Fatalf("expected registry path in error, got: %v", err)
	}
}

func TestLoadRejectsUnknownRegistryField(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
unexpected: nope
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	_, err := Load(root)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "knowledge", "domains.yaml")) {
		t.Fatalf("expected registry path in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "field unexpected not found") {
		t.Fatalf("expected unknown field error, got: %v", err)
	}
}

func TestAddProjectWritesStructuredProjectsYAML(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, database]
business_domains: [account, catalog]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"backend", "database", "backend", " "},
		BusinessDomains: []string{"account", "catalog", "account"},
	})
	if err != nil {
		t.Fatalf("AddProject returned error: %v", err)
	}

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.Projects) != 1 {
		t.Fatalf("expected one project, got %#v", reg.Projects)
	}
	got := reg.Projects[0]
	if got.ID != "mall-api" || got.Name != "Mall API" || got.Path != "services/mall-api" {
		t.Fatalf("unexpected project identity: %#v", got)
	}
	if strings.Join(got.TechDomains, ",") != "backend,database" {
		t.Fatalf("unexpected tech domains: %#v", got.TechDomains)
	}
	if strings.Join(got.BusinessDomains, ",") != "account,catalog" {
		t.Fatalf("unexpected business domains: %#v", got.BusinessDomains)
	}
}

func TestAddProjectNormalizesBackslashPath(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            `services\mall-api`,
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
	})
	if err != nil {
		t.Fatalf("AddProject returned error: %v", err)
	}

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.Projects) != 1 {
		t.Fatalf("expected one project, got %#v", reg.Projects)
	}
	if reg.Projects[0].Path != "services/mall-api" {
		t.Fatalf("unexpected project path: %q", reg.Projects[0].Path)
	}
}

func TestAddProjectPreservesExistingProjects(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend, frontend]
business_domains: [account, catalog]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-web",
		Name:            "Mall Web",
		Path:            "apps/mall-web",
		TechDomains:     []string{"frontend"},
		BusinessDomains: []string{"catalog"},
	})
	if err != nil {
		t.Fatalf("AddProject returned error: %v", err)
	}

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.Projects) != 2 {
		t.Fatalf("expected two projects, got %#v", reg.Projects)
	}
	if reg.Projects[0].ID != "mall-api" || reg.Projects[0].Name != "Mall API" || reg.Projects[0].Path != "services/mall-api" {
		t.Fatalf("existing project was not preserved: %#v", reg.Projects[0])
	}
	if reg.Projects[1].ID != "mall-web" || reg.Projects[1].Name != "Mall Web" || reg.Projects[1].Path != "apps/mall-web" {
		t.Fatalf("new project was not appended: %#v", reg.Projects[1])
	}
}

func TestAddProjectRejectsLockedProjectsRegistry(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")
	lockPath := filepath.Join(root, "knowledge", "projects.yaml.lock")
	if err := os.WriteFile(lockPath, []byte("locked"), 0o644); err != nil {
		t.Fatalf("write %s: %v", lockPath, err)
	}

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
	})
	if err == nil {
		t.Fatal("expected locked projects registry error")
	}
	if !strings.Contains(err.Error(), "projects registry is locked") {
		t.Fatalf("expected locked registry error, got: %v", err)
	}

	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(reg.Projects) != 0 {
		t.Fatalf("expected no appended projects, got %#v", reg.Projects)
	}
}

func TestAddProjectRemovesProjectsRegistryLockAfterSuccess(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
	})
	if err != nil {
		t.Fatalf("AddProject returned error: %v", err)
	}

	lockPath := filepath.Join(root, "knowledge", "projects.yaml.lock")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected projects registry lock to be removed, got stat error: %v", err)
	}
}

func TestAddProjectRejectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{ID: "mall-api", Name: "Mall API 2", Path: "services/mall-api-2"})
	if err == nil {
		t.Fatal("expected duplicate project error")
	}
	if !strings.Contains(err.Error(), "project already exists: mall-api") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddProjectRejectsUnknownDomains(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", "projects: []\n")
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	err := AddProject(root, Project{
		ID:              "mall-api",
		Name:            "Mall API",
		Path:            "services/mall-api",
		TechDomains:     []string{"mobile"},
		BusinessDomains: []string{"catalog"},
	})
	if err == nil {
		t.Fatal("expected unknown domain error")
	}
	for _, want := range []string{"unknown tech domain: mobile", "unknown business domain: catalog"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestListProjectsReturnsRegisteredProjects(t *testing.T) {
	root := t.TempDir()
	writeRegistryFile(t, root, "knowledge/domains.yaml", `tech_domains: [backend]
business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/projects.yaml", `projects:
  - id: mall-api
    name: Mall API
    path: services/mall-api
    tech_domains: [backend]
    business_domains: [account]
`)
	writeRegistryFile(t, root, "knowledge/types.yaml", "types: [rule]\n")

	projects, err := ListProjects(root)
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != "mall-api" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}

func writeRegistryFile(t *testing.T, root, rel, body string) {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
