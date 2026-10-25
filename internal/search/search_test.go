package search

import (
	"os"
	"path/filepath"
	"strings"
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

	_ = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("ignored_dir/\nsecret.txt\n*.log\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "regular.go"), []byte("func ShellixMatch() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "secret.txt"), []byte("func ShellixMatch() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "debug.log"), []byte("func ShellixMatch() {}\n"), 0644)

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

func TestSearchDirectoryRollupAndReporting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-rollup-*")
	if err != nil {
		t.Fatalf("failed temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create 3 folder subtrees simulating checked-out projects:
	// folder-alpha/src/service.go
	// folder-beta/main.go
	// folder-gamma/readme.txt (no matches)
	alphaDir := filepath.Join(tempDir, "folder-alpha", "src")
	_ = os.MkdirAll(alphaDir, 0755)
	_ = os.WriteFile(filepath.Join(alphaDir, "service.go"), []byte("package service\n// TODO: auditTokenCheck\nfunc auditTokenCheck() {}\n"), 0644)

	betaDir := filepath.Join(tempDir, "folder-beta")
	_ = os.MkdirAll(betaDir, 0755)
	_ = os.WriteFile(filepath.Join(betaDir, "main.go"), []byte("package main\nconst auditTokenCheck = true\n"), 0644)

	gammaDir := filepath.Join(tempDir, "folder-gamma")
	_ = os.MkdirAll(gammaDir, 0755)
	_ = os.WriteFile(filepath.Join(gammaDir, "readme.txt"), []byte("nothing here\n"), 0644)

	reportPath := filepath.Join(tempDir, "report.md")
	engine := NewEngine()
	opts := SearchOptions{
		Query:      "auditTokenCheck",
		Paths:      []string{tempDir},
		GroupByDir: true,
		ReportFile: reportPath,
	}

	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// 2 folders should have hits: folder-alpha (2 hits) and folder-beta (1 hit)
	if summary.TotalMatches != 3 {
		t.Fatalf("expected 3 total matches, got %d", summary.TotalMatches)
	}
	if len(summary.Folders) != 2 {
		t.Fatalf("expected 2 folders with hits, got %d", len(summary.Folders))
	}
	if summary.Folders[0].Name != "folder-alpha" || summary.Folders[0].TotalHits != 2 {
		t.Errorf("expected folder-alpha to have 2 hits, got %+v", summary.Folders[0])
	}
	if summary.Folders[1].Name != "folder-beta" || summary.Folders[1].TotalHits != 1 {
		t.Errorf("expected folder-beta to have 1 hit, got %+v", summary.Folders[1])
	}

	// Verify report.md was generated
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed reading report file: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "# Code Search Audit Report: `auditTokenCheck`") {
		t.Errorf("report missing title: %s", reportStr)
	}
	if !strings.Contains(reportStr, "folder-alpha") || !strings.Contains(reportStr, "folder-beta") {
		t.Errorf("report missing folders: %s", reportStr)
	}
}

func TestSearchExtensionsAndExclude(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-filter-*")
	if err != nil {
		t.Fatalf("failed temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	_ = os.WriteFile(filepath.Join(tempDir, "app.go"), []byte("const TARGET_KEY = 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "app.js"), []byte("const TARGET_KEY = 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "bundle.min.js"), []byte("const TARGET_KEY = 1\n"), 0644)

	engine := NewEngine()

	// 1. Filter by extension: only .go
	optsGo := SearchOptions{
		Query:      "TARGET_KEY",
		Paths:      []string{tempDir},
		Extensions: []string{"go"},
	}
	summaryGo, err := engine.Search(optsGo)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if summaryGo.TotalMatches != 1 || filepath.Base(summaryGo.Matches[0].Path) != "app.go" {
		t.Fatalf("expected only app.go match, got %d matches", summaryGo.TotalMatches)
	}

	// 2. Exclude globs: exclude *.min.js
	optsExclude := SearchOptions{
		Query:        "TARGET_KEY",
		Paths:        []string{tempDir},
		ExcludeGlobs: []string{"*.min.js"},
	}
	summaryEx, err := engine.Search(optsExclude)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if summaryEx.TotalMatches != 2 {
		t.Fatalf("expected 2 matches excluding min.js, got %d", summaryEx.TotalMatches)
	}
}

func TestSearchContextLines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-ctx-*")
	if err != nil {
		t.Fatalf("failed temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	content := "line 1: header\nline 2: setup\nline 3: MATCH_LINE\nline 4: teardown\nline 5: footer\n"
	_ = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)

	engine := NewEngine()
	opts := SearchOptions{
		Query:         "MATCH_LINE",
		Paths:         []string{tempDir},
		ContextBefore: 1,
		ContextAfter:  1,
	}

	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	if summary.TotalMatches != 1 {
		t.Fatalf("expected 1 match, got %d", summary.TotalMatches)
	}

	m := summary.Matches[0]
	if len(m.Context) != 3 {
		t.Fatalf("expected 3 lines in context (1 before, 1 match, 1 after), got %d: %+v", len(m.Context), m.Context)
	}
	if m.Context[0].Content != "line 2: setup" || m.Context[0].IsMatch {
		t.Errorf("unexpected before context line: %+v", m.Context[0])
	}
	if m.Context[1].Content != "line 3: MATCH_LINE" || !m.Context[1].IsMatch {
		t.Errorf("unexpected match line: %+v", m.Context[1])
	}
	if m.Context[2].Content != "line 4: teardown" || m.Context[2].IsMatch {
		t.Errorf("unexpected after context line: %+v", m.Context[2])
	}
}

func TestSearchReportsFormats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-search-reports-*")
	if err != nil {
		t.Fatalf("failed temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	subDir := filepath.Join(tempDir, "my-service")
	_ = os.MkdirAll(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "app.go"), []byte("const API_SECRET = \"xyz\"\n"), 0644)

	engine := NewEngine()
	opts := SearchOptions{
		Query: "API_SECRET",
		Paths: []string{tempDir},
	}
	summary, err := engine.Search(opts)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// 1. JSON Report
	jsonPath := filepath.Join(tempDir, "audit.json")
	if err := WriteReportToFile(summary, jsonPath); err != nil {
		t.Fatalf("WriteReportToFile JSON failed: %v", err)
	}
	jsonData, _ := os.ReadFile(jsonPath)
	if !strings.Contains(string(jsonData), "API_SECRET") || !strings.Contains(string(jsonData), "my-service") {
		t.Errorf("unexpected JSON report: %s", string(jsonData))
	}

	// 2. CSV Report
	csvPath := filepath.Join(tempDir, "audit.csv")
	if err := WriteReportToFile(summary, csvPath); err != nil {
		t.Fatalf("WriteReportToFile CSV failed: %v", err)
	}
	csvData, _ := os.ReadFile(csvPath)
	if !strings.Contains(string(csvData), "Folder,FilePath,LineNumber,Snippet") {
		t.Errorf("unexpected CSV header: %s", string(csvData))
	}
}
