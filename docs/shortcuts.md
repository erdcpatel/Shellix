# Developer Shortcuts: Bookmarks & Snippets

Shellix includes a built-in shortcut engine to manage your frequently visited filesystem paths and frequently typed shell commands.

---

## 1. Directory Bookmarks (`sx bm`)

Bookmarks allow you to name and jump to long or deeply nested project paths without writing manual shell aliases.

```bash
# Add a bookmark for the current working directory
sx bm add myapp .

# Add a bookmark for an arbitrary path
sx bm add infra /Users/dhavalpatel/workspace/infra

# List all saved bookmarks
sx bm list

# Get path for a bookmark (useful in scripts or cd)
sx bm get myapp

# Delete a bookmark
sx bm rm myapp
```

---

## 2. Command Snippets (`sx snip`)

Snippets save long, complex shell commands that you can execute anytime by alias.

```bash
# Save a snippet
sx snip add cleanup "docker system prune -af && docker volume prune -f" -d "Clean all docker artifacts"
sx snip add bench "go test -bench=. -benchmem ./..."

# List saved snippets
sx snip list

# Execute a snippet directly
sx snip run cleanup
sx snip run bench

# Delete a snippet
sx snip rm cleanup
```

---

## 3. Storage Location

Shortcuts are persisted locally in JSON format at:
```
~/.shellix/shortcuts.json
```
They are fully offline, versionable, and portable across machines.
