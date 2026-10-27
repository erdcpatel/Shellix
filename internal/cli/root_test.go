package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootZeroArgExecution(t *testing.T) {
	app := NewAppContext()
	runnerCalled := false
	receivedShell := ""

	app.ShellRunner = func(sh string) error {
		runnerCalled = true
		receivedShell = sh
		return nil
	}

	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{})

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("rootCmd.Execute() failed: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// 1. Verifies automatic transition into shell runner
	if !runnerCalled {
		t.Errorf("expected ShellRunner to be automatically called on zero arguments")
	}

	if receivedShell != app.Config.Shell.Default {
		t.Errorf("expected shell runner to receive default shell %q, got %q", app.Config.Shell.Default, receivedShell)
	}

	// 2. Verifies banner was printed
	if !strings.Contains(output, "Shellix") {
		t.Errorf("expected banner in output, got: %s", output)
	}

	// 3. Verifies superpowers hint was printed
	if !strings.Contains(output, "Quick Superpowers") {
		t.Errorf("expected superpowers in output, got: %s", output)
	}
}

func TestRootFlagHandling(t *testing.T) {
	t.Run("help flag does not launch subshell", func(t *testing.T) {
		app := NewAppContext()
		runnerCalled := false
		app.ShellRunner = func(sh string) error {
			runnerCalled = true
			return nil
		}

		rootCmd := NewRootCmd(app)
		rootCmd.SetArgs([]string{"--help"})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("rootCmd.Execute(--help) failed: %v", err)
		}

		if runnerCalled {
			t.Errorf("runner should NOT be called when --help is passed")
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "Usage:") {
			t.Errorf("expected help output, got: %s", buf.String())
		}
	})

	t.Run("version flag does not launch subshell", func(t *testing.T) {
		app := NewAppContext()
		runnerCalled := false
		app.ShellRunner = func(sh string) error {
			runnerCalled = true
			return nil
		}

		rootCmd := NewRootCmd(app)
		rootCmd.SetArgs([]string{"--version"})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("rootCmd.Execute(--version) failed: %v", err)
		}

		if runnerCalled {
			t.Errorf("runner should NOT be called when --version is passed")
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "shellix") {
			t.Errorf("expected version output, got: %s", buf.String())
		}
	})

	t.Run("version subcommand does not launch subshell", func(t *testing.T) {
		app := NewAppContext()
		runnerCalled := false
		app.ShellRunner = func(sh string) error {
			runnerCalled = true
			return nil
		}

		rootCmd := NewRootCmd(app)
		rootCmd.SetArgs([]string{"version"})

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		err := rootCmd.Execute()

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("rootCmd.Execute(version) failed: %v", err)
		}

		if runnerCalled {
			t.Errorf("runner should NOT be called when version subcommand is passed")
		}

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		if !strings.Contains(buf.String(), "version") {
			t.Errorf("expected version output, got: %s", buf.String())
		}
	})
}
