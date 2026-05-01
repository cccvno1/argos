package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

func TestActiveSurfacesDoNotUseRetiredSharedKnowledgeNames(t *testing.T) {
	root := repoRootForActiveSurfaceTest(t)
	activeRoots := []string{
		"README.md",
		"internal",
		"testdata",
		"docs/superpowers/templates",
		"docs/superpowers/checklists",
		"docs/superpowers/specs/2026-04-30-argos-discovery-layer-design.md",
		"docs/superpowers/specs/2026-04-30-argos-discovery-validation-harness-design.md",
		"docs/superpowers/specs/2026-04-30-argos-shared-knowledge-discovery-semantics-design.md",
	}

	for _, rel := range activeRoots {
		if file, term, ok := firstRetiredNameMatch(t, root, rel, retiredSharedKnowledgeNames(), activeSurfaceGuardFile(t)); ok {
			t.Fatalf("active surface %s contains retired term %q", file, term)
		}
	}
}

func retiredSharedKnowledgeNames() []string {
	parts := [][]string{
		{"argos", "_", "map"},
		{"argos", "_", "discover"},
		{"get", "_", "knowledge", "_", "item"},
		{"coverage", "_", "gaps"},
		{"Coverage", "Gap"},
		{"action", "_", "policy"},
		{"Action", "Policy"},
		{"author", "ity"},
		{"rec", "all"},
		{"Recall", "State"},
		{"dis", "closure"},
		{"Dis", "closure"},
		{"next", "_", "calls"},
		{"Recommended", "Call"},
	}
	retired := make([]string, 0, len(parts))
	for _, term := range parts {
		retired = append(retired, strings.Join(term, ""))
	}
	return retired
}

func TestFirstRetiredNameMatchSkipsGuardFileAndReportsNestedPath(t *testing.T) {
	root := t.TempDir()
	guardFile := filepath.Join(root, "internal", "a_guard.go")
	if err := os.MkdirAll(filepath.Dir(guardFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(guardFile, []byte("usage"), 0o644); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(root, "internal", "nested", "fixture.go")
	if err := os.MkdirAll(filepath.Dir(nestedFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nestedFile, []byte("missing_needs"), 0o644); err != nil {
		t.Fatal(err)
	}

	file, term, ok := firstRetiredNameMatch(t, root, "internal", []string{"usage", "missing_needs"}, guardFile)
	if !ok {
		t.Fatal("expected retired-name match")
	}
	if file != filepath.ToSlash(filepath.Join("internal", "nested", "fixture.go")) {
		t.Fatalf("expected nested match path, got %q", file)
	}
	if term != "missing_needs" {
		t.Fatalf("expected nested retired term, got %q", term)
	}
}

func TestFirstRetiredNameMatchSkipsDirectExcludedFile(t *testing.T) {
	root := t.TempDir()
	guardFile := filepath.Join(root, "internal", "query", "query_test.go")
	if err := os.MkdirAll(filepath.Dir(guardFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(guardFile, []byte("usage"), 0o644); err != nil {
		t.Fatal(err)
	}

	file, term, ok := firstRetiredNameMatch(t, root, filepath.Join("internal", "query", "query_test.go"), []string{"usage"}, guardFile)
	if ok {
		t.Fatalf("expected excluded direct file to have no match, got %s %q", file, term)
	}
}

func repoRootForActiveSurfaceTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func firstRetiredNameMatch(t *testing.T, root, rel string, retired []string, excludeFiles ...string) (string, string, bool) {
	t.Helper()
	path := filepath.Join(root, rel)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat active surface %s: %v", path, err)
	}
	if !info.IsDir() {
		if isExcludedActiveSurfaceFile(path, excludeFiles) {
			return "", "", false
		}
		return retiredNameMatchInFile(t, root, path, retired)
	}
	var matchFile string
	var matchTerm string
	err = filepath.WalkDir(path, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || isExcludedActiveSurfaceFile(file, excludeFiles) || !isActiveSurfaceFile(file) {
			return nil
		}
		if relFile, term, ok := retiredNameMatchInFile(t, root, file, retired); ok {
			matchFile = relFile
			matchTerm = term
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk active surface %s: %v", path, err)
	}
	return matchFile, matchTerm, matchFile != ""
}

func isExcludedActiveSurfaceFile(file string, excludeFiles []string) bool {
	for _, exclude := range excludeFiles {
		if file == filepath.Clean(exclude) {
			return true
		}
	}
	return false
}

func activeSurfaceGuardFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard file path")
	}
	return filepath.Clean(file)
}

func isActiveSurfaceFile(file string) bool {
	return strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".go") || strings.HasSuffix(file, ".json")
}

func retiredNameMatchInFile(t *testing.T, root, file string, retired []string) (string, string, bool) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read active surface %s: %v", file, err)
	}
	body := string(data)
	for _, term := range retired {
		if strings.Contains(body, term) {
			rel, err := filepath.Rel(root, file)
			if err != nil {
				t.Fatalf("resolve active surface path %s: %v", file, err)
			}
			return filepath.ToSlash(rel), term, true
		}
	}
	return "", "", false
}

func TestStandardsReturnsActiveRulesWithMatchReason(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].WhyMatched == "" {
		t.Fatalf("expected match reason")
	}
}

func TestStandardsIncludesDraftRulesAndExcludesDeprecatedRules(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/draft.md",
			ID:              "rule:backend.draft.v1",
			Title:           "Draft rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "draft",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Draft guidance still applies.",
		},
		{
			Path:            "knowledge/items/backend/deprecated.md",
			ID:              "rule:backend.deprecated.v1",
			Title:           "Deprecated rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "deprecated",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Deprecated guidance should not apply.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "rule:backend.draft.v1" {
		t.Fatalf("expected draft rule, got %q", result.Items[0].ID)
	}
}

func TestStandardsRanksResultsByPriorityBeforeApplyingLimit(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		ruleWithPriority("rule:priority.may.v1", "may"),
		ruleWithPriority("rule:priority.must.v1", "must"),
		ruleWithPriority("rule:priority.should.v1", "should"),
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	got := []string{result.Items[0].ID, result.Items[1].ID}
	want := []string{"rule:priority.must.v1", "rule:priority.should.v1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected items %v, got %v", want, got)
		}
	}
}

func TestStandardsPrefersFileScopedMatchBeforeApplyingLimit(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/generic.md",
			ID:              "rule:priority.generic.v1",
			Title:           "Generic must rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			UpdatedAt:       "2026-04-29",
			Body:            "Generic guidance applies.",
		},
		{
			Path:            "knowledge/items/backend/specific.md",
			ID:              "rule:priority.specific.v1",
			Title:           "Specific must rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
			UpdatedAt:       "2026-04-29",
			Body:            "File-specific guidance applies.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := New(store)
	result, err := service.Standards(StandardsRequest{
		Project: "mall-api",
		Files:   []string{"internal/auth/session.go"},
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("Standards returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ID != "rule:priority.specific.v1" {
		t.Fatalf("expected file-scoped rule, got %q", result.Items[0].ID)
	}
}

func TestFindKnowledgeReturnsStrongMatchedResultsWithoutFullBodies(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "strong" {
		t.Fatalf("expected strong coverage, got %#v", result.Support)
	}
	if len(result.Items) == 0 {
		t.Fatal("expected knowledge summaries")
	}
	first := result.Items[0]
	if first.ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule first, got %#v", result.Items)
	}
	if first.Body != "" {
		t.Fatalf("find must not return full body: %#v", first)
	}
	if first.ReadStatus.ReadTool != "argos_read_knowledge" || first.ReadStatus.Level != "summary" {
		t.Fatalf("unexpected read status: %#v", first.ReadStatus)
	}
	if len(first.WhyMatched) == 0 {
		t.Fatalf("expected why_matched reasons")
	}
	if len(result.NextSteps) == 0 || result.NextSteps[0].Tool != "argos_read_knowledge" {
		t.Fatalf("expected argos_read_knowledge next step: %#v", result.NextSteps)
	}
}

func TestFindKnowledgeResponsePreservesMetadata(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Project != "mall-api" || result.Phase != "implementation" || result.Query != "add refresh token endpoint refresh token" {
		t.Fatalf("expected request metadata on response, got %#v", result)
	}
	if result.Capabilities.Metadata != "enabled" || result.Capabilities.FTS != "enabled" || result.Capabilities.Semantic != "disabled" {
		t.Fatalf("expected discovery capabilities on response, got %#v", result.Capabilities)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal find response: %v", err)
	}
	for _, field := range []string{`"project"`, `"phase"`, `"query"`, `"capabilities"`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("expected JSON field %s in %s", field, data)
		}
	}
}

func TestFindKnowledgePublicWordingUsesSupportAndReadTerms(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	strong, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if strings.Contains(strong.Support.Recommendation, "Load") || strings.Contains(strong.Support.Recommendation, "load") {
		t.Fatalf("support recommendation should use read terminology: %q", strong.Support.Recommendation)
	}
	if strong.Items[0].RecommendedStep != "read_full_before_implementation" {
		t.Fatalf("expected read recommended step, got %#v", strong.Items[0])
	}
	if len(strong.NextSteps) == 0 || strong.NextSteps[0].Reason != "Read selected knowledge before applying it." {
		t.Fatalf("expected read next step reason, got %#v", strong.NextSteps)
	}
	if strong.Usage.Cite != "after_read_and_used" {
		t.Fatalf("expected read-based cite guidance, got %#v", strong.Usage)
	}

	partial, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if !strings.Contains(partial.Support.Reason, "support") || !strings.Contains(partial.Support.Recommendation, "missing needs") {
		t.Fatalf("expected support/missing-needs wording, got %#v", partial.Support)
	}
	for _, value := range []string{partial.Support.Reason, partial.Support.Recommendation} {
		if strings.Contains(value, "coverage") || strings.Contains(value, "gaps") || strings.Contains(value, "Load") {
			t.Fatalf("support wording should avoid old terminology, got %q", value)
		}
	}
}

func TestFindKnowledgeReportsNoneSupportForUnmatchedTask(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner warehouse",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no items, got %#v", result.Items)
	}
	for _, call := range result.NextSteps {
		if call.Tool == "argos_cite_knowledge" {
			t.Fatalf("did not expect citation recommendation for no match: %#v", result.NextSteps)
		}
	}
}

func TestFindKnowledgeUsageFollowsSupport(t *testing.T) {
	tests := []struct {
		name      string
		req       FindKnowledgeRequest
		level     string
		wantUsage UsageGuidance
	}{
		{
			name: "strong",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add refresh token endpoint",
				Query:   "refresh token",
				Files:   []string{"internal/auth/session.go"},
				Limit:   5,
			},
			level: "strong",
			wantUsage: UsageGuidance{
				Read:  "recommended",
				Cite:  "after_read_and_used",
				Claim: "allowed",
			},
		},
		{
			name: "partial",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "debugging",
				Task:    "debug session renewal test failure",
				Query:   "session renewal tests fail logs",
				Limit:   5,
			},
			level: "partial",
			wantUsage: UsageGuidance{
				Read:  "allowed",
				Cite:  "after_read_and_used",
				Claim: "must_separate_argos_backed_and_general_reasoning",
			},
		},
		{
			name: "weak",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add warehouse barcode scanner",
				Query:   "barcode scanner token",
				Limit:   5,
			},
			level: "weak",
			wantUsage: UsageGuidance{
				Read:  "forbidden",
				Cite:  "forbidden",
				Claim: "forbidden",
			},
		},
		{
			name: "none",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add payment webhook signature verification",
				Query:   "payment webhook signature",
				Limit:   5,
			},
			level: "none",
			wantUsage: UsageGuidance{
				Read:  "forbidden",
				Cite:  "forbidden",
				Claim: "forbidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := buildFindKnowledgeTestStore(t)
			defer store.Close()
			service := New(store)

			result, err := service.FindKnowledge(tt.req)
			if err != nil {
				t.Fatalf("FindKnowledge returned error: %v", err)
			}
			if result.Support.Level != tt.level {
				t.Fatalf("expected support level %q, got %#v", tt.level, result.Support)
			}
			assertUsageGuidance(t, result.Usage, tt.wantUsage)
		})
	}
}

func TestFindKnowledgeSearchStatusDefaultsSemanticDisabled(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.SearchStatus.Semantic.Status != "disabled" {
		t.Fatalf("expected semantic search disabled, got %#v", result.SearchStatus)
	}
	if result.SearchStatus.Semantic.Reason == "" {
		t.Fatalf("expected semantic search reason, got %#v", result.SearchStatus)
	}
}

func TestFindKnowledgeMissingNeedsForNoneSupport(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"not_found"})
	for _, gap := range result.MissingNeeds {
		if gap.Need == "" {
			t.Fatalf("expected missing need need, got %#v", gap)
		}
		if gap.Reason == "" {
			t.Fatalf("expected missing need reason, got %#v", gap)
		}
		if gap.Severity != "blocking" {
			t.Fatalf("expected blocking severity for none coverage, got %#v", gap)
		}
		if gap.ArgosBacked {
			t.Fatalf("missing needs must not be Argos-backed: %#v", gap)
		}
	}
}

func TestFindKnowledgeMissingNeedsReportRestrictiveFilterExclusion(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Tags:    []string{"payments"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"filtered_out"})
}

func TestFindKnowledgeMissingNeedsDoNotReportFilterExclusionForOnlyWeakFilteredKnowledge(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/generic-token.md",
		ID:              "rule:backend.generic-token.v1",
		Title:           "Generic token rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Tags:            []string{"auth"},
		Body:            "Token guidance applies to platform work.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Tags:    []string{"payments"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"not_found"})
}

func TestFindKnowledgeMissingNeedsReportFilterExclusionForPartialSupport(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{
		{
			Path:            "knowledge/items/backend/session-debug.md",
			ID:              "lesson:backend.session-debug.v1",
			Title:           "Session debugging lesson",
			Type:            "lesson",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "should",
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"debugging"},
			Body:            "When session renewal tests fail, inspect token rotation logs first.",
		},
		{
			Path:            "knowledge/items/backend/session-rule.md",
			ID:              "rule:backend.session-rule.v1",
			Title:           "Session renewal rule",
			Type:            "rule",
			TechDomains:     []string{"backend"},
			BusinessDomains: []string{"account"},
			Projects:        []string{"mall-api"},
			Status:          "active",
			Priority:        "must",
			AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
			UpdatedAt:       "2026-04-29",
			Tags:            []string{"auth"},
			Body:            "Session renewal tests require token rotation logs and auth middleware checks.",
		},
	})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Files:   []string{"internal/auth/session.go"},
		Tags:    []string{"debugging"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "partial" {
		t.Fatalf("expected partial coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"filtered_out"})
}

func TestFindKnowledgeMissingNeedsIncludeDeprecatedIsNotRestrictiveFilter(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project:           "mall-api",
		Phase:             "implementation",
		Task:              "add payment webhook signature verification",
		Query:             "payment webhook signature",
		IncludeDeprecated: true,
		Limit:             5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"not_found"})
}

func TestFindKnowledgeMissingNeedsReportCrossDomainMismatch(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/warehouse/auth.md",
		ID:              "rule:warehouse.auth.v1",
		Title:           "Warehouse auth rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"order"},
		Projects:        []string{"warehouse-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Tags:            []string{"warehouse"},
		Body:            "Warehouse-only auth guidance must not match Mall API tasks.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "update warehouse picking flow",
		Query:   "warehouse picking scanner",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"wrong_scope"})
}

func TestFindKnowledgeWeakMissingNeedsAreNotArgosBacked(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/generic-token.md",
		ID:              "rule:backend.generic-token.v1",
		Title:           "Generic token rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Token guidance applies to platform work.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "weak" {
		t.Fatalf("expected weak coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"weak_match"})
	for _, gap := range result.MissingNeeds {
		if gap.ArgosBacked {
			t.Fatalf("missing needs must not be Argos-backed: %#v", gap)
		}
	}
}

func TestFindKnowledgePartialSupportUsesAttributionClaim(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "partial" {
		t.Fatalf("expected partial coverage, got %#v", result.Support)
	}
	assertMissingNeedSources(t, result.MissingNeeds, []string{"partial_match"})
	if result.Usage.Claim != "must_separate_argos_backed_and_general_reasoning" {
		t.Fatalf("expected attribution claim policy, got %#v", result.Usage)
	}
}

func TestFindKnowledgeStrongSupportOmitsMissingNeeds(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level != "strong" {
		t.Fatalf("expected strong coverage, got %#v", result.Support)
	}
	if len(result.MissingNeeds) != 0 {
		t.Fatalf("strong coverage should not produce missing needs: %#v", result.MissingNeeds)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal find response: %v", err)
	}
	if strings.Contains(string(data), `"missing_needs"`) {
		t.Fatalf("strong response should omit missing_needs from JSON: %s", data)
	}
}

func TestFindKnowledgeJSONIncludesNonEmptyMissingNeedsAndNoLegacyGaps(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal discovery response: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"missing_needs"`) {
		t.Fatalf("expected missing_needs in JSON: %s", body)
	}
	if strings.Contains(body, "gap_"+"candidates") {
		t.Fatalf("did not expect legacy gap key in JSON: %s", body)
	}
	if strings.Contains(body, "capture_"+"candidate") || strings.Contains(body, "candidate_"+"only") || strings.Contains(body, "proposal_"+"required") {
		t.Fatalf("did not expect capture-oriented gap semantics in JSON: %s", body)
	}
}

func TestListKnowledgeUsageForbidsLoadCitationAndClaims(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.ListKnowledge(ListKnowledgeRequest{Project: "mall-api"})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	assertUsageGuidance(t, result.Usage, UsageGuidance{
		Read:  "forbidden",
		Cite:  "forbidden",
		Claim: "forbidden",
	})
}

func assertUsageGuidance(t *testing.T, got UsageGuidance, want UsageGuidance) {
	t.Helper()
	if got.Read != want.Read || got.Cite != want.Cite || got.Claim != want.Claim {
		t.Fatalf("expected usage guidance %#v, got %#v", want, got)
	}
	if got.Reason == "" {
		t.Fatalf("expected usage guidance reason: %#v", got)
	}
}

func assertMissingNeedSources(t *testing.T, got []MissingNeed, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected missing need sources %v, got %#v", want, got)
	}
	seen := map[string]bool{}
	for _, gap := range got {
		seen[gap.Source] = true
		if gap.Need == "" {
			t.Fatalf("expected need for %#v", gap)
		}
		if gap.Reason == "" {
			t.Fatalf("expected reason for %#v", gap)
		}
		if gap.Severity == "" {
			t.Fatalf("expected severity for %#v", gap)
		}
		if gap.ArgosBacked {
			t.Fatalf("missing need must not be Argos-backed: %#v", gap)
		}
	}
	for _, source := range want {
		if !seen[source] {
			t.Fatalf("expected missing need source %q in %#v", source, got)
		}
	}
}

func TestFindKnowledgeFiltersTypesTagsAndDeprecated(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Query:   "auth",
		Types:   []string{"lesson"},
		Tags:    []string{"auth"},
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
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

func TestFindKnowledgeTagsFilterMatchesAnyRequestedTag(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Query:   "auth",
		Tags:    []string{"missing", "auth"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if !containsKnowledgeSummary(result.Items, "rule:backend.auth.v1") {
		t.Fatalf("expected auth item with one matching requested tag, got %#v", result.Items)
	}
}

func TestFindKnowledgeDomainsFilterMatchesAnyRequestedDomain(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Query:   "auth",
		Domains: []string{"missing", "security"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if !containsKnowledgeSummary(result.Items, "rule:backend.auth.v1") {
		t.Fatalf("expected auth item with one matching requested domain, got %#v", result.Items)
	}
}

func TestFindKnowledgeExplicitTagsAndDomainsMustMatchAtLeastOneValuePerField(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	for _, tc := range []struct {
		name string
		req  FindKnowledgeRequest
	}{
		{
			name: "tags",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Query:   "auth",
				Tags:    []string{"missing"},
				Limit:   5,
			},
		},
		{
			name: "domains",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Query:   "auth",
				Domains: []string{"missing"},
				Limit:   5,
			},
		},
		{
			name: "tags and domains",
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Query:   "auth",
				Tags:    []string{"auth"},
				Domains: []string{"missing"},
				Limit:   5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.FindKnowledge(tc.req)
			if err != nil {
				t.Fatalf("FindKnowledge returned error: %v", err)
			}
			if len(result.Items) != 0 {
				t.Fatalf("expected no items for non-matching explicit filter, got %#v", result.Items)
			}
			if result.Support.Level == "strong" {
				t.Fatalf("expected non-strong coverage for non-matching explicit filter, got %#v", result.Support)
			}
		})
	}
}

func TestFindKnowledgeWeakSingleTermGenericLexicalMatchDoesNotProduceStrongSupportOrLoadCalls(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/generic-token.md",
		ID:              "rule:backend.generic-token.v1",
		Title:           "Generic token rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Token guidance applies to platform work.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if result.Support.Level == "strong" {
		t.Fatalf("single weak generic term must not produce strong coverage: %#v", result.Support)
	}
	for _, call := range result.NextSteps {
		if call.Tool == "argos_read_knowledge" {
			t.Fatalf("weak generic lexical match must not recommend loading full bodies: %#v", result.NextSteps)
		}
	}
}

func TestFindKnowledgeTypeAndPhaseFiltersAloneDoNotReturnUnrelatedKnowledge(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner warehouse",
		Types:   []string{"rule"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected type/phase filters alone not to prove relevance, got %#v", result.Items)
	}
	if result.Support.Level != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Support)
	}
}

func TestFindKnowledgeWeakAndNoneNextStepsOnlyUseImplementedTools(t *testing.T) {
	implementedTools := map[string]bool{
		"argos_read_knowledge": true,
		"argos_cite_knowledge": true,
	}

	for _, tc := range []struct {
		name  string
		items []knowledge.Item
		req   FindKnowledgeRequest
	}{
		{
			name: "none",
			items: []knowledge.Item{{
				Path:            "knowledge/items/backend/auth.md",
				ID:              "rule:backend.auth.v1",
				Title:           "Auth rule",
				Type:            "rule",
				TechDomains:     []string{"backend"},
				BusinessDomains: []string{"account"},
				Projects:        []string{"mall-api"},
				Status:          "active",
				Priority:        "must",
				UpdatedAt:       "2026-04-29",
				Body:            "Require explicit auth middleware.",
			}},
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add warehouse barcode scanner",
				Query:   "barcode scanner warehouse",
				Limit:   5,
			},
		},
		{
			name: "weak",
			items: []knowledge.Item{{
				Path:            "knowledge/items/backend/generic-token.md",
				ID:              "rule:backend.generic-token.v1",
				Title:           "Generic token rule",
				Type:            "rule",
				TechDomains:     []string{"backend"},
				BusinessDomains: []string{"account"},
				Projects:        []string{"mall-api"},
				Status:          "active",
				Priority:        "must",
				UpdatedAt:       "2026-04-29",
				Body:            "Token guidance applies to platform work.",
			}},
			req: FindKnowledgeRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add warehouse barcode scanner",
				Query:   "barcode scanner token",
				Limit:   5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := buildFindKnowledgeStore(t, tc.items)
			defer store.Close()
			service := New(store)

			result, err := service.FindKnowledge(tc.req)
			if err != nil {
				t.Fatalf("FindKnowledge returned error: %v", err)
			}
			if result.Support.Level != tc.name {
				t.Fatalf("expected %s coverage, got %#v", tc.name, result.Support)
			}
			for _, call := range result.NextSteps {
				if !implementedTools[call.Tool] {
					t.Fatalf("expected implemented tool only, got %#v", result.NextSteps)
				}
			}
		})
	}
}

func TestFindKnowledgeAndListKnowledgeTreatEmptyProjectsAsGlobalKnowledge(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/global-refresh.md",
		ID:              "rule:backend.global-refresh.v1",
		Title:           "Global refresh token rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Status:          "active",
		Priority:        "must",
		UpdatedAt:       "2026-04-29",
		Tags:            []string{"refresh-token"},
		Body:            "Refresh token endpoints must rotate tokens.",
	}})
	defer store.Close()
	service := New(store)

	discovered, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Query:   "refresh token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	if len(discovered.Items) != 1 || discovered.Items[0].ID != "rule:backend.global-refresh.v1" {
		t.Fatalf("expected global knowledge summary, got %#v", discovered.Items)
	}

	mapped, err := service.ListKnowledge(ListKnowledgeRequest{Project: "mall-api", Domain: "backend"})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	if mapped.Inventory.Types["rule"] != 1 {
		t.Fatalf("expected global map item, got %#v", mapped.Inventory)
	}
}

func TestFindKnowledgeReturnsErrorForInvalidFileScopeGlob(t *testing.T) {
	store := buildFindKnowledgeStore(t, []knowledge.Item{{
		Path:            "knowledge/items/backend/bad-glob.md",
		ID:              "rule:backend.bad-glob.v1",
		Title:           "Bad glob rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"["}},
		UpdatedAt:       "2026-04-29",
		Body:            "Refresh token guidance.",
	}})
	defer store.Close()
	service := New(store)

	_, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Query:   "refresh token",
		Files:   []string{"internal/auth/session.go"},
		Limit:   5,
	})
	if err == nil {
		t.Fatal("expected invalid glob error")
	}
}

func TestListKnowledgeReturnsInventoryWithoutFullBodies(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.ListKnowledge(ListKnowledgeRequest{Project: "mall-api", Domain: "backend"})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
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
				t.Fatalf("list must not return full body: %#v", item)
			}
		}
	}
}

func TestListKnowledgeResponsePreservesProject(t *testing.T) {
	store := buildFindKnowledgeTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.ListKnowledge(ListKnowledgeRequest{Project: "mall-api", Domain: "backend"})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	if result.Project != "mall-api" {
		t.Fatalf("expected project on response, got %#v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal list response: %v", err)
	}
	if !strings.Contains(string(data), `"project"`) {
		t.Fatalf("expected project JSON field in %s", data)
	}
}

func TestContextRecommendsNextSteps(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "add refresh token endpoint",
		Files:   []string{"internal/auth/session.go"},
	})

	if result.Project != "mall-api" {
		t.Fatalf("unexpected project: %s", result.Project)
	}
	if len(result.RecommendedNextSteps) == 0 {
		t.Fatal("expected recommended next steps")
	}
}

func TestContextRecommendationsOnlyUseCallableTools(t *testing.T) {
	service := New(nil)
	callableTools := map[string]bool{
		"argos_find_knowledge": true,
		"argos_list_knowledge": true,
		"argos_standards":      true,
		"argos_read_knowledge": true,
		"argos_cite_knowledge": true,
	}
	unavailableTools := map[string]bool{
		"argos_requirements": true,
		"argos_risks":        true,
		"argos_operations":   true,
	}

	for _, phase := range []string{
		"planning",
		"",
		"implementation",
		"review",
		"debugging",
		"operations",
		"deployment",
	} {
		t.Run(phase, func(t *testing.T) {
			result := service.Context(ContextRequest{
				Project: "mall-api",
				Phase:   phase,
				Task:    "add refresh token endpoint",
			})

			if len(result.RecommendedNextSteps) == 0 {
				t.Fatal("expected recommended next steps")
			}
			for _, call := range result.RecommendedNextSteps {
				if !callableTools[call.Tool] {
					t.Fatalf("expected callable recommendation, got %q", call.Tool)
				}
				if unavailableTools[call.Tool] {
					t.Fatalf("did not expect unavailable recommendation %q", call.Tool)
				}
			}
		})
	}
}

func TestContextRecommendsFindKnowledgeForBroadWork(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "understand auth refresh token flow",
	})

	var tools []string
	for _, call := range result.RecommendedNextSteps {
		tools = append(tools, call.Tool)
	}
	if !contains(tools, "argos_list_knowledge") || !contains(tools, "argos_find_knowledge") || !contains(tools, "argos_standards") {
		t.Fatalf("expected list, find, and standards recommendations, got %#v", result.RecommendedNextSteps)
	}
}

func TestContextDoesNotRecommendListKnowledgeForNarrowImplementationWork(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "fix typo",
	})

	var tools []string
	for _, call := range result.RecommendedNextSteps {
		tools = append(tools, call.Tool)
	}
	if !contains(tools, "argos_find_knowledge") || !contains(tools, "argos_standards") {
		t.Fatalf("expected find and standards recommendations, got %#v", result.RecommendedNextSteps)
	}
	if contains(tools, "argos_list_knowledge") {
		t.Fatalf("did not expect list recommendation for narrow implementation work: %#v", result.RecommendedNextSteps)
	}
}

func TestReadKnowledgeReturnsFullBody(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	item, err := service.ReadKnowledge("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("ReadKnowledge returned error: %v", err)
	}
	if item.Body != "Require explicit auth middleware for account endpoints.\nThis is the full rule body." {
		t.Fatalf("expected full body, got %q", item.Body)
	}
}

func TestCiteKnowledgeReportsMissingIDs(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	result := service.CiteKnowledge([]string{"rule:backend.auth.v1", "missing.v1"})
	if len(result.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(result.Citations))
	}
	if result.Citations[0].ID != "rule:backend.auth.v1" {
		t.Fatalf("expected auth rule citation, got %#v", result.Citations[0])
	}
	if len(result.Missing) != 1 || result.Missing[0] != "missing.v1" {
		t.Fatalf("expected missing id, got %#v", result.Missing)
	}
}

func containsKnowledgeSummary(items []KnowledgeSummary, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func ruleWithPriority(id string, priority string) knowledge.Item {
	return knowledge.Item{
		Path:            "knowledge/items/backend/priority.md",
		ID:              id,
		Title:           priority + " priority rule",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        priority,
		UpdatedAt:       "2026-04-29",
		Body:            "Priority guidance applies.",
	}
}

func buildQueryTestStore(t *testing.T) *index.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "argos/index.db")
	err := index.Rebuild(dbPath, []knowledge.Item{{
		Path:            "knowledge/items/backend/auth.md",
		ID:              "rule:backend.auth.v1",
		Title:           "Auth middleware",
		Type:            "rule",
		TechDomains:     []string{"backend"},
		BusinessDomains: []string{"account"},
		Projects:        []string{"mall-api"},
		Status:          "active",
		Priority:        "must",
		AppliesTo:       knowledge.Scope{Files: []string{"internal/auth/**"}},
		UpdatedAt:       "2026-04-29",
		Body:            "Require explicit auth middleware for account endpoints.\nThis is the full rule body.",
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func buildFindKnowledgeTestStore(t *testing.T) *index.Store {
	t.Helper()

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
	return buildFindKnowledgeStore(t, items)
}

func buildFindKnowledgeStore(t *testing.T, items []knowledge.Item) *index.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "argos/index.db")
	if err := index.Rebuild(dbPath, items); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}
	store, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	return store
}
