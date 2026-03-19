package config_test

import (
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
