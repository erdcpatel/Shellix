package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDoctorDiagnostics(t *testing.T) {
	app := NewAppContext()

	// Redirect stdout to avoid noisy test output
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	runDoctorDiagnostics(app)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if len(output) == 0 {
		t.Errorf("expected doctor diagnostics output, got empty string")
	}

	if strings.Contains(output, "Java Multi-JDK Ecosystem") || strings.Contains(output, "JAVA_HOME") {
		t.Errorf("expected doctor diagnostics not to contain Java/JDK references, got:\n%s", output)
	}

	expectedSections := []string{
		"Git Subsystem",
		"Security, Keyring & Permissions",
		"SSH & Known Hosts Hardening",
		"Automation Flow Engine",
	}
	for _, sec := range expectedSections {
		if !strings.Contains(output, sec) {
			t.Errorf("expected doctor diagnostics to contain section %q", sec)
		}
	}
}
