//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// No session isolation on Windows
}

func killProcessGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

func notifySignals(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, os.Interrupt)
}

func isResizeSignal(_ os.Signal) bool {
	return false
}

func selfSuspend() {
	// No SIGSTOP on Windows; no-op
}

func execProcess(path string, args []string, _ []string) error {
	return fmt.Errorf("exec not supported on Windows; use os/exec to start %s %v", path, args)
}
