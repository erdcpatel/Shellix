package search

import "time"

// SearchOptions defines criteria for code and file search
type SearchOptions struct {
	Query       string
	Paths       []string
	IsRegex     bool
	MaxResults  int
	MaxFileSize int64 // in bytes, defaults to 5MB
}

// MatchResult represents a single matching line in a file
type MatchResult struct {
	Path        string
	LineNumber  int
	LineContent string
	MatchStart  int
	MatchEnd    int
}

// SearchSummary aggregates results and performance metrics
type SearchSummary struct {
	TotalFilesSearched int
	TotalMatches       int
	Duration           time.Duration
	Matches            []MatchResult
}
