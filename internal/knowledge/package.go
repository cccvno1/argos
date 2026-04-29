package knowledge

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var packageSegmentPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

var packageRequiredSections = []string{
	"Purpose",
	"When To Use",
	"Start Here",
	"Load On Demand",
}

func validatePackageItem(item Item) []error {
	var errs []error
	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", item.Path, fmt.Sprintf(format, args...)))
	}
	if filepath.Base(item.Path) != "KNOWLEDGE.md" {
		addErr("package entrypoint must be KNOWLEDGE.md")
	}
	if !strings.Contains(item.Path, "/packages/") && !strings.Contains(item.Path, `\packages\`) {
		addErr("package entrypoint must be under a packages directory")
	}
	for _, section := range packageRequiredSections {
		if !hasMarkdownSection(item.Body, section) {
			addErr("missing package section: %s", section)
		}
	}
	return errs
}

func hasMarkdownSection(body string, name string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")) == name {
			return true
		}
	}
	return false
}

func ValidatePackagePath(relPath string) error {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return err
	}
	parts := strings.Split(filepath.ToSlash(clean), "/")
	if len(parts) < 3 {
		return fmt.Errorf("%s: package path is too short", relPath)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("%s: package path contains unsafe segment", relPath)
		}
		if strings.HasPrefix(part, ".") && part != ".inbox" {
			return fmt.Errorf("%s: package path contains hidden segment %s", relPath, part)
		}
	}
	if len(parts) >= 3 && parts[0] == "knowledge" && parts[1] == "packages" {
		return validatePackageSegments(relPath, parts[2:])
	}
	if len(parts) >= 4 && parts[0] == "knowledge" && parts[1] == ".inbox" && parts[2] == "packages" {
		return validatePackageSegments(relPath, parts[3:])
	}
	return fmt.Errorf("%s: package path must be under knowledge/packages or knowledge/.inbox/packages", relPath)
}

func validatePackageSegments(relPath string, segments []string) error {
	if len(segments) == 0 {
		return fmt.Errorf("%s: package path must include at least one package segment", relPath)
	}
	for _, segment := range segments {
		if !packageSegmentPattern.MatchString(segment) {
			return fmt.Errorf("%s: invalid package path segment %s", relPath, segment)
		}
	}
	return nil
}
