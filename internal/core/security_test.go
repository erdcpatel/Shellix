package core

import (
	"strings"
	"testing"
)

func TestSecretSanitizer(t *testing.T) {
	sanitizer := NewSecretSanitizer()

	rawSecret := "ghp_mySuperSecretGitHubPersonalAccessToken99"
	sanitizer.RegisterSecret(rawSecret)

	logLine := "Command failed while connecting with token: ghp_mySuperSecretGitHubPersonalAccessToken99 at host"
	sanitized := sanitizer.Sanitize(logLine)

	if strings.Contains(sanitized, rawSecret) {
		t.Errorf("expected secret to be redacted, got: %s", sanitized)
	}

	if !strings.Contains(sanitized, "[REDACTED_SECRET]") {
		t.Errorf("expected [REDACTED_SECRET] placeholder, got: %s", sanitized)
	}
}

func TestSecretSanitizerShortStringIgnored(t *testing.T) {
	sanitizer := NewSecretSanitizer()
	sanitizer.RegisterSecret("abc") // < 4 chars

	text := "the alphabet begins with abc and ends with z"
	if sanitizer.Sanitize(text) != text {
		t.Errorf("short string should not have been redacted")
	}
}
