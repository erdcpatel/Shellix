# GitHub Integration via HTTPS & Personal Access Token

Shellix includes native GitHub client capabilities without requiring the external `gh` CLI. All communication is strictly encrypted over **HTTPS** (`https://api.github.com`) using a **Personal Access Token (PAT)**.

---

## 1. Security Architecture

- **HTTPS-Only**: Zero plaintext HTTP calls. All endpoints target `https://api.github.com` with TLS 1.3.
- **Zero Local Server Listeners**: No OAuth browser popups and no listening HTTP server ports.
- **Native OS Vault Integration**: Your token is stored securely in **Windows Credential Manager** on Windows and **macOS Keychain** on macOS.
- **Secret Redaction**: The token is automatically registered with Shellix's in-memory `SecretSanitizer` so it is never printed to logs or terminal output.

---

## 2. Authentication Setup

```bash
# Store your GitHub PAT in the native OS Vault
sx github auth ghp_yourPersonalAccessTokenHere

# Or enter interactively with hidden password input:
sx github auth

# Verify your credentials and view account scopes over HTTPS
sx github whoami
```

Shellix automatically discovers your token from:
1. Native OS Keyring / Windows Credential Manager (`sx vault`)
2. `GITHUB_TOKEN` environment variable
3. `GH_TOKEN` environment variable

---

## 3. Pull Request Management (`sx pr`)

Shellix automatically detects the `owner` and `repo` from your local Git remote origin (`git remote get-url origin`).

```bash
# List open pull requests
sx pr list
# or alias
sx pr ls

# Filter by state (open, closed, all)
sx pr list --state all --limit 15

# View PR description and details
sx pr view 42

# Create a new Pull Request
sx pr create --title "feat: multi-jdk manager" --body "Implements Java 8-25 switching"

# Create a draft Pull Request
sx pr create --title "WIP: experimental flow triggers" --draft
```

---

## 4. GitHub Actions CI Checks (`sx ci`)

Monitor CI build and test workflow runs for your current Git branch:

```bash
# Check CI runs for current branch
sx ci

# Check runs for a specific branch
sx ci --branch main --limit 5
```

---

## 5. Issue Management (`sx issue`)

Inspect, create, and close issues directly from your terminal:

```bash
# List repository issues
sx issue list
sx issue list --state closed --limit 10

# View detailed issue description and comment count
sx issue view 42

# Open a new issue
sx issue create --title "Investigate high disk I/O" --body "Walk operations should skip .git and vendor"

# Close an issue
sx issue close 42
```
