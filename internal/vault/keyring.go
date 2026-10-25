package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"shellix/internal/core"

	"github.com/zalando/go-keyring"
)

const ServiceName = "shellix"

// Vault manages secrets using OS-native keychains with encrypted fallback
type Vault struct {
	mu           sync.RWMutex
	fallbackPath string
	useFallback  bool
}

// FallbackStore is used if OS keyring is unavailable
type FallbackStore struct {
	Keys map[string]string `json:"keys"`
}

// NewVault initializes the vault
func NewVault(customFallbackPath string) *Vault {
	path := customFallbackPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".shellix", "vault.enc")
	}

	return &Vault{
		fallbackPath: path,
	}
}

// ProviderName reports which credential backend is active
func (v *Vault) ProviderName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows Credential Manager (wincred)"
	case "darwin":
		return "macOS Keychain"
	case "linux":
		return "Linux Secret Service / DBus"
	default:
		return "Encrypted Local File Vault (AES-256-GCM)"
	}
}

// Set stores a secret in the OS credential manager
func (v *Vault) Set(key, secret string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Register secret with sanitizer immediately to prevent leakage
	core.GlobalSanitizer.RegisterSecret(secret)

	// Try OS keyring first
	err := keyring.Set(ServiceName, key, secret)
	if err == nil {
		v.trackKey(key)
		return nil
	}

	// Fallback to local encrypted vault
	return v.setFallback(key, secret)
}

// Get retrieves a secret by key
func (v *Vault) Get(key string) (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Try OS keyring first
	val, err := keyring.Get(ServiceName, key)
	if err == nil {
		core.GlobalSanitizer.RegisterSecret(val)
		return val, nil
	}

	// Fallback to local encrypted vault
	val, err = v.getFallback(key)
	if err == nil {
		core.GlobalSanitizer.RegisterSecret(val)
		return val, nil
	}

	return "", fmt.Errorf("credential '%s' not found: %w", key, err)
}

// Delete removes a secret
func (v *Vault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	_ = keyring.Delete(ServiceName, key)
	v.untrackKey(key)
	_ = v.deleteFallback(key)
	return nil
}

// ListKeys returns names of stored credentials
func (v *Vault) ListKeys() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.listKeysInternal()
}

func (v *Vault) listKeysInternal() []string {
	indexFile := v.getIndexFilePath()
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return []string{}
	}

	var keys []string
	_ = json.Unmarshal(data, &keys)
	return keys
}

func (v *Vault) getIndexFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".shellix", "vault_index.json")
}

func (v *Vault) trackKey(key string) {
	keys := v.listKeysInternal()
	for _, k := range keys {
		if k == key {
			return
		}
	}
	keys = append(keys, key)
	data, _ := json.Marshal(keys)
	_ = os.WriteFile(v.getIndexFilePath(), data, 0600)
}

func (v *Vault) untrackKey(key string) {
	keys := v.listKeysInternal()
	filtered := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			filtered = append(filtered, k)
		}
	}
	data, _ := json.Marshal(filtered)
	_ = os.WriteFile(v.getIndexFilePath(), data, 0600)
}

// AES-256-GCM fallback encryption routines

func getMachineKey() []byte {
	hostname, _ := os.Hostname()
	salt := "shellix-vault-key-salt"
	hash := sha256.Sum256([]byte(hostname + salt))
	return hash[:]
}

func (v *Vault) setFallback(key, secret string) error {
	store := v.loadFallbackStore()
	store.Keys[key] = secret

	plainData, err := json.Marshal(store)
	if err != nil {
		return err
	}

	keyBytes := getMachineKey()
	block, err := aes.NewCipher(keyBytes)
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
	v.trackKey(key)

	_ = os.MkdirAll(filepath.Dir(v.fallbackPath), 0700)
	return os.WriteFile(v.fallbackPath, ciphertext, 0600)
}

func (v *Vault) getFallback(key string) (string, error) {
	store := v.loadFallbackStore()
	val, ok := store.Keys[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found", key)
	}
	return val, nil
}

func (v *Vault) deleteFallback(key string) error {
	store := v.loadFallbackStore()
	delete(store.Keys, key)

	plainData, err := json.Marshal(store)
	if err != nil {
		return err
	}

	keyBytes := getMachineKey()
	block, err := aes.NewCipher(keyBytes)
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
	return os.WriteFile(v.fallbackPath, ciphertext, 0600)
}

func (v *Vault) loadFallbackStore() FallbackStore {
	store := FallbackStore{Keys: make(map[string]string)}

	ciphertext, err := os.ReadFile(v.fallbackPath)
	if err != nil {
		return store
	}

	keyBytes := getMachineKey()
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return store
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return store
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return store
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return store
	}

	_ = json.Unmarshal(plain, &store)
	return store
}
