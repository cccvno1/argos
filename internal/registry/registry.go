package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	TechDomains     []string  `yaml:"tech_domains"`
	BusinessDomains []string  `yaml:"business_domains"`
	Projects        []Project `yaml:"projects"`
	Types           []string  `yaml:"types"`
}

type Project struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Path            string   `yaml:"path"`
	TechDomains     []string `yaml:"tech_domains"`
	BusinessDomains []string `yaml:"business_domains"`
}

func Load(root string) (Registry, error) {
	var reg Registry

	var domains Registry
	if err := readYAML(filepath.Join(root, "knowledge", "domains.yaml"), &domains); err != nil {
		return Registry{}, err
	}
	reg.TechDomains = domains.TechDomains
	reg.BusinessDomains = domains.BusinessDomains

	var projects struct {
		Projects []Project `yaml:"projects"`
	}
	if err := readYAML(filepath.Join(root, "knowledge", "projects.yaml"), &projects); err != nil {
		return Registry{}, err
	}
	reg.Projects = projects.Projects

	var types struct {
		Types []string `yaml:"types"`
	}
	if err := readYAML(filepath.Join(root, "knowledge", "types.yaml"), &types); err != nil {
		return Registry{}, err
	}
	reg.Types = types.Types

	return reg, nil
}

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
