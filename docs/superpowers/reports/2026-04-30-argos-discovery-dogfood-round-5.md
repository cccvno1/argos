# Argos Discovery Dogfood Round 5

Date: 2026-04-30
Workspace: `/tmp/argos-round5-full-n40XmC`
Binary: `/tmp/argos-dogfood`

## Goal

Validate shared-knowledge Discovery semantics after migrating to
`coverage_gaps`.

## Cases

### strong_shared_knowledge_no_coverage_gaps

- Actual coverage: `strong`, confidence `0.886`.
- Action policy: `authority=strong`, `load=recommended`, `cite=after_loaded_and_used`, `claim=allowed`.
- Recall: semantic `disabled`; reason `semantic provider is not configured`.
- Coverage gaps: none.
- Discovered IDs: `rule:backend.auth-refresh.v1`, `decision:backend.session-renewal.v1`, `package:backend.auth-refresh.v1`, `lesson:backend.auth-debug.v1`, `reference:backend.global-refresh.v1`.
- Loaded IDs: none.
- Cited IDs: none.
- Next calls: `get_knowledge_item` for `rule:backend.auth-refresh.v1`; `cite_knowledge` for loaded-and-used knowledge.
- Progressive disclosure: `pass`; discovered items were summaries with `full_body_available=true` and `load_tool=get_knowledge_item`.
- Attribution boundary: `not-applicable`; strong coverage had no uncovered needs.
- Coverage gaps not cited: `not-applicable`; no coverage gaps were present.
- No Discovery-triggered upload/capture: `pass`; Discovery recommended only load/cite tools.
- Result: `pass`.

### partial_shared_knowledge_requires_attribution_boundary

- Actual coverage: `partial`, confidence observed by runner as partial coverage for session renewal debugging.
- Action policy: `authority=partial`, `load=allowed`, `cite=after_loaded_and_used`, `claim=must_separate_argos_backed_and_general_reasoning`.
- Recall: semantic `disabled`; reason `semantic provider is not configured`.
- Coverage gaps: one gap with `source=partial_match`.
- Discovered IDs: included `lesson:backend.auth-debug.v1`.
- Loaded IDs: `lesson:backend.auth-debug.v1`.
- Cited IDs: none in final dogfood report; loaded ID was identified as the only Argos-backed source.
- Next calls: `get_knowledge_item` for `lesson:backend.auth-debug.v1`.
- Progressive disclosure: `pass`; full body was loaded only after a routed `get_knowledge_item` call was allowed.
- Attribution boundary: `pass`; runner separated the loaded token-rotation-log lesson from general debugging reasoning.
- Coverage gaps not cited: `pass`; coverage gap was not cited as knowledge.
- No Discovery-triggered upload/capture: `pass`; no upload, capture, inbox, or proposal workflow was started.
- Result: `pass`.

### weak_shared_knowledge_not_argos_backed

- Actual coverage: `weak`, with weak shared-knowledge match.
- Action policy: `authority=weak`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`.
- Recall: semantic `disabled`; reason `semantic provider is not configured`.
- Coverage gaps: one gap with `source=weak_match` and `argos_backed=false`.
- Discovered IDs: weakly related summary items were returned.
- Loaded IDs: none.
- Cited IDs: none.
- Next calls: none.
- Progressive disclosure: `pass`; only summaries were inspected.
- Attribution boundary: `pass`; runner made no Argos-backed claim.
- Coverage gaps not cited: `pass`; coverage gap was not cited or loaded.
- No Discovery-triggered upload/capture: `pass`; no upload or capture action was started.
- Result: `pass`.

### none_shared_knowledge_can_continue_without_argos_claim

- Actual coverage: `none`.
- Action policy: `authority=none`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`.
- Recall: semantic `disabled`; reason `semantic provider is not configured`.
- Coverage gaps: one gap with `source=unmatched_intent` and `argos_backed=false`.
- Discovered IDs: none.
- Loaded IDs: none.
- Cited IDs: none.
- Next calls: none.
- Progressive disclosure: `pass`; no item bodies were available or loaded.
- Attribution boundary: `pass`; runner confirmed normal work can continue without Argos-backed attribution.
- Coverage gaps not cited: `pass`; coverage gap was not cited or treated as knowledge.
- No Discovery-triggered upload/capture: `pass`; no upload, capture, inbox, or proposal workflow was started.
- Result: `pass`.

### filter_excluded_shared_knowledge_boundary

- Actual coverage: `none`.
- Action policy: `authority=none`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`.
- Recall: semantic `disabled`; reason `semantic provider is not configured`.
- Coverage gaps: one gap with `source=filter_excluded` and `argos_backed=false`.
- Discovered IDs: none.
- Loaded IDs: none.
- Cited IDs: none.
- Next calls: none.
- Progressive disclosure: `pass`; no filtered-out knowledge was loaded.
- Attribution boundary: `pass`; runner made no Argos-backed claim under none coverage.
- Coverage gaps not cited: `pass`; coverage gap was not cited or loaded.
- No Discovery-triggered upload/capture: `pass`; no upload or capture workflow was started.
- Result: `pass`.

## Summary

- Pass count: 5.
- Review-needed count: 0.
- Fail count: 0.
- Follow-up: run the planned global naming audit. `action_policy.reason` and `coverage.missing_knowledge_hints` still contain older wording in some responses and should be addressed in that audit rather than expanding this migration.
