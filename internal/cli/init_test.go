package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootCommandAliases(t *testing.T) {
	app := NewAppContext()
	rootCmd := NewRootCmd(app)

	if len(rootCmd.Aliases) < 2 {
		t.Fatalf("expected at least 2 aliases on root command, got: %v", rootCmd.Aliases)
	}

	hasSX := false
	hasSLX := false
	for _, a := range rootCmd.Aliases {
		if a == "sx" {
			hasSX = true
		}
		if a == "slx" {
			hasSLX = true
		}
	}

	if !hasSX || !hasSLX {
		t.Errorf("expected sx and slx aliases on root command, got: %v", rootCmd.Aliases)
	}
}

func TestShellIntegrationOutput(t *testing.T) {
	shells := []string{"zsh", "bash", "fish", "powershell"}

	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			r, w, _ := os.Pipe()
			oldStdout := os.Stdout
			os.Stdout = w

			err := outputShellIntegration(sh)

			_ = w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("outputShellIntegration failed for %s: %v", sh, err)
			}

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			out := buf.String()

			if !strings.Contains(out, "sx") || !strings.Contains(out, "slx") {
				t.Errorf("expected shell integration for %s to configure sx and slx aliases, got: %s", sh, out)
			}
		})
	}

	// Unsupported shell
	if err := outputShellIntegration("unsupported_shell"); err == nil {
		t.Errorf("expected error for unsupported shell")
	}
}
