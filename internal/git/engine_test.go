package git

import (
	"os"
	"testing"
)

func TestEngineStatusCurrentRepo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	engine := NewEngine(true)
	status, err := engine.GetStatus(cwd)
	if err != nil {
		t.Fatalf("unexpected error getting status: %v", err)
	}

	if status.Branch == "" {
		t.Errorf("expected branch to be populated")
	}
	if status.HeadCommit == "" {
		t.Errorf("expected latest head commit to be populated")
	}
}

func TestEngineStatusNonRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "non-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	engine := NewEngine(true)
	_, err = engine.GetStatus(tempDir)
	if err == nil {
		t.Errorf("expected error for non-repo directory, got nil")
	}
}
