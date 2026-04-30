---
id: rule:backend.auth-refresh.v1
title: Refresh Token Endpoint Rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  languages: [go]
  files: ["internal/auth/**"]
updated_at: 2026-04-30
tags: [auth, refresh-token, session-renewal]
---
# Refresh Token Endpoint Rule

Refresh token endpoints must authenticate the session, rotate refresh tokens,
and reject reuse attempts.

Implementation details:

- require auth middleware before touching account state
- rotate refresh token identifiers on every successful renewal
- emit an audit event when reuse is detected
