package knowledge

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Item struct {
	Path            string   `yaml:"-"`
	ID              string   `yaml:"id"`
	Title           string   `yaml:"title"`
	Type            string   `yaml:"type"`
	TechDomains     []string `yaml:"tech_domains"`
	BusinessDomains []string `yaml:"business_domains"`
	Projects        []string `yaml:"projects"`
	Status          string   `yaml:"status"`
	Priority        string   `yaml:"priority"`
	AppliesTo       Scope    `yaml:"applies_to"`
	UpdatedAt       string   `yaml:"updated_at"`
	Body            string   `yaml:"-"`
}

type Scope struct {
	Languages  []string `yaml:"languages"`
	Frameworks []string `yaml:"frameworks"`
	Files      []string `yaml:"files"`
	Services   []string `yaml:"services"`
	Envs       []string `yaml:"envs"`
}

func ParseItem(path string, data []byte) (Item, error) {
	const opening = "---\n"
	const closing = "\n---\n"

	text := string(data)
	if !strings.HasPrefix(text, opening) {
		return Item{}, fmt.Errorf("%s: missing frontmatter opening delimiter", path)
	}

	end := strings.Index(text[len(opening):], closing)
	if end < 0 {
		return Item{}, fmt.Errorf("%s: missing frontmatter closing delimiter", path)
	}

	frontmatterEnd := len(opening) + end
	frontmatter := text[len(opening):frontmatterEnd]
	body := text[frontmatterEnd+len(closing):]

	var item Item
	if err := yaml.Unmarshal([]byte(frontmatter), &item); err != nil {
		return Item{}, fmt.Errorf("%s: parse frontmatter: %w", path, err)
	}
	item.Path = path
	item.Body = strings.TrimLeft(body, "\r\n")

	return item, nil
}
