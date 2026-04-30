package cli

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"argos/internal/discoverytest"
	"argos/internal/query"
)

func TestGoldenCLIDiscoverMatchesQueryBehavior(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_discover_matches_query")

	args := []string{
		"discover", "--json",
		"--project", tc.Input.Project,
		"--phase", tc.Input.Phase,
		"--task", tc.Input.Task,
		"--query", tc.Input.Query,
		"--limit", strconv.Itoa(tc.Input.Limit),
	}
	for _, file := range tc.Input.Files {
		args = append(args, "--files", file)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.DiscoveryResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse discover JSON: %v\n%s", err, stdout.String())
	}
	if result.Coverage.Status != tc.Expected.Coverage {
		t.Fatalf("expected coverage %q, got %#v", tc.Expected.Coverage, result.Coverage)
	}
	if result.ActionPolicy.Authority != tc.Expected.ActionAuthority {
		t.Fatalf("expected action authority %q, got %#v", tc.Expected.ActionAuthority, result.ActionPolicy)
	}
	if !containsGoldenDiscoveryID(result.Items, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Items)
	}
	for _, item := range result.Items {
		if tc.Expected.NoBodies && item.Body != "" {
			t.Fatalf("CLI discover returned body for %s", item.ID)
		}
	}
}

func TestGoldenCLIMapMatchesQueryBehavior(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)
	tc := discoverytest.CaseByID(t, discoverytest.LoadCases(t), "interface_cli_map_matches_query")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"map", "--json", "--project", tc.Input.Project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result query.MapResponse
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parse map JSON: %v\n%s", err, stdout.String())
	}
	if result.ActionPolicy.Authority != tc.Expected.ActionAuthority {
		t.Fatalf("expected action authority %q, got %#v", tc.Expected.ActionAuthority, result.ActionPolicy)
	}
	for typ, min := range tc.Expected.InventoryTypesMin {
		if result.Inventory.Types[typ] < min {
			t.Fatalf("expected type %s >= %d, got %#v", typ, min, result.Inventory.Types)
		}
	}
	if !containsGoldenMapID(result.Groups, tc.Expected.IncludeIDs[0]) {
		t.Fatalf("expected ID %s in %#v", tc.Expected.IncludeIDs[0], result.Groups)
	}
}

func TestGoldenCLIValidationErrorsStayExplicit(t *testing.T) {
	root, store := discoverytest.BuildIndexedWorkspace(t)
	defer store.Close()
	chdir(t, root)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "discover missing task and query", args: []string{"discover", "--json", "--project", "mall-api"}, want: "discover: --task or --query is required"},
		{name: "discover bad limit", args: []string{"discover", "--json", "--project", "mall-api", "--query", "auth", "--limit", "99"}, want: "discover: --limit must be between 1 and 20"},
		{name: "map missing project", args: []string{"map", "--json"}, want: "map: --project is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("expected nonzero exit code")
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.want, stderr.String())
			}
		})
	}
}

func containsGoldenDiscoveryID(items []query.DiscoveryItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func containsGoldenMapID(groups []query.MapGroup, id string) bool {
	for _, group := range groups {
		for _, item := range group.Items {
			if item.ID == id {
				return true
			}
		}
	}
	return false
}
