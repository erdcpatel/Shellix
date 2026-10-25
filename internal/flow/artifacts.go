package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ArtifactFile describes a collected file artifact from a workflow step
type ArtifactFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// ArtifactRunManifest tracks all artifacts produced in a given workflow run
type ArtifactRunManifest struct {
	FlowName  string         `json:"flow_name"`
	RunID     string         `json:"run_id"`
	Timestamp time.Time      `json:"timestamp"`
	Files     []ArtifactFile `json:"files"`
}

// GetArtifactsBaseDir returns ~/.shellix/artifacts/
func GetArtifactsBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".shellix", "artifacts")
}

// CollectArtifacts searches workDir for matching glob patterns and copies them to the run archive
func CollectArtifacts(flowName, runID, workDir string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	if workDir == "" {
		workDir = "."
	}

	targetDir := filepath.Join(GetArtifactsBaseDir(), flowName, runID)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating artifact archive directory: %w", err)
	}

	var collected []ArtifactFile
	var collectedNames []string

	for _, pattern := range patterns {
		cleanPattern := pattern
		if !filepath.IsAbs(cleanPattern) {
			cleanPattern = filepath.Join(workDir, pattern)
		}

		matches, err := filepath.Glob(cleanPattern)
		if err != nil {
			continue
		}

		for _, src := range matches {
			stat, err := os.Stat(src)
			if err != nil || stat.IsDir() {
				continue
			}

			destPath := filepath.Join(targetDir, stat.Name())
			if err := copyFile(src, destPath); err != nil {
				continue
			}

			collected = append(collected, ArtifactFile{
				Name: stat.Name(),
				Size: stat.Size(),
				Path: destPath,
			})
			collectedNames = append(collectedNames, stat.Name())
		}
	}

	if len(collected) > 0 {
		manifestPath := filepath.Join(targetDir, "manifest.json")
		manifest := ArtifactRunManifest{
			FlowName:  flowName,
			RunID:     runID,
			Timestamp: time.Now(),
			Files:     collected,
		}
		raw, _ := json.MarshalIndent(manifest, "", "  ")
		_ = os.WriteFile(manifestPath, raw, 0600)
	}

	return collectedNames, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ListArtifactRuns returns all preserved artifact runs for a specific flow
func ListArtifactRuns(flowName string) ([]ArtifactRunManifest, error) {
	flowDir := filepath.Join(GetArtifactsBaseDir(), flowName)
	if _, err := os.Stat(flowDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(flowDir)
	if err != nil {
		return nil, err
	}

	var manifests []ArtifactRunManifest
	for _, e := range entries {
		if e.IsDir() {
			manifestFile := filepath.Join(flowDir, e.Name(), "manifest.json")
			if data, err := os.ReadFile(manifestFile); err == nil {
				var m ArtifactRunManifest
				if err := json.Unmarshal(data, &m); err == nil {
					manifests = append(manifests, m)
				}
			}
		}
	}

	return manifests, nil
}

// FormatFileSize formats byte count for human display
func FormatFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
