package yolo

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test")
	}

	// Create a temporary test git repo
	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	assert.NoError(t, err)

	// Configure git for the test repo
	exec.Command("git", "config", "user.email", "test@yolo.ai").Run()
	exec.Command("git", "config", "user.name", "YOLO Test").Run()

	// Create and add a file
	if err := exec.Command("bash", "-c", `echo "test content" > test.txt`).Run(); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", "test.txt").Run()

	gitExecutor := New()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "Add command", input: `{"command":"status","args":{}}`, wantErr: false},
		{name: "Diff command", input: `{"command":"diff","args":{"name":"HEAD..main"}}`, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitExecutor.Execute(map[string]interface{}{
				"command":   "git",
				"name":      tt.input,
				"args":      map[string]interface{}{},
				"resultMap": []map[string]string{{"name": tt.name}},
			})

			if (result.Error != nil) != tt.wantErr {
				t.Errorf("GitCommand.Execute() error = %v, wantErr %v", result.Error, tt.wantErr)
				return
			}

			assert.NotNil(t, result.Output)
			if !tt.wantErr {
				assert.NotEmpty(t, *result.Output)
			}
		})
	}
}
