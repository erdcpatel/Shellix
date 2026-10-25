# Multi-Repo Search Engine

The Shellix Search Engine is built for ultra-fast, multi-repository code and text search. It traverses directories in parallel using Go's worker pool concurrency model and automatically filters out non-text files and build artifacts.

---

## 1. Key Performance Characteristics

- **Parallel Worker Pool**: Dynamically scales to `2 * CPU cores` goroutines.
- **Smart Directory Filtering**: Ignores `.git`, `node_modules`, `vendor`, `bin`, `dist`, `.idea`, and other artifact folders.
- **Binary Detection**: Reads the leading 512 bytes of files to skip binary and non-text formats before allocating scanners.
- **Sub-15ms Query Times**: Typical response across codebases with thousands of files is well under 15ms.

---

## 2. CLI Usage

```bash
# Literal text search in current directory
sx search "ValidateToken"

# Regular expression search
sx search --regex "^func\s+[A-Z]"

# Search across specific directories or external repositories
sx search "HandleAuth" --path /Users/dhavalpatel/workspace/repo1 --path /Users/dhavalpatel/workspace/repo2

# Limit results
sx search "TODO" --max 25
```

---

## 3. Options Reference

| Flag | Short | Default | Description |
|---|---|---|---|
| `--regex` | `-r` | `false` | Enable regular expression parsing |
| `--path` | `-p` | `.` | Target path(s) to search (repeatable) |
| `--max` | `-m` | `500` | Maximum matching lines to return |
| `--max-size` | | `5MB` | Maximum file size in bytes to index |
