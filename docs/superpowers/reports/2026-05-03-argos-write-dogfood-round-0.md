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

## Required Cases

- Go template standard
- Redis best practices
- Business interface knowledge
- Missing content design-only
- Existing knowledge decision
- Draft check and query findback

## Fixes Made

- Tightened `knowledge design` existing-knowledge detection so generic index text, shared project, or broad domains such as `backend` do not force unrelated new knowledge into `design_only`.
- Preserved the blocking behavior for clearly related existing knowledge, such as Redis cache requests matching an existing Redis cache package.
- Added regression tests for index-only generic matches, broad domain matches, and real related existing knowledge.

## Evidence

- Real CLI smoke output was written to `/tmp/argos-release-smoke/results.json`.
- Targeted regression command: `go test ./internal/knowledgewrite -count=1`.

## Next Action

Run full repository verification and commit the release-readiness smoke branch.
