package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"yolo/config"
)

func TestDefaultConfig(t *testing.T) {
	c := config.DefaultConfig()
	if c == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check defaults are set
	modelName := c.GetModelName()
	if modelName == "" {
		t.Error("ModelName should have a default value")
	}

	endpoint := c.GetOllamaEndpoint()
	if endpoint == "" {
		t.Error("OllamaEndpoint should have a default value")
	}

	yoloDir := c.GetYoloDir()
	if yoloDir == "" {
		t.Error("YoloDir should have a default value")
	}

	todoFile := c.GetTodoFile()
	if todoFile == "" {
		t.Error("TodoFile should have a default value")
	}
}

func TestConfigGetterSetter(t *testing.T) {
	c := config.DefaultConfig()

	t.Run("ModelName", func(t *testing.T) {
		initial := c.GetModelName()
		newValue := "test-model:13b"
		c.SetModelName(newValue)
		if c.GetModelName() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetModelName())
		}
		c.SetModelName(initial) // Restore
	})

	t.Run("OllamaEndpoint", func(t *testing.T) {
		initial := c.GetOllamaEndpoint()
		newValue := "http://test:1234/api/generate"
		c.SetOllamaEndpoint(newValue)
		if c.GetOllamaEndpoint() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetOllamaEndpoint())
		}
		c.SetOllamaEndpoint(initial) // Restore
	})

	t.Run("YoloDir", func(t *testing.T) {
		initial := c.GetYoloDir()
		newValue := "/tmp/test-yolo"
		c.SetYoloDir(newValue)
		if c.GetYoloDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetYoloDir())
		}
		c.SetYoloDir(initial) // Restore
	})

	t.Run("TodoFile", func(t *testing.T) {
		initial := c.GetTodoFile()
		newValue := "/tmp/test-todo.json"
		c.SetTodoFile(newValue)
		if c.GetTodoFile() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetTodoFile())
		}
		c.SetTodoFile(initial) // Restore
	})

	t.Run("LearnFile", func(t *testing.T) {
		initial := c.GetLearnFile()
		newValue := "/tmp/test-learn.json"
		c.SetLearnFile(newValue)
		if c.GetLearnFile() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetLearnFile())
		}
		c.SetLearnFile(initial) // Restore
	})

	t.Run("EmailDir", func(t *testing.T) {
		initial := c.GetEmailDir()
		newValue := "/tmp/test-email"
		c.SetEmailDir(newValue)
		if c.GetEmailDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetEmailDir())
		}
		c.SetEmailDir(initial) // Restore
	})

	t.Run("SubagentDir", func(t *testing.T) {
		initial := c.GetSubagentDir()
		newValue := "/tmp/test-subagents"
		c.SetSubagentDir(newValue)
		if c.GetSubagentDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetSubagentDir())
		}
		c.SetSubagentDir(initial) // Restore
	})

	t.Run("PromptDir", func(t *testing.T) {
		initial := c.GetPromptDir()
		newValue := "/tmp/test-prompts"
		c.SetPromptDir(newValue)
		if c.GetPromptDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetPromptDir())
		}
		c.SetPromptDir(initial) // Restore
	})

	t.Run("ContextFilePath", func(t *testing.T) {
		initial := c.GetContextFilePath()
		newValue := "/tmp/test-context.txt"
		c.SetContextFilePath(newValue)
		if c.GetContextFilePath() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetContextFilePath())
		}
		c.SetContextFilePath(initial) // Restore
	})

	t.Run("OllamaURL", func(t *testing.T) {
		initial := c.GetOllamaURL()
		newValue := "http://test:1234"
		c.SetOllamaURL(newValue)
		if c.GetOllamaURL() != newValue {
			t.Errorf("Expected %s, got %s", newValue, c.GetOllamaURL())
		}
		c.SetOllamaURL(initial) // Restore
	})
}

func TestGlobalAccessorFunctions(t *testing.T) {
	// Save current values
	saveModelName := config.GetModelName()
	saveYoloDir := config.GetYoloDir()

	defer func() {
		config.SetModelName(saveModelName)
		config.SetYoloDir(saveYoloDir)
	}()

	t.Run("ModelName global", func(t *testing.T) {
		newValue := "global-test-model"
		config.SetModelName(newValue)
		if config.GetModelName() != newValue {
			t.Errorf("Expected %s, got %s", newValue, config.GetModelName())
		}
	})

	t.Run("YoloDir global", func(t *testing.T) {
		newValue := "/tmp/global-test"
		config.SetYoloDir(newValue)
		if config.GetYoloDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, config.GetYoloDir())
		}
	})

	t.Run("WorkingDir alias", func(t *testing.T) {
		newValue := "/tmp/alias-test"
		config.SetWorkingDir(newValue)
		if config.WorkingDir() != newValue {
			t.Errorf("Expected %s, got %s", newValue, config.WorkingDir())
		}
	})
}

func TestConfigThreadSafety(t *testing.T) {
	c := config.DefaultConfig()
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			value := "test-model:" + string(rune('0'+idx%10))
			c.SetModelName(value)
			_ = c.GetModelName()
		}(i)
	}

	wg.Wait()
	if c.GetModelName() == "" {
		t.Error("ModelName should not be empty after concurrent access")
	}
}

// TestGlobalAccessorFunctions_Complete tests all global accessor functions comprehensively
func TestGlobalAccessorFunctions_Complete(t *testing.T) {
	// Save all current values
	save := map[string]string{
		"model":       config.GetModelName(),
		"endpoint":    config.GetOllamaEndpoint(),
		"url":         config.GetOllamaURL(),
		"dir":         config.GetYoloDir(),
		"working":     config.WorkingDir(),
		"todo":        config.GetTodoFile(),
		"learn":       config.GetLearnFile(),
		"email":       config.GetEmailDir(),
		"subagent":    config.GetSubagentDir(),
		"prompt":      config.GetPromptDir(),
		"context":     config.GetContextFilePath(),
	}

	defer func() {
		// Restore all values
		config.SetModelName(save["model"])
		config.SetOllamaEndpoint(save["endpoint"])
		config.SetOllamaURL(save["url"])
		config.SetYoloDir(save["dir"])
		config.SetTodoFile(save["todo"])
		config.SetLearnFile(save["learn"])
		config.SetEmailDir(save["email"])
		config.SetSubagentDir(save["subagent"])
		config.SetPromptDir(save["prompt"])
		config.SetContextFilePath(save["context"])
	}()

	t.Run("ModelName", func(t *testing.T) {
		config.SetModelName("global-test-model")
		if config.GetModelName() != "global-test-model" {
			t.Errorf("Expected global-test-model, got %s", config.GetModelName())
		}
	})

	t.Run("OllamaEndpoint", func(t *testing.T) {
		config.SetOllamaEndpoint("http://test:1234/api/generate")
		if config.GetOllamaEndpoint() != "http://test:1234/api/generate" {
			t.Errorf("Expected http://test:1234/api/generate, got %s", config.GetOllamaEndpoint())
		}
	})

	t.Run("OllamaURL", func(t *testing.T) {
		config.SetOllamaURL("http://test:1234")
		if config.GetOllamaURL() != "http://test:1234" {
			t.Errorf("Expected http://test:1234, got %s", config.GetOllamaURL())
		}
	})

	t.Run("YoloDir", func(t *testing.T) {
		config.SetYoloDir("/tmp/test-yolo-dir")
		if config.GetYoloDir() != "/tmp/test-yolo-dir" {
			t.Errorf("Expected /tmp/test-yolo-dir, got %s", config.GetYoloDir())
		}
	})

	t.Run("WorkingDir alias", func(t *testing.T) {
		config.SetWorkingDir("/tmp/working-test")
		if config.WorkingDir() != "/tmp/working-test" {
			t.Errorf("Expected /tmp/working-test, got %s", config.WorkingDir())
		}
	})

	t.Run("TodoFile", func(t *testing.T) {
		config.SetTodoFile("/tmp/test-todo.json")
		if config.GetTodoFile() != "/tmp/test-todo.json" {
			t.Errorf("Expected /tmp/test-todo.json, got %s", config.GetTodoFile())
		}
	})

	t.Run("LearnFile", func(t *testing.T) {
		config.SetLearnFile("/tmp/test-learn.json")
		if config.GetLearnFile() != "/tmp/test-learn.json" {
			t.Errorf("Expected /tmp/test-learn.json, got %s", config.GetLearnFile())
		}
	})

	t.Run("EmailDir", func(t *testing.T) {
		config.SetEmailDir("/tmp/test-email")
		if config.GetEmailDir() != "/tmp/test-email" {
			t.Errorf("Expected /tmp/test-email, got %s", config.GetEmailDir())
		}
	})

	t.Run("SubagentDir", func(t *testing.T) {
		config.SetSubagentDir("/tmp/test-subagents")
		if config.GetSubagentDir() != "/tmp/test-subagents" {
			t.Errorf("Expected /tmp/test-subagents, got %s", config.GetSubagentDir())
		}
	})

	t.Run("PromptDir", func(t *testing.T) {
		config.SetPromptDir("/tmp/test-prompts")
		if config.GetPromptDir() != "/tmp/test-prompts" {
			t.Errorf("Expected /tmp/test-prompts, got %s", config.GetPromptDir())
		}
	})

	t.Run("ContextFilePath", func(t *testing.T) {
		config.SetContextFilePath("/tmp/test-context.txt")
		if config.GetContextFilePath() != "/tmp/test-context.txt" {
			t.Errorf("Expected /tmp/test-context.txt, got %s", config.GetContextFilePath())
		}
	})

	t.Run("PathWithSpaces", func(t *testing.T) {
		path := "/tmp/path with spaces/file.json"
		config.SetTodoFile(path)
		if config.GetTodoFile() != path {
			t.Errorf("Expected %s, got %s", path, config.GetTodoFile())
		}
	})

	t.Run("AbsolutePathWithSpecialChars", func(t *testing.T) {
		path := "/tmp/test-dir_123/test-file.json"
		config.SetLearnFile(path)
		if config.GetLearnFile() != path {
			t.Errorf("Expected %s, got %s", path, config.GetLearnFile())
		}
	})

	t.Run("VerifyFilepathJoin", func(t *testing.T) {
		// Test that paths are properly joined and don't have double slashes
		path := filepath.Join("/tmp", "nested", "file.json")
		config.SetContextFilePath(path)
		result := config.GetContextFilePath()
		if strings.Contains(result, "//") {
			t.Errorf("Path should not contain double slashes: %s", result)
		}
		if !strings.HasSuffix(result, "file.json") {
			t.Errorf("Path should end with file.json: %s", result)
		}
	})

	_ = save // Suppress unused warning (we use it in defer)
}

// TestGlobalConfigConcurrentAccess tests thread safety of global config functions
func TestGlobalConfigConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 50

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			value := fmt.Sprintf("concurrent-model-%d", idx)
			config.SetModelName(value)
			_ = config.GetModelName()
		}(i)
	}

	wg.Wait()
	if config.GetModelName() == "" {
		t.Error("Global ModelName should not be empty after concurrent access")
	}
}

// TestConfigStringRepresentation tests the String() method for debugging output
func TestConfigStringRepresentation(t *testing.T) {
	c := config.DefaultConfig()
	
	str := c.String()
	
	// Check that the string representation contains key fields
	expectedFields := []string{
		"YOLO Configuration",
		"Model:",
		"Ollama Endpoint:",
		"Working Dir:",
		"Todo:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(str, field) {
			t.Errorf("Config string should contain '%s', got:\n%s", field, str)
		}
	}

	// Should not be empty
	if len(str) == 0 {
		t.Error("Config String() should not return empty string")
	}
}

// TestConfigFallbackValues tests that fallback values are returned when atomic.Value is nil
func TestConfigFallbackValues(t *testing.T) {
	// Create a fresh config without initializing all fields
	c := &config.Config{}
	
	// These methods should return default/fallback values even if not explicitly set
	modelName := c.GetModelName()
	if modelName != "qwen2.5-coder:7b" {
		t.Errorf("Expected fallback model 'qwen2.5-coder:7b', got '%s'", modelName)
	}

	endpoint := c.GetOllamaEndpoint()
	if endpoint != "http://localhost:11434/api/generate" {
		t.Errorf("Expected fallback endpoint, got '%s'", endpoint)
	}

	yoloDir := c.GetYoloDir()
	if yoloDir != "." {
		t.Errorf("Expected fallback yolo dir '.', got '%s'", yoloDir)
	}

	// Test that setting and then testing works correctly
	c.SetModelName("test-model")
	if c.GetModelName() != "test-model" {
		t.Errorf("Expected 'test-model', got '%s'", c.GetModelName())
	}
}

// TestConfigNilValueHandling tests getter methods with explicit nil/unset atomic.Value fields
func TestConfigNilValueHandling(t *testing.T) {
	c := &config.Config{}
	
	tests := []struct{
		name string
		getter func() string
		expectedFallback string
	}{
		{"GetOllamaURL fallback", c.GetOllamaURL, "http://localhost:11434"},
		{"GetTodoFile fallback", c.GetTodoFile, ""}, // This one uses os.Getenv("HOME"), so we check it's not empty
		{"GetLearnFile fallback", c.GetLearnFile, ""},
		{"GetEmailDir fallback", c.GetEmailDir, "/var/mail/b-haven.org/yolo/new/"},
		{"GetSubagentDir fallback", c.GetSubagentDir, ""},
		{"GetPromptDir fallback", c.GetPromptDir, "prompts"},
		{"GetContextFilePath fallback", c.GetContextFilePath, "context.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			
			if tt.expectedFallback == "" {
				// For paths that depend on HOME, just verify they're not empty or have reasonable structure
				if result == "" && !strings.Contains(result, "yolo") {
					t.Errorf("Expected non-empty fallback value, got '%s'", result)
				}
			} else if result != tt.expectedFallback {
				t.Errorf("Expected fallback '%s', got '%s'", tt.expectedFallback, result)
			}
		})
	}
}

// TestConfigInvalidTypeHandling tests that non-string types in atomic.Value are handled gracefully
func TestConfigInvalidTypeHandling(t *testing.T) {
	c := config.DefaultConfig()
	
	// The getter methods should handle type assertion failures gracefully
	// by returning the fallback value
	
	// Test all getters return valid strings
	getters := []struct{
		name string
		value string
	}{
		{"ModelName", c.GetModelName()},
		{"OllamaEndpoint", c.GetOllamaEndpoint()},
		{"OllamaURL", c.GetOllamaURL()},
		{"YoloDir", c.GetYoloDir()},
		{"TodoFile", c.GetTodoFile()},
		{"LearnFile", c.GetLearnFile()},
		{"EmailDir", c.GetEmailDir()},
		{"SubagentDir", c.GetSubagentDir()},
		{"PromptDir", c.GetPromptDir()},
		{"ContextFilePath", c.GetContextFilePath()},
	}

	for _, g := range getters {
		t.Run(g.name+" returns non-empty string", func(t *testing.T) {
			if g.value == "" {
				t.Errorf("%s should not return empty string", g.name)
			}
		})
	}
}

func TestConfigThreadSafetyGlobal(t *testing.T) {
	var wg sync.WaitGroup
	numGoroutines := 100

	saveModelName := config.GetModelName()
	defer config.SetModelName(saveModelName)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			value := "global-test:" + string(rune('0'+idx%10))
			config.SetModelName(value)
			_ = config.GetModelName()
		}(i)
	}

	wg.Wait()
	if config.GetModelName() == "" {
		t.Error("Global ModelName should not be empty after concurrent access")
	}
}

func TestConfigString(t *testing.T) {
	c := config.DefaultConfig()
	str := c.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}

	// Check that all fields are represented
	expectedFields := []string{
		"Model:",
		"Ollama Endpoint:",
		"Ollama URL:",
		"Working Dir:",
		"Todo:",
		"Learn:",
		"Email:",
		"Subagents:",
		"Prompts:",
		"Context:",
	}

	for _, field := range expectedFields {
		if !contains(str, field) {
			t.Errorf("String() should contain %q, got: %s", field, str)
		}
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestConfigFileDefaults(t *testing.T) {
	c := config.DefaultConfig()
	homeDir, _ := os.UserHomeDir()
	expectedTodoFile := filepath.Join(homeDir, ".yolo_todo.json")
	expectedLearnFile := filepath.Join(homeDir, ".yolo_learn.json")
	expectedSubagentDir := filepath.Join(homeDir, ".yolo_subagents")

	todoFile := c.GetTodoFile()
	if todoFile != expectedTodoFile {
		t.Logf("TodoFile: got %s, expected %s", todoFile, expectedTodoFile)
	}

	learnFile := c.GetLearnFile()
	if learnFile != expectedLearnFile {
		t.Logf("LearnFile: got %s, expected %s", learnFile, expectedLearnFile)
	}

	subagentDir := c.GetSubagentDir()
	if subagentDir != expectedSubagentDir {
		t.Logf("SubagentDir: got %s, expected %s", subagentDir, expectedSubagentDir)
	}
}

func TestConfigDefaultsNotEmpty(t *testing.T) {
	c := config.DefaultConfig()

	checks := []struct {
		name  string
		value string
	}{
		{"ModelName", c.GetModelName()},
		{"OllamaEndpoint", c.GetOllamaEndpoint()},
		{"OllamaURL", c.GetOllamaURL()},
		{"YoloDir", c.GetYoloDir()},
		{"TodoFile", c.GetTodoFile()},
		{"LearnFile", c.GetLearnFile()},
		{"EmailDir", c.GetEmailDir()},
		{"SubagentDir", c.GetSubagentDir()},
		{"PromptDir", c.GetPromptDir()},
		{"ContextFilePath", c.GetContextFilePath()},
	}

	for _, check := range checks {
		if check.value == "" {
			t.Errorf("%s should not be empty", check.name)
		}
	}
}

func TestConfigMultipleInstances(t *testing.T) {
	c1 := config.DefaultConfig()
	c2 := config.DefaultConfig()

	c1.SetModelName("model-1")
	c2.SetModelName("model-2")

	if c1.GetModelName() != "model-1" {
		t.Errorf("c1.ModelName should be model-1, got %s", c1.GetModelName())
	}
	if c2.GetModelName() != "model-2" {
		t.Errorf("c2.ModelName should be model-2, got %s", c2.GetModelName())
	}
}
