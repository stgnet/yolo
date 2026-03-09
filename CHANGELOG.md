# Changelog

All notable changes to YOLO will be documented in this file.

## [Unreleased]

### Added
- **MCP (Model Context Protocol) Support** (#1): Full implementation of MCP protocol including:
  - Standalone MCP server for external tool hosting
  - Native LLM-powered tools via MCP clients
  - Tool registry system for dynamic tool discovery
  - Comprehensive test suite with 10+ test cases

### Fixed
- **Format 5 Regex Parsing** (#2): Fixed incorrect Perl lookahead syntax in regex pattern
- **Bracket Format Tool Calls** (#3): Enabled parameters for bracket-format tool calls in Format 5 parsing
- **setLevel Tool Response** (#4): Fixed setLevel to return nil result instead of malformed response

### Changed
- **Code Formatting**: Ran `gofmt` to ensure consistent code formatting across all files

### Security
- Added comprehensive file validation for all file manipulation tools
- Implemented proper error handling for edge cases in file operations

## [0.1.0] - 2024-03-09

### Initial Release
- Core YOLO agent functionality
- Tool execution system (read_file, write_file, run_command, create_subagent)
- Autonomous thinking cycles
- Subagent creation and management
- Terminal UI with ANSI color support
- Context management with history tracking

## [0.0.1] - 2024-03-08

### Initial Commit
- Project structure setup
- README documentation
- Basic configuration constants
