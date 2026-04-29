package workspace

import (
	"os"
	"path/filepath"
)

func Init(root string) error {
	dirs := []string{
		"knowledge/.inbox",
		"knowledge/items",
		"argos/generated",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"knowledge/domains.yaml":  defaultDomainsYAML,
		"knowledge/projects.yaml": defaultProjectsYAML,
		"knowledge/types.yaml":    defaultTypesYAML,
	}
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}

	return nil
}

const defaultDomainsYAML = `tech_domains:
  - backend
  - frontend
  - database
  - devops
  - testing
  - security
  - ai-engineering
business_domains:
  - account
  - order
  - payment
  - risk
  - content
  - analytics
`

const defaultProjectsYAML = `projects: []
`

const defaultTypesYAML = `types:
  - rule
  - guide
  - runbook
  - decision
  - lesson
  - reference
`
