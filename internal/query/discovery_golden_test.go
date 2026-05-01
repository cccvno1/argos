package query

import (
	"strings"
	"testing"

	"argos/internal/discoverytest"
)

func TestGoldenFindKnowledgeCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "find" && tc.Operation != "workflow" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.FindKnowledge(FindKnowledgeRequest{
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
				t.Fatalf("FindKnowledge returned error: %v", err)
			}
			assertSupport(t, result.Support, tc.Expected.Support, result.Items)
			assertSupportLevelMatchesExpected(t, result.Support, tc.Expected, result.Items)
			assertUsageMatchesExpected(t, result.Usage, tc.Expected)
			assertSearchStatusMatchesExpected(t, result.SearchStatus, tc.Expected)
			assertMissingNeedsMatchExpected(t, result.MissingNeeds, tc.Expected.MissingNeedSources)
			assertKnowledgeIDs(t, result.Items, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertTopID(t, result.Items, tc.Expected.TopID)
			assertNoKnowledgeBodies(t, result.Items, tc.Expected.NoBodies)
			assertNextSteps(t, result.NextSteps, tc.Expected.RequireNextStepTools, tc.Expected.ForbidNextStepTools)
			assertMissingHints(t, result.Support, tc.Expected.RequireMissingHints)
			assertWhyContains(t, result.Items, tc.Expected.WhyContains)
			if tc.Expected.ItemsEmpty && len(result.Items) != 0 {
				t.Fatalf("expected empty items, got %#v", result.Items)
			}
		})
	}
}

func TestGoldenListKnowledgeCases(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)

	for _, tc := range discoverytest.LoadCases(t) {
		if tc.Operation != "list" {
			continue
		}
		t.Run(tc.ID, func(t *testing.T) {
			result, err := service.ListKnowledge(ListKnowledgeRequest{Project: tc.Input.Project})
			if err != nil {
				t.Fatalf("ListKnowledge returned error: %v", err)
			}
			assertUsageMatchesExpected(t, result.Usage, tc.Expected)
			assertInventoryMinimums(t, result.Inventory.Types, tc.Expected.InventoryTypesMin)
			assertStringIncludes(t, result.Inventory.Domains, tc.Expected.IncludeDomains)
			assertStringIncludes(t, result.Inventory.Tags, tc.Expected.IncludeTags)
			assertListIDs(t, result.Groups, tc.Expected.IncludeIDs, tc.Expected.ExcludeIDs)
			assertNoListBodies(t, result.Groups, tc.Expected.NoBodies)
		})
	}
}

func TestGoldenListKnowledgeEmptyCase(t *testing.T) {
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "list_inventory_empty")
	if tc.Fixture != "empty" {
		t.Fatalf("expected list_inventory_empty to use empty fixture, got %q", tc.Fixture)
	}
	_, store := discoverytest.BuildIndexedEmptyWorkspace(t)
	defer store.Close()
	service := New(store)

	result, err := service.ListKnowledge(ListKnowledgeRequest{Project: tc.Input.Project})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	assertUsageMatchesExpected(t, result.Usage, tc.Expected)
	if tc.Expected.GroupsEmpty && len(result.Groups) != 0 {
		t.Fatalf("expected empty groups, got %#v", result.Groups)
	}
	assertNoListBodies(t, result.Groups, tc.Expected.NoBodies)
}

func TestGoldenDeprecatedListKnowledgeCase(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "list_hides_deprecated_by_default")

	result, err := service.ListKnowledge(ListKnowledgeRequest{Project: tc.Input.Project, IncludeDeprecated: true})
	if err != nil {
		t.Fatalf("ListKnowledge returned error: %v", err)
	}
	assertUsageMatchesExpected(t, result.Usage, tc.Expected)
	assertListIDs(t, result.Groups, []string{tc.Expected.IncludeDeprecatedIDWhenRequested}, nil)
}

func TestGoldenProgressiveReadStatusAndCitationGuard(t *testing.T) {
	_, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	service := New(store)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "progressive_read_status_and_citation_guard")

	found, err := service.FindKnowledge(FindKnowledgeRequest{
		Project: tc.Input.Project,
		Phase:   tc.Input.Phase,
		Task:    tc.Input.Task,
		Query:   tc.Input.Query,
		Files:   tc.Input.Files,
		Limit:   tc.Input.Limit,
	})
	if err != nil {
		t.Fatalf("FindKnowledge returned error: %v", err)
	}
	assertNoKnowledgeBodies(t, found.Items, true)

	for _, id := range tc.Expected.LoadIDs {
		item, err := service.ReadKnowledge(id)
		if err != nil {
			t.Fatalf("ReadKnowledge(%s) returned error: %v", id, err)
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

func assertSupport(t *testing.T, got Support, want string, items []KnowledgeSummary) {
	t.Helper()
	if want != "" && got.Level != want {
		t.Fatalf("expected support %q, got %#v with items %#v", want, got, items)
	}
}

func assertSupportLevelMatchesExpected(t *testing.T, got Support, expected discoverytest.Expected, items []KnowledgeSummary) {
	t.Helper()
	if expected.SupportLevel == "" {
		return
	}
	if got.Level != expected.SupportLevel {
		t.Fatalf("expected support level %q, got %#v with items %#v", expected.SupportLevel, got, items)
	}
}

func assertUsageMatchesExpected(t *testing.T, got UsageGuidance, expected discoverytest.Expected) {
	t.Helper()
	if expected.UsageRead == "" {
		return
	}
	want := UsageGuidance{
		Read:  expected.UsageRead,
		Cite:  expected.UsageCite,
		Claim: expected.UsageClaim,
	}
	assertUsageGuidance(t, got, want)
}

func assertSearchStatusMatchesExpected(t *testing.T, got SearchStatus, expected discoverytest.Expected) {
	t.Helper()
	if expected.SearchSemanticStatus == "" {
		return
	}
	if got.Semantic.Status != expected.SearchSemanticStatus {
		t.Fatalf("expected semantic search status %q, got %#v", expected.SearchSemanticStatus, got)
	}
}

func assertMissingNeedsMatchExpected(t *testing.T, got []MissingNeed, want []string) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("expected no missing needs, got %#v", got)
		}
		return
	}
	assertMissingNeedSources(t, got, want)
}

func assertKnowledgeIDs(t *testing.T, items []KnowledgeSummary, include []string, exclude []string) {
	t.Helper()
	ids := knowledgeIDs(items)
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected knowledge ID %s in %#v", id, items)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect knowledge ID %s in %#v", id, items)
		}
	}
}

func assertTopID(t *testing.T, items []KnowledgeSummary, want string) {
	t.Helper()
	if want == "" {
		return
	}
	if len(items) == 0 || items[0].ID != want {
		t.Fatalf("expected top ID %s, got %#v", want, items)
	}
}

func assertNoKnowledgeBodies(t *testing.T, items []KnowledgeSummary, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, item := range items {
		if item.Body != "" {
			t.Fatalf("find returned body for %s", item.ID)
		}
	}
}

func assertNoListBodies(t *testing.T, groups []ListGroup, required bool) {
	t.Helper()
	if !required {
		return
	}
	for _, group := range groups {
		for _, item := range group.Items {
			if item.Body != "" {
				t.Fatalf("list returned body for %s", item.ID)
			}
		}
	}
}

func assertNextSteps(t *testing.T, steps []NextStep, require []string, forbid []string) {
	t.Helper()
	tools := map[string]bool{}
	for _, step := range steps {
		tools[step.Tool] = true
	}
	for _, tool := range require {
		if !tools[tool] {
			t.Fatalf("expected next step %s in %#v", tool, steps)
		}
	}
	for _, tool := range forbid {
		if tools[tool] {
			t.Fatalf("did not expect next step %s in %#v", tool, steps)
		}
	}
}

func assertMissingHints(t *testing.T, support Support, required bool) {
	t.Helper()
	if required && len(support.MissingKnowledgeHints) == 0 {
		t.Fatalf("expected missing knowledge hints in %#v", support)
	}
}

func assertWhyContains(t *testing.T, items []KnowledgeSummary, fragments []string) {
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

func assertListIDs(t *testing.T, groups []ListGroup, include []string, exclude []string) {
	t.Helper()
	ids := map[string]bool{}
	for _, group := range groups {
		for _, item := range group.Items {
			ids[item.ID] = true
		}
	}
	for _, id := range include {
		if !ids[id] {
			t.Fatalf("expected list ID %s in %#v", id, groups)
		}
	}
	for _, id := range exclude {
		if ids[id] {
			t.Fatalf("did not expect list ID %s in %#v", id, groups)
		}
	}
}

func knowledgeIDs(items []KnowledgeSummary) map[string]bool {
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	return ids
}
