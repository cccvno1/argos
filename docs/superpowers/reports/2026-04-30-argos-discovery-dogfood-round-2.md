# Argos Discovery Dogfood Round 2

Date: 2026-04-30

## Scope

Targeted weak/none/missing-knowledge validation.

Cases:

- `weak_single_generic_term`
- `weak_broad_tag_only`
- `none_payment_webhook`
- `none_project_scope_mismatch`
- `none_explicit_filter_excludes_match`

Isolation:

- Each case ran in a fresh runner session.
- Runners received only the case input, workspace path, binary path, allowed
  flow, and reporting instructions.
- Runners did not receive golden expected values.
- The shared fixture workspace indexed 9 official knowledge items.

## Result

Overall: `pass`

- `weak_single_generic_term`: pass
- `weak_broad_tag_only`: pass
- `none_payment_webhook`: pass
- `none_project_scope_mismatch`: pass
- `none_explicit_filter_excludes_match`: pass

## Findings

### Weak Coverage

Both weak cases returned summary-level matches and no `next_calls`.

- `weak_single_generic_term` surfaced auth/session-related IDs as weak,
  non-authoritative context for a barcode scanner task.
- `weak_broad_tag_only` surfaced the generic backend rule with weak coverage,
  despite a numeric confidence of `0.742`.

In both cases the runners inspected summaries/map output only. They did not
call `get_knowledge_item`, did not call `cite_knowledge`, and did not treat the
matches as authoritative guidance.

Status: pass.

### None Coverage

All none cases returned `coverage.status: none`, no discovered IDs, no loaded
IDs, no cited IDs, and no `next_calls`.

- `none_payment_webhook` did not invent payment webhook guidance from unrelated
  backend/auth knowledge.
- `none_project_scope_mismatch` did not use warehouse-scoped knowledge for
  `mall-api`.
- `none_explicit_filter_excludes_match` respected the explicit `payments` tag
  filter and did not bypass it to use refresh-token knowledge.

Status: pass.

### Missing Knowledge Hints

Weak and none responses produced missing knowledge hints. Runners treated those
hints as gap indicators only, not as citation or load recommendations.

Status: pass.

## Design Notes

The current discovery layer gives AI enough signal to avoid overclaiming when
knowledge is absent or weak:

- `coverage.status` is more important than raw confidence.
- `next_calls: null` prevents accidental load/cite behavior.
- Missing hints help identify gaps without implying that matching knowledge
  exists.

One future refinement may be useful: expose a clearer machine-readable
`authority` or `allowed_actions` field so agents do not have to infer behavior
from status, recommendation text, and `next_calls`.

## Verification

Commands run:

```bash
go build -o /tmp/argos-dogfood ./cmd/argos
go test ./internal/discoverytest ./internal/query ./internal/adapters -count=1
```

Result: passed.

## Decision

Round 2 supports continuing the current route. Do not add a mandatory local
vector database yet. The next useful step is to design the optional semantic
enhancement boundary: when lexical/metadata discovery misses, Argos can add a
semantic provider later without changing the weak/none no-overclaim contract.
