package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProjectTest(t *testing.T) (*ToolExecutor, string) {
	dir := t.TempDir()
	te := &ToolExecutor{baseDir: dir, agent: &YoloAgent{}}

	// Create a mini project structure
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg", "auth"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755)) // should be skipped

	// Go files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import (
	"fmt"
	"os"
	"github.com/example/pkg"
)

func main() {
	fmt.Println("hello")
}

type Config struct {
	Name string
}

var Version = "1.0"

const MaxRetries = 3
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "pkg", "auth", "auth.go"), []byte(`package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

type User struct {
	ID   int
	Name string
}
`), 0o644))

	// Python file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script.py"), []byte(`import os
import sys
from pathlib import Path
from collections import defaultdict

def process_data(data):
    return data

class DataProcessor:
    def __init__(self):
        pass

API_KEY = "test"
`), 0o644))

	// JS file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "app.js"), []byte(`import React from 'react';
const express = require('express');
import { useState } from './hooks';

function handleRequest(req, res) {
    res.send('ok');
}

class Server {
    constructor() {}
}

const PORT = 3000;
`), 0o644))

	// Rust file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lib.rs"), []byte(`use std::collections::HashMap;
use serde::Deserialize;

pub fn calculate(x: i32) -> i32 {
    x * 2
}

pub struct Engine {
    data: Vec<u8>,
}

pub enum Status {
    Active,
    Inactive,
}
`), 0o644))

	// A binary file (should be skipped by symbol search)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binary.dat"), []byte{0, 1, 2, 3, 0, 0, 0}, 0o644))

	return te, dir
}

// ─── project_map tests ──────────────────────────────────────────────

func TestProjectMap_Basic(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectMap(nil)
	assert.Contains(t, result, "main.go")
	assert.Contains(t, result, "script.py")
	assert.Contains(t, result, "pkg/")
	assert.Contains(t, result, "cmd/")
	assert.NotContains(t, result, ".git")
	assert.Contains(t, result, "files")
	assert.Contains(t, result, "directories")
}

func TestProjectMap_MaxDepth(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectMap(map[string]any{"max_depth": 1})
	assert.Contains(t, result, "main.go")
	assert.Contains(t, result, "pkg/")
	// Should not contain nested files
	assert.NotContains(t, result, "auth.go")
}

func TestProjectMap_Pattern(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectMap(map[string]any{"pattern": "*.go"})
	assert.Contains(t, result, "main.go")
	assert.NotContains(t, result, "script.py")
	assert.NotContains(t, result, "app.js")
}

func TestProjectMap_NoSizes(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectMap(map[string]any{"show_sizes": false})
	assert.Contains(t, result, "main.go")
	assert.NotContains(t, result, "KB")
	assert.NotContains(t, result, "B)")
}

// ─── dependency_graph tests ─────────────────────────────────────────

func TestDependencyGraph_Go(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.dependencyGraph(nil)
	assert.Contains(t, result, "Dependency Graph")
	assert.Contains(t, result, "fmt")
	assert.Contains(t, result, "os")
	assert.Contains(t, result, "github.com/example/pkg")
}

func TestDependencyGraph_Python(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.dependencyGraph(nil)
	assert.Contains(t, result, "Python")
	assert.Contains(t, result, "pathlib")
	assert.Contains(t, result, "collections")
}

func TestDependencyGraph_JS(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.dependencyGraph(nil)
	assert.Contains(t, result, "react")
	assert.Contains(t, result, "express")
}

func TestDependencyGraph_Rust(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.dependencyGraph(nil)
	assert.Contains(t, result, "Rust")
	assert.Contains(t, result, "std")
	assert.Contains(t, result, "serde")
}

func TestDependencyGraph_PackageFilter(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.dependencyGraph(map[string]any{"package": "pkg/auth"})
	assert.Contains(t, result, "auth")
	assert.Contains(t, result, "crypto/sha256")
	assert.NotContains(t, result, "main.go")
}

func TestDependencyGraph_Empty(t *testing.T) {
	dir := t.TempDir()
	te := &ToolExecutor{baseDir: dir, agent: &YoloAgent{}}

	result := te.dependencyGraph(nil)
	assert.Contains(t, result, "No recognized source packages")
}

// ─── symbol_search tests ────────────────────────────────────────────

func TestSymbolSearch_GoFunc(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "main"})
	assert.Contains(t, result, "func")
	assert.Contains(t, result, "main.go")
}

func TestSymbolSearch_GoType(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "Config", "kind": "type"})
	assert.Contains(t, result, "type")
	assert.Contains(t, result, "Config")
}

func TestSymbolSearch_GoVar(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "Version", "kind": "var"})
	assert.Contains(t, result, "var")
	assert.Contains(t, result, "Version")
}

func TestSymbolSearch_PythonFunc(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "process"})
	assert.Contains(t, result, "func")
	assert.Contains(t, result, "process_data")
	assert.Contains(t, result, "script.py")
}

func TestSymbolSearch_PythonClass(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "DataProcessor", "kind": "class"})
	assert.Contains(t, result, "class")
	assert.Contains(t, result, "DataProcessor")
}

func TestSymbolSearch_JSFunc(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "handleRequest"})
	assert.Contains(t, result, "func")
	assert.Contains(t, result, "handleRequest")
	assert.Contains(t, result, "app.js")
}

func TestSymbolSearch_JSClass(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "Server", "kind": "class"})
	assert.Contains(t, result, "class")
	assert.Contains(t, result, "Server")
}

func TestSymbolSearch_RustFunc(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "calculate"})
	assert.Contains(t, result, "func")
	assert.Contains(t, result, "calculate")
	assert.Contains(t, result, "lib.rs")
}

func TestSymbolSearch_RustStruct(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "Engine", "kind": "type"})
	assert.Contains(t, result, "type")
	assert.Contains(t, result, "Engine")
}

func TestSymbolSearch_CaseInsensitive(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "config"})
	assert.Contains(t, result, "Config")
}

func TestSymbolSearch_NoResults(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "nonexistent_symbol_xyz"})
	assert.Contains(t, result, "No symbols matching")
}

func TestSymbolSearch_InvalidKind(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "foo", "kind": "invalid"})
	assert.Contains(t, result, "Error:")
}

func TestSymbolSearch_PatternFilter(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.symbolSearch(map[string]any{"query": "Hash", "pattern": "**/*.go"})
	assert.Contains(t, result, "HashPassword")
	assert.NotContains(t, result, "script.py")
}

// ─── project_summary tests ──────────────────────────────────────────

func TestProjectSummary_FirstRun(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectSummary(map[string]any{"refresh": true})
	assert.Contains(t, result, "updated")
	assert.Contains(t, result, "Files:")
	assert.Contains(t, result, "Total lines:")
	assert.Contains(t, result, "Go")
}

func TestProjectSummary_Cached(t *testing.T) {
	te, _ := setupProjectTest(t)

	// First call to populate
	te.projectSummary(map[string]any{"refresh": true})

	// Second call should use cache
	result := te.projectSummary(nil)
	assert.Contains(t, result, "cached")
	assert.Contains(t, result, "Files:")
}

func TestProjectSummary_MultiLanguage(t *testing.T) {
	te, _ := setupProjectTest(t)

	result := te.projectSummary(map[string]any{"refresh": true})
	assert.Contains(t, result, "Go")
	assert.Contains(t, result, "Python")
	assert.Contains(t, result, "JavaScript")
	assert.Contains(t, result, "Rust")
}

// ─── Helper tests ────────────────────────────────────────────────────

func TestFormatSize(t *testing.T) {
	assert.Equal(t, "0B", formatSize(0))
	assert.Equal(t, "100B", formatSize(100))
	assert.Equal(t, "1.0KB", formatSize(1024))
	assert.Equal(t, "1.5KB", formatSize(1536))
	assert.Equal(t, "1.0MB", formatSize(1<<20))
}

func TestDetectLanguage(t *testing.T) {
	assert.Equal(t, "Go", detectLanguage("main.go"))
	assert.Equal(t, "Python", detectLanguage("script.py"))
	assert.Equal(t, "JavaScript", detectLanguage("app.js"))
	assert.Equal(t, "TypeScript", detectLanguage("index.ts"))
	assert.Equal(t, "Rust", detectLanguage("lib.rs"))
	assert.Equal(t, "C", detectLanguage("foo.c"))
	assert.Equal(t, "", detectLanguage("unknown.xyz"))
}

func TestIsRelativeImport(t *testing.T) {
	assert.True(t, isRelativeImport("./foo"))
	assert.True(t, isRelativeImport("../bar"))
	assert.False(t, isRelativeImport("react"))
	assert.False(t, isRelativeImport("github.com/foo"))
}

func TestIsExternalImport(t *testing.T) {
	assert.True(t, isExternalImport("github.com/foo/bar"))
	assert.True(t, isExternalImport("@types/node"))
	assert.False(t, isExternalImport("fmt"))
	assert.False(t, isExternalImport("os"))
}

func TestParseGoImports(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	require.NoError(t, os.WriteFile(f, []byte(`package main

import "fmt"

import (
	"os"
	"strings"
	"github.com/pkg/errors"
)
`), 0o644))

	imports := parseGoImports(f)
	assert.Contains(t, imports, "fmt")
	assert.Contains(t, imports, "os")
	assert.Contains(t, imports, "strings")
	assert.Contains(t, imports, "github.com/pkg/errors")
}

func TestParsePythonImports(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.py")
	require.NoError(t, os.WriteFile(f, []byte(`import os
import sys
from pathlib import Path
from collections import defaultdict
`), 0o644))

	imports := parsePythonImports(f)
	assert.Contains(t, imports, "os")
	assert.Contains(t, imports, "sys")
	assert.Contains(t, imports, "pathlib")
	assert.Contains(t, imports, "collections")
}

func TestParseJSImports(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.js")
	require.NoError(t, os.WriteFile(f, []byte(`import React from 'react';
import { foo } from './local';
const bar = require('express');
`), 0o644))

	imports := parseJSImports(f)
	assert.Contains(t, imports, "react")
	assert.Contains(t, imports, "./local")
	assert.Contains(t, imports, "express")
}

func TestParseRustImports(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.rs")
	require.NoError(t, os.WriteFile(f, []byte(`use std::collections::HashMap;
use serde::Deserialize;
use crate::config;
`), 0o644))

	imports := parseRustImports(f)
	assert.Contains(t, imports, "std")
	assert.Contains(t, imports, "serde")
	assert.Contains(t, imports, "crate")
}

func TestProjectMapCache_UpdateAndStats(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.py"), []byte("def foo():\n    pass\n"), 0o644))

	yoloDir := filepath.Join(dir, ".yolo")
	cache := NewProjectMapCache(yoloDir)

	added, updated, removed := cache.Update(dir)
	assert.Equal(t, 2, added)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, removed)

	files, totalLines, languages := cache.GetStats()
	assert.Equal(t, 2, files)
	assert.Greater(t, totalLines, 0)
	assert.Equal(t, 1, languages["Go"])
	assert.Equal(t, 1, languages["Python"])

	// Save and reload
	require.NoError(t, cache.Save())
	cache2 := NewProjectMapCache(yoloDir)
	cache2.Load()
	files2, _, _ := cache2.GetStats()
	assert.Equal(t, 2, files2)
}
