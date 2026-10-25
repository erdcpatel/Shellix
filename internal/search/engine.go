package search

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var defaultIgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".idea":        true,
	".vscode":      true,
	"bin":          true,
	"dist":         true,
	"target":       true,
	"obj":          true,
}

// Engine implements high-speed parallel code search
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

// Search executes concurrent search across the specified root paths
func (e *Engine) Search(opts SearchOptions) (*SearchSummary, error) {
	start := time.Now()

	if len(opts.Paths) == 0 {
		opts.Paths = []string{"."}
	}
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = 5 * 1024 * 1024 // 5 MB
	}
	if opts.MaxResults == 0 {
		opts.MaxResults = 500
	}

	var matchFn func(line string) (bool, int, int)
	if opts.IsRegex {
		re, err := regexp.Compile(opts.Query)
		if err != nil {
			return nil, err
		}
		matchFn = func(line string) (bool, int, int) {
			loc := re.FindStringIndex(line)
			if loc != nil {
				return true, loc[0], loc[1]
			}
			return false, -1, -1
		}
	} else {
		lowerQuery := strings.ToLower(opts.Query)
		matchFn = func(line string) (bool, int, int) {
			idx := strings.Index(strings.ToLower(line), lowerQuery)
			if idx >= 0 {
				return true, idx, idx + len(opts.Query)
			}
			return false, -1, -1
		}
	}

	filesChan := make(chan string, 1024)
	resultsChan := make(chan MatchResult, 1024)

	var filesCount int64
	var matchesCount int64

	// 1. Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < e.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range filesChan {
				atomic.AddInt64(&filesCount, 1)
				e.searchFile(path, opts.MaxFileSize, matchFn, resultsChan, &matchesCount, int64(opts.MaxResults))
			}
		}()
	}

	// 2. Walk directories in background
	go func() {
		for _, root := range opts.Paths {
			_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					if defaultIgnoredDirs[info.Name()] {
						return filepath.SkipDir
					}
					return nil
				}
				if !info.Mode().IsRegular() || info.Size() > opts.MaxFileSize {
					return nil
				}
				filesChan <- path
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

	return &SearchSummary{
		TotalFilesSearched: int(filesCount),
		TotalMatches:       len(allMatches),
		Duration:           time.Since(start),
		Matches:            allMatches,
	}, nil
}

func (e *Engine) searchFile(path string, maxSize int64, matchFn func(string) (bool, int, int), out chan<- MatchResult, totalMatches *int64, maxMatches int64) {
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
	for scanner.Scan() {
		if atomic.LoadInt64(totalMatches) >= maxMatches {
			return
		}

		line := scanner.Text()
		if matched, start, end := matchFn(line); matched {
			atomic.AddInt64(totalMatches, 1)
			out <- MatchResult{
				Path:        path,
				LineNumber:  lineNum,
				LineContent: line,
				MatchStart:  start,
				MatchEnd:    end,
			}
		}
		lineNum++
	}
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) != -1
}
