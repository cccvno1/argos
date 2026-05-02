---
id: rule:backend.cache-ttl.v1
title: Cache TTL Rule
type: rule
tech_domains: [backend, database]
business_domains: [catalog]
tags: [cache, ttl, redis]
projects: [mall-api]
status: active
priority: should
updated_at: 2026-05-03
applies_to:
  files:
    - internal/catalog/**
---

Catalog Redis cache TTL rules already exist for this project. New Redis cache
TTL drafts should compare scope with this active rule before creating another
durable rule.
