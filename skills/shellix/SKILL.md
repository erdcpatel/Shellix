---
name: shellix
description: Work with Shellix (sx) — a high-performance developer terminal workbench for instant Git queries, YAML automation flows, cross-repo searches, and developer shortcuts.
---

# Shellix Agent Skill

This skill guides AI agents on how to leverage the **Shellix (`sx`)** toolchain for developer operations, CI/CD automation, and fast repository inspection.

## When to Use Shellix (`sx`)

Use `sx` whenever you need to:
1. **Query Git state rapidly without process fork lag**: `sx g s` or `sx git status` provides in-process repository inspection in ~2ms.
2. **Execute structured multistep pipelines**: Use `sx flow run <flow.yaml>` instead of writing raw nested bash scripts.
3. **Search across multiple repositories**: Use `sx search <query>` to traverse multiple repos in parallel.
4. **Trigger developer shortcuts**: Access bookmarks and snippets via `sx bm` and `sx snip`.

---

## Command Reference for Agents

### 1. Git Operations, Branch Switcher & Dashboard

| Command | Purpose | Expected Output / Behavior |
|---|---|---|
| `sx g` or `sx git` | Interactive full-screen Git Dashboard | Keyboard-driven TUI to stage (`s`), unstage (`u`), commit (`c`), view live diffs, push (`p`), pull (`P`) |
| `sx b` or `sx branch` | Interactive Git Branch Switcher | Fuzzy searchable branch selector with ahead/behind status and instant checkout |
| `sx stash` | Git Stash Management | Inspect, push, and pop saved stashes (`sx stash list`, `push -m "..."`, `pop`) |
| `sx g s` or `sx git status` | Sub-2ms repository status | Current branch, latest commit hash/author, staged, unstaged, untracked files |
| `sx g <any-git-arg>` | Transparent native Git passthrough | Runs system Git directly with native speed |
| `sx web` | Zero-Latency Embedded Web Workbench | Launches embedded Linear-dark browser console with side-by-side diffs, data grid, and flow waves |

### 2. Zero-Latency Embedded Web Workbench

Instant, zero-dependency browser workbench embedded directly inside the single Go binary:

```bash
# Launch web workbench and automatically open default browser
sx web

# Specify custom port and host
sx web --port 7890 --host 127.0.0.1

# Headless mode without auto-opening browser (ideal for remotes / background)
sx web --no-browser
```

Key Capabilities:
- **Pre-Flight Code Review**: Side-by-side split diffs, changed file filtering, and change counters.
- **High-Performance Data Grid**: In-memory virtual grid with live client-side caching, search filtering, and column sorting.
- **Automation Flow DAG**: Visual execution waves and step inspector.
- **Pure-Local Search Explorer**: Multi-folder code search with directory rollups and line jumping.
- **Command Palette**: `Cmd+K` / `Ctrl+K` for instant keyboard navigation.

### 3. Automation Flows with Parallel Steps, Watcher & Interactive TUI

Shellix Flows are declarative YAML files that replace complex shell scripts.

```bash
# Validate a flow before running
sx flow validate path/to/flow.yaml

# Execute a flow sequentially or in parallel batches
sx flow run path/to/flow.yaml

# Execute with full-screen interactive Bubble Tea dashboard
sx flow run path/to/flow.yaml --tui
# Or shortcut
sx flow tui path/to/flow.yaml

# Automatically re-execute flow on file changes (Watcher Mode)
sx flow watch path/to/flow.yaml
sx flow run path/to/flow.yaml --watch

# Interactively record shell commands and synthesize a new flow YAML
sx flow record my-pipeline
sx flow record my-pipeline --global

# Replay a recorded workflow with timing verification
sx flow replay my-pipeline

# Inspect execution audit logs
sx flow logs
sx flow logs <filename.log>

# Dry-run execution blueprint, parameter overrides, and waves
sx flow plan <flow-name>
sx flow plan <flow-name> --set env=prod --set version=2.0.0
sx flow plan <flow-name> --mermaid

# Inspect preserved workflow build artifacts
sx flow artifacts
sx flow artifacts <flow-name>
```

**Advanced Workflow Schema Specification**:
```yaml
name: enterprise-release-pipeline
description: Production release automation pipeline with parameterization and gates
params:
  env: staging
  version: 1.0.0

notify:
  on_success: true   # Windows BalloonTip/Toast & macOS notifications
  on_failure: true
  webhook: "https://events.internal.corp/deployments"

steps:
  - name: lint-and-test
    run: go test -short ./...
    timeout: 30s
    retries: 2
    backoff: exponential

  - name: compile
    run: go build -o dist/app cmd/main.go
    depends_on: [lint-and-test]
    artifacts:
      - "dist/*"

  - name: gate
    run: echo "Proceeding with deployment to {{params.env}}"
    depends_on: [compile]
    prompt: "Authorize release of v{{params.version}} to environment '{{params.env}}'?"

  - name: deploy
    run: echo "Deployed v{{params.version}} to {{params.env}}"
    depends_on: [gate]
```

### 4. Native OS Credential Manager (Vault)

Uses Windows Credential Manager (`wincred`) on Windows and macOS Keychain on macOS.

```bash
# Check provider info
sx vault info

# Store a secret
sx vault set api_token "secret_val"

# Retrieve a secret
sx vault get api_token

# List stored keys
sx vault list

# Delete a secret
sx vault rm api_token
```

### 5. SSH Server Management & Multi-Exec

```bash
# Configure host profile
sx ssh add prod-web --host 192.168.1.10 --user deploy

# Execute a command in parallel across multiple Linux nodes
sx ssh exec --hosts prod-web,prod-db "uptime && df -h"

# Launch interactive remote session
sx ssh connect prod-web

# Secure local port forwarding tunnel (LocalPort:RemoteHost:RemotePort)
sx ssh tunnel prod-db 5432:localhost:5432
sx ssh tunnel prod-web 8080:80
```

### 6. System Diagnostics & Rapid Onboarding

```bash
# Run comprehensive health diagnostics on Git, Keyring, and SSH
sx doctor

# Initialize Shellix environment or generate shell integration
sx init
sx init --workspace   # Scaffold .shellix/flows/ in current repo
eval "$(sx init zsh)" # Instant shell alias setup
```

### 6. GitHub & GitHub Enterprise Client (HTTPS + PAT)

Manage Pull Requests, Issues, and CI checks without leaving the terminal (fully compatible with public `github.com`, GitHub Enterprise Cloud `*.ghe.com`, and on-premise GitHub Enterprise Server):

```bash
# Launch interactive GitHub TUI Workbench (PRs, Issues, CI Runs)
sx gh
sx github

# Save PAT to OS Keyring (defaults to github.com)
sx github auth <token>

# Save PAT for GitHub Enterprise host
sx github auth <token> --host github.mycompany.internal
sx github whoami --host github.mycompany.internal
sx github logout --host github.mycompany.internal

# Pull requests (host is auto-detected from repo remote!)
sx pr list
sx pr view <number>
sx pr create --title "..." --body "..."

# Issues
sx issue list
sx issue view <number>
sx issue create --title "..." --body "..."
sx issue close <number>

# CI status
sx ci
```

### 7. High-Throughput Folder & Code Search

Pure-local multi-threaded search across hundreds of checked-out folders without network or Git dependencies:

```bash
# Basic search across current directory
sx search "HandleAuth"

# Search specific path(s)
sx search "HandleAuth" --path /path/to/folder -p /another/folder

# Directory rollup summary with folder table
sx search "HandleAuth" --by-dir

# Include surrounding context lines (-C before/after, -A after, -B before)
sx search "HandleAuth" -C 2

# Filter by file extension and exclude unwanted files
sx search "HandleAuth" --ext go,java,json --exclude "*.min.js"

# Case-sensitive and whole-word matching
sx search "HandleAuth" -s -w

# Listing modes for shell piping and scripts
sx search "HandleAuth" --dirs-only       # List only folder paths with hits
sx search "HandleAuth" --files-only      # List only unique matching file paths (-l)
sx search "HandleAuth" --count           # Show match counts per folder (-c)

# Generate export audit reports to disk (.md, .json, or .csv)
sx search "HandleAuth" --by-dir --report audit_report.md
sx search "HandleAuth" --report findings.json
sx search "HandleAuth" --report findings.csv
```

### 8. Command History & Execution Analytics

```bash
# View recent executed commands (with exit status, duration, and deduplication)
sx history
sx history list -n 25

# Search command history by keyword or directory
sx history search "docker"

# Display execution analytics dashboard (success rate, latency, top commands)
sx history stats

# Clear history store
sx history clear -y
```

### 9. Universal Command Palette & Launcher

```bash
# Launch interactive fuzzy launcher across all flows, SSH, bookmarks & history
sx palette
sx run
sx p
```

### 10. Bookmarks and Snippets

```bash
# List all bookmarks
sx bm list

# Resolve a bookmark path
sx bm get <name>

# Run a saved snippet
sx snip run <name>
```

### 11. HTTP & API Terminal Client & Endpoint Diffing

```bash
# Execute HTTP request with microsecond network latency breakdown (DNS, TCP, TLS, TTFB)
sx http get https://api.github.com/zen
sx http post https://httpbin.org/post -d '{"event":"deploy"}' -H "Content-Type: application/json"
sx http get https://httpbin.org/bearer -b "my_secret_token"

# Automated API Health Assertions (exits with 0 on success, 1 on failure for CI/CD)
sx http get https://api.prod.corp/health --assert-status 200
sx http get https://api.prod.corp/health --assert-json ".status=healthy" --assert-max-latency 500ms

# Deep structural diff between two API endpoints (staging vs production)
sx http diff https://api.staging.example.com/v1/health https://api.prod.example.com/v1/health
```

### 12. Project Environment & Secret Injection (`.env.vault`)

```bash
# AES-256-GCM encrypted local environment store
# Master key derived automatically from native OS Credential Vault (WinCred / Keychain)
sx env set API_KEY="sk_live_123" DATABASE_URL="postgres://user:pass@db:5432/app"

# Scoped Multi-Environment support (default, dev, staging, prod)
sx env set --env staging STAGING_DB="postgres://staging:5432/app"
sx env list --env staging
sx env run --env staging -- ./deploy.sh

# Rotate 256-bit AES master key in native OS Keyring & re-encrypt vault
sx env rotate-key

# direnv-style directory auto-load hook (Windows PowerShell & macOS Zsh)
Invoke-Expression (& sx env hook powershell)  # PowerShell ($PROFILE)
eval "$(sx env hook zsh)"                     # Zsh (~/.zshrc)
```

### 13. Flow Cron & Recurring Scheduler

```bash
# Schedule flows using standard 5-part cron syntax or macros (@hourly, @daily, @weekly)
sx flow schedule add build-and-test "*/15 * * * *"
sx flow schedule add nightly-backup "@daily"

# List all scheduled jobs with next trigger times and run counts
sx flow schedule list

# Run the background scheduler daemon
sx flow schedule start
sx flow schedule logs -f          # Stream live daemon logs
sx flow schedule remove <job-id-or-flow>

# Register scheduler as native background OS service (runs automatically on logon)
sx flow schedule daemon install   # Windows Task Scheduler on Windows, LaunchAgent on macOS
sx flow schedule daemon status
sx flow schedule daemon uninstall
```

### 14. Background Process & Service Monitor (`sx ps`)

```bash
# Monitor active Shellix daemons, PIDs, uptimes, heartbeats, and OS service hooks
sx ps
```

### 14. Streaming Large File & JSON Explorer

```bash
# Interactive collapsible JSON Tree Explorer (Bubble Tea TUI)
sx json tui config.json
curl -s https://api.example.com/cluster | sx json tui

# High-throughput streaming log viewer (chunked reverse tail seek for gigabyte files)
sx view /var/log/app.log -n 100 --tail
sx view /var/log/app.log -f --search "ERROR"

# JSONPath querying (.data.items[0], .users[*].id) from files or pipes
sx json query ".data.items[0]" payload.json
curl -s https://api.example.com/status | sx json query ".status"

# Deep structural schema comparison between two JSON documents
sx json diff v1_schema.json v2_schema.json
sx json format data.json
```

### 15. Dynamic Shell Auto-Completion

```powershell
# Windows PowerShell (First-Class Support):
# Add to $PROFILE for persistent tab completion across flows, keys, branches & hosts:
if (Get-Command sx -ErrorAction SilentlyContinue) {
    sx completion powershell | Out-String | Invoke-Expression
}
```

```bash
# macOS Zsh:
source <(sx completion zsh)
```
```

---

## Agent Best Practices

- Prefer `sx g s` over raw `git status` when polling repository cleanliness during automated code edits.
- When creating automated workflows for users, generate a YAML flow under `~/.shellix/flows/<name>.yaml` and run it via `sx flow run`.
- Check exit codes: `sx` returns `0` on success and non-zero with structured error messages to `stderr`.
