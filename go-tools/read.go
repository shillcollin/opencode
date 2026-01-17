package tools

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	DEFAULT_READ_LIMIT = 2000
	MAX_LINE_LENGTH    = 2000
	MAX_BYTES          = 50 * 1024
)

// ReadTool reads a file
func ReadTool(filePath string, offset, limit int) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		// Attempt to provide suggestions (simplified)
		dir := filepath.Dir(absPath)
		base := filepath.Base(absPath)

		suggestions := []string{}
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.Contains(strings.ToLower(name), strings.ToLower(base)) ||
				   strings.Contains(strings.ToLower(base), strings.ToLower(name)) {
					suggestions = append(suggestions, filepath.Join(dir, name))
					if len(suggestions) >= 3 {
						break
					}
				}
			}
		}

		errMsg := fmt.Sprintf("File not found: %s", absPath)
		if len(suggestions) > 0 {
			errMsg += fmt.Sprintf("\n\nDid you mean one of these?\n%s", strings.Join(suggestions, "\n"))
		}
		return "", fmt.Errorf("%s", errMsg)
	}
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", absPath)
	}

	// Check binary
	isBin, err := isBinaryFile(absPath)
	if err != nil {
		return "", err
	}
	if isBin {
		return "", fmt.Errorf("cannot read binary file: %s", absPath)
	}

	if limit == 0 {
		limit = DEFAULT_READ_LIMIT
	}

	file, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var raw []string
	bytesRead := 0
	truncatedByBytes := false

	scanner := bufio.NewScanner(file)
	// Increase buffer size for scanner if needed, though default is usually fine for code

	currentLine := 0
	linesRead := 0

	for scanner.Scan() {
		if currentLine < offset {
			currentLine++
			continue
		}

		if linesRead >= limit {
			break
		}

		line := scanner.Text()
		if len(line) > MAX_LINE_LENGTH {
			line = line[:MAX_LINE_LENGTH] + "..."
		}

		size := len(line) + 1 // +1 for newline approximation
		if bytesRead+size > MAX_BYTES {
			truncatedByBytes = true
			break
		}

		raw = append(raw, line)
		bytesRead += size
		linesRead++
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	var content []string
	for i, line := range raw {
		content = append(content, fmt.Sprintf("%05d| %s", i+offset+1, line))
	}

	output := "<file>\n" + strings.Join(content, "\n")

	hasMoreLines := false
	// Check if there are more lines
	if scanner.Scan() {
		hasMoreLines = true
	}

	lastReadLine := offset + len(raw)

	if truncatedByBytes {
		output += fmt.Sprintf("\n\n(Output truncated at %d bytes. Use 'offset' parameter to read beyond line %d)", MAX_BYTES, lastReadLine)
	} else if hasMoreLines {
		output += fmt.Sprintf("\n\n(File has more lines. Use 'offset' parameter to read beyond line %d)", lastReadLine)
	} else {
		// We need total lines for "End of file - total X lines"
		// To do this efficiently without reading the whole file again, we might just skip this info or make a rough estimate/count.
		// The TS implementation reads the WHOLE file into memory `file.text()`.
		// For large files this is bad, but for parity:
		output += fmt.Sprintf("\n\n(End of file - read %d lines)", linesRead)
	}

	output += "\n</file>"
	return output, nil
}

func isBinaryFile(pathStr string) (bool, error) {
	// Extension check
	ext := filepath.Ext(pathStr)
	switch strings.ToLower(ext) {
	case ".zip", ".tar", ".gz", ".exe", ".dll", ".so", ".class", ".jar", ".war", ".7z",
		".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp",
		".bin", ".dat", ".obj", ".o", ".a", ".lib", ".wasm", ".pyc", ".pyo":
		return true, nil
	}

	f, err := os.Open(pathStr)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Use http.DetectContentType
	contentType := http.DetectContentType(buffer[:n])
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "application/pdf") {
		// Treat as binary for reading purposes (or handle separately as in TS)
		return true, nil
	}

	// Also check for null bytes or control characters as in TS implementation
	if n == 0 {
		return false, nil
	}

	nonPrintableCount := 0
	for _, b := range buffer[:n] {
		if b == 0 {
			return true, nil
		}
		if b < 9 || (b > 13 && b < 32) {
			nonPrintableCount++
		}
	}

	if float64(nonPrintableCount)/float64(n) > 0.3 {
		return true, nil
	}

	// Also check for valid UTF-8
	if !utf8.Valid(buffer[:n]) {
		// This might be too aggressive for some encodings, but good proxy for binary vs text
	}

	return false, nil
}
