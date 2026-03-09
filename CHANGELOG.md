# Changelog

All notable changes to YOLO will be documented in this file.

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
