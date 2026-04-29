package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadItemsReadsMarkdownFiles(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/items/backend/auth.md", validKnowledgeItem("backend.auth.jwt-refresh-token.v1", "rule"))

	items, err := LoadItems(root)
	if err != nil {
		t.Fatalf("LoadItems returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "backend.auth.jwt-refresh-token.v1" {
		t.Fatalf("unexpected item ID: %s", items[0].ID)
	}
}

func TestLoadOfficialKnowledgeReadsItemsAndPackages(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/items/backend/auth.md", validKnowledgeItem("rule:backend.auth.v1", "rule"))
	writeKnowledgeFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validPackageItem("package:backend.redis.best-practices.v1"))
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/draft/KNOWLEDGE.md", validPackageItem("package:backend.draft.v1"))

	items, err := LoadOfficial(root)
	if err != nil {
		t.Fatalf("LoadOfficial returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected official item and package, got %d: %#v", len(items), items)
	}
	assertLoadedID(t, items, "rule:backend.auth.v1")
	assertLoadedID(t, items, "package:backend.redis.best-practices.v1")
}

func TestLoadInboxKnowledgeReadsInboxItemsAndPackages(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/.inbox/items/backend/auth.md", validKnowledgeItem("rule:backend.auth.draft.v1", "rule"))
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/redis/draft/KNOWLEDGE.md", validPackageItem("package:backend.redis.draft.v1"))
	writeKnowledgeFile(t, root, "knowledge/packages/backend/redis/best-practices/KNOWLEDGE.md", validPackageItem("package:backend.redis.best-practices.v1"))

	items, err := LoadInbox(root)
	if err != nil {
		t.Fatalf("LoadInbox returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected inbox item and package, got %d: %#v", len(items), items)
	}
	assertLoadedID(t, items, "rule:backend.auth.draft.v1")
	assertLoadedID(t, items, "package:backend.redis.draft.v1")
}

func TestLoadPathReadsSinglePackageDirectory(t *testing.T) {
	root := t.TempDir()
	writeKnowledgeFile(t, root, "knowledge/.inbox/packages/backend/redis/draft/KNOWLEDGE.md", validPackageItem("package:backend.redis.draft.v1"))

	items, err := LoadPath(root, "knowledge/.inbox/packages/backend/redis/draft")
	if err != nil {
		t.Fatalf("LoadPath returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "package:backend.redis.draft.v1" {
		t.Fatalf("unexpected loaded items: %#v", items)
	}
}

func assertLoadedID(t *testing.T, items []Item, id string) {
	t.Helper()
	for _, item := range items {
		if item.ID == id {
			return
		}
	}
	t.Fatalf("expected loaded id %s in %#v", id, items)
}

func writeKnowledgeFile(t *testing.T, root, rel, body string) {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validKnowledgeItem(id string, itemType string) string {
	return `---
id: ` + id + `
title: Auth rule
type: ` + itemType + `
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`
}

func validPackageItem(id string) string {
	return `---
id: ` + id + `
title: Redis Best Practices
type: package
tech_domains: [backend]
business_domains: []
projects: []
status: draft
priority: should
tags: [redis]
updated_at: 2026-04-29
---
## Purpose

Document Redis usage.

## When To Use

Use when Redis is involved.

## Start Here

Read the short rules first.

## Load On Demand

- references/key-design.md when designing keys.
`
}
