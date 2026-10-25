# Shellix Security & Privacy Architecture

Security, privacy, and local data isolation are foundational pillars in Shellix. This document outlines the guarantees, design principles, and automated safeguards built into the codebase.

---

## 1. Zero Telemetry & Air-Gapped Operation

- **No Analytics**: Shellix does not include Google Analytics, Sentry, Mixpanel, or any tracking SDK.
- **Zero Phone-Home**: Shellix never transmits telemetry, usage statistics, repository names, branch names, file paths, or commands to external servers.
- **Air-Gapped Friendly**: Functions 100% offline. Network connections are initiated strictly when the user invokes commands targeted at remote servers (e.g., `sx ssh`, `sx g push`, or network commands in user-defined flows).

---

## 2. Automated Secret & Credential Redaction

Developers frequently pass tokens, passwords, and API keys through environment variables and build pipelines. Shellix features an automated in-memory secret scrubber (`internal/core/security.go`):

1. **Auto-Detection**: On startup, Shellix scans system environment variables for keys matching sensitive keywords (`TOKEN`, `SECRET`, `PASSWORD`, `API_KEY`, `AUTH`, `CREDENTIAL`).
2. **In-Memory Vault**: Secret values are kept in memory solely for pattern matching and are never written to disk unencrypted.
3. **Log Sanitization**: When flows execute, all raw stdout, stderr, and command strings pass through `GlobalSanitizer.Sanitize()` before being written to `~/.shellix/logs/flows/`.
4. **Redaction Placeholder**: Sensitive strings are replaced with `[REDACTED_SECRET]`.

---

## 3. Strict Local File Permissions

Shellix enforces operating-system level access controls:
- **Configuration & Storage Directories**: `~/.shellix/` and `~/.shellix/logs/` are created with `0700` (readable and writable only by the current user).
- **Credential & Host Files**: Sensitive files like `ssh_hosts.json` and credentials are saved with `0600` (no read/write permissions for other users or groups).

---

## 4. Modern Dependency & Supply Chain Hygiene

Shellix maintains a strict dependency policy:
- **Zero CGO**: Completely eliminates buffer overflow and memory corruption vulnerabilities from unmanaged C libraries.
- **Actively Maintained Libraries**: All dependencies are sourced from industry-standard, actively maintained repositories (`spf13/cobra`, `charmbracelet/lipgloss`, `go-git/go-git/v5`, `golang.org/x/crypto`, `golang.org/x/term`).
- **No Abandonware**: Deprecated, unmaintained, or stale packages without recent releases are strictly prohibited from the dependency graph.
- **Reproducible Static Builds**: Compiles into a single self-contained binary with symbols stripped via `-ldflags="-s -w"`.
