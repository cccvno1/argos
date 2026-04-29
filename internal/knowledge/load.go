package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadItems(root string) ([]Item, error) {
	return LoadOfficial(root)
}

func LoadOfficial(root string) ([]Item, error) {
	return loadFromRoots(root, []string{
		filepath.Join("knowledge", "items"),
		filepath.Join("knowledge", "packages"),
	})
}

func LoadInbox(root string) ([]Item, error) {
	return loadFromRoots(root, []string{
		filepath.Join("knowledge", ".inbox", "items"),
		filepath.Join("knowledge", ".inbox", "packages"),
	})
}

func LoadPath(root string, relPath string) ([]Item, error) {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return nil, err
	}
	abs := filepath.Join(root, clean)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", clean, err)
	}
	if info.IsDir() {
		entrypoint := filepath.Join(abs, "KNOWLEDGE.md")
		item, err := readItem(root, entrypoint)
		if err != nil {
			return nil, err
		}
		return []Item{item}, nil
	}
	if filepath.Ext(abs) != ".md" {
		return nil, fmt.Errorf("%s: expected markdown file or package directory", clean)
	}
	item, err := readItem(root, abs)
	if err != nil {
		return nil, err
	}
	return []Item{item}, nil
}

func loadFromRoots(root string, relRoots []string) ([]Item, error) {
	var items []Item
	for _, relRoot := range relRoots {
		absRoot := filepath.Join(root, relRoot)
		info, err := os.Stat(absRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", relRoot, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s exists but is not a directory", relRoot)
		}
		err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !isLoadableKnowledgeFile(path) {
				return nil
			}
			item, err := readItem(root, path)
			if err != nil {
				return err
			}
			items = append(items, item)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func isLoadableKnowledgeFile(path string) bool {
	return filepath.Base(path) == "KNOWLEDGE.md" ||
		filepath.Ext(path) == ".md" && !strings.Contains(path, string(filepath.Separator)+"packages"+string(filepath.Separator))
}

func readItem(root string, path string) (Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Item{}, fmt.Errorf("read %s: %w", path, err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Item{}, fmt.Errorf("resolve relative path for %s: %w", path, err)
	}
	return ParseItem(rel, data)
}

func cleanRelativePath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("%s: path must be relative", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: path must stay inside workspace", relPath)
	}
	return clean, nil
}
