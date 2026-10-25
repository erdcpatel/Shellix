package java

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// PrintEnv displays styled information about the detected Java setup
func PrintEnv(dir string) error {
	info, err := DetectEnvironment(dir)
	if err != nil {
		return err
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AF68"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(18)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C0CAF5"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	fmt.Println(titleStyle.Render("☕ Shellix Java Environment & Toolchain"))
	fmt.Println()

	javaVer := info.JavaVersion
	if javaVer == "" {
		javaVer = warnStyle.Render("Not found in PATH")
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Runtime:"), valStyle.Render(javaVer))

	javacVer := info.JavacVersion
	if javacVer == "" {
		javacVer = metaStyle.Render("Not available (JRE only or not in PATH)")
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Compiler:"), valStyle.Render(javacVer))

	javaHome := info.JavaHome
	if javaHome == "" {
		javaHome = metaStyle.Render("(JAVA_HOME not explicitly exported)")
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("JAVA_HOME:"), valStyle.Render(javaHome))

	buildToolStr := string(info.BuildTool)
	if info.BuildTool == BuildToolMaven {
		buildToolStr = fmt.Sprintf("Maven (file: %s)", info.BuildFile)
		if info.HasWrapper {
			buildToolStr += " [Wrapper present: " + info.WrapperBinary + "]"
		}
	} else if info.BuildTool == BuildToolGradle {
		buildToolStr = fmt.Sprintf("Gradle (file: %s)", info.BuildFile)
		if info.HasWrapper {
			buildToolStr += " [Wrapper present: " + info.WrapperBinary + "]"
		}
	} else {
		buildToolStr = metaStyle.Render("None detected in current directory")
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Build System:"), valStyle.Render(buildToolStr))
	fmt.Printf("  %s %s\n", labelStyle.Render("Java Sources:"), valStyle.Render(fmt.Sprintf("%d files", info.SourceFilesCount)))

	return nil
}

// Build executes the project's build tool (Maven or Gradle) with auto-wrapper resolution
func Build(dir string, userArgs []string) error {
	info, err := DetectEnvironment(dir)
	if err != nil {
		return err
	}

	var bin string
	var defaultArgs []string

	if info.BuildTool == BuildToolMaven {
		if info.HasWrapper {
			bin = info.WrapperBinary
		} else if info.SystemBinary != "" {
			bin = info.SystemBinary
		} else {
			bin = "mvn"
		}
		if len(userArgs) == 0 {
			defaultArgs = []string{"clean", "compile"}
		}
	} else if info.BuildTool == BuildToolGradle {
		if info.HasWrapper {
			bin = info.WrapperBinary
		} else if info.SystemBinary != "" {
			bin = info.SystemBinary
		} else {
			bin = "gradle"
		}
		if len(userArgs) == 0 {
			defaultArgs = []string{"build"}
		}
	} else {
		return fmt.Errorf("no pom.xml or build.gradle found in %s; cannot trigger automated Java build", dir)
	}

	args := defaultArgs
	if len(userArgs) > 0 {
		args = userArgs
	}

	actionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	fmt.Printf("%s Invoking %s %v in %s\n\n", actionStyle.Render("⚡ [JAVA BUILD]"), bin, args, dir)

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunSource executes a single .java source file directly (supported in Java 11+)
func RunSource(sourcePath string, appArgs []string) error {
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	args := append([]string{absPath}, appArgs...)
	cmd := exec.Command("java", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
