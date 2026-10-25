package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultFallbackEncryption(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellix-vault-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	fallbackPath := filepath.Join(tempDir, "vault.enc")
	v := NewVault(fallbackPath)

	testKey := "api_token"
	testVal := "ghp_1234567890abcdef"

	// 1. Test set fallback directly
	err = v.setFallback(testKey, testVal)
	if err != nil {
		t.Fatalf("failed to set secret: %v", err)
	}

	// 2. Test get fallback
	got, err := v.getFallback(testKey)
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if got != testVal {
		t.Errorf("expected '%s', got '%s'", testVal, got)
	}

	// 3. Test delete fallback
	err = v.deleteFallback(testKey)
	if err != nil {
		t.Fatalf("failed to delete secret: %v", err)
	}
	_, err = v.getFallback(testKey)
	if err == nil {
		t.Errorf("expected error after deleting secret")
	}
}

func TestVaultProviderName(t *testing.T) {
	v := NewVault("")
	pName := v.ProviderName()
	if pName == "" {
		t.Errorf("provider name should not be empty")
	}
	t.Logf("Active Credential Manager: %s", pName)
}
