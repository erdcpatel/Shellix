package shortcuts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBookmarksAndSnippets(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-shortcuts-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	jsonPath := filepath.Join(tempDir, "shortcuts.json")
	mgr, err := NewManager(jsonPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// 1. Bookmarks test
	err = mgr.AddBookmark("proj", tempDir)
	if err != nil {
		t.Fatalf("failed to add bookmark: %v", err)
	}

	bm, ok := mgr.GetBookmark("proj")
	if !ok {
		t.Fatalf("expected bookmark 'proj' to exist")
	}
	if bm.Path != tempDir {
		t.Errorf("expected path %s, got %s", tempDir, bm.Path)
	}

	allBms := mgr.ListBookmarks()
	if len(allBms) != 1 {
		t.Fatalf("expected 1 bookmark, got %d", len(allBms))
	}

	// 2. Snippets test
	err = mgr.AddSnippet("hello", "echo world", "test snippet")
	if err != nil {
		t.Fatalf("failed to add snippet: %v", err)
	}

	snip, ok := mgr.GetSnippet("hello")
	if !ok {
		t.Fatalf("expected snippet 'hello' to exist")
	}
	if snip.Command != "echo world" {
		t.Errorf("expected command 'echo world', got '%s'", snip.Command)
	}

	// Reload from disk to verify persistence
	mgr2, err := NewManager(jsonPath)
	if err != nil {
		t.Fatalf("failed to reload manager: %v", err)
	}
	if _, ok := mgr2.GetBookmark("proj"); !ok {
		t.Errorf("expected persisted bookmark")
	}
	if _, ok := mgr2.GetSnippet("hello"); !ok {
		t.Errorf("expected persisted snippet")
	}

	// Delete
	_ = mgr.DeleteBookmark("proj")
	if _, ok := mgr.GetBookmark("proj"); ok {
		t.Errorf("expected bookmark to be deleted")
	}

	// Verify file permissions (0600)
	fi, err := os.Stat(jsonPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 file permissions, got %04o", fi.Mode().Perm())
	}
}
