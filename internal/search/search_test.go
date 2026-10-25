package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchLiteral(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test files
	file1 := filepath.Join(tempDir, "auth.go")
	_ = os.WriteFile(file1, []byte("package auth\n\nfunc ValidateToken() bool {\n  return true\n}\n"), 0644)

	file2 := filepath.Join(tempDir, "user.go")
	_ = os.WriteFile(file2, []byte("package user\n\ntype User struct {\n  Name string\n}\n"), 0644)

	engine := NewEngine()
	opts := SearchOptions{
		Query: "ValidateToken",
		Paths: []string{tempDir},
	}

	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	if summary.TotalMatches != 1 {
		t.Fatalf("expected 1 match, got %d", summary.TotalMatches)
	}
	if summary.Matches[0].LineNumber != 3 {
		t.Errorf("expected line number 3, got %d", summary.Matches[0].LineNumber)
	}
}

func TestSearchRegex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-regex-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	file1 := filepath.Join(tempDir, "app.log")
	_ = os.WriteFile(file1, []byte("INFO [2026-10-25] server ready\nERROR [2026-10-25] connection timeout\n"), 0644)

	engine := NewEngine()
	opts := SearchOptions{
		Query:   `ERROR.*timeout`,
		Paths:   []string{tempDir},
		IsRegex: true,
	}

	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("regex search error: %v", err)
	}

	if summary.TotalMatches != 1 {
		t.Fatalf("expected 1 match, got %d", summary.TotalMatches)
	}
	if summary.Matches[0].LineNumber != 2 {
		t.Errorf("expected match on line 2, got %d", summary.Matches[0].LineNumber)
	}
}

func TestSearchGitignore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-gi-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create .gitignore
	_ = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("ignored_dir/\nsecret.txt\n*.log\n"), 0644)

	// Create matching file in root
	_ = os.WriteFile(filepath.Join(tempDir, "regular.go"), []byte("func ShellixMatch() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "secret.txt"), []byte("func ShellixMatch() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "debug.log"), []byte("func ShellixMatch() {}\n"), 0644)

	// Create ignored subdir
	subDir := filepath.Join(tempDir, "ignored_dir")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "inner.go"), []byte("func ShellixMatch() {}\n"), 0644)

	engine := NewEngine()
	opts := SearchOptions{
		Query: "ShellixMatch",
		Paths: []string{tempDir},
	}

	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if summary.TotalMatches != 1 {
		t.Fatalf("expected exactly 1 match (in regular.go), got %d matches", summary.TotalMatches)
	}
	if filepath.Base(summary.Matches[0].Path) != "regular.go" {
		t.Errorf("expected match in regular.go, got %s", summary.Matches[0].Path)
	}
}
