package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ─── Command and System Tools ────────────────────────────────────────

func (t *ToolExecutor) runCommand(args map[string]any) string {
	command := getStringArg(args, "command", "")
	if command == "" {
		return errorMessage("command is required")
	}

	// Execute the command with full unrestricted access - NO RESTRICTIONS
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = t.baseDir

	// Run in a new session so child processes have no controlling terminal.
	// This prevents programs like ssh/git from opening /dev/tty directly to
	// prompt for passwords/passphrases, which would steal keystrokes from
	// yolo and leak output onto the user's terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// Explicitly connect stdin to /dev/null so child processes that try to
	// read input will get immediate EOF instead of hanging.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}

	// Capture stderr explicitly so it is always available, not just on
	// non-zero exit.  This also ensures stderr never leaks to the terminal.
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	done := make(chan struct{})
	var stdout []byte
	var cmdErr error

	go func() {
		defer close(done)
		stdout, cmdErr = cmd.Output()
	}()

	select {
	case <-done:
		// Command completed
	case <-time.After(time.Duration(CommandTimeout) * time.Second):
		if cmd.Process != nil {
			// Kill the entire process group (negative PID) since we used Setsid.
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return errorMessage("command timed out (%ds)", CommandTimeout)
	}

	var out strings.Builder
	if len(stdout) > 0 {
		out.Write(stdout)
	}
	stderrStr := stderrBuf.String()
	if len(stderrStr) > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString("STDERR: ")
		out.WriteString(stderrStr)
	}
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			out.WriteString(fmt.Sprintf("\n(exit code %d)", exitErr.ExitCode()))
		}
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return "Command completed successfully (no output)."
	}
	// Sanitize command output to prevent terminal escape sequences from
	// corrupting the display.
	result = sanitizeOutput(result)
	return result
}

func (t *ToolExecutor) restart(args map[string]any) string {
	exePath, err := os.Executable()
	if err != nil {
		return errorMessage("could not get executable path: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return errorMessage("could not get current directory: %v", err)
	}

	fmt.Fprintf(os.Stderr, "[RESTART] Rebuilding YOLO from source...\n")

	buildCmd := exec.Command("go", "build", "-o", filepath.Base(exePath), ".")
	buildCmd.Dir = cwd

	// Fully isolate: new session (no controlling terminal) + stdin from /dev/null.
	buildCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devNull, derr := os.Open(os.DevNull); derr == nil {
		buildCmd.Stdin = devNull
		defer devNull.Close()
	}

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return errorMessage("build failed: %v\n%s", err, string(output))
	}

	fmt.Fprintf(os.Stderr, "[RESTART] Build successful. Replacing current process...\n")

	if t.agent != nil {
		t.agent.history.AddMessage("tool", "[restart] Build successful. Restarting process with new binary...", nil)
		t.agent.history.AddEvolution("restart", "Rebuilt and restarting with new binary")
	}

	newExePath := filepath.Join(cwd, filepath.Base(exePath))

	var executableArgs []string
	for _, arg := range os.Args[1:] {
		if arg != "--restart" {
			executableArgs = append(executableArgs, arg)
		}
	}

	// Restore terminal state before exec so it's not left in raw mode
	if t.agent != nil && t.agent.inputMgr != nil {
		t.agent.inputMgr.Stop()
	}

	err = syscall.Exec(newExePath, append([]string{filepath.Base(exePath)}, executableArgs...), os.Environ())
	if err != nil {
		return errorMessage("could not start new process: %v", err)
	}

	return "Process replaced"
}
