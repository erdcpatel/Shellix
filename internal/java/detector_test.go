package java

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectEnvironmentMaven(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "java-test-mvn-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create fake pom.xml
	_ = os.WriteFile(filepath.Join(tempDir, "pom.xml"), []byte("<project></project>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "Main.java"), []byte("public class Main {}"), 0644)

	info, err := DetectEnvironment(tempDir)
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	if info.BuildTool != BuildToolMaven {
		t.Errorf("expected Maven build tool, got %v", info.BuildTool)
	}
	if info.BuildFile != "pom.xml" {
		t.Errorf("expected pom.xml, got %v", info.BuildFile)
	}
	if info.SourceFilesCount != 1 {
		t.Errorf("expected 1 source file, got %d", info.SourceFilesCount)
	}
}

func TestDetectEnvironmentGradle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "java-test-gradle-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create fake build.gradle
	_ = os.WriteFile(filepath.Join(tempDir, "build.gradle"), []byte("plugins { id 'java' }"), 0644)

	info, err := DetectEnvironment(tempDir)
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	if info.BuildTool != BuildToolGradle {
		t.Errorf("expected Gradle build tool, got %v", info.BuildTool)
	}
}
