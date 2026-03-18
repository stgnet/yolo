[
  {
    "title": "Security: Add command allowlist and sandboxing to run_command tool",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "CRITICAL"
  },
  {
    "title": "Security: Fix symlink bypass vulnerability in safePath function",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Security: Add defense against email prompt injection attacks",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Security: Add authentication layer for Ollama API access",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Fix data race: Add mutex to OllamaClient.ctxCache",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Fix race condition: Synchronize globalUI and bufferUI access across goroutines",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Fix race condition: Add mutex lock to HistoryManager.Load() method",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Fix race condition: Proper ordering in handoffRemainingTools (Done channel vs Results write)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Fix goroutine leak: Implement proper cleanup in executeWithTimeout after timeout",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Dead code: Delete session.go (260 lines of unused code)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Dead code: Remove checkEmailCooldown() and recordEmailSent() no-op stubs",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Dead code: Remove _SourceCodeLocation and _UseRestartTool unreferenced constants",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Dead code: Remove obsolete HistoryManager.GetModel() and SetModel() methods",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Config: Make email default recipient configurable via environment variable",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Config: Make email sender address and sendmail path configurable",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Config: Make Maildir inbox path configurable via YOLO_MAILDIR_PATH env var",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Config: Use YoloConfig model name instead of hardcoded qwen3.5:27b in email generator",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Config: Add YOLO_AUTONOMOUS_TIMEOUT env var with configurable timeout (default 5min)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Config: Remove hardcoded Mac platform assumption from email system prompt",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Config: Replace magic numbers (truncation limits, timeouts) with named configurable constants",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Email: Replace hand-rolled parseEmail with Go's net/mail package for proper MIME/multipart support",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "CRITICAL"
  },
  {
    "title": "Email: Add review/confirmation step before sending autonomous emails with sent-mail archive",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Email: Add confirmation before deleting email after auto-response (currently destructive)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Email: Make todo list attachment optional in send_report tool (currently always included)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Tests: Add symlink traversal test cases for safePath function (currently untested vulnerability)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Tests: Add concurrent sub-agent file access integration test with -race flag verification",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Tests: Add terminal escape injection attack tests for sanitizeOutput function",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Tests: Add MIME/multipart/base64 encoded email parsing test cases",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Tests: Test autonomous mode under various conditions (interrupts, hallucinations, network failures)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Tests: Test tool timeout goroutine cleanup and resource leak detection",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Tests: Add full agent loop integration test with mock Ollama HTTP server",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Tests: Add tests for invalid/malformed JSON from LLM to verify error handling and recovery",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Refactor: Split YoloAgent god object into separate types (LLMClient, ToolExecutor, UIOutput, InputHandler)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Refactor: Extract shared tool execution code from agent loops into reusable functions",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Refactor: Standardize error handling - use errors package instead of string returns with 'Error:' prefix",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Refactor: Reduce HTTP client timeout from 300s to 60s with optional per-request timeout extension",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Refactor: Fix retry logic to reuse HTTP client instead of creating new one on every call",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Performance: Optimize sanitizeOutput from byte-by-byte to bulk string/regex operations",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Docs: Remove machine-specific paths (/Users/sgriepentrog/src/yolo) from all documentation",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Docs: Remove documentation of non-existent CLI flags (--autonomous, --version)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Docs: Fix security checklist to only include relevant items (remove meaningless SQL injection claim)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Docs: Replace hardcoded 'Last autonomous check: Just now' with dynamic status tracking or remove claim",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "LOW"
  },
  {
    "title": "Security: Add audit log for all autonomous operations (run_command, write_file, remove_dir, email sending)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Security: Add user confirmation required for dangerous autonomous operations (run_command, remove_dir)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  },
  {
    "title": "Wire up concurrency package (group.go, pool.go, limiter.go) for sub-agent management instead of bare go func() calls",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Split monolithic main package into separate packages with clean interfaces (OllamaClient, ToolExecutor, InputManager, TerminalUI)",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "MEDIUM"
  },
  {
    "title": "Use errors package custom error types consistently across all tools instead of string returns",
    "created_at": "2026-03-14T00:00:00Z",
    "done": false,
    "completed_at": null,
    "priority": "HIGH"
  }
]
