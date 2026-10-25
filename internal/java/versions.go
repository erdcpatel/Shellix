package java

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// InstalledJDK represents a Java development kit discovered on the system
type InstalledJDK struct {
	Version  string `json:"version"`
	Major    int    `json:"major"`
	Vendor   string `json:"vendor"`
	HomePath string `json:"home_path"`
	Arch     string `json:"arch"`
	IsActive bool   `json:"is_active"`
}

var versionMajorRegex = regexp.MustCompile(`^(\d+)`)

// DiscoverAllJDKs scans system directories and tooling for all installed JDKs
func DiscoverAllJDKs() ([]InstalledJDK, error) {
	currentJavaHome := os.Getenv("JAVA_HOME")
	seenPaths := make(map[string]bool)
	var jdks []InstalledJDK

	addJDK := func(jdk InstalledJDK) {
		clean := filepath.Clean(jdk.HomePath)
		if seenPaths[clean] {
			return
		}
		seenPaths[clean] = true
		if currentJavaHome != "" && clean == filepath.Clean(currentJavaHome) {
			jdk.IsActive = true
		}
		jdks = append(jdks, jdk)
	}

	// 1. macOS specific: /usr/libexec/java_home -V
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("/usr/libexec/java_home", "-V")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		_ = cmd.Run()

		// Output lines format:
		//     25.0.1 (arm64) "Eclipse Adoptium" - "OpenJDK 25.0.1" /Library/Java/JavaVirtualMachines/.../Home
		lineRegex := regexp.MustCompile(`^\s*(\d+[\.\d_]*)\s+\(([^)]+)\)\s+"([^"]+)"\s+-\s+"([^"]+)"\s+(.+)$`)
		for _, line := range strings.Split(out.String(), "\n") {
			m := lineRegex.FindStringSubmatch(line)
			if len(m) >= 6 {
				ver := m[1]
				arch := m[2]
				vendor := m[3]
				homePath := strings.TrimSpace(m[5])

				major := parseMajor(ver)
				addJDK(InstalledJDK{
					Version:  ver,
					Major:    major,
					Vendor:   vendor,
					HomePath: homePath,
					Arch:     arch,
				})
			}
		}

		// Also check ~/Library/Java/JavaVirtualMachines and /Library/Java/JavaVirtualMachines
		macRoots := []string{
			"/Library/Java/JavaVirtualMachines",
		}
		if home, err := os.UserHomeDir(); err == nil {
			macRoots = append(macRoots, filepath.Join(home, "Library", "Java", "JavaVirtualMachines"))
		}
		for _, r := range macRoots {
			scanMacJVMDir(r, addJDK)
		}
	}

	// 2. Linux specific paths
	if runtime.GOOS == "linux" {
		scanDirectoryForJDKs("/usr/lib/jvm", addJDK)
	}

	// 3. Windows specific paths
	if runtime.GOOS == "windows" {
		programFiles := os.Getenv("ProgramFiles")
		if programFiles != "" {
			scanDirectoryForJDKs(filepath.Join(programFiles, "Java"), addJDK)
			scanDirectoryForJDKs(filepath.Join(programFiles, "Eclipse Adoptium"), addJDK)
			scanDirectoryForJDKs(filepath.Join(programFiles, "Amazon Corretto"), addJDK)
			scanDirectoryForJDKs(filepath.Join(programFiles, "Microsoft"), addJDK)
		}
	}

	// 4. Common cross-platform toolchains: SDKMAN!
	if home, err := os.UserHomeDir(); err == nil {
		sdkmanDir := filepath.Join(home, ".sdkman", "candidates", "java")
		scanDirectoryForJDKs(sdkmanDir, addJDK)

		asdfDir := filepath.Join(home, ".asdf", "installs", "java")
		scanDirectoryForJDKs(asdfDir, addJDK)
	}

	// 5. If nothing matched active, check if default `java` matches any path
	if len(jdks) > 0 && currentJavaHome == "" {
		jdks[0].IsActive = true
	}

	return jdks, nil
}

func scanMacJVMDir(root string, addFn func(InstalledJDK)) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			homePath := filepath.Join(root, e.Name(), "Contents", "Home")
			binJava := filepath.Join(homePath, "bin", "java")
			if fileExists(binJava) {
				ver, vendor := queryJavaVersion(binJava)
				addFn(InstalledJDK{
					Version:  ver,
					Major:    parseMajor(ver),
					Vendor:   vendor,
					HomePath: homePath,
					Arch:     runtime.GOARCH,
				})
			}
		}
	}
}

func scanDirectoryForJDKs(root string, addFn func(InstalledJDK)) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			candidateHome := filepath.Join(root, e.Name())
			binName := "java"
			if runtime.GOOS == "windows" {
				binName = "java.exe"
			}
			binPath := filepath.Join(candidateHome, "bin", binName)
			if fileExists(binPath) {
				ver, vendor := queryJavaVersion(binPath)
				addFn(InstalledJDK{
					Version:  ver,
					Major:    parseMajor(ver),
					Vendor:   vendor,
					HomePath: candidateHome,
					Arch:     runtime.GOARCH,
				})
			}
		}
	}
}

func queryJavaVersion(javaBin string) (string, string) {
	cmd := exec.Command(javaBin, "-version")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "unknown", "unknown"
	}

	lines := strings.Split(buf.String(), "\n")
	ver := "unknown"
	vendor := "OpenJDK"

	if len(lines) > 0 {
		verLine := lines[0]
		// extract version "17.0.17"
		re := regexp.MustCompile(`"([^"]+)"`)
		m := re.FindStringSubmatch(verLine)
		if len(m) >= 2 {
			ver = m[1]
		}
	}
	if len(lines) > 1 {
		vendor = strings.TrimSpace(lines[1])
	}
	return ver, vendor
}

func parseMajor(ver string) int {
	parts := strings.Split(ver, ".")
	if len(parts) > 0 {
		if parts[0] == "1" && len(parts) > 1 {
			// legacy 1.8 -> 8
			v, _ := strconv.Atoi(parts[1])
			return v
		}
		v, _ := strconv.Atoi(parts[0])
		return v
	}
	return 0
}

// FindMatchingJDK searches discovered JDKs by major version (17, 19, 25) or keyword
func FindMatchingJDK(pattern string) (*InstalledJDK, error) {
	jdks, err := DiscoverAllJDKs()
	if err != nil {
		return nil, err
	}
	if len(jdks) == 0 {
		return nil, fmt.Errorf("no Java JDK installations discovered on this system")
	}

	norm := strings.ToLower(strings.TrimSpace(pattern))

	// 1. Exact or partial major number match (e.g. "17", "25", "19")
	if reqMajor, err := strconv.Atoi(norm); err == nil {
		for _, j := range jdks {
			if j.Major == reqMajor {
				return &j, nil
			}
		}
	}

	// 2. Substring match on version, vendor, or path
	for _, j := range jdks {
		if strings.Contains(strings.ToLower(j.Version), norm) ||
			strings.Contains(strings.ToLower(j.Vendor), norm) ||
			strings.Contains(strings.ToLower(j.HomePath), norm) {
			return &j, nil
		}
	}

	return nil, fmt.Errorf("no installed JDK matching '%s' found (use 'sx java list' to view available versions)", pattern)
}

// GenerateShellExport generates the appropriate command to activate the JDK in the user's shell
func GenerateShellExport(jdk *InstalledJDK) string {
	if runtime.GOOS == "windows" {
		binDir := filepath.Join(jdk.HomePath, "bin")
		return fmt.Sprintf("$env:JAVA_HOME = \"%s\"; $env:Path = \"%s;\" + $env:Path", jdk.HomePath, binDir)
	}

	binDir := filepath.Join(jdk.HomePath, "bin")
	return fmt.Sprintf("export JAVA_HOME=\"%s\"; export PATH=\"%s:$PATH\"", jdk.HomePath, binDir)
}

// LaunchSubshellWithJDK launches an interactive shell with the requested JDK applied
func LaunchSubshellWithJDK(jdk *InstalledJDK, customShell string) error {
	shell := customShell
	if shell == "" || shell == "auto" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			if runtime.GOOS == "windows" {
				shell = "powershell.exe"
			} else {
				shell = "/bin/bash"
			}
		}
	}

	binDir := filepath.Join(jdk.HomePath, "bin")
	newPath := binDir + string(os.PathListSeparator) + os.Getenv("PATH")

	cmd := exec.Command(shell)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("JAVA_HOME=%s", jdk.HomePath),
		fmt.Sprintf("PATH=%s", newPath),
		fmt.Sprintf("SHELLIX_JAVA_VERSION=%s", jdk.Version),
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
