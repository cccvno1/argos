# Argos Discovery Dogfood Round 4

Date: 2026-04-30
Workspace: `/tmp/argos-round4-full-v3jaVr`
Binary: `/tmp/argos-dogfood`

## Goal

Validate the first recall/gap workflow slice against fresh runner sessions:

- semantic recall is explicitly reported as disabled;
- lexical/metadata routing still works without semantic recall;
- none/weak/partial coverage exposes structured `gap_candidates`;
- `gap_candidates` remain candidate-only and are not cited or loaded;
- capture-knowledge is gated on user approval.

## Cases

### semantic_disabled_strong_still_works

Runner: `019ddd68-9d2b-7fb1-9db9-5ebf98b73968`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add refresh token endpoint`
- Query: `refresh token session renewal`
- Files: `internal/auth/session.go`

Observed Results:

- Actual coverage: `strong`, confidence `0.886`
- Action policy: `authority=strong`, `load=recommended`, `cite=after_loaded_and_used`, `claim=allowed`
- Recall: `semantic.status=disabled`, reason `semantic provider is not configured`
- Discovered IDs: `rule:backend.auth-refresh.v1`, `decision:backend.session-renewal.v1`, `package:backend.auth-refresh.v1`, `lesson:backend.auth-debug.v1`, `reference:backend.global-refresh.v1`
- Loaded IDs: none
- Cited IDs: none
- Missing knowledge hints: none
- Gap candidates: none
- Next calls: `get_knowledge_item` for `rule:backend.auth-refresh.v1`, then `cite_knowledge` for used loaded knowledge

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `not-applicable`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `not-applicable`
- Capture permission gate: `not-applicable`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

### semantic_disabled_none_has_gap_candidates

Runner: `019ddd68-9cda-7e62-ac36-ff1011f29ef4`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add payment webhook signature verification`
- Query: `payment webhook signature`

Observed Results:

- Actual coverage: `none`
- Action policy: `authority=none`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`
- Recall: `semantic.status=disabled`
- Discovered IDs: none
- Loaded IDs: none
- Cited IDs: none
- Missing knowledge hints: generated from task/query
- Gap candidates: 3 entries, all `authority=candidate_only`
- Next calls: none

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `pass`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `pass`
- Capture permission gate: `not-applicable`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

### weak_gap_does_not_load_or_cite

Runner: `019ddd68-9df1-7a52-b2ce-8f1604072425`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add warehouse barcode scanner`
- Query: `barcode scanner token`

Observed Results:

- Actual coverage: `weak`, confidence `0.5560000867747952`
- Action policy: `authority=weak`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`
- Recall: `semantic.status=disabled`
- Discovered IDs: `rule:backend.auth-refresh.v1`, `lesson:backend.auth-debug.v1`, `decision:backend.session-renewal.v1`, `reference:backend.global-refresh.v1`
- Loaded IDs: none
- Cited IDs: none
- Missing knowledge hints: generated from task/query
- Gap candidates: `standard`, `decision`, `lesson`
- Next calls: none

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `pass`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `pass`
- Capture permission gate: `not-applicable`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

### partial_gap_mentions_gap_before_capture

Runner: `019ddd68-9ea8-7a71-a93e-acb9a956164e`

Inputs:

- Project: `mall-api`
- Phase: `debugging`
- Task: `debug session renewal test failure`
- Query: `session renewal tests fail logs`

Observed Results:

- Actual coverage: `partial`, confidence `0.7555`
- Action policy: `authority=partial`, `load=allowed`, `claim=must_mention_gap`
- Recall: `semantic.status=disabled`
- Discovered IDs: included `lesson:backend.auth-debug.v1`
- Loaded IDs: `lesson:backend.auth-debug.v1`
- Cited IDs: none
- Missing knowledge hints: generated from task/query
- Gap candidates: candidate-only; not cited or treated as official knowledge
- Next calls: `get_knowledge_item` for `lesson:backend.auth-debug.v1`

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `not-applicable`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `pass`
- Capture permission gate: `pass`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

### filter_excluded_match_produces_filter_gap

Runner: `019ddd68-9faa-7933-8053-eb0799710072`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add refresh token endpoint`
- Query: `refresh token`
- Filters: `tags=payments`

Observed Results:

- Actual coverage: `none`
- Action policy: `authority=none`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`
- Recall: `semantic.status=disabled`
- Discovered IDs: none
- Loaded IDs: none
- Cited IDs: none
- Missing knowledge hints: generated from task/query
- Gap candidates: `standard`, `decision`, `lesson`, all candidate-only
- Next calls: none

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `pass`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `pass`
- Capture permission gate: `not-applicable`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

### agent_requests_capture_permission

Runner: `019ddd68-a083-7f52-8455-ba1db4c3a6a9`

Inputs:

- Project: `mall-api`
- Phase: `implementation`
- Task: `add payment webhook signature verification`
- Query: `payment webhook signature`

Observed Results:

- Actual coverage: `none`
- Action policy: `authority=none`, `load=forbidden`, `cite=forbidden`, `claim=forbidden`
- Recall: `semantic.status=disabled`
- Discovered IDs: none
- Loaded IDs: none
- Cited IDs: none
- Gap candidates: `standard`, `decision`, `lesson`
- Proposed approval question: "Argos found no official knowledge for this task and produced candidate-only gap prompts for payment webhook signature verification. Do you want me to create a capture-knowledge proposal from these candidates?"

Guards:

- Progressive disclosure: `pass`
- Weak/none no-overclaim: `pass`
- Citation accountability: `pass`
- Cited IDs subset of loaded-and-used IDs: `not-applicable`
- Gap candidates not cited: `pass`
- Capture permission gate: `pass`
- Action policy followed: `pass`
- Context contamination: `pass`

Result: `pass`

## Summary

Round 4 passed all target cases. The current discovery layer can expose what it knows, state that semantic recall is unavailable, route strong matches without semantic recall, and surface missing knowledge as non-authoritative `gap_candidates`. The action policy remained the final guard: weak/none cases did not load or cite knowledge, partial cases loaded only routed IDs, and capture remained approval-gated.

## Follow-Up

- Consider improving `suggested_title` wording to avoid duplicated task/query phrases.
- Keep semantic recall as a later optional provider behind the same `recall.semantic` surface.
