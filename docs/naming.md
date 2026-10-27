# Shellix Naming, Brand & Trademark Assessment

## 1. Nomenclature & Architecture
- **Official Product Name**: **Shellix**
- **Primary Binary Name**: `shellix`
- **Ergonomic CLI Aliases**: `sx` (also symlinked as `bin/sx`) and `slx` (symlinked as `bin/slx`)

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

## 3. Ergonomic Aliases (`sx` and `slx`) Analysis

The 2-character alias `sx` and 3-character alias `slx` allow instant developer muscle memory (similar to `gh`, `rg`, `fd`, `jq`):
- `sx g` / `slx g` -> Instant Git TUI dashboard
- `sx flow run <name>` / `slx flow run <name>` -> Execute automation workflow
- `sx palette` / `slx palette` -> Universal fuzzy command palette
- `sx search <query>` / `slx search <query>` -> Parallel Git-aware code search
- `sx doctor` / `slx doctor` -> Comprehensive environment health diagnostics

On POSIX platforms, both `sx` and `slx` are provided as companion symlinks alongside `shellix`. On Windows, binary shims and aliases are configured for both `sx.exe` and `slx.exe`. If a user already possesses a local alias for `sx`, the `slx` alias or the primary binary `shellix` operates identically without conflict.

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
