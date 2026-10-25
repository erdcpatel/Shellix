package flow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// WatcherConfig parameters for automated flow re-execution
type WatcherConfig struct {
	Flow        *Flow
	Context     *Context
	WatchDir    string
	Extensions  []string
	DebounceDur time.Duration
	Silent      bool
}

// DefaultWatchExtensions monitored by the flow watcher
var DefaultWatchExtensions = []string{
	".go", ".yaml", ".yml", ".java", ".ts", ".js", ".py", ".rs", ".json", ".md",
}

// RunWatcher monitors the workspace for file changes and triggers flow re-execution
func RunWatcher(ctx context.Context, cfg WatcherConfig) error {
	if cfg.WatchDir == "" {
		cfg.WatchDir = "."
	}
	if len(cfg.Extensions) == 0 {
		cfg.Extensions = DefaultWatchExtensions
	}
	if cfg.DebounceDur <= 0 {
		cfg.DebounceDur = 300 * time.Millisecond
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	watchStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))

	fmt.Printf("%s Monitoring %s for changes...\n",
		watchStyle.Render("👀 [FLOW WATCHER]"), infoStyle.Render(cfg.WatchDir))
	fmt.Printf("   Target Flow: %s (%d steps)\n", titleStyle.Render(cfg.Flow.Name), len(cfg.Flow.Steps))
	fmt.Printf("   Watched Exts: %s\n", metaStyle.Render(strings.Join(cfg.Extensions, ", ")))
	fmt.Println(metaStyle.Render("   Press Ctrl+C to exit watcher mode.\n"))

	// Initial execution
	runIteration(cfg)

	snapshot := takeDirSnapshot(cfg.WatchDir, cfg.Extensions)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nWatcher stopped.")
			return nil
		case <-ticker.C:
			current := takeDirSnapshot(cfg.WatchDir, cfg.Extensions)
			changedFile := diffSnapshots(snapshot, current)
			if changedFile != "" {
				// Debounce
				time.Sleep(cfg.DebounceDur)
				snapshot = takeDirSnapshot(cfg.WatchDir, cfg.Extensions)

				timestamp := time.Now().Format("15:04:05")
				fmt.Printf("\n%s File modified: %s at %s\n",
					watchStyle.Render("🔄 [RE-TRIGGER]"), infoStyle.Render(changedFile), metaStyle.Render(timestamp))

				runIteration(cfg)
			}
		}
	}
}

func runIteration(cfg WatcherConfig) {
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3F58")).Render(strings.Repeat("─", 65)))
	executor := NewExecutor(cfg.Flow, cfg.Context)
	executor.Silent = cfg.Silent
	_, _ = executor.Execute()
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3F58")).Render(strings.Repeat("─", 65)))
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("👀 Watching for new file changes..."))
}

// takeDirSnapshot walks the directory and records modification times
func takeDirSnapshot(root string, validExts []string) map[string]time.Time {
	snap := make(map[string]time.Time)

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if isWatchedExt(ext, validExts) {
			snap[path] = info.ModTime()
		}
		return nil
	})

	return snap
}

func isWatchedExt(ext string, validExts []string) bool {
	for _, v := range validExts {
		if ext == v || ext == "."+strings.TrimPrefix(v, ".") {
			return true
		}
	}
	return false
}

// diffSnapshots returns the first file path found with changed modTime, added, or deleted
func diffSnapshots(oldSnap, newSnap map[string]time.Time) string {
	for path, newTime := range newSnap {
		oldTime, exists := oldSnap[path]
		if !exists || !newTime.Equal(oldTime) {
			return path
		}
	}
	for path := range oldSnap {
		if _, exists := newSnap[path]; !exists {
			return path + " (deleted)"
		}
	}
	return ""
}
