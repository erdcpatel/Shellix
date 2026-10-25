package core

import "fmt"

// Version is populated at build time via -ldflags
var Version = "0.1.0-dev"

// BuildInfo holds release and environment information
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// Banner returns the branded ASCII startup banner for Shellix
func Banner() string {
	return fmt.Sprintf(`  ___ _        _ _ _      
 / __| |_  ___| | (_)_ __ 
 \__ \ ' \/ -_) | | \ \ / 
 |___/_||_\___|_|_|_/_\_\  v%s
 High-Performance Terminal Workbench
`, Version)
}
