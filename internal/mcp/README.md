# YOLO MCP Server

MCP (Model Context Protocol) implementation for YOLO, enabling AI assistants to use YOLO's full capabilities.

## What is MCP?

The Model Context Protocol is an open standard introduced by Anthropic in November 2024 that allows AI systems like LLMs to integrate and share data with external tools, systems, and data sources through a universal interface.

## Features

This implementation provides:
- **Tools**: Execute YOLO commands (build, run tests, lint, analyze code)
- **Resources**: Access project files and configuration
- **Prompts**: Pre-defined prompt templates for common tasks
- **Stdio Transport**: Standard input/output communication
- **SSE Transport**: Server-Sent Events for real-time updates

## Usage

### As a standalone MCP server:
```bash
yolo mcp
```

### Integration with Claude Desktop:
Add to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "yolo": {
      "command": "yolo",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

## Available Tools

1. **build** - Build the project with specified flags
2. **test** - Run tests with coverage and profiling
3. **lint** - Lint Go code with static analysis
4. **fmt** - Format Go code
5. **readFile** - Read file contents securely
6. **listFiles** - List files in directory (recursive)
7. **searchText** - Search text in files
8. **analyzeProject** - Analyze project structure and dependencies
9. **generateCode** - Generate boilerplate code
10. **explainError** - Explain compiler/test errors

## MCP Core Concepts

### Tools
Executable functions that can be called by AI assistants with parameters.

### Resources
URI-addressable data sources (files, configs, etc.) that can be read.

### Prompts
Pre-defined prompt templates for common tasks.

### Images
Support for returning image data (e.g., test results, diagrams).

## Protocol Specification

This implementation follows the [Model Context Protocol specification](https://modelcontextprotocol.io/specification/latest) and implements the latest JSON-RPC based protocol.
