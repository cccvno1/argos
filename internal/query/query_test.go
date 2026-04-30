package query

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"argos/internal/index"
	"argos/internal/knowledge"
)

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

func TestDiscoverActionPolicyFollowsCoverage(t *testing.T) {
	tests := []struct {
		name string
		req  DiscoverRequest
		want ActionPolicy
	}{
		{
			name: "strong",
			req: DiscoverRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add refresh token endpoint",
				Query:   "refresh token",
				Files:   []string{"internal/auth/session.go"},
				Limit:   5,
			},
			want: ActionPolicy{
				Authority: "strong",
				Load:      "recommended",
				Cite:      "after_loaded_and_used",
				Claim:     "allowed",
			},
		},
		{
			name: "partial",
			req: DiscoverRequest{
				Project: "mall-api",
				Phase:   "debugging",
				Task:    "debug session renewal test failure",
				Query:   "session renewal tests fail logs",
				Limit:   5,
			},
			want: ActionPolicy{
				Authority: "partial",
				Load:      "allowed",
				Cite:      "after_loaded_and_used",
				Claim:     "must_separate_argos_backed_and_general_reasoning",
			},
		},
		{
			name: "weak",
			req: DiscoverRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add warehouse barcode scanner",
				Query:   "barcode scanner token",
				Limit:   5,
			},
			want: ActionPolicy{
				Authority: "weak",
				Load:      "forbidden",
				Cite:      "forbidden",
				Claim:     "forbidden",
			},
		},
		{
			name: "none",
			req: DiscoverRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add payment webhook signature verification",
				Query:   "payment webhook signature",
				Limit:   5,
			},
			want: ActionPolicy{
				Authority: "none",
				Load:      "forbidden",
				Cite:      "forbidden",
				Claim:     "forbidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := buildDiscoveryTestStore(t)
			defer store.Close()
			service := New(store)

			result, err := service.Discover(tt.req)
			if err != nil {
				t.Fatalf("Discover returned error: %v", err)
			}
			assertActionPolicy(t, result.ActionPolicy, tt.want)
			if result.ActionPolicy.Authority != result.Coverage.Status {
				t.Fatalf("expected policy authority to mirror coverage status, got policy=%#v coverage=%#v", result.ActionPolicy, result.Coverage)
			}
		})
	}
}

func TestDiscoverRecallStateDefaultsSemanticDisabled(t *testing.T) {
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
	if result.Recall.Semantic.Status != "disabled" {
		t.Fatalf("expected semantic recall disabled, got %#v", result.Recall)
	}
	if result.Recall.Semantic.Reason == "" {
		t.Fatalf("expected semantic recall reason, got %#v", result.Recall)
	}
}

func TestDiscoverCoverageGapsForNoneCoverage(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"unmatched_intent"})
	for _, gap := range result.CoverageGaps {
		if gap.Need == "" {
			t.Fatalf("expected coverage gap need, got %#v", gap)
		}
		if gap.Reason == "" {
			t.Fatalf("expected coverage gap reason, got %#v", gap)
		}
		if gap.Severity != "blocking" {
			t.Fatalf("expected blocking severity for none coverage, got %#v", gap)
		}
		if gap.ArgosBacked {
			t.Fatalf("coverage gaps must not be Argos-backed: %#v", gap)
		}
	}
}

func TestDiscoverCoverageGapsReportRestrictiveFilterExclusion(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Query:   "refresh token",
		Tags:    []string{"payments"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"filter_excluded"})
}

func TestDiscoverCoverageGapsDoNotReportFilterExclusionForOnlyWeakFilteredKnowledge(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Tags:    []string{"payments"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"unmatched_intent"})
}

func TestDiscoverCoverageGapsReportFilterExclusionForPartialCoverage(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{
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

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Files:   []string{"internal/auth/session.go"},
		Tags:    []string{"debugging"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "partial" {
		t.Fatalf("expected partial coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"filter_excluded"})
}

func TestDiscoverCoverageGapsIncludeDeprecatedIsNotRestrictiveFilter(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project:           "mall-api",
		Phase:             "implementation",
		Task:              "add payment webhook signature verification",
		Query:             "payment webhook signature",
		IncludeDeprecated: true,
		Limit:             5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"unmatched_intent"})
}

func TestDiscoverCoverageGapsReportCrossDomainMismatch(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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
		Body:            "Warehouse-only auth guidance must not route to Mall API tasks.",
	}})
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "update warehouse picking flow",
		Query:   "warehouse picking scanner",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"cross_domain_mismatch"})
}

func TestDiscoverWeakCoverageGapsAreNotArgosBacked(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "weak" {
		t.Fatalf("expected weak coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"weak_match"})
	for _, gap := range result.CoverageGaps {
		if gap.ArgosBacked {
			t.Fatalf("coverage gaps must not be Argos-backed: %#v", gap)
		}
	}
}

func TestDiscoverPartialCoverageUsesAttributionClaim(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "debugging",
		Task:    "debug session renewal test failure",
		Query:   "session renewal tests fail logs",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status != "partial" {
		t.Fatalf("expected partial coverage, got %#v", result.Coverage)
	}
	assertCoverageGapSources(t, result.CoverageGaps, []string{"partial_match"})
	if result.ActionPolicy.Claim != "must_separate_argos_backed_and_general_reasoning" {
		t.Fatalf("expected attribution claim policy, got %#v", result.ActionPolicy)
	}
}

func TestDiscoverStrongCoverageOmitsCoverageGaps(t *testing.T) {
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
	if len(result.CoverageGaps) != 0 {
		t.Fatalf("strong coverage should not produce coverage gaps: %#v", result.CoverageGaps)
	}
}

func TestDiscoverJSONDoesNotExposeLegacyGaps(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add payment webhook signature verification",
		Query:   "payment webhook signature",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal discovery response: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"coverage_gaps"`) {
		t.Fatalf("expected coverage_gaps in JSON: %s", body)
	}
	if strings.Contains(body, "gap_"+"candidates") {
		t.Fatalf("did not expect legacy gap key in JSON: %s", body)
	}
	if strings.Contains(body, "capture_"+"candidate") || strings.Contains(body, "candidate_"+"only") || strings.Contains(body, "proposal_"+"required") {
		t.Fatalf("did not expect capture-oriented gap semantics in JSON: %s", body)
	}
}

func TestMapActionPolicyForbidsLoadCitationAndClaims(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Map(MapRequest{Project: "mall-api"})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	assertActionPolicy(t, result.ActionPolicy, ActionPolicy{
		Authority: "inventory",
		Load:      "forbidden",
		Cite:      "forbidden",
		Claim:     "forbidden",
	})
}

func assertActionPolicy(t *testing.T, got ActionPolicy, want ActionPolicy) {
	t.Helper()
	if got.Authority != want.Authority || got.Load != want.Load || got.Cite != want.Cite || got.Claim != want.Claim {
		t.Fatalf("expected action policy %#v, got %#v", want, got)
	}
	if got.Reason == "" {
		t.Fatalf("expected action policy reason: %#v", got)
	}
}

func assertCoverageGapSources(t *testing.T, got []CoverageGap, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected coverage gap sources %v, got %#v", want, got)
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
			t.Fatalf("coverage gap must not be Argos-backed: %#v", gap)
		}
	}
	for _, source := range want {
		if !seen[source] {
			t.Fatalf("expected coverage gap source %q in %#v", source, got)
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

func TestDiscoverTagsFilterMatchesAnyRequestedTag(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Query:   "auth",
		Tags:    []string{"missing", "auth"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if !containsDiscoveryItem(result.Items, "rule:backend.auth.v1") {
		t.Fatalf("expected auth item with one matching requested tag, got %#v", result.Items)
	}
}

func TestDiscoverDomainsFilterMatchesAnyRequestedDomain(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Query:   "auth",
		Domains: []string{"missing", "security"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if !containsDiscoveryItem(result.Items, "rule:backend.auth.v1") {
		t.Fatalf("expected auth item with one matching requested domain, got %#v", result.Items)
	}
}

func TestDiscoverExplicitTagsAndDomainsMustMatchAtLeastOneValuePerField(t *testing.T) {
	store := buildDiscoveryTestStore(t)
	defer store.Close()
	service := New(store)

	for _, tc := range []struct {
		name string
		req  DiscoverRequest
	}{
		{
			name: "tags",
			req: DiscoverRequest{
				Project: "mall-api",
				Query:   "auth",
				Tags:    []string{"missing"},
				Limit:   5,
			},
		},
		{
			name: "domains",
			req: DiscoverRequest{
				Project: "mall-api",
				Query:   "auth",
				Domains: []string{"missing"},
				Limit:   5,
			},
		},
		{
			name: "tags and domains",
			req: DiscoverRequest{
				Project: "mall-api",
				Query:   "auth",
				Tags:    []string{"auth"},
				Domains: []string{"missing"},
				Limit:   5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.Discover(tc.req)
			if err != nil {
				t.Fatalf("Discover returned error: %v", err)
			}
			if len(result.Items) != 0 {
				t.Fatalf("expected no items for non-matching explicit filter, got %#v", result.Items)
			}
			if result.Coverage.Status == "strong" {
				t.Fatalf("expected non-strong coverage for non-matching explicit filter, got %#v", result.Coverage)
			}
		})
	}
}

func TestDiscoverWeakSingleTermGenericLexicalMatchDoesNotProduceStrongCoverageOrLoadCalls(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if result.Coverage.Status == "strong" {
		t.Fatalf("single weak generic term must not produce strong coverage: %#v", result.Coverage)
	}
	for _, call := range result.NextCalls {
		if call.Tool == "get_knowledge_item" {
			t.Fatalf("weak generic lexical match must not recommend loading full bodies: %#v", result.NextCalls)
		}
	}
}

func TestDiscoverTypeAndPhaseFiltersAloneDoNotReturnUnrelatedKnowledge(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	result, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add warehouse barcode scanner",
		Query:   "barcode scanner warehouse",
		Types:   []string{"rule"},
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected type/phase filters alone not to prove relevance, got %#v", result.Items)
	}
	if result.Coverage.Status != "none" {
		t.Fatalf("expected none coverage, got %#v", result.Coverage)
	}
}

func TestDiscoverWeakAndNoneNextCallsOnlyUseImplementedTools(t *testing.T) {
	implementedTools := map[string]bool{
		"get_knowledge_item": true,
		"cite_knowledge":     true,
	}

	for _, tc := range []struct {
		name  string
		items []knowledge.Item
		req   DiscoverRequest
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
			req: DiscoverRequest{
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
			req: DiscoverRequest{
				Project: "mall-api",
				Phase:   "implementation",
				Task:    "add warehouse barcode scanner",
				Query:   "barcode scanner token",
				Limit:   5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := buildDiscoveryStore(t, tc.items)
			defer store.Close()
			service := New(store)

			result, err := service.Discover(tc.req)
			if err != nil {
				t.Fatalf("Discover returned error: %v", err)
			}
			if result.Coverage.Status != tc.name {
				t.Fatalf("expected %s coverage, got %#v", tc.name, result.Coverage)
			}
			for _, call := range result.NextCalls {
				if !implementedTools[call.Tool] {
					t.Fatalf("expected implemented tool only, got %#v", result.NextCalls)
				}
			}
		})
	}
}

func TestDiscoverAndMapTreatEmptyProjectsAsGlobalKnowledge(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	discovered, err := service.Discover(DiscoverRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Query:   "refresh token",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(discovered.Items) != 1 || discovered.Items[0].ID != "rule:backend.global-refresh.v1" {
		t.Fatalf("expected global discovery item, got %#v", discovered.Items)
	}

	mapped, err := service.Map(MapRequest{Project: "mall-api", Domain: "backend"})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	if mapped.Inventory.Types["rule"] != 1 {
		t.Fatalf("expected global map item, got %#v", mapped.Inventory)
	}
}

func TestDiscoverReturnsErrorForInvalidFileScopeGlob(t *testing.T) {
	store := buildDiscoveryStore(t, []knowledge.Item{{
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

	_, err := service.Discover(DiscoverRequest{
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

func TestContextRecommendsNextCalls(t *testing.T) {
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
	if len(result.RecommendedNextCalls) == 0 {
		t.Fatal("expected recommended next calls")
	}
}

func TestContextRecommendationsOnlyUseCallableTools(t *testing.T) {
	service := New(nil)
	callableTools := map[string]bool{
		"argos_context":      true,
		"argos_discover":     true,
		"argos_map":          true,
		"argos_standards":    true,
		"get_knowledge_item": true,
		"cite_knowledge":     true,
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

			if len(result.RecommendedNextCalls) == 0 {
				t.Fatal("expected recommended next calls")
			}
			for _, call := range result.RecommendedNextCalls {
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

func TestContextDoesNotRecommendMapForNarrowImplementationWork(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "fix typo",
	})

	var tools []string
	for _, call := range result.RecommendedNextCalls {
		tools = append(tools, call.Tool)
	}
	if !contains(tools, "argos_discover") || !contains(tools, "argos_standards") {
		t.Fatalf("expected discover and standards recommendations, got %#v", result.RecommendedNextCalls)
	}
	if contains(tools, "argos_map") {
		t.Fatalf("did not expect map recommendation for narrow implementation work: %#v", result.RecommendedNextCalls)
	}
}

func TestGetKnowledgeItemReturnsFullBody(t *testing.T) {
	store := buildQueryTestStore(t)
	defer store.Close()
	service := New(store)

	item, err := service.GetKnowledgeItem("rule:backend.auth.v1")
	if err != nil {
		t.Fatalf("GetKnowledgeItem returned error: %v", err)
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

func containsDiscoveryItem(items []DiscoveryItem, id string) bool {
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

func buildDiscoveryTestStore(t *testing.T) *index.Store {
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
	return buildDiscoveryStore(t, items)
}

func buildDiscoveryStore(t *testing.T, items []knowledge.Item) *index.Store {
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
