# Shellix v0.1.0 — Initial Public Release

> **Shellix (`sx`) — Blazing-Fast Developer Terminal Workbench**  
> *Single static binary. Zero CGO. Sub-2ms in-process Git. Declarative YAML workflows with DAG parallelism. GitHub Enterprise ready. Windows first-class.*

---

## ⚡ What is Shellix?

Shellix replaces slow POSIX emulation layers (like Git Bash on Windows) with an all-in-one developer console built entirely in Go. It packages an instant Git engine, interactive workflow orchestrator, multi-repo code search, SSH connection manager, and native OS credential management into a single static executable under 12 MB.

---

## 🌟 Highlights & Superpowers in v0.1.0

### 1. Dual-Engine Git Architecture
- **In-Process Git Engine (`sx g s`)**: Powered by pure Go `go-git`, inspecting repository status, branch tracking, and commits in **< 2ms** with zero fork latency.
- **Interactive Staging & Commit Dashboard (`sx g`)**: Full-screen Bubble Tea TUI to stage (`s`), unstage (`u`), commit (`c`), and view live syntax-highlighted diffs.
- **Interactive Branch Switcher (`sx branch`)**: Fuzzy branch search with ahead/behind commit tracking.
- **Git Stash Manager (`sx stash`)**: Inspect and manipulate stashes (`push`, `pop`, `list`).

### 2. Workflow Automation Engine (Shellix Flows)
- **Declarative YAML Pipelines (`sx flow run <flow>`)**: Multi-step workflows with DAG dependencies (`depends_on`), conditional runs (`if`), and step output interpolation (`{{steps.<name>.output}}`).
- **Parallel Execution Waves**: Concurrently executes adjacent parallel steps (`parallel: true`) partitioned into dependency waves.
- **Interactive Flow TUI (`sx flow run --tui`)**: Split-pane Bubble Tea dashboard with live log streaming and step progress badges.
- **Automated Flow File Watcher (`sx flow watch <flow>`)**: Continuous TDD loop that auto-triggers workflows on file modification.

### 3. GitHub & GitHub Enterprise Integration (`sx gh` / `sx pr` / `sx issue` / `sx ci`)
- **Full Support for GHES & GHEC**: Auto-detects whether the repository belongs to `github.com` or on-premise `github.corp.internal` (`/api/v3`).
- **Native OS Vault Auth**: Personal Access Tokens (PAT) stored in Windows Credential Manager or macOS Keychain.
- **Interactive GitHub Workbench (`sx gh`)**: Unified multi-tab TUI for Pull Requests, Issues, and CI runs.

### 4. Command History & Execution Analytics (`sx history`)
- In-memory index with deduplication, latency metrics, and success rates (`sx history stats`).
- Pattern-based automated secret sanitization preventing credential leakage.

### 5. Multi-Repo Intelligent Search (`sx search`)
- Parallel file tree walking across registered repositories with `.gitignore` awareness.

### 6. Multi-JDK Manager (`sx java`)
- Auto-discovers all installed JDKs (Adoptium, Corretto, Temurin) and switches `JAVA_HOME`.

### 7. SSH Target Manager & Port Forwarding Tunnels (`sx ssh`)
- Host profiles, multi-node parallel commands, and encrypted local TCP port forwarding tunnels (`sx ssh tunnel`).

### 8. Universal Command Palette (`sx palette` / `sx run`)
- Fuzzy launcher connecting all tools, workflows, Java versions, SSH hosts, and bookmarks.

---

## 📦 Installation

### Pre-Compiled Binaries
Download the binary for your platform from the Releases page:
- **Windows**: `shellix_0.1.0_windows_amd64.zip` / `shellix_0.1.0_windows_arm64.zip`
- **macOS**: `shellix_0.1.0_darwin_arm64.tar.gz` (Apple Silicon) / `shellix_0.1.0_darwin_amd64.tar.gz` (Intel)
- **Linux**: `shellix_0.1.0_linux_amd64.tar.gz` / `shellix_0.1.0_linux_arm64.tar.gz`

### Scoop (Windows)
```powershell
scoop bucket add shellix https://github.com/erdcpatel/Shellix
scoop install shellix
```

### Homebrew (macOS / Linux)
```bash
brew install erdcpatel/tap/shellix
```
