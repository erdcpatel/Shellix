package envvault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"shellix/internal/core"
	"shellix/internal/vault"
)

const (
	DefaultVaultFile = ".env.vault"
	CurrentVersion   = 1
)

// Manager manages encrypted project environment variables
type Manager struct {
	dir       string
	vaultPath string
	key       []byte
	data      *EnvVaultData
}

// NewManager initializes the environment vault for the given workspace directory
func NewManager(dir string) (*Manager, error) {
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	vaultPath := filepath.Join(absDir, DefaultVaultFile)

	key, err := resolveMasterKey(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed resolving vault encryption key: %w", err)
	}

	mgr := &Manager{
		dir:       absDir,
		vaultPath: vaultPath,
		key:       key,
		data: &EnvVaultData{
			Version:   CurrentVersion,
			Project:   filepath.Base(absDir),
			Variables: make(map[string]string),
			UpdatedAt: time.Now(),
		},
	}

	if err := mgr.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed reading .env.vault: %w", err)
	}

	return mgr, nil
}

func resolveMasterKey(absDir string) ([]byte, error) {
	// 1. Check SHELLIX_ENV_KEY environment variable
	if envKey := strings.TrimSpace(os.Getenv("SHELLIX_ENV_KEY")); envKey != "" {
		h := sha256.Sum256([]byte(envKey))
		return h[:], nil
	}

	// 2. Check native OS Vault (WinCred / Keychain)
	dirHash := sha256.Sum256([]byte(absDir))
	vaultKeyID := fmt.Sprintf("env_vault_%s", hex.EncodeToString(dirHash[:8]))

	v := vault.NewVault("")
	if storedHex, err := v.Get(vaultKeyID); err == nil && len(strings.TrimSpace(storedHex)) == 64 {
		keyBytes, err := hex.DecodeString(strings.TrimSpace(storedHex))
		if err == nil && len(keyBytes) == 32 {
			return keyBytes, nil
		}
	}

	// 3. Generate new 32-byte cryptographically secure key and persist in OS Vault
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, fmt.Errorf("failed generating crypto key: %w", err)
	}

	_ = v.Set(vaultKeyID, hex.EncodeToString(newKey))
	return newKey, nil
}

func (m *Manager) load() error {
	cipherData, err := os.ReadFile(m.vaultPath)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherData) < nonceSize {
		return fmt.Errorf("corrupted vault file: payload too short")
	}

	nonce, ciphertext := cipherData[:nonceSize], cipherData[nonceSize:]
	plainData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed (invalid key or tampered vault): %w", err)
	}

	var data EnvVaultData
	if err := json.Unmarshal(plainData, &data); err != nil {
		return fmt.Errorf("failed parsing vault data: %w", err)
	}

	m.data = &data
	if m.data.Variables == nil {
		m.data.Variables = make(map[string]string)
	}

	// Register all sensitive variables with GlobalSanitizer immediately
	for k, v := range m.data.Variables {
		if core.IsSensitiveEnvKey(k) || len(v) >= 8 {
			core.GlobalSanitizer.RegisterSecret(v)
		}
	}

	return nil
}

func (m *Manager) save() error {
	m.data.UpdatedAt = time.Now()
	plainData, err := json.Marshal(m.data)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, plainData, nil)
	return os.WriteFile(m.vaultPath, ciphertext, 0600)
}

// Set stores an environment variable in the encrypted vault
func (m *Manager) Set(key, val string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("variable key cannot be empty")
	}

	m.data.Variables[key] = val
	if core.IsSensitiveEnvKey(key) || len(val) >= 8 {
		core.GlobalSanitizer.RegisterSecret(val)
	}
	return m.save()
}

// Get retrieves a variable value by key
func (m *Manager) Get(key string) (string, bool) {
	val, exists := m.data.Variables[key]
	return val, exists
}

// ListSummaries returns all variables with masking applied
func (m *Manager) ListSummaries(showSecrets bool) []EnvVarSummary {
	var list []EnvVarSummary
	for k, v := range m.data.Variables {
		isSecret := core.IsSensitiveEnvKey(k) || len(v) >= 12
		masked := v
		if isSecret && !showSecrets {
			if len(v) <= 4 {
				masked = "••••"
			} else {
				masked = v[:2] + strings.Repeat("•", len(v)-4) + v[len(v)-2:]
			}
		}
		list = append(list, EnvVarSummary{
			Key:      k,
			Value:    v,
			Masked:   masked,
			IsSecret: isSecret,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Key < list[j].Key
	})
	return list
}

// Delete removes a variable from the vault
func (m *Manager) Delete(key string) error {
	delete(m.data.Variables, key)
	return m.save()
}

// BuildEnvSlice constructs os.Environ() with injected vault variables
func (m *Manager) BuildEnvSlice() []string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Override with vault variables
	for k, v := range m.data.Variables {
		envMap[k] = v
	}

	var out []string
	for k, v := range envMap {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

// RunWithEnv executes a command with the decrypted vault variables injected
func (m *Manager) RunWithEnv(command string, args []string) error {
	c := exec.Command(command, args...)
	c.Env = m.BuildEnvSlice()
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
