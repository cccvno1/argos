package provenance

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	designAbs, err := resolvedPathInsideRoot(root, designPath)
	if err != nil {
		return Record{}, err
	}
	design, err := knowledgewrite.LoadDesign(designAbs)
	if err != nil {
		return Record{}, err
	}
	designHash, err := HashFile(root, designPath)
	if err != nil {
		return Record{}, err
	}

	rootDir, err := safeMkdirAllInsideRoot(root, filepath.Join("knowledge", ".inbox", "provenance"))
	if err != nil {
		return Record{}, err
	}
	for attempt := 0; attempt < 16; attempt++ {
		id, err := newProvenanceID(design.Project, design.DraftOutput.Title)
		if err != nil {
			return Record{}, err
		}
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
		dir := filepath.Join(rootDir, id)
		if err := os.Mkdir(dir, 0o755); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return Record{}, fmt.Errorf("create provenance dir: %w", err)
		}
		if err := writeRecordExclusive(filepath.Join(dir, "provenance.json"), record); err != nil {
			_ = os.Remove(dir)
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return Record{}, err
		}
		return record, nil
	}
	return Record{}, fmt.Errorf("could not allocate unique provenance id")
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
	abs, ok, err := existingPathInsideRoot(root, candidate)
	if err != nil || !ok {
		return Loaded{}, ok, err
	}
	info, err := os.Stat(abs)
	if err != nil {
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
			if len(found) > 1 {
				var paths []string
				for _, match := range found {
					paths = append(paths, match.Path)
				}
				return Loaded{}, false, fmt.Errorf("ambiguous provenance id %q matched %d records: %s", original, len(found), strings.Join(paths, ", "))
			}
			return Loaded{}, false, nil
		}
		abs, ok, err = existingPathInsideRoot(root, filepath.Join(candidate, "provenance.json"))
		if err != nil || !ok {
			return Loaded{}, ok, err
		}
	}
	loaded, err := readRecord(root, abs)
	if err != nil {
		return Loaded{}, false, err
	}
	return loaded, true, nil
}

func readRecord(root string, absPath string) (Loaded, error) {
	rel, err := relPathInsideRoot(root, absPath)
	if err != nil {
		return Loaded{}, err
	}
	safeAbs, err := resolvedPathInsideRoot(root, rel)
	if err != nil {
		return Loaded{}, err
	}
	data, err := os.ReadFile(safeAbs)
	if err != nil {
		return Loaded{}, fmt.Errorf("read provenance: %w", err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Loaded{}, fmt.Errorf("parse provenance JSON: %w", err)
	}
	return Loaded{Record: record, Path: filepath.ToSlash(rel), Dir: filepath.ToSlash(filepath.Dir(rel))}, nil
}

func writeRecord(path string, record Record) error {
	data, err := marshalRecord(record)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeRecordExclusive(path string, record Record) error {
	data, err := marshalRecord(record)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("write provenance: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write provenance: %w", err)
	}
	return nil
}

func marshalRecord(record Record) ([]byte, error) {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal provenance: %w", err)
	}
	return append(data, '\n'), nil
}

func RecordDecision(root string, idOrPath string, input DecisionInput) (Decision, error) {
	if err := validateDecisionInput(input); err != nil {
		return Decision{}, err
	}
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return Decision{}, err
	}
	hashes, err := hashesForStage(root, loaded.Record, input.Stage)
	if err != nil {
		return Decision{}, err
	}
	decisions, err := LoadDecisions(root, idOrPath)
	if err != nil {
		return Decision{}, err
	}
	decision := Decision{
		SchemaVersion: DecisionSchemaVersion,
		DecisionID:    fmt.Sprintf("dec-%03d", len(decisions)+1),
		Stage:         input.Stage,
		Decision:      input.Decision,
		DecidedBy:     strings.TrimSpace(input.DecidedBy),
		DeciderRole:   strings.TrimSpace(input.Role),
		RecordedBy:    strings.TrimSpace(input.RecordedBy),
		Source:        strings.TrimSpace(input.Source),
		Reason:        strings.TrimSpace(input.Reason),
		Hashes:        hashes,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	path, err := decisionsPath(root, loaded.Dir)
	if err != nil {
		return Decision{}, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return Decision{}, fmt.Errorf("open decisions: %w", err)
	}
	defer file.Close()
	data, err := json.Marshal(decision)
	if err != nil {
		return Decision{}, fmt.Errorf("marshal decision: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return Decision{}, fmt.Errorf("write decision: %w", err)
	}
	return decision, nil
}

func LoadDecisions(root string, idOrPath string) ([]Decision, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return nil, err
	}
	path, err := decisionsPath(root, loaded.Dir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Decision{}, nil
		}
		return nil, fmt.Errorf("read decisions: %w", err)
	}
	var decisions []Decision
	for i, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var decision Decision
		if err := json.Unmarshal([]byte(line), &decision); err != nil {
			return nil, fmt.Errorf("parse decisions line %d: %w", i+1, err)
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func RecordCheck(root string, idOrPath string) (knowledgewrite.CheckResponse, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	check, err := knowledgewrite.Check(root, knowledgewrite.CheckRequest{
		DesignPath: loaded.Record.Subject.DesignPath,
		DraftPath:  loaded.Record.Subject.DraftPath,
	})
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	checksDir, err := safeMkdirAllInsideRoot(root, filepath.Join(loaded.Dir, "checks"))
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	next := nextCheckPath(checksDir)
	data, err := json.MarshalIndent(check, "", "  ")
	if err != nil {
		return knowledgewrite.CheckResponse{}, fmt.Errorf("marshal check: %w", err)
	}
	if err := os.WriteFile(filepath.Join(checksDir, next), append(data, '\n'), 0o644); err != nil {
		return knowledgewrite.CheckResponse{}, fmt.Errorf("write check: %w", err)
	}

	loaded.Record.Hashes.DesignSHA256, err = HashFile(root, loaded.Record.Subject.DesignPath)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	loaded.Record.Hashes.DraftTreeSHA256, err = HashTree(root, loaded.Record.Subject.DraftPath)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	checkRel := filepath.Join(loaded.Dir, "checks", next)
	loaded.Record.Hashes.LatestCheckSHA256, err = HashFile(root, checkRel)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	loaded.Record.LatestCheck = &LatestCheck{
		Path:   filepath.ToSlash(filepath.Join("checks", next)),
		Result: check.Result,
	}
	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	return check, nil
}

func Verify(root string, idOrPath string) (VerifyResult, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return VerifyResult{}, err
	}
	var findings []string
	record := loaded.Record
	if record.SchemaVersion != SchemaVersion {
		findings = append(findings, "schema_version must be "+SchemaVersion)
	}
	designHash, err := HashFile(root, record.Subject.DesignPath)
	if err != nil {
		findings = append(findings, err.Error())
	} else if record.Hashes.DesignSHA256 == "" {
		findings = append(findings, "design hash is required")
	} else if designHash != record.Hashes.DesignSHA256 {
		findings = append(findings, "design hash changed")
	}
	draftHash, err := HashTree(root, record.Subject.DraftPath)
	if err != nil {
		findings = append(findings, err.Error())
	} else if record.Hashes.DraftTreeSHA256 == "" {
		findings = append(findings, "draft tree hash is required")
	} else if draftHash != record.Hashes.DraftTreeSHA256 {
		findings = append(findings, "draft tree hash changed")
	}
	if record.LatestCheck == nil {
		findings = append(findings, "latest check is required")
	} else {
		if record.LatestCheck.Result != "pass" {
			findings = append(findings, "latest check must pass")
		}
		if record.LatestCheck.Path == "" {
			findings = append(findings, "latest check path is required")
		} else {
			checkHash, err := HashFile(root, filepath.Join(loaded.Dir, record.LatestCheck.Path))
			if err != nil {
				findings = append(findings, err.Error())
			} else if record.Hashes.LatestCheckSHA256 == "" {
				findings = append(findings, "latest check hash is required")
			} else if checkHash != record.Hashes.LatestCheckSHA256 {
				findings = append(findings, "latest check hash changed")
			}
		}
	}

	decisions, err := LoadDecisions(root, idOrPath)
	if err != nil {
		return VerifyResult{}, err
	}
	requireApprovedDecision(&findings, decisions, StageDesign, record.Hashes)
	requireApprovedDecision(&findings, decisions, StageDraftWrite, record.Hashes)
	requireApprovedDecision(&findings, decisions, StagePublish, record.Hashes)

	result := "pass"
	if len(findings) > 0 {
		result = "fail"
	}
	return VerifyResult{Result: result, ID: record.ProvenanceID, Path: loaded.Dir, Findings: findings}, nil
}

func validateDecisionInput(input DecisionInput) error {
	if !validStage(input.Stage) {
		return fmt.Errorf("stage must be design, draft_write, or publish")
	}
	if !validDecision(input.Decision) {
		return fmt.Errorf("decision must be approved, changes_requested, or rejected")
	}
	if strings.TrimSpace(input.DecidedBy) == "" {
		return fmt.Errorf("decided_by is required")
	}
	if strings.TrimSpace(input.Role) == "" {
		return fmt.Errorf("role is required")
	}
	if strings.TrimSpace(input.Source) == "" {
		return fmt.Errorf("source is required")
	}
	if strings.TrimSpace(input.RecordedBy) == "" {
		return fmt.Errorf("recorded_by is required")
	}
	if strings.TrimSpace(input.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

func hashesForStage(root string, record Record, stage string) (Hashes, error) {
	designHash, err := HashFile(root, record.Subject.DesignPath)
	if err != nil {
		return Hashes{}, err
	}
	hashes := Hashes{DesignSHA256: designHash}
	if stage == StagePublish {
		draftHash, err := HashTree(root, record.Subject.DraftPath)
		if err != nil {
			return Hashes{}, err
		}
		hashes.DraftTreeSHA256 = draftHash
		hashes.LatestCheckSHA256 = record.Hashes.LatestCheckSHA256
	}
	return hashes, nil
}

func validStage(stage string) bool {
	return stage == StageDesign || stage == StageDraftWrite || stage == StagePublish
}

func validDecision(decision string) bool {
	return decision == DecisionApproved || decision == DecisionChangesRequested || decision == DecisionRejected
}

func decisionsPath(root string, provenanceDir string) (string, error) {
	clean, err := cleanRelPath(filepath.Join(provenanceDir, "decisions.jsonl"))
	if err != nil {
		return "", err
	}
	raw := filepath.Join(root, clean)
	info, err := os.Lstat(raw)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s: path must not be a symlink", filepath.ToSlash(clean))
		}
		return resolvedPathInsideRoot(root, clean)
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
	}
	parent, err := resolvedPathInsideRoot(root, filepath.Dir(clean))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, "decisions.jsonl"), nil
}

func nextCheckPath(checksDir string) string {
	entries, err := os.ReadDir(checksDir)
	if err != nil {
		return "check-001.json"
	}
	return fmt.Sprintf("check-%03d.json", len(entries)+1)
}

func requireApprovedDecision(findings *[]string, decisions []Decision, stage string, hashes Hashes) {
	for i := len(decisions) - 1; i >= 0; i-- {
		decision := decisions[i]
		if decision.Stage != stage {
			continue
		}
		if decision.Decision != DecisionApproved {
			*findings = append(*findings, stage+" decision must be approved")
			return
		}
		if stage == StagePublish {
			if decision.Hashes.DesignSHA256 != hashes.DesignSHA256 ||
				decision.Hashes.DraftTreeSHA256 != hashes.DraftTreeSHA256 ||
				decision.Hashes.LatestCheckSHA256 != hashes.LatestCheckSHA256 {
				*findings = append(*findings, "publish decision hashes do not match current record")
			}
		}
		return
	}
	*findings = append(*findings, stage+" approval decision is required")
}

func newProvenanceID(project string, title string) (string, error) {
	suffix, err := randomHex(4)
	if err != nil {
		return "", err
	}
	return "prov-" + time.Now().UTC().Format("20060102") + "-" + slug(project+"-"+title) + "-" + suffix, nil
}

var randomBytes = rand.Read

func randomHex(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := randomBytes(buf); err != nil {
		return "", fmt.Errorf("generate random provenance id: %w", err)
	}
	return hex.EncodeToString(buf), nil
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

func safeMkdirAllInsideRoot(root string, relDir string) (string, error) {
	clean, err := cleanRelPath(relDir)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root symlinks: %w", err)
	}
	current := rootResolved
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("%s: path must not contain symlinks", filepath.ToSlash(clean))
			}
			if !info.IsDir() {
				return "", fmt.Errorf("%s exists but is not a directory", filepath.ToSlash(clean))
			}
			continue
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
		}
		if err := os.Mkdir(current, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", filepath.ToSlash(clean), err)
		}
	}
	return current, nil
}

func existingPathInsideRoot(root string, relPath string) (string, bool, error) {
	clean, err := cleanRelPath(relPath)
	if err != nil {
		return "", false, err
	}
	raw := filepath.Join(root, clean)
	if _, err := os.Lstat(raw); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
	}
	resolved, err := resolvedPathInsideRoot(root, clean)
	if err != nil {
		return "", false, err
	}
	return resolved, true, nil
}

func relPathInsideRoot(root string, absPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root symlinks: %w", err)
	}
	targetResolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve provenance path symlinks: %w", err)
	}
	rel, err := filepath.Rel(rootResolved, targetResolved)
	if err != nil {
		return "", fmt.Errorf("resolve provenance path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%s: resolved path must stay inside workspace", filepath.ToSlash(absPath))
	}
	return filepath.ToSlash(rel), nil
}
