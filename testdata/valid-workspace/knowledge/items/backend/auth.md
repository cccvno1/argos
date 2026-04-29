---
id: rule:backend.auth.v1
title: Auth middleware rule
type: rule
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
applies_to:
  languages: [go]
  frameworks: [gin]
  files: ["internal/auth/**"]
updated_at: 2026-04-29
---

Require explicit auth middleware for account endpoints.
