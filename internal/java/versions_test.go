package java

import (
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverAllJDKs(t *testing.T) {
	jdks, err := DiscoverAllJDKs()
	if err != nil {
		t.Fatalf("unexpected discovery error: %v", err)
	}

	t.Logf("Discovered %d JDK(s) on host (%s/%s)", len(jdks), runtime.GOOS, runtime.GOARCH)
	for i, j := range jdks {
		t.Logf("  [%d] v%s (major %d) by %s -> %s [active=%v]",
			i+1, j.Version, j.Major, j.Vendor, j.HomePath, j.IsActive)
	}

	if len(jdks) == 0 {
		t.Log("No system JDKs found, skipping match tests")
		return
	}

	// Test matching first discovered JDK
	first := jdks[0]
	matched, err := FindMatchingJDK(first.Version)
	if err != nil {
		t.Fatalf("failed to match first JDK: %v", err)
	}
	if matched.HomePath != first.HomePath {
		t.Errorf("expected path %s, got %s", first.HomePath, matched.HomePath)
	}

	// Test export string generation
	exp := GenerateShellExport(matched)
	if !strings.Contains(exp, matched.HomePath) {
		t.Errorf("export command should contain home path, got: %s", exp)
	}
}

func TestParseMajor(t *testing.T) {
	cases := map[string]int{
		"17.0.17": 17,
		"19.0.2":  19,
		"25.0.1":  25,
		"1.8.0_352": 8,
		"11":      11,
	}

	for input, expected := range cases {
		got := parseMajor(input)
		if got != expected {
			t.Errorf("parseMajor(%s): expected %d, got %d", input, expected, got)
		}
	}
}
