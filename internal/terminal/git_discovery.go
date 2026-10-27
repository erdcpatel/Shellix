package terminal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CandidateGitPaths returns standard candidate Git installation directories on Windows
func CandidateGitPaths() []string {
	var candidates []string

	addPath := func(dir string) {
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		for _, c := range candidates {
			if strings.EqualFold(c, clean) {
				return
			}
		}
		candidates = append(candidates, clean)
	}

	// Dynamic environment variable based paths
	if pw64 := os.Getenv("ProgramW6432"); pw64 != "" {
		addPath(filepath.Join(pw64, "Git", "cmd"))
		addPath(filepath.Join(pw64, "Git", "bin"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		addPath(filepath.Join(pf, "Git", "cmd"))
		addPath(filepath.Join(pf, "Git", "bin"))
	}
	if pfx86 := os.Getenv("ProgramFiles(x86)"); pfx86 != "" {
		addPath(filepath.Join(pfx86, "Git", "cmd"))
		addPath(filepath.Join(pfx86, "Git", "bin"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		addPath(filepath.Join(localAppData, "Programs", "Git", "cmd"))
		addPath(filepath.Join(localAppData, "Programs", "Git", "bin"))
	}
	if progData := os.Getenv("ProgramData"); progData != "" {
		addPath(filepath.Join(progData, "chocolatey", "bin"))
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		addPath(filepath.Join(userProfile, "scoop", "shims"))
		addPath(filepath.Join(userProfile, "scoop", "apps", "git", "current", "cmd"))
		addPath(filepath.Join(userProfile, "scoop", "apps", "git", "current", "bin"))
	}

	// Well-known standard installation fallbacks with dynamic drive resolution
	sysDrive := os.Getenv("SystemDrive")
	if sysDrive == "" {
		sysDrive = "C:"
	}
	fallbacks := []string{
		filepath.Join(sysDrive, `\Program Files\Git\cmd`),
		filepath.Join(sysDrive, `\Program Files\Git\bin`),
		filepath.Join(sysDrive, `\Program Files (x86)\Git\cmd`),
		filepath.Join(sysDrive, `\Program Files (x86)\Git\bin`),
		filepath.Join(sysDrive, `\Git\cmd`),
		filepath.Join(sysDrive, `\Git\bin`),
		filepath.Join(sysDrive, `\ProgramData\chocolatey\bin`),
	}
	for _, fb := range fallbacks {
		addPath(fb)
	}

	return candidates
}

// FindGitInPaths iterates over candidate paths and returns the first directory containing git.exe or git.cmd
func FindGitInPaths(candidates []string, checkFile func(string) bool) string {
	for _, dir := range candidates {
		gitExe := filepath.Join(dir, "git.exe")
		gitCmd := filepath.Join(dir, "git.cmd")
		if checkFile(gitExe) || checkFile(gitCmd) {
			return dir
		}
	}
	return ""
}

// DiscoverWindowsGit detects if Git is already present in PATH; if not, it probes candidate locations
func DiscoverWindowsGit() string {
	// If git is already reachable in PATH, no action needed
	if _, err := exec.LookPath("git.exe"); err == nil {
		return ""
	}
	if _, err := exec.LookPath("git"); err == nil {
		return ""
	}

	return FindGitInPaths(CandidateGitPaths(), func(target string) bool {
		fi, err := os.Stat(target)
		return err == nil && !fi.IsDir()
	})
}
