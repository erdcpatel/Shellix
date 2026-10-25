package shortcuts

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"shellix/internal/core"
)

// Manager handles CRUD for bookmarks and snippets
type Manager struct {
	filePath string
	mu       sync.RWMutex
	data     DataStore
}

// NewManager creates or loads the shortcuts store from disk
func NewManager(customPath string) (*Manager, error) {
	path := customPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".shellix")
		_ = core.EnsureSecureDir(dir)
		path = filepath.Join(dir, "shortcuts.json")
	}

	mgr := &Manager{
		filePath: path,
		data: DataStore{
			Bookmarks: make(map[string]Bookmark),
			Snippets:  make(map[string]Snippet),
		},
	}

	_ = mgr.load()
	return mgr, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var ds DataStore
	if err := json.Unmarshal(data, &ds); err != nil {
		return err
	}
	if ds.Bookmarks == nil {
		ds.Bookmarks = make(map[string]Bookmark)
	}
	if ds.Snippets == nil {
		ds.Snippets = make(map[string]Snippet)
	}
	m.data = ds
	return nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(m.filePath, data, 0600); err != nil {
		return err
	}
	return core.EnsureSecureFile(m.filePath)
}

// AddBookmark registers a new path bookmark
func (m *Manager) AddBookmark(name, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	m.data.Bookmarks[name] = Bookmark{
		Name:      name,
		Path:      absPath,
		CreatedAt: time.Now(),
	}
	return m.save()
}

// GetBookmark retrieves a bookmark by alias
func (m *Manager) GetBookmark(name string) (Bookmark, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bm, ok := m.data.Bookmarks[name]
	return bm, ok
}

// ListBookmarks returns all registered bookmarks
func (m *Manager) ListBookmarks() []Bookmark {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]Bookmark, 0, len(m.data.Bookmarks))
	for _, b := range m.data.Bookmarks {
		res = append(res, b)
	}
	return res
}

// DeleteBookmark removes a bookmark
func (m *Manager) DeleteBookmark(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data.Bookmarks, name)
	return m.save()
}

// AddSnippet saves a reusable command
func (m *Manager) AddSnippet(name, command, description string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data.Snippets[name] = Snippet{
		Name:        name,
		Command:     command,
		Description: description,
		CreatedAt:   time.Now(),
	}
	return m.save()
}

// GetSnippet retrieves a snippet by name
func (m *Manager) GetSnippet(name string) (Snippet, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snip, ok := m.data.Snippets[name]
	return snip, ok
}

// ListSnippets returns all saved snippets
func (m *Manager) ListSnippets() []Snippet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]Snippet, 0, len(m.data.Snippets))
	for _, s := range m.data.Snippets {
		res = append(res, s)
	}
	return res
}

// DeleteSnippet removes a snippet
func (m *Manager) DeleteSnippet(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data.Snippets, name)
	return m.save()
}

// RunSnippet executes a snippet directly in shell
func (m *Manager) RunSnippet(name string) error {
	snip, ok := m.GetSnippet(name)
	if !ok {
		return fmt.Errorf("snippet '%s' not found", name)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", snip.Command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", snip.Command)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
