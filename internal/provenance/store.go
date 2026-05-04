package provenance

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"argos/internal/knowledgewrite"
)

type StartRequest struct {
	DesignPath string
	DraftPath  string
	CreatedBy  string
}

type Loaded struct {
	Record Record
	Path   string
	Dir    string
}

func Start(root string, req StartRequest) (Record, error) {
	designPath, err := cleanRelPath(req.DesignPath)
	if err != nil {
		return Record{}, err
	}
	draftPath, err := cleanRelPath(req.DraftPath)
	if err != nil {
		return Record{}, err
	}
	design, err := knowledgewrite.LoadDesign(filepath.Join(root, designPath))
	if err != nil {
		return Record{}, err
	}
	designHash, err := HashFile(root, designPath)
	if err != nil {
		return Record{}, err
	}

	id := newProvenanceID(design.Project, design.DraftOutput.Title)
	record := Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID:  id,
		State:         StateDraft,
		Subject: Subject{
			Kind:         strings.TrimSpace(design.DraftOutput.Kind),
			KnowledgeID:  strings.TrimSpace(design.DraftOutput.ID),
			Project:      strings.TrimSpace(design.Project),
			DesignPath:   filepath.ToSlash(designPath),
			DraftPath:    filepath.ToSlash(draftPath),
			OfficialPath: officialPathForDraft(draftPath),
		},
		Hashes: Hashes{
			DesignSHA256: designHash,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		CreatedBy: strings.TrimSpace(req.CreatedBy),
	}
	if record.CreatedBy == "" {
		record.CreatedBy = "unknown"
	}

	dir := filepath.Join(root, "knowledge", ".inbox", "provenance", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Record{}, fmt.Errorf("create provenance dir: %w", err)
	}
	if err := writeRecord(filepath.Join(dir, "provenance.json"), record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func Load(root string, idOrPath string) (Loaded, error) {
	clean, err := cleanRelPath(idOrPath)
	if err != nil {
		return Loaded{}, err
	}
	candidates := []string{clean}
	if !strings.Contains(filepath.ToSlash(clean), "/") {
		candidates = []string{
			filepath.Join("knowledge", ".inbox", "provenance", clean),
			filepath.Join("knowledge", "provenance"),
		}
	}
	for _, candidate := range candidates {
		loaded, ok, err := tryLoad(root, candidate, clean)
		if err != nil || ok {
			return loaded, err
		}
	}
	return Loaded{}, fmt.Errorf("%s: provenance record not found", idOrPath)
}

func tryLoad(root string, candidate string, original string) (Loaded, bool, error) {
	abs := filepath.Join(root, candidate)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return Loaded{}, false, nil
		}
		return Loaded{}, false, fmt.Errorf("stat %s: %w", filepath.ToSlash(candidate), err)
	}
	if info.IsDir() {
		if filepath.Base(candidate) == "provenance" && !strings.Contains(filepath.ToSlash(original), "/") {
			var found []Loaded
			err := filepath.WalkDir(abs, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || entry.Name() != "provenance.json" {
					return nil
				}
				loaded, err := readRecord(root, path)
				if err != nil {
					return err
				}
				if loaded.Record.ProvenanceID == original {
					found = append(found, loaded)
				}
				return nil
			})
			if err != nil {
				return Loaded{}, false, err
			}
			if len(found) == 1 {
				return found[0], true, nil
			}
			return Loaded{}, false, nil
		}
		candidate = filepath.Join(candidate, "provenance.json")
		abs = filepath.Join(root, candidate)
	}
	loaded, err := readRecord(root, abs)
	if err != nil {
		return Loaded{}, false, err
	}
	return loaded, true, nil
}

func readRecord(root string, absPath string) (Loaded, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("read provenance: %w", err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Loaded{}, fmt.Errorf("parse provenance JSON: %w", err)
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve provenance path: %w", err)
	}
	return Loaded{Record: record, Path: filepath.ToSlash(rel), Dir: filepath.ToSlash(filepath.Dir(rel))}, nil
}

func writeRecord(path string, record Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func newProvenanceID(project string, title string) string {
	return "prov-" + time.Now().UTC().Format("20060102") + "-" + slug(project+"-"+title) + "-" + randomHex(4)
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(buf)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out []rune
	lastDash := false
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out = append(out, r)
			lastDash = false
			continue
		}
		if !lastDash {
			out = append(out, '-')
			lastDash = true
		}
	}
	s := strings.Trim(string(out), "-")
	if s == "" {
		return "knowledge"
	}
	if len(s) > 48 {
		return strings.Trim(s[:48], "-")
	}
	return s
}

func officialPathForDraft(draftPath string) string {
	slash := filepath.ToSlash(draftPath)
	switch {
	case strings.HasPrefix(slash, "knowledge/.inbox/packages/"):
		return "knowledge/packages/" + strings.TrimPrefix(slash, "knowledge/.inbox/packages/")
	case strings.HasPrefix(slash, "knowledge/.inbox/items/"):
		return "knowledge/items/" + strings.TrimPrefix(slash, "knowledge/.inbox/items/")
	default:
		return slash
	}
}
