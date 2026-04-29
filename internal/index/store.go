package index

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/knowledge"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Rebuild(dbPath string, items []knowledge.Item) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing index: %w", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.createSchema(); err != nil {
		return err
	}
	for _, item := range items {
		if err := store.InsertItem(item); err != nil {
			return err
		}
	}
	return nil
}

func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open index database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE knowledge_items (
	id TEXT PRIMARY KEY,
	path TEXT NOT NULL,
	title TEXT NOT NULL,
	type TEXT NOT NULL,
	tech_domains TEXT NOT NULL,
	business_domains TEXT NOT NULL,
	projects TEXT NOT NULL,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	scope TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	summary TEXT NOT NULL,
	body TEXT NOT NULL
)`)
	if err != nil {
		return fmt.Errorf("create index schema: %w", err)
	}
	return nil
}

func (s *Store) InsertItem(item knowledge.Item) error {
	techDomains, err := marshalJSON(item.TechDomains)
	if err != nil {
		return fmt.Errorf("%s: serialize tech domains: %w", item.ID, err)
	}
	businessDomains, err := marshalJSON(item.BusinessDomains)
	if err != nil {
		return fmt.Errorf("%s: serialize business domains: %w", item.ID, err)
	}
	projects, err := marshalJSON(item.Projects)
	if err != nil {
		return fmt.Errorf("%s: serialize projects: %w", item.ID, err)
	}
	scope, err := marshalJSON(item.AppliesTo)
	if err != nil {
		return fmt.Errorf("%s: serialize scope: %w", item.ID, err)
	}

	_, err = s.db.Exec(`
INSERT INTO knowledge_items (
	id, path, title, type, tech_domains, business_domains, projects,
	status, priority, scope, updated_at, summary, body
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.Path,
		item.Title,
		item.Type,
		techDomains,
		businessDomains,
		projects,
		item.Status,
		item.Priority,
		scope,
		item.UpdatedAt,
		summary(item.Body),
		item.Body,
	)
	if err != nil {
		return fmt.Errorf("%s: insert knowledge item: %w", item.ID, err)
	}
	return nil
}

func (s *Store) GetItem(id string) (knowledge.Item, error) {
	var item knowledge.Item
	var techDomains string
	var businessDomains string
	var projects string
	var scope string
	var unusedSummary string

	err := s.db.QueryRow(`
SELECT id, path, title, type, tech_domains, business_domains, projects,
	status, priority, scope, updated_at, summary, body
FROM knowledge_items
WHERE id = ?`, id).Scan(
		&item.ID,
		&item.Path,
		&item.Title,
		&item.Type,
		&techDomains,
		&businessDomains,
		&projects,
		&item.Status,
		&item.Priority,
		&scope,
		&item.UpdatedAt,
		&unusedSummary,
		&item.Body,
	)
	if err != nil {
		return knowledge.Item{}, fmt.Errorf("get knowledge item %s: %w", id, err)
	}
	if err := unmarshalJSON(techDomains, &item.TechDomains); err != nil {
		return knowledge.Item{}, fmt.Errorf("%s: deserialize tech domains: %w", id, err)
	}
	if err := unmarshalJSON(businessDomains, &item.BusinessDomains); err != nil {
		return knowledge.Item{}, fmt.Errorf("%s: deserialize business domains: %w", id, err)
	}
	if err := unmarshalJSON(projects, &item.Projects); err != nil {
		return knowledge.Item{}, fmt.Errorf("%s: deserialize projects: %w", id, err)
	}
	if err := unmarshalJSON(scope, &item.AppliesTo); err != nil {
		return knowledge.Item{}, fmt.Errorf("%s: deserialize scope: %w", id, err)
	}

	return item, nil
}

func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalJSON(data string, value any) error {
	return json.Unmarshal([]byte(data), value)
}

func summary(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
