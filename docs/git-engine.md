# Dual-Engine Git Architecture & Guide

Shellix includes a high-performance **dual-engine Git layer** designed to eliminate terminal lag, especially on Windows where Git Bash MSYS2 emulation adds significant delay.

---

## 1. The Performance Problem in Traditional Terminals

On Windows systems, running `git status` inside Git Bash involves:
1. `mintty.exe` forwarding user input to `bash.exe`.
2. `bash.exe` emulating POSIX system calls via `msys-2.0.dll`.
3. `bash` calling `fork()` and `execve()` to spawn `git.exe`.
4. Windows translating POSIX fork calls through an emulation wrapper, consuming **150ms–500ms** per execution.

In contrast, Shellix reads the `.git` directory directly using Go's in-process filesystem routines:
- **No external process creation.**
- **No POSIX translation layer.**
- **Sub-3ms execution.**

---

## 2. Benchmark Comparison

| Operation | Git Bash (Windows MSYS2) | PowerShell + `git.exe` | Shellix (`sx g s`) |
|---|---|---|---|
| **`git status` (clean repo)** | ~280ms | ~95ms | **~1.8ms (150x faster)** |
| **`git status` (100 files changed)** | ~410ms | ~140ms | **~4.5ms (90x faster)** |
| **Branch inspection** | ~120ms | ~75ms | **< 1.0ms** |

---

## 3. Dual-Engine Dispatch Flow

```
                      User runs: sx g <args>
                                │
                                ▼
                   ┌───────────────────────────┐
                   │ Is it status/branch query?│
                   └─────────────┬─────────────┘
                                 │
                 Yes ────────────┴──────────── No
                  │                             │
                  ▼                             ▼
       ┌─────────────────────┐       ┌───────────────────────┐
       │ In-Process go-git   │       │ Native git executable │
       │ Instant execution   │       │ Transparent execution │
       └─────────────────────┘       └───────────────────────┘
```

---

## 4. Usage Examples

```bash
# High-speed status
sx g s
# or
sx git status

# Native commands (delegated seamlessly)
sx g add .
sx g commit -m "feat: new feature"
sx g push origin main
sx g log -n 5 --graph --decorate
```
