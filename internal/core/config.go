package core

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the primary Shellix configuration
type Config struct {
	Shell struct {
		Default  string `mapstructure:"default"`
		Fallback string `mapstructure:"fallback"`
	} `mapstructure:"shell"`

	Terminal struct {
		Scrollback  int    `mapstructure:"scrollback"`
		CursorStyle string `mapstructure:"cursor_style"`
		CursorBlink bool   `mapstructure:"cursor_blink"`
	} `mapstructure:"terminal"`

	Git struct {
		UseInProcessEngine bool `mapstructure:"use_in_process_engine"`
		AutoFetch          bool `mapstructure:"auto_fetch"`
	} `mapstructure:"git"`
}

// DefaultConfig returns safe sensible defaults
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Shell.Default = "auto"
	cfg.Shell.Fallback = "bash"
	cfg.Terminal.Scrollback = 10000
	cfg.Terminal.CursorStyle = "block"
	cfg.Terminal.CursorBlink = true
	cfg.Git.UseInProcessEngine = true
	cfg.Git.AutoFetch = false
	return cfg
}

// ConfigDir returns ~/.shellix
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".shellix")
	return dir, nil
}

// InitConfig sets up viper configuration paths and defaults
func InitConfig() (*Config, error) {
	dir, err := ConfigDir()
	if err == nil {
		_ = os.MkdirAll(dir, 0755)
		viper.AddConfigPath(dir)
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("shell.default", "auto")
	viper.SetDefault("shell.fallback", "bash")
	viper.SetDefault("terminal.scrollback", 10000)
	viper.SetDefault("terminal.cursor_style", "block")
	viper.SetDefault("git.use_in_process_engine", true)

	if err := viper.ReadInConfig(); err != nil {
		// If config file doesn't exist, use defaults
		return DefaultConfig(), nil
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return DefaultConfig(), nil
	}
	return &cfg, nil
}
