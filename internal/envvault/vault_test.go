package envvault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvVaultMultiEnvironmentAndRotation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-env-multienv-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed creating manager: %v", err)
	}

	// 1. Default environment
	if err := mgr.Set("DEFAULT_KEY", "def_val"); err != nil {
		t.Fatalf("failed setting default key: %v", err)
	}

	// 2. Staging environment
	if err := mgr.SetEnv("staging", "STAGING_DB", "postgres://staging"); err != nil {
		t.Fatalf("failed setting staging key: %v", err)
	}

	// 3. Prod environment
	if err := mgr.SetEnv("prod", "PROD_API_KEY", "sk_live_999999999999"); err != nil {
		t.Fatalf("failed setting prod key: %v", err)
	}

	// Check listing environments
	envs := mgr.ListEnvironments()
	if len(envs) < 3 {
		t.Errorf("expected at least 3 environments, got %v", envs)
	}

	// Check retrieval
	val, ok := mgr.GetEnv("staging", "STAGING_DB")
	if !ok || val != "postgres://staging" {
		t.Errorf("unexpected staging db: %s", val)
	}

	// Check masking
	summaries := mgr.ListSummariesEnv("prod", false)
	if len(summaries) != 1 || summaries[0].Masked == summaries[0].Value {
		t.Errorf("expected prod key to be masked, got: %+v", summaries)
	}

	// 4. Test Key Rotation
	oldKey := make([]byte, len(mgr.key))
	copy(oldKey, mgr.key)

	newKeyHex, err := mgr.RotateMasterKey()
	if err != nil {
		t.Fatalf("RotateMasterKey failed: %v", err)
	}
	if len(newKeyHex) != 64 {
		t.Errorf("expected 64-char hex key, got %s", newKeyHex)
	}

	// Reopen manager and verify data still decrypts
	reloaded, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed reopening vault after rotation: %v", err)
	}

	val, ok = reloaded.GetEnv("staging", "STAGING_DB")
	if !ok || val != "postgres://staging" {
		t.Errorf("failed decrypting staging value after rotation: %v", val)
	}

	// 5. Test Shell Hook generation
	pwshHook, err := GenerateHook("powershell")
	if err != nil || !strings.Contains(pwshHook, "Set-ShellixProjectEnv") {
		t.Errorf("unexpected powershell hook: %v", pwshHook)
	}

	zshHook, err := GenerateHook("zsh")
	if err != nil || !strings.Contains(zshHook, "_shellix_env_hook") {
		t.Errorf("unexpected zsh hook: %v", zshHook)
	}
}

func TestEnvVaultCRUDAndEncryption(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shellix-env-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Test Set and Get
	if err := mgr.Set("API_KEY", "sk_live_1234567890abcdef"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	val, exists := mgr.Get("API_KEY")
	if !exists || val != "sk_live_1234567890abcdef" {
		t.Fatalf("unexpected value: %s (exists=%v)", val, exists)
	}

	// Test Summaries masking
	summaries := mgr.ListSummaries(false)
	if len(summaries) != 1 || summaries[0].Masked == summaries[0].Value {
		t.Fatalf("expected masked secret, got: %+v", summaries[0])
	}

	// Test Tamper detection: flip bit in ciphertext
	vaultPath := filepath.Join(tmpDir, DefaultVaultFile)
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0xFF
	if err := os.WriteFile(vaultPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	tamperedMgr, err := NewManager(tmpDir)
	if err == nil && tamperedMgr != nil {
		t.Fatalf("expected error loading tampered vault, got success")
	}
}
