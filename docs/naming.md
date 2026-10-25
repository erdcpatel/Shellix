# Shellix Naming, Brand & Trademark Assessment

## 1. Nomenclature & Architecture
- **Official Product Name**: **Shellix**
- **Primary Binary Name**: `shellix`
- **Ergonomic CLI Alias**: `sx` (also symlinked as `bin/sx`)

The name **Shellix** embodies the convergence of a persistent PTY Unix terminal shell with a modern developer workbench matrix (automation flows, Git indexing, JVM switching, and native OS credential management).

---

## 2. Collision & Ecosystem Analysis

Before adopting the name, a systematic collision analysis across open-source package repositories, binary registries, and trademarks was conducted:

| Existing Project | Domain | Collision Risk | Differentiation |
|---|---|---|---|
| **ShellCheck** | Static AST analysis for bash/sh scripts | None | ShellCheck is an offline bash script linter (`koalaman/shellcheck`), not a developer workbench or terminal environment. |
| **ShellHub** | Cloud SSH gateway & device manager | None | ShellHub is an enterprise reverse-SSH bastion for IoT/fleet devices (`shellhub-io/shellhub`). |
| **ShellJS** | Node.js portable POSIX API | None | ShellJS is an npm library implementing Unix commands on top of Node.js child processes. |
| **ShellGPT** | CLI wrapper for OpenAI ChatGPT | None | ShellGPT (`shell-gpt`) is a Python command generator. |
| **ShellInABox** | Web-based terminal emulator | None | ShellInABox runs an in-browser AJAX terminal daemon. Shellix is a pure native terminal workbench with zero web/browser dependencies. |

---

## 3. Ergonomic Alias (`sx`) Analysis

The 2-character alias `sx` allows instant developer muscle memory (similar to `gh`, `rg`, `fd`, `jq`):
- `sx g` -> Instant Git TUI dashboard
- `sx flow run <name>` -> Execute automation workflow
- `sx java ls` -> Multi-JDK version discovery
- `sx search <query>` -> Parallel Git-aware code search
- `sx doctor` -> Comprehensive environment health diagnostics

On POSIX platforms, `sx` is provided as a companion symlink alongside `shellix`. If a user already possesses a local alias for `sx`, the primary binary `shellix` operates identically without conflict.

---

## 4. License Hygiene & Compliance
All external libraries consumed by Shellix strictly utilize permissive open-source licenses:
- `github.com/spf13/cobra` (Apache-2.0)
- `github.com/charmbracelet/lipgloss` (MIT)
- `github.com/charmbracelet/bubbletea` (MIT)
- `github.com/go-git/go-git/v5` (Apache-2.0)
- `github.com/zalando/go-keyring` (MIT)
- `golang.org/x/crypto` (BSD-3-Clause)
- `golang.org/x/term` (BSD-3-Clause)

Zero GPL, AGPL, or viral copyleft dependencies are incorporated into the Shellix binary distribution.
