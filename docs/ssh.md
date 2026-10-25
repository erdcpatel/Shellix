# SSH Connection Manager & Parallel Execution

Shellix features a pure Go `crypto/ssh` manager that replaces manual `~/.ssh/config` juggling and supports concurrent command execution across multiple Linux servers.

---

## 1. Features

- **Named Host Profiles**: Securely store target servers with host, port, user, key path, or password.
- **PTY Interactive Shells**: Full interactive pseudo-terminal allocation with window resizing.
- **Parallel Multi-Exec**: Broadcast commands to multiple Linux nodes simultaneously with color-coded host prefixes.
- **Local Persistence**: Profiles stored in `~/.shellix/ssh_hosts.json` with restricted `0600` permissions.

---

## 2. Command Reference

```bash
# Add a remote Linux server
sx ssh add prod-web-01 --host 192.168.1.101 --user deploy
sx ssh add prod-db-01  --host 192.168.1.102 --user postgres --port 2222 --key ~/.ssh/db_rsa

# List configured targets
sx ssh list

# Launch an interactive shell on a server
sx ssh connect prod-web-01

# Execute a command in parallel across multiple Linux servers
sx ssh exec --hosts prod-web-01,prod-db-01 "uptime && df -h /"
```

---

## 3. Parallel Multi-Exec Output Example

When running against multiple hosts, lines are streamed in real time prefixed by the server alias:

```
⚡ [SSH MULTI-EXEC] Dispatching 'uptime' to 2 server(s)...

[prod-web-01]            18:15:02 up 45 days, 3 users,  load average: 0.12, 0.08, 0.05
[prod-db-01]             18:15:02 up 12 days, 1 user,   load average: 0.50, 0.42, 0.38
```
