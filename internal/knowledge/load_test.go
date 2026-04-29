package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadItemsReadsMarkdownFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "knowledge/items/backend")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "auth.md"), []byte(`---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
---
Use short-lived access tokens.
`), 0o644); err != nil {
		t.Fatal(err)
	}

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
