# Argos Write Dogfood Report

Case: `<case-handle>`
Runner Session: `<fresh-session-id>`
Workspace: `<workspace-path>`

## Inputs

- User request:
- Project:
- Context hints:
- Available sources:
- Constraints:

## Write Guidance

- State: `<replace-with-state>`
- Next action: `<replace-with-next-action>`
- Design path: `<replace-with-workspace-relative-design-path>`
- Draft path: `<replace-with-workspace-relative-draft-path-or-none>`
- Draft allowed: `<replace-with-true-or-false>`
- Design only: `<replace-with-true-or-false>`
- Check result: `<replace-with-pass-fail-review-needed-or-not-run>`

Use `none` for an intentionally absent draft path and `not-run` when check is intentionally skipped.

## Artifacts

- Design path: `knowledge/.inbox/designs/example/design.json`
- Draft path: `knowledge/.inbox/packages/example`
- Check result: `<replace-with-pass-fail-review-needed-or-not-run>`

Artifact paths must be workspace-relative paths, not absolute filesystem paths.

## Review Decisions

- Design approved: `<replace-with-true-or-false>`
- Draft write approved: `<replace-with-true-or-false>`
- Priority must approved: `<replace-with-true-or-false>`
- Official write approved: `<replace-with-true-or-false>`
- Publish approved: `<replace-with-true-or-false>`

## Guards

- Design reviewed before draft write: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Sources and scope documented: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Future use documented: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Draft stayed in approved area: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Official knowledge unchanged: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Publish not run: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`
- Check run: `<replace-with-pass-fail-review-needed-not-applicable-or-not-run>`

## Result

Result: `<replace-with-pass-fail-or-review-needed>`

Readiness notes: `none | source review | design review | content-readiness review`

Result guidance:

- Use `pass` only when the draft is ready for human review inside the current approval boundary.
- Use `review-needed` when approval, authorization, source state, design state, findability, or substantive content still needs a human decision.
- Missing actionable knowledge content means `review-needed`.
- Unauthorized elevated priority means `review-needed` unless a boundary was violated, which is `fail`.
- Use `fail` when workflow boundaries were violated or an artifact cannot be inspected.

Notes:
