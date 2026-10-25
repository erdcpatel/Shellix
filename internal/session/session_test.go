package session

import (
	"testing"
)

func TestSessionManager(t *testing.T) {
	mgr := NewManager()

	// 1. Register local session
	s1 := mgr.RegisterSession("Local Shell", SessionLocal, "/bin/zsh")
	if s1.ID != "tab-1" {
		t.Errorf("expected ID tab-1, got %s", s1.ID)
	}

	// 2. Register SSH session
	s2 := mgr.RegisterSession("Production DB", SessionSSH, "10.0.1.20")
	if s2.ID != "tab-2" {
		t.Errorf("expected ID tab-2, got %s", s2.ID)
	}

	// 3. List
	list := mgr.ListSessions()
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}

	// 4. Get & Close
	_, ok := mgr.GetSession("tab-1")
	if !ok {
		t.Errorf("expected tab-1 to be found")
	}

	err := mgr.CloseSession("tab-1")
	if err != nil {
		t.Fatalf("failed to close tab-1: %v", err)
	}

	if _, ok := mgr.GetSession("tab-1"); ok {
		t.Errorf("expected tab-1 to be removed")
	}
}
