// Package tools provides YOLO's autonomous tool capabilities
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"yolo/config"
)

// ToolType represents the type of tool operation
type ToolType string

const (
	ToolTypeFile       ToolType = "file"
	ToolTypeWeb        ToolType = "web"
	ToolTypeSystem     ToolType = "system"
	ToolTypeSearch     ToolType = "search"
	ToolTypeAI         ToolType = "ai"
	ToolTypeCommunication ToolType = "communication"
)

// Tool defines the interface for all YOLO tools
type Tool interface {
	Name() string
	Description() string
	Type() ToolType
	Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success   bool
	Output    string
	Error     string
	Metadata  map[string]interface{}
	Duration  time.Duration
}

// Register all tools
func RegisterTools() []Tool {
	return []Tool{
		// File operations
		&FileReadTool{},
		&FileWriteTool{},
		&FileEditTool{},
		&FileListTool{},
		&FileSearchTool{},
		
		// System operations
		&RunCommandTool{},
		&MakeDirTool{},
		&RemoveDirTool{},
		&CopyFileTool{},
		&MoveFileTool{},
		
		// Web operations
		&WebSearchTool{},
		&ReadWebpageTool{},
		&RedditTool{},
		&PlaywrightTool{},
		
		// Communication tools
		&SendEmailTool{},
		&SendReportTool{},
		&CheckInboxTool{},
		&ProcessInboxTool{},
		&GOGTool{},
		
		// AI tools
		&LearnTool{},
		&ImplementTool{},
		&ListModelsTool{},
		&SwitchModelTool{},
		
		// Todo management
		&AddTodoTool{},
		&CompleteTodoTool{},
		&DeleteTodoTool{},
		&ListTodosTool{},
	}
}

// File operations

type FileReadTool struct{}

func (t *FileReadTool) Name() string { return "read_file" }
func (t *FileReadTool) Description() string { return "Read a file's contents. For large files, use offset and limit to read in chunks." }
func (t *FileReadTool) Type() ToolType { return ToolTypeFile }

func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Success: false, Error: "path is required", Duration: time.Since(start)}, nil
	}
	
	offset, _ := args["offset"].(int)
	if offset == 0 {
		offset = 1
	}
	
	limit, _ := args["limit"].(int)
	if limit == 0 {
		limit = 200
	}
	
	content, err := readFileChunks(path, offset, limit)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   content,
		Metadata: map[string]interface{}{"lines_read": len(strings.Split(content, "\n"))},
		Duration: time.Since(start),
	}, nil
}

type FileWriteTool struct{}

func (t *FileWriteTool) Name() string { return "write_file" }
func (t *FileWriteTool) Description() string { return "Create or overwrite a file" }
func (t *FileWriteTool) Type() ToolType { return ToolTypeFile }

func (t *FileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Success: false, Error: "path is required", Duration: time.Since(start)}, nil
	}
	
	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{Success: false, Error: "content is required", Duration: time.Since(start)}, nil
	}
	
	err := writeToFile(path, content)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
		Duration: time.Since(start),
	}, nil
}

type FileEditTool struct{}

func (t *FileEditTool) Name() string { return "edit_file" }
func (t *FileEditTool) Description() string { return "Replace first occurrence of old_text with new_text in a file" }
func (t *FileEditTool) Type() ToolType { return ToolTypeFile }

func (t *FileEditTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	path, _ := args["path"].(string)
	oldText, _ := args["old_text"].(string)
	newText, _ := args["new_text"].(string)
	
	if path == "" || oldText == "" {
		return &ToolResult{Success: false, Error: "path and old_text are required", Duration: time.Since(start)}, nil
	}
	
	result, err := replaceInFile(path, oldText, newText)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   result,
		Duration: time.Since(start),
	}, nil
}

type FileListTool struct{}

func (t *FileListTool) Name() string { return "list_files" }
func (t *FileListTool) Description() string { return "List files matching a glob pattern" }
func (t *FileListTool) Type() ToolType { return ToolTypeFile }

func (t *FileListTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		pattern = "*"
	}
	
	files, err := listFiles(pattern)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   strings.Join(files, "\n"),
		Metadata: map[string]interface{}{"count": len(files)},
		Duration: time.Since(start),
	}, nil
}

type FileSearchTool struct{}

func (t *FileSearchTool) Name() string { return "search_files" }
func (t *FileSearchTool) Description() string { return "Search file contents using regex" }
func (t *FileSearchTool) Type() ToolType { return ToolTypeFile }

func (t *FileSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &ToolResult{Success: false, Error: "query is required", Duration: time.Since(start)}, nil
	}
	
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		pattern = "**/*"
	}
	
	results, err := searchFiles(query, pattern)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   results,
		Duration: time.Since(start),
	}, nil
}

// System operations

type RunCommandTool struct{}

func (t *RunCommandTool) Name() string { return "run_command" }
func (t *RunCommandTool) Description() string { return "Execute a shell command with timeout support" }
func (t *RunCommandTool) Type() ToolType { return ToolTypeSystem }

func (t *RunCommandTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return &ToolResult{Success: false, Error: "command is required", Duration: time.Since(start)}, nil
	}
	
	output, exitCode, err := runCommandWithTimeout(command, 30*time.Second)
	result := &ToolResult{Duration: time.Since(start)}
	
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
		result.Output = output
		result.Metadata = map[string]interface{}{"exit_code": exitCode}
	}
	
	return result, nil
}

type MakeDirTool struct{}

func (t *MakeDirTool) Name() string { return "make_dir" }
func (t *MakeDirTool) Description() string { return "Create a new directory recursively" }
func (t *MakeDirTool) Type() ToolType { return ToolTypeSystem }

func (t *MakeDirTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Success: false, Error: "path is required", Duration: time.Since(start)}, nil
	}
	
	err := createDirectory(path)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Successfully created directory: %s", path),
		Duration: time.Since(start),
	}, nil
}

type RemoveDirTool struct{}

func (t *RemoveDirTool) Name() string { return "remove_dir" }
func (t *RemoveDirTool) Description() string { return "Remove a directory and all its contents recursively" }
func (t *RemoveDirTool) Type() ToolType { return ToolTypeSystem }

func (t *RemoveDirTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return &ToolResult{Success: false, Error: "path is required", Duration: time.Since(start)}, nil
	}
	
	err := removeDirectory(path)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   fmt.Sprintf("Successfully removed directory: %s", path),
		Duration: time.Since(start),
	}, nil
}

// File operations helpers
func readFileChunks(path string, offset, limit int) (string, error) {
	data, err := os.ReadFile(filepath.Join(config.WorkingDir(), path))
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	lines := strings.Split(string(data), "\n")
	startIdx := offset - 1
	if startIdx < 0 {
		startIdx = 0
	}
	
	endIdx := startIdx + limit
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	
	return strings.Join(lines[startIdx:endIdx], "\n"), nil
}

func writeToFile(path, content string) error {
	fullPath := filepath.Join(config.WorkingDir(), path)
	
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

func replaceInFile(path, oldText, newText string) (string, error) {
	fullPath := filepath.Join(config.WorkingDir(), path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	content := string(data)
	idx := strings.Index(content, oldText)
	if idx == -1 {
		return "", fmt.Errorf("old_text not found in file")
	}
	
	newContent := content[:idx] + newText + content[idx+len(oldText):]
	
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return fmt.Sprintf("Successfully replaced '%s' with '%s'", oldText, newText), nil
}

func listFiles(pattern string) ([]string, error) {
	var files []string
	
	matches, err := filepath.Glob(filepath.Join(config.WorkingDir(), pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob: %w", err)
	}
	
	for _, match := range matches {
		relPath, _ := filepath.Rel(config.WorkingDir(), match)
		files = append(files, relPath)
	}
	
	return files, nil
}

func searchFiles(query, pattern string) (string, error) {
	// Implementation using grep or custom regex search
	cmd := exec.Command("grep", "-r", "--include=*", query, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	
	return string(output), nil
}

func runCommandWithTimeout(command string, timeout time.Duration) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = config.WorkingDir()
	output, err := cmd.CombinedOutput()
	
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return "", exitCode, fmt.Errorf("command failed: %w", err)
	}
	
	return string(output), exitCode, nil
}

func createDirectory(path string) error {
	fullPath := filepath.Join(config.WorkingDir(), path)
	return os.MkdirAll(fullPath, 0755)
}

func removeDirectory(path string) error {
	fullPath := filepath.Join(config.WorkingDir(), path)
	return os.RemoveAll(fullPath)
}
