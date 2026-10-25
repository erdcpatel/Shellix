package cli

import (
	"bytes"
	"io"
	"os"
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
}
