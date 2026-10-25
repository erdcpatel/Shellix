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
