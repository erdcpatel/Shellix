package core

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(token|secret|password|passwd|api_key|apikey|private_key|auth|bearer|credential)`)

// IsSensitiveEnvKey reports whether an environment variable key matches sensitive patterns
func IsSensitiveEnvKey(key string) bool {
	return sensitiveKeyPattern.MatchString(key)
}

// SecretSanitizer masks sensitive strings from logs and terminal outputs
type SecretSanitizer struct {
	mu      sync.RWMutex
	secrets []string
}

// GlobalSanitizer is the application-wide scrubber
var GlobalSanitizer = NewSecretSanitizer()

// NewSecretSanitizer creates an isolated sanitizer
func NewSecretSanitizer() *SecretSanitizer {
	s := &SecretSanitizer{
		secrets: make([]string, 0),
	}
	s.AutoRegisterEnv()
	return s
}

// RegisterSecret adds a sensitive value to the redaction dictionary
func (s *SecretSanitizer) RegisterSecret(value string) {
	if len(strings.TrimSpace(value)) < 4 {
		// Ignore trivial or very short strings to avoid over-redacting common words
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.secrets {
		if existing == value {
			return
		}
	}
	s.secrets = append(s.secrets, value)
}

// AutoRegisterEnv scans system environment variables for sensitive names and registers their values
func (s *SecretSanitizer) AutoRegisterEnv() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key, val := parts[0], parts[1]
			if sensitiveKeyPattern.MatchString(key) && len(strings.TrimSpace(val)) >= 4 {
				s.RegisterSecret(val)
			}
		}
	}
}

// Sanitize replaces all registered secrets with [REDACTED_SECRET]
func (s *SecretSanitizer) Sanitize(input string) string {
	if input == "" {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := input
	for _, secret := range s.secrets {
		if strings.Contains(result, secret) {
			result = strings.ReplaceAll(result, secret, "[REDACTED_SECRET]")
		}
	}
	return result
}

// EnsureSecureDir creates or validates a directory with strict 0700 permissions
func EnsureSecureDir(path string) error {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0700)
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("path %s exists but is not a directory", path)
	}
	return os.Chmod(path, 0700)
}

// EnsureSecureFile validates that a file has strict 0600 permissions
func EnsureSecureFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("path %s is a directory, expected file", path)
	}
	return os.Chmod(path, 0600)
}
