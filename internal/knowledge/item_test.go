package knowledge

import (
	"strings"
	"testing"
)

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

func TestParseItemAcceptsCRLFFrontmatterDelimiters(t *testing.T) {
	input := []byte("---\r\nid: backend.auth.jwt-refresh-token.v1\r\ntitle: JWT refresh token handling convention\r\ntype: rule\r\ntech_domains: [backend]\r\nbusiness_domains: [account]\r\nprojects: [mall-api]\r\nstatus: active\r\npriority: must\r\nupdated_at: 2026-04-29\r\n---\r\n\r\n# Body\r\n\r\nUse short-lived access tokens.\r\n")

	item, err := ParseItem("knowledge/items/backend/auth.md", input)
	if err != nil {
		t.Fatalf("ParseItem returned error: %v", err)
	}
	if item.Body != "# Body\n\nUse short-lived access tokens.\n" {
		t.Fatalf("unexpected body: %q", item.Body)
	}
}

func TestParseItemRejectsMissingClosingDelimiter(t *testing.T) {
	_, err := ParseItem("knowledge/items/backend/auth.md", []byte("---\nid: backend.auth.jwt-refresh-token.v1\n\n# Body\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "knowledge/items/backend/auth.md: missing frontmatter closing delimiter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseItemRejectsUnknownFrontmatterField(t *testing.T) {
	input := []byte(`---
id: backend.auth.jwt-refresh-token.v1
title: JWT refresh token handling convention
type: rule
tech_domains: [backend]
business_domains: [account]
projects: [mall-api]
status: active
priority: must
updated_at: 2026-04-29
unexpected: nope
---

# Body
`)

	_, err := ParseItem("knowledge/items/backend/auth.md", input)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "knowledge/items/backend/auth.md: parse frontmatter:") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "field unexpected not found") {
		t.Fatalf("expected unknown field error, got: %v", err)
	}
}
