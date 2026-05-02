# Argos Context Entrypoint Tightening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `argos_context` and `argos context` a reliable first-step entrypoint that carries `project`, `phase`, `task`, and `files` into concrete follow-up Argos calls.

**Architecture:** Keep `query.Service.Context` as the single source of the context response. Extend its response type with echoed task inputs and argument-bearing next steps, then make CLI, MCP, adapters, and dogfood consume that contract without changing retrieval ranking or knowledge storage.

**Tech Stack:** Go, standard `flag` CLI parsing, JSON serialization, MCP JSON-RPC over stdio, existing `internal/query`, `internal/cli`, `internal/mcp`, `internal/adapters`, `internal/dogfood`, and `testdata/discovery-golden`.

---

## File Structure

- Modify `internal/query/query.go`
  - Add `ContextNextStep`.
  - Extend `ContextResponse`.
  - Normalize context files and return arguments for follow-up tools.
- Modify `internal/query/query_test.go`
  - Lock context echo fields, argument payloads, broad/narrow ordering, and no direct read/cite recommendations.
- Modify `internal/cli/cli.go`
  - Make `argos context` require `--json`, `--project`, `--phase`, `--task`.
  - Add repeated `--files`.
- Modify `internal/cli/cli_test.go`
  - Cover strict context validation and repeated files.
- Modify `internal/mcp/server_test.go`
  - Assert MCP `argos_context` response includes task, files, and step arguments.
- Modify `internal/adapters/adapters.go`
  - Add adapter guidance to preserve context-returned arguments for follow-up calls.
- Modify `internal/adapters/adapters_test.go`
  - Lock adapter wording.
- Modify `testdata/discovery-golden/cases.json`
  - Add one context-driven dogfood case.
- Modify `internal/dogfood/packet.go`
  - Add operation note for the context-driven case.
- Modify `internal/dogfood/dogfood_test.go`
  - Lock operation note and packet coverage for the new case.
- Modify `docs/superpowers/reports/`
  - Add a short dogfood report after fresh runner validation.

---

### Task 1: Query Context Response Contract

**Files:**
- Modify: `internal/query/query.go`
- Modify: `internal/query/query_test.go`

- [ ] **Step 1: Write failing tests for context echo fields and follow-up arguments**

Add these tests near the existing context tests in `internal/query/query_test.go`:

```go
func TestContextEchoesTaskInputsAndReturnsStepArguments(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
		Files:   []string{" internal/auth/session.go ", "", "internal/auth/session_test.go"},
	})

	if result.Project != "mall-api" {
		t.Fatalf("Project = %q, want mall-api", result.Project)
	}
	if result.Phase != "implementation" {
		t.Fatalf("Phase = %q, want implementation", result.Phase)
	}
	if result.Task != "add refresh token endpoint" {
		t.Fatalf("Task = %q, want task echo", result.Task)
	}
	if got, want := result.Files, []string{"internal/auth/session.go", "internal/auth/session_test.go"}; !sameStrings(got, want) {
		t.Fatalf("Files = %#v, want %#v", got, want)
	}

	find := contextStepByTool(t, result.RecommendedNextSteps, "argos_find_knowledge")
	if find.Arguments["project"] != "mall-api" || find.Arguments["phase"] != "implementation" || find.Arguments["task"] != "add refresh token endpoint" {
		t.Fatalf("find arguments did not preserve context: %#v", find.Arguments)
	}
	if got, want := stringSliceArgument(t, find.Arguments["files"]), []string{"internal/auth/session.go", "internal/auth/session_test.go"}; !sameStrings(got, want) {
		t.Fatalf("find files = %#v, want %#v", got, want)
	}

	standards := contextStepByTool(t, result.RecommendedNextSteps, "argos_standards")
	if standards.Arguments["project"] != "mall-api" || standards.Arguments["task_type"] != "implementation" {
		t.Fatalf("standards arguments did not preserve context: %#v", standards.Arguments)
	}
	if got, want := stringSliceArgument(t, standards.Arguments["files"]), []string{"internal/auth/session.go", "internal/auth/session_test.go"}; !sameStrings(got, want) {
		t.Fatalf("standards files = %#v, want %#v", got, want)
	}
}

func TestContextDoesNotRecommendReadOrCiteDirectly(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "add refresh token endpoint",
	})

	for _, step := range result.RecommendedNextSteps {
		if step.Tool == "argos_read_knowledge" || step.Tool == "argos_cite_knowledge" {
			t.Fatalf("context must not recommend %s directly: %#v", step.Tool, result.RecommendedNextSteps)
		}
	}
}

func TestContextBroadPlanningAddsInventoryBeforeFindAndStandards(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "planning",
		Task:    "understand auth refresh token flow",
	})

	if got, want := contextTools(result.RecommendedNextSteps), []string{"argos_list_knowledge", "argos_find_knowledge", "argos_standards"}; !sameStrings(got, want) {
		t.Fatalf("context tools = %#v, want %#v", got, want)
	}
	list := contextStepByTool(t, result.RecommendedNextSteps, "argos_list_knowledge")
	if list.Arguments["project"] != "mall-api" {
		t.Fatalf("list arguments = %#v, want project", list.Arguments)
	}
}

func TestContextNarrowImplementationOmitsInventory(t *testing.T) {
	service := New(nil)
	result := service.Context(ContextRequest{
		Project: "mall-api",
		Phase:   "implementation",
		Task:    "fix auth token rotation bug",
		Files:   []string{"internal/auth/session.go"},
	})

	if got, want := contextTools(result.RecommendedNextSteps), []string{"argos_find_knowledge", "argos_standards"}; !sameStrings(got, want) {
		t.Fatalf("context tools = %#v, want %#v", got, want)
	}
}

func contextStepByTool(t *testing.T, steps []ContextNextStep, tool string) ContextNextStep {
	t.Helper()
	for _, step := range steps {
		if step.Tool == tool {
			return step
		}
	}
	t.Fatalf("missing context step %s in %#v", tool, steps)
	return ContextNextStep{}
}

func contextTools(steps []ContextNextStep) []string {
	tools := make([]string, 0, len(steps))
	for _, step := range steps {
		tools = append(tools, step.Tool)
	}
	return tools
}

func stringSliceArgument(t *testing.T, value any) []string {
	t.Helper()
	values, ok := value.([]string)
	if !ok {
		t.Fatalf("argument %#v has type %T, want []string", value, value)
	}
	return values
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/query -run 'TestContextEchoesTaskInputsAndReturnsStepArguments|TestContextDoesNotRecommendReadOrCiteDirectly|TestContextBroadPlanningAddsInventoryBeforeFindAndStandards|TestContextNarrowImplementationOmitsInventory' -count=1
```

Expected: FAIL because `ContextResponse` has no `Task`, `Files`, `ContextNextStep`, or step `Arguments`.

- [ ] **Step 3: Implement the response type and context builders**

In `internal/query/query.go`, replace `ContextResponse` and add `ContextNextStep`:

```go
type ContextResponse struct {
	Project              string            `json:"project"`
	Phase                string            `json:"phase"`
	Task                 string            `json:"task"`
	Files                []string          `json:"files,omitempty"`
	RecommendedNextSteps []ContextNextStep `json:"recommended_next_steps"`
}

type ContextNextStep struct {
	Tool      string         `json:"tool"`
	Reason    string         `json:"reason"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
```

Replace `Context` with:

```go
func (s *Service) Context(req ContextRequest) ContextResponse {
	project := strings.TrimSpace(req.Project)
	phase := strings.TrimSpace(req.Phase)
	task := strings.TrimSpace(req.Task)
	files := normalizedContextFiles(req.Files)

	reason := "standards are useful before code changes"
	switch phase {
	case "planning":
		reason = "planning should start from active project standards"
	case "implementation":
		reason = "implementation should follow active coding and architecture standards"
	case "review":
		reason = "review should check changes against active standards"
	case "debugging":
		reason = "debugging should account for active standards before changing behavior"
	case "operations", "deployment":
		reason = "operational work should respect active project standards"
	}

	calls := []ContextNextStep{
		{
			Tool:      "argos_find_knowledge",
			Reason:    "find task-relevant Argos knowledge without reading full bodies",
			Arguments: contextFindArguments(project, phase, task, files),
		},
		{
			Tool:      "argos_standards",
			Reason:    reason,
			Arguments: contextStandardsArguments(project, phase, files),
		},
	}
	if contextNeedsInventory(phase, task, files) {
		calls = append([]ContextNextStep{{
			Tool:      "argos_list_knowledge",
			Reason:    "inventory available project knowledge before broad work",
			Arguments: map[string]any{"project": project},
		}}, calls...)
	}

	return ContextResponse{
		Project:              project,
		Phase:                phase,
		Task:                 task,
		Files:                files,
		RecommendedNextSteps: calls,
	}
}

func normalizedContextFiles(files []string) []string {
	var normalized []string
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file != "" {
			normalized = append(normalized, file)
		}
	}
	return normalized
}

func contextFindArguments(project string, phase string, task string, files []string) map[string]any {
	args := map[string]any{
		"project": project,
		"phase":   phase,
		"task":    task,
	}
	if len(files) > 0 {
		args["files"] = files
	}
	return args
}

func contextStandardsArguments(project string, phase string, files []string) map[string]any {
	args := map[string]any{
		"project":   project,
		"task_type": phase,
	}
	if len(files) > 0 {
		args["files"] = files
	}
	return args
}

func contextNeedsInventory(phase string, task string, files []string) bool {
	if len(files) > 0 {
		return false
	}
	task = strings.ToLower(task)
	if phase == "planning" {
		return true
	}
	for _, term := range []string{"understand", "explore", "orient", "map"} {
		if strings.Contains(task, term) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run focused query tests**

Run:

```bash
go test ./internal/query -run 'TestContext' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 1**

```bash
git add internal/query/query.go internal/query/query_test.go
git commit -m "feat: return concrete context next steps"
```

---

### Task 2: CLI Context Contract

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add these tests near `TestRunContextPrintsWorkflowContractJSON` in `internal/cli/cli_test.go`:

```go
func TestRunContextRequiresJSONAndTaskContext(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing json", args: []string{"context", "--project", "mall-api", "--phase", "implementation", "--task", "add refresh token endpoint"}, want: "context: --json is required"},
		{name: "missing project", args: []string{"context", "--json", "--phase", "implementation", "--task", "add refresh token endpoint"}, want: "context: --project is required"},
		{name: "missing phase", args: []string{"context", "--json", "--project", "mall-api", "--task", "add refresh token endpoint"}, want: "context: --phase is required"},
		{name: "missing task", args: []string{"context", "--json", "--project", "mall-api", "--phase", "implementation"}, want: "context: --task is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestRunContextAcceptsRepeatedFilesAndReturnsStepArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"context",
		"--json",
		"--project", "mall-api",
		"--phase", "implementation",
		"--task", "add refresh token endpoint",
		"--files", " internal/auth/session.go ",
		"--files", "",
		"--files", "internal/auth/session_test.go",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var result struct {
		Project string   `json:"project"`
		Phase   string   `json:"phase"`
		Task    string   `json:"task"`
		Files   []string `json:"files"`
		Steps   []struct {
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		} `json:"recommended_next_steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON output, got error %v and output %q", err, stdout.String())
	}
	if result.Project != "mall-api" || result.Phase != "implementation" || result.Task != "add refresh token endpoint" {
		t.Fatalf("context echo mismatch: %#v", result)
	}
	if got, want := result.Files, []string{"internal/auth/session.go", "internal/auth/session_test.go"}; !sameCLIStrings(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
	if len(result.Steps) == 0 || result.Steps[0].Tool != "argos_find_knowledge" {
		t.Fatalf("expected find as first step, got %#v", result.Steps)
	}
	if result.Steps[0].Arguments["project"] != "mall-api" || result.Steps[0].Arguments["phase"] != "implementation" || result.Steps[0].Arguments["task"] != "add refresh token endpoint" {
		t.Fatalf("find arguments did not preserve context: %#v", result.Steps[0].Arguments)
	}
}

func sameCLIStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run CLI context tests and verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRunContextRequiresJSONAndTaskContext|TestRunContextAcceptsRepeatedFilesAndReturnsStepArguments' -count=1
```

Expected: FAIL because CLI context does not enforce `--json`, does not require phase/task, and does not accept `--files`.

- [ ] **Step 3: Implement CLI validation and files**

In `internal/cli/cli.go`, replace the `case "context":` block with:

```go
case "context":
	flags := flag.NewFlagSet("context", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	phase := flags.String("phase", "", "workflow phase")
	task := flags.String("task", "", "task description")
	var files multiValueFlag
	flags.Var(&files, "files", "file path relevant to the current task; may be repeated")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "context: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "context: --project is required")
		return 2
	}
	if strings.TrimSpace(*phase) == "" {
		fmt.Fprintln(stderr, "context: --phase is required")
		return 2
	}
	if strings.TrimSpace(*task) == "" {
		fmt.Fprintln(stderr, "context: --task is required")
		return 2
	}

	result := query.New(nil).Context(query.ContextRequest{
		Project: *project,
		Phase:   *phase,
		Task:    *task,
		Files:   files,
	})
	return printJSON(stdout, stderr, result)
```

- [ ] **Step 4: Update existing context CLI test**

Update `TestRunContextPrintsWorkflowContractJSON` so the command includes all required flags:

```go
code := Run([]string{
	"context",
	"--json",
	"--project", "mall-api",
	"--phase", "planning",
	"--task", "understand auth refresh token flow",
}, &stdout, &stderr)
```

Extend its decoded struct with `Phase` and `Task`, and assert they are echoed:

```go
if result.Project != "mall-api" {
	t.Fatalf("unexpected project: %s", result.Project)
}
```

Keep the existing recommended-next-steps assertion.

- [ ] **Step 5: Run focused CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRunContext' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 2**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat: align cli context contract"
```

---

### Task 3: MCP And Adapter Surface Alignment

**Files:**
- Modify: `internal/mcp/server_test.go`
- Modify: `internal/adapters/adapters.go`
- Modify: `internal/adapters/adapters_test.go`

- [ ] **Step 1: Write failing MCP response assertion**

Update `TestToolCallArgosContextWorksWithoutIndex` in `internal/mcp/server_test.go` by adding these assertions after the existing project assertion:

```go
if !strings.Contains(text, `"phase": "implementation"`) {
	t.Fatalf("expected phase in context response: %s", text)
}
if !strings.Contains(text, `"task": "add refresh token endpoint"`) {
	t.Fatalf("expected task in context response: %s", text)
}
if !strings.Contains(text, `"files": [`) || !strings.Contains(text, `"internal/auth/session.go"`) {
	t.Fatalf("expected files in context response: %s", text)
}
if !strings.Contains(text, `"arguments"`) || !strings.Contains(text, `"argos_find_knowledge"`) || !strings.Contains(text, `"argos_standards"`) {
	t.Fatalf("expected argument-bearing next steps in context response: %s", text)
}
```

- [ ] **Step 2: Run MCP context test**

Run:

```bash
go test ./internal/mcp -run TestToolCallArgosContextWorksWithoutIndex -count=1
```

Expected: PASS if Task 1 is complete. If it fails, the query response is not being serialized through MCP as expected.

- [ ] **Step 3: Write failing adapter wording assertion**

In `internal/adapters/adapters_test.go`, add this expected string to `TestRenderedAdaptersIncludeStableKnowledgeContract`:

```go
"Preserve arguments returned by argos_context when calling follow-up Argos tools.",
```

- [ ] **Step 4: Run adapter wording test and verify it fails**

Run:

```bash
go test ./internal/adapters -run TestRenderedAdaptersIncludeStableKnowledgeContract -count=1
```

Expected: FAIL because adapter text does not yet mention preserving context arguments.

- [ ] **Step 5: Update adapter protocol wording**

In `internal/adapters/adapters.go`, add a work protocol line immediately after the `argos_context` line:

```text
2. Preserve arguments returned by argos_context when calling follow-up Argos tools.
```

Renumber the following lines so the protocol remains ordered and readable.

- [ ] **Step 6: Run MCP and adapter tests**

Run:

```bash
go test ./internal/mcp ./internal/adapters -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 3**

```bash
git add internal/mcp/server_test.go internal/adapters/adapters.go internal/adapters/adapters_test.go
git commit -m "docs: teach adapters context argument carryover"
```

---

### Task 4: Context-Driven Dogfood Case

**Files:**
- Modify: `testdata/discovery-golden/cases.json`
- Modify: `internal/dogfood/packet.go`
- Modify: `internal/dogfood/dogfood_test.go`

- [ ] **Step 1: Add a failing dogfood operation note test**

Add this row to `TestPacketAddsOperationSpecificNotes` in `internal/dogfood/dogfood_test.go`:

```go
{name: "context workflow", id: "context_entrypoint_carries_task_arguments", want: "Context workflow case: call context first, then use returned arguments for find before read/cite."},
```

- [ ] **Step 2: Add the golden case**

Append this case before `interface_mcp_strict_schema` in `testdata/discovery-golden/cases.json`:

```json
{
  "id": "context_entrypoint_carries_task_arguments",
  "operation": "context-workflow",
  "input": {
    "project": "mall-api",
    "phase": "implementation",
    "task": "add refresh token endpoint",
    "query": "refresh token session renewal",
    "files": ["internal/auth/session.go"],
    "limit": 5
  },
  "expected": {
    "support": "strong",
    "support_level": "strong",
    "usage_read": "recommended",
    "usage_cite": "after_read_and_used",
    "usage_claim": "allowed",
    "search_semantic_status": "disabled",
    "include_ids": ["rule:backend.auth-refresh.v1"],
    "load_ids": ["rule:backend.auth-refresh.v1"],
    "cite_ids": ["rule:backend.auth-refresh.v1"],
    "no_bodies": true
  }
}
```

Keep JSON commas valid after inserting the case.

- [ ] **Step 3: Run dogfood test and verify it fails**

Run:

```bash
go test ./internal/dogfood -run TestPacketAddsOperationSpecificNotes -count=1
```

Expected: FAIL because `operationNotes` rejects `context-workflow`.

- [ ] **Step 4: Implement dogfood operation note**

In `internal/dogfood/packet.go`, add this switch case in `operationNotes`:

```go
case "context-workflow":
	notes = append(notes, "Context workflow case: call context first, then use returned arguments for find before read/cite.")
```

- [ ] **Step 5: Run dogfood and discovery tests**

Run:

```bash
go test ./internal/dogfood ./internal/discoverytest -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add testdata/discovery-golden/cases.json internal/dogfood/packet.go internal/dogfood/dogfood_test.go
git commit -m "test: add context entrypoint dogfood case"
```

---

### Task 5: End-To-End Verification And Round 9 Dogfood Prep

**Files:**
- Create: `docs/superpowers/reports/2026-05-02-argos-context-entrypoint-round-9.md`
- No production code changes.

- [ ] **Step 1: Run full automated verification**

Run:

```bash
go test ./... -count=1
```

Expected: all packages PASS.

- [ ] **Step 2: Build a dogfood binary**

Run:

```bash
ROUND_ROOT="$(mktemp -d /tmp/argos-context-dogfood-20260502-XXXXXX)"
go build -o "$ROUND_ROOT/argos" ./cmd/argos
```

Expected: command exits 0 and `$ROUND_ROOT/argos` exists.

- [ ] **Step 3: Prepare an indexed fixture workspace**

Run:

```bash
mkdir -p "$ROUND_ROOT/full" "$ROUND_ROOT/packets" "$ROUND_ROOT/reports"
cp -R testdata/discovery-golden/. "$ROUND_ROOT/full/"
(cd "$ROUND_ROOT/full" && "$ROUND_ROOT/argos" index)
```

Expected: `indexed 9 knowledge item(s)` and `$ROUND_ROOT/full/argos/index.db` exists.

- [ ] **Step 4: Generate the context dogfood packet**

Run:

```bash
"$ROUND_ROOT/argos" dogfood packet --case context_entrypoint_carries_task_arguments --workspace "$ROUND_ROOT/full" --argos-binary "$ROUND_ROOT/argos" > "$ROUND_ROOT/packets/context-entrypoint.md"
```

Expected: packet contains public case handle, operation `context-workflow`, context workflow note, and no `expected` block.

- [ ] **Step 5: Run one fresh dogfood runner**

Use a fresh agent context with only this packet:

```text
$ROUND_ROOT/packets/context-entrypoint.md
```

Runner must:

1. call `argos context --json --project mall-api --phase implementation --task "add refresh token endpoint" --files internal/auth/session.go`;
2. use the returned `argos_find_knowledge` arguments to call `knowledge find`;
3. read only selected IDs allowed by find;
4. cite only read-and-used IDs;
5. save report to `$ROUND_ROOT/reports/context-entrypoint.md`.

Expected: report result is `pass`.

- [ ] **Step 6: Evaluate the report**

Run:

```bash
"$ROUND_ROOT/argos" dogfood evaluate --case context_entrypoint_carries_task_arguments --report "$ROUND_ROOT/reports/context-entrypoint.md" --json
```

Expected:

```json
{
  "result": "pass"
}
```

The exact `case_id` may be the public handle assigned by case order.

- [ ] **Step 7: Record the dogfood report**

Create `docs/superpowers/reports/2026-05-02-argos-context-entrypoint-round-9.md`:

```markdown
# Argos Context Entrypoint Dogfood Round 9

Date: 2026-05-02

## Goal

Validate that `argos context` carries project, phase, task, and files into a
follow-up find/read/cite workflow without argument drift.

## Result

`pass`

## Evidence

- `go test ./... -count=1`: pass.
- Fresh runner packet: `$ROUND_ROOT/packets/context-entrypoint.md`.
- Fresh runner report: `$ROUND_ROOT/reports/context-entrypoint.md`.
- Evaluator result: pass.

## Notes

The runner used context-returned arguments before calling find, then read and
cited only IDs selected from the find workflow.
```

- [ ] **Step 8: Run final checks**

Run:

```bash
go test ./... -count=1
git diff --check
```

Expected: tests PASS and `git diff --check` prints no errors.

- [ ] **Step 9: Commit Task 5**

```bash
git add docs/superpowers/reports/2026-05-02-argos-context-entrypoint-round-9.md
git commit -m "docs: record context entrypoint dogfood round"
```

---

## Final Verification

After all tasks are complete, run:

```bash
go test ./... -count=1
git status --short --branch
```

Expected:

- every Go package passes;
- worktree is clean;
- branch is ahead by the task commits.

Then summarize:

- changed files;
- final test output;
- dogfood evaluator result;
- any behavior intentionally made stricter, especially CLI `argos context`.
