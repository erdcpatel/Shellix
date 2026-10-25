package flow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSnapshotAndDiff(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-watch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	fileA := filepath.Join(tmpDir, "main.go")
	fileB := filepath.Join(tmpDir, "flow.yaml")
	fileIgnored := filepath.Join(tmpDir, "notes.txt")
	hiddenDir := filepath.Join(tmpDir, ".git")
	_ = os.MkdirAll(hiddenDir, 0755)
	fileInHidden := filepath.Join(hiddenDir, "config.go")

	_ = os.WriteFile(fileA, []byte("package main"), 0644)
	_ = os.WriteFile(fileB, []byte("name: test"), 0644)
	_ = os.WriteFile(fileIgnored, []byte("ignore me"), 0644)
	_ = os.WriteFile(fileInHidden, []byte("hidden"), 0644)

	exts := []string{".go", ".yaml"}

	// 1. Initial snapshot
	snap1 := takeDirSnapshot(tmpDir, exts)
	if len(snap1) != 2 {
		t.Errorf("expected exactly 2 watched files in snapshot, got %d", len(snap1))
	}
	if _, ok := snap1[fileA]; !ok {
		t.Errorf("expected main.go in snapshot")
	}
	if _, ok := snap1[fileB]; !ok {
		t.Errorf("expected flow.yaml in snapshot")
	}

	// 2. Modify fileA
	time.Sleep(15 * time.Millisecond)
	_ = os.WriteFile(fileA, []byte("package main\nfunc init() {}"), 0644)

	snap2 := takeDirSnapshot(tmpDir, exts)
	diff := diffSnapshots(snap1, snap2)
	if diff != fileA {
		t.Errorf("expected diff to be %s, got %s", fileA, diff)
	}

	// 3. Add fileC
	fileC := filepath.Join(tmpDir, "service.go")
	_ = os.WriteFile(fileC, []byte("package main"), 0644)

	snap3 := takeDirSnapshot(tmpDir, exts)
	diff = diffSnapshots(snap2, snap3)
	if diff != fileC {
		t.Errorf("expected new file %s in diff, got %s", fileC, diff)
	}

	// 4. Delete fileC
	_ = os.Remove(fileC)
	snap4 := takeDirSnapshot(tmpDir, exts)
	diff = diffSnapshots(snap3, snap4)
	if diff != fileC+" (deleted)" {
		t.Errorf("expected deleted file in diff, got %s", diff)
	}
}
