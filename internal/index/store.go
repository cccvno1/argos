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

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func Rebuild(dbPath string, items []knowledge.Item) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}

	tempPath := dbPath + ".tmp"
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale temporary index: %w", err)
	}

	store, err := Open(tempPath)
	if err != nil {
		return err
	}

	tx, err := store.db.Begin()
	if err != nil {
		cleanupTemp(store, tempPath)
		return fmt.Errorf("begin index rebuild transaction: %w", err)
	}

	if err := store.createSchema(tx); err != nil {
		tx.Rollback()
		cleanupTemp(store, tempPath)
		return err
	}
	for _, item := range items {
		if err := store.insertItem(tx, item); err != nil {
			tx.Rollback()
			cleanupTemp(store, tempPath)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		cleanupTemp(store, tempPath)
		return fmt.Errorf("commit index rebuild transaction: %w", err)
	}
	if err := store.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("close temporary index: %w", err)
	}
	if err := replaceIndex(tempPath, dbPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("replace index: %w", err)
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

func (s *Store) CheckSchema() error {
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("ping index database: %w", err)
	}
	if _, err := s.db.Exec(`SELECT 1 FROM knowledge_items LIMIT 0`); err != nil {
		return fmt.Errorf("check index schema: %w", err)
	}
	return nil
}

func (s *Store) createSchema(db execer) error {
	_, err := db.Exec(`
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
	return s.insertItem(s.db, item)
}

func (s *Store) insertItem(db execer, item knowledge.Item) error {
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

	_, err = db.Exec(`
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

func cleanupTemp(store *Store, tempPath string) {
	store.Close()
	os.Remove(tempPath)
}

func replaceIndex(tempPath string, dbPath string) error {
	if err := os.Rename(tempPath, dbPath); err == nil {
		return nil
	} else if _, statErr := os.Stat(dbPath); statErr != nil {
		return err
	}

	backupPath := dbPath + ".bak"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale backup index: %w", err)
	}
	if err := os.Rename(dbPath, backupPath); err != nil {
		return fmt.Errorf("backup existing index: %w", err)
	}
	if err := os.Rename(tempPath, dbPath); err != nil {
		restoreErr := os.Rename(backupPath, dbPath)
		if restoreErr != nil {
			return fmt.Errorf("install rebuilt index: %w; restore existing index: %v", err, restoreErr)
		}
		return fmt.Errorf("install rebuilt index: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("remove backup index: %w", err)
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

func (s *Store) ListItems() ([]knowledge.Item, error) {
	rows, err := s.db.Query(`
SELECT id, path, title, type, tech_domains, business_domains, projects,
	status, priority, scope, updated_at, summary, body
FROM knowledge_items
ORDER BY CASE priority
	WHEN 'must' THEN 0
	WHEN 'should' THEN 1
	WHEN 'may' THEN 2
	ELSE 3
END, id`)
	if err != nil {
		return nil, fmt.Errorf("list knowledge items: %w", err)
	}
	defer rows.Close()

	var items []knowledge.Item
	for rows.Next() {
		var item knowledge.Item
		var techDomains string
		var businessDomains string
		var projects string
		var scope string
		var unusedSummary string

		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan knowledge item: %w", err)
		}
		if err := unmarshalJSON(techDomains, &item.TechDomains); err != nil {
			return nil, fmt.Errorf("%s: deserialize tech domains: %w", item.ID, err)
		}
		if err := unmarshalJSON(businessDomains, &item.BusinessDomains); err != nil {
			return nil, fmt.Errorf("%s: deserialize business domains: %w", item.ID, err)
		}
		if err := unmarshalJSON(projects, &item.Projects); err != nil {
			return nil, fmt.Errorf("%s: deserialize projects: %w", item.ID, err)
		}
		if err := unmarshalJSON(scope, &item.AppliesTo); err != nil {
			return nil, fmt.Errorf("%s: deserialize scope: %w", item.ID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list knowledge items: %w", err)
	}

	return items, nil
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
