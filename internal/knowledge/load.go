package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadItems(root string) ([]Item, error) {
	itemsRoot := filepath.Join(root, "knowledge", "items")
	var items []Item

	err := filepath.WalkDir(itemsRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		item, err := ParseItem(rel, data)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}
