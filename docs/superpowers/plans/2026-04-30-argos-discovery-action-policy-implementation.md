# Argos Discovery Action Policy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a machine-readable `action_policy` contract to Argos discovery and map responses so agents know when loading, citation, and Argos-backed claims are allowed.

**Architecture:** Keep policy generation inside `internal/query` beside coverage and next-call generation. `DiscoveryResponse` derives policy from `coverage.status`; `MapResponse` always returns inventory-only policy. CLI and MCP inherit the field through existing JSON serialization, while adapters get short policy-following instructions.

**Tech Stack:** Go, table-driven tests, existing query/CLI/MCP/adapters packages, JSON response structs, Markdown dogfood docs.

---

## File Structure

- Modify `internal/query/query.go`: add `ActionPolicy`, attach it to `DiscoveryResponse` and `MapResponse`, and add small policy helper functions.
- Modify `internal/query/query_test.go`: add focused policy matrix tests for strong, partial, weak, none, and map inventory.
- Modify `internal/discoverytest/golden.go`: add expected action policy fields to golden cases.
- Modify `testdata/discovery-golden/cases.json`: add expected policy fields to representative cases.
- Modify `internal/query/discovery_golden_test.go`: assert golden policy expectations.
- Modify `internal/cli/discovery_golden_test.go`: assert CLI JSON includes policy.
- Modify `internal/mcp/server_test.go`: assert MCP JSON includes policy.
- Modify `internal/adapters/adapters_test.go`: assert adapter contract mentions `action_policy`.
- Modify `internal/adapters/adapters.go`: render the adapter policy guidance.
- Modify `docs/superpowers/templates/argos-discovery-dogfood-report.md`: add policy fields to dogfood reports.
- Add `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-3.md` after running dogfood.

## Task 1: Add Query Action Policy Core

**Files:**
- Modify: `internal/query/query_test.go`
- Modify: `internal/query/query.go`

- [ ] **Step 1: Write the failing policy matrix tests**

Append these helpers and tests near the discovery tests in `internal/query/query_test.go`:

```go
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
				Claim:     "must_mention_gap",
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
```

- [ ] **Step 2: Run the policy matrix tests and verify they fail**

Run:

```bash
go test ./internal/query -run 'TestDiscoverActionPolicyFollowsCoverage|TestMapActionPolicyForbidsLoadCitationAndClaims' -count=1
```

Expected: FAIL to compile because `ActionPolicy` and response fields do not exist yet.

- [ ] **Step 3: Add the response type and fields**

In `internal/query/query.go`, add `ActionPolicy` after `Coverage`:

```go
type ActionPolicy struct {
	Authority string `json:"authority"`
	Load      string `json:"load"`
	Cite      string `json:"cite"`
	Claim     string `json:"claim"`
	Reason    string `json:"reason"`
}
```

Add fields to the response structs:

```go
type DiscoveryResponse struct {
	Project      string                      `json:"project"`
	Phase        string                      `json:"phase"`
	Query        string                      `json:"query"`
	Capabilities index.DiscoveryCapabilities `json:"capabilities"`
	Coverage     Coverage                    `json:"coverage"`
	ActionPolicy ActionPolicy                `json:"action_policy"`
	Items        []DiscoveryItem             `json:"items"`
	NextCalls    []RecommendedCall           `json:"next_calls"`
}

type MapResponse struct {
	Project      string       `json:"project"`
	ActionPolicy ActionPolicy `json:"action_policy"`
	Inventory    Inventory    `json:"inventory"`
	Groups       []MapGroup   `json:"groups"`
}
```

- [ ] **Step 4: Add policy helper functions**

Add these helpers near `discoveryNextCalls` in `internal/query/query.go`:

```go
func discoveryActionPolicy(coverage Coverage) ActionPolicy {
	switch coverage.Status {
	case "strong":
		return ActionPolicy{
			Authority: "strong",
			Load:      "recommended",
			Cite:      "after_loaded_and_used",
			Claim:     "allowed",
			Reason:    "Strong Argos coverage; load selected items before applying and cite only loaded knowledge actually used.",
		}
	case "partial":
		return ActionPolicy{
			Authority: "partial",
			Load:      "allowed",
			Cite:      "after_loaded_and_used",
			Claim:     "must_mention_gap",
			Reason:    "Partial Argos coverage; load only relevant items and mention coverage gaps when applying them.",
		}
	case "weak":
		return ActionPolicy{
			Authority: "weak",
			Load:      "forbidden",
			Cite:      "forbidden",
			Claim:     "forbidden",
			Reason:    "Weak Argos coverage; inspect summaries only and do not make Argos-backed claims.",
		}
	default:
		return ActionPolicy{
			Authority: "none",
			Load:      "forbidden",
			Cite:      "forbidden",
			Claim:     "forbidden",
			Reason:    "No Argos coverage; use missing knowledge hints as gaps only and do not cite Argos knowledge.",
		}
	}
}

func mapActionPolicy() ActionPolicy {
	return ActionPolicy{
		Authority: "inventory",
		Load:      "forbidden",
		Cite:      "forbidden",
		Claim:     "forbidden",
		Reason:    "Map inventory is for orientation only; do not load, cite, or make task claims from inventory alone.",
	}
}
```

- [ ] **Step 5: Attach policy in `Discover` and `Map`**

In `Discover`, after coverage and next calls are computed, return:

```go
return DiscoveryResponse{
	Project:      req.Project,
	Phase:        req.Phase,
	Query:        intent,
	Capabilities: caps,
	Coverage:     coverage,
	ActionPolicy: discoveryActionPolicy(coverage),
	Items:        results,
	NextCalls:    nextCalls,
}, nil
```

In `Map`, return:

```go
return MapResponse{Project: req.Project, ActionPolicy: mapActionPolicy(), Inventory: inventory, Groups: groups}, nil
```

- [ ] **Step 6: Run focused tests and commit**

Run:

```bash
gofmt -w internal/query/query.go internal/query/query_test.go
go test ./internal/query -run 'TestDiscoverActionPolicyFollowsCoverage|TestMapActionPolicyForbidsLoadCitationAndClaims' -count=1
go test ./internal/query -count=1
```

Expected: PASS.

Commit:

```bash
git add internal/query/query.go internal/query/query_test.go
git commit -m "feat: add discovery action policy"
```

## Task 2: Extend Golden, CLI, And MCP Assertions

**Files:**
- Modify: `internal/discoverytest/golden.go`
- Modify: `testdata/discovery-golden/cases.json`
- Modify: `internal/query/discovery_golden_test.go`
- Modify: `internal/cli/discovery_golden_test.go`
- Modify: `internal/mcp/server_test.go`

- [ ] **Step 1: Add failing golden expected fields**

In `internal/discoverytest/golden.go`, add fields to `Expected`:

```go
	ActionAuthority string `json:"action_authority"`
	ActionLoad      string `json:"action_load"`
	ActionCite      string `json:"action_cite"`
	ActionClaim     string `json:"action_claim"`
```

In `testdata/discovery-golden/cases.json`, add these fields to representative cases:

```json
"action_authority": "strong",
"action_load": "recommended",
"action_cite": "after_loaded_and_used",
"action_claim": "allowed"
```

Use this mapping:

- all `coverage: strong` discover/workflow cases: `strong`, `recommended`, `after_loaded_and_used`, `allowed`
- all `coverage: partial` cases: `partial`, `allowed`, `after_loaded_and_used`, `must_mention_gap`
- all `coverage: weak` cases: `weak`, `forbidden`, `forbidden`, `forbidden`
- all `coverage: none` cases: `none`, `forbidden`, `forbidden`, `forbidden`
- map cases: `inventory`, `forbidden`, `forbidden`, `forbidden`

- [ ] **Step 2: Assert golden policies**

In `internal/query/discovery_golden_test.go`, after `assertCoverage`, add:

```go
assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
```

In map tests, after `Map`, add:

```go
assertActionPolicyMatchesExpected(t, result.ActionPolicy, tc.Expected)
```

Add this helper:

```go
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
```

- [ ] **Step 3: Run golden tests and verify they pass**

Run:

```bash
gofmt -w internal/discoverytest/golden.go internal/query/discovery_golden_test.go
go test ./internal/discoverytest ./internal/query -count=1
```

Expected: PASS.

- [ ] **Step 4: Assert CLI JSON includes policy**

In `internal/cli/discovery_golden_test.go`, add to `TestGoldenCLIDiscoverMatchesQueryBehavior` after coverage assertion:

```go
if result.ActionPolicy.Authority != tc.Expected.ActionAuthority {
	t.Fatalf("expected action authority %q, got %#v", tc.Expected.ActionAuthority, result.ActionPolicy)
}
```

Add to `TestGoldenCLIMapMatchesQueryBehavior` after JSON unmarshal:

```go
if result.ActionPolicy.Authority != tc.Expected.ActionAuthority {
	t.Fatalf("expected action authority %q, got %#v", tc.Expected.ActionAuthority, result.ActionPolicy)
}
```

Run:

```bash
gofmt -w internal/cli/discovery_golden_test.go
go test ./internal/cli -run 'TestGoldenCLIDiscoverMatchesQueryBehavior|TestGoldenCLIMapMatchesQueryBehavior' -count=1
```

Expected: PASS.

- [ ] **Step 5: Assert MCP JSON includes policy**

In `internal/mcp/server_test.go`, update `TestToolCallArgosDiscoverReturnsRoutes`:

```go
if !strings.Contains(text, `"action_policy"`) || !strings.Contains(text, `"authority": "strong"`) {
	t.Fatalf("expected action policy in discover response: %s", text)
}
```

Update `TestToolCallArgosMapReturnsInventory`:

```go
if !strings.Contains(text, `"action_policy"`) || !strings.Contains(text, `"authority": "inventory"`) {
	t.Fatalf("expected inventory action policy in map response: %s", text)
}
```

Run:

```bash
gofmt -w internal/mcp/server_test.go
go test ./internal/mcp -run 'TestToolCallArgosDiscoverReturnsRoutes|TestToolCallArgosMapReturnsInventory' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run focused interface tests and commit**

Run:

```bash
go test ./internal/discoverytest ./internal/query ./internal/cli ./internal/mcp -count=1
```

Expected: PASS.

Commit:

```bash
git add internal/discoverytest/golden.go testdata/discovery-golden/cases.json internal/query/discovery_golden_test.go internal/cli/discovery_golden_test.go internal/mcp/server_test.go
git commit -m "test: assert discovery action policy interfaces"
```

## Task 3: Update Adapter Contract And Dogfood Template

**Files:**
- Modify: `internal/adapters/adapters_test.go`
- Modify: `internal/adapters/adapters.go`
- Modify: `docs/superpowers/templates/argos-discovery-dogfood-report.md`

- [ ] **Step 1: Write failing adapter contract assertions**

In `internal/adapters/adapters_test.go`, add these expected strings to `TestRenderedAdaptersIncludeStableKnowledgeContract`:

```go
"Follow action_policy.load before loading full knowledge items.",
"Follow action_policy.cite before calling cite_knowledge.",
"Mention Argos coverage gaps when action_policy.claim is must_mention_gap.",
```

Run:

```bash
go test ./internal/adapters -run TestRenderedAdaptersIncludeStableKnowledgeContract -count=1
```

Expected: FAIL because rendered adapters do not contain these strings yet.

- [ ] **Step 2: Render adapter policy guidance**

In `internal/adapters/adapters.go`, add these lines under `## Work Protocol`:

```text
10. Follow action_policy.load before loading full knowledge items.
11. Follow action_policy.cite before calling cite_knowledge.
12. Mention Argos coverage gaps when action_policy.claim is must_mention_gap.
```

- [ ] **Step 3: Update dogfood report template**

In `docs/superpowers/templates/argos-discovery-dogfood-report.md`, add to `## Observed Results`:

```md
- Action policy:
```

Add to `## Guards`:

```md
- Action policy followed: `pass|fail`
```

- [ ] **Step 4: Run adapter tests and commit**

Run:

```bash
gofmt -w internal/adapters/adapters.go internal/adapters/adapters_test.go
go test ./internal/adapters -count=1
rg -n "Action policy|action_policy.load|action_policy.cite|must_mention_gap" internal/adapters docs/superpowers/templates/argos-discovery-dogfood-report.md
```

Expected: PASS, and `rg` finds the new policy text.

Commit:

```bash
git add internal/adapters/adapters.go internal/adapters/adapters_test.go docs/superpowers/templates/argos-discovery-dogfood-report.md
git commit -m "docs: teach adapters discovery action policy"
```

## Task 4: Run Round 3 Dogfood And Record Results

**Files:**
- Add: `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-3.md`

- [ ] **Step 1: Build dogfood binary and fixture workspace**

Run:

```bash
go build -o /tmp/argos-dogfood ./cmd/argos
full=$(mktemp -d /tmp/argos-round3-full-XXXXXX)
cp -R testdata/discovery-golden/knowledge "$full/knowledge"
(cd "$full" && /tmp/argos-dogfood index)
printf 'FULL=%s\n' "$full"
```

Expected: `indexed 9 knowledge item(s)`.

- [ ] **Step 2: Run five fresh runner cases**

Run fresh runner sessions with `fork_context:false`, one per case:

- strong workflow: `progressive_disclosure_and_citation_guard`
- partial workflow: `partial_lesson_without_rule`
- weak: `weak_single_generic_term`
- none: `none_payment_webhook`
- map-only: `map_inventory_normal`

Each runner receives only:

- case ID
- fixture workspace path
- `/tmp/argos-dogfood`
- case input
- allowed flow
- requirement to follow `action_policy`
- report template path

Each runner must not receive golden expected values, previous transcripts, or dogfood reports.

- [ ] **Step 3: Evaluate dogfood results**

Use these pass criteria:

- strong follows `action_policy.load=recommended`, loads before citing, and cites only loaded-and-used IDs;
- partial follows `action_policy.load=allowed` and mentions coverage gaps if applying loaded knowledge;
- weak follows `load=forbidden`, `cite=forbidden`, and does not make Argos-backed claims;
- none follows `load=forbidden`, `cite=forbidden`, and treats missing hints as gaps only;
- map-only follows inventory policy and does not load or cite.

- [ ] **Step 4: Write Round 3 report**

Create `docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-3.md` with this shape:

```md
# Argos Discovery Dogfood Round 3

Date: 2026-04-30

## Scope

Action policy dogfood validation.

## Result

Overall: `pass|fail|review-needed`

## Findings

### Strong

### Partial

### Weak

### None

### Map Inventory

## Decision

State whether action_policy is clear enough to proceed to optional semantic enhancement design.
```

- [ ] **Step 5: Run verification and commit**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

Commit:

```bash
git add docs/superpowers/reports/2026-04-30-argos-discovery-dogfood-round-3.md
git commit -m "docs: record discovery dogfood round 3"
```

## Task 5: Final Verification

**Files:**
- No new file changes expected.

- [ ] **Step 1: Run full verification**

Run:

```bash
go test ./... -count=1
git status --short
```

Expected:

- all packages pass;
- `git status --short` is empty.

- [ ] **Step 2: Summarize final state**

Report:

- commits created;
- tests run;
- Round 3 result;
- whether action policy is ready to become the stable safety boundary for optional semantic retrieval.

## Self-Review

Spec coverage:

- Response shape is implemented in Task 1.
- Policy matrix is implemented and tested in Tasks 1 and 2.
- Compatibility with existing `coverage` and `next_calls` is preserved by adding fields without removing existing ones.
- CLI/MCP serialization is covered in Task 2.
- Adapter policy guidance is covered in Task 3.
- Dogfood Round 3 is covered in Task 4.
- Full verification is covered in Task 5.

Placeholder scan: no unfinished placeholders are intentionally left in this plan.

Type consistency: the plan uses one Go type name, `ActionPolicy`, and one JSON field name, `action_policy`, consistently across query, CLI, MCP, adapters, and dogfood reports.
