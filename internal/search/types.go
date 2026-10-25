package search

import "time"

// LineContext holds a context line preceding or following a match
type LineContext struct {
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
	IsMatch    bool   `json:"is_match"`
}

// SearchOptions defines criteria for code and file search
type SearchOptions struct {
	Query         string   `json:"query"`
	Paths         []string `json:"paths"`
	IsRegex       bool     `json:"is_regex"`
	CaseSensitive bool     `json:"case_sensitive"`
	WholeWord     bool     `json:"whole_word"`
	MaxResults    int      `json:"max_results"`
	MaxFileSize   int64    `json:"max_file_size"` // in bytes, defaults to 5MB
	Extensions    []string `json:"extensions,omitempty"`
	ExcludeGlobs  []string `json:"exclude_globs,omitempty"`
	ContextBefore int      `json:"context_before,omitempty"`
	ContextAfter  int      `json:"context_after,omitempty"`
	GroupByDir    bool     `json:"group_by_dir"`
	DirsOnly      bool     `json:"dirs_only"`
	FilesOnly     bool     `json:"files_only"`
	CountOnly     bool     `json:"count_only"`
	ReportFile    string   `json:"report_file,omitempty"`
}

// MatchResult represents a single matching line in a file
type MatchResult struct {
	Path        string        `json:"path"`
	Directory   string        `json:"directory"` // Top-level project directory relative to search root
	LineNumber  int           `json:"line_number"`
	LineContent string        `json:"line_content"`
	MatchStart  int           `json:"match_start"`
	MatchEnd    int           `json:"match_end"`
	Context     []LineContext `json:"context,omitempty"`
}

// FolderSummary tracks aggregate statistics for a top-level directory/folder
type FolderSummary struct {
	Name         string        `json:"name"`
	Path         string        `json:"path"`
	MatchedFiles int           `json:"matched_files"`
	TotalHits    int           `json:"total_hits"`
	Files        []string      `json:"files"`
	Matches      []MatchResult `json:"matches,omitempty"`
}

// SearchSummary aggregates results and performance metrics
type SearchSummary struct {
	Query              string          `json:"query"`
	TotalFilesSearched int             `json:"total_files_searched"`
	TotalDirsScanned   int             `json:"total_dirs_scanned"`
	TotalMatches       int             `json:"total_matches"`
	Duration           time.Duration   `json:"duration"`
	DurationStr        string          `json:"duration_str"`
	Folders            []FolderSummary `json:"folders,omitempty"`
	Matches            []MatchResult   `json:"matches"`
	UniqueFiles        []string        `json:"unique_files,omitempty"`
	UniqueDirs         []string        `json:"unique_dirs,omitempty"`
}
