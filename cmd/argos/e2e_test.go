package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCLIEndToEndValidWorkspace(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "workspace")
	copyDir(t, fixturePath(t), workspace)

	binary := filepath.Join(tmp, "argos-test")
	build := exec.Command("go", "build", "-o", binary, ".")
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	runCommand(t, workspace, binary, "validate")
	runCommand(t, workspace, binary, "index")
	runCommand(t, workspace, binary, "knowledge", "list", "--json", "--project", "mall-api")
	runCommand(t, workspace, binary, "knowledge", "find", "--json", "--project", "mall-api", "--query", "auth")
	runCommand(t, workspace, binary, "knowledge", "read", "--json", "rule:backend.auth.v1")
	runCommand(t, workspace, binary, "knowledge", "cite", "--json", "rule:backend.auth.v1")
	runCommand(t, workspace, binary, "install-adapters")

	assertExists(t, filepath.Join(workspace, "argos", "index.db"))
	assertExists(t, filepath.Join(workspace, "argos", "generated", "mall-api", "AGENTS.md"))
}

func fixturePath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "valid-workspace")
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	if err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture from %s to %s: %v", src, dst, err)
	}
}

func runCommand(t *testing.T, dir, binary string, args ...string) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s: %v\nstdout:\n%s\nstderr:\n%s", args[0], err, stdout.String(), stderr.String())
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
