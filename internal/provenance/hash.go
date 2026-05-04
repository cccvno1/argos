package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func HashFile(root string, relPath string) (string, error) {
	clean, err := cleanRelPath(relPath)
	if err != nil {
		return "", err
	}
	file, err := os.Open(filepath.Join(root, clean))
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filepath.ToSlash(clean), err)
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", filepath.ToSlash(clean), err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func HashTree(root string, relPath string) (string, error) {
	clean, err := cleanRelPath(relPath)
	if err != nil {
		return "", err
	}
	absRoot := filepath.Join(root, clean)
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", filepath.ToSlash(clean), err)
	}
	if !info.IsDir() {
		return HashFile(root, clean)
	}

	var files []string
	if err := filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return "", fmt.Errorf("walk %s: %w", filepath.ToSlash(clean), err)
	}
	sort.Strings(files)

	sum := sha256.New()
	for _, rel := range files {
		fileHash, err := HashFile(root, rel)
		if err != nil {
			return "", err
		}
		sum.Write([]byte(rel))
		sum.Write([]byte{0})
		sum.Write([]byte(fileHash))
		sum.Write([]byte{'\n'})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func cleanRelPath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("%s: path must be relative", relPath)
	}
	clean := filepath.Clean(relPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: path must stay inside workspace", relPath)
	}
	return clean, nil
}
