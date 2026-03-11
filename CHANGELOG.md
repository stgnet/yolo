# Changelog

All notable changes to YOLO will be documented in this file.

## [Unreleased] - 2026-03-10

### Added
- **Multi-line input**: The input area now expands upward to show the full
  message as it's being typed, with word wrapping instead of horizontal
  scrolling. The scroll region, divider, and input area resize dynamically.
- **Email sending tools** (`tools_email.go`): YOLO can now send automated progress reports:
  - `send_report`: Send a progress report email to scott@stg.net
  - `send_email`: Send custom emails with subject, body, and recipient
  - Both use SMTP from yolo@b-haven.org (requires EMAIL_PASSWORD env var)
- **Email integration tests** (`email/email_integration_test.go`): End-to-end tests for email sending functionality that skip safely when EMAIL_PASSWORD is not configured
- **Autonomous learning system** (`learning.go`, `tools_learning.go`): Self-improvement through internet research:
  - `learn`: Autonomously discovers improvements from web and Reddit sources
  - Tracks improvement history in `.yolo_learning.json` with categorization and trending
  - Categorizes findings by priority (1-5), category, and status

### Fixed
- **LimitedConcurrency deadlock**: Race condition when multiple jobs executed
  concurrently without proper synchronization. Changed `wg.Add()` to run before
  goroutine creation in worker loop, and moved semaphore acquisition outside the
  job execution to prevent deadlocks with concurrent job completion.
- **Barrier race conditions**: Refactored Barrier to use atomic operations (`int32` counter) instead of channel array for better thread safety. Simplified from 47 lines to 38 lines with improved performance and clearer code. Test race conditions fixed in `TestLimiterGroup` by properly synchronizing `wg.Done()` calls.
- **Visible queued messages**: Messages typed while the agent is busy are
  displayed as `[queued] text` between the divider and input prompt. They
  remain visible until processed, making it clear what's pending.
- **Agent nudge on queued input**: During tool-calling loops, if the user
  has queued a message, the agent is nudged via a system message to wrap up
  and process it.
- **GOG Google Integration** (`tools_gog.go`): Full OAuth2-enabled access to Google services:
  - Gmail: search, list, send emails with filters (inbox:unread, newer_than:1d)
  - Calendar: list events, create events with titles/descriptions/times
  - Drive: list files/folders, upload/download content
- **GOG documentation** (`GOOGLE_INTEGRATION.md`): Setup instructions, OAuth2 configuration, and usage examples
- **GOG tests** (`gog_test.go`): Integration tests verifying Gmail, Calendar, and Drive functionality
- **Enhanced web_search**: Now uses DuckDuckGo's Instant Answer API with proper Wikipedia fallback for better search results

### Fixed
- **web_search DuckDuckGo empty result detection**: `isEmptySearchResult()` now properly detects when DuckDuckGo returns "Try a different search term" messages, allowing correct fallback to Wikipedia. This fixes queries like "golang dependency injection" that previously returned unhelpful empty responses.
- **Output line overwrite glitch**: `rawWrite()` was converting standalone
  `\r` (carriage return) to `\r\n`, causing the cursor position tracker to
  drift from the actual terminal cursor. This made output sometimes write
  over the same line twice. Standalone `\r` is now preserved as-is.
- **GOG tool example**: Corrected `drive list` → `drive ls` in error message

### Changed
- **Test coverage**: Added GOG integration tests with realistic golden files for calendar and Drive operations
- **Test coverage**: Enhanced web_search tests with multiple DuckDuckGo query scenarios and isEmptySearchResult validation cases
- **Documentation**: Updated tools.md to reflect DuckDuckGo + Wikipedia search implementation

### Removed
- **Spinner**: The animated "thinking..." spinner has been removed. It was
  the primary source of `\r`-based cursor positioning issues.

## [Unreleased] - 2026-03-10 (earlier)

### Added
- **ARCHITECTURE.md**: Comprehensive system design document with component
  diagrams, data-flow descriptions, and file layout reference.
- **CONTRIBUTING.md**: Development workflow, code style, testing guide, and
  instructions for adding new tools.
- **Expanded test coverage**: ~100 new test cases across 6 new test files
  covering HistoryManager, ToolExecutor, TerminalUI, InputManager, and agent
  tool-call parsing.
- **UTF-8 terminal input**: Multi-byte characters (accented letters, CJK,
  emoji) are now correctly assembled from the raw byte stream.

### Fixed
- **Slice mutation bug**: `append(baseMsgs, roundMsgs...)` in the chat loop
  could corrupt `baseMsgs` across iterations; replaced with explicit copy.
- **Silent save failures**: `HistoryManager.Save()` now returns errors
  instead of silently ignoring `os.MkdirAll`, `os.WriteFile`, and
  `os.Rename` failures. Callers log a warning.

### Removed
- **Dead code**: `MessageHistory`, `MessageHistoryItem`, `Color` enum,
  `escapeMarkdown`, `LoadMessageHistory`, and `NewMessageHistory` — all
  unused legacy code.
- **Auto .gitignore**: `make_dir` no longer silently creates a `.gitignore`
  with `*` in every new directory.
- **Unused MCP packages**: `internal/mcp/` and `internal/mcpclient/` — not
  imported by any code in the main module.
- **Stale docs**: `MIGRATION_SUMMARY.md` (completed migration), `TOOLS.md`
  (outdated tool names).
- **Misc**: `send_email.sh` (unrelated script), `test_move_source.txt`
  (leftover test fixture).

### Changed
- **Improved documentation**: Added godoc comments to all exported types,
  functions, and constants across `agent.go`, `ollama.go`, `tools.go`,
  `history.go`, `config.go`. Updated README.md to reflect current features
  and project state.

## [Unreleased] - 2026-03-09

### Added
- **Edge Case Tests** (#15): Comprehensive test coverage for all file manipulation and search tools:
  - read_file: invalid offset/limit, negative values, empty files, non-existent paths
  - write_file: invalid paths, empty content, special characters, reserved names
  - edit_file: no match found, multiple matches, empty operations, case sensitivity
  - list_files: invalid patterns, empty results, non-existent directories
  - search_files: invalid regex patterns, empty search terms, non-existent directories

### Fixed
- **Format 5 Regex Parsing** (#12): Removed incorrect Perl lookahead syntax in tool block regex pattern
- **Bracket Format Tool Calls** (#13): Enabled parameters for bracket-format `[[tool name param=value]]` calls in Format 5 parsing
- **setLevel Tool Response** (#14): Fixed setLevel to return nil result instead of malformed response

### Changed
- **Code Formatting**: Ran `gofmt` to ensure consistent code formatting across all files

### Security
- Added comprehensive file validation for all file manipulation tools (paths, special characters, reserved names)
- Implemented proper error handling for edge cases in file operations

## [Unreleased] - Previous Work

### Added
- **MCP (Model Context Protocol) Support** (#1): Full implementation of MCP protocol including:
  - Standalone MCP server for external tool hosting (`mcpserver`)
  - Native LLM-powered tools via MCP clients (`mcpclient` package)
  - Tool registry system for dynamic tool discovery and loading
  - JSON-RPC 2.0 compliant request/response handling

### Changed
- **Code Formatting**: Applied consistent formatting standards across codebase

## [0.1.0] - 2024-03-09

### Initial Release
- Core YOLO agent functionality with autonomous thinking cycles
- Tool execution system (read_file, write_file, edit_file, list_files, search_files, run_command, create_subagent)
- Terminal UI with ANSI color support and formatted output
- Context management with history tracking
- Subagent creation and management

## [0.0.1] - 2024-03-08

### Initial Commit
- Project structure setup
- README documentation
- Basic configuration constants
