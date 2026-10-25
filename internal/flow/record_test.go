package flow

import (
	"strings"
	"testing"
	"time"
)

func TestSynthesizeStepName(t *testing.T) {
	cases := []struct {
		cmd      string
		expected string
	}{
		{"go test -v ./...", "run-go-tests"},
		{"go build ./cmd/shellix", "build-go-binary"},
		{"go vet ./...", "vet-go-code"},
		{"make build", "make-build"},
		{"npm test", "npm-test"},
		{"mvn clean install", "java-build"},
		{"docker build -t app .", "docker-build"},
		{"curl -s https://api.github.com", "http-request"},
		{"git status", "git-status"},
		{"python3 script.py", "python3-1"},
	}

	for _, c := range cases {
		got := SynthesizeStepName(c.cmd, 0)
		if got != c.expected {
			t.Errorf("for %q, expected %q, got %q", c.cmd, c.expected, got)
		}
	}
}

func TestSynthesizeFlowYAML(t *testing.T) {
	session := RecordSession{
		FlowName:    "my-test-pipeline",
		Description: "Automated verification flow",
		Commands: []RecordedCommand{
			{
				Command:  "go vet ./...",
				Duration: 500 * time.Millisecond,
			},
			{
				Command:  "go test -v ./...",
				Duration: 1200 * time.Millisecond,
			},
			// Duplicate tool to test name collision disambiguation
			{
				Command:  "go test -race ./...",
				Duration: 3000 * time.Millisecond,
			},
			{
				Command:  "make build",
				Duration: 850 * time.Millisecond,
			},
			// Command with embedded secret
			{
				Command:  "curl -H 'Authorization: token ghp_SECRET1234567890abcdef' https://api.github.com",
				Duration: 200 * time.Millisecond,
			},
		},
	}

	yamlStr, err := SynthesizeFlowYAML(session)
	if err != nil {
		t.Fatalf("SynthesizeFlowYAML failed: %v", err)
	}

	// 1. Verify secrets are redacted in synthesized YAML
	if strings.Contains(yamlStr, "ghp_SECRET1234567890abcdef") {
		t.Errorf("secret token found in synthesized flow YAML")
	}

	// 2. Verify duplicate name disambiguation
	if !strings.Contains(yamlStr, "run-go-tests-2") {
		t.Errorf("expected disambiguated step name 'run-go-tests-2' in YAML:\n%s", yamlStr)
	}

	// 3. Verify parseability with Shellix Flow parser
	parsed, err := Parse([]byte(yamlStr))
	if err != nil {
		t.Fatalf("failed to parse synthesized YAML: %v", err)
	}

	if parsed.Name != "my-test-pipeline" {
		t.Errorf("expected flow name 'my-test-pipeline', got %q", parsed.Name)
	}
	if len(parsed.Steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(parsed.Steps))
	}
}
