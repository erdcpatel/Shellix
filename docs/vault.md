# Native OS Credential Manager & Vault

Shellix integrates directly with operating system credential stores to securely store tokens, passwords, and private keys.

---

## 1. Supported Native Providers

| Operating System | Native Keyring Provider | Mechanism |
|---|---|---|
| **Windows** | **Windows Credential Manager** | Native Win32 `wincred.dll` (`CredWriteW` / `CredReadW`) |
| **macOS** | **macOS Keychain** | Apple Security Framework / Keychain Services |
| **Linux** | **Secret Service API** | Freedesktop DBus Secret Service |
| **Headless / Container** | **AES-256-GCM Vault** | Local encrypted vault fallback (`~/.shellix/vault.enc`) |

---

## 2. CLI Usage

```bash
# Check active credential provider
sx vault info
# or alias
sx cred info

# Store a secret (prompts securely with hidden input if value is omitted)
sx vault set github_token
# or pass directly
sx vault set prod_api_key "my-secure-key"

# Retrieve a secret
sx vault get github_token

# List all stored credential keys
sx vault list

# Delete a credential
sx vault rm github_token
```

---

## 3. Automated Secret Scrubbing Integration

Every secret stored in or retrieved from the vault is automatically registered with Shellix's in-memory `SecretSanitizer`. If an automated workflow or command outputs that secret string, it is automatically redacted as `[REDACTED_SECRET]` before ever touching log files or terminal scrollback.
