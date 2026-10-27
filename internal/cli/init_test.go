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

func TestGenerateWindowsShortcutScript(t *testing.T) {
	target := `C:\Program Files\Shellix\shellix.exe`
	workingDir := `C:\Users\Developer`

	script := GenerateWindowsShortcutScript(target, workingDir)

	expectedSnippets := []string{
		"New-Object -ComObject WScript.Shell",
		"$desktop = $ws.SpecialFolders.Item('Desktop')",
		"Join-Path $desktop 'Shellix.lnk'",
		"$shortcut = $ws.CreateShortcut($shortcutPath)",
		"$shortcut.TargetPath = 'C:\\Program Files\\Shellix\\shellix.exe'",
		"$shortcut.WorkingDirectory = 'C:\\Users\\Developer'",
		"$shortcut.IconLocation = 'C:\\Program Files\\Shellix\\shellix.exe,0'",
		"$shortcut.Save()",
		"Write-Output $shortcutPath",
	}

	for _, s := range expectedSnippets {
		if !strings.Contains(script, s) {
			t.Errorf("expected shortcut script to contain %q, but got:\n%s", s, script)
		}
	}

	t.Run("escaping single quotes", func(t *testing.T) {
		tWithQuotes := `C:\User's Tool\shellix.exe`
		wWithQuotes := `C:\User's Home`
		s := GenerateWindowsShortcutScript(tWithQuotes, wWithQuotes)
		if !strings.Contains(s, "C:\\User''s Tool\\shellix.exe") {
			t.Errorf("expected single quote in target to be escaped, got:\n%s", s)
		}
		if !strings.Contains(s, "C:\\User''s Home") {
			t.Errorf("expected single quote in working dir to be escaped, got:\n%s", s)
		}
	})
}

func TestInitDesktopCommand(t *testing.T) {
	app := NewAppContext()
	initCmd := NewInitCmd(app)

	// Subcommand routing
	initCmd.SetArgs([]string{"desktop"})

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := initCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("initCmd.Execute(desktop) failed: %v", err)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Desktop") {
		t.Errorf("expected output to mention Desktop, got: %s", output)
	}
}

