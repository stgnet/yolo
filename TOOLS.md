# File Management Tools Reference

This document provides comprehensive documentation for the file management tools available in the yolo MCP system.

## Table of Contents

- [create_dir](#create_dir)
- [remove_dir](#remove_dir)
- [move_file](#move_file)
- [copy_file](#copy_file)
- [read_file](#read_file)
- [write_file](#write_file)
- [list_dir](#list_dir)

---

## create_dir

Creates a directory with the specified name. If parent directories don't exist, they will be created automatically.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | The name of the directory to create (relative path supported) |

### Example Usage

```json
{
  "tool": "create_dir",
  "arguments": {
    "name": "new_project/src/components"
  }
}
```

### Response

Success: Created directory at /base/path/new_project/src/components  
Error: Error creating directory 'path': permission denied

---

## remove_dir

Removes a directory and all its contents recursively. The `path` is relative to the working directory.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `path` | string | Yes | Path to the directory to remove (relative to base directory) |

### Example Usage

```json
{
  "tool": "remove_dir",
  "arguments": {
    "path": "old_project/build"
  }
}
```

### Response

Success: Removed directory at /base/path/old_project/build  
Error: Error removing 'path': no such file or directory

---

## move_file

Moves (renames) a file from source to destination. If the destination directory doesn't exist, it will be created automatically. The source must be a file, not a directory.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `source` | string | Yes | Source file path (relative to base directory) |
| `dest` | string | Yes | Destination path including new filename |

### Example Usage

```json
{
  "tool": "move_file",
  "arguments": {
    "source": "old_name.txt",
    "dest": "new_name.txt"
  }
}
```

Move to different directory:

```json
{
  "tool": "move_file",
  "arguments": {
    "source": "docs/readme.txt",
    "dest": "archive/old_readme.txt"
  }
}
```

Move with auto-creating directories:

```json
{
  "tool": "move_file",
  "arguments": {
    "source": "backup/config.json",
    "dest": "backups/2024/january/config.json"
  }
}
```

### Response

Success: Moved backup/config.json to backups/2024/january/config.json  
Error: Error getting source file info: no such file or directory  
Error: Source path 'dir_name' is a directory, not a file

---

## copy_file

Creates a copy of a file from source to destination. If the destination directory doesn't exist, it will be created automatically. The source must be a file, not a directory.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `source` | string | Yes | Source file path (relative to base directory) |
| `dest` | string | Yes | Destination path including new filename (optional) |

### Example Usage

```json
{
  "tool": "copy_file",
  "arguments": {
    "source": "original.txt",
    "dest": "backup/original_copy.txt"
  }
}
```

Copy with auto-creating directories:

```json
{
  "tool": "copy_file",
  "arguments": {
    "source": "config.json",
    "dest": "backups/2024/config_backup.json"
  }
}
```

### Response

Success: Copied original.txt to backup/original_copy.txt  
Error: Error opening source file: permission denied

---

## read_file

Reads and returns the content of a file. Supports UTF-8 encoding.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `path` | string | Yes | Path to the file to read (relative to base directory) |

### Example Usage

```json
{
  "tool": "read_file",
  "arguments": {
    "path": "config.json"
  }
}
```

### Response

Success: Returns the full content of the file  
Error: Error reading 'config.json': no such file or directory

---

## write_file

Writes content to a file, creating it if it doesn't exist. Overwrites existing files. Parent directories are created automatically if needed.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `path` | string | Yes | Path to the file (relative to base directory) |
| `content` | string | Yes | Content to write to the file |

### Example Usage

```json
{
  "tool": "write_file",
  "arguments": {
    "path": "new_file.txt",
    "content": "Hello, World!"
  }
}
```

Write with auto-creating directories:

```json
{
  "tool": "write_file",
  "arguments": {
    "path": "docs/guides/setup.md",
    "content": "# Setup Guide\n\nFollow these steps..."
  }
}
```

### Response

Success: Written 16 characters to new_file.txt  
Error: Error writing 'path': permission denied

---

## list_dir

Lists all files and directories in the specified path. Includes detailed information about each item.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `path` | string | No | Path to list (relative to base directory, defaults to current directory) |

### Example Usage

```json
{
  "tool": "list_dir",
  "arguments": {
    "path": "."
  }
}
```

List a specific directory:

```json
{
  "tool": "list_dir",
  "arguments": {
    "path": "src/components"
  }
}
```

### Response

Returns a JSON array with file/directory information including names, types, sizes, and modification times.

---

## Common Patterns

### Project Setup

```json
[
  {"tool": "create_dir", "arguments": {"name": "my_project/src"}},
  {"tool": "write_file", "arguments": {"path": "my_project/src/main.go", "content": "package main\n\nfunc main() {}"}},
  {"tool": "write_file", "arguments": {"path": "my_project/README.md", "content": "# My Project"}}
]
```

### File Organization

```json
[
  {"tool": "create_dir", "arguments": {"name": "archive/old_files"}},
  {"tool": "move_file", "arguments": {"source": "temp_data.txt", "dest": "archive/old_files/data_2023.txt"}},
  {"tool": "remove_dir", "arguments": {"path": "temp"}}
]
```

### Backup Files

```json
[
  {"tool": "create_dir", "arguments": {"name": "backups/timestamp"}},
  {"tool": "copy_file", "arguments": {"source": "config.json", "dest": "backups/timestamp/config.json"}}
]
```

---

## Error Handling

All tools return error messages starting with "Error: " when something goes wrong. Common errors include:

- **Missing required argument**: "Error: Missing required argument 'arg_name'"
- **File not found**: "Error: ... no such file or directory"
- **Permission denied**: "Error: ... permission denied"
- **Invalid path type**: "Source path 'X' is a directory, not a file"

---

## Security Notes

1. All paths are relative to the working directory - tools cannot access files outside this boundary
2. Operations like `remove_dir` can delete entire directory trees - use with caution
3. `write_file` will overwrite existing files without warning
4. No undo functionality exists for file operations

---

## Testing

The test suite in `tools_test.go` covers:
- All argument validation
- Edge cases (empty directories, nested paths)
- Error conditions (missing files, permissions)
- Auto-creation of parent directories

Run tests with: `go test -v ./...`