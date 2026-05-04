package provenance

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

type PublishMove struct {
	Record  Record
	FromDir string
	ToDir   string
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
	next, err := writeCheckFile(checksDir, check)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
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

func PreparePublishMove(root string, idOrPath string, publishedBy string) (PublishMove, func() error, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return PublishMove{}, nil, err
	}
	verify, err := Verify(root, idOrPath)
	if err != nil {
		return PublishMove{}, nil, err
	}
	if verify.Result != "pass" && shouldVerifyPublishedDraft(root, loaded.Record) {
		publishedDraftHash, hashErr := hashPublishedDraftAsDraft(root, loaded.Record)
		verify, err = verifyLoadedWithDraftHash(root, idOrPath, loaded, publishedDraftHash, hashErr)
		if err != nil {
			return PublishMove{}, nil, err
		}
	}
	if verify.Result != "pass" {
		return PublishMove{}, nil, fmt.Errorf("provenance verify failed: %s", strings.Join(verify.Findings, "; "))
	}

	original := loaded.Record
	fromDir := loaded.Dir
	fromAbs, ok, err := existingPathInsideRoot(root, fromDir)
	if err != nil {
		return PublishMove{}, nil, err
	}
	if !ok {
		return PublishMove{}, nil, fmt.Errorf("%s: provenance record not found", fromDir)
	}

	knowledgeID := safeKnowledgeID(loaded.Record.Subject.KnowledgeID)
	toRel := filepath.Join("knowledge", "provenance", knowledgeID, loaded.Record.ProvenanceID)
	toClean, err := cleanRelPath(toRel)
	if err != nil {
		return PublishMove{}, nil, err
	}
	if filepath.ToSlash(toClean) != "knowledge/provenance/"+knowledgeID+"/"+loaded.Record.ProvenanceID {
		return PublishMove{}, nil, fmt.Errorf("%s: invalid published provenance path", filepath.ToSlash(toRel))
	}
	if _, ok, err := existingPathInsideRoot(root, toClean); err != nil {
		return PublishMove{}, nil, err
	} else if ok {
		return PublishMove{}, nil, fmt.Errorf("published provenance target already exists: %s", filepath.ToSlash(toClean))
	}

	loaded.Record.State = StatePublished
	loaded.Record.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	loaded.Record.PublishedBy = strings.TrimSpace(publishedBy)
	if loaded.Record.PublishedBy == "" {
		loaded.Record.PublishedBy = "unknown"
	}
	loaded.Record.PublishedFrom = loaded.Record.Subject.DraftPath
	loaded.Record.PublishedTo = loaded.Record.Subject.OfficialPath

	recordPath, err := resolvedPathInsideRoot(root, loaded.Path)
	if err != nil {
		return PublishMove{}, nil, err
	}
	if err := writeRecord(recordPath, loaded.Record); err != nil {
		return PublishMove{}, nil, err
	}

	toParent, err := safeMkdirAllInsideRoot(root, filepath.Dir(toClean))
	if err != nil {
		_ = writeRecord(recordPath, original)
		return PublishMove{}, nil, err
	}
	toAbs := filepath.Join(toParent, filepath.Base(toClean))
	if err := os.Rename(fromAbs, toAbs); err != nil {
		_ = writeRecord(recordPath, original)
		return PublishMove{}, nil, fmt.Errorf("move provenance: %w", err)
	}

	rollback := func() error {
		var failures []string
		fromParent, err := safeMkdirAllInsideRoot(root, filepath.Dir(fromDir))
		if err != nil {
			failures = append(failures, fmt.Sprintf("create provenance rollback parent: %v", err))
		} else if err := os.Rename(toAbs, filepath.Join(fromParent, filepath.Base(fromDir))); err != nil {
			failures = append(failures, fmt.Sprintf("move provenance back: %v", err))
		}
		restoredPath, err := resolvedPathInsideRoot(root, loaded.Path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("resolve restored provenance: %v", err))
		} else if err := writeRecord(restoredPath, original); err != nil {
			failures = append(failures, fmt.Sprintf("restore provenance record: %v", err))
		}
		if len(failures) > 0 {
			return fmt.Errorf("rollback provenance publish failed: %s", strings.Join(failures, "; "))
		}
		return nil
	}

	return PublishMove{
		Record:  loaded.Record,
		FromDir: filepath.ToSlash(fromDir),
		ToDir:   filepath.ToSlash(toClean),
	}, rollback, nil
}

func Verify(root string, idOrPath string) (VerifyResult, error) {
	loaded, err := Load(root, idOrPath)
	if err != nil {
		return VerifyResult{}, err
	}
	return verifyLoaded(root, idOrPath, loaded, loaded.Record.Subject.DraftPath)
}

func verifyLoaded(root string, idOrPath string, loaded Loaded, draftHashPath string) (VerifyResult, error) {
	draftHash, draftHashErr := HashTree(root, draftHashPath)
	return verifyLoadedWithDraftHash(root, idOrPath, loaded, draftHash, draftHashErr)
}

func verifyLoadedWithDraftHash(root string, idOrPath string, loaded Loaded, draftHash string, draftHashErr error) (VerifyResult, error) {
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
	if draftHashErr != nil {
		findings = append(findings, draftHashErr.Error())
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
			checkRel := filepath.Join(loaded.Dir, record.LatestCheck.Path)
			checkHash, err := HashFile(root, checkRel)
			if err != nil {
				findings = append(findings, err.Error())
			} else if record.Hashes.LatestCheckSHA256 == "" {
				findings = append(findings, "latest check hash is required")
			} else if checkHash != record.Hashes.LatestCheckSHA256 {
				findings = append(findings, "latest check hash changed")
			}
			storedCheck, err := loadCheckResponse(root, checkRel)
			if err != nil {
				findings = append(findings, err.Error())
			} else {
				if storedCheck.Result != record.LatestCheck.Result {
					findings = append(findings, "latest check result does not match stored check")
				}
				if storedCheck.Result != "pass" {
					findings = append(findings, "stored latest check must pass")
				}
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

func hashPublishedDraftAsDraft(root string, record Record) (string, error) {
	officialClean, err := cleanRelPath(record.Subject.OfficialPath)
	if err != nil {
		return "", err
	}
	draftClean, err := cleanRelPath(record.Subject.DraftPath)
	if err != nil {
		return "", err
	}
	officialAbs, err := resolvedPathInsideRoot(root, officialClean)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(officialAbs)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(officialClean), err)
	}
	if !info.IsDir() {
		return hashPublishedFileAsDraft(root, officialClean, draftClean)
	}

	var files []string
	if err := filepath.WalkDir(officialAbs, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return "", fmt.Errorf("walk %s: %w", filepath.ToSlash(officialClean), err)
	}
	sort.Strings(files)

	sum := sha256.New()
	for _, officialRel := range files {
		draftRel := rebaseProvenancePath(officialRel, filepath.ToSlash(officialClean), filepath.ToSlash(draftClean))
		fileHash, err := hashPublishedFileAsDraft(root, officialRel, draftRel)
		if err != nil {
			return "", err
		}
		sum.Write([]byte(draftRel))
		sum.Write([]byte{0})
		sum.Write([]byte(fileHash))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func hashPublishedFileAsDraft(root string, officialRel string, draftRel string) (string, error) {
	clean, err := cleanRelPath(officialRel)
	if err != nil {
		return "", err
	}
	absPath, err := resolvedPathInsideRoot(root, clean)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s: expected regular file", filepath.ToSlash(clean))
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filepath.ToSlash(clean), err)
	}
	if strings.HasSuffix(filepath.ToSlash(draftRel), ".md") {
		data = normalizePublishedKnowledgeDraftStatus(data)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func normalizePublishedKnowledgeDraftStatus(data []byte) []byte {
	const opening = "---\n"
	const closing = "\n---\n"
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, opening) {
		return data
	}
	end := strings.Index(text[len(opening):], closing)
	if end < 0 {
		return data
	}
	frontmatterEnd := len(opening) + end
	frontmatter := text[len(opening):frontmatterEnd]
	body := text[frontmatterEnd+len(closing):]
	lines := strings.Split(frontmatter, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "status:") {
			lines[i] = "status: draft"
			break
		}
	}
	return []byte(opening + strings.Join(lines, "\n") + closing + body)
}

func rebaseProvenancePath(path string, from string, to string) string {
	path = filepath.ToSlash(path)
	if path == from {
		return to
	}
	if strings.HasPrefix(path, from+"/") {
		return to + strings.TrimPrefix(path, from)
	}
	return path
}

func shouldVerifyPublishedDraft(root string, record Record) bool {
	if strings.TrimSpace(record.Subject.OfficialPath) == "" {
		return false
	}
	_, draftExists, err := existingPathInsideRoot(root, record.Subject.DraftPath)
	if err != nil || draftExists {
		return false
	}
	_, officialExists, err := existingPathInsideRoot(root, record.Subject.OfficialPath)
	return err == nil && officialExists
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

func writeCheckFile(checksDir string, check knowledgewrite.CheckResponse) (string, error) {
	data, err := json.MarshalIndent(check, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal check: %w", err)
	}
	for attempt := 0; attempt < 16; attempt++ {
		next, err := nextCheckPath(checksDir)
		if err != nil {
			return "", err
		}
		path := filepath.Join(checksDir, next)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("write check: %w", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			file.Close()
			os.Remove(path)
			return "", fmt.Errorf("write check: %w", err)
		}
		if err := file.Close(); err != nil {
			os.Remove(path)
			return "", fmt.Errorf("close check: %w", err)
		}
		return next, nil
	}
	return "", fmt.Errorf("write check: exhausted unique check path attempts")
}

func loadCheckResponse(root string, relPath string) (knowledgewrite.CheckResponse, error) {
	checkPath, err := resolvedPathInsideRoot(root, relPath)
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	data, err := os.ReadFile(checkPath)
	if err != nil {
		return knowledgewrite.CheckResponse{}, fmt.Errorf("read latest check: %w", err)
	}
	var check knowledgewrite.CheckResponse
	if err := json.Unmarshal(data, &check); err != nil {
		return knowledgewrite.CheckResponse{}, fmt.Errorf("parse latest check JSON: %w", err)
	}
	return check, nil
}

func nextCheckPath(checksDir string) (string, error) {
	entries, err := os.ReadDir(checksDir)
	if err != nil {
		return "", fmt.Errorf("read checks dir: %w", err)
	}
	max := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "check-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		numberText := strings.TrimSuffix(strings.TrimPrefix(name, "check-"), ".json")
		number, err := strconv.Atoi(numberText)
		if err != nil {
			continue
		}
		if number > max {
			max = number
		}
	}
	return fmt.Sprintf("check-%03d.json", max+1), nil
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
		if decision.Hashes.DesignSHA256 != hashes.DesignSHA256 {
			*findings = append(*findings, stage+" decision hashes do not match current record")
			return
		}
		if stage == StagePublish &&
			(decision.Hashes.DraftTreeSHA256 != hashes.DraftTreeSHA256 ||
				decision.Hashes.LatestCheckSHA256 != hashes.LatestCheckSHA256) {
			*findings = append(*findings, "publish decision hashes do not match current record")
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

func safeKnowledgeID(id string) string {
	return strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(strings.TrimSpace(id))
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
