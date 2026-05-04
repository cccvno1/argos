package provenance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func List(root string, filter ListFilter) (ListResponse, error) {
	state := strings.TrimSpace(filter.State)
	if state == "" {
		state = "all"
	}
	if state != "all" && state != StateDraft && state != StatePublished {
		return ListResponse{}, fmt.Errorf("state must be draft, published, or all")
	}

	var loaded []Loaded
	for _, relRoot := range []string{
		filepath.Join("knowledge", ".inbox", "provenance"),
		filepath.Join("knowledge", "provenance"),
	} {
		records, err := listRecordsUnder(root, relRoot)
		if err != nil {
			return ListResponse{}, err
		}
		loaded = append(loaded, records...)
	}
	sort.Slice(loaded, func(i int, j int) bool {
		return loaded[i].Path < loaded[j].Path
	})

	project := strings.TrimSpace(filter.Project)
	knowledgeID := strings.TrimSpace(filter.KnowledgeID)
	var records []ListRecord
	for _, item := range loaded {
		record := item.Record
		if state != "all" && record.State != state {
			continue
		}
		if project != "" && record.Subject.Project != project {
			continue
		}
		if knowledgeID != "" && record.Subject.KnowledgeID != knowledgeID {
			continue
		}
		records = append(records, listRecordFromLoaded(item))
	}
	return ListResponse{Records: records}, nil
}

func listRecordsUnder(root string, relRoot string) ([]Loaded, error) {
	absRoot := filepath.Join(root, relRoot)
	info, err := os.Lstat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", filepath.ToSlash(relRoot), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: path must not contain symlinks", filepath.ToSlash(relRoot))
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s exists but is not a directory", filepath.ToSlash(relRoot))
	}

	var records []Loaded
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			rel, _, err := lexicalPathInsideRoot(root, path)
			if err != nil {
				return err
			}
			return fmt.Errorf("%s: path must not contain symlinks", filepath.ToSlash(rel))
		}
		if entry.IsDir() || entry.Name() != "provenance.json" {
			return nil
		}
		loaded, err := readRecord(root, path)
		if err != nil {
			rel, _, relErr := lexicalPathInsideRoot(root, path)
			if relErr != nil {
				return err
			}
			return fmt.Errorf("%s: %w", filepath.ToSlash(rel), err)
		}
		records = append(records, loaded)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func listRecordFromLoaded(loaded Loaded) ListRecord {
	record := loaded.Record
	latest := ""
	if record.LatestCheck != nil {
		latest = record.LatestCheck.Result
	}
	return ListRecord{
		ProvenanceID:      record.ProvenanceID,
		State:             record.State,
		Path:              loaded.Dir,
		Project:           record.Subject.Project,
		KnowledgeID:       record.Subject.KnowledgeID,
		Kind:              record.Subject.Kind,
		DesignPath:        record.Subject.DesignPath,
		DraftPath:         record.Subject.DraftPath,
		OfficialPath:      record.Subject.OfficialPath,
		LatestCheckResult: latest,
		CreatedAt:         record.CreatedAt,
		PublishedAt:       record.PublishedAt,
	}
}
