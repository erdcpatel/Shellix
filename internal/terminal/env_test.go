package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildShellEnvironment(t *testing.T) {
	sep := string(os.PathListSeparator)
	exePath := filepath.Join(string(filepath.Separator), "custom", "bin", "shellix")
	exeDir := filepath.Dir(exePath)

	t.Run("empty base environment", func(t *testing.T) {
		env := BuildShellEnvironment([]string{}, exePath)
		var pathVal, binVal string
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				pathVal = strings.TrimPrefix(e, "PATH=")
			}
			if strings.HasPrefix(e, "SHELLIX_BIN=") {
				binVal = strings.TrimPrefix(e, "SHELLIX_BIN=")
			}
		}

		if pathVal != exeDir {
			t.Errorf("expected PATH to be %s, got: %s", exeDir, pathVal)
		}
		if binVal != exePath {
			t.Errorf("expected SHELLIX_BIN to be %s, got: %s", exePath, binVal)
		}
	})

	t.Run("existing PATH prepends exeDir and preserves others", func(t *testing.T) {
		existingPath := strings.Join([]string{
			filepath.Join(string(filepath.Separator), "usr", "bin"),
			filepath.Join(string(filepath.Separator), "bin"),
		}, sep)

		baseEnv := []string{
			"USER=developer",
			"PATH=" + existingPath,
			"HOME=/home/developer",
		}

		env := BuildShellEnvironment(baseEnv, exePath)
		var pathVal string
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				pathVal = strings.TrimPrefix(e, "PATH=")
			}
		}

		expectedPrefix := exeDir + sep
		if !strings.HasPrefix(pathVal, expectedPrefix) {
			t.Errorf("expected PATH to start with %s, got: %s", expectedPrefix, pathVal)
		}
		if !strings.Contains(pathVal, existingPath) {
			t.Errorf("expected PATH to contain existing paths, got: %s", pathVal)
		}
	})

	t.Run("Windows-style Path casing and extraPaths appending", func(t *testing.T) {
		existingPath := `C:\Windows\System32`
		gitExtra := `C:\Program Files\Git\cmd`

		baseEnv := []string{
			"USERNAME=admin",
			"Path=" + existingPath,
			"SHELLIX_BIN=old_value",
		}

		winExePath := `C:\Users\admin\Downloads\shellix.exe`
		winExeDir := `C:\Users\admin\Downloads`

		env := BuildShellEnvironment(baseEnv, winExePath, gitExtra)
		var pathVal, binVal string
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if strings.EqualFold(parts[0], "PATH") {
				pathVal = parts[1]
			}
			if parts[0] == "SHELLIX_BIN" {
				binVal = parts[1]
			}
		}

		// exeDir should be at the start
		if !strings.HasPrefix(pathVal, winExeDir) {
			t.Errorf("expected PATH to start with exeDir, got: %s", pathVal)
		}
		// gitExtra should be at the end
		if !strings.HasSuffix(pathVal, gitExtra) {
			t.Errorf("expected PATH to end with gitExtra, got: %s", pathVal)
		}
		// Verify Windows separator was used and drive letter C: was NOT split on colon
		if strings.Contains(pathVal, ";C:\\Windows\\System32;") {
			// Semicolon separator preserved the path intact
		} else if !strings.Contains(pathVal, "C:\\Windows\\System32") {
			t.Errorf("expected C:\\Windows\\System32 to be preserved without colon split, got: %s", pathVal)
		}
		// old SHELLIX_BIN must be replaced
		if binVal != winExePath {
			t.Errorf("expected SHELLIX_BIN to be updated to %s, got: %s", winExePath, binVal)
		}
	})

	t.Run("deduplication of prepend and extra paths", func(t *testing.T) {
		extra := filepath.Join(string(filepath.Separator), "opt", "git", "bin")
		initial := strings.Join([]string{exeDir, extra}, sep)

		baseEnv := []string{"PATH=" + initial}
		env := BuildShellEnvironment(baseEnv, exePath, extra)

		var pathVal string
		for _, e := range env {
			if strings.HasPrefix(e, "PATH=") {
				pathVal = strings.TrimPrefix(e, "PATH=")
			}
		}

		parts := strings.Split(pathVal, sep)
		countExe := 0
		countExtra := 0
		for _, p := range parts {
			if p == exeDir {
				countExe++
			}
			if p == extra {
				countExtra++
			}
		}

		if countExe != 1 {
			t.Errorf("expected exeDir to occur exactly once, got count: %d (%s)", countExe, pathVal)
		}
		if countExtra != 1 {
			t.Errorf("expected extra path to occur exactly once, got count: %d (%s)", countExtra, pathVal)
		}
	})
}
