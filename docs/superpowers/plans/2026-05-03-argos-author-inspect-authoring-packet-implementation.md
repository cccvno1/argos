# Author Inspect Authoring Packet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `authoring_packet` field to `argos author inspect --json` so agents get safe, explicit write-side next actions.

**Architecture:** Keep `author inspect` as the single pre-write entrypoint. Add focused packet structs and deterministic packet-building helpers in `internal/author`, derived from the existing `InspectResponse`, `InspectRequest`, and `proposal_scaffold`. Keep the change backward-compatible by only adding JSON fields.

**Tech Stack:** Go, standard library JSON structs, existing `internal/author` tests, existing CLI tests, existing authoring dogfood harness.

---

## File Structure

- Create `internal/author/authoring_packet.go`: packet structs plus packet-building helpers. This keeps operational packet logic out of the already broad `inspect.go`.
- Modify `internal/author/inspect.go`: add `AuthoringPacket` to `InspectResponse`, build it after `ProposalScaffold`, and add missing-content review-only signal integration.
- Modify `internal/author/inspect_test.go`: add TDD coverage for normal, overlap review-only, missing-content review-only, and consumer API packet guidance.
- Modify `internal/cli/cli_test.go`: assert CLI JSON includes the new packet field.
- Modify `README.md`: document that `author inspect` returns an agent-ready packet.
- Modify `skills/capture-knowledge/SKILL.md`: tell agents to follow `authoring_packet` for next action and stop conditions.

## Task 1: Normal Inspect Packet Contract

**Files:**
- Modify: `internal/author/inspect_test.go`
- Test: `internal/author/inspect_test.go`

- [ ] **Step 1: Write the failing normal-path packet test**

Add this test after `TestInspectIncludesValidProposalV2Scaffold`:

```go
func TestInspectIncludesAuthoringPacketForProposalPath(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project:    "mall-api",
		Goal:       "Create reusable Redis draft practices for future agents.",
		FutureTask: "design Redis cache practices",
		Phase:      "planning",
		Query:      "redis cache practices",
		Tags:       []string{"redis", "cache"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	packet := result.AuthoringPacket
	if packet.State != "ready_for_proposal" {
		t.Fatalf("state = %q, want ready_for_proposal: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_proposal" {
		t.Fatalf("recommended_action = %q, want write_proposal: %#v", packet.RecommendedAction, packet)
	}
	if packet.ProposalPath != "knowledge/.inbox/proposals/mall-api/create-reusable-redis-draft-practices-for-future-agents/proposal.json" {
		t.Fatalf("proposal_path = %q", packet.ProposalPath)
	}
	if packet.CandidatePath != result.ProposalScaffold.ProposedShape.Path {
		t.Fatalf("candidate_path = %q, want scaffold path %q", packet.CandidatePath, result.ProposalScaffold.ProposedShape.Path)
	}
	if packet.CandidateAllowed {
		t.Fatalf("inspect must not authorize candidate writing: %#v", packet)
	}
	if packet.ReviewOnly {
		t.Fatalf("normal packet should not be review-only: %#v", packet)
	}
	if !containsText(packet.StopConditions, "Do not write candidate files until human_review.candidate_write_approved is true.") {
		t.Fatalf("packet missing candidate stop condition: %#v", packet.StopConditions)
	}
	if !containsText(packet.ProposalFocus, "Separate user-stated intent, observed facts, imported facts, synthesized guidance, assumptions, and open questions.") {
		t.Fatalf("packet missing source-separation focus: %#v", packet.ProposalFocus)
	}
	if packet.Commands.VerifyCandidate == "" || !strings.Contains(packet.Commands.VerifyCandidate, "--proposal "+packet.ProposalPath) || !strings.Contains(packet.Commands.VerifyCandidate, "--path "+packet.CandidatePath) {
		t.Fatalf("verify command does not reference proposal and candidate paths: %#v", packet.Commands)
	}
	if len(packet.HumanReviewQuestions) == 0 {
		t.Fatalf("packet should include human review questions: %#v", packet)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal inspect response: %v", err)
	}
	if !strings.Contains(string(data), `"authoring_packet"`) {
		t.Fatalf("inspect JSON missing authoring_packet: %s", string(data))
	}
}
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
go test ./internal/author -run TestInspectIncludesAuthoringPacketForProposalPath -count=1
```

Expected: FAIL with this compiler message:

```text
result.AuthoringPacket undefined
```

- [ ] **Step 3: Commit the failing test**

```bash
git add internal/author/inspect_test.go
git commit -m "test: require inspect authoring packet"
```

## Task 2: Packet Types And Normal Builder

**Files:**
- Create: `internal/author/authoring_packet.go`
- Modify: `internal/author/inspect.go`
- Test: `internal/author/inspect_test.go`

- [ ] **Step 1: Add packet structs and normal-path builder**

Create `internal/author/authoring_packet.go` with this content:

```go
package author

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AuthoringPacket struct {
	State                string                `json:"state"`
	RecommendedAction    string                `json:"recommended_action"`
	ActionReason         string                `json:"action_reason"`
	ProposalPath         string                `json:"proposal_path"`
	CandidatePath        string                `json:"candidate_path,omitempty"`
	CandidateAllowed     bool                  `json:"candidate_allowed"`
	ReviewOnly           bool                  `json:"review_only"`
	StopConditions       []string              `json:"stop_conditions"`
	ProposalFocus        []string              `json:"proposal_focus"`
	SourceWork           AuthoringSourceWork   `json:"source_work"`
	HumanReviewQuestions []string              `json:"human_review_questions"`
	Commands             AuthoringCommands     `json:"commands"`
}

type AuthoringSourceWork struct {
	UserConfirmed []string `json:"user_confirmed,omitempty"`
	Observed      []string `json:"observed,omitempty"`
	Imported      []string `json:"imported,omitempty"`
	Synthesized   []string `json:"synthesized,omitempty"`
	Templates     []string `json:"templates,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	Assumptions   []string `json:"assumptions,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
}

type AuthoringCommands struct {
	WriteProposal   string `json:"write_proposal"`
	VerifyCandidate string `json:"verify_candidate,omitempty"`
}

func buildAuthoringPacket(response InspectResponse, req InspectRequest) AuthoringPacket {
	proposal := response.ProposalScaffold
	reviewOnly := proposal.ProposedShape.ArtifactState == "review_only"
	proposalPath := inspectProposalPath(response.Project, response.Goal)
	candidatePath := proposal.ProposedShape.Path
	state := "ready_for_proposal"
	action := "write_proposal"
	reason := "No blocking overlap was detected; write a proposal for human review before any candidate files."

	if reviewOnly {
		state = "review_only"
		action = "write_review_only_proposal"
		reason = "A review-only proposal is required before candidate writing."
		proposalPath = proposal.ProposedShape.Path
		candidatePath = ""
	}
	if response.PathRisk.Status == "unsafe" {
		state = "review_only"
		action = "write_review_only_proposal"
		reason = "The requested candidate path is unsafe; write a review-only proposal and ask for a safe inbox path."
		candidatePath = ""
	}

	packet := AuthoringPacket{
		State:             state,
		RecommendedAction: action,
		ActionReason:      reason,
		ProposalPath:      proposalPath,
		CandidatePath:     candidatePath,
		CandidateAllowed:  false,
		ReviewOnly:        reviewOnly || response.PathRisk.Status == "unsafe",
		StopConditions: []string{
			"Do not write candidate files until human_review.candidate_write_approved is true.",
			"Do not mutate official knowledge unless official mutation is explicitly authorized.",
			"Do not promote candidates unless promotion is explicitly authorized.",
			"Do not use priority must unless priority_must_authorized is true.",
		},
		ProposalFocus: []string{
			"Clarify the future agent task this knowledge supports.",
			"Separate user-stated intent, observed facts, imported facts, synthesized guidance, assumptions, and open questions.",
			"Record applicability boundaries and negative triggers.",
			"Record overlap decision before candidate writing.",
		},
		SourceWork: AuthoringSourceWork{
			UserConfirmed: []string{"Preserve the user's request as user-stated intent."},
			Observed:      []string{"Add observed repo facts only after inspecting files or commands."},
			Imported:      []string{"Use imported only for external material supplied by the user."},
			Synthesized:   []string{"Mark agent recommendations as synthesized and review-needed."},
			Assumptions:   []string{"Keep missing details visible as assumptions or open questions."},
			OpenQuestions: []string{"Ask only for decisions needed to make the proposal accurate and usable."},
		},
		HumanReviewQuestions: append([]string{}, proposal.HumanReview.ReviewQuestions...),
		Commands: AuthoringCommands{
			WriteProposal: "write JSON to " + proposalPath,
		},
	}
	if packet.Commands.WriteProposal == "write JSON to " {
		packet.Commands.WriteProposal = "write proposal JSON under knowledge/.inbox/proposals/"
	}
	if candidatePath != "" {
		packet.Commands.VerifyCandidate = fmt.Sprintf("argos author verify --json --proposal %s --path %s", proposalPath, candidatePath)
	}
	if requestLooksConsumerFacing(req) {
		packet.ProposalFocus = append(packet.ProposalFocus, "For consumer-facing knowledge, separate observed interface facts from user interpretation before advising consumers.")
	}
	if len(packet.HumanReviewQuestions) == 0 {
		packet.HumanReviewQuestions = []string{
			"Is the knowledge goal correct and specific enough?",
			"Is candidate writing approved after reviewing the proposal?",
		}
	}
	return packet
}

func inspectProposalPath(project string, goal string) string {
	project = strings.TrimSpace(project)
	if project == "" {
		project = "project"
	}
	return filepath.ToSlash(filepath.Join("knowledge/.inbox/proposals", project, slugForAuthoring(goal), "proposal.json"))
}
```

- [ ] **Step 2: Wire packet into `InspectResponse`**

In `internal/author/inspect.go`, add this field to `InspectResponse` after `ProposalScaffold`:

```go
	AuthoringPacket     AuthoringPacket   `json:"authoring_packet"`
```

Then replace the end of `Inspect`:

```go
	response.ProposalScaffold = buildProposalScaffold(response, req)
	return response, nil
```

with:

```go
	response.ProposalScaffold = buildProposalScaffold(response, req)
	response.AuthoringPacket = buildAuthoringPacket(response, req)
	return response, nil
```

- [ ] **Step 3: Run the normal packet test**

Run:

```bash
gofmt -w internal/author/authoring_packet.go internal/author/inspect.go internal/author/inspect_test.go
go test ./internal/author -run TestInspectIncludesAuthoringPacketForProposalPath -count=1
```

Expected: PASS.

- [ ] **Step 4: Run existing inspect tests**

Run:

```bash
go test ./internal/author -run TestInspect -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit normal packet implementation**

```bash
git add internal/author/authoring_packet.go internal/author/inspect.go internal/author/inspect_test.go
git commit -m "author: add inspect authoring packet"
```

## Task 3: Review-Only Overlap Packet

**Files:**
- Modify: `internal/author/inspect_test.go`
- Modify: `internal/author/authoring_packet.go`
- Test: `internal/author/inspect_test.go`

- [ ] **Step 1: Write the failing overlap packet test**

Add these assertions to the end of `TestInspectProposalScaffoldUsesReviewOnlyForUnresolvedOverlap`:

```go
	packet := result.AuthoringPacket
	if packet.State != "review_only" {
		t.Fatalf("packet state = %q, want review_only: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_review_only_proposal" {
		t.Fatalf("packet recommended_action = %q, want write_review_only_proposal: %#v", packet.RecommendedAction, packet)
	}
	if !packet.ReviewOnly {
		t.Fatalf("packet should be review-only: %#v", packet)
	}
	if packet.CandidatePath != "" {
		t.Fatalf("review-only packet candidate_path = %q, want empty", packet.CandidatePath)
	}
	if packet.Commands.VerifyCandidate != "" {
		t.Fatalf("review-only packet verify command = %q, want empty", packet.Commands.VerifyCandidate)
	}
	if packet.ProposalPath != proposal.ProposedShape.Path {
		t.Fatalf("proposal_path = %q, want review-only scaffold path %q", packet.ProposalPath, proposal.ProposedShape.Path)
	}
	if !containsText(packet.HumanReviewQuestions, "Is the proposed knowledge goal correct and specific enough?") {
		t.Fatalf("packet should preserve scaffold review questions: %#v", packet.HumanReviewQuestions)
	}
```

- [ ] **Step 2: Run the overlap packet test**

Run:

```bash
go test ./internal/author -run TestInspectProposalScaffoldUsesReviewOnlyForUnresolvedOverlap -count=1
```

Expected: PASS if Task 2 already handles review-only from `proposal_scaffold`; if it fails, adjust `buildAuthoringPacket` so `reviewOnly` is driven by `proposal.ProposedShape.ArtifactState == "review_only"`.

- [ ] **Step 3: Commit overlap packet coverage**

```bash
git add internal/author/inspect_test.go internal/author/authoring_packet.go
git commit -m "test: cover review-only authoring packet"
```

## Task 4: Missing-Content Review-Only Path

**Files:**
- Modify: `internal/author/inspect_test.go`
- Modify: `internal/author/inspect.go`
- Modify: `internal/author/authoring_packet.go`
- Test: `internal/author/inspect_test.go`

- [ ] **Step 1: Write the failing missing-content inspect test**

Add this test after `TestInspectProposalScaffoldUsesReviewOnlyForUnresolvedOverlap`:

```go
func TestInspectUsesReviewOnlyPacketWhenPersonalConventionContentIsMissing(t *testing.T) {
	root := t.TempDir()
	writeAuthorRegistry(t, root)

	result, err := Inspect(root, InspectRequest{
		Project: "mall-api",
		Goal:    "I have a personal convention for this project. Preserve it for future agents without making it global truth.",
		Tags:    []string{"personal", "project-convention"},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	proposal := result.ProposalScaffold
	if proposal.ProposedShape.ArtifactState != "review_only" {
		t.Fatalf("artifact_state = %q, want review_only: %#v", proposal.ProposedShape.ArtifactState, proposal.ProposedShape)
	}
	if proposal.Scope.Distribution != "personal" {
		t.Fatalf("distribution = %q, want personal", proposal.Scope.Distribution)
	}
	if proposal.HumanReview.CandidateWriteApproved {
		t.Fatalf("missing-content scaffold must not approve candidate writing: %#v", proposal.HumanReview)
	}
	if len(proposal.CandidateFiles) != 0 {
		t.Fatalf("missing-content scaffold should not contain candidate files: %#v", proposal.CandidateFiles)
	}

	packet := result.AuthoringPacket
	if packet.State != "review_only" {
		t.Fatalf("packet state = %q, want review_only: %#v", packet.State, packet)
	}
	if packet.RecommendedAction != "write_review_only_proposal" {
		t.Fatalf("packet action = %q, want write_review_only_proposal: %#v", packet.RecommendedAction, packet)
	}
	if !containsText(packet.HumanReviewQuestions, "What exact convention should future agents preserve?") {
		t.Fatalf("packet should ask for exact convention content: %#v", packet.HumanReviewQuestions)
	}
	if !containsText(packet.StopConditions, "Do not write candidate files until human_review.candidate_write_approved is true.") {
		t.Fatalf("packet missing candidate stop condition: %#v", packet.StopConditions)
	}
}
```

- [ ] **Step 2: Run the missing-content test**

Run:

```bash
go test ./internal/author -run TestInspectUsesReviewOnlyPacketWhenPersonalConventionContentIsMissing -count=1
```

Expected: FAIL because the scaffold still defaults to candidate shape for this request.

- [ ] **Step 3: Implement missing-content detection**

In `internal/author/inspect.go`, change:

```go
	reviewOnly := len(overlapIDs) > 0
```

to:

```go
	missingContent := requestLooksMissingSubstantiveContent(req)
	reviewOnly := len(overlapIDs) > 0 || missingContent
```

Still in `buildProposalScaffold`, after the normal proposal is constructed and before the existing `if reviewOnly` block, add:

```go
	if missingContent {
		proposal.Scope.Distribution = "personal"
		proposal.SourceProfile.Assumptions = []string{"The exact convention content was not provided in the inspect request."}
		proposal.SourceProfile.OpenQuestions = []string{"What exact convention should future agents preserve?"}
		proposal.SourceProfile.Claims = append(proposal.SourceProfile.Claims, SourceClaimV2{
			Claim:          "The exact actionable convention content is missing and must be reviewed before candidate writing.",
			Kind:           "recommendation",
			Trust:          "synthesized",
			Source:         []string{"argos author inspect"},
			RequiresReview: true,
		})
		proposal.FutureUse.MissingNeeds = []string{"Exact convention content."}
		proposal.HumanReview.ReviewQuestions = []string{"What exact convention should future agents preserve?"}
	}
```

Add this helper near `requestLooksConsumerFacing`:

```go
func requestLooksMissingSubstantiveContent(req InspectRequest) bool {
	text := strings.ToLower(strings.Join(append([]string{
		req.Goal,
		req.FutureTask,
		req.Query,
	}, append(append([]string{}, req.Tags...), req.Files...)...), " "))
	if text == "" {
		return false
	}
	hasConventionIntent := strings.Contains(text, "personal convention") ||
		strings.Contains(text, "project convention") ||
		strings.Contains(text, "project-convention") ||
		strings.Contains(text, "preserve it for future agents")
	hasNoConcreteContent := strings.Contains(text, "i have a personal convention") ||
		strings.Contains(text, "preserve it") ||
		strings.Contains(text, "without making it global truth")
	return hasConventionIntent && hasNoConcreteContent
}
```

- [ ] **Step 4: Make packet reason specific for missing content**

In `internal/author/authoring_packet.go`, inside `buildAuthoringPacket` after the `if reviewOnly` block, add:

```go
	if requestLooksMissingSubstantiveContent(req) {
		reason = "Substantive convention content is missing; write a review-only proposal and ask for the exact convention before candidate writing."
	}
```

Then ensure this assignment happens before constructing `packet`.

- [ ] **Step 5: Run missing-content and review-only tests**

Run:

```bash
gofmt -w internal/author/inspect.go internal/author/authoring_packet.go internal/author/inspect_test.go
go test ./internal/author -run 'TestInspect(UsesReviewOnlyPacketWhenPersonalConventionContentIsMissing|ProposalScaffoldUsesReviewOnlyForUnresolvedOverlap|IncludesAuthoringPacketForProposalPath)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit missing-content behavior**

```bash
git add internal/author/inspect.go internal/author/authoring_packet.go internal/author/inspect_test.go
git commit -m "author: make missing content inspect review-only"
```

## Task 5: CLI JSON Contract

**Files:**
- Modify: `internal/cli/cli_test.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Extend CLI inspect JSON test**

In `TestRunAuthorInspectReturnsAuthoringPolicyJSON`, extend the local result struct:

```go
	var result struct {
		Project string `json:"project"`
		Policy  struct {
			PriorityMust string `json:"priority_must"`
		} `json:"policy"`
		AuthoringPacket struct {
			State             string `json:"state"`
			RecommendedAction string `json:"recommended_action"`
			CandidateAllowed  bool   `json:"candidate_allowed"`
			Commands          struct {
				WriteProposal   string `json:"write_proposal"`
				VerifyCandidate string `json:"verify_candidate"`
			} `json:"commands"`
		} `json:"authoring_packet"`
		RecommendedNextSteps []struct {
			Step string `json:"step"`
		} `json:"recommended_next_steps"`
	}
```

After the existing `RecommendedNextSteps` assertion, add:

```go
	if result.AuthoringPacket.State != "ready_for_proposal" {
		t.Fatalf("unexpected authoring packet state: %s", stdout.String())
	}
	if result.AuthoringPacket.RecommendedAction != "write_proposal" {
		t.Fatalf("unexpected authoring packet action: %s", stdout.String())
	}
	if result.AuthoringPacket.CandidateAllowed {
		t.Fatalf("inspect must not authorize candidate writing: %s", stdout.String())
	}
	if result.AuthoringPacket.Commands.WriteProposal == "" || result.AuthoringPacket.Commands.VerifyCandidate == "" {
		t.Fatalf("authoring packet should include write and verify command guidance: %s", stdout.String())
	}
```

- [ ] **Step 2: Run CLI inspect test**

Run:

```bash
go test ./internal/cli -run TestRunAuthorInspectReturnsAuthoringPolicyJSON -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit CLI contract coverage**

```bash
git add internal/cli/cli_test.go
git commit -m "test: assert cli inspect authoring packet"
```

## Task 6: Docs And Capture Skill Guidance

**Files:**
- Modify: `README.md`
- Modify: `skills/capture-knowledge/SKILL.md`
- Test: `go test ./...`

- [ ] **Step 1: Update README authoring flow**

In the `Agent Knowledge Authoring` section of `README.md`, replace:

```markdown
2. The agent inspects existing knowledge and authoring policy:
```

with:

```markdown
2. The agent inspects existing knowledge, authoring policy, and the agent-ready
   authoring packet:
```

After the inspect command block, add:

```markdown
The `authoring_packet` in the inspect response gives the agent the next safe
action, suggested proposal and candidate paths, stop conditions, human review
questions, and the subsequent `author verify` command. Inspect never approves
candidate writing; candidate files still require human review of the proposal.
```

- [ ] **Step 2: Update capture-knowledge skill**

In `skills/capture-knowledge/SKILL.md`, after the `argos author inspect` command block, add:

```markdown
Read `authoring_packet` before writing files. Treat it as the operational
control packet for this authoring attempt:

- follow `recommended_action` for the next safe step;
- use `proposal_path` for the proposal artifact;
- use `candidate_path` only as a proposed path, not as write approval;
- obey every `stop_conditions` entry;
- ask the `human_review_questions` before candidate writing when correctness or
  authorization depends on the answer;
- run `commands.verify_candidate` only after candidate writing is approved and
  candidate files exist.
```

- [ ] **Step 3: Run full tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS for every package.

- [ ] **Step 4: Commit docs and skill guidance**

```bash
git add README.md skills/capture-knowledge/SKILL.md
git commit -m "docs: document inspect authoring packet"
```

## Task 7: Targeted Authoring Dogfood Verification

**Files:**
- No planned file edits
- Test: authoring dogfood targeted cases

- [ ] **Step 1: Build a fresh Argos binary**

Run:

```bash
ROUND_ROOT=$(mktemp -d -t argos-authoring-packet.XXXXXX)
mkdir -p "$ROUND_ROOT/packets" "$ROUND_ROOT/reports" "$ROUND_ROOT/evaluations"
GOCACHE="$ROUND_ROOT/gocache" go build -o "$ROUND_ROOT/argos" ./cmd/argos
printf '%s\n' "$ROUND_ROOT"
```

Expected: command prints a temp path like `/tmp/argos-authoring-packet.ABC123`.

- [ ] **Step 2: Generate targeted packets**

Run:

```bash
for case_id in case-001 case-003 case-008; do
  "$ROUND_ROOT/argos" dogfood authoring packet \
    --case "$case_id" \
    --workspace "$ROUND_ROOT/$case_id" \
    --argos-binary "$ROUND_ROOT/argos" \
    > "$ROUND_ROOT/packets/$case_id.md"
done
```

Expected: each workspace is seeded and each packet file is written.

- [ ] **Step 3: Inspect packet JSON directly for the three scenarios**

Run:

```bash
cd "$ROUND_ROOT/case-003"
"$ROUND_ROOT/argos" author inspect --json \
  --project "mall-api" \
  --goal "This project is called by many people. I want future agents to understand the business points and interfaces so they can help other developers use it." \
  --files internal/api/README.md \
  --tags api \
  --tags consumer \
  --tags business-capability \
  > "$ROUND_ROOT/evaluations/case-003-inspect.json"
rg '"authoring_packet"|"state"|"recommended_action"|"candidate_allowed"' "$ROUND_ROOT/evaluations/case-003-inspect.json"
```

Expected: output includes `authoring_packet`, `ready_for_proposal`, `write_proposal`, and `candidate_allowed` false.

- [ ] **Step 4: Run existing focused test suites**

Run:

```bash
go test ./internal/author ./internal/cli ./internal/authoringdogfood -count=1
```

Expected: PASS.

- [ ] **Step 5: Final verification**

Run:

```bash
go test ./... -count=1
git diff --check
```

Expected: tests pass and `git diff --check` has no output.

## Self-Review

- Spec coverage: Tasks 1-5 implement `authoring_packet`, states, actions, candidate permission, review-only behavior, CLI compatibility, and missing-content behavior. Task 6 covers README and skill rollout. Task 7 covers targeted verification.
- Placeholder scan: no task uses incomplete implementation instructions, empty report templates, or value slots.
- Type consistency: `AuthoringPacket`, `AuthoringSourceWork`, and `AuthoringCommands` are defined before subsequent tasks use them. `InspectResponse.AuthoringPacket` uses the same type and JSON key throughout.
