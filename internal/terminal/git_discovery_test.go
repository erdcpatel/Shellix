package terminal

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGitPaths(t *testing.T) {
	candidates := CandidateGitPaths()
	if len(candidates) == 0 {
		t.Fatalf("expected non-empty Git candidate directories")
	}

	foundFallback := false
	for _, c := range candidates {
		if strings.Contains(c, "Git") {
			foundFallback = true
			break
		}
	}

	if !foundFallback {
		t.Errorf("expected candidate paths to contain Git directories, got: %v", candidates)
	}
}

func TestFindGitInPaths(t *testing.T) {
	mockDirs := []string{
		filepath.Join("nonexistent", "path1"),
		filepath.Join("nonexistent", "path2"),
		filepath.Join("valid", "git", "bin"),
		filepath.Join("another", "valid", "git"),
	}

	t.Run("git found at third path", func(t *testing.T) {
		targetFile := filepath.Join("valid", "git", "bin", "git.exe")
		checkFile := func(p string) bool {
			return p == targetFile
		}

		result := FindGitInPaths(mockDirs, checkFile)
		expected := filepath.Join("valid", "git", "bin")
		if result != expected {
			t.Errorf("expected %s, got: %s", expected, result)
		}
	})

	t.Run("git.cmd found at fourth path", func(t *testing.T) {
		targetFile := filepath.Join("another", "valid", "git", "git.cmd")
		checkFile := func(p string) bool {
			return p == targetFile
		}

		result := FindGitInPaths(mockDirs, checkFile)
		expected := filepath.Join("another", "valid", "git")
		if result != expected {
			t.Errorf("expected %s, got: %s", expected, result)
		}
	})

	t.Run("git not found anywhere", func(t *testing.T) {
		checkFile := func(p string) bool {
			return false
		}

		result := FindGitInPaths(mockDirs, checkFile)
		if result != "" {
			t.Errorf("expected empty string when git is missing, got: %s", result)
		}
	})

	t.Run("empty candidate list", func(t *testing.T) {
		result := FindGitInPaths(nil, func(p string) bool { return true })
		if result != "" {
			t.Errorf("expected empty result for nil candidates, got: %s", result)
		}
	})
}
