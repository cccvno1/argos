package knowledgewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovedWriteVocabularyAbsentFromLiveSurface(t *testing.T) {
	root := repoRootForVocabularyTest(t)
	var matches []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if shouldSkipVocabularyScanDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isLiveVocabularyScanFile(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(data)
		for _, term := range removedAuthoringTermsForTests() {
			if strings.Contains(body, term) {
				matches = append(matches, rel+": "+term)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
	if len(matches) > 0 {
		t.Fatalf("removed write vocabulary found in live surface:\n%s", strings.Join(matches, "\n"))
	}
}

func repoRootForVocabularyTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root containing go.mod")
		}
		dir = parent
	}
}

func shouldSkipVocabularyScanDir(rel string) bool {
	switch rel {
	case ".git", ".worktrees":
		return true
	}
	for _, prefix := range []string{
		"docs/superpowers/specs/",
		"docs/superpowers/plans/",
		"docs/superpowers/reports/",
	} {
		if strings.HasPrefix(rel+"/", prefix) {
			return true
		}
	}
	return false
}

func isLiveVocabularyScanFile(rel string) bool {
	switch filepath.Ext(rel) {
	case ".go", ".md", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func removedAuthoringTermsForTests() []string {
	return []string{
		removedWriteVocabularyTerm("authoring", "_", "packet"),
		removedWriteVocabularyTerm("proposal", "_", "scaffold"),
		removedWriteVocabularyTerm("source", "_", "profile"),
		removedWriteVocabularyTerm("proposed", "_", "shape"),
		removedWriteVocabularyTerm("overlap", "_", "decision"),
		removedWriteVocabularyTerm("verification", "_", "plan"),
		removedWriteVocabularyTerm("human", "_", "review"),
		removedWriteVocabularyTerm("artifact", "_", "state"),
		removedWriteVocabularyTerm("author", " ", "inspect"),
		removedWriteVocabularyTerm("author", " ", "verify"),
		removedWriteVocabularyTerm("authoring", ".", "proposal", ".", "v2"),
		removedWriteVocabularyTerm("argos", " ", "author"),
		removedWriteVocabularyTerm("dogfood", " ", "authoring"),
		removedWriteVocabularyTerm("authoring", "-", "golden"),
		removedWriteVocabularyTerm("argos", "-", "authoring", "-", "dogfood"),
		removedWriteVocabularyTerm("pro", "mote"),
	}
}

func removedWriteVocabularyTerm(parts ...string) string {
	return strings.Join(parts, "")
}
