//go:build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func killProcessGroup(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}

func notifySignals(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, os.Interrupt, syscall.SIGWINCH)
}

func isResizeSignal(sig os.Signal) bool {
	return sig == syscall.SIGWINCH
}

func selfSuspend() {
	syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
}

func execProcess(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}
