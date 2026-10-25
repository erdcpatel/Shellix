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
```

---

## 📚 Documentation Index

- [Architecture Overview](docs/architecture.md)
- [Dual-Engine Git Specification](docs/git-engine.md)
- [Shellix Flows Automation Guide](docs/flows.md)
- [Multi-Repo Search Engine](docs/search.md)
- [Developer Shortcuts (Bookmarks & Snippets)](docs/shortcuts.md)
- [AI Agent Skill & Guidelines](skills/shellix/SKILL.md)

---

## 🛡️ License

Apache 2.0 / MIT Dual License.
