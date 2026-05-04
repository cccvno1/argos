# Argos Write Dogfood Round 0

Date: 2026-05-03
Run date: 2026-05-04

## Purpose

Validate the first-release write module flow:

intent -> knowledge design -> draft package -> check -> publish -> index -> query findback

## Result

Status: pass

The release-readiness smoke ran against a clean temporary workspace with the public CLI:

`argos init -> knowledge design -> write reviewed design/draft -> knowledge check -> knowledge publish --design -> index -> knowledge find`

| Case | Check | Publish | Findback |
| --- | --- | --- | --- |
| Redis best practices | pass | pass | `package:mall-api.redis-cache.v1` found |
| Go template standard | pass | pass | `package:mall-api.go-service-template.v1` found |
| Business interface knowledge | pass | pass | `package:mall-api.consumer-api.v1` found |

## Expanded Smoke

The expanded smoke reran the public CLI flow with eight positive scenarios, three design-boundary scenarios, and two guard-rejection scenarios.

| Group | Case | Expected | Result |
| --- | --- | --- | --- |
| Positive | Redis best practices | design, check, publish, index, findback | pass |
| Positive | Go template standard | design, check, publish, index, findback | pass |
| Positive | Business interface knowledge | design, check, publish, index, findback | pass |
| Positive | Database migration rules | design, check, publish, index, findback | pass |
| Positive | API error contract | design, check, publish, index, findback | pass |
| Positive | Frontend design token rules | design, check, publish, index, findback | pass |
| Positive | Release rollback runbook | design, check, publish, index, findback | pass |
| Positive | Agent code review checklist | design, check, publish, index, findback | pass |
| Boundary | Missing content design-only | no draft path | pass |
| Boundary | Unsafe draft path design-only | no draft path | pass |
| Boundary | Existing Redis requires decision | existing official match blocks draft writing | pass |
| Guard | Unapproved design check rejected | check fails | pass |
| Guard | Missing publish approval rejected | publish fails | pass |

## Required Cases

- Go template standard
- Redis best practices
- Business interface knowledge
- Missing content design-only
- Existing knowledge decision
- Draft check and query findback

## Fixes Made

- Tightened `knowledge design` existing-knowledge detection so generic index text, shared project, or broad domains such as `backend` do not force unrelated new knowledge into `design_only`.
- Treated registry domains as applicability context instead of standalone blocking evidence.
- Ignored generic knowledge-shape terms such as `rules`, `runbook`, `checklist`, `contract`, and `review` as standalone blocking evidence.
- Honored project scope when existing knowledge is project-specific, so one project's package does not block unrelated new knowledge for another project.
- Preserved the blocking behavior for clearly related existing knowledge, such as Redis cache requests matching an existing Redis cache package.
- Added regression tests for index-only generic matches, broad domain matches, and real related existing knowledge.

## Evidence

- Real CLI smoke output was written to `/tmp/argos-release-smoke/results.json`.
- Expanded CLI smoke output was written to `/tmp/argos-release-smoke-more/results.json`.
- Targeted regression command: `go test ./internal/knowledgewrite -count=1`.

## Next Action

Run full repository verification and commit the release-readiness smoke branch.
