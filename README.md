# 🐚 Shellix (`sx`)

> **A blazing-fast, developer-first terminal workbench built in pure Go.**  
> Single static binary. Cross-platform. Secure. Built-in automation flows, instant Git engine, and multi-repo search.

---

## ⚡ Why Shellix?

Traditional terminals and toolchains on Windows and cross-platform environments suffer from high fork latencies and fractured tooling:
- **Git Bash on Windows** relies on MSYS2 POSIX emulation layers, introducing 200–500ms overhead on process calls.
- **Context-Switching Tax**: Developers constantly juggle separate tools for terminal emulation, Git GUIs, SSH clients, search tools, and deployment scripts.

**Shellix solves this by combining these capabilities into a single native binary (under 15MB) with zero external runtime dependencies (no Electron, no Node.js, no Python).**

```
┌─────────────────────────────────────────────────────────────────┐
│                       Shellix Workbench                         │
├─────────────────┬──────────────────┬────────────────────────────┤
│ Dual-Engine Git │ Automation Flows │ Cross-Repo Search Engine   │
│ In-process Go   │ YAML-defined     │ Parallel goroutine search  │
│ (< 2ms status)  │ tasks & pipelines│ across multiple codebases  │
├─────────────────┼──────────────────┼────────────────────────────┤
│ SSH Multiplexer │ TUI Interface    │ Developer Shortcuts        │
│ Crypto/SSH with │ Elm-architecture │ Snippets, bookmarks, and   │
│ connection pool │ Bubble Tea views │ smart history              │
└─────────────────┴──────────────────┴────────────────────────────┘
```

---

## 📸 Screenshots & UI Walkthrough

### 1. Interactive Git Dashboard (`sx g`)
Full-screen, keyboard-driven dashboard to stage (`s`), unstage (`u`), commit (`c`), and view live syntax-highlighted diffs:
![Interactive Git Dashboard](docs/images/11_shellix_git_tui.png)

### 2. Instant Status & Banner
Sub-2ms Git inspection directly from disk:
![Instant Status](docs/images/01_shellix_status.png)

### 2. Flow Automation Engine
Run declarative pipelines with parallel step execution and audit logging:
![Flow Engine](docs/images/02_shellix_flow.png)

### 3. Parallel Java & Multi-Module Builds
Run Java and Go builds concurrently in a single flow:
![Parallel Builds](docs/images/07_shellix_parallel_builds.png)

### 4. Java Multi-Version Discovery & Switching
Discover all installed JDKs (Adoptium, Corretto, Temurin, Homebrew) and switch seamlessly:
![Java Version Manager](docs/images/09_shellix_java_list.png)

### 5. Native OS Credential Manager (Vault)
Zero-leak secrets backed by Windows Credential Manager and macOS Keychain:
![Vault](docs/images/10_shellix_vault_info.png)

### 6. Multi-Repo Parallel Search
Search across multiple codebases with syntax-highlighted matching lines:
![Search Engine](docs/images/03_shellix_search.png)

### 6. SSH Multi-Server Target Management
Manage server targets and execute commands in parallel across Linux nodes:
![SSH Manager](docs/images/08_shellix_ssh_manager.png)

### 7. GitHub & GitHub Enterprise Integration (HTTPS + PAT)
Manage PRs, Issues, and CI runs directly from your terminal across public GitHub and GitHub Enterprise Server:
```bash
sx github auth            # Save PAT for github.com into OS Keyring
sx github auth <token> --host github.corp.internal # Save PAT for Enterprise Server
sx github whoami          # Verify authenticated profile (auto-detects host from repo)
sx pr list                # List open pull requests
sx pr view 42             # View PR details
sx pr create --title "..." # Open new pull request
sx issue list             # List open issues
sx issue view 10          # View issue details
sx issue create --title "..." # Create new issue
sx ci                     # Check CI workflow runs for current branch
```

### 8. System Doctor & Diagnostics
Run deep environment health checks on Git, JDKs, Keyring, and SSH hardening:
![System Doctor](docs/images/12_shellix_doctor.png)

### 9. SSH Port Forwarding Tunneling
Encrypted local port forwarding over SSH without third-party tools:
![SSH Tunnel](docs/images/16_shellix_ssh_tunnel.png)

### 10. Rapid Developer Onboarding (`sx init`)
Initialize ~/.shellix storage, secure permissions, and instant shell aliases:
![Shellix Init](docs/images/15_shellix_init.png)

### 11. Developer Shortcuts
Manage directory bookmarks and reusable command snippets:
![Shortcuts](docs/images/04_shellix_shortcuts.png)

### 12. Interactive Flow TUI (`sx flow run --tui`)
Full-screen Bubble Tea dashboard with real-time spinners, live log streaming, and step progress:
![Interactive Flow TUI](docs/images/18_shellix_flow_tui.png)

### 13. Command History & Execution Analytics (`sx history`)
In-memory searchable command history with deduplication, latency metrics, and success rates:
![Command History](docs/images/19_shellix_history.png)

### 14. Interactive GitHub Workbench (`sx gh`)
Multi-tab TUI for Pull Requests, Issues, and CI checks across public and Enterprise GitHub:
![GitHub Workbench](docs/images/20_shellix_github_tui.png)

### 15. Universal Command Palette (`sx palette`)
Fuzzy command launcher connecting flows, Java runtimes, SSH profiles, bookmarks, and history:
![Command Palette](docs/images/21_shellix_palette.png)

### 16. Flow Automated File Watcher (`sx flow watch`)
Continuous test-driven development loop that auto-runs pipelines upon file modifications:
![Flow Watcher](docs/images/22_shellix_flow_watch.png)

### 17. Interactive Git Branch Switcher (`sx branch`)
Fuzzy branch selector with ahead/behind status, commit summaries, and instant checkout:
![Branch Switcher](docs/images/23_shellix_git_branch.png)

### 18. Live Workflow Recorder (`sx flow record`)
Interactively record shell commands and automatically synthesize declarative YAML workflows:
![Flow Recorder](docs/images/24_shellix_flow_record.png)

### 19. HTTP & API Terminal Client (`sx http` / `sx api`)
High-precision HTTP client with sub-millisecond network timing metrics (DNS, TCP, TLS, TTFB), automatic JSON detection, and side-by-side API endpoint diffing:
![HTTP Diff](docs/images/25_shellix_http_diff.png)

### 20. Project Environment & Secret Injection (`sx env` / `.env.vault`)
Hardware-backed AES-256-GCM encrypted local environment store with automatic key management via OS Credential Vault (Windows Credential Manager / macOS Keychain) and secret-masked child process execution:
![Env Vault](docs/images/26_shellix_env_vault.png)

### 21. Flow Cron & Recurring Scheduler (`sx flow schedule`)
Standard 5-field cron automation scheduler with macros (`@hourly`, `@daily`, `@weekly`), next trigger calculations, state tracking, and background daemon runner:
![Flow Cron](docs/images/27_shellix_flow_cron.png)

### 22. Streaming Large File & JSON Diff Engine (`sx json` / `sx view`)
High-throughput streaming viewer for multi-gigabyte logs (with chunked reverse tail seek) and JSONPath querying (`.data.items[0]`) with deep schema structural diffing:
![JSON Diff](docs/images/28_shellix_json_diff.png)

---

## 🚀 Key Features

1. **Dual-Engine Git Layer**:
   - **Fast Path (`go-git`)**: In-process status, branch tracking, and commit inspection running in **~1.5ms to 2.5ms** with zero fork or emulation delay.
   - **Full Path (Native Git)**: Seamless and transparent passthrough to the system Git binary for complex rebase, merge, and stash workflows.
2. **Shellix Flows (Automation Engine)**:
   - Declarative YAML workflows to automate builds, tests, deploys, and remote tasks.
   - Support for sequential and parallel steps, conditional execution, timeouts, retries, and variable interpolation (`{{project_root}}`, `{{git.branch}}`, `{{env.VAR}}`).
3. **Multi-Repo Intelligent Search**:
   - High-throughput parallel file tree walking and regex/literal searching across dozens of registered repositories simultaneously.
4. **Developer Productivity Shortcuts**:
   - Path bookmarking (`sx bm`) and command snippet management (`sx snip`).
5. **Interactive TUI**:
   - Powered by Charmbracelet's Bubble Tea and Lip Gloss.
6. **Agent-Friendly & Richly Documented**:
   - Native AI Agent skills and specifications inside the repository so coding agents can seamlessly interact with Shellix workflows.

---

## 📦 Installation & Quick Start

### Build from Source

Requirements: Go 1.24+

```bash
# Clone or navigate to the workspace
cd /Users/dhavalpatel/workspace/shellix

# Build the static binary
make build

# The binary is placed in bin/shellix with a symlink to bin/sx
./bin/sx version
```

### Quick Commands

```bash
# 1. Instant Status (Sub-2ms Git inspection)
sx g s

# 2. Transparent Git passthrough
sx g log -n 5 --oneline

# 3. Interactive Shell
sx shell

# 4. Automation Flows
sx flow run deploy-staging.yaml
sx flow list

# 5. Multi-Repo Search
sx search "func HandleAuth"

# 6. Bookmarks & Snippets
sx bm add myproject .
sx bm list
sx snip add test "go test -v ./..."
sx snip run test

# 7. HTTP & API Diffing
sx http get https://api.github.com/zen
sx http diff https://api.staging.corp/health https://api.prod.corp/health

# 8. Project Environment Vault & Secret Injection
sx env set API_KEY=sk_live_9381 DATABASE_URL=postgres://...
sx env list
sx env run -- ./deploy.sh

# 9. Flow Cron Scheduler
sx flow schedule add build-and-test "*/15 * * * *"
sx flow schedule list
sx flow schedule start

# 10. Streaming File & JSON Viewer
sx json query ".users[0].name" data.json
sx json diff v1_service.json v2_service.json
sx view large_service.log --tail -n 100
```

---

## 📚 Documentation Index

- [Architecture Overview](docs/architecture.md)
- [Security & Privacy Architecture](docs/security.md)
- [Dual-Engine Git & Interactive TUI](docs/git-engine.md)
- [GitHub Integration (HTTPS + PAT)](docs/github.md)
- [Shellix Flows Automation & Parallel Builds](docs/flows.md)
- [Java Toolchain & Multi-Version Manager](docs/java.md)
- [Native OS Credential Manager (Vault)](docs/vault.md)
- [SSH Multi-Server Manager](docs/ssh.md)
- [Multi-Repo Search Engine](docs/search.md)
- [Developer Shortcuts (Bookmarks & Snippets)](docs/shortcuts.md)
- [AI Agent Skill & Guidelines](skills/shellix/SKILL.md)

---

## 🛡️ License

Apache 2.0 / MIT Dual License.
