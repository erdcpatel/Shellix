package java

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectEnvironment scans the current directory and host for Java SDK and build setups
func DetectEnvironment(dir string) (*EnvironmentInfo, error) {
	if dir == "" {
		dir = "."
	}

	info := &EnvironmentInfo{
		JavaHome:  os.Getenv("JAVA_HOME"),
		BuildTool: BuildToolUnknown,
	}

	// 1. Detect java runtime version
	cmd := exec.Command("java", "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		lines := strings.Split(out.String(), "\n")
		if len(lines) > 0 {
			info.JavaVersion = strings.TrimSpace(lines[0])
		}
	}

	// 2. Detect javac compiler version
	cmdJavac := exec.Command("javac", "-version")
	var outJavac bytes.Buffer
	cmdJavac.Stdout = &outJavac
	cmdJavac.Stderr = &outJavac
	if err := cmdJavac.Run(); err == nil {
		info.JavacVersion = strings.TrimSpace(outJavac.String())
	}

	// 3. Detect build tools in directory
	pomPath := filepath.Join(dir, "pom.xml")
	gradlePath := filepath.Join(dir, "build.gradle")
	gradleKtsPath := filepath.Join(dir, "build.gradle.kts")

	if fileExists(pomPath) {
		info.BuildTool = BuildToolMaven
		info.BuildFile = "pom.xml"

		wrapperName := "mvnw"
		if runtime.GOOS == "windows" {
			wrapperName = "mvnw.cmd"
		}
		wrapperPath := filepath.Join(dir, wrapperName)
		if fileExists(wrapperPath) {
			info.HasWrapper = true
			info.WrapperBinary = "./" + wrapperName
		}
		if p, err := exec.LookPath("mvn"); err == nil {
			info.SystemBinary = p
		}
	} else if fileExists(gradlePath) || fileExists(gradleKtsPath) {
		info.BuildTool = BuildToolGradle
		if fileExists(gradlePath) {
			info.BuildFile = "build.gradle"
		} else {
			info.BuildFile = "build.gradle.kts"
		}

		wrapperName := "gradlew"
		if runtime.GOOS == "windows" {
			wrapperName = "gradlew.bat"
		}
		wrapperPath := filepath.Join(dir, wrapperName)
		if fileExists(wrapperPath) {
			info.HasWrapper = true
			info.WrapperBinary = "./" + wrapperName
		}
		if p, err := exec.LookPath("gradle"); err == nil {
			info.SystemBinary = p
		}
	}

	// Count .java source files
	_ = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() && (fi.Name() == ".git" || fi.Name() == "target" || fi.Name() == "build") {
			return filepath.SkipDir
		}
		if !fi.IsDir() && strings.HasSuffix(fi.Name(), ".java") {
			info.SourceFilesCount++
		}
		return nil
	})

	return info, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
