# Argos Write Release Hardening Design

Date: 2026-05-04

## Goal

Bring the write side to first-release standard without adding the audit trail
module yet. A fresh workspace should let an agent register a project, design
knowledge, write a reviewed draft, check it, publish it, rebuild the index, and
find the published knowledge back without hand-editing registry YAML or relying
on ambiguous storage state.

The release-standard write path is:

```text
argos init
-> argos project add
-> argos knowledge design
-> human reviews design JSON
-> agent writes inbox draft
-> argos knowledge check
-> human approves publish
-> argos knowledge publish
-> argos index
-> argos knowledge find
```

## Scope

This round hardens the write module itself:

- Project registration commands for agent-operated fresh workspaces.
- Storage contract checks for inbox and official knowledge roots.
- Publish contract checks that keep `publish` as the standard transition from
  draft knowledge to official active knowledge.
- Documentation and skill updates that describe the supported write contract.
- Readiness report updates that classify write dogfood as a source-tree internal
  harness instead of an installed-binary user contract.

## Non-Goals

This round does not add the audit and review-history module. Specifically it
does not add:

- `knowledge/.reviews/...` or another persistent review store.
- `approved_by`, reviewer identity, or multi-person approval semantics.
- A permissions system.
- Remote service storage.
- Database-backed knowledge storage.
- Long-term audit trail retention rules.

Those belong in the next design round.

## Current State

The write command chain already exists:

- `argos knowledge design --json` produces a `knowledge.design.v1` template.
- `argos knowledge check --json` validates design, draft, policy, and
  findability.
- `argos knowledge publish` requires publish approval, moves inbox drafts to the
  official root, converts published knowledge to `status: active`, and validates
  the official path.
- MCP exposes `argos_design_knowledge` and `argos_check_knowledge`.
- The capture skill describes the design, write, check, and publish flow.
- Write dogfood/golden tests exercise real authoring scenarios.

The remaining write-release gaps are:

- Fresh workspaces still need manual `knowledge/projects.yaml` edits.
- The storage contract is not consistently enforced by `validate`.
- The distinction between source-tree dogfood harness and installed user
  commands is not explicit enough.
- The documentation does not yet present project registration, storage rules,
  and publish rules as one release contract.

## Project Registry Commands

Add a top-level `project` command group:

```bash
argos project add --id <id> --name <name> --path <path> \
  --tech-domain <domain> --business-domain <domain>

argos project list --json
```

`project add` writes `knowledge/projects.yaml` through structured YAML handling.
It must:

- Require `--id`, `--name`, and `--path`.
- Accept repeated `--tech-domain` and `--business-domain`.
- Trim empty repeated flags.
- Reject duplicate project IDs.
- Reject unknown tech or business domains by checking `knowledge/domains.yaml`.
- Preserve existing projects.
- Print a concise success message for CLI use.

`project list --json` returns the registered projects in a stable JSON shape:

```json
{
  "projects": [
    {
      "id": "mall-api",
      "name": "Mall API",
      "path": "services/mall-api",
      "tech_domains": ["backend"],
      "business_domains": ["account"]
    }
  ]
}
```

The command is intentionally simple. Domain creation is outside this round;
`knowledge/domains.yaml` remains the explicit place to define allowed domain
vocabulary.

## Storage Contract

Argos keeps a filesystem-first knowledge store.

Inbox draft roots:

```text
knowledge/.inbox/items/
knowledge/.inbox/packages/
```

Official roots:

```text
knowledge/items/
knowledge/packages/
```

Storage rules:

- Inbox knowledge must use `status: draft`.
- Official knowledge must not use `status: draft`.
- Official published knowledge normally uses `status: active`.
- `status: deprecated` is allowed only in official roots.
- Package directories are indexed through `KNOWLEDGE.md`.
- Markdown files under `knowledge/packages/**` are loadable only when their
  basename is `KNOWLEDGE.md`; supporting package files are on-demand references.

This keeps draft state, official state, and query state aligned. It also prevents
agents from bypassing `publish` by placing draft metadata directly under the
official roots.

## Validation Behavior

`argos validate` should enforce the storage contract according to its scope:

- `argos validate` loads official knowledge and rejects official `status: draft`.
- `argos validate --inbox` loads inbox knowledge and rejects non-draft statuses.
- `argos validate --path <path>` derives the storage scope from the path:
  - `knowledge/.inbox/...` uses inbox rules.
  - `knowledge/items/...` or `knowledge/packages/...` uses official rules.
  - other paths keep the existing item/package validation but report that the
    path is outside standard knowledge roots when applicable.

Validation errors should state the fix in terms an agent can act on:

- For inbox active knowledge: move through publish or set the draft back to
  `status: draft`.
- For official draft knowledge: publish from inbox or change official metadata to
  `status: active` after review.
- For unknown project: run `argos project list --json` and, if needed,
  `argos project add`.

## Publish Contract

`argos knowledge publish` remains the standard official transition. It must:

- Require a design path.
- Require `review.publish_approved`.
- Re-run `knowledge check` and require `result: pass`.
- Accept only inbox draft paths under `knowledge/.inbox/items/` or
  `knowledge/.inbox/packages/`.
- Refuse to overwrite existing official targets.
- Move the draft to the corresponding official root.
- Convert published metadata to `status: active`.
- Validate the official target after activation.
- Roll back the moved draft if activation or final official validation fails.

This round does not add a separate persisted audit record for publication. The
design JSON plus check/publish gates are the write contract for first release.

## Documentation And Skill Contract

README and `skills/capture-knowledge/SKILL.md` should describe one path:

1. Register or list the project with `argos project`.
2. Design knowledge with `argos knowledge design`.
3. Persist and review the design JSON.
4. Write inbox draft knowledge only after draft-write approval.
5. Check the draft.
6. Publish only after explicit publish approval.
7. Rebuild the index and find the knowledge back.

The docs should explicitly name:

- Storage contract.
- Publish contract.
- Project registry setup through CLI commands.
- Dogfood write harness as source-tree internal release validation.
- Audit/review-history as a later module.

## Dogfood Decision

For first release, write dogfood is a source-tree internal harness. It is allowed
to depend on repository `testdata`. It is not an installed-binary user feature
and should not block write release readiness.

The CLI can keep `dogfood write` available, but docs and readiness reports must
not imply that normal users or downstream agents need it to write knowledge.

## Testing

Implementation must add focused tests for:

- `project add` creates a project in `knowledge/projects.yaml`.
- `project add` rejects duplicate IDs.
- `project add` rejects unknown tech and business domains.
- `project list --json` returns stable JSON.
- A fresh workspace can complete project registration, design, check, publish,
  index, and findback without hand-editing `knowledge/projects.yaml`.
- `validate` rejects official `status: draft`.
- `validate --inbox` rejects inbox `status: active`.
- `validate --path` applies inbox rules for inbox paths and official rules for
  official paths.
- `knowledge publish` still rejects missing design, missing publish approval,
  failed check, existing target, and non-inbox paths.
- Published knowledge remains findable with `status: active`.
- README and the capture skill do not reintroduce removed write vocabulary.

Final verification:

```bash
go test ./... -count=1
git diff --check
```

## Acceptance Criteria

The write side reaches release standard when all of the following are true:

- A fresh workspace can use CLI commands, not hand-written project YAML, to get
  through the full write-to-query loop.
- Inbox and official roots cannot silently carry the wrong status semantics.
- `publish` is the only supported inbox-to-official transition in the standard
  flow.
- Documentation and skill instructions teach the same storage and publish
  contract that the CLI enforces.
- The global readiness report can classify write as release-ready, with audit
  history explicitly deferred to the next module.
