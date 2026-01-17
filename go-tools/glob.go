package tools

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
)

// GlobTool matches files against a glob pattern
func GlobTool(pattern string, pathStr string) (string, error) {
	searchPath := pathStr
	if searchPath == "" {
		var err error
		searchPath, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	absPath, err := filepath.Abs(searchPath)
	if err != nil {
		return "", err
	}

	// Use doublestar for globbing
	// We need to run glob relative to searchPath?
	// The original tool uses ripgrep with glob, which runs in cwd.

	fsys := os.DirFS(absPath)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return "", err
	}

	var files []struct {
		Path  string
		Mtime int64
	}

	for _, m := range matches {
		fullPath := filepath.Join(absPath, m)
		info, err := os.Stat(fullPath)
		if err == nil {
			files = append(files, struct{Path string; Mtime int64}{
				Path: fullPath,
				Mtime: info.ModTime().Unix(),
			})
		}
	}

	// Sort by mtime descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].Mtime > files[j].Mtime
	})

	limit := 100
	truncated := false
	if len(files) > limit {
		files = files[:limit]
		truncated = true
	}

	if len(files) == 0 {
		return "No files found", nil
	}

	output := ""
	for _, f := range files {
		output += f.Path + "\n"
	}

	if truncated {
		output += "\n(Results are truncated. Consider using a more specific path or pattern.)"
	}

	return output, nil
}
