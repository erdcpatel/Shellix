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

func TestIsSensitiveEnvKey(t *testing.T) {
	sensitive := []string{
		"AWS_SECRET_ACCESS_KEY",
		"GITHUB_TOKEN",
		"DB_PASSWORD",
		"API_KEY",
		"PRIVATE_KEY",
		"AUTH_TOKEN",
		"MY_BEARER_TOKEN",
		"USER_PASSWD",
		"CREDENTIAL_STORE",
	}
	for _, key := range sensitive {
		if !IsSensitiveEnvKey(key) {
			t.Errorf("expected '%s' to be identified as sensitive", key)
		}
	}

	nonSensitive := []string{
		"PATH",
		"HOME",
		"USER",
		"SHELL",
		"GOOS",
		"TERM",
	}
	for _, key := range nonSensitive {
		if IsSensitiveEnvKey(key) {
			t.Errorf("expected '%s' to NOT be identified as sensitive", key)
		}
	}
}
