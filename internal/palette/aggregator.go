package palette

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"shellix/internal/history"
	"shellix/internal/shortcuts"
	"shellix/internal/ssh"
)

// CollectAllActions aggregates available commands and items from all Shellix modules
func CollectAllActions(histStore *history.Store, scMgr *shortcuts.Manager, sshMgr *ssh.ProfileManager) []PaletteAction {
	var actions []PaletteAction

	// 1. Core Superpowers
	actions = append(actions, []PaletteAction{
		{
			ID:          "core-git",
			Title:       "Interactive Git Dashboard (sx g)",
			Description: "Stage, unstage, commit & diff changes",
			Category:    CatCore,
			Command:     "sx g",
		},
		{
			ID:          "core-search",
			Title:       "Parallel Code Search (sx search)",
			Description: "High-speed multi-repo search (.gitignore aware)",
			Category:    CatCore,
			Command:     "sx search",
		},
		{
			ID:          "core-github",
			Title:       "GitHub Workbench Dashboard (sx gh)",
			Description: "Pull requests, issues, and CI actions",
			Category:    CatCore,
			Command:     "sx gh",
		},
		{
			ID:          "core-history",
			Title:       "Execution History & Analytics (sx history)",
			Description: "View command stats, success rates & search runs",
			Category:    CatCore,
			Command:     "sx history stats",
		},
		{
			ID:          "core-doctor",
			Title:       "System Diagnostics (sx doctor)",
			Description: "Inspect Git, Keyring, and SSH hardening",
			Category:    CatCore,
			Command:     "sx doctor",
		},
		{
			ID:          "core-vault",
			Title:       "OS Credential Manager (sx vault)",
			Description: "Inspect native Windows Credential Manager / Keychain",
			Category:    CatCore,
			Command:     "sx vault info",
		},
	}...)

	// 2. Flows (Local & Global)
	flowDirs := []string{
		filepath.Join(".", ".shellix", "flows"),
		filepath.Join(".", "tests", "fixtures", "flows"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		flowDirs = append(flowDirs, filepath.Join(home, ".shellix", "flows"))
	}

	seenFlows := make(map[string]bool)
	for _, dir := range flowDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
				flowName := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".yaml"), ".yml")
				if !seenFlows[flowName] {
					seenFlows[flowName] = true
					actions = append(actions, PaletteAction{
						ID:          "flow-" + flowName,
						Title:       fmt.Sprintf("Run Flow: %s", flowName),
						Description: fmt.Sprintf("Execute pipeline with interactive TUI (%s)", e.Name()),
						Category:    CatFlow,
						Command:     fmt.Sprintf("sx flow run %s --tui", flowName),
					})
				}
			}
		}
	}

	// 3. SSH Profiles
	if sshMgr != nil {
		hosts := sshMgr.ListHosts()
		for _, p := range hosts {
			actions = append(actions, PaletteAction{
				ID:          "ssh-" + p.Name,
				Title:       fmt.Sprintf("Connect to SSH: %s", p.Name),
				Description: fmt.Sprintf("%s@%s:%d", p.User, p.Host, p.Port),
				Category:    CatSSH,
				Command:     fmt.Sprintf("sx ssh connect %s", p.Name),
			})
		}
	}

	// 5. Bookmarks & Snippets
	if scMgr != nil {
		bms := scMgr.ListBookmarks()
		for _, b := range bms {
			actions = append(actions, PaletteAction{
				ID:          "bm-" + b.Name,
				Title:       fmt.Sprintf("Bookmark: %s", b.Name),
				Description: b.Path,
				Category:    CatBookmark,
				Command:     fmt.Sprintf("sx bm get %s", b.Name),
			})
		}

		snips := scMgr.ListSnippets()
		for _, s := range snips {
			actions = append(actions, PaletteAction{
				ID:          "snip-" + s.Name,
				Title:       fmt.Sprintf("Snippet: %s", s.Name),
				Description: s.Command,
				Category:    CatSnippet,
				Command:     fmt.Sprintf("sx snip run %s", s.Name),
			})
		}
	}

	// 6. Recent History commands
	if histStore != nil {
		recent := histStore.GetRecent(5)
		for _, h := range recent {
			actions = append(actions, PaletteAction{
				ID:          "hist-" + h.ID,
				Title:       fmt.Sprintf("History: %s", h.Command),
				Description: fmt.Sprintf("Last executed %s in %s", h.Duration.Round(100*time.Millisecond), h.Directory),
				Category:    CatHistory,
				Command:     h.Command,
			})
		}
	}

	return actions
}
