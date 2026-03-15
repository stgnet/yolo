// Executor types for Git operations
package main

import (
	"os/exec"
)

// GitExecutor executes git commands
type GitExecutor struct {
	path string
}

// ToolExecutor interface defines the contract for tool executors
type ToolExecutor interface {
	gitListBranches(args map[string]any) (string, error)
	gitDiff(args map[string]any) (string, error)
	gitStatus(args map[string]any) (string, error)
}

// NewToolExecutor creates a new GitExecutor instance
func NewToolExecutor(basePath string, config interface{}) *GitExecutor {
	return &GitExecutor{path: basePath}
}

// gitListBranches lists all branches in the repository
func (g *GitExecutor) gitListBranches(args map[string]any) (string, error) {
	cmd := exec.Command("git", "branch", "-a")
	if g.path != "" {
		cmd.Dir = g.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// gitDiff shows the diff between commits or working tree
func (g *GitExecutor) gitDiff(args map[string]any) (string, error) {
	cmd := exec.Command("git", "diff")
	if g.path != "" {
		cmd.Dir = g.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// gitStatus shows the working tree status
func (g *GitExecutor) gitStatus(args map[string]any) (string, error) {
	cmd := exec.Command("git", "status")
	if g.path != "" {
		cmd.Dir = g.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
