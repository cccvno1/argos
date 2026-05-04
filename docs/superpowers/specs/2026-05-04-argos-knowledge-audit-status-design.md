# Argos Knowledge Audit And Status Design

Date: 2026-05-04

## Goal

Add a small review-assistance layer over the Git-native knowledge store and
provenance records.

The goal is not to make Argos the reviewer. The goal is to let agents and
humans answer these questions with stable, machine-readable commands:

- What provenance records exist?
- Which write attempts are blocked, ready to publish, or already published?
- What evidence is missing or stale?
- Which official knowledge files lack provenance evidence?
- What should a personal user or PR reviewer inspect next?

This should serve both personal local use and team PR review:

```text
personal mode = user reviews Argos audit/status output before publishing
team mode = reviewer inspects knowledge diff plus provenance diff before merge
Argos = evidence organizer and preflight checker
PR / human decision = authority boundary
```

## Current State

Argos already has:

- `argos knowledge design --json`
- `argos knowledge check --json`
- `argos provenance start --json`
- `argos provenance record-decision --json`
- `argos provenance record-check --json`
- `argos provenance verify --json`
- `argos knowledge publish --provenance`
- Git-tracked provenance under `knowledge/.inbox/provenance/` and
  `knowledge/provenance/`

The missing layer is a stable way to list, classify, and summarize this evidence
for review.

## Non-Goals

This round does not add:

- GitHub, GitLab, or other PR API integration.
- CODEOWNERS parsing or permission checks.
- Cryptographic signing.
- A human-facing review UI.
- Remote object storage.
- MCP write/provenance tools.
- Auto-publish behavior.

MCP provenance tools can come later after CLI JSON contracts settle.

## Concepts

### Provenance Status

`provenance status` answers one-record questions.

It should load one provenance record, recompute the same evidence that
`provenance verify` cares about, and explain whether the record is blocked,
ready, published, or inconsistent.

### Knowledge Audit

`knowledge audit` answers repository-level questions.

It should scan designs, inbox drafts, official knowledge, and provenance records
to produce a review queue. It is a dashboard-shaped JSON response, not a gate.

### Result Vocabulary

Use simple result values:

- `pass`: evidence is complete for the requested state.
- `blocked`: required evidence is missing, stale, or failing.
- `warning`: evidence is usable but reviewer attention is needed.
- `problem`: filesystem or provenance shape is inconsistent.

Use actionable categories for audit items:

- `needs_design_decision`
- `needs_draft_write_decision`
- `needs_draft`
- `needs_check`
- `check_failed`
- `needs_publish_decision`
- `ready_to_publish`
- `published`
- `published_inconsistent`
- `official_missing_provenance`
- `orphan_provenance`

## Commands

### `argos provenance list --json`

List provenance records without loading full check bodies.

Flags:

- `--json`: required.
- `--state draft|published|all`: optional, default `all`.
- `--project <project>`: optional.
- `--knowledge-id <id>`: optional.

Output:

```json
{
  "records": [
    {
      "provenance_id": "prov-20260504-redis-cache-a1b2c3d4",
      "state": "draft",
      "path": "knowledge/.inbox/provenance/prov-20260504-redis-cache-a1b2c3d4",
      "project": "mall-api",
      "knowledge_id": "package:mall-api.redis-cache.v1",
      "kind": "package",
      "design_path": "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
      "draft_path": "knowledge/.inbox/packages/mall-api/redis-cache",
      "official_path": "knowledge/packages/mall-api/redis-cache",
      "latest_check_result": "pass",
      "created_at": "2026-05-04T00:00:00Z",
      "published_at": ""
    }
  ]
}
```

Errors:

- Reject missing `--json`.
- Reject invalid `--state`.
- Reject unreadable or malformed provenance records with a nonzero exit.

### `argos provenance status --json --provenance <id-or-path>`

Return one provenance record plus computed evidence status.

Flags:

- `--json`: required.
- `--provenance <id-or-path>`: required.

Output:

```json
{
  "result": "blocked",
  "provenance_id": "prov-20260504-redis-cache-a1b2c3d4",
  "state": "draft",
  "path": "knowledge/.inbox/provenance/prov-20260504-redis-cache-a1b2c3d4",
  "subject": {
    "project": "mall-api",
    "knowledge_id": "package:mall-api.redis-cache.v1",
    "design_path": "knowledge/.inbox/designs/mall-api/redis-cache/design.json",
    "draft_path": "knowledge/.inbox/packages/mall-api/redis-cache",
    "official_path": "knowledge/packages/mall-api/redis-cache"
  },
  "evidence": {
    "design_bound": "pass",
    "draft_bound": "pass",
    "latest_check": "pass",
    "design_decision": "pass",
    "draft_write_decision": "pass",
    "publish_decision": "missing",
    "official_target": "not_published"
  },
  "actions": [
    "record publish decision before publishing"
  ],
  "findings": [
    {
      "severity": "blocked",
      "category": "needs_publish_decision",
      "message": "publish decision is missing"
    }
  ]
}
```

Status must reuse the same safety semantics as `provenance verify`:

- Design decision binds to the current design hash.
- Draft-write decision binds to the current design hash.
- Publish decision binds to current design, draft tree, and latest passing check.
- Changed design, draft, or check evidence blocks publish readiness.
- Published provenance must point to an existing official target.
- Published official content must still match the published record's subject and
  expected path.

`provenance status` should not mutate files.

### `argos knowledge audit --json`

Return a repository-level review queue.

Flags:

- `--json`: required.
- `--project <project>`: optional.
- `--include-published`: optional, default false.

Default behavior should focus on open work and problems:

- draft provenance records that are not ready to publish
- draft provenance records ready to publish
- official knowledge without provenance
- published provenance records with inconsistencies
- orphan provenance records whose design or draft paths no longer exist

With `--include-published`, include healthy published records too.

Output:

```json
{
  "result": "warning",
  "summary": {
    "open": 3,
    "ready_to_publish": 1,
    "blocked": 2,
    "problems": 1,
    "published": 12,
    "official_missing_provenance": 1
  },
  "items": [
    {
      "category": "ready_to_publish",
      "severity": "warning",
      "provenance_id": "prov-20260504-redis-cache-a1b2c3d4",
      "project": "mall-api",
      "knowledge_id": "package:mall-api.redis-cache.v1",
      "path": "knowledge/.inbox/provenance/prov-20260504-redis-cache-a1b2c3d4",
      "action": "review evidence and run argos knowledge publish --provenance prov-20260504-redis-cache-a1b2c3d4"
    }
  ]
}
```

Result calculation:

- `pass`: no open items, no problems, and no missing provenance.
- `warning`: open review queue exists but no structural problems.
- `blocked`: at least one item cannot proceed because required evidence is
  missing, stale, or failing.
- `problem`: malformed records, unsafe paths, duplicate provenance IDs, or
  official knowledge with contradictory provenance.

## Review Boundaries

Argos output must not imply that Argos itself approved knowledge, completed
human review, approved a PR, or allowed a merge.

Preferred language:

- "decision recorded"
- "evidence complete"
- "ready to publish"
- "needs reviewer attention"
- "official knowledge lacks provenance"

Avoid wording that assigns approval authority to Argos or to the audit command.

Team mode wording must be explicit:

```text
Argos can prepare a candidate and summarize evidence. The repository review
process decides whether the branch is accepted.
```

## Implementation Shape

Add a small service layer rather than putting audit logic directly in CLI:

```text
internal/provenance
  List(root, filter) -> []Loaded
  Status(root, idOrPath) -> StatusResult

internal/audit
  Knowledge(root, request) -> AuditResult
```

`internal/audit` may depend on:

- `internal/provenance`
- `internal/knowledge`
- `internal/knowledgewrite`
- registry/path helpers as needed

It should not depend on CLI formatting.

CLI should stay thin:

- parse flags
- require `--json`
- call service
- print JSON
- map errors to exit codes

## Test Strategy

Use TDD and cover the public behavior first.

Focused provenance tests:

- `provenance list` returns draft and published records.
- `provenance list --state draft` excludes published records.
- `provenance list --project mall-api` filters records.
- `provenance status` reports `needs_publish_decision` before publish decision.
- `provenance status` reports `ready_to_publish` after design, draft-write,
  check, and publish decisions are complete.
- `provenance status` blocks when design changes after a decision.
- `provenance status` blocks when draft changes after publish decision.
- `provenance status` reports published records as `pass`.

Focused audit tests:

- Empty fresh workspace returns `pass` with empty items.
- Draft provenance without decisions appears as blocked.
- Passing check without publish decision appears as `needs_publish_decision`.
- Complete draft provenance appears as `ready_to_publish`.
- Official knowledge without provenance appears as `official_missing_provenance`.
- Malformed provenance appears as `problem`.
- `--project` filters audit items.
- `--include-published` includes healthy published records.

End-to-end CLI test:

1. Initialize workspace.
2. Add project.
3. Run design.
4. Write design JSON.
5. Start provenance.
6. Record design and draft-write decisions.
7. Write draft.
8. Record check.
9. Run audit and see `needs_publish_decision`.
10. Record publish decision.
11. Run status and see ready to publish.
12. Publish.
13. Run audit with `--include-published` and see published pass.

## Documentation Updates

Update README and `skills/capture-knowledge/SKILL.md` after implementation:

- Add `provenance list` and `provenance status` to agent/internal commands.
- Add `knowledge audit` to the review-assistance section.
- Clarify personal and team review flows.
- Keep provenance wording distinct from PR review.

## Release Criteria

This module is ready when:

- CLI JSON outputs are stable and covered by tests.
- Audit/status never mutate files.
- Audit/status do not require an index.
- Existing design/check/publish flows keep working.
- `go test ./... -count=1` passes.
- `git diff --check` passes.
