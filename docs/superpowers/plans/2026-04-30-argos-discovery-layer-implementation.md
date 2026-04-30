# Argos Discovery Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Argos Discovery v1 harness so agents can inventory available knowledge, route current work to relevant knowledge, load selected items progressively, and avoid Argos-backed claims when coverage is weak or absent.

**Architecture:** Extend the existing SQLite index in `internal/index`, add discovery/map query APIs in `internal/query`, expose them through CLI JSON and MCP tools, then update adapters and README so agents use the new progressive discovery flow. The default implementation is SQLite-only and local-first; semantic/vector support is represented as capability metadata and schema readiness, but no model runtime is required.

**Tech Stack:** Go, SQLite via `modernc.org/sqlite`, SQLite FTS5, existing `knowledge.Item`, table-driven tests, CLI JSON, MCP JSON-RPC tool calls, generated Markdown adapters.

---

## File Structure

- Modify `internal/index/store.go`: add tags to the item schema, add FTS/chunk/vector/metadata tables, populate them during rebuild, expose lightweight search helpers.
- Modify `internal/index/store_test.go`: cover tags, FTS, chunks, package entrypoints, and schema checks.
- Modify `internal/query/query.go`: add discovery/map request and response types, scoring, coverage, next-call generation, and capability reporting.
- Modify `internal/query/query_test.go`: cover strong/partial/weak/none coverage, progressive disclosure, ranking, filters, and package grouping.
- Modify `internal/cli/cli.go`: add `argos discover --json` and `argos map --json`.
- Modify `internal/cli/cli_test.go`: cover CLI JSON, missing index errors, argument validation, and progressive disclosure.
- Modify `internal/mcp/server.go`: expose MCP tools `argos_discover` and `argos_map` with strict schemas and tool-call handlers.
- Modify `internal/mcp/server_test.go`: cover tool discovery, schemas, success calls, strict args, index-missing errors, and no full bodies.
- Modify `internal/adapters/adapters.go`: teach generated adapters to prefer `argos_discover` and `argos_map` while preserving progressive loading.
- Modify `internal/adapters/adapters_test.go`: assert the stronger discovery contract and guard against unimplemented next calls.
- Modify `README.md`: document Discovery v1, CLI commands, MCP tools, coverage semantics, and no-embedding default.

## Task 1: Extend The Index Schema For Discovery

**Files:**
- Modify: `internal/index/store_test.go`
- Modify: `internal/index/store.go`

- [ ] **Step 1: Write failing index tests for tags, FTS, chunks, and metadata**

Add these tests to `internal/index/store_test.go` after `TestCheckSchemaAcceptsRebuiltIndex`:

```go
func TestRebuildStoresDiscoveryMetadata(t *testing.T) {
	root := t.TempDir()
	item := testItem("rule:backend.auth.v1", "Refresh token auth rule")
	item.Tags = []string{"auth", "refresh-token"}
	item.Body = "Refresh token rotation must be explicit."

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{item}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	got, err := store.GetItem("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("GetItem returned error: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "auth" || got.Tags[1] != "refresh-token" {
		t.Fatalf("expected tags to round-trip, got %#v", got.Tags)
	}

	caps, err := store.DiscoveryCapabilities()
	if err != nil {
		t.Fatalf("DiscoveryCapabilities returned error: %v", err)
	}
	if caps.Metadata != "enabled" || caps.FTS != "enabled" || caps.Semantic != "disabled" {
		t.Fatalf("unexpected capabilities: %#v", caps)
	}
	if caps.SemanticReason != "semantic provider is not configured" {
		t.Fatalf("unexpected semantic reason: %q", caps.SemanticReason)
	}
}

func TestSearchTextFindsTitleBodyAndTags(t *testing.T) {
	root := t.TempDir()
	auth := testItem("rule:backend.auth.v1", "Refresh token auth rule")
	auth.Tags = []string{"session-renewal"}
	auth.Body = "Access tokens are short lived."
	cache := testItem("rule:backend.cache.v1", "Redis cache rule")
	cache.Tags = []string{"redis"}
	cache.Body = "Cache TTLs must be explicit."

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{auth, cache}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	matches, err := store.SearchText("session-renewal refresh", 10)
	if err != nil {
		t.Fatalf("SearchText returned error: %v", err)
	}
	if len(matches) == 0 || matches[0].ItemID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule match, got %#v", matches)
	}
	if matches[0].Score <= 0 {
		t.Fatalf("expected positive lexical score, got %#v", matches[0])
	}
}

func TestRebuildIndexesPackageEntrypointChunks(t *testing.T) {
	root := t.TempDir()
	pkg := testItem("package:backend.auth-refresh.v1", "Auth refresh package")
	pkg.Type = "package"
	pkg.Path = "knowledge/packages/backend/auth-refresh/KNOWLEDGE.md"
	pkg.Body = "## Purpose\nRefresh auth flows.\n\n## When To Use\nUse for refresh token endpoints.\n\n## Start Here\nRead the rule first.\n\n## Load On Demand\nOpen examples only when needed.\n"

	dbPath := filepath.Join(root, "argos/index.db")
	if err := Rebuild(dbPath, []knowledge.Item{pkg}); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer store.Close()

	chunks, err := store.ListChunks("package:backend.auth-refresh.v1")
	if err != nil {
		t.Fatalf("ListChunks returned error: %v", err)
	}
	gotSections := map[string]bool{}
	for _, chunk := range chunks {
		gotSections[chunk.Section] = true
		if chunk.Text == "" {
			t.Fatalf("expected chunk text: %#v", chunk)
		}
	}
	for _, section := range []string{"Purpose", "When To Use", "Start Here", "Load On Demand"} {
		if !gotSections[section] {
			t.Fatalf("expected section %q in chunks: %#v", section, chunks)
		}
	}
}
```

- [ ] **Step 2: Run index tests to verify they fail**

Run:

```bash
go test ./internal/index -run 'TestRebuildStoresDiscoveryMetadata|TestSearchTextFindsTitleBodyAndTags|TestRebuildIndexesPackageEntrypointChunks' -count=1
```

Expected: FAIL because `DiscoveryCapabilities`, `SearchText`, `ListChunks`, and tag persistence do not exist yet.

- [ ] **Step 3: Add discovery index types**

In `internal/index/store.go`, add these exported types near `type Store`:

```go
type DiscoveryCapabilities struct {
	Metadata       string `json:"metadata"`
	FTS            string `json:"fts"`
	Semantic       string `json:"semantic"`
	SemanticReason string `json:"semantic_reason,omitempty"`
}

type TextMatch struct {
	ItemID  string
	Section string
	Score   float64
}

type Chunk struct {
	ChunkID       string
	ItemID        string
	Path          string
	Section       string
	HeadingPath   string
	Ordinal       int
	Text          string
	TokenEstimate int
}
```

- [ ] **Step 4: Replace the index schema with discovery-ready tables**

Update `createSchema` in `internal/index/store.go` so it creates tags, FTS, chunks, vector placeholder, and metadata:

```go
func (s *Store) createSchema(db execer) error {
	statements := []string{
		`CREATE TABLE knowledge_items (
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
	tags TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	summary TEXT NOT NULL,
	body TEXT NOT NULL
)`,
		`CREATE VIRTUAL TABLE knowledge_fts USING fts5(
	item_id UNINDEXED,
	section UNINDEXED,
	title,
	summary,
	body,
	tags,
	domains
)`,
		`CREATE TABLE knowledge_chunks (
	chunk_id TEXT PRIMARY KEY,
	item_id TEXT NOT NULL,
	path TEXT NOT NULL,
	section TEXT NOT NULL,
	heading_path TEXT NOT NULL,
	ordinal INTEGER NOT NULL,
	text TEXT NOT NULL,
	token_estimate INTEGER NOT NULL
)`,
		`CREATE TABLE knowledge_vectors (
	chunk_id TEXT NOT NULL,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	dimensions INTEGER NOT NULL,
	embedding BLOB NOT NULL,
	PRIMARY KEY (chunk_id, provider, model)
)`,
		`CREATE TABLE index_metadata (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
)`,
		`INSERT INTO index_metadata (key, value) VALUES
	('schema_version', '2'),
	('semantic_enabled', 'false'),
	('semantic_reason', 'semantic provider is not configured')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create index schema: %w", err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Persist tags and populate FTS/chunks during inserts**

Update `insertItem` so it serializes tags, writes `tags` into `knowledge_items`, inserts item-level FTS, and inserts chunks. The body of `insertItem` should include this structure:

```go
tags, err := marshalJSON(item.Tags)
if err != nil {
	return fmt.Errorf("%s: serialize tags: %w", item.ID, err)
}

_, err = db.Exec(`
INSERT INTO knowledge_items (
	id, path, title, type, tech_domains, business_domains, projects,
	status, priority, scope, tags, updated_at, summary, body
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
	tags,
	item.UpdatedAt,
	summary(item.Body),
	item.Body,
)
if err != nil {
	return fmt.Errorf("%s: insert knowledge item: %w", item.ID, err)
}

domainText := strings.Join(append(append([]string{}, item.TechDomains...), item.BusinessDomains...), " ")
if _, err := db.Exec(`
INSERT INTO knowledge_fts (item_id, section, title, summary, body, tags, domains)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
	item.ID,
	"",
	item.Title,
	summary(item.Body),
	item.Body,
	strings.Join(item.Tags, " "),
	domainText,
); err != nil {
	return fmt.Errorf("%s: insert item fts: %w", item.ID, err)
}

for _, chunk := range chunksForItem(item) {
	if _, err := db.Exec(`
INSERT INTO knowledge_chunks (chunk_id, item_id, path, section, heading_path, ordinal, text, token_estimate)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		chunk.ChunkID,
		chunk.ItemID,
		chunk.Path,
		chunk.Section,
		chunk.HeadingPath,
		chunk.Ordinal,
		chunk.Text,
		chunk.TokenEstimate,
	); err != nil {
		return fmt.Errorf("%s: insert chunk: %w", item.ID, err)
	}
	if _, err := db.Exec(`
INSERT INTO knowledge_fts (item_id, section, title, summary, body, tags, domains)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		chunk.Section,
		item.Title,
		summary(chunk.Text),
		chunk.Text,
		strings.Join(item.Tags, " "),
		domainText,
	); err != nil {
		return fmt.Errorf("%s: insert chunk fts: %w", item.ID, err)
	}
}
```

- [ ] **Step 6: Add chunk helpers**

Add these helpers near the bottom of `internal/index/store.go`:

```go
func chunksForItem(item knowledge.Item) []Chunk {
	sections := markdownSections(item.Body)
	if len(sections) == 0 {
		return []Chunk{{
			ChunkID:       item.ID + "#body",
			ItemID:        item.ID,
			Path:          item.Path,
			Section:       "",
			HeadingPath:   "",
			Ordinal:       0,
			Text:          strings.TrimSpace(item.Body),
			TokenEstimate: estimateTokens(item.Body),
		}}
	}
	chunks := make([]Chunk, 0, len(sections))
	for i, section := range sections {
		chunks = append(chunks, Chunk{
			ChunkID:       fmt.Sprintf("%s#%d", item.ID, i),
			ItemID:        item.ID,
			Path:          item.Path,
			Section:       section.heading,
			HeadingPath:   section.heading,
			Ordinal:       i,
			Text:          strings.TrimSpace(section.text),
			TokenEstimate: estimateTokens(section.text),
		})
	}
	return chunks
}

type markdownSection struct {
	heading string
	text    string
}

func markdownSections(body string) []markdownSection {
	var sections []markdownSection
	var current *markdownSection
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &markdownSection{heading: strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))}
			continue
		}
		if current != nil {
			current.text += line + "\n"
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}
	return sections
}

func estimateTokens(text string) int {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0
	}
	return len(fields)
}
```

- [ ] **Step 7: Update GetItem and ListItems to read tags**

In `GetItem` and `ListItems`, add `tags` to the SELECT list and scan target, then unmarshal it into `item.Tags`:

```go
var tags string
```

Use:

```go
if err := unmarshalJSON(tags, &item.Tags); err != nil {
	return knowledge.Item{}, fmt.Errorf("%s: deserialize tags: %w", id, err)
}
```

For `ListItems`, return `nil, fmt.Errorf("%s: deserialize tags: %w", item.ID, err)` on tag unmarshal failure.

- [ ] **Step 8: Add search, chunk, and capability methods**

Add these methods to `internal/index/store.go`:

```go
func (s *Store) SearchText(query string, limit int) ([]TextMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
SELECT item_id, section, bm25(knowledge_fts) AS rank
FROM knowledge_fts
WHERE knowledge_fts MATCH ?
ORDER BY rank
LIMIT ?`, ftsQuery(query), limit)
	if err != nil {
		return nil, fmt.Errorf("search text: %w", err)
	}
	defer rows.Close()

	var matches []TextMatch
	for rows.Next() {
		var match TextMatch
		var rank float64
		if err := rows.Scan(&match.ItemID, &match.Section, &rank); err != nil {
			return nil, fmt.Errorf("scan text match: %w", err)
		}
		match.Score = 1 / (1 + maxFloat(rank, 0))
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate text matches: %w", err)
	}
	return matches, nil
}

func (s *Store) ListChunks(itemID string) ([]Chunk, error) {
	rows, err := s.db.Query(`
SELECT chunk_id, item_id, path, section, heading_path, ordinal, text, token_estimate
FROM knowledge_chunks
WHERE item_id = ?
ORDER BY ordinal`, itemID)
	if err != nil {
		return nil, fmt.Errorf("list chunks for %s: %w", itemID, err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ChunkID, &chunk.ItemID, &chunk.Path, &chunk.Section, &chunk.HeadingPath, &chunk.Ordinal, &chunk.Text, &chunk.TokenEstimate); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}
	return chunks, nil
}

func (s *Store) DiscoveryCapabilities() (DiscoveryCapabilities, error) {
	if err := s.CheckSchema(); err != nil {
		return DiscoveryCapabilities{}, err
	}
	return DiscoveryCapabilities{
		Metadata:       "enabled",
		FTS:            "enabled",
		Semantic:       "disabled",
		SemanticReason: "semantic provider is not configured",
	}, nil
}

func ftsQuery(query string) string {
	var terms []string
	for _, term := range strings.Fields(query) {
		clean := strings.Trim(term, `"'():*`)
		if clean != "" {
			terms = append(terms, clean+"*")
		}
	}
	return strings.Join(terms, " OR ")
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
```

- [ ] **Step 9: Update CheckSchema**

Update `CheckSchema` so it verifies the new tables:

```go
for _, stmt := range []string{
	`SELECT 1 FROM knowledge_items LIMIT 0`,
	`SELECT 1 FROM knowledge_fts LIMIT 0`,
	`SELECT 1 FROM knowledge_chunks LIMIT 0`,
	`SELECT 1 FROM knowledge_vectors LIMIT 0`,
	`SELECT 1 FROM index_metadata LIMIT 0`,
} {
	if _, err := s.db.Exec(stmt); err != nil {
		return fmt.Errorf("check index schema: %w", err)
	}
}
```

- [ ] **Step 10: Run index tests**

Run:

```bash
go test ./internal/index -count=1
```

Expected: PASS.

- [ ] **Step 11: Commit the discovery index schema**

Run:

```bash
git add internal/index/store.go internal/index/store_test.go
git commit -m "feat: extend index for discovery"
```

## Task 2: Add Discovery Query Types And Core Ranking

**Files:**
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/query.go`

- [ ] **Step 1: Write failing discovery query tests**

Add these tests to `internal/query/query_test.go` near the existing standards tests:

```go
func TestDiscoverReturnsStrongMatchedRoutesWithoutFullBodies(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "strong" {
		t.Fatalf("expected strong coverage, got %#v", result.Coverage)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected discovery items")
	}
	first := result.Items[0]
	if first.ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule first, got %#v", result.Items)
	}
	if first.Body != "" {
		t.Fatalf("discover must not return full body: %#v", first)
	}
	if first.Disclosure.LoadTool != "get_knowledge_item" || first.Disclosure.Level != "summary" {
		t.Fatalf("unexpected disclosure: %#v", first.Disclosure)
	}
	if len(first.WhyMatched) == 0 {
		t.Fatalf("expected why_matched reasons")
	}
	if len(result.NextCalls) == 0 || result.NextCalls[0].Tool != "get_knowledge_item" {
		t.Fatalf("expected get_knowledge_item next call: %#v", result.NextCalls)
	}
}

func TestDiscoverReportsNoneCoverageForUnmatchedTask(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner warehouse",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no items, got %#v", result.Items)
	}
	for _, call := range result.NextCalls {
		if call.Tool == "cite_knowledge" {
			t.Fatalf("did not expect citation recommendation for no match: %#v", result.NextCalls)
		}
	}
}

func TestDiscoverFiltersTypesTagsAndDeprecated(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Query:   "auth",
		Types:   []string{"lesson"},
		Tags:    []string{"auth"},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "lesson:backend.auth-debug.v1" {
		t.Fatalf("expected auth lesson only, got %#v", result.Items)
	}
	for _, item := range result.Items {
		if item.Status == "deprecated" {
			t.Fatalf("deprecated item should be excluded by default: %#v", item)
		}
	}
}

func TestMapReturnsInventoryWithoutFullBodies(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Map(MapRequest{Project: "mall-api", Domain: "backend"})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	if result.Inventory.Types["rule"] == 0 || result.Inventory.Types["package"] == 0 {
		t.Fatalf("expected rule and package counts: %#v", result.Inventory.Types)
	}
	if len(result.Inventory.Packages) != 1 {
		t.Fatalf("expected package inventory, got %#v", result.Inventory.Packages)
	}
	for _, group := range result.Groups {
		for _, item := range group.Items {
			if item.Body != "" {
				t.Fatalf("map must not return full body: %#v", item)
			}
		}
	}
}
```

Add this helper near existing query test helpers:

```go
func buildDiscoveryTestStore(t *testing.T) *index.Store {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	items := []knowledge.Item{
		{
			Path:            "knowledge/items/backend/auth.md",
			ID:              "rule:backend.auth.v1",
			Title:           "Refresh token auth rule",
			Type:            "rule",
			TechDomains:     []string{"backend", "security"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"auth", "refresh-token"},
			Body:            "Refresh token endpoints must rotate tokens and require auth middleware.",
		},
		{
			Path:            "knowledge/items/backend/auth-debug.md",
			ID:              "lesson:backend.auth-debug.v1",
			Title:           "Auth debugging lesson",
			Type:            "lesson",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "should",
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"auth"},
			Body:            "When auth tests fail, inspect session renewal logs first.",
		},
		{
			Path:            "knowledge/packages/backend/auth-refresh/KNOWLEDGE.md",
			ID:              "package:backend.auth-refresh.v1",
			Title:           "Auth refresh package",
			Type:            "package",
			TechDomains:     []string{"backend", "security"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "should",
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"auth", "refresh-token"},
			Body:            "## Purpose\nRefresh token implementation guidance.\n\n## When To Use\nUse for refresh token endpoints.\n\n## Start Here\nLoad rules first.\n\n## Load On Demand\nOpen examples only when needed.\n",
		},
		{
			Path:            "knowledge/items/backend/old.md",
			ID:              "rule:backend.old-auth.v1",
			Title:           "Old auth rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "deprecated",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"auth"},
			Body:            "Deprecated auth guidance.",
		},
	}
	if err := index.Rebuild(dbPath, items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	return store
}
```

- [ ] **Step 2: Run query tests to verify they fail**

Run:

```bash
go test ./internal/query -run 'TestDiscover|TestMap' -count=1
```

Expected: FAIL because `Discover`, `Map`, and their types do not exist.

- [ ] **Step 3: Add discovery response types**

Add these types to `internal/query/query.go` after `ContextRequest`:

```go
type DiscoverRequest struct {
	Project           string   `json:"project"`
	Phase             string   `json:"phase"`
	Task              string   `json:"task"`
	Query             string   `json:"query"`
	Files             []string `json:"files"`
	Types             []string `json:"types"`
	Tags              []string `json:"tags"`
	Domains           []string `json:"domains"`
	Status            []string `json:"status"`
	IncludeInbox      bool     `json:"include_inbox"`
	IncludeDeprecated bool     `json:"include_deprecated"`
	Limit             int      `json:"limit"`
}

type MapRequest struct {
	Project           string   `json:"project"`
	Domain            string   `json:"domain"`
	Types             []string `json:"types"`
	IncludeInbox      bool     `json:"include_inbox"`
	IncludeDeprecated bool     `json:"include_deprecated"`
}

type DiscoveryResponse struct {
	Project      string                      `json:"project"`
	Phase        string                      `json:"phase"`
	Query        string                      `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Coverage     Coverage                    `json:"coverage"`
	Items        []DiscoveryItem             `json:"items"`
	NextCalls    []RecommendedCall           `json:"next_calls"`
}

type MapResponse struct {
	Project   string       `json:"project"`
	Inventory Inventory    `json:"inventory"`
	Groups    []MapGroup   `json:"groups"`
}

type Coverage struct {
	Status                string   `json:"status"`
	Confidence            float64  `json:"confidence"`
	Reason                string   `json:"reason"`
	Recommendation        string   `json:"recommendation"`
	MissingKnowledgeHints  []string `json:"missing_knowledge_hints,omitempty"`
}

type Inventory struct {
	Types    map[string]int   `json:"types"`
	Domains  []string         `json:"domains"`
	Tags     []string         `json:"tags"`
	Packages []DiscoveryItem  `json:"packages"`
}

type MapGroup struct {
	Key   string          `json:"key"`
	Title string          `json:"title"`
	Items []DiscoveryItem `json:"items"`
}

type DiscoveryItem struct {
	ID                string           `json:"id"`
	Type              string           `json:"type"`
	Title             string           `json:"title"`
	Summary           string           `json:"summary"`
	Status            string           `json:"status"`
	Priority          string           `json:"priority"`
	Path              string           `json:"path"`
	Score             float64          `json:"score"`
	ScoreComponents   ScoreComponents  `json:"score_components"`
	WhyMatched        []string         `json:"why_matched"`
	MatchedSections   []string         `json:"matched_sections"`
	Disclosure        Disclosure       `json:"disclosure"`
	RecommendedAction string           `json:"recommended_action"`
	Body              string           `json:"-"`
}

type ScoreComponents struct {
	Project   float64 `json:"project"`
	FileScope float64 `json:"file_scope"`
	TypePhase float64 `json:"type_phase"`
	Priority  float64 `json:"priority"`
	Status    float64 `json:"status"`
	TagDomain float64 `json:"tag_domain"`
	Lexical   float64 `json:"lexical"`
	Semantic  float64 `json:"semantic"`
}

type Disclosure struct {
	Level             string `json:"level"`
	FullBodyAvailable bool   `json:"full_body_available"`
	LoadTool          string `json:"load_tool"`
}
```

Update `RecommendedCall` to support IDs:

```go
type RecommendedCall struct {
	Tool   string   `json:"tool"`
	Reason string   `json:"reason"`
	IDs    []string `json:"ids,omitempty"`
}
```

- [ ] **Step 4: Implement Discover and Map**

Add `Discover` and `Map` methods to `internal/query/query.go`:

```go
func (s *Service) Discover(req DiscoverRequest) (DiscoveryResponse, error) {
	caps, err := s.store.DiscoveryCapabilities()
	if err != nil {
		return DiscoveryResponse{}, err
	}
	items, err := s.store.ListItems()
	if err != nil {
		return DiscoveryResponse{}, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	intent := strings.TrimSpace(strings.Join([]string{req.Task, req.Query}, " "))
	textMatches, err := s.store.SearchText(intent, 50)
	if err != nil {
		return DiscoveryResponse{}, err
	}
	lexical := lexicalScores(textMatches)
	sections := matchedSections(textMatches)

	var results []DiscoveryItem
	for _, item := range items {
		if !discoverCandidateAllowed(item, req) {
			continue
		}
		result := discoveryItem(item, req, lexical[item.ID], sections[item.ID])
		if result.Score <= 0.25 {
			continue
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if priorityRank(results[i].Priority) != priorityRank(results[j].Priority) {
			return priorityRank(results[i].Priority) < priorityRank(results[j].Priority)
		}
		return results[i].ID < results[j].ID
	})
	if len(results) > limit {
		results = results[:limit]
	}

	coverage := discoveryCoverage(results, intent)
	nextCalls := discoveryNextCalls(results, coverage, req.Phase)

	return DiscoveryResponse{
		Project:      req.Project,
		Phase:        req.Phase,
		Query:        strings.TrimSpace(strings.Join([]string{req.Task, req.Query}, " ")),
		Capabilities: caps,
		Coverage:     coverage,
		Items:        results,
		NextCalls:    nextCalls,
	}, nil
}

func (s *Service) Map(req MapRequest) (MapResponse, error) {
	items, err := s.store.ListItems()
	if err != nil {
		return MapResponse{}, err
	}
	inventory := Inventory{
		Types: map[string]int{},
	}
	grouped := map[string][]DiscoveryItem{}
	domainSet := map[string]bool{}
	tagSet := map[string]bool{}

	for _, item := range items {
		if !mapCandidateAllowed(item, req) {
			continue
		}
		inventory.Types[item.Type]++
		for _, domain := range append(append([]string{}, item.TechDomains...), item.BusinessDomains...) {
			domainSet[domain] = true
		}
		for _, tag := range item.Tags {
			tagSet[tag] = true
		}
		route := discoveryItemFromKnowledge(item)
		if item.Type == "package" {
			inventory.Packages = append(inventory.Packages, route)
		}
		key := mapGroupKey(item)
		grouped[key] = append(grouped[key], route)
	}

	inventory.Domains = sortedKeys(domainSet)
	inventory.Tags = sortedKeys(tagSet)
	sort.Slice(inventory.Packages, func(i, j int) bool { return inventory.Packages[i].ID < inventory.Packages[j].ID })

	var groups []MapGroup
	for key, groupItems := range grouped {
		sort.Slice(groupItems, func(i, j int) bool { return groupItems[i].ID < groupItems[j].ID })
		groups = append(groups, MapGroup{Key: key, Title: strings.Title(strings.ReplaceAll(key, "/", " ")), Items: groupItems})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })

	return MapResponse{Project: req.Project, Inventory: inventory, Groups: groups}, nil
}
```

- [ ] **Step 5: Add discovery scoring helpers**

Add these helpers below existing helpers in `internal/query/query.go`:

```go
func discoveryItem(item knowledge.Item, req DiscoverRequest, lexical float64, sections []string) DiscoveryItem {
	components := ScoreComponents{
		Project:   boolScore(contains(item.Projects, req.Project)),
		FileScope: fileScopeScore(item, req.Files),
		TypePhase: typePhaseScore(item.Type, req.Phase),
		Priority:  priorityScore(item.Priority),
		Status:    statusScore(item.Status),
		TagDomain: tagDomainScore(item, req.Tags, req.Domains),
		Lexical:   lexical,
		Semantic:  0,
	}
	score := weightedScore(components)
	result := discoveryItemFromKnowledge(item)
	result.Score = score
	result.ScoreComponents = components
	result.WhyMatched = whyMatched(item, req, components)
	result.MatchedSections = sections
	result.RecommendedAction = recommendedAction(item, score, req.Phase)
	return result
}

func discoveryItemFromKnowledge(item knowledge.Item) DiscoveryItem {
	return DiscoveryItem{
		ID:       item.ID,
		Type:     item.Type,
		Title:    item.Title,
		Summary:  firstSentence(item.Body),
		Status:   item.Status,
		Priority: item.Priority,
		Path:     item.Path,
		Disclosure: Disclosure{
			Level:             "summary",
			FullBodyAvailable: true,
			LoadTool:          "get_knowledge_item",
		},
		RecommendedAction: "skim_summary_only",
	}
}

func weightedScore(c ScoreComponents) float64 {
	total := c.Project*0.18 + c.FileScope*0.18 + c.TypePhase*0.14 + c.Priority*0.12 + c.Status*0.08 + c.TagDomain*0.12 + c.Lexical*0.18
	if total > 1 {
		return 1
	}
	return total
}

func boolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func discoverCandidateAllowed(item knowledge.Item, req DiscoverRequest) bool {
	if item.Status == "deprecated" && !req.IncludeDeprecated {
		return false
	}
	if req.Project != "" && !contains(item.Projects, req.Project) {
		return false
	}
	if len(req.Types) > 0 && !contains(req.Types, item.Type) {
		return false
	}
	if len(req.Status) > 0 && !contains(req.Status, item.Status) {
		return false
	}
	for _, tag := range req.Tags {
		if !contains(item.Tags, tag) {
			return false
		}
	}
	for _, domain := range req.Domains {
		if !contains(item.TechDomains, domain) && !contains(item.BusinessDomains, domain) {
			return false
		}
	}
	return true
}

func mapCandidateAllowed(item knowledge.Item, req MapRequest) bool {
	if item.Status == "deprecated" && !req.IncludeDeprecated {
		return false
	}
	if req.Project != "" && !contains(item.Projects, req.Project) {
		return false
	}
	if req.Domain != "" && !contains(item.TechDomains, req.Domain) && !contains(item.BusinessDomains, req.Domain) {
		return false
	}
	if len(req.Types) > 0 && !contains(req.Types, item.Type) {
		return false
	}
	return true
}

func fileScopeScore(item knowledge.Item, files []string) float64 {
	if len(item.AppliesTo.Files) == 0 {
		return 0.4
	}
	for _, file := range files {
		for _, pattern := range item.AppliesTo.Files {
			matched, err := doublestar.PathMatch(pattern, file)
			if err == nil && matched {
				return 1
			}
		}
	}
	return 0
}

func typePhaseScore(itemType string, phase string) float64 {
	preferences := map[string][]string{
		"planning":       {"decision", "guide", "package", "reference"},
		"implementation": {"rule", "package", "runbook", "decision"},
		"review":         {"rule", "decision", "lesson"},
		"debugging":      {"lesson", "runbook", "decision"},
		"operations":     {"runbook", "decision", "rule"},
		"deployment":     {"runbook", "decision", "rule"},
	}
	for i, preferred := range preferences[phase] {
		if preferred == itemType {
			return 1 - float64(i)*0.15
		}
	}
	if phase == "" {
		return 0.5
	}
	return 0.2
}

func priorityScore(priority string) float64 {
	switch priority {
	case "must":
		return 1
	case "should":
		return 0.75
	case "may":
		return 0.45
	default:
		return 0.25
	}
}

func statusScore(status string) float64 {
	switch status {
	case "active":
		return 1
	case "draft":
		return 0.65
	default:
		return 0
	}
}

func tagDomainScore(item knowledge.Item, tags []string, domains []string) float64 {
	if len(tags) == 0 && len(domains) == 0 {
		return 0.3
	}
	matches := 0
	total := len(tags) + len(domains)
	for _, tag := range tags {
		if contains(item.Tags, tag) {
			matches++
		}
	}
	for _, domain := range domains {
		if contains(item.TechDomains, domain) || contains(item.BusinessDomains, domain) {
			matches++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(matches) / float64(total)
}

func lexicalScores(matches []index.TextMatch) map[string]float64 {
	scores := map[string]float64{}
	for _, match := range matches {
		if match.Score > scores[match.ItemID] {
			scores[match.ItemID] = match.Score
		}
	}
	return scores
}

func matchedSections(matches []index.TextMatch) map[string][]string {
	seen := map[string]map[string]bool{}
	for _, match := range matches {
		if match.Section == "" {
			continue
		}
		if seen[match.ItemID] == nil {
			seen[match.ItemID] = map[string]bool{}
		}
		seen[match.ItemID][match.Section] = true
	}
	result := map[string][]string{}
	for id, sections := range seen {
		result[id] = sortedKeys(sections)
	}
	return result
}

func whyMatched(item knowledge.Item, req DiscoverRequest, c ScoreComponents) []string {
	var reasons []string
	if c.Project > 0 {
		reasons = append(reasons, fmt.Sprintf("project %s matched", req.Project))
	}
	if c.FileScope >= 1 {
		reasons = append(reasons, "file scope matched applies_to.files")
	}
	if c.TypePhase >= 0.7 {
		reasons = append(reasons, fmt.Sprintf("%s phase prefers %s knowledge", req.Phase, item.Type))
	}
	if c.TagDomain > 0.3 {
		reasons = append(reasons, "requested tags or domains matched")
	}
	if c.Lexical > 0 {
		reasons = append(reasons, "task or query text matched indexed knowledge")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "general project knowledge matched")
	}
	return reasons
}

func recommendedAction(item knowledge.Item, score float64, phase string) string {
	if score < 0.45 {
		return "skim_summary_only"
	}
	switch phase {
	case "implementation":
		if item.Priority == "must" || item.Type == "package" {
			return "load_full_before_implementation"
		}
	case "review":
		if item.Priority == "must" || item.Type == "decision" {
			return "load_full_before_review"
		}
	case "debugging":
		if item.Type == "lesson" || item.Type == "runbook" {
			return "load_full_before_debugging"
		}
	case "planning":
		if item.Type == "decision" || item.Type == "package" {
			return "load_full_before_planning"
		}
	}
	return "cite_if_used"
}

func discoveryCoverage(items []DiscoveryItem, intent string) Coverage {
	if len(items) == 0 {
		return Coverage{
			Status:         "none",
			Confidence:     0,
			Reason:         "No active Argos knowledge matched this request strongly.",
			Recommendation: "Proceed without Argos-specific claims and do not cite Argos knowledge for this task.",
			MissingKnowledgeHints: missingKnowledgeHints(intent),
		}
	}
	top := items[0].Score
	switch {
	case top >= 0.75:
		return Coverage{Status: "strong", Confidence: top, Reason: "Found active project knowledge matching this request.", Recommendation: "Load high-priority matched knowledge before work."}
	case top >= 0.5:
		return Coverage{Status: "partial", Confidence: top, Reason: "Found related Argos knowledge, but task-specific coverage has gaps.", Recommendation: "Load only high-confidence IDs and mention gaps when relevant.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	default:
		return Coverage{Status: "weak", Confidence: top, Reason: "Only broad or low-confidence Argos knowledge matched.", Recommendation: "Skim summaries or inspect the map; do not treat results as authoritative.", MissingKnowledgeHints: missingKnowledgeHints(intent)}
	}
}

func discoveryNextCalls(items []DiscoveryItem, coverage Coverage, phase string) []RecommendedCall {
	if coverage.Status == "none" || coverage.Status == "weak" {
		return []RecommendedCall{{Tool: "argos_map", Reason: "Inspect available project knowledge if the task scope changes."}}
	}
	var ids []string
	for _, item := range items {
		if strings.HasPrefix(item.RecommendedAction, "load_full") {
			ids = append(ids, item.ID)
		}
	}
	var calls []RecommendedCall
	if len(ids) > 0 {
		calls = append(calls, RecommendedCall{Tool: "get_knowledge_item", Reason: "Load selected routed knowledge before applying it.", IDs: ids})
	}
	calls = append(calls, RecommendedCall{Tool: "cite_knowledge", Reason: "Cite Argos knowledge IDs actually used in the final response."})
	return calls
}

func missingKnowledgeHints(intent string) []string {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return nil
	}
	return []string{intent + " standard", intent + " decision", intent + " lesson"}
}

func mapGroupKey(item knowledge.Item) string {
	if len(item.TechDomains) > 0 {
		if len(item.Tags) > 0 {
			return item.TechDomains[0] + "/" + item.Tags[0]
		}
		return item.TechDomains[0]
	}
	if len(item.BusinessDomains) > 0 {
		return item.BusinessDomains[0]
	}
	return item.Type
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
```

- [ ] **Step 6: Run query tests**

Run:

```bash
go test ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit query discovery**

Run:

```bash
git add internal/query/query.go internal/query/query_test.go
git commit -m "feat: add discovery query service"
```

## Task 3: Expose Discovery Through CLI JSON

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Write failing CLI tests**

Add these tests to `internal/cli/cli_test.go` near the context/index tests:

```go
func TestRunDiscoverReturnsJSONRoutes(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"discover", "--json", "--project", "mall-api", "--phase", "implementation", "--task", "add refresh token endpoint", "--query", "refresh token", "--files", "internal/auth/session.go"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		Coverage struct {
			Status string `json:"status"`
		} `json:"coverage"`
		Items []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if result.Coverage.Status != "strong" {
		t.Fatalf("expected strong coverage: %s", stdout.String())
	}
	if len(result.Items) == 0 || result.Items[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Refresh token endpoints must rotate tokens") {
		t.Fatalf("discover should not print full body: %s", stdout.String())
	}
}

func TestRunMapReturnsJSONInventory(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)
	if code := Run([]string{"index"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("index failed with code %d", code)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"map", "--json", "--project", "mall-api", "--domain", "backend"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"packages"`) || !strings.Contains(stdout.String(), `"rule"`) {
		t.Fatalf("expected inventory JSON: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Refresh token endpoints must rotate tokens") {
		t.Fatalf("map should not print full body: %s", stdout.String())
	}
}

func TestRunDiscoverRequiresIndex(t *testing.T) {
	root := t.TempDir()
	writeCLIDiscoveryWorkspace(t, root)
	chdir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"discover", "--json", "--project", "mall-api", "--query", "auth"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "index not available: run argos index first") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
```

Add helper:

```go
func writeCLIDiscoveryWorkspace(t *testing.T, root string) {
	t.Helper()
	writeCLIRegistry(t, root)
	writeCLIFile(t, root, "knowledge/items/backend/auth.md", `---
id: rule:backend.auth.v1
title: Refresh token auth rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  files: ["internal/auth/**"]
updated_at: 2026-04-29
tags: [auth, refresh-token]
---
Refresh token endpoints must rotate tokens and require auth middleware.
`)
	writeCLIFile(t, root, "knowledge/packages/backend/auth-refresh/KNOWLEDGE.md", validCLIPackage("package:backend.auth-refresh.v1"))
}
```

- [ ] **Step 2: Run CLI tests to verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunDiscover|TestRunMap' -count=1
```

Expected: FAIL because `discover` and `map` commands do not exist.

- [ ] **Step 3: Add CLI command cases**

In `internal/cli/cli.go`, add `discover` and `map` cases before `mcp`:

```go
case "discover":
	flags := flag.NewFlagSet("discover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	phase := flags.String("phase", "", "workflow phase")
	task := flags.String("task", "", "task description")
	queryText := flags.String("query", "", "search query")
	files := multiValueFlag{}
	flags.Var(&files, "files", "file relevant to the current task")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "discover: --json is required")
		return 2
	}
	store, closeStore, ok := openIndexStore(stderr)
	if !ok {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).Discover(query.DiscoverRequest{
		Project: *project,
		Phase:   *phase,
		Task:    *task,
		Query:   *queryText,
		Files:   files,
	})
	if err != nil {
		fmt.Fprintf(stderr, "discover: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
case "map":
	flags := flag.NewFlagSet("map", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	domain := flags.String("domain", "", "domain filter")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "map: --json is required")
		return 2
	}
	store, closeStore, ok := openIndexStore(stderr)
	if !ok {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).Map(query.MapRequest{Project: *project, Domain: *domain})
	if err != nil {
		fmt.Fprintf(stderr, "map: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
```

- [ ] **Step 4: Add CLI helpers**

Add these helpers to `internal/cli/cli.go`:

```go
type multiValueFlag []string

func (m *multiValueFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValueFlag) Set(value string) error {
	if strings.TrimSpace(value) != "" {
		*m = append(*m, strings.TrimSpace(value))
	}
	return nil
}

func openIndexStore(stderr io.Writer) (*index.Store, func(), bool) {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get current directory: %v\n", err)
		return nil, func() {}, false
	}
	store, err := index.Open(filepath.Join(root, "argos", "index.db"))
	if err != nil {
		fmt.Fprintln(stderr, "index not available: run argos index first")
		return nil, func() {}, false
	}
	if err := store.CheckSchema(); err != nil {
		_ = store.Close()
		fmt.Fprintln(stderr, "index not available: run argos index first")
		return nil, func() {}, false
	}
	return store, func() { _ = store.Close() }, true
}

func printJSON(stdout io.Writer, stderr io.Writer, value any) int {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "marshal JSON response: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(body))
	return 0
}
```

- [ ] **Step 5: Run CLI tests**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit CLI discovery commands**

Run:

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: expose discovery in cli"
```

## Task 4: Expose Discovery Through MCP

**Files:**
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/mcp/server.go`

- [ ] **Step 1: Write failing MCP tests**

Update `TestServerHandlesToolsList` and `TestToolsListIncludesConcreteSchemasForImplementedTools` to include `argos_discover` and `argos_map`.

Add these tests to `internal/mcp/server_test.go`:

```go
func TestToolCallArgosDiscoverReturnsRoutes(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","phase":"implementation","task":"add refresh token endpoint","query":"auth","files":["internal/auth/session.go"]}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
	if !strings.Contains(text, `"coverage"`) || !strings.Contains(text, `"items"`) {
		t.Fatalf("expected discovery JSON: %s", text)
	}
	if strings.Contains(text, `"body"`) {
		t.Fatalf("discover should not expose full body: %s", text)
	}
}

func TestToolCallArgosMapReturnsInventory(t *testing.T) {
	store := buildMCPTestStore(t)
	defer store.Close()
	server := NewServerWithStore(store)

	var out bytes.Buffer
	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_map","arguments":{"project":"mall-api","domain":"backend"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	text := firstContentText(t, resultMap(t, decodeRPCResponse(t, out.Bytes())))
	if !strings.Contains(text, `"inventory"`) {
		t.Fatalf("expected map JSON: %s", text)
	}
	if strings.Contains(text, `"body"`) {
		t.Fatalf("map should not expose full body: %s", text)
	}
}

func TestToolCallArgosDiscoverWithoutIndexReturnsToolError(t *testing.T) {
	server := NewServer(query.New(nil))
	var out bytes.Buffer

	err := server.HandleLine([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"argos_discover","arguments":{"project":"mall-api","query":"auth"}}}`), &out)
	if err != nil {
		t.Fatalf("HandleLine returned error: %v", err)
	}

	assertToolErrorContains(t, out.Bytes(), "index not available: run argos index first")
}
```

- [ ] **Step 2: Run MCP tests to verify they fail**

Run:

```bash
go test ./internal/mcp -run 'TestServerHandlesToolsList|TestToolsListIncludesConcreteSchemasForImplementedTools|TestToolCallArgosDiscover|TestToolCallArgosMap' -count=1
```

Expected: FAIL because MCP tools do not exist.

- [ ] **Step 3: Add MCP tool call cases**

In `internal/mcp/server.go`, add cases to `callTool` before `get_knowledge_item`:

```go
case "argos_discover":
	var req query.DiscoverRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_discover: " + err.Error()), nil, nil
	}
	if err := requireStringFields(map[string]string{"project": req.Project}, "project"); err != nil {
		return textToolError("invalid arguments for argos_discover: " + err.Error()), nil, nil
	}
	if strings.TrimSpace(req.Task) == "" && strings.TrimSpace(req.Query) == "" {
		return textToolError("invalid arguments for argos_discover: task or query is required"), nil, nil
	}
	if req.Limit < 0 || req.Limit > 20 {
		return textToolError("invalid arguments for argos_discover: limit must be between 0 and 20"), nil, nil
	}
	if s.store == nil {
		return textToolError("index not available: run argos index first"), nil, nil
	}
	resp, err := s.service.Discover(req)
	if err != nil {
		return textToolError("discover: " + err.Error()), nil, nil
	}
	result, err := textResult(resp)
	return result, nil, err
case "argos_map":
	var req query.MapRequest
	if err := decodeArgs(params.Arguments, &req); err != nil {
		return textToolError("invalid arguments for argos_map: " + err.Error()), nil, nil
	}
	if err := requireStringFields(map[string]string{"project": req.Project}, "project"); err != nil {
		return textToolError("invalid arguments for argos_map: " + err.Error()), nil, nil
	}
	if s.store == nil {
		return textToolError("index not available: run argos index first"), nil, nil
	}
	resp, err := s.service.Map(req)
	if err != nil {
		return textToolError("map: " + err.Error()), nil, nil
	}
	result, err := textResult(resp)
	return result, nil, err
```

- [ ] **Step 4: Add MCP schemas**

Add tool definitions to `tools()`:

```go
{
	Name:        "argos_discover",
	Description: "Discover relevant Argos knowledge routes for a task without loading full bodies.",
	InputSchema: objectSchema(map[string]any{
		"project":            stringProperty("Project identifier."),
		"phase":              stringProperty("Workflow phase."),
		"task":               stringProperty("Current task description."),
		"query":              stringProperty("Search query."),
		"files":              stringArrayProperty("Files relevant to the current task."),
		"types":              stringArrayProperty("Knowledge types to include."),
		"tags":               stringArrayProperty("Tags to require."),
		"domains":            stringArrayProperty("Tech or business domains to include."),
		"status":             stringArrayProperty("Statuses to include."),
		"include_inbox":      booleanProperty("Include inbox candidates."),
		"include_deprecated": booleanProperty("Include deprecated knowledge."),
		"limit":              integerProperty("Maximum number of routes to return.", 1, 20),
	}, []string{"project"}),
},
{
	Name:        "argos_map",
	Description: "Return a lightweight inventory of Argos knowledge without full bodies.",
	InputSchema: objectSchema(map[string]any{
		"project":            stringProperty("Project identifier."),
		"domain":             stringProperty("Domain filter."),
		"types":              stringArrayProperty("Knowledge types to include."),
		"include_inbox":      booleanProperty("Include inbox candidates."),
		"include_deprecated": booleanProperty("Include deprecated knowledge."),
	}, []string{"project"}),
},
```

Add:

```go
func booleanProperty(description string) map[string]any {
	return map[string]any{
		"type":        "boolean",
		"description": description,
	}
}
```

- [ ] **Step 5: Run MCP tests**

Run:

```bash
go test ./internal/mcp -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit MCP discovery tools**

Run:

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat: expose discovery over mcp"
```

## Task 5: Update Context Recommendations And Adapter Contract

**Files:**
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/query.go`
- Modify: `internal/adapters/adapters_test.go`
- Modify: `internal/adapters/adapters.go`

- [ ] **Step 1: Update context recommendation tests**

In `TestContextRecommendationsOnlyUseCallableTools`, add:

```go
"argos_discover": true,
"argos_map":      true,
```

Add a test:

```go
func TestContextRecommendsDiscoveryForBroadWork(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "understand auth refresh token flow",
	})

	var tools []string
	for _, call := range result.RecommendedNextCalls {
		tools = append(tools, call.Tool)
	}
	if !contains(tools, "argos_map") || !contains(tools, "argos_discover") {
		t.Fatalf("expected map and discover recommendations, got %#v", result.RecommendedNextCalls)
	}
}
```

- [ ] **Step 2: Implement context recommendations**

Update `Context` in `internal/query/query.go` so it returns:

```go
calls := []RecommendedCall{
	{Tool: "argos_discover", Reason: "discover task-relevant Argos knowledge without loading full bodies"},
	{Tool: "argos_standards", Reason: reason},
}
if req.Phase == "planning" || strings.Contains(strings.ToLower(req.Task), "understand") {
	calls = append([]RecommendedCall{{Tool: "argos_map", Reason: "inventory available project knowledge before broad work"}}, calls...)
}
```

Return `calls` as `RecommendedNextCalls`.

- [ ] **Step 3: Update adapter contract tests**

In `internal/adapters/adapters_test.go`, add these expected strings to `TestRenderedAdaptersIncludeStableKnowledgeContract`:

```go
"Use argos_discover to route current work to relevant Argos knowledge.",
"Use argos_map for broad orientation before unfamiliar project work.",
"Load full knowledge items only through get_knowledge_item when routed to specific IDs or paths.",
```

Update the old expectation:

```go
"Load full knowledge items only when routed to specific IDs or paths.",
```

to the new explicit `get_knowledge_item` wording.

- [ ] **Step 4: Update adapter rendering**

In `internal/adapters/adapters.go`, update `renderMarkdown` Work Protocol:

```md
1. Before substantial project work, call or emulate argos_context.
2. Use argos_map for broad orientation before unfamiliar project work.
3. Use argos_discover to route current work to relevant Argos knowledge.
4. Before implementation or review, call or emulate argos_standards.
5. Follow recommended calls returned by Argos.
6. Load full knowledge items only through get_knowledge_item when routed to specific IDs or paths.
7. Cite Argos knowledge IDs used in final responses.
```

- [ ] **Step 5: Run query and adapter tests**

Run:

```bash
go test ./internal/query ./internal/adapters -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit context and adapter updates**

Run:

```bash
git add internal/query/query.go internal/query/query_test.go internal/adapters/adapters.go internal/adapters/adapters_test.go
git commit -m "feat: route agents through discovery"
```

## Task 6: Document Discovery V1

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add README discovery section**

After the Agent Experience section, add:

```md
## Discovery

Discovery is the agent-facing knowledge navigation layer. It helps agents
inventory available knowledge, route current work to relevant knowledge, and
avoid Argos-backed claims when no strong match exists.

The default discovery path is local and lightweight: SQLite metadata, file
scope matching, tags, package entrypoints, and FTS5 full-text search. Embedding
and vector search are optional future enhancements; `argos discover` works
without Ollama, model downloads, GPU hardware, or a background service.

Use:

```bash
argos discover --json --project <project> --phase <phase> --task "<task>" --query "<query>"
argos map --json --project <project> --domain <domain>
```

`argos map` returns inventory and orientation. `argos discover` returns ranked
routes, `why_matched`, `coverage`, and `next_calls`. Neither command returns
full Markdown bodies. Load selected full items with `get_knowledge_item` and
cite used IDs with `cite_knowledge`.

Coverage states:

- `strong`: load recommended high-priority IDs before work.
- `partial`: useful knowledge exists, but gaps remain.
- `weak`: skim summaries or inspect the map; do not treat results as authority.
- `none`: proceed without Argos-specific claims and do not cite Argos knowledge.
```

- [ ] **Step 2: Update MCP tool list**

In README MCP section, add:

```md
- `argos_discover`: returns ranked knowledge routes, coverage, explanations,
  and next calls without full bodies. Arguments: `project`, `phase`, `task`,
  `query`, `files`, `types`, `tags`, `domains`, `status`, `include_inbox`,
  `include_deprecated`, `limit`.
- `argos_map`: returns project/domain knowledge inventory without full bodies.
  Arguments: `project`, `domain`, `types`, `include_inbox`,
  `include_deprecated`.
```

- [ ] **Step 3: Update command list**

Add:

```bash
argos discover --json --project <project> --task <task>
argos map --json --project <project>
```

- [ ] **Step 4: Verify README text**

Run:

```bash
rg -n "## Discovery|argos discover|argos map|coverage|without Ollama|argos_discover|argos_map" README.md
```

Expected: all new Discovery v1 references are found.

- [ ] **Step 5: Commit docs**

Run:

```bash
git add README.md
git commit -m "docs: document discovery workflow"
```

## Task 7: Full Verification And Plan Closure

**Files:**
- Verify all changed files.

- [ ] **Step 1: Format Go files**

Run:

```bash
gofmt -w internal/index/store.go internal/index/store_test.go internal/query/query.go internal/query/query_test.go internal/cli/cli.go internal/cli/cli_test.go internal/mcp/server.go internal/mcp/server_test.go internal/adapters/adapters.go internal/adapters/adapters_test.go
```

Expected: command exits 0.

- [ ] **Step 2: Run focused package tests**

Run:

```bash
go test ./internal/index ./internal/query ./internal/cli ./internal/mcp ./internal/adapters -count=1
```

Expected: PASS.

- [ ] **Step 3: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Run vet**

Run:

```bash
go vet ./...
```

Expected: exit 0.

- [ ] **Step 5: Check diff hygiene**

Run:

```bash
git diff --check
```

Expected: exit 0.

- [ ] **Step 6: Verify no unimplemented tools are recommended**

Run:

```bash
rg -n "argos_requirements|argos_risks|argos_operations" internal README.md docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md
```

Expected: only historical design/spec references may appear; generated adapters, query recommendations, README Discovery docs, and MCP tool list must not advertise these as callable tools.

- [ ] **Step 7: Inspect final status**

Run:

```bash
git status --short
```

Expected: no uncommitted implementation changes except plan checkbox updates if the executing agent is tracking this plan.

## Self-Review

Spec coverage:

- Inventory: Task 2 adds `Map`; Task 3 and Task 4 expose `argos map` and `argos_map`.
- Task routing: Task 2 adds `Discover`; Task 3 and Task 4 expose CLI/MCP.
- Progressive disclosure: Task 2 query tests, Task 3 CLI tests, Task 4 MCP tests, and Task 5 adapter contract all require summary-only discovery and explicit `get_knowledge_item` loading.
- Absence control: Task 2 covers `coverage.status = none`; Task 6 documents coverage states.
- Lightweight core: Task 1 uses SQLite metadata, FTS, chunks, and semantic-disabled metadata; Task 6 documents no Ollama/model/GPU requirement.
- Optional semantic readiness: Task 1 adds semantic capability metadata and vector placeholder table; implementation does not require a provider.
- Governed retrieval: Task 5 updates `argos_context` and adapters to route through implemented tools only.
- Package-aware discovery: Task 1 chunks package entrypoints; Task 2 map/discover tests include package items.

Known intentional deferrals:

- No real embedding provider in this implementation plan.
- No package asset indexing.
- No graph database or web UI.
- No direct agent vector database access.
