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
