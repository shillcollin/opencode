package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const DEFAULT_TIMEOUT = 2 * 60 * 1000 * time.Millisecond // 2 minutes

// BashTool executes a shell command
func BashTool(command string, workdir string, timeoutMs int) (string, error) {
	cwd := workdir
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	timeout := DEFAULT_TIMEOUT
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Using "bash" -c command
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = cwd
	cmd.Env = os.Environ() // Pass current environment

	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		outputStr += fmt.Sprintf("\n\nbash tool terminated command after exceeding timeout %v", timeout)
		return outputStr, fmt.Errorf("command timed out")
	}

	if err != nil {
		// Command failed, but we still want the output
		// We return nil error because the tool execution itself didn't fail (in the sense of the harness),
		// the command just returned non-zero. The output contains the error message.
		// However, adhering to the return signature (string, error), maybe we should return it.
		// But in Agent tools, usually exit code is part of metadata/output.
		// We'll append exit code info.
		outputStr += fmt.Sprintf("\nCommand exited with error: %v", err)
	}

	return outputStr, nil
}
