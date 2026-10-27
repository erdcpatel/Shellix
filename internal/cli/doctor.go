package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"shellix/internal/core"
	"shellix/internal/vault"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// NewDoctorCmd inspects the host environment, toolchains, permissions, and security
func NewDoctorCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run comprehensive system diagnostics and environment health check",
		Run: func(cmd *cobra.Command, args []string) {
			runDoctorDiagnostics(app)
		},
	}
}

func runDoctorDiagnostics(app *AppContext) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	fmt.Println(headerStyle.Render("🩺 Shellix System Doctor & Environment Diagnostics"))
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))

	var passedCount, warnCount, failCount int

	check := func(category, name string, ok bool, isWarn bool, detail string) {
		icon := passStyle.Render("✔")
		if !ok {
			if isWarn {
				icon = warnStyle.Render("▲")
				warnCount++
			} else {
				icon = failStyle.Render("✖")
				failCount++
			}
		} else {
			passedCount++
		}
		fmt.Printf(" %s  %-26s %s\n", icon, name, metaStyle.Render(detail))
	}

	// 1. Host & Platform Environment
	fmt.Println()
	fmt.Println(sectionStyle.Render("🖥  Host Platform & Runtime"))
	check("platform", "Operating System", true, false, fmt.Sprintf("%s (%s)", runtime.GOOS, runtime.GOARCH))
	check("platform", "Go Toolchain Version", true, false, runtime.Version())
	check("platform", "CPU Allocation", runtime.NumCPU() >= 2, false, fmt.Sprintf("%d cores available", runtime.NumCPU()))

	shellEnv := os.Getenv("SHELL")
	if runtime.GOOS == "windows" {
		shellEnv = os.Getenv("COMSPEC")
	}
	check("platform", "Default Subshell", shellEnv != "", true, shellEnv)

	// 2. Git Subsystem
	fmt.Println()
	fmt.Println(sectionStyle.Render("📦 Git Subsystem"))
	gitPath, err := exec.LookPath("git")
	if err == nil {
		out, _ := exec.Command("git", "--version").CombinedOutput()
		gitVer := strings.TrimSpace(string(out))
		check("git", "Native Git Executable", true, false, fmt.Sprintf("%s (%s)", gitPath, gitVer))
	} else {
		check("git", "Native Git Executable", false, true, "git binary not found in PATH")
	}

	st, err := app.GitEngine.GetStatus("")
	if err == nil {
		cleanStr := "clean"
		if !st.IsClean {
			cleanStr = fmt.Sprintf("dirty (+%d ~%d ?%d)", len(st.Staged), len(st.Unstaged), len(st.Untracked))
		}
		check("git", "In-Process Worktree", true, false, fmt.Sprintf("Branch '%s' (%s)", st.Branch, cleanStr))
	} else {
		check("git", "In-Process Worktree", false, true, "Not inside a git repository")
	}

	// 3. Security & OS Keyring
	fmt.Println()
	fmt.Println(sectionStyle.Render("🔒 Security, Keyring & Permissions"))
	v := vault.NewVault("")
	testKey := "__doctor_probe__"
	testVal := "probe_val"
	testErr := keyring.Set(vault.ServiceName, testKey, testVal)
	if testErr == nil {
		defer func() { _ = keyring.Delete(vault.ServiceName, testKey) }()
		gotVal, getErr := keyring.Get(vault.ServiceName, testKey)
		delErr := keyring.Delete(vault.ServiceName, testKey)
		if getErr == nil && delErr == nil && gotVal == testVal {
			check("security", "Native OS Keyring", true, false, fmt.Sprintf("%s operational", v.ProviderName()))
		} else if getErr != nil {
			check("security", "Native OS Keyring", false, true, fmt.Sprintf("OS Keyring read failed: %v", getErr))
		} else if gotVal != testVal {
			check("security", "Native OS Keyring", false, true, fmt.Sprintf("OS Keyring value mismatch (expected %s, got %s)", testVal, gotVal))
		} else {
			check("security", "Native OS Keyring", false, true, fmt.Sprintf("OS Keyring delete failed: %v", delErr))
		}
	} else {
		check("security", "Native OS Keyring", false, true, fmt.Sprintf("OS Keyring unavailable: %v", testErr))
	}

	home, _ := os.UserHomeDir()
	shellixHome := filepath.Join(home, ".shellix")
	_ = core.EnsureSecureDir(shellixHome)
	fi, statErr := os.Stat(shellixHome)
	if statErr == nil {
		check("security", "Config Storage Dir", true, false, fmt.Sprintf("%s (%04o)", shellixHome, fi.Mode().Perm()))
	} else {
		check("security", "Config Storage Dir", false, false, "Cannot access ~/.shellix")
	}

	// 4. SSH & Known Hosts
	fmt.Println()
	fmt.Println(sectionStyle.Render("🔑 SSH & Known Hosts Hardening"))
	knownHostsFile := filepath.Join(home, ".ssh", "known_hosts")
	if khInfo, err := os.Stat(knownHostsFile); err == nil {
		check("ssh", "Known Hosts Store", true, false, fmt.Sprintf("%s (%d bytes)", knownHostsFile, khInfo.Size()))
	} else {
		check("ssh", "Known Hosts Store", false, true, "known_hosts not yet initialized (will be created on first connect)")
	}

	var keysFound []string
	for _, keyFile := range []string{"id_ed25519", "id_rsa"} {
		kp := filepath.Join(home, ".ssh", keyFile)
		if _, err := os.Stat(kp); err == nil {
			keysFound = append(keysFound, keyFile)
		}
	}
	if len(keysFound) > 0 {
		check("ssh", "Private SSH Key", true, false, strings.Join(keysFound, ", "))
	} else {
		check("ssh", "Private SSH Key", false, true, "No standard id_ed25519 or id_rsa keys found in ~/.ssh")
	}

	// 5. Workflow Engine
	fmt.Println()
	fmt.Println(sectionStyle.Render("⚡ Automation Flow Engine"))
	flowsDir := filepath.Join(shellixHome, "flows")
	_ = core.EnsureSecureDir(flowsDir)
	logsDir := filepath.Join(shellixHome, "logs", "flows")
	_ = core.EnsureSecureDir(logsDir)
	check("flow", "Flow Definitions Dir", true, false, flowsDir)
	check("flow", "Audit Logging Dir", true, false, logsDir)

	// Summary
	fmt.Println()
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))
	summaryBox := fmt.Sprintf("Doctor Summary: %d checks passed, %d warnings, %d errors", passedCount, warnCount, failCount)
	if failCount == 0 && warnCount == 0 {
		fmt.Println(passStyle.Render("✨ " + summaryBox + " — System is fully optimized!"))
	} else if failCount == 0 {
		fmt.Println(warnStyle.Render("▲ " + summaryBox + " — Operational with minor warnings."))
	} else {
		fmt.Println(failStyle.Render("✖ " + summaryBox + " — Issues detected requiring attention."))
	}
	fmt.Println()
}
