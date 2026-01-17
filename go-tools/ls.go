package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IGNORE_PATTERNS defines default patterns to ignore
var IGNORE_PATTERNS = []string{
	"node_modules",
	"__pycache__",
	".git",
	"dist",
	"build",
	"target",
	"vendor",
	"bin",
	"obj",
	".idea",
	".vscode",
	".zig-cache",
	"zig-out",
	".coverage",
	"coverage",
	"tmp",
	"temp",
	".cache",
	"cache",
	"logs",
	".venv",
	"venv",
	"env",
}

const LIST_LIMIT = 100

// ListTool lists files in a directory
func ListTool(dirPath string, ignore []string) (string, error) {
	searchPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", err
	}

	// Check if directory exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("error accessing path: %v", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", searchPath)
	}

	// Combine ignore patterns
	ignorePatterns := make(map[string]bool)
	for _, p := range IGNORE_PATTERNS {
		ignorePatterns[p] = true
	}
	for _, p := range ignore {
		ignorePatterns[p] = true
	}

	var files []string

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path for filtering
		relPath, err := filepath.Rel(searchPath, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// Check ignore patterns
		// This is a simple check, a full glob implementation might be needed for strict parity
		// but checking path components against the list is a good start.
		parts := strings.Split(relPath, string(os.PathSeparator))
		for _, part := range parts {
			if ignorePatterns[part] || ignorePatterns[part+"/"] {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			files = append(files, relPath)
		}

		if len(files) >= LIST_LIMIT {
			// Optimization: we could stop walking, but we need to know if it's truncated
			// For exact parity with the TS tool which uses ripgrep, we might want to use ripgrep if available
			// But the prompt asks for logic recreation.
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	isTruncated := false
	if len(files) > LIST_LIMIT {
		files = files[:LIST_LIMIT]
		isTruncated = true
	}

	// Build directory structure for display
	dirs := make(map[string]bool)
	filesByDir := make(map[string][]string)

	for _, file := range files {
		dir := filepath.Dir(file)
		if dir == "." {
			// handled separately or as empty string
		}

		// Add all parent directories
		parts := strings.Split(dir, string(os.PathSeparator))
		currentPath := ""
		for i, part := range parts {
			if part == "." {
				continue
			}
			if i > 0 {
				currentPath += string(os.PathSeparator)
			}
			currentPath += part
			dirs[currentPath] = true
		}
		if dir == "." {
			dirs["."] = true
		} else {
			dirs[dir] = true
		}

		filesByDir[dir] = append(filesByDir[dir], filepath.Base(file))
	}

	output := searchPath + "/\n" + renderDir(".", 0, dirs, filesByDir)

	if isTruncated {
		output += fmt.Sprintf("\n(Truncated to %d files)", LIST_LIMIT)
	}

	return output, nil
}

func renderDir(dirPath string, depth int, dirs map[string]bool, filesByDir map[string][]string) string {
	indent := strings.Repeat("  ", depth)
	output := ""

	if depth > 0 && dirPath != "." {
		output += fmt.Sprintf("%s%s/\n", indent, filepath.Base(dirPath))
	}

	childIndent := strings.Repeat("  ", depth+1)

	// Find child directories
	var children []string
	for d := range dirs {
		parent := filepath.Dir(d)
		if parent == "." && dirPath == "." && d != "." {
			children = append(children, d)
		} else if parent == dirPath && d != dirPath {
			children = append(children, d)
		}
	}
	sort.Strings(children)

	for _, child := range children {
		output += renderDir(child, depth+1, dirs, filesByDir)
	}

	// Render files
	files := filesByDir[dirPath]
	sort.Strings(files)
	for _, file := range files {
		output += fmt.Sprintf("%s%s\n", childIndent, file)
	}

	return output
}
