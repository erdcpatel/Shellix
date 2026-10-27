package cli

import (
	"fmt"
	"os"

	"shellix/internal/core"
	"shellix/internal/git"
	"shellix/internal/history"
	"shellix/internal/session"
	"shellix/internal/shortcuts"
	"shellix/internal/ssh"
)

// AppContext centralizes shared services and state across all CLI subcommands
type AppContext struct {
	Config       *core.Config
	GitEngine    *git.Engine
	ShortcutMgr  *shortcuts.Manager
	SSHMgr       *ssh.ProfileManager
	SessionMgr   *session.Manager
	HistoryStore *history.Store
	ShellRunner  func(string) error
}

// NewAppContext constructs the core application context
func NewAppContext() *AppContext {
	cfg, err := core.InitConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = core.DefaultConfig()
	}

	gitEngine := git.NewEngine(cfg.Git.UseInProcessEngine)
	shortcutMgr, _ := shortcuts.NewManager("")
	sshMgr, _ := ssh.NewProfileManager("")
	sessionMgr := session.NewManager()
	historyStore, _ := history.NewStore("")

	return &AppContext{
		Config:       cfg,
		GitEngine:    gitEngine,
		ShortcutMgr:  shortcutMgr,
		SSHMgr:       sshMgr,
		SessionMgr:   sessionMgr,
		HistoryStore: historyStore,
	}
}

// Execute is the main entry point called by cmd/shellix/main.go
func Execute() error {
	app := NewAppContext()
	rootCmd := NewRootCmd(app)
	return rootCmd.Execute()
}
