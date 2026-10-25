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
	DefaultEnvName   = "default"
	CurrentVersion   = 1
)

// Manager manages encrypted project environment variables
type Manager struct {
	dir        string
	vaultPath  string
	vaultKeyID string
	key        []byte
	data       *EnvVaultData
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

	key, vaultKeyID, err := resolveMasterKey(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed resolving vault encryption key: %w", err)
	}

	mgr := &Manager{
		dir:        absDir,
		vaultPath:  vaultPath,
		vaultKeyID: vaultKeyID,
		key:        key,
		data: &EnvVaultData{
			Version:      CurrentVersion,
			Project:      filepath.Base(absDir),
			Variables:    make(map[string]string),
			Environments: make(map[string]map[string]string),
			UpdatedAt:    time.Now(),
		},
	}

	if err := mgr.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed reading .env.vault: %w", err)
	}

	return mgr, nil
}

func resolveMasterKey(absDir string) ([]byte, string, error) {
	dirHash := sha256.Sum256([]byte(absDir))
	vaultKeyID := fmt.Sprintf("env_vault_%s", hex.EncodeToString(dirHash[:8]))

	// 1. Check SHELLIX_ENV_KEY environment variable
	if envKey := strings.TrimSpace(os.Getenv("SHELLIX_ENV_KEY")); envKey != "" {
		h := sha256.Sum256([]byte(envKey))
		return h[:], vaultKeyID, nil
	}

	// 2. Check native OS Vault (WinCred / Keychain)
	v := vault.NewVault("")
	if storedHex, err := v.Get(vaultKeyID); err == nil && len(strings.TrimSpace(storedHex)) == 64 {
		keyBytes, err := hex.DecodeString(strings.TrimSpace(storedHex))
		if err == nil && len(keyBytes) == 32 {
			return keyBytes, vaultKeyID, nil
		}
	}

	// 3. Generate new 32-byte cryptographically secure key and persist in OS Vault
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, "", fmt.Errorf("failed generating crypto key: %w", err)
	}

	_ = v.Set(vaultKeyID, hex.EncodeToString(newKey))
	return newKey, vaultKeyID, nil
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
	if m.data.Environments == nil {
		m.data.Environments = make(map[string]map[string]string)
	}

	// Sync legacy variables into default environment
	if len(m.data.Variables) > 0 {
		if _, exists := m.data.Environments[DefaultEnvName]; !exists {
			m.data.Environments[DefaultEnvName] = make(map[string]string)
		}
		for k, v := range m.data.Variables {
			m.data.Environments[DefaultEnvName][k] = v
		}
	}

	// Register all sensitive variables with GlobalSanitizer
	for _, envMap := range m.data.Environments {
		for k, v := range envMap {
			if core.IsSensitiveEnvKey(k) || len(v) >= 8 {
				core.GlobalSanitizer.RegisterSecret(v)
			}
		}
	}

	return nil
}

func (m *Manager) save() error {
	m.data.UpdatedAt = time.Now()

	// Sync default environment to legacy variables
	if defaultVars, exists := m.data.Environments[DefaultEnvName]; exists {
		m.data.Variables = defaultVars
	}

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

	// Atomic write via temp file to guard against corruption on crash/power loss
	tmpPath := fmt.Sprintf("%s.tmp.%d", m.vaultPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, ciphertext, 0600); err != nil {
		return fmt.Errorf("failed writing temporary vault file: %w", err)
	}
	return os.Rename(tmpPath, m.vaultPath)
}

func normalizeEnv(env string) string {
	clean := strings.TrimSpace(env)
	if clean == "" {
		return DefaultEnvName
	}
	return strings.ToLower(clean)
}

func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for i, r := range key {
		if i == 0 {
			if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && r != '_' {
				return false
			}
		} else {
			if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
				return false
			}
		}
	}
	return true
}

// Set stores an environment variable in the specified environment
func (m *Manager) SetEnv(env, key, val string) error {
	targetEnv := normalizeEnv(env)
	key = strings.TrimSpace(key)
	if !isValidEnvKey(key) {
		return fmt.Errorf("invalid variable name %q: must start with letter or underscore and contain only alphanumeric characters and underscores", key)
	}

	if m.data.Environments[targetEnv] == nil {
		m.data.Environments[targetEnv] = make(map[string]string)
	}

	m.data.Environments[targetEnv][key] = val
	if targetEnv == DefaultEnvName {
		m.data.Variables[key] = val
	}

	if core.IsSensitiveEnvKey(key) || len(val) >= 8 {
		core.GlobalSanitizer.RegisterSecret(val)
	}

	return m.save()
}

// Set stores an environment variable in the default environment
func (m *Manager) Set(key, val string) error {
	return m.SetEnv(DefaultEnvName, key, val)
}

// GetEnv retrieves a variable from the specified environment
func (m *Manager) GetEnv(env, key string) (string, bool) {
	targetEnv := normalizeEnv(env)
	envMap, exists := m.data.Environments[targetEnv]
	if !exists {
		return "", false
	}
	val, ok := envMap[key]
	return val, ok
}

// Get retrieves a variable value from default environment
func (m *Manager) Get(key string) (string, bool) {
	return m.GetEnv(DefaultEnvName, key)
}

// ListSummariesEnv returns all variables in the given environment with masking applied
func (m *Manager) ListSummariesEnv(env string, showSecrets bool) []EnvVarSummary {
	targetEnv := normalizeEnv(env)
	envMap := m.data.Environments[targetEnv]
	if envMap == nil && targetEnv == DefaultEnvName {
		envMap = m.data.Variables
	}

	var list []EnvVarSummary
	for k, v := range envMap {
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
			Env:      targetEnv,
		})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Key < list[j].Key
	})
	return list
}

// ListSummaries returns all default environment variables with masking applied
func (m *Manager) ListSummaries(showSecrets bool) []EnvVarSummary {
	return m.ListSummariesEnv(DefaultEnvName, showSecrets)
}

// ListEnvironments returns names of all configured environments
func (m *Manager) ListEnvironments() []string {
	var envs []string
	for k := range m.data.Environments {
		envs = append(envs, k)
	}
	if len(envs) == 0 {
		envs = append(envs, DefaultEnvName)
	}
	sort.Strings(envs)
	return envs
}

// DeleteEnv removes a variable from the specified environment
func (m *Manager) DeleteEnv(env, key string) error {
	targetEnv := normalizeEnv(env)
	if m.data.Environments[targetEnv] != nil {
		delete(m.data.Environments[targetEnv], key)
	}
	if targetEnv == DefaultEnvName {
		delete(m.data.Variables, key)
	}
	return m.save()
}

// Delete removes a variable from default environment
func (m *Manager) Delete(key string) error {
	return m.DeleteEnv(DefaultEnvName, key)
}

// BuildEnvSliceForEnv constructs os.Environ() with injected variables from specified environment
func (m *Manager) BuildEnvSliceForEnv(env string) []string {
	targetEnv := normalizeEnv(env)
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	vars := m.data.Environments[targetEnv]
	if vars == nil && targetEnv == DefaultEnvName {
		vars = m.data.Variables
	}

	for k, v := range vars {
		envMap[k] = v
	}

	var out []string
	for k, v := range envMap {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

// BuildEnvSlice constructs os.Environ() with injected default vault variables
func (m *Manager) BuildEnvSlice() []string {
	return m.BuildEnvSliceForEnv(DefaultEnvName)
}

// RunWithEnv executes a command with decrypted variables injected
func (m *Manager) RunWithEnv(env, command string, args []string) error {
	c := exec.Command(command, args...)
	c.Env = m.BuildEnvSliceForEnv(env)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RotateMasterKey generates a fresh 32-byte key, saves to OS Keyring, and re-encrypts vault
func (m *Manager) RotateMasterKey() (string, error) {
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return "", fmt.Errorf("failed generating new crypto key: %w", err)
	}

	v := vault.NewVault("")
	newKeyHex := hex.EncodeToString(newKey)
	if err := v.Set(m.vaultKeyID, newKeyHex); err != nil {
		return "", fmt.Errorf("failed saving new key to OS Keyring: %w", err)
	}

	m.key = newKey
	if err := m.save(); err != nil {
		return "", fmt.Errorf("failed re-encrypting vault with new key: %w", err)
	}

	return newKeyHex, nil
}

// GenerateHook returns shell auto-load scripts (PowerShell, Zsh, Bash)
func GenerateHook(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "powershell", "pwsh":
		return `# Shellix Environment Auto-Hook for PowerShell (Windows)
# Add to your $PROFILE: Invoke-Expression (& sx env hook powershell)

function Set-ShellixProjectEnv {
    $vaultPath = Join-Path (Get-Location) ".env.vault"
    if (Test-Path $vaultPath) {
        $exports = & sx env export --shell powershell 2>$null
        if ($exports) {
            Invoke-Expression ($exports -join "` + "`" + `n")
            $global:_SHELLIX_ENV_ACTIVE = $true
        }
    } elseif ($global:_SHELLIX_ENV_ACTIVE) {
        $unsets = & sx env export --unset --shell powershell 2>$null
        if ($unsets) {
            Invoke-Expression ($unsets -join "` + "`" + `n")
        }
        $global:_SHELLIX_ENV_ACTIVE = $false
    }
}

if (Test-Path Function:\prompt) {
    $global:_OldShellixPrompt = $Function:prompt
}
function prompt {
    Set-ShellixProjectEnv
    if ($global:_OldShellixPrompt) {
        & $global:_OldShellixPrompt
    } else {
        "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
    }
}
`, nil

	case "zsh":
		return `# Shellix Environment Auto-Hook for Zsh (macOS)
# Add to your ~/.zshrc: eval "$(sx env hook zsh)"

_shellix_env_hook() {
  if [ -f ".env.vault" ]; then
    eval "$(sx env export --shell zsh 2>/dev/null)"
    export _SHELLIX_ENV_ACTIVE=1
  elif [ -n "$_SHELLIX_ENV_ACTIVE" ]; then
    eval "$(sx env export --unset --shell zsh 2>/dev/null)"
    unset _SHELLIX_ENV_ACTIVE
  fi
}
autoload -U add-zsh-hook 2>/dev/null && add-zsh-hook chpwd _shellix_env_hook
_shellix_env_hook
`, nil

	case "bash":
		return `# Shellix Environment Auto-Hook for Bash
# Add to your ~/.bashrc: eval "$(sx env hook bash)"

_shellix_env_hook() {
  if [ -f ".env.vault" ]; then
    eval "$(sx env export --shell bash 2>/dev/null)"
    export _SHELLIX_ENV_ACTIVE=1
  elif [ -n "$_SHELLIX_ENV_ACTIVE" ]; then
    eval "$(sx env export --unset --shell bash 2>/dev/null)"
    unset _SHELLIX_ENV_ACTIVE
  fi
}
PROMPT_COMMAND="_shellix_env_hook; $PROMPT_COMMAND"
`, nil

	default:
		return "", fmt.Errorf("unsupported shell %q (supported: powershell, zsh, bash)", shell)
	}
}
