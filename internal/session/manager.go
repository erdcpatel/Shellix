package session

import (
	"fmt"
	"sync"
	"time"
)

// SessionType defines the nature of the terminal session
type SessionType string

const (
	SessionLocal SessionType = "local"
	SessionSSH   SessionType = "ssh"
)

// Session represents an active terminal or remote server connection tab
type Session struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Type      SessionType `json:"type"`
	Target    string      `json:"target"` // host profile or shell binary
	StartedAt time.Time   `json:"started_at"`
	IsActive  bool        `json:"is_active"`
}

// Manager tracks and orchestrates concurrent active sessions/tabs
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	counter  int
}

// NewManager initializes the in-memory session registry
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// RegisterSession adds an active session tab
func (m *Manager) RegisterSession(title string, sType SessionType, target string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counter++
	id := fmt.Sprintf("tab-%d", m.counter)
	s := &Session{
		ID:        id,
		Title:     title,
		Type:      sType,
		Target:    target,
		StartedAt: time.Now(),
		IsActive:  true,
	}

	m.sessions[id] = s
	return s
}

// ListSessions returns all tracked sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Session
	for _, s := range m.sessions {
		list = append(list, s)
	}
	return list
}

// GetSession returns a session by ID
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[id]
	return s, ok
}

// CloseSession marks a session closed and removes it
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session '%s' not found", id)
	}

	s.IsActive = false
	delete(m.sessions, id)
	return nil
}
