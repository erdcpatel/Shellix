package envvault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvVaultCRUDAndEncryption(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-env-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// 1. Set variables
	if err := mgr.Set("DATABASE_URL", "postgres://user:secretpass@localhost:5432/mydb"); err != nil {
		t.Fatalf("Set DATABASE_URL failed: %v", err)
	}
	if err := mgr.Set("PORT", "8080"); err != nil {
		t.Fatalf("Set PORT failed: %v", err)
	}

	// 2. Get variables
	val, ok := mgr.Get("DATABASE_URL")
	if !ok || val != "postgres://user:secretpass@localhost:5432/mydb" {
		t.Errorf("expected DATABASE_URL to match, got %s (ok=%v)", val, ok)
	}

	// 3. Verify .env.vault exists and is encrypted on disk
	vaultFile := filepath.Join(tmpDir, ".env.vault")
	cipherData, err := os.ReadFile(vaultFile)
	if err != nil {
		t.Fatalf("failed reading .env.vault file: %v", err)
	}
	if string(cipherData) == "" || (len(cipherData) > 0 && string(cipherData) == "postgres://user:secretpass@localhost:5432/mydb") {
		t.Errorf("plaintext secret found in .env.vault file!")
	}

	// 4. Test masking in list summaries
	summaries := mgr.ListSummaries(false)
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	for _, s := range summaries {
		if s.Key == "DATABASE_URL" {
			if s.Masked == s.Value {
				t.Errorf("expected DATABASE_URL to be masked")
			}
		}
	}

	// 5. Tamper detection: modifying ciphertext should fail decryption
	tampered := make([]byte, len(cipherData))
	copy(tampered, cipherData)
	tampered[len(tampered)-1] ^= 0xFF // flip bit in GCM auth tag
	_ = os.WriteFile(vaultFile, tampered, 0600)

	_, err = NewManager(tmpDir)
	if err == nil {
		t.Errorf("expected error opening tampered vault, got nil")
	}

	// 6. Delete variable
	_ = os.WriteFile(vaultFile, cipherData, 0600)
	mgr2, _ := NewManager(tmpDir)
	if err := mgr2.Delete("PORT"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, exists := mgr2.Get("PORT"); exists {
		t.Errorf("expected PORT to be deleted")
	}
}

func TestBuildEnvSlice(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-env-slice-*")
	if err != nil {
		t.Fatalf("failed temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr, _ := NewManager(tmpDir)
	_ = mgr.Set("TEST_INJECTED_VAR", "super-secret-42")

	envSlice := mgr.BuildEnvSlice()
	found := false
	for _, e := range envSlice {
		if e == "TEST_INJECTED_VAR=super-secret-42" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected TEST_INJECTED_VAR in built env slice")
	}
}
