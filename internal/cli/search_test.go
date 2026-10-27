package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchPathsAlias(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-cli-search-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "sample.txt")
	_ = os.WriteFile(testFile, []byte("UniqueCliSearchToken\n"), 0644)

	t.Run("using --path singular", func(t *testing.T) {
		app := NewAppContext()
		cmd := NewSearchCmd(app)
		cmd.SetArgs([]string{"UniqueCliSearchToken", "--path", tempDir})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := cmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("search with --path failed: %v", err)
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "UniqueCliSearchToken") {
			t.Errorf("expected search output to contain match, got: %s", buf.String())
		}
	})

	t.Run("using --paths plural alias", func(t *testing.T) {
		app := NewAppContext()
		cmd := NewSearchCmd(app)
		cmd.SetArgs([]string{"UniqueCliSearchToken", "--paths", tempDir})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := cmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("search with --paths failed: %v", err)
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "UniqueCliSearchToken") {
			t.Errorf("expected search output with --paths to contain match, got: %s", buf.String())
		}
	})

	t.Run("using -p shorthand", func(t *testing.T) {
		app := NewAppContext()
		cmd := NewSearchCmd(app)
		cmd.SetArgs([]string{"UniqueCliSearchToken", "-p", tempDir})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := cmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("search with -p failed: %v", err)
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "UniqueCliSearchToken") {
			t.Errorf("expected search output with -p to contain match, got: %s", buf.String())
		}
	})

	t.Run("nonexistent path displays warning", func(t *testing.T) {
		app := NewAppContext()
		cmd := NewSearchCmd(app)
		ghostPath := filepath.Join(tempDir, "nonexistent-dir-for-warning")
		cmd.SetArgs([]string{"UniqueCliSearchToken", "--paths", ghostPath})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := cmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("search should succeed with warning, got: %v", err)
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		out := buf.String()
		if !strings.Contains(out, "Warning:") || !strings.Contains(out, "does not exist") {
			t.Errorf("expected warning in output for nonexistent path, got: %s", out)
		}
	})
}
