package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ExecStdoutWithEnv(cmd *exec.Cmd, env map[string]string) error {
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	return ExecStdout(cmd)
}

// ExecStdout runs a command and streams stdout+stderr to the terminal.
func ExecStdout(cmd *exec.Cmd) error {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Start the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error starting command: %w", err)
	}

	return nil
}
