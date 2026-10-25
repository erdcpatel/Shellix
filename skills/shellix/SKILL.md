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

### 2. Automation Flows with Parallel Steps, Watcher & Interactive TUI

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

# Inspect execution audit logs
sx flow logs
sx flow logs <filename.log>
```

**Parallel Flow Schema Structure**:
```yaml
name: parallel-build
steps:
  - name: prep
    run: echo "Initializing..."
  # Adjacent parallel steps execute concurrently:
  - name: build-backend
    run: sx j build
    parallel: true
  - name: test-suite
    run: go test ./...
    parallel: true
  - name: summary
    run: echo "All done!"
```

### 3. Java Toolchain & Multi-Version Manager

| Command | Purpose |
|---|---|
| `sx j ls` or `sx java list` | Discovers and lists all installed JDKs across the machine |
| `sx j use <version>` | Resolves matching JDK and outputs activation exports |
| `eval $(sx j use <version> -e)` | Activates the chosen Java version in the current shell |
| `sx j shell <version>` | Spawns a subshell configured with that JDK |
| `sx j env` | Inspects current Java runtime, compiler, and project build file |
| `sx j build [args...]` | Builds project with Maven or Gradle wrapper automatically |
| `sx j run <file.java>` | Executes a single Java source file directly |

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

### 3. System Diagnostics & Rapid Onboarding

```bash
# Run comprehensive health diagnostics on Git, JDKs, Keyring, and SSH
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

### 7. Multi-Repo Search

```bash
# Search across registered repos
sx search "type Engine struct"

# Search specific repo or path
sx search "HandleAuth" --path /path/to/repo
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
# Launch interactive fuzzy launcher across all flows, JDKs, SSH, bookmarks & history
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

---

## Agent Best Practices

- Prefer `sx g s` over raw `git status` when polling repository cleanliness during automated code edits.
- When creating automated workflows for users, generate a YAML flow under `~/.shellix/flows/<name>.yaml` and run it via `sx flow run`.
- Check exit codes: `sx` returns `0` on success and non-zero with structured error messages to `stderr`.
