package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GrepTool searches for a pattern in files
func GrepTool(pattern string, pathStr string, include string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

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

	// Try to find ripgrep (rg)
	rgPath, err := exec.LookPath("rg")
	if err != nil {
		// Fallback to grep? Or return error?
		// The original tool uses Ripgrep explicitly.
		// For the logic recreation, using `grep -r` is a reasonable fallback if rg is missing,
		// but let's assume we want to use `rg` if possible or fail/implement basic search in Go.
		// Implementing grep in pure Go is complex for full regex support.
		// Let's assume `grep` is available as a fallback.

		grepPath, err := exec.LookPath("grep")
		if err != nil {
			return "", fmt.Errorf("neither 'rg' nor 'grep' found in PATH")
		}

		// Basic grep fallback
		args := []string{"-rn", pattern, absPath}
		cmd := exec.Command(grepPath, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// grep returns 1 if no matches found
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
				return "No files found", nil
			}
			return "", err
		}
		return string(out), nil
	}

	// Construct rg arguments to match the TS tool
	args := []string{
		"-nH", // line number, file name
		"--hidden",
		"--follow",
		"--no-messages",
		"--field-match-separator=|",
		"--regexp", pattern,
	}

	if include != "" {
		args = append(args, "--glob", include)
	}

	args = append(args, absPath)

	cmd := exec.Command(rgPath, args...)
	outputBytes, _ := cmd.CombinedOutput()
	// Ignore error as rg returns 1 for no matches, which is fine.

	output := string(outputBytes)
	if output == "" {
		return "No files found", nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Parse and format output to match the tool's format
	// Format: filePath|lineNum|text

	formattedLines := []string{fmt.Sprintf("Found %d matches", len(lines))}
	currentFile := ""

	count := 0
	limit := 100

	for _, line := range lines {
		if count >= limit {
			formattedLines = append(formattedLines, "", "(Results are truncated. Consider using a more specific path or pattern.)")
			break
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		fPath := parts[0]
		lineNum := parts[1]
		text := parts[2]

		if fPath != currentFile {
			if currentFile != "" {
				formattedLines = append(formattedLines, "")
			}
			currentFile = fPath
			formattedLines = append(formattedLines, fPath+":")
		}

		if len(text) > 2000 {
			text = text[:2000] + "..."
		}

		formattedLines = append(formattedLines, fmt.Sprintf("  Line %s: %s", lineNum, text))
		count++
	}

	return strings.Join(formattedLines, "\n"), nil
}
