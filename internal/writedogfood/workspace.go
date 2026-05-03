package writedogfood

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SeedFixtureWorkspace(fixturesRoot string, fixture string, workspace string) error {
	fixturesRoot = strings.TrimSpace(fixturesRoot)
	fixture = fixtureName(fixture)
	workspace = strings.TrimSpace(workspace)
	if fixturesRoot == "" {
		return fmt.Errorf("fixtures root is required")
	}
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	sourceRoot := filepath.Join(filepath.FromSlash(fixturesRoot), filepath.FromSlash(fixture))
	sourceInfo, err := os.Stat(sourceRoot)
	if err != nil {
		return fmt.Errorf("inspect fixture %q: %w", fixture, err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("fixture %q is not a directory", fixture)
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	return copyFixtureTree(sourceRoot, workspace)
}

func copyFixtureTree(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(targetRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFixtureFile(sourcePath, targetPath, info.Mode().Perm())
	})
}

func copyFixtureFile(sourcePath string, targetPath string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return target.Chmod(perm)
}
