package terminal

import (
	"os"
	"path/filepath"
	"strings"
)

// getExeDir extracts the directory of an executable path across both Windows and Unix
func getExeDir(exePath string) string {
	if idx := strings.LastIndexAny(exePath, `/\`); idx != -1 {
		return exePath[:idx]
	}
	return filepath.Dir(exePath)
}

// determineSeparator detects if the PATH uses Windows semicolon or Unix colon
func determineSeparator(currentVal string) string {
	if strings.Contains(currentVal, ";") || os.PathListSeparator == ';' || strings.Contains(currentVal, `\`) {
		return ";"
	}
	return string(os.PathListSeparator)
}

// BuildShellEnvironment constructs the process environment for the spawned child subshell.
// It prepends the directory containing exePath to PATH, appends any extraPaths to PATH,
// and ensures SHELLIX_BIN is exported with exePath.
func BuildShellEnvironment(baseEnv []string, exePath string, extraPaths ...string) []string {
	exeDir := getExeDir(exePath)
	var newEnv []string
	pathFound := false

	for _, entry := range baseEnv {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "PATH") {
			pathFound = true
			key := parts[0]
			currentVal := parts[1]

			updatedPath := composePath(exeDir, currentVal, extraPaths)
			newEnv = append(newEnv, key+"="+updatedPath)
		} else if len(parts) == 2 && strings.EqualFold(parts[0], "SHELLIX_BIN") {
			// Will be explicitly set below to ensure up-to-date value
			continue
		} else {
			newEnv = append(newEnv, entry)
		}
	}

	if !pathFound {
		updatedPath := composePath(exeDir, "", extraPaths)
		newEnv = append(newEnv, "PATH="+updatedPath)
	}

	newEnv = append(newEnv, "SHELLIX_BIN="+exePath)
	return newEnv
}

// composePath joins prependDir, currentVal, and appendDirs using the OS path list separator
// while removing empty elements and duplicates.
func composePath(prependDir, currentVal string, appendDirs []string) string {
	sep := determineSeparator(currentVal)
	var elements []string

	cleanPath := func(p string) string {
		p = strings.TrimRight(p, `/\`)
		return filepath.Clean(p)
	}

	if prependDir != "" {
		elements = append(elements, prependDir)
	}

	if currentVal != "" {
		for _, p := range strings.Split(currentVal, sep) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			cleanP := cleanPath(p)
			// Avoid duplicating prependDir if it was already at the front
			if len(elements) > 0 && strings.EqualFold(cleanP, cleanPath(elements[0])) {
				continue
			}
			elements = append(elements, p)
		}
	}

	for _, extra := range appendDirs {
		extra = strings.TrimSpace(extra)
		if extra == "" {
			continue
		}
		cleanExtra := cleanPath(extra)
		alreadyExists := false
		for _, el := range elements {
			if strings.EqualFold(cleanPath(el), cleanExtra) {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			elements = append(elements, extra)
		}
	}

	return strings.Join(elements, sep)
}
