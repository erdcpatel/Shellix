package history

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"shellix/internal/core"
)

const (
	DefaultMaxEntries = 5000
)

// Store manages in-memory command history with persistent disk backing
type Store struct {
	mu         sync.RWMutex
	filePath   string
	entries    []HistoryEntry
	maxEntries int
}

// NewStore initializes a command history store backed by the specified file path
func NewStore(filePath string) (*Store, error) {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user home: %w", err)
		}
		dir := filepath.Join(home, ".shellix")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create shellix dir: %w", err)
		}
		filePath = filepath.Join(dir, "history.json")
	} else {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	s := &Store{
		filePath:   filePath,
		entries:    make([]HistoryEntry, 0),
		maxEntries: DefaultMaxEntries,
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		// Non-fatal, start with empty history
		s.entries = make([]HistoryEntry, 0)
	}

	return s, nil
}

// load reads history entries from disk
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return err
	}

	// Ensure file permissions are 0600
	_ = os.Chmod(s.filePath, 0600)
	return nil
}

// save writes history entries to disk atomically
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	dir := filepath.Dir(s.filePath)
	tmpFile, err := os.CreateTemp(dir, "history-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpName, s.filePath); err != nil {
		return fmt.Errorf("failed to replace history file: %w", err)
	}

	return nil
}

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Record captures an executed command, sanitizing secrets and enforcing max bounds
func (s *Store) Record(cmd string, dir string, duration time.Duration, exitCode int) (*HistoryEntry, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, nil
	}

	// Security: Sanitize any embedded secrets before storage
	cmd = core.GlobalSanitizer.Sanitize(cmd)

	s.mu.Lock()
	defer s.mu.Unlock()

	entry := HistoryEntry{
		ID:        generateID(),
		Command:   cmd,
		Directory: dir,
		Timestamp: time.Now(),
		Duration:  duration,
		ExitCode:  exitCode,
		Count:     1,
	}

	// Deduplication: if immediate predecessor is identical command in same dir, increment count
	if len(s.entries) > 0 {
		last := &s.entries[len(s.entries)-1]
		if last.Command == cmd && last.Directory == dir {
			last.Count++
			last.Timestamp = entry.Timestamp
			last.Duration = duration
			last.ExitCode = exitCode
			_ = s.save()
			return last, nil
		}
	}

	s.entries = append(s.entries, entry)
	if len(s.entries) > s.maxEntries {
		s.entries = s.entries[len(s.entries)-s.maxEntries:]
	}

	if err := s.save(); err != nil {
		return &entry, err
	}

	return &entry, nil
}

// GetRecent returns the most recent commands up to limit, ordered newest first
func (s *Store) GetRecent(limit int) []HistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.entries)
	if total == 0 {
		return []HistoryEntry{}
	}

	if limit <= 0 || limit > total {
		limit = total
	}

	out := make([]HistoryEntry, limit)
	for i := 0; i < limit; i++ {
		out[i] = s.entries[total-1-i]
	}
	return out
}

// Search performs fast in-memory filtering across command text and directory
func (s *Store) Search(query string, limit int) []HistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.TrimSpace(strings.ToLower(query))
	var matches []HistoryEntry

	// Scan reverse chronological (newest first)
	for i := len(s.entries) - 1; i >= 0; i-- {
		e := s.entries[i]
		if query == "" ||
			strings.Contains(strings.ToLower(e.Command), query) ||
			strings.Contains(strings.ToLower(e.Directory), query) {
			matches = append(matches, e)
			if limit > 0 && len(matches) >= limit {
				break
			}
		}
	}
	return matches
}

// GetStats compiles execution analytics from recorded history
func (s *Store) GetStats() HistoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := HistoryStats{
		TopCommands: make([]CommandFreq, 0),
	}

	if len(s.entries) == 0 {
		return stats
	}

	var totalDur time.Duration
	freqMap := make(map[string]*CommandFreq)

	for _, e := range s.entries {
		cnt := e.Count
		if cnt <= 0 {
			cnt = 1
		}
		stats.TotalCommands += cnt
		if e.ExitCode == 0 {
			stats.SuccessCount += cnt
		} else {
			stats.FailureCount += cnt
		}
		totalDur += e.Duration * time.Duration(cnt)

		cf, exists := freqMap[e.Command]
		if !exists {
			cf = &CommandFreq{
				Command:  e.Command,
				LastUsed: e.Timestamp,
			}
			freqMap[e.Command] = cf
		}
		cf.Count += cnt
		if e.Timestamp.After(cf.LastUsed) {
			cf.LastUsed = e.Timestamp
		}
		if e.ExitCode == 0 {
			cf.Successes += cnt
		} else {
			cf.Failures += cnt
		}
	}

	stats.UniqueCommands = len(freqMap)
	if stats.TotalCommands > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCommands) * 100.0
		stats.AvgDuration = totalDur / time.Duration(stats.TotalCommands)
	}

	// Convert map to slice and sort by frequency descending
	freqList := make([]CommandFreq, 0, len(freqMap))
	for _, cf := range freqMap {
		if cf.Count > 0 {
			// Compute average duration if applicable
			freqList = append(freqList, *cf)
		}
	}

	sort.Slice(freqList, func(i, j int) bool {
		if freqList[i].Count == freqList[j].Count {
			return freqList[i].LastUsed.After(freqList[j].LastUsed)
		}
		return freqList[i].Count > freqList[j].Count
	})

	if len(freqList) > 10 {
		stats.TopCommands = freqList[:10]
	} else {
		stats.TopCommands = freqList
	}

	return stats
}

// Clear wipes all recorded command history
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make([]HistoryEntry, 0)
	return s.save()
}
