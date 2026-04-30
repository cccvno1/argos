package query

import (
	"strings"
	"testing"

	"argos/internal/discoverytest"
)

func TestGoldenDiscoveryCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "discover" && tc.Operation != "workflow" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.Discover(DiscoverRequest{
				Project:           tc.Input.Project,
				Phase:             tc.Input.Phase,
				Task:              tc.Input.Task,
				Query:             tc.Input.Query,
				Files:             tc.Input.Files,
				Types:             tc.Input.Types,
				Tags:              tc.Input.Tags,
				Domains:           tc.Input.Domains,
				Status:            tc.Input.Status,
				IncludeDeprecated: tc.Input.IncludeDeprecated,
				Limit:             tc.Input.Limit,
			})
			if err != nil {
				t.Fatalf("Discover returned error: %v", err)
			}
			assertCoverage(t, result.Coverage, tc.Expected.Coverage, result.Items)
			assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
			assertRecallMatchesExpected(t, result.Recall, tc.Expected)
			assertGapCandidatesMatchExpected(t, result.GapCandidates, tc.Expected.GapCandidateKinds)
			assertDiscoveryIDs(t, result.Items, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertTopID(t, result.Items, tc.Expected.TopID)
			assertNoDiscoveryBodies(t, result.Items, tc.Expected.NoBodies)
			assertNextCalls(t, result.NextCalls, tc.Expected.RequireNextCallTools, tc.Expected.ForbidNextCallTools)
			assertMissingHints(t, result.Coverage, tc.Expected.RequireMissingHints)
			assertWhyContains(t, result.Items, tc.Expected.WhyContains)
			if tc.Expected.ItemsEmpty && len(result.Items) != 0 {
				t.Fatalf("expected empty items, got %#v", result.Items)
			}
		})
	}
}

func TestGoldenMapCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "map" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.Map(MapRequest{Project: tc.Input.Project})
			if err != nil {
				t.Fatalf("Map returned error: %v", err)
			}
			assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
			assertInventoryMinimums(t, result.Inventory.Types, tc.Expected.InventoryTypesMin)
			assertStringIncludes(t, result.Inventory.Domains, tc.Expected.IncludeDomains)
			assertStringIncludes(t, result.Inventory.Tags, tc.Expected.IncludeTags)
			assertMapIDs(t, result.Groups, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertNoMapBodies(t, result.Groups, tc.Expected.NoBodies)
		})
	}
}

func TestGoldenMapEmptyCase(t *testing.T) {
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "map_inventory_empty")
	if tc.Fixture != "empty" {
		t.Fatalf("expected map_inventory_empty to use empty fixture, got %q", tc.Fixture)
	}
	_, store := discoverytest.BuildIndexedEmptyWorkspace(t)
	defer store.Close()
	service := New(store)

	result, err := service.Map(MapRequest{Project: tc.Input.Project})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
	if tc.Expected.GroupsEmpty && len(result.Groups) != 0 {
		t.Fatalf("expected empty groups, got %#v", result.Groups)
	}
	assertNoMapBodies(t, result.Groups, tc.Expected.NoBodies)
}

func TestGoldenDeprecatedMapCase(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "map_hides_deprecated_by_default")

	result, err := service.Map(MapRequest{Project: tc.Input.Project, IncludeDeprecated: true})
	if err != nil {
		t.Fatalf("Map returned error: %v", err)
	}
	assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
	assertMapIDs(t, result.Groups, []string{tc.Expected.IncludeDeprecatedIDWhenRequested}, nil)
}

func TestGoldenProgressiveDisclosureAndCitationGuard(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "progressive_disclosure_and_citation_guard")

	discovered, err := service.Discover(DiscoverRequest{
		Project: tc.Input.Project,
		Phase:   tc.Input.Phase,
		Task:    tc.Input.Task,
		Query:   tc.Input.Query,
		Files:   tc.Input.Files,
		Limit:   tc.Input.Limit,
	})
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	assertNoDiscoveryBodies(t, discovered.Items, true)

	for _, id := range tc.Expected.LoadIDs {
		item, err := service.GetKnowledgeItem(id)
		if err != nil {
			t.Fatalf("GetKnowledgeItem(%s) returned error: %v", id, err)
		}
		if item.Body == "" {
			t.Fatalf("expected full body for loaded ID %s", id)
		}
	}
	cited := service.CiteKnowledge(tc.Expected.CiteIDs)
	if len(cited.Missing) != 0 {
		t.Fatalf("expected no missing citations, got %#v", cited.Missing)
	}
	if len(cited.Citations) != len(tc.Expected.CiteIDs) {
		t.Fatalf("expected citations for %v, got %#v", tc.Expected.CiteIDs, cited.Citations)
	}
}

func assertCoverage(t *testing.T, got Coverage, want string, items []DiscoveryItem) {
	t.Helper()
	if want != "" && got.Status != want {
		t.Fatalf("expected coverage %q, got %#v with items %#v", want, got, items)
	}
}

func assertActionPolicyMatchesExpected(t *testing.T, got ActionPolicy, expected discoverytest.Expected) {
	t.Helper()
	if expected.ActionAuthority == "" {
		return
	}
	want := ActionPolicy{
		Authority: expected.ActionAuthority,
		Load:      expected.ActionLoad,
		Cite:      expected.ActionCite,
		Claim:     expected.ActionClaim,
	}
	assertActionPolicy(t, got, want)
}

func assertRecallMatchesExpected(t *testing.T, got RecallState, expected discoverytest.Expected) {
	t.Helper()
	if expected.RecallSemanticStatus == "" {
		return
	}
	if got.Semantic.Status != expected.RecallSemanticStatus {
		t.Fatalf("expected semantic recall status %q, got %#v", expected.RecallSemanticStatus, got)
	}
}

func assertGapCandidatesMatchExpected(t *testing.T, got []GapCandidate, want []string) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("expected no gap candidates, got %#v", got)
		}
		return
	}
	assertGapCandidateKinds(t, got, want)
}

func assertDiscoveryIDs(t *testing.T, items []DiscoveryItem, include []string, exclude []string) {
	t.Helper()
	ids := discoveryIDs(items)
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected discovery ID %s in %#v", id, items)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect discovery ID %s in %#v", id, items)
		}
	}
}

func assertTopID(t *testing.T, items []DiscoveryItem, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if len(items) == 0 || items[0].ID != want {
		t.Fatalf("expected top ID %s, got %#v", want, items)
	}
}

func assertNoDiscoveryBodies(t *testing.T, items []DiscoveryItem, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, item := range items {
		if item.Body != "" {
			t.Fatalf("discover returned body for %s", item.ID)
		}
	}
}

func assertNoMapBodies(t *testing.T, groups []MapGroup, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, group := range groups {
		for _, item := range group.Items {
			if item.Body != "" {
				t.Fatalf("map returned body for %s", item.ID)
			}
		}
	}
}

func assertNextCalls(t *testing.T, calls []RecommendedCall, require []string, forbid []string) {
	t.Helper()
	tools := map[string]bool{}
	for _, call := range calls {
		tools[call.Tool] = true
	}
	for _, tool := range require {
		if !tools[tool] {
			t.Fatalf("expected next call %s in %#v", tool, calls)
		}
	}
	for _, tool := range forbid {
		if tools[tool] {
			t.Fatalf("did not expect next call %s in %#v", tool, calls)
		}
	}
}

func assertMissingHints(t *testing.T, coverage Coverage, required bool) {
	t.Helper()
	if required && len(coverage.MissingKnowledgeHints) == 0 {
		t.Fatalf("expected missing knowledge hints in %#v", coverage)
	}
}

func assertWhyContains(t *testing.T, items []DiscoveryItem, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		found := false
		for _, item := range items {
			for _, why := range item.WhyMatched {
				if strings.Contains(why, fragment) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("expected why_matched fragment %q in %#v", fragment, items)
		}
	}
}

func assertInventoryMinimums(t *testing.T, got map[string]int, minimums map[string]int) {
	t.Helper()
	for typ, min := range minimums {
		if got[typ] < min {
			t.Fatalf("expected inventory type %s >= %d, got %#v", typ, min, got)
		}
	}
}

func assertStringIncludes(t *testing.T, got []string, include []string) {
	t.Helper()
	present := map[string]bool{}
	for _, value := range got {
		present[value] = true
	}
	for _, value := range include {
		if !present[value] {
			t.Fatalf("expected %q in %#v", value, got)
		}
	}
}

func assertMapIDs(t *testing.T, groups []MapGroup, include []string, exclude []string) {
	t.Helper()
	ids := map[string]bool{}
	for _, group := range groups {
		for _, item := range group.Items {
			ids[item.ID] = true
		}
	}
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected map ID %s in %#v", id, groups)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect map ID %s in %#v", id, groups)
		}
	}
}

func discoveryIDs(items []DiscoveryItem) map[string]bool {
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	return ids
}
