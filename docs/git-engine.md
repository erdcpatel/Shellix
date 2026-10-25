# Dual-Engine Git & Interactive Dashboard

Shellix includes a high-performance **dual-engine Git layer** and a **full-screen interactive Git TUI dashboard** designed to completely replace Git Bash and heavy GUI clients like lazygit.

---

## 1. Dual-Engine Dispatch Architecture

```
                      User runs: sx g [args...]
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
      (no args)           ("status" / "s")        (other args)
          │                     │                     │
          ▼                     ▼                     ▼
┌───────────────────┐ ┌───────────────────┐ ┌───────────────────────┐
│ Interactive TUI   │ │ In-Process go-git │ │ Native git execution  │
│ Full-screen Bubble│ │ Sub-2ms status    │ │ Transparent fallback  │
│ Tea Git Dashboard │ │ report            │ │ for rebase, pull, etc.│
└───────────────────┘ └───────────────────┘ └───────────────────────┘
```

---

## 2. Interactive Git Dashboard (`sx g`)

Running `sx g` (or `sx git`) without arguments launches an interactive keyboard-driven dashboard:

![Interactive Git Dashboard](images/11_shellix_git_tui.png)

### Layout:
- **Header Bar**: Active branch, latest commit hash, and repository status.
- **Left Panel (Changes)**: Displays staged files (`[S]` in green), unstaged files (`[M]` in red), and untracked files (`[?]` in blue).
- **Right Panel (Live Diff Viewer)**: Live syntax-highlighted diff for the selected file (`+` lines in green, `-` lines in red, `@@` hunk headers in cyan).
- **Status & Footer Bar**: Real-time feedback messages and commit prompt.

### Keyboard Shortcuts:

| Key | Action |
|---|---|
| `↑` / `k` | Move cursor up in file list |
| `↓` / `j` | Move cursor down in file list |
| `s` | **Stage** the currently selected file |
| `u` | **Unstage** the currently selected file |
| `a` | **Stage all** changes (`git add -A`) |
| `c` | Open inline **commit** input bar |
| `Enter` (in commit) | Commit staged changes with message |
| `Esc` (in commit) | Cancel commit |
| `p` | **Push** to remote (`git push`) |
| `P` | **Pull** from remote (`git pull`) |
| `r` | **Refresh** status from disk |
| `q` / `Ctrl+C` | Quit dashboard |

---

## 3. Instant CLI Status (`sx g s`)

When you simply want a quick text report without entering the interactive dashboard, use:
```bash
sx g s
# or
sx git status
```
This queries the repository in-memory via `go-git` in **~1.5ms to 2.5ms** with zero fork or MSYS2 emulation overhead.
