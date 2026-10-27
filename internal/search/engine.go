package search

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var defaultIgnoredDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	".idea":         true,
	".vscode":       true,
	"bin":           true,
	"dist":          true,
	"target":        true,
	"obj":           true,
	".gradle":       true,
	"build":         true,
	".next":         true,
	".nuxt":         true,
	"coverage":      true,
	".terraform":    true,
	"__pycache__":    true,
	".pytest_cache": true,
	".cache":        true,
}

// Engine implements high-speed parallel folder and code search
type Engine struct {
	WorkerCount int
}

// NewEngine creates a search engine configured for CPU concurrency
func NewEngine() *Engine {
	workers := runtime.NumCPU() * 2
	if workers < 4 {
		workers = 4
	}
	return &Engine{
		WorkerCount: workers,
	}
}

type fileSearchTask struct {
	path   string
	topDir string
}

// Search executes concurrent search across the specified root paths
func (e *Engine) Search(opts SearchOptions) (*SearchSummary, error) {
	start := time.Now()

	if len(opts.Paths) == 0 {
		opts.Paths = []string{"."}
	}

	var validPaths []string
	var warnings []string
	for _, p := range opts.Paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("Search path does not exist: %s", p))
		} else if err != nil {
			warnings = append(warnings, fmt.Sprintf("Error accessing search path %s: %v", p, err))
		} else {
			validPaths = append(validPaths, p)
		}
	}

	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = 5 * 1024 * 1024 // 5 MB
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 500
	}

	matchFn, err := buildMatchFunction(opts)
	if err != nil {
		return nil, err
	}

	extMap := make(map[string]bool)
	for _, ext := range opts.Extensions {
		clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
		if clean != "" {
			extMap[clean] = true
		}
	}

	filesChan := make(chan fileSearchTask, 2048)
	resultsChan := make(chan MatchResult, 2048)

	var filesCount int64
	var matchesCount int64

	// 1. Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < e.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range filesChan {
				atomic.AddInt64(&filesCount, 1)
				e.searchFile(task.path, task.topDir, opts, matchFn, resultsChan, &matchesCount, int64(opts.MaxResults))
			}
		}()
	}

	// Track scanned top-level directories
	var dirsMu sync.Mutex
	scannedDirs := make(map[string]bool)

	// 2. Walk directories in background
	go func() {
		for _, root := range validPaths {
			giPatterns := loadGitignorePatterns(root)

			_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				name := d.Name()
				rel, err := filepath.Rel(root, path)
				if err != nil || rel == "." {
					return nil
				}

				// Cross-platform normalized relative path (consistent forward slashes)
				normRel := filepath.ToSlash(rel)
				topDir := extractTopLevelDir(normRel)
				if topDir != "." {
					dirsMu.Lock()
					scannedDirs[topDir] = true
					dirsMu.Unlock()
				}

				if d.IsDir() {
					if defaultIgnoredDirs[name] {
						return filepath.SkipDir
					}
					if len(giPatterns) > 0 && matchesAnyPattern(normRel, name, giPatterns) {
						return filepath.SkipDir
					}
					if len(opts.ExcludeGlobs) > 0 && matchesExclude(name, normRel, opts.ExcludeGlobs) {
						return filepath.SkipDir
					}
					return nil
				}

				if len(giPatterns) > 0 && matchesAnyPattern(normRel, name, giPatterns) {
					return nil
				}
				if len(opts.ExcludeGlobs) > 0 && matchesExclude(name, normRel, opts.ExcludeGlobs) {
					return nil
				}

				// Extension check
				if len(extMap) > 0 {
					fileExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
					if !extMap[fileExt] {
						return nil
					}
				}

				// Only fetch DirEntry Info when regular file passed all filter checks
				info, err := d.Info()
				if err != nil || !info.Mode().IsRegular() || info.Size() > opts.MaxFileSize {
					return nil
				}

				filesChan <- fileSearchTask{
					path:   filepath.ToSlash(filepath.Clean(path)),
					topDir: topDir,
				}
				return nil
			})
		}
		close(filesChan)
	}()

	// 3. Collector
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var allMatches []MatchResult
	for m := range resultsChan {
		if len(allMatches) < opts.MaxResults {
			allMatches = append(allMatches, m)
		}
	}

	duration := time.Since(start)

	// 4. Build Aggregation Summary
	summary := aggregateResults(opts.Query, allMatches, int(filesCount), len(scannedDirs), duration)
	summary.Warnings = warnings

	// 5. Generate and write report if requested
	if opts.ReportFile != "" {
		_ = WriteReportToFile(summary, opts.ReportFile)
	}

	return summary, nil
}

func buildMatchFunction(opts SearchOptions) (func(string) (bool, int, int), error) {
	if opts.IsRegex {
		pattern := opts.Query
		if !opts.CaseSensitive && !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return func(line string) (bool, int, int) {
			loc := re.FindStringIndex(line)
			if loc != nil {
				return true, loc[0], loc[1]
			}
			return false, -1, -1
		}, nil
	}

	if opts.WholeWord {
		pattern := `\b` + regexp.QuoteMeta(opts.Query) + `\b`
		if !opts.CaseSensitive {
			pattern = `(?i)` + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return func(line string) (bool, int, int) {
			loc := re.FindStringIndex(line)
			if loc != nil {
				return true, loc[0], loc[1]
			}
			return false, -1, -1
		}, nil
	}

	if opts.CaseSensitive {
		target := opts.Query
		return func(line string) (bool, int, int) {
			idx := strings.Index(line, target)
			if idx >= 0 {
				return true, idx, idx + len(target)
			}
			return false, -1, -1
		}, nil
	}

	// Default: Case-insensitive literal substring
	lowerQuery := strings.ToLower(opts.Query)
	return func(line string) (bool, int, int) {
		idx := strings.Index(strings.ToLower(line), lowerQuery)
		if idx >= 0 {
			return true, idx, idx + len(opts.Query)
		}
		return false, -1, -1
	}, nil
}

func (e *Engine) searchFile(path, topDir string, opts SearchOptions, matchFn func(string) (bool, int, int), out chan<- MatchResult, totalMatches *int64, maxMatches int64) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	// Quick binary check on first 512 bytes
	header := make([]byte, 512)
	n, _ := file.Read(header)
	if isBinary(header[:n]) {
		return
	}
	_, _ = file.Seek(0, io.SeekStart)

	scanner := bufio.NewScanner(file)
	lineNum := 1

	needContext := opts.ContextBefore > 0 || opts.ContextAfter > 0
	var recentLines []LineContext
	var fileMatches []*MatchResult
	pendingAfterRemaining := make([]int, 0)

	for scanner.Scan() {
		if atomic.LoadInt64(totalMatches) >= maxMatches {
			break
		}

		line := scanner.Text()

		// Update pending after-context lines for previous matches
		if needContext && len(fileMatches) > 0 {
			for i := range fileMatches {
				if pendingAfterRemaining[i] > 0 {
					fileMatches[i].Context = append(fileMatches[i].Context, LineContext{
						LineNumber: lineNum,
						Content:    line,
						IsMatch:    false,
					})
					pendingAfterRemaining[i]--
				}
			}
		}

		if matched, start, end := matchFn(line); matched {
			atomic.AddInt64(totalMatches, 1)

			res := &MatchResult{
				Path:        filepath.ToSlash(path),
				Directory:   topDir,
				LineNumber:  lineNum,
				LineContent: line,
				MatchStart:  start,
				MatchEnd:    end,
			}

			if needContext {
				// 1. Attach before-context lines
				for _, prev := range recentLines {
					res.Context = append(res.Context, prev)
				}
				// 2. Attach matching line
				res.Context = append(res.Context, LineContext{
					LineNumber: lineNum,
					Content:    line,
					IsMatch:    true,
				})

				fileMatches = append(fileMatches, res)
				pendingAfterRemaining = append(pendingAfterRemaining, opts.ContextAfter)
			} else {
				out <- *res
			}
		}

		if needContext {
			recentLines = append(recentLines, LineContext{
				LineNumber: lineNum,
				Content:    line,
				IsMatch:    false,
			})
			if len(recentLines) > opts.ContextBefore {
				recentLines = recentLines[1:]
			}
		}

		lineNum++
	}

	if needContext {
		for _, m := range fileMatches {
			out <- *m
		}
	}
}

func aggregateResults(query string, matches []MatchResult, totalFiles, totalDirs int, duration time.Duration) *SearchSummary {
	foldersMap := make(map[string]*FolderSummary)
	uniqueFilesMap := make(map[string]bool)
	uniqueDirsMap := make(map[string]bool)

	for _, m := range matches {
		uniqueFilesMap[m.Path] = true
		dirName := m.Directory
		if dirName == "" {
			dirName = "."
		}
		uniqueDirsMap[dirName] = true

		fSummary, exists := foldersMap[dirName]
		if !exists {
			fSummary = &FolderSummary{
				Name: dirName,
				Path: dirName,
			}
			foldersMap[dirName] = fSummary
		}

		fSummary.TotalHits++
		fSummary.Matches = append(fSummary.Matches, m)

		fileFound := false
		for _, f := range fSummary.Files {
			if f == m.Path {
				fileFound = true
				break
			}
		}
		if !fileFound {
			fSummary.Files = append(fSummary.Files, m.Path)
			fSummary.MatchedFiles = len(fSummary.Files)
		}
	}

	var folders []FolderSummary
	for _, f := range foldersMap {
		folders = append(folders, *f)
	}

	// Sort folders by hit count descending, then by name
	sort.Slice(folders, func(i, j int) bool {
		if folders[i].TotalHits != folders[j].TotalHits {
			return folders[i].TotalHits > folders[j].TotalHits
		}
		return folders[i].Name < folders[j].Name
	})

	var uniqueFiles []string
	for f := range uniqueFilesMap {
		uniqueFiles = append(uniqueFiles, f)
	}
	sort.Strings(uniqueFiles)

	var uniqueDirs []string
	for d := range uniqueDirsMap {
		uniqueDirs = append(uniqueDirs, d)
	}
	sort.Strings(uniqueDirs)

	if totalDirs < len(uniqueDirs) {
		totalDirs = len(uniqueDirs)
	}

	return &SearchSummary{
		Query:              query,
		TotalFilesSearched: totalFiles,
		TotalDirsScanned:   totalDirs,
		TotalMatches:       len(matches),
		Duration:           duration,
		DurationStr:        duration.Round(time.Millisecond).String(),
		Folders:            folders,
		Matches:            matches,
		UniqueFiles:        uniqueFiles,
		UniqueDirs:         uniqueDirs,
	}
}

func extractTopLevelDir(relPath string) string {
	clean := filepath.ToSlash(relPath)
	parts := strings.Split(clean, "/")
	if len(parts) <= 1 {
		return "."
	}
	return parts[0]
}

func matchesExclude(name, relPath string, globs []string) bool {
	cleanRel := filepath.ToSlash(relPath)
	for _, g := range globs {
		g = strings.TrimSpace(g)
		if matched, _ := filepath.Match(g, name); matched {
			return true
		}
		if matched, _ := filepath.Match(g, cleanRel); matched {
			return true
		}
		if strings.Contains(cleanRel, g) {
			return true
		}
	}
	return false
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) != -1
}

// loadGitignorePatterns loads pattern rules from a .gitignore file at the root
func loadGitignorePatterns(root string) []string {
	var patterns []string
	giPath := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(giPath)
	if err != nil {
		return patterns
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// matchesAnyPattern determines whether a file or directory matches any gitignore rule
func matchesAnyPattern(relPath, baseName string, patterns []string) bool {
	cleanRel := filepath.ToSlash(relPath)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		cleanP := strings.Trim(p, "/")
		if baseName == cleanP {
			return true
		}
		if matched, _ := filepath.Match(cleanP, baseName); matched {
			return true
		}
		if matched, _ := filepath.Match(cleanP, cleanRel); matched {
			return true
		}
		if strings.HasPrefix(cleanRel, cleanP+"/") {
			return true
		}
	}
	return false
}
