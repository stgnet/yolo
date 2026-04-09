# Changelog

All notable changes to the YOLO project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to semantic versioning.

## [Unreleased]

### Added

#### Victron Energy BLE Support
- Complete implementation of Victron Energy device support via Bluetooth Low Energy
- Support for SmartSolar MPPT charge controllers and SmartShunt battery monitors
- Platform-specific initialization files for different operating systems:
  - `victron/bluez_init_linux.go` - Linux BlueZ backend initialization
  - `victron/bluez_init_macos.go` - macOS BLE backend initialization
  - `victron/bluez/backend.go` - Linux BlueZ implementation (9.9KB)
  - `victron/macos/backend.go` - macOS BLE implementation (5.8KB)
- Mock backend for testing without hardware (`victron/mock_backend.go`)
- Comprehensive client implementation with connection management, subscription support, and parsing

#### Testing Improvements
- Added comprehensive unit tests for Victron client functions (`victron/client_test.go`, 11.2KB)
- Added comprehensive unit tests for email package functions (`email/email_coverage_test.go`, 8.3KB)
- Added comprehensive tests for todo management functions (`tools_todo_test.go`, 7.6KB)
- Added integration tests for Victron backend (`victron/integration_test.go`, 7.0KB)
- Made Scan and Discover tests resilient to missing BLE hardware (no crashes on machines without Bluetooth)

#### Documentation Enhancements
- **TESTING.md** (3.8KB): Comprehensive testing guide with best practices
  - How to run tests in development environments
  - Guidelines for writing safe, non-destructive tests
  - CI/CD integration recommendations
- **TEST_AUDIT_REPORT.md** (7.5KB): Detailed test audit findings
  - Identified tests that cause side effects
  - Listed files created by tests
  - Recommended improvements for test safety
- **TEST_SAFETY_AUDIT.md** (5.5KB): Safety audit with actionable recommendations
  - Tests creating temporary directories and files in working directory
  - Tests performing actual git operations and network requests
  - Clear categorization of safe vs unsafe tests
- **TEST_IMPROVEMENTS_SUMMARY.md** (3.5KB): Summary of test improvements made
  - Tests now use isolated temp directories for file operations
  - File creation tests check existence rather than causing side effects
  - Network tests handle missing hardware gracefully
  - All tests follow unit test best practices

### Security Enhancements
- Added input validation for playwright_mcp tool parameters
- Improved sanitization of user inputs in browser automation functions
- Enhanced security measures in email processing and inbox handling

### Performance Improvements
- Optimized file search operations with better glob pattern matching
- Improved memory management in subagent spawning and result retrieval
- Better error handling in Victron BLE connection lifecycle

## [1.0.0] - 2026-03-18

### Major Features
- Self-evolving AI agent architecture
- Autonomous mode for independent operation
- Email processing with auto-responses
- Web connectivity (DuckDuckGo, Reddit, Google Workspace)
- Browser automation via Playwright MCP
- Complete Git integration with 9 tools
- Task management system with todos
- Memory and history tracking systems

### Core Components
- **YoloAgent**: Central orchestrator for chat loops and command handling
- **OllamaClient**: HTTP client for Ollama REST API with streaming support
- **ToolExecutor**: Dispatcher for 41 concrete tool implementations
- **HistoryManager**: Thread-safe persistence in `.yolo/history.json`
- **InputManager**: Raw terminal input handling in separate goroutine
- **TerminalUI**: Split-screen layout with scrollable output

### Tool Categories (41 total)
- File Operations: read, write, edit, list, search, copy, move, create/delete directories
- Agent Management: spawn/list/read subagents, think, restart, check Ollama status
- External Services: web search, webpage reading, Reddit API, Google Workspace integration
- Email Processing: send emails, check inbox, auto-respond and delete
- Task Management: add, complete, delete, list todos
- Version Control: full Git CLI integration (status, diff, log, branch, checkout, commit, add, show, remote)
- System Commands: shell command execution with timeout
- Browser Automation: Navigate URLs, interact with DOM, fill forms, take screenshots
- Model Management: List and switch Ollama models

### Configuration
- Environment variable support (OLLAMA_HOST, YOLO_MODEL, YOLO_NUM_CTX)
- JSON configuration file (config.json)
- Terminal mode settings for interactive vs autonomous operation

---

## Version History

### Current Development Status
**Branch**: `main`  
**Commits ahead of origin/main**: 8  
**Working Tree**: Clean

### Recent Commits (Latest to Oldest)

1. **docs: Add summary of victron test improvements** (5c09ecf)
   - Documented Victron BLE test safety improvements
   
2. **test(victron): Make Scan and Discover tests resilient to missing BLE hardware** (cde0818)
   - Tests now handle absence of Bluetooth gracefully
   - No crashes on development machines without BLE hardware

3. **Refactor Victron BLE backend initialization with platform-specific init functions** (f9ec431)
   - Separated Linux BlueZ and macOS BLE initializations
   - Added platform-specific init files for cleaner architecture

4. **Merge Victron backend improvements and add platform initialization files** (fe4108c)
   - Integrated all Victron backend changes
   - Added bluez_init_linux.go and bluez_init_macos.go

5. **Add comprehensive tests for todo management functions** (dbca8de)
   - Full test coverage for add, complete, delete, list operations

6. **Add comprehensive unit tests for Victron client functions** (ac4230b)
   - 11.2KB of tests covering connection lifecycle and data parsing

7. **test: Add comprehensive tests for email package functions** (7d05dab)
   - 8.3KB of coverage tests for email creation, formatting, parsing

8. **Implement Bluetooth LE support for Victron Energy devices** (673995f)
   - Complete implementation with scan, connect, disconnect, get_values, subscribe actions
   - Support for SmartSolar MPPT and SmartShunt devices

---

## Breaking Changes

No breaking changes in the current development cycle.

## Deprecations

None currently.

## Known Issues

- Victron BLE features require Bluetooth hardware to function fully (tests now handle this gracefully)
- Some browser automation features may require additional dependencies (Playwright/Chromium)

## Future Roadmap

- [ ] CI/CD pipeline integration for automated testing
- [ ] Performance benchmarks for critical functions
- [ ] Additional Victron device support (VE.Direct adapters)
- [ ] Enhanced subagent parallelization strategies
- [ ] Improved memory leak detection and prevention

---

*For more detailed information about specific features, see the documentation in the `DOCS/` directory.*
*For testing guidelines, see `TESTING.md`.*
