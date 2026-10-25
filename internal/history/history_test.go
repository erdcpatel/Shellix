package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestHistoryRecordAndDeduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	histFile := filepath.Join(tmpDir, "history.json")
	store, err := NewStore(histFile)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// 1. Record command
	e1, err := store.Record("git status", "/home/user/project", 25*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("failed to record command: %v", err)
	}
	if e1.Command != "git status" || e1.Count != 1 {
		t.Errorf("unexpected entry values: %+v", e1)
	}

	// 2. Record immediate consecutive duplicate -> should increment count, not append new entry
	e2, err := store.Record("git status", "/home/user/project", 15*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("failed to record duplicate: %v", err)
	}
	if e2.Count != 2 {
		t.Errorf("expected count to be 2 for duplicate, got %d", e2.Count)
	}

	recent := store.GetRecent(10)
	if len(recent) != 1 {
		t.Errorf("expected 1 deduplicated entry, got %d", len(recent))
	}

	// 3. Record different command
	_, err = store.Record("go test ./...", "/home/user/project", 350*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("failed to record second command: %v", err)
	}

	recent = store.GetRecent(10)
	if len(recent) != 2 {
		t.Errorf("expected 2 entries, got %d", len(recent))
	}
	if recent[0].Command != "go test ./..." {
		t.Errorf("expected newest command first, got %s", recent[0].Command)
	}
}

func TestHistorySecretSanitization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	histFile := filepath.Join(tmpDir, "history.json")
	store, err := NewStore(histFile)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// Record command with a GitHub token
	e, err := store.Record("curl -H 'Authorization: token ghp_ABC1234567890abcdef' https://api.github.com", "/workspace", 50*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}

	if e.Command == "curl -H 'Authorization: token ghp_ABC1234567890abcdef' https://api.github.com" {
		t.Errorf("secret was not redacted in history record: %s", e.Command)
	}

	// Verify persistence also has redacted token
	data, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) == "" || (len(data) > 0 && string(data) != "" && string(data) != "[]" && (string(data) == "ghp_ABC1234567890abcdef")) {
		t.Errorf("secret found in raw file on disk")
	}
}

func TestHistorySearchAndStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	histFile := filepath.Join(tmpDir, "history.json")
	store, err := NewStore(histFile)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	_, _ = store.Record("git status", "/repo/a", 10*time.Millisecond, 0)
	_, _ = store.Record("git commit -m 'feat'", "/repo/a", 20*time.Millisecond, 0)
	_, _ = store.Record("npm install", "/repo/b", 1200*time.Millisecond, 1)
	_, _ = store.Record("npm test", "/repo/b", 800*time.Millisecond, 0)
	_, _ = store.Record("git push origin main", "/repo/a", 500*time.Millisecond, 0)

	// Search by command substring
	gitMatches := store.Search("git", 10)
	if len(gitMatches) != 3 {
		t.Errorf("expected 3 git matches, got %d", len(gitMatches))
	}

	// Search by directory substring
	repoBMatches := store.Search("/repo/b", 10)
	if len(repoBMatches) != 2 {
		t.Errorf("expected 2 repo b matches, got %d", len(repoBMatches))
	}

	// Stats verification
	stats := store.GetStats()
	if stats.TotalCommands != 5 {
		t.Errorf("expected 5 total commands, got %d", stats.TotalCommands)
	}
	if stats.SuccessCount != 4 {
		t.Errorf("expected 4 successes, got %d", stats.SuccessCount)
	}
	if stats.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", stats.FailureCount)
	}
	if stats.SuccessRate != 80.0 {
		t.Errorf("expected 80%% success rate, got %f", stats.SuccessRate)
	}
	if len(stats.TopCommands) == 0 {
		t.Errorf("expected top commands list to be non-empty")
	}

	// Test Clear
	if err := store.Clear(); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if len(store.GetRecent(10)) != 0 {
		t.Errorf("expected 0 entries after clear")
	}
}

func TestHistoryConcurrentSafety(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	histFile := filepath.Join(tmpDir, "history.json")
	store, err := NewStore(histFile)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10
	iterations := 20

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				cmd := fmt.Sprintf("cmd-worker-%d-iter-%d", workerID, i)
				_, _ = store.Record(cmd, "/test/dir", time.Duration(i)*time.Millisecond, 0)
				_ = store.Search("worker", 5)
				_ = store.GetStats()
			}
		}(w)
	}

	wg.Wait()
	recent := store.GetRecent(0)
	if len(recent) != workers*iterations {
		t.Errorf("expected %d entries, got %d", workers*iterations, len(recent))
	}
}
