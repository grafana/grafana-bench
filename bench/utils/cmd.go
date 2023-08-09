package utils

import (
	"fmt"
	"io"
	"log"
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

	// Set up stderr pipe
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Error creating stderr pipe: %w", err)
	}

	// Set up stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Error creating stdout pipe: %w", err)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting command: %w", err)
	}

	// Copy the output to stdout in real-time
	go func() {
		_, err := io.Copy(os.Stdout, stdout)
		if err != nil {
			log.Println("Error outputting to stdout:", err)
		}
	}()

	// Copy the output to stderr in real-time
	go func() {
		_, err := io.Copy(os.Stderr, stderr)
		if err != nil {
			log.Println("Error outputting to stdout:", err)
		}
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("Error waiting for command: %w", err)
	}

	return nil
}
