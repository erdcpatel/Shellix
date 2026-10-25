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

### 1. Git Inspection & Operations

| Command | Purpose | Expected Output / Behavior |
|---|---|---|
| `sx g s` or `sx git status` | Sub-2ms repository status | Current branch, latest commit hash/author, staged, unstaged, untracked files |
| `sx g <any-git-arg>` | Transparent native Git passthrough | Runs system Git directly with native speed |

### 2. Automation Flows with Parallel Steps

Shellix Flows are declarative YAML files that replace complex shell scripts.

```bash
# Validate a flow before running
sx flow validate path/to/flow.yaml

# Execute a flow
sx flow run path/to/flow.yaml

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

### 3. Java Toolchain & Builds

| Command | Purpose |
|---|---|
| `sx j env` or `sx java env` | Detects JDK version, compiler, JAVA_HOME, and build system |
| `sx j build [args...]` | Automatically invokes Maven/Gradle or wrapper |
| `sx j run <file.java>` | Executes a single Java source file directly |

### 4. SSH Server Management & Multi-Exec

```bash
# Configure host profile
sx ssh add prod-web --host 192.168.1.10 --user deploy

# Execute a command in parallel across multiple Linux nodes
sx ssh exec --hosts prod-web,prod-db "uptime && df -h"

# Launch interactive remote session
sx ssh connect prod-web
```

### 3. Multi-Repo Search

```bash
# Search across registered repos
sx search "type Engine struct"

# Search specific repo or path
sx search "HandleAuth" --path /path/to/repo
```

### 4. Bookmarks and Snippets

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
