# Argos Knowledge Provenance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Git-native knowledge provenance records and make provenance the publish gate for official knowledge.

**Architecture:** Add a focused `internal/provenance` package for provenance records, decisions, checks, hashes, and verification. Keep Markdown knowledge loading in `internal/knowledge`, write design/check behavior in `internal/knowledgewrite`, and wire user-facing commands through `internal/cli`. Publish remains filesystem-first but gains an integrated provenance move and rollback path.

**Tech Stack:** Go standard library, existing Argos filesystem layout, JSON/JSONL files, SHA-256, existing `knowledgewrite.Check`, existing CLI tests.

---

## File Structure

- Create `internal/provenance/model.go`
  - Schema constants and JSON structs for `provenance.json`, decisions, hashes, latest check, verification results.
- Create `internal/provenance/hash.go`
  - Workspace-safe SHA-256 helpers for design files, check files, and draft trees.
- Create `internal/provenance/store.go`
  - Start/load/find/save provenance records, append decisions, store check results, verify records, and move records during publish.
- Create `internal/provenance/provenance_test.go`
  - Unit tests for start, decisions, checks, hash mismatch, verification, and move behavior.
- Modify `internal/workspace/workspace.go`
  - Create `knowledge/.inbox/provenance` and `knowledge/provenance`.
- Modify `internal/cli/cli.go`
  - Add `argos provenance ...` command group.
  - Change public publish command to `argos knowledge publish --provenance <id-or-path>`.
  - Keep existing internal publish helpers, but integrate provenance rollback.
- Modify `internal/cli/cli_test.go`
  - Add CLI coverage for provenance commands and provenance-gated publish.
  - Update usage vocabulary assertions.
  - Replace design/path publish tests with provenance publish tests.
- Modify `README.md`
  - Document Git source of truth, provenance records, and PR review relationship.
- Modify `skills/capture-knowledge/SKILL.md`
  - Replace design-embedded approval authority with provenance decision records.

---

### Task 1: Provenance Model And Hashing

**Files:**
- Create: `internal/provenance/model.go`
- Create: `internal/provenance/hash.go`
- Create: `internal/provenance/provenance_test.go`

- [ ] **Step 1: Write failing hash and model tests**

Add `internal/provenance/provenance_test.go`:

```go
package provenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFileChangesWhenFileChanges(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/redis/design.json", `{"schema_version":"knowledge.design.v1"}`)

	first, err := HashFile(root, "knowledge/.inbox/designs/redis/design.json")
	if err != nil {
		t.Fatalf("HashFile returned error: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/designs/redis/design.json", `{"schema_version":"knowledge.design.v1","changed":true}`)
	second, err := HashFile(root, "knowledge/.inbox/designs/redis/design.json")
	if err != nil {
		t.Fatalf("HashFile returned error after change: %v", err)
	}

	if first == second {
		t.Fatalf("expected hash to change, got %q", first)
	}
	if len(first) != 64 || len(second) != 64 {
		t.Fatalf("expected hex sha256 hashes, got %q and %q", first, second)
	}
}

func TestHashTreeIncludesSupportingFilesInStableOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", "entrypoint\n")
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "reference\n")

	first, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	if err != nil {
		t.Fatalf("HashTree returned error: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed\n")
	second, err := HashTree(root, "knowledge/.inbox/packages/mall-api/redis-cache")
	if err != nil {
		t.Fatalf("HashTree returned error after change: %v", err)
	}

	if first == second {
		t.Fatalf("expected tree hash to change when supporting file changes")
	}
}

func TestHashRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"", ".", "../outside", "/tmp/outside"} {
		if _, err := HashFile(root, rel); err == nil {
			t.Fatalf("HashFile(%q) expected error", rel)
		}
	}
}

func TestRecordJSONShape(t *testing.T) {
	record := Record{
		SchemaVersion: SchemaVersion,
		ProvenanceID: "prov-20260504-redis-cache-a1b2c3d4",
		State:        StateDraft,
		Subject: Subject{
			Kind:         "package",
			KnowledgeID:  "package:mall-api.redis-cache.v1",
			Project:      "mall-api",
			DesignPath:   "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
			DraftPath:    "knowledge/.inbox/packages/mall-api/redis-cache",
			OfficialPath: "knowledge/packages/mall-api/redis-cache",
		},
	}
	if record.SchemaVersion != "knowledge.provenance.v1" {
		t.Fatalf("unexpected schema version: %q", record.SchemaVersion)
	}
	if !strings.HasPrefix(record.ProvenanceID, "prov-") {
		t.Fatalf("unexpected provenance id: %q", record.ProvenanceID)
	}
}

func writeTestFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/provenance -run 'TestHash|TestRecordJSONShape' -count=1
```

Expected: fail because package `internal/provenance` and symbols do not exist.

- [ ] **Step 3: Add the provenance model**

Create `internal/provenance/model.go`:

```go
package provenance

const (
	SchemaVersion         = "knowledge.provenance.v1"
	DecisionSchemaVersion = "knowledge.provenance.decision.v1"

	StateDraft     = "draft"
	StatePublished = "published"

	StageDesign     = "design"
	StageDraftWrite = "draft_write"
	StagePublish    = "publish"

	DecisionApproved         = "approved"
	DecisionChangesRequested = "changes_requested"
	DecisionRejected         = "rejected"
)

type Record struct {
	SchemaVersion string      `json:"schema_version"`
	ProvenanceID string      `json:"provenance_id"`
	State        string      `json:"state"`
	Subject      Subject     `json:"subject"`
	Hashes       Hashes      `json:"hashes"`
	LatestCheck  LatestCheck `json:"latest_check,omitempty"`
	CreatedAt    string      `json:"created_at"`
	CreatedBy    string      `json:"created_by"`
	PublishedAt  string      `json:"published_at,omitempty"`
	PublishedBy  string      `json:"published_by,omitempty"`
	PublishedFrom string     `json:"published_from,omitempty"`
	PublishedTo   string     `json:"published_to,omitempty"`
}

type Subject struct {
	Kind         string `json:"kind"`
	KnowledgeID  string `json:"knowledge_id"`
	Project      string `json:"project"`
	DesignPath   string `json:"design_path"`
	DraftPath    string `json:"draft_path"`
	OfficialPath string `json:"official_path"`
}

type Hashes struct {
	DesignSHA256      string `json:"design_sha256,omitempty"`
	DraftTreeSHA256   string `json:"draft_tree_sha256,omitempty"`
	LatestCheckSHA256 string `json:"latest_check_sha256,omitempty"`
}

type LatestCheck struct {
	Path   string `json:"path,omitempty"`
	Result string `json:"result,omitempty"`
}

type Decision struct {
	SchemaVersion string `json:"schema_version"`
	DecisionID    string `json:"decision_id"`
	Stage         string `json:"stage"`
	Decision      string `json:"decision"`
	DecidedBy     string `json:"decided_by"`
	DeciderRole   string `json:"decider_role"`
	RecordedBy    string `json:"recorded_by"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	Hashes        Hashes `json:"hashes"`
	CreatedAt     string `json:"created_at"`
}

type DecisionInput struct {
	Stage       string
	Decision    string
	DecidedBy   string
	Role        string
	Source      string
	Reason      string
	RecordedBy  string
}

type VerifyResult struct {
	Result   string   `json:"result"`
	ID       string   `json:"provenance_id"`
	Path     string   `json:"path"`
	Findings []string `json:"findings,omitempty"`
}
```

- [ ] **Step 4: Add safe hash helpers**

Create `internal/provenance/hash.go`:

```go
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func HashFile(root string, relPath string) (string, error) {
	clean, err := cleanRelPath(relPath)
	if err != nil {
		return "", err
	}
	file, err := os.Open(filepath.Join(root, clean))
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filepath.ToSlash(clean), err)
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", filepath.ToSlash(clean), err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func HashTree(root string, relPath string) (string, error) {
	clean, err := cleanRelPath(relPath)
	if err != nil {
		return "", err
	}
	absRoot := filepath.Join(root, clean)
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
	}
	if !info.IsDir() {
		return HashFile(root, clean)
	}

	var files []string
	if err := filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
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
		return "", fmt.Errorf("walk %s: %w", filepath.ToSlash(clean), err)
	}
	sort.Strings(files)

	sum := sha256.New()
	for _, rel := range files {
		fileHash, err := HashFile(root, rel)
		if err != nil {
			return "", err
		}
		sum.Write([]byte(rel))
		sum.Write([]byte{0})
		sum.Write([]byte(fileHash))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func cleanRelPath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("%s: path must be relative", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: path must stay inside workspace", relPath)
	}
	return clean, nil
}
```

- [ ] **Step 5: Run model and hash tests**

Run:

```bash
go test ./internal/provenance -run 'TestHash|TestRecordJSONShape' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/provenance/model.go internal/provenance/hash.go internal/provenance/provenance_test.go
git commit -m "feat: add provenance model and hashing"
```

---

### Task 2: Provenance Start And Workspace Roots

**Files:**
- Modify: `internal/workspace/workspace.go`
- Modify: `internal/workspace/workspace_test.go`
- Create/Modify: `internal/provenance/store.go`
- Modify: `internal/provenance/provenance_test.go`

- [ ] **Step 1: Write failing workspace and start tests**

Append to `internal/provenance/provenance_test.go`:

```go
func TestStartCreatesInboxProvenanceRecord(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)

	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if record.State != StateDraft {
		t.Fatalf("expected draft state, got %#v", record)
	}
	if record.Subject.KnowledgeID != "package:mall-api.redis-cache.v1" {
		t.Fatalf("unexpected subject: %#v", record.Subject)
	}
	if record.Subject.OfficialPath != "knowledge/packages/mall-api/redis-cache" {
		t.Fatalf("unexpected official path: %#v", record.Subject)
	}
	if record.Hashes.DesignSHA256 == "" {
		t.Fatalf("expected design hash: %#v", record.Hashes)
	}
	loaded, err := Load(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Record.ProvenanceID != record.ProvenanceID {
		t.Fatalf("loaded wrong record: %#v", loaded.Record)
	}
}
```

Add to `internal/workspace/workspace_test.go`:

```go
func TestInitCreatesProvenanceRoots(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	for _, rel := range []string{
		"knowledge/.inbox/provenance",
		"knowledge/provenance",
	} {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s should be a directory", rel)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
go test ./internal/provenance ./internal/workspace -run 'TestStartCreatesInboxProvenanceRecord|TestInitCreatesProvenanceRoots' -count=1
```

Expected: fail because `Start`, `Load`, `StartRequest`, and provenance roots are not implemented.

- [ ] **Step 3: Add workspace provenance roots**

Modify the `dirs` slice in `internal/workspace/workspace.go`:

```go
dirs := []string{
	"knowledge/.inbox/items",
	"knowledge/.inbox/packages",
	"knowledge/.inbox/designs",
	"knowledge/.inbox/provenance",
	"knowledge/items",
	"knowledge/packages",
	"knowledge/provenance",
	"argos/generated",
}
```

- [ ] **Step 4: Add start/load store implementation**

Create `internal/provenance/store.go`:

```go
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
		ProvenanceID: id,
		State:        StateDraft,
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
```

- [ ] **Step 5: Run start and workspace tests**

Run:

```bash
go test ./internal/provenance ./internal/workspace -run 'TestStartCreatesInboxProvenanceRecord|TestInitCreatesProvenanceRoots' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go internal/provenance/store.go internal/provenance/provenance_test.go
git commit -m "feat: start knowledge provenance records"
```

---

### Task 3: Decision Records

**Files:**
- Modify: `internal/provenance/store.go`
- Modify: `internal/provenance/provenance_test.go`

- [ ] **Step 1: Write failing decision tests**

Append to `internal/provenance/provenance_test.go`:

```go
func TestRecordDecisionAppendsDecisionWithStageHashes(t *testing.T) {
	root := t.TempDir()
	record := startTestRecord(t, root)

	decision, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StageDesign,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Design scope is correct.",
		RecordedBy: "codex",
	})
	if err != nil {
		t.Fatalf("RecordDecision returned error: %v", err)
	}
	if decision.Hashes.DesignSHA256 == "" {
		t.Fatalf("expected design hash on decision: %#v", decision)
	}
	decisions, err := LoadDecisions(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("LoadDecisions returned error: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Stage != StageDesign {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}
}

func TestRecordDecisionRejectsInvalidDecisionInput(t *testing.T) {
	root := t.TempDir()
	record := startTestRecord(t, root)
	input := DecisionInput{
		Stage:      "unknown",
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Bad stage should fail.",
		RecordedBy: "codex",
	}
	if _, err := RecordDecision(root, record.ProvenanceID, input); err == nil {
		t.Fatalf("expected invalid stage error")
	}
}
```

Add this helper near existing test helpers:

```go
func startTestRecord(t *testing.T, root string) Record {
	t.Helper()
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", `{
  "schema_version": "knowledge.design.v1",
  "project": "mall-api",
  "draft_output": {
    "kind": "package",
    "id": "package:mall-api.redis-cache.v1",
    "title": "Redis Cache",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache"
  }
}`)
	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	return record
}
```

- [ ] **Step 2: Run decision tests to verify they fail**

Run:

```bash
go test ./internal/provenance -run TestRecordDecision -count=1
```

Expected: fail because `RecordDecision` and `LoadDecisions` are missing.

- [ ] **Step 3: Implement decision appending**

Append to `internal/provenance/store.go`:

```go
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
	path := filepath.Join(root, loaded.Dir, "decisions.jsonl")
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
	path := filepath.Join(root, loaded.Dir, "decisions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
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
```

- [ ] **Step 4: Run decision tests**

Run:

```bash
go test ./internal/provenance -run TestRecordDecision -count=1
```

Expected: pass.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/provenance/store.go internal/provenance/provenance_test.go
git commit -m "feat: record provenance decisions"
```

---

### Task 4: Check Recording And Verification

**Files:**
- Modify: `internal/provenance/store.go`
- Modify: `internal/provenance/provenance_test.go`

- [ ] **Step 1: Write failing check and verify tests**

Append to `internal/provenance/provenance_test.go`:

```go
func TestRecordCheckStoresCheckAndVerifyPasses(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)

	check, err := RecordCheck(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	if check.Result != "pass" {
		t.Fatalf("expected pass check, got %#v", check)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved after check passed.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "pass" {
		t.Fatalf("expected verify pass, got %#v", result)
	}
}

func TestVerifyFailsWhenDraftChangesAfterPublishDecision(t *testing.T) {
	root := t.TempDir()
	record := startPublishableTestRecord(t, root)
	recordDesignAndDraftApprovals(t, root, record.ProvenanceID)
	if _, err := RecordCheck(root, record.ProvenanceID); err != nil {
		t.Fatalf("RecordCheck returned error: %v", err)
	}
	if _, err := RecordDecision(root, record.ProvenanceID, DecisionInput{
		Stage:      StagePublish,
		Decision:   DecisionApproved,
		DecidedBy:  "chenchi",
		Role:       "knowledge_owner",
		Source:     "conversation",
		Reason:     "Publish approved.",
		RecordedBy: "codex",
	}); err != nil {
		t.Fatalf("record publish decision: %v", err)
	}
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "changed after approval\n")

	result, err := Verify(root, record.ProvenanceID)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Result != "fail" {
		t.Fatalf("expected verify fail, got %#v", result)
	}
	if !containsFinding(result.Findings, "draft tree hash changed") {
		t.Fatalf("expected draft hash finding, got %#v", result.Findings)
	}
}
```

Add helpers:

```go
func startPublishableTestRecord(t *testing.T, root string) Record {
	t.Helper()
	writeTestFile(t, root, "knowledge/domains.yaml", "tech_domains: [backend]\nbusiness_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/projects.yaml", "projects:\n  - id: mall-api\n    name: Mall API\n    path: services/mall-api\n    tech_domains: [backend]\n    business_domains: [catalog]\n")
	writeTestFile(t, root, "knowledge/types.yaml", "types: [package]\n")
	writeTestFile(t, root, "knowledge/.inbox/designs/mall-api/redis-cache/design.json", validProvenanceDesignJSON())
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md", validProvenanceDraftPackage())
	writeTestFile(t, root, "knowledge/.inbox/packages/mall-api/redis-cache/references/redis.md", "reference\n")
	record, err := Start(root, StartRequest{
		DesignPath: "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
		DraftPath:  "knowledge/.inbox/packages/mall-api/redis-cache",
		CreatedBy:  "codex",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	return record
}

func recordDesignAndDraftApprovals(t *testing.T, root string, id string) {
	t.Helper()
	for _, stage := range []string{StageDesign, StageDraftWrite} {
		if _, err := RecordDecision(root, id, DecisionInput{
			Stage:      stage,
			Decision:   DecisionApproved,
			DecidedBy:  "chenchi",
			Role:       "knowledge_owner",
			Source:     "conversation",
			Reason:     stage + " approved.",
			RecordedBy: "codex",
		}); err != nil {
			t.Fatalf("RecordDecision(%s) returned error: %v", stage, err)
		}
	}
}

func containsFinding(findings []string, text string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, text) {
			return true
		}
	}
	return false
}
```

Add valid JSON fixtures based on the existing `knowledgewrite` schema:

```go
func validProvenanceDesignJSON() string {
	return `{
  "schema_version": "knowledge.design.v1",
  "user_request": "Create Redis cache best practices.",
  "knowledge_goal": "Document Redis cache best practices.",
  "project": "mall-api",
  "audience": {
    "primary": "implementer_agent",
    "agent_actions_supported": ["implement Redis cache behavior"]
  },
  "scope": {
    "projects": ["mall-api"],
    "stability": "reviewed",
    "distribution": "project",
    "tech_domains": ["backend"],
    "subject_domains": ["catalog"],
    "file_globs": ["**/*"]
  },
  "sources": {
    "user_input": ["User requested Redis cache best practices."],
    "observed": [],
    "imported": [],
    "ai_suggested": [],
    "templates": [],
    "examples": [],
    "assumptions": [],
    "open_questions": [],
    "claims": [{
      "claim": "Redis cache best practices are needed.",
      "kind": "fact",
      "source": ["user request"],
      "trust": "user_input",
      "requires_review": false
    }]
  },
  "draft_output": {
    "kind": "package",
    "type": "package",
    "title": "Redis Cache Best Practices",
    "id": "package:mall-api.redis-cache.v1",
    "path": "knowledge/.inbox/packages/mall-api/redis-cache",
    "status": "draft",
    "priority": "should",
    "rationale": "A package can hold guidance and references.",
    "entrypoint_load": "read_before_implementation",
    "draft_state": "draft"
  },
  "future_use": {
    "trigger_requests": ["implement Redis cache behavior"],
    "negative_triggers": ["unrelated auth work"],
    "phases": ["implementation"],
    "query_phrases": ["redis cache best practices"],
    "expected_use": "Read before implementing Redis cache behavior.",
    "citation_policy": "cite_after_use",
    "missing_needs": []
  },
  "applicability": {
    "when_to_use": ["When implementing Redis cache behavior."],
    "when_not_to_use": ["When work does not touch Redis."],
    "tradeoffs": []
  },
  "existing_knowledge": {
    "official_matches": [],
    "inbox_matches": [],
    "possible_matches": [],
    "decision": "create_new",
    "reason": "No existing Redis cache package.",
    "review_choice_required": false
  },
  "write_boundary": {
    "path": "inbox",
    "write_requires_review_approval": true,
    "priority_must_approved": false,
    "publish_approved": true,
    "official_write_approved": false,
    "review_packet_required": true
  },
  "draft_files": [{
    "path": "knowledge/.inbox/packages/mall-api/redis-cache/KNOWLEDGE.md",
    "purpose": "Package entrypoint.",
    "load": "read_before_implementation"
  }],
  "check_plan": {
    "validate_path": "knowledge/.inbox/packages/mall-api/redis-cache",
    "findability_checks": [{
      "project": "mall-api",
      "phase": "implementation",
      "task": "implement Redis cache behavior",
      "query": "redis cache best practices",
      "files": []
    }]
  },
  "review": {
    "questions": ["Is publication approved?"],
    "design_approved": true,
    "draft_write_approved": true,
    "priority_must_approved": false,
    "official_write_approved": false,
    "publish_approved": true
  }
}`
}

func validProvenanceDraftPackage() string {
	return `---
id: package:mall-api.redis-cache.v1
title: Redis Cache Best Practices
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: draft
priority: should
tags: [redis, cache]
updated_at: 2026-05-04
applies_to:
  files: ["**/*"]
---
## Purpose

Document Redis cache best practices.

## When To Use

Use when implementing Redis cache behavior.

## Start Here

Read before implementing Redis cache behavior.

## Load On Demand

- references/redis.md when deeper Redis detail is needed.
`
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/provenance -run 'TestRecordCheck|TestVerify' -count=1
```

Expected: fail because `RecordCheck` and `Verify` are not implemented.

- [ ] **Step 3: Implement check recording and verification**

Append to `internal/provenance/store.go`:

```go
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
	checksDir := filepath.Join(root, loaded.Dir, "checks")
	if err := os.MkdirAll(checksDir, 0o755); err != nil {
		return knowledgewrite.CheckResponse{}, fmt.Errorf("create checks dir: %w", err)
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
	loaded.Record.Hashes.LatestCheckSHA256, err = HashFile(root, filepath.Join(loaded.Dir, "checks", next))
	if err != nil {
		return knowledgewrite.CheckResponse{}, err
	}
	loaded.Record.LatestCheck = LatestCheck{Path: filepath.ToSlash(filepath.Join("checks", next)), Result: check.Result}
	if err := writeRecord(filepath.Join(root, loaded.Path), loaded.Record); err != nil {
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
	} else if record.Hashes.DesignSHA256 != "" && designHash != record.Hashes.DesignSHA256 {
		findings = append(findings, "design hash changed")
	}
	draftHash, err := HashTree(root, record.Subject.DraftPath)
	if err != nil {
		findings = append(findings, err.Error())
	} else if record.Hashes.DraftTreeSHA256 != "" && draftHash != record.Hashes.DraftTreeSHA256 {
		findings = append(findings, "draft tree hash changed")
	}
	if record.LatestCheck.Result != "pass" {
		findings = append(findings, "latest check must pass")
	}
	if record.LatestCheck.Path == "" {
		findings = append(findings, "latest check is required")
	} else {
		checkHash, err := HashFile(root, filepath.Join(loaded.Dir, record.LatestCheck.Path))
		if err != nil {
			findings = append(findings, err.Error())
		} else if record.Hashes.LatestCheckSHA256 != "" && checkHash != record.Hashes.LatestCheckSHA256 {
			findings = append(findings, "latest check hash changed")
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

func nextCheckPath(checksDir string) string {
	entries, _ := os.ReadDir(checksDir)
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
```

- [ ] **Step 4: Run check and verify tests**

Run:

```bash
go test ./internal/provenance -run 'TestRecordCheck|TestVerify' -count=1
```

Expected: pass.

- [ ] **Step 5: Run all provenance tests**

Run:

```bash
go test ./internal/provenance -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/provenance/store.go internal/provenance/provenance_test.go
git commit -m "feat: verify knowledge provenance"
```

---

### Task 5: Provenance CLI Commands

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests for provenance commands**

Add tests to `internal/cli/cli_test.go` near write CLI tests:

```go
func TestRunProvenanceStartRecordCheckAndVerify(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign("knowledge/.inbox/packages/backend/redis/best-practices", "package:backend.redis.best-practices.v1"))
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage("package:backend.redis.best-practices.v1"))
	chdir(t, root)

	startOutput := runOK(t, root, []string{"provenance", "start", "--json", "--design", designPath, "--draft", draftPath, "--created-by", "codex"})
	var startResult struct {
		ProvenanceID string `json:"provenance_id"`
	}
	if err := json.Unmarshal([]byte(startOutput), &startResult); err != nil {
		t.Fatalf("parse start JSON: %v\n%s", err, startOutput)
	}
	if startResult.ProvenanceID == "" {
		t.Fatalf("start output missing provenance_id: %s", startOutput)
	}
	for _, stage := range []string{"design", "draft_write"} {
		runOK(t, root, []string{
			"provenance", "record-decision", "--json",
			"--provenance", startResult.ProvenanceID,
			"--stage", stage,
			"--decision", "approved",
			"--decided-by", "chenchi",
			"--role", "knowledge_owner",
			"--source", "conversation",
			"--reason", stage + " approved",
			"--recorded-by", "codex",
		})
	}
	runOK(t, root, []string{"provenance", "record-check", "--json", "--provenance", startResult.ProvenanceID})
	runOK(t, root, []string{
		"provenance", "record-decision", "--json",
		"--provenance", startResult.ProvenanceID,
		"--stage", "publish",
		"--decision", "approved",
		"--decided-by", "chenchi",
		"--role", "knowledge_owner",
		"--source", "conversation",
		"--reason", "publish approved",
		"--recorded-by", "codex",
	})
	verifyOutput := runOK(t, root, []string{"provenance", "verify", "--json", "--provenance", startResult.ProvenanceID})
	if !strings.Contains(verifyOutput, `"result": "pass"`) {
		t.Fatalf("expected verify pass, got %s", verifyOutput)
	}
}

func TestRunProvenanceRequiresJSONForMachineReadableCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"provenance", "start", "--design", "design.json", "--draft", "knowledge/.inbox/packages/x"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "provenance start: --json is required") {
		t.Fatalf("expected json error, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run TestRunProvenance -count=1
```

Expected: fail because `provenance` command is unknown.

- [ ] **Step 3: Import provenance and route command**

Modify imports in `internal/cli/cli.go`:

```go
"argos/internal/provenance"
```

Add to `run` switch:

```go
case "provenance":
	return runProvenance(args[1:], stdout, stderr)
```

- [ ] **Step 4: Add provenance CLI handlers**

Add after `runProjectList`:

```go
func runProvenance(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "provenance: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "start":
		return runProvenanceStart(args[1:], stdout, stderr)
	case "record-decision":
		return runProvenanceRecordDecision(args[1:], stdout, stderr)
	case "record-check":
		return runProvenanceRecordCheck(args[1:], stdout, stderr)
	case "verify":
		return runProvenanceVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "provenance: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}
```

Add the handlers:

```go
func runProvenanceStart(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance start", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	designPath := flags.String("design", "", "knowledge design JSON path")
	draftPath := flags.String("draft", "", "draft item or package path")
	createdBy := flags.String("created-by", "", "actor recording provenance")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance start: --json is required")
		return 2
	}
	if strings.TrimSpace(*designPath) == "" || strings.TrimSpace(*draftPath) == "" {
		fmt.Fprintln(stderr, "provenance start: --design and --draft are required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance start: get current directory: %v\n", err)
		return 1
	}
	record, err := provenance.Start(root, provenance.StartRequest{
		DesignPath: *designPath,
		DraftPath:  *draftPath,
		CreatedBy:  *createdBy,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance start: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, record)
}

func runProvenanceRecordDecision(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance record-decision", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	id := flags.String("provenance", "", "provenance id or path")
	stage := flags.String("stage", "", "decision stage")
	decisionValue := flags.String("decision", "", "decision")
	decidedBy := flags.String("decided-by", "", "decision actor")
	role := flags.String("role", "", "decision actor role")
	source := flags.String("source", "", "decision source")
	reason := flags.String("reason", "", "decision reason")
	recordedBy := flags.String("recorded-by", "", "recording actor")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance record-decision: --json is required")
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(stderr, "provenance record-decision: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-decision: get current directory: %v\n", err)
		return 1
	}
	decision, err := provenance.RecordDecision(root, *id, provenance.DecisionInput{
		Stage:      *stage,
		Decision:   *decisionValue,
		DecidedBy:  *decidedBy,
		Role:       *role,
		Source:     *source,
		Reason:     *reason,
		RecordedBy: *recordedBy,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-decision: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, decision)
}

func runProvenanceRecordCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance record-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	id := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance record-check: --json is required")
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(stderr, "provenance record-check: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-check: get current directory: %v\n", err)
		return 1
	}
	check, err := provenance.RecordCheck(root, *id)
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-check: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, check)
}

func runProvenanceVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	id := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance verify: --json is required")
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(stderr, "provenance verify: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance verify: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.Verify(root, *id)
	if err != nil {
		fmt.Fprintf(stderr, "provenance verify: %v\n", err)
		return 1
	}
	if result.Result != "pass" {
		printJSON(stdout, stderr, result)
		return 1
	}
	return printJSON(stdout, stderr, result)
}
```

- [ ] **Step 5: Update root usage**

Add to `printUsage`:

```go
fmt.Fprintln(w, "  argos provenance start --json --design <design.json> --draft <draft>")
fmt.Fprintln(w, "  argos provenance record-decision --json --provenance <id>")
fmt.Fprintln(w, "  argos provenance record-check --json --provenance <id>")
fmt.Fprintln(w, "  argos provenance verify --json --provenance <id>")
```

- [ ] **Step 6: Run CLI provenance tests**

Run:

```bash
go test ./internal/cli -run TestRunProvenance -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: add provenance CLI commands"
```

---

### Task 6: Provenance-Gated Publish

**Files:**
- Modify: `internal/provenance/store.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing publish tests**

Add to `internal/cli/cli_test.go`:

```go
func TestRunKnowledgePublishWithProvenanceMovesDraftAndProvenance(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)

	provenanceID := createPublishableCLIProvenance(t, root, designPath, draftPath)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--provenance", provenanceID}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected official package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/.inbox/provenance", provenanceID)); !os.IsNotExist(err) {
		t.Fatalf("expected inbox provenance moved, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "knowledge/provenance", strings.ReplaceAll(draftID, ":", "_"), provenanceID, "provenance.json")); err != nil {
		t.Fatalf("expected published provenance: %v", err)
	}
	if !strings.Contains(stdout.String(), "published knowledge/packages/backend/redis/best-practices") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunKnowledgePublishWithProvenanceRejectsChangedDraft(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	draftPath := "knowledge/.inbox/packages/backend/redis/best-practices"
	draftID := "package:backend.redis.best-practices.v1"
	designPath := writeCLIKnowledgeDesign(t, root, "knowledge/.inbox/designs/redis/design.json", validCLIKnowledgeDesign(draftPath, draftID))
	writeCLIFile(t, root, draftPath+"/KNOWLEDGE.md", validCLICheckDraftPackage(draftID))
	chdir(t, root)
	provenanceID := createPublishableCLIProvenance(t, root, designPath, draftPath)
	writeCLIFile(t, root, draftPath+"/references/changed.md", "changed after approval\n")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"knowledge", "publish", "--provenance", provenanceID}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "draft tree hash changed") {
		t.Fatalf("expected hash mismatch error, got %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, draftPath, "KNOWLEDGE.md")); err != nil {
		t.Fatalf("expected draft to remain: %v", err)
	}
}
```

Add helper:

```go
func createPublishableCLIProvenance(t *testing.T, root string, designPath string, draftPath string) string {
	t.Helper()
	startOutput := runOK(t, root, []string{"provenance", "start", "--json", "--design", designPath, "--draft", draftPath, "--created-by", "codex"})
	var start struct {
		ProvenanceID string `json:"provenance_id"`
	}
	if err := json.Unmarshal([]byte(startOutput), &start); err != nil {
		t.Fatalf("parse provenance start: %v\n%s", err, startOutput)
	}
	for _, stage := range []string{"design", "draft_write"} {
		runOK(t, root, []string{"provenance", "record-decision", "--json", "--provenance", start.ProvenanceID, "--stage", stage, "--decision", "approved", "--decided-by", "chenchi", "--role", "knowledge_owner", "--source", "conversation", "--reason", stage + " approved", "--recorded-by", "codex"})
	}
	runOK(t, root, []string{"provenance", "record-check", "--json", "--provenance", start.ProvenanceID})
	runOK(t, root, []string{"provenance", "record-decision", "--json", "--provenance", start.ProvenanceID, "--stage", "publish", "--decision", "approved", "--decided-by", "chenchi", "--role", "knowledge_owner", "--source", "conversation", "--reason", "publish approved", "--recorded-by", "codex"})
	return start.ProvenanceID
}
```

- [ ] **Step 2: Run publish tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgePublishWithProvenance' -count=1
```

Expected: fail because `knowledge publish --provenance` is not implemented.

- [ ] **Step 3: Add provenance movement helpers**

Append to `internal/provenance/store.go`:

```go
type PublishMove struct {
	Record Record
	FromDir string
	ToDir   string
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
	if verify.Result != "pass" {
		return PublishMove{}, nil, fmt.Errorf("provenance verify failed: %s", strings.Join(verify.Findings, "; "))
	}
	record := loaded.Record
	fromDir := filepath.Join(root, loaded.Dir)
	toRel := filepath.Join("knowledge", "provenance", safeKnowledgeID(record.Subject.KnowledgeID), record.ProvenanceID)
	toDir := filepath.Join(root, toRel)
	if _, err := os.Stat(toDir); err == nil {
		return PublishMove{}, nil, fmt.Errorf("published provenance target already exists: %s", filepath.ToSlash(toRel))
	} else if err != nil && !os.IsNotExist(err) {
		return PublishMove{}, nil, fmt.Errorf("stat published provenance target: %w", err)
	}
	original := record
	record.State = StatePublished
	record.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	record.PublishedBy = strings.TrimSpace(publishedBy)
	if record.PublishedBy == "" {
		record.PublishedBy = "unknown"
	}
	record.PublishedFrom = record.Subject.DraftPath
	record.PublishedTo = record.Subject.OfficialPath
	if err := writeRecord(filepath.Join(fromDir, "provenance.json"), record); err != nil {
		return PublishMove{}, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(toDir), 0o755); err != nil {
		_ = writeRecord(filepath.Join(fromDir, "provenance.json"), original)
		return PublishMove{}, nil, fmt.Errorf("create published provenance parent: %w", err)
	}
	if err := os.Rename(fromDir, toDir); err != nil {
		_ = writeRecord(filepath.Join(fromDir, "provenance.json"), original)
		return PublishMove{}, nil, fmt.Errorf("move provenance: %w", err)
	}
	rollback := func() error {
		if err := os.MkdirAll(filepath.Dir(fromDir), 0o755); err != nil {
			return err
		}
		if err := os.Rename(toDir, fromDir); err != nil {
			return err
		}
		return writeRecord(filepath.Join(fromDir, "provenance.json"), original)
	}
	return PublishMove{Record: record, FromDir: filepath.ToSlash(loaded.Dir), ToDir: filepath.ToSlash(toRel)}, rollback, nil
}

func safeKnowledgeID(id string) string {
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_")
	return replacer.Replace(strings.TrimSpace(id))
}
```

- [ ] **Step 4: Refactor publish side-effect rollback**

Modify `internal/cli/cli.go`:

1. Add a side-effect type:

```go
type publishSideEffect func(target string) (func() error, error)
```

2. Change `publishDraft` to call a new helper:

```go
func publishDraft(root string, relPath string, stderr io.Writer) (string, error) {
	return publishDraftWithSideEffect(root, relPath, stderr, nil)
}
```

3. Rename the current `publishDraft` body to:

```go
func publishDraftWithSideEffect(root string, relPath string, stderr io.Writer, sideEffect publishSideEffect) (string, error) {
```

4. After `activatePublishedKnowledge` succeeds and before final official validation, add:

```go
var rollbackSideEffect func() error
if sideEffect != nil {
	var err error
	rollbackSideEffect, err = sideEffect(target)
	if err != nil {
		if rollbackErr := rollbackPublishedKnowledge(root, target, clean, snapshots); rollbackErr != nil {
			return "", fmt.Errorf("publish side effect failed: %w; %v", err, rollbackErr)
		}
		return "", fmt.Errorf("publish side effect failed: %w", err)
	}
}
```

5. In the final validation rollback block, call the side-effect rollback first:

```go
if _, err := loadAndValidateKnowledge(root, stderr, validationScope{Path: target}); err != nil {
	var rollbackFailures []string
	if rollbackSideEffect != nil {
		if rollbackErr := rollbackSideEffect(); rollbackErr != nil {
			rollbackFailures = append(rollbackFailures, rollbackErr.Error())
		}
	}
	if rollbackErr := rollbackPublishedKnowledge(root, target, clean, snapshots); rollbackErr != nil {
		rollbackFailures = append(rollbackFailures, rollbackErr.Error())
	}
	if len(rollbackFailures) > 0 {
		return "", fmt.Errorf("official validation failed after publish: %w; rollback failed: %s", err, strings.Join(rollbackFailures, "; "))
	}
	return "", fmt.Errorf("official validation failed after publish: %w", err)
}
```

- [ ] **Step 5: Wire `knowledge publish --provenance`**

Replace `runKnowledgePublish` parsing with:

```go
provenancePath := flags.String("provenance", "", "knowledge provenance id or path")
publishedBy := flags.String("published-by", "", "actor publishing knowledge")
```

Require provenance and remove the public `--design` / `--path` path:

```go
if strings.TrimSpace(*provenancePath) == "" {
	fmt.Fprintln(stderr, "knowledge publish: --provenance is required")
	return 2
}
root, err := os.Getwd()
if err != nil {
	fmt.Fprintf(stderr, "knowledge publish: get current directory: %v\n", err)
	return 1
}
loaded, err := provenance.Load(root, *provenancePath)
if err != nil {
	fmt.Fprintf(stderr, "knowledge publish: %v\n", err)
	return 1
}
target, err := publishDraftWithSideEffect(root, loaded.Record.Subject.DraftPath, stderr, func(target string) (func() error, error) {
	if filepath.ToSlash(target) != loaded.Record.Subject.OfficialPath {
		return nil, fmt.Errorf("publish target %s does not match provenance official path %s", filepath.ToSlash(target), loaded.Record.Subject.OfficialPath)
	}
	_, rollback, err := provenance.PreparePublishMove(root, *provenancePath, *publishedBy)
	return rollback, err
})
if err != nil {
	fmt.Fprintf(stderr, "knowledge publish: %v\n", err)
	return 1
}
fmt.Fprintf(stdout, "published %s\n", target)
fmt.Fprintln(stdout, "run argos index to refresh query results")
return 0
```

- [ ] **Step 6: Run provenance publish tests**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgePublishWithProvenance' -count=1
```

Expected: pass.

- [ ] **Step 7: Run existing publish tests and update expectations**

Run:

```bash
go test ./internal/cli -run 'TestRunKnowledgePublish|TestKnowledgeWritePublishAndFindbackFlow' -count=1
```

Expected initially: old tests that use `--design` and `--path` fail with `knowledge publish: --provenance is required`.

Update those tests to create publishable provenance via `createPublishableCLIProvenance` and call:

```go
Run([]string{"knowledge", "publish", "--provenance", provenanceID}, &stdout, &stderr)
```

For tests that intentionally check missing args, replace old missing-design/path assertions with:

```go
if !strings.Contains(stderr.String(), "knowledge publish: --provenance is required") {
	t.Fatalf("expected missing provenance error, got %q", stderr.String())
}
```

Run again:

```bash
go test ./internal/cli -run 'TestRunKnowledgePublish|TestKnowledgeWritePublishAndFindbackFlow' -count=1
```

Expected: pass.

- [ ] **Step 8: Commit**

Run:

```bash
git add internal/provenance/store.go internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: publish knowledge with provenance"
```

---

### Task 7: Full Flow, Usage, Docs, And Skill Updates

**Files:**
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing usage vocabulary test update**

Modify `TestUsageUsesWriteVocabulary` expected strings in `internal/cli/cli_test.go`:

```go
for _, want := range []string{
	"argos project add --id <project> --name <name> --path <path>",
	"argos project list --json",
	"argos knowledge design --json --project <project> --intent <intent>",
	"argos knowledge check --json --design <design.json> --draft <draft>",
	"argos provenance start --json --design <design.json> --draft <draft>",
	"argos provenance record-decision --json --provenance <id>",
	"argos provenance record-check --json --provenance <id>",
	"argos provenance verify --json --provenance <id>",
	"argos knowledge publish --provenance <id>",
}
```

Add removed surface checks:

```go
for _, forbidden := range []string{
	"review.publish_approved is required before publish",
	"argos knowledge publish --design <design.json> --path <draft>",
} {
	if strings.Contains(body, forbidden) {
		t.Fatalf("usage contains old publish authority %q:\n%s", forbidden, body)
	}
}
```

- [ ] **Step 2: Run usage test to verify it fails**

Run:

```bash
go test ./internal/cli -run TestUsageUsesWriteVocabulary -count=1
```

Expected: fail until usage is updated.

- [ ] **Step 3: Update usage text**

In `printUsage`, replace the old publish line:

```go
fmt.Fprintln(w, "  argos knowledge publish --design <design.json> --path <draft>")
```

with:

```go
fmt.Fprintln(w, "  argos provenance start --json --design <design.json> --draft <draft>")
fmt.Fprintln(w, "  argos provenance record-decision --json --provenance <id>")
fmt.Fprintln(w, "  argos provenance record-check --json --provenance <id>")
fmt.Fprintln(w, "  argos provenance verify --json --provenance <id>")
fmt.Fprintln(w, "  argos knowledge publish --provenance <id>")
```

- [ ] **Step 4: Update README authoring flow**

In `README.md`, update "Designing And Publishing Knowledge" to the new flow:

```markdown
1. Run `argos project list --json`; if the target project is missing, run `argos project add --id <project> --name <name> --path <path>`.
2. Run `argos knowledge design --json --project <project> --intent <intent>`.
3. Write the returned `knowledge_design_template` to `write_guidance.design_path`.
4. Start provenance with `argos provenance start --json --design <design.json> --draft <draft-path>`.
5. Record the human design decision with `argos provenance record-decision --json --stage design`.
6. Write draft knowledge only after the design and draft-write decisions are recorded.
7. Run `argos provenance record-check --json --provenance <id>`.
8. Record the publish decision with `argos provenance record-decision --json --stage publish`.
9. Run `argos provenance verify --json --provenance <id>`.
10. Publish with `argos knowledge publish --provenance <id>`.
11. Run `argos index`.
12. Confirm the new knowledge is discoverable with `argos knowledge find --json`.
```

Add a short provenance section:

```markdown
### Provenance Contract

Git-tracked knowledge files are the source of truth. SQLite indexes are generated
caches. Argos provenance records under `knowledge/.inbox/provenance/` and
`knowledge/provenance/` record design hashes, draft tree hashes, check results,
and human decision records for each publish attempt.

Argos provenance does not replace PR review. In team mode, agents publish
official candidates on a branch and reviewers inspect the knowledge diff plus
provenance diff before merge.
```

- [ ] **Step 5: Update capture skill**

In `skills/capture-knowledge/SKILL.md`:

Replace guidance that treats design review fields as publish authority with:

```markdown
Design JSON describes intent, scope, sources, draft output, and check plan. It
does not prove approval. Approval and publish evidence must be recorded in Argos
provenance.
```

Update the publish command block to:

```bash
argos knowledge publish --provenance PROVENANCE_ID
```

Add the standard provenance sequence:

```bash
argos provenance start --json --design DESIGN_PATH --draft DRAFT_PATH
argos provenance record-decision --json --provenance PROVENANCE_ID --stage design --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
argos provenance record-decision --json --provenance PROVENANCE_ID --stage draft_write --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
argos provenance record-check --json --provenance PROVENANCE_ID
argos provenance record-decision --json --provenance PROVENANCE_ID --stage publish --decision approved --decided-by ACTOR --role knowledge_owner --source conversation --reason "..." --recorded-by AGENT
argos provenance verify --json --provenance PROVENANCE_ID
```

- [ ] **Step 6: Run usage and removed-vocabulary scans**

Run:

```bash
go test ./internal/cli -run TestUsageUsesWriteVocabulary -count=1
rg -n "review.publish_approved is required before publish|argos knowledge publish --design <design.json> --path <draft>|review package|approval package|release package|human_review|authoring_packet|proposal_scaffold|proposal artifact" README.md skills/capture-knowledge/SKILL.md internal/cli/cli.go internal/cli/cli_test.go
```

Expected:

- Go test passes.
- `rg` exits 1 with no output.

- [ ] **Step 7: Commit**

Run:

```bash
git add README.md skills/capture-knowledge/SKILL.md internal/cli/cli.go internal/cli/cli_test.go
git commit -m "docs: document provenance publishing"
```

---

### Task 8: End-To-End Provenance Release Verification

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`

- [ ] **Step 1: Update fresh write-to-query flow test**

Modify `TestKnowledgeWritePublishAndFindbackFlow` so it uses the complete provenance path:

```go
provenanceID := createPublishableCLIProvenance(t, root, designPath, draftPath)
runOK(t, root, []string{"knowledge", "publish", "--provenance", provenanceID})
runOK(t, root, []string{"index"})
```

After publish, assert the provenance record moved:

```go
publishedProvenance := filepath.Join(root, "knowledge/provenance/package_mall-api.redis-cache.v1", provenanceID, "provenance.json")
if _, err := os.Stat(publishedProvenance); err != nil {
	t.Fatalf("expected published provenance: %v", err)
}
```

- [ ] **Step 2: Add no-provenance-query test**

Add:

```go
func TestKnowledgeFindDoesNotLoadProvenanceAsKnowledge(t *testing.T) {
	root := t.TempDir()
	initWorkspace(t, root)
	writeCLIFile(t, root, "knowledge/provenance/package_mall-api.redis-cache.v1/prov-20260504-redis/KNOWLEDGE.md", `---
id: package:provenance.should-not-index.v1
title: Should Not Index
type: package
tech_domains: [backend]
business_domains: [catalog]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-05-04
---
This provenance directory must not be indexed as knowledge.
`)
	chdir(t, root)

	runOK(t, root, []string{"index"})
	output := runOK(t, root, []string{"knowledge", "find", "--json", "--project", "mall-api", "--query", "Should Not Index"})
	if strings.Contains(output, "package:provenance.should-not-index.v1") {
		t.Fatalf("provenance directory was indexed as knowledge: %s", output)
	}
}
```

- [ ] **Step 3: Run end-to-end tests**

Run:

```bash
go test ./internal/cli -run 'TestKnowledgeWritePublishAndFindbackFlow|TestKnowledgeFindDoesNotLoadProvenanceAsKnowledge' -count=1
```

Expected: pass.

- [ ] **Step 4: Update readiness report**

In `docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md`, update:

- Status remains `ready for write/query core`.
- Fresh workspace smoke notes write-to-query loop passes with provenance.
- Release blockers remain none for core write/query.
- Later audit/review-history item becomes provenance extensions:

```markdown
- Add optional external PR metadata ingestion and signed attestations for stricter team environments.
```

- [ ] **Step 5: Run full verification**

Run:

```bash
go test ./... -count=1
git diff --check
```

Expected:

- All Go packages pass.
- `git diff --check` exits 0 with no output.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/cli/cli_test.go docs/superpowers/reports/2026-05-04-argos-global-release-readiness.md
git commit -m "test: cover provenance write flow"
```

---

## Final Verification

Run:

```bash
go test ./... -count=1
git diff --check
git status --short --branch
git log --oneline --decorate -12
git diff --stat main...HEAD
```

Expected:

- Full test suite passes.
- No whitespace errors.
- Branch is `knowledge-provenance`.
- Commits follow the task boundaries.
- Changed files are limited to provenance implementation, CLI wiring, docs, skill, tests, workspace roots, and readiness report.

## Spec Coverage Review

This plan covers the design spec as follows:

- Git source of truth: Task 7 README and skill updates.
- Provenance storage layout: Tasks 1, 2, 6.
- Provenance ID and schema: Tasks 1, 2.
- Decisions JSONL: Task 3.
- Check records: Task 4.
- Hashing: Task 1 and Task 4 verification.
- CLI contract: Task 5.
- Publish contract: Task 6.
- Agent flow and PR fit: Task 7 docs/skill updates.
- Query behavior: Task 8 no-provenance-query test.
- Migration away from design approval authority: Tasks 6 and 7.
- Readiness definition: Task 8 and Final Verification.
