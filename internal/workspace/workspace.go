package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Init(root string) error {
	dirs := []string{
		"knowledge/.inbox/items",
		"knowledge/.inbox/packages",
		"knowledge/.inbox/designs",
		"knowledge/items",
		"knowledge/packages",
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
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("%s exists but is not a regular file", rel)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
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
  - package
`
