package knowledge

import "testing"

func TestParseItemReadsFrontmatterAndBody(t *testing.T) {
	input := []byte(`---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
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

# Body

Use short-lived access tokens.
`)

	item, err := ParseItem("knowledge/items/backend/auth.md", input)
	if err != nil {
		t.Fatalf("ParseItem returned error: %v", err)
	}
	if item.ID != "backend.auth.jwt-refresh-token.v1" {
		t.Fatalf("unexpected ID: %s", item.ID)
	}
	if item.AppliesTo.Files[0] != "internal/auth/**" {
		t.Fatalf("unexpected file scope: %#v", item.AppliesTo.Files)
	}
	if item.Body != "# Body\n\nUse short-lived access tokens.\n" {
		t.Fatalf("unexpected body: %q", item.Body)
	}
}
