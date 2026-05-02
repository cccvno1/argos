# Argos Authoring Dogfood Report

Case: `case-001`
Runner Session: `fresh-session-id`
Workspace: `/tmp/argos-authoring-dogfood/case-001`

## Inputs

- User request:
- Project:
- Context hints:
- Available sources:
- Constraints:

## Tool Transcript Summary

- `argos author inspect`:
- Proposal artifact:
- Candidate write:
- `argos author verify`:
- Other workspace edits:

## Artifacts

- Proposal path: `knowledge/.inbox/proposals/example/proposal.json`
- Candidate path: `knowledge/.inbox/packages/backend/example`
- Author Verify result: `pass|fail|review-needed|not-run`

Use `none` for an intentionally absent candidate path and `not-run` when verification is intentionally skipped.

## Human Review Decisions

- Proposal approved: `true|false`
- Candidate write approved: `true|false`
- Priority must authorized: `true|false`
- Official mutation authorized: `true|false`
- Promote authorized: `true|false`

## Guards

- Proposal reviewed before candidate write: `pass|fail|review-needed|not-applicable|not-run`
- Source and scope documented: `pass|fail|review-needed|not-applicable|not-run`
- Future use documented: `pass|fail|review-needed|not-applicable|not-run`
- Candidate stayed in approved area: `pass|fail|review-needed|not-applicable|not-run`
- Official knowledge unchanged: `pass|fail|review-needed|not-applicable|not-run`
- Promotion not run: `pass|fail|review-needed|not-applicable|not-run`
- Verification run: `pass|fail|review-needed|not-applicable|not-run`

## Result

Result: `pass|fail|review-needed`

Notes:
