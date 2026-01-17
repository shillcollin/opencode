package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteTool writes content to a file
func WriteTool(filePath string, content string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// In the real tool, it handles diffs and permissions.
	// Here we just write the file.

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	err = os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	// The original tool returns LSP diagnostics. We will skip that for now as it requires an LSP client.
	return "Wrote file successfully.", nil
}
