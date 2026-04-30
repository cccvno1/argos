---
id: decision:backend.session-renewal.v1
title: Session Renewal Decision
type: decision
tech_domains: [backend, security]
business_domains: [account]
projects: [mall-api]
status: active
priority: should
applies_to:
  files: ["internal/auth/**", "internal/session/**"]
updated_at: 2026-04-30
tags: [auth, refresh-token, session]
---
# Session Renewal Decision

Mall API renews sessions through refresh token rotation instead of extending
access token lifetime.

The decision exists because short access token lifetimes limit exposure while
refresh rotation preserves user experience.
