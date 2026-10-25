package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileManagerCRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-ssh-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	filePath := filepath.Join(tempDir, "hosts.json")
	mgr, err := NewProfileManager(filePath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// 1. Add host
	p := HostProfile{
		Name: "prod-web-01",
		Host: "192.168.1.100",
		Port: 22,
		User: "deploy",
		Tags: []string{"web", "prod"},
	}
	err = mgr.AddHost(p)
	if err != nil {
		t.Fatalf("failed to add host: %v", err)
	}

	// 2. Get host
	got, ok := mgr.GetHost("prod-web-01")
	if !ok {
		t.Fatalf("expected host to be found")
	}
	if got.Host != "192.168.1.100" || got.User != "deploy" {
		t.Errorf("host profile mismatch: %+v", got)
	}

	// 3. List hosts
	list := mgr.ListHosts()
	if len(list) != 1 {
		t.Fatalf("expected 1 host, got %d", len(list))
	}

	// 4. Persistence reload
	mgr2, err := NewProfileManager(filePath)
	if err != nil {
		t.Fatalf("failed to reload manager: %v", err)
	}
	if _, ok := mgr2.GetHost("prod-web-01"); !ok {
		t.Errorf("expected host to persist across reloads")
	}

	// 5. Delete host
	_ = mgr.DeleteHost("prod-web-01")
	if _, ok := mgr.GetHost("prod-web-01"); ok {
		t.Errorf("expected host to be deleted")
	}
}
