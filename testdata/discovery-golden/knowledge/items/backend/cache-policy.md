---
id: reference:backend.cache-policy.v1
title: Cache Policy Reference
type: reference
tech_domains: [backend, database]
business_domains: []
projects: [mall-api]
status: active
priority: should
applies_to:
  files: ["internal/cache/**"]
updated_at: 2026-04-30
tags: [cache, redis, ttl]
---
# Cache Policy Reference

Cache entries should have explicit TTLs and should not hide source-of-truth
database writes.
