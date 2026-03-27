package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Project Understanding / Code Intelligence ───────────────────────
//
// Tools for understanding codebase structure:
//   - project_map: hierarchical file tree with sizes and descriptions
//   - dependency_graph: parse Go imports to show package relationships
//   - symbol_search: find function/type/variable definitions
//   - project_summary: cached per-file metadata in .yolo/project-map.json

// ─── Skip Directories ────────────────────────────────────────────────

var skipDirs = map[string]bool{
	".git": true, ".yolo": true, "__pycache__": true,
	"node_modules": true, "vendor": true, ".cache": true,
	".idea": true, ".vscode": true,
}

// ─── Project Map ─────────────────────────────────────────────────────

// projectMap generates a hierarchical file tree with metadata.
func (e *ToolExecutor) projectMap(args map[string]any) string {
	maxDepth := getIntArg(args, "max_depth", 4)
	showSizes := getBoolArg(args, "show_sizes", true)
	pattern := getStringArg(args, "pattern", "")

	type fileEntry struct {
		relPath string
		size    int64
		isDir   bool
	}

	var entries []fileEntry
	err := filepath.Walk(e.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(e.baseDir, path)
		if rel == "." {
			return nil
		}

		// Skip hidden/noise directories
		base := filepath.Base(path)
		if info.IsDir() && skipDirs[base] {
			return filepath.SkipDir
		}

		// Check depth
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply pattern filter if specified
		if pattern != "" && !info.IsDir() {
			matched, _ := filepath.Match(pattern, base)
			if !matched {
				return nil
			}
		}

		entries = append(entries, fileEntry{
			relPath: rel,
			size:    info.Size(),
			isDir:   info.IsDir(),
		})
		return nil
	})

	if err != nil {
		return errorMessage("walk error: %v", err)
	}

	if len(entries) == 0 {
		return "Empty project directory"
	}

	// Build tree output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project: %s\n", filepath.Base(e.baseDir)))

	fileCount := 0
	dirCount := 0
	var totalSize int64

	for _, entry := range entries {
		if entry.isDir {
			dirCount++
		} else {
			fileCount++
			totalSize += entry.size
		}

		depth := strings.Count(entry.relPath, string(filepath.Separator))
		indent := strings.Repeat("  ", depth)
		name := filepath.Base(entry.relPath)

		if entry.isDir {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, name))
		} else if showSizes {
			sb.WriteString(fmt.Sprintf("%s%s (%s)\n", indent, name, formatSize(entry.size)))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, name))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%d files, %d directories, %s total", fileCount, dirCount, formatSize(totalSize)))

	// Truncate if too large
	result := sb.String()
	if len(result) > 8000 {
		lines := strings.Split(result, "\n")
		if len(lines) > 200 {
			result = strings.Join(lines[:200], "\n") + fmt.Sprintf("\n... (%d more entries, use max_depth or pattern to narrow)", len(lines)-200)
		}
	}
	return result
}

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// ─── Dependency Graph ────────────────────────────────────────────────

// goImportRegex matches Go import statements.
var goImportRegex = regexp.MustCompile(`^\s*"([^"]+)"`)
var goImportBlockStart = regexp.MustCompile(`^\s*import\s*\(`)
var goSingleImport = regexp.MustCompile(`^\s*import\s+"([^"]+)"`)

// dependencyGraph parses imports/requires across languages and shows relationships.
func (e *ToolExecutor) dependencyGraph(args map[string]any) string {
	pkgFilter := getStringArg(args, "package", "")

	type pkgInfo struct {
		files   []string
		imports map[string]bool
		langs   map[string]bool
	}

	packages := make(map[string]*pkgInfo)

	// Import parsers by extension
	type importParser struct {
		lang    string
		skipExt string // skip files ending with this (e.g., _test.go)
		parse   func(path string) []string
	}

	parsers := map[string]importParser{
		".go":  {lang: "Go", skipExt: "_test.go", parse: parseGoImports},
		".py":  {lang: "Python", parse: parsePythonImports},
		".js":  {lang: "JavaScript", parse: parseJSImports},
		".ts":  {lang: "TypeScript", parse: parseJSImports},
		".jsx": {lang: "JavaScript", parse: parseJSImports},
		".tsx": {lang: "TypeScript", parse: parseJSImports},
		".rs":  {lang: "Rust", parse: parseRustImports},
	}

	err := filepath.Walk(e.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[filepath.Base(path)] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(info.Name())
		parser, ok := parsers[ext]
		if !ok {
			return nil
		}

		if parser.skipExt != "" && strings.HasSuffix(info.Name(), parser.skipExt) {
			return nil
		}

		rel, _ := filepath.Rel(e.baseDir, path)
		pkgDir := filepath.Dir(rel)
		if pkgDir == "." {
			pkgDir = "."
		}

		if pkgFilter != "" && pkgDir != pkgFilter && !strings.HasPrefix(pkgDir, pkgFilter+"/") {
			return nil
		}

		if packages[pkgDir] == nil {
			packages[pkgDir] = &pkgInfo{imports: make(map[string]bool), langs: make(map[string]bool)}
		}
		pkg := packages[pkgDir]
		pkg.files = append(pkg.files, filepath.Base(path))
		pkg.langs[parser.lang] = true

		imports := parser.parse(path)
		for _, imp := range imports {
			pkg.imports[imp] = true
		}

		return nil
	})

	if err != nil {
		return errorMessage("walk error: %v", err)
	}

	if len(packages) == 0 {
		return "No recognized source packages found"
	}

	// Sort package names
	var pkgNames []string
	for name := range packages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	var sb strings.Builder
	sb.WriteString("Dependency Graph\n\n")

	for _, name := range pkgNames {
		pkg := packages[name]
		var langList []string
		for l := range pkg.langs {
			langList = append(langList, l)
		}
		sort.Strings(langList)
		sb.WriteString(fmt.Sprintf("Package: %s [%s] (%d files)\n", name, strings.Join(langList, ", "), len(pkg.files)))

		// Separate stdlib/local vs external imports
		var stdlib, local, external []string
		for imp := range pkg.imports {
			if isRelativeImport(imp) {
				local = append(local, imp)
			} else if isExternalImport(imp) {
				external = append(external, imp)
			} else {
				stdlib = append(stdlib, imp)
			}
		}
		sort.Strings(stdlib)
		sort.Strings(local)
		sort.Strings(external)

		if len(stdlib) > 0 {
			sb.WriteString(fmt.Sprintf("  stdlib: %s\n", strings.Join(stdlib, ", ")))
		}
		if len(local) > 0 {
			sb.WriteString(fmt.Sprintf("  local: %s\n", strings.Join(local, ", ")))
		}
		if len(external) > 0 {
			sb.WriteString("  external:\n")
			for _, ext := range external {
				sb.WriteString(fmt.Sprintf("    - %s\n", ext))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// parseGoImports extracts import paths from a Go source file.
func parseGoImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		// Single-line import
		if m := goSingleImport.FindStringSubmatch(line); len(m) > 1 {
			imports = append(imports, m[1])
			continue
		}

		// Import block start
		if goImportBlockStart.MatchString(line) {
			inBlock = true
			continue
		}

		if inBlock {
			if strings.Contains(line, ")") {
				inBlock = false
				continue
			}
			if m := goImportRegex.FindStringSubmatch(line); len(m) > 1 {
				imports = append(imports, m[1])
			}
		}
	}

	return imports
}

// isRelativeImport returns true for relative imports (./foo, ../bar).
func isRelativeImport(imp string) bool {
	return strings.HasPrefix(imp, "./") || strings.HasPrefix(imp, "../")
}

// isExternalImport returns true for external (non-stdlib) imports.
func isExternalImport(imp string) bool {
	// Go external imports contain dots (github.com/...)
	// npm packages use @ or contain /
	parts := strings.SplitN(imp, "/", 2)
	return strings.Contains(parts[0], ".") || strings.HasPrefix(imp, "@")
}

// ─── Import Parsers (multi-language) ────────────────────────────────

var (
	pyImportRe     = regexp.MustCompile(`^\s*import\s+(\S+)`)
	pyFromImportRe = regexp.MustCompile(`^\s*from\s+(\S+)\s+import`)
	jsImportRe     = regexp.MustCompile(`(?:import\s+.*from\s+['"]([^'"]+)['"]|require\s*\(\s*['"]([^'"]+)['"]\s*\))`)
	rustUseRe      = regexp.MustCompile(`^\s*use\s+(\w+(?:::\w+)*)`)
)

// parsePythonImports extracts import paths from a Python file.
func parsePythonImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := pyFromImportRe.FindStringSubmatch(line); len(m) > 1 {
			seen[m[1]] = true
		} else if m := pyImportRe.FindStringSubmatch(line); len(m) > 1 {
			// Handle "import foo, bar"
			for _, pkg := range strings.Split(m[1], ",") {
				pkg = strings.TrimSpace(pkg)
				if pkg != "" {
					seen[pkg] = true
				}
			}
		}
	}

	var imports []string
	for imp := range seen {
		imports = append(imports, imp)
	}
	return imports
}

// parseJSImports extracts import/require paths from JS/TS files.
func parseJSImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := jsImportRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if m[1] != "" {
				seen[m[1]] = true
			}
			if m[2] != "" {
				seen[m[2]] = true
			}
		}
	}

	var imports []string
	for imp := range seen {
		imports = append(imports, imp)
	}
	return imports
}

// parseRustImports extracts use paths from Rust files.
func parseRustImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	seen := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := rustUseRe.FindStringSubmatch(line); len(m) > 1 {
			// Extract the top-level crate name
			parts := strings.SplitN(m[1], "::", 2)
			seen[parts[0]] = true
		}
	}

	var imports []string
	for imp := range seen {
		imports = append(imports, imp)
	}
	return imports
}

// ─── Symbol Search ───────────────────────────────────────────────────

// symbolSearch finds function, type, class, and variable definitions across languages.
func (e *ToolExecutor) symbolSearch(args map[string]any) string {
	query := getStringArg(args, "query", "")
	kind := getStringArg(args, "kind", "") // "func", "type", "class", "var", "const", or "" for all
	filePattern := getStringArg(args, "pattern", "")

	if query == "" {
		return errorMessage("query is required")
	}

	// Auto-detect file pattern if not specified
	if filePattern == "" {
		filePattern = "**/*"
	}

	queryLower := strings.ToLower(query)

	type symbolHit struct {
		File   string
		Line   int
		Kind   string
		Name   string
		Detail string
	}

	var hits []symbolHit

	// Language-agnostic symbol patterns, keyed by kind
	symbolPatterns := map[string][]*regexp.Regexp{
		// Functions / methods
		"func": {
			regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?(\w+)`),                                                // Go
			regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`),                                        // Rust
			regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)`),                               // JS/TS
			regexp.MustCompile(`^\s*(?:def|async\s+def)\s+(\w+)`),                                               // Python
			regexp.MustCompile(`^\s*(?:public|private|protected|static|\s)*\s+\w+\s+(\w+)\s*\(`),                // Java/C#
			regexp.MustCompile(`^\s*(?:pub\s+)?(?:const\s+)?(\w+)\s*[:=]\s*(?:async\s+)?(?:\([^)]*\)|[^=])*=>`), // JS arrow
		},
		// Types / structs / interfaces
		"type": {
			regexp.MustCompile(`^type\s+(\w+)\s+`),                             // Go
			regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+(\w+)`),                // Rust
			regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+(\w+)`),                  // Rust
			regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+(\w+)`),                 // Rust
			regexp.MustCompile(`^\s*(?:export\s+)?(?:type|interface)\s+(\w+)`), // TS
			regexp.MustCompile(`^\s*typedef\s+.*\s+(\w+)\s*;`),                 // C/C++
			regexp.MustCompile(`^\s*(?:pub\s+)?(?:struct|union)\s+(\w+)`),      // C
		},
		// Classes
		"class": {
			regexp.MustCompile(`^\s*(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`),                                // JS/TS/Java
			regexp.MustCompile(`^\s*class\s+(\w+)`),                                                              // Python/Ruby
			regexp.MustCompile(`^\s*(?:public|internal|private)?\s*(?:abstract|sealed|static)?\s*class\s+(\w+)`), // C#
		},
		// Variables / constants
		"var": {
			regexp.MustCompile(`^var\s+(\w+)`),                                       // Go
			regexp.MustCompile(`^\s*(?:let|const|var)\s+(\w+)`),                      // JS/TS
			regexp.MustCompile(`^\s*(?:pub\s+)?(?:static\s+)?(?:let|const)\s+(\w+)`), // Rust
			regexp.MustCompile(`^(\w+)\s*=\s*`),                                      // Python top-level
		},
		"const": {
			regexp.MustCompile(`^const\s+(\w+)`),      // Go
			regexp.MustCompile(`^\s*#define\s+(\w+)`), // C/C++
		},
	}

	// Filter by kind
	activePatterns := symbolPatterns
	if kind != "" {
		if p, ok := symbolPatterns[kind]; ok {
			activePatterns = map[string][]*regexp.Regexp{kind: p}
		} else {
			return errorMessage("unknown kind %q (use: func, type, class, var, const)", kind)
		}
	}

	files, err := globFiles(e.baseDir, filePattern)
	if err != nil {
		return errorMessage("glob error: %v", err)
	}

	for _, filePath := range files {
		// Skip binary-looking files
		info, err := os.Stat(filePath)
		if err != nil || info.Size() == 0 || info.Size() > 1<<20 {
			continue
		}
		if isBinaryFile(filePath, info.Size()) {
			continue
		}

		rel, _ := filepath.Rel(e.baseDir, filePath)

		f, err := os.Open(filePath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			for symKind, regexes := range activePatterns {
				for _, re := range regexes {
					matches := re.FindStringSubmatch(line)
					if len(matches) < 2 {
						continue
					}

					name := matches[1]
					detail := strings.TrimSpace(line)
					if len(detail) > 100 {
						detail = detail[:97] + "..."
					}

					if strings.Contains(strings.ToLower(name), queryLower) {
						hits = append(hits, symbolHit{
							File:   rel,
							Line:   lineNum,
							Kind:   symKind,
							Name:   name,
							Detail: detail,
						})
						break // don't match same line with multiple regexes
					}
				}
			}
		}
		f.Close()

		if len(hits) >= 100 {
			break
		}
	}

	if len(hits) == 0 {
		return fmt.Sprintf("No symbols matching '%s' found", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d symbol(s) matching '%s':\n\n", len(hits), query))
	for _, h := range hits {
		sb.WriteString(fmt.Sprintf("  [%s] %s:%d  %s\n", h.Kind, h.File, h.Line, h.Detail))
	}
	if len(hits) >= 100 {
		sb.WriteString("\n  (results truncated at 100)")
	}
	return sb.String()
}

// ─── File Summary Cache ──────────────────────────────────────────────

// FileSummary stores metadata about a single file.
type FileSummary struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time"`
	Lines    int    `json:"lines"`
	Language string `json:"language,omitempty"`
	Symbols  int    `json:"symbols,omitempty"` // count of top-level symbols
}

// ProjectMapData is persisted to .yolo/project-map.json.
type ProjectMapData struct {
	Version int                    `json:"version"`
	Updated string                 `json:"updated"`
	RootDir string                 `json:"root_dir"`
	Files   map[string]FileSummary `json:"files"`
}

// ProjectMapCache manages the per-file summary cache.
type ProjectMapCache struct {
	yoloDir  string
	dataFile string
	Data     ProjectMapData
	mu       sync.Mutex
}

// NewProjectMapCache creates a cache manager.
func NewProjectMapCache(yoloDir string) *ProjectMapCache {
	return &ProjectMapCache{
		yoloDir:  yoloDir,
		dataFile: filepath.Join(yoloDir, "project-map.json"),
		Data: ProjectMapData{
			Version: 1,
			Files:   make(map[string]FileSummary),
		},
	}
}

// Load reads the cache from disk.
func (c *ProjectMapCache) Load() {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := os.ReadFile(c.dataFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &c.Data)
	if c.Data.Files == nil {
		c.Data.Files = make(map[string]FileSummary)
	}
}

// Save writes the cache to disk atomically.
func (c *ProjectMapCache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.MkdirAll(c.yoloDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.Data, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.dataFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.dataFile)
}

// Update scans the project directory and updates the cache incrementally.
// Only files whose mod time has changed are re-analyzed.
func (c *ProjectMapCache) Update(baseDir string) (added, updated, removed int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	seen := make(map[string]bool)

	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[filepath.Base(path)] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(baseDir, path)
		seen[rel] = true

		modTime := info.ModTime().Format(time.RFC3339)
		existing, ok := c.Data.Files[rel]
		if ok && existing.ModTime == modTime && existing.Size == info.Size() {
			return nil // unchanged
		}

		// Analyze file
		summary := FileSummary{
			Path:     rel,
			Size:     info.Size(),
			ModTime:  modTime,
			Language: detectLanguage(rel),
		}

		// Count lines for text files
		if !isBinaryFile(path, info.Size()) {
			summary.Lines = countLines(path)
			if strings.HasSuffix(rel, ".go") {
				summary.Symbols = countGoSymbols(path)
			}
		}

		c.Data.Files[rel] = summary
		if ok {
			updated++
		} else {
			added++
		}
		return nil
	})

	// Remove files that no longer exist
	for rel := range c.Data.Files {
		if !seen[rel] {
			delete(c.Data.Files, rel)
			removed++
		}
	}

	c.Data.Updated = time.Now().Format(time.RFC3339)
	c.Data.RootDir = baseDir
	return
}

// GetStats returns summary statistics.
func (c *ProjectMapCache) GetStats() (files, totalLines int, languages map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	languages = make(map[string]int)
	for _, f := range c.Data.Files {
		files++
		totalLines += f.Lines
		if f.Language != "" {
			languages[f.Language]++
		}
	}
	return
}

// ─── Helpers ─────────────────────────────────────────────────────────

// detectLanguage returns a language label based on file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "Go"
	case ".py":
		return "Python"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".jsx", ".tsx":
		return "React"
	case ".java":
		return "Java"
	case ".rs":
		return "Rust"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "C++"
	case ".rb":
		return "Ruby"
	case ".sh", ".bash":
		return "Shell"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".sql":
		return "SQL"
	case ".proto":
		return "Protobuf"
	case ".mod":
		return "Go Module"
	case ".sum":
		return "Go Sum"
	default:
		return ""
	}
}

// countLines counts the number of lines in a text file.
func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count
}

// countGoSymbols counts top-level func/type/var/const declarations in a Go file.
func countGoSymbols(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "func ") ||
			strings.HasPrefix(line, "type ") ||
			strings.HasPrefix(line, "var ") ||
			strings.HasPrefix(line, "const ") {
			count++
		}
	}
	return count
}

// isBinaryFile checks if a file is likely binary (by reading first few bytes).
func isBinaryFile(path string, size int64) bool {
	if size == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return isBinaryData(buf[:n])
}

// ─── Tool: project_summary ──────────────────────────────────────────

// projectSummary updates and returns the cached project summary.
func (e *ToolExecutor) projectSummary(args map[string]any) string {
	refresh := getBoolArg(args, "refresh", false)

	yoloDir := resolveYoloDir()
	if e.agent != nil && e.agent.config != nil {
		yoloDir = e.agent.config.GetYoloDir()
	}
	cache := NewProjectMapCache(yoloDir)
	cache.Load()

	// Always update if refresh requested or cache is empty
	if refresh || len(cache.Data.Files) == 0 {
		added, updated, removed := cache.Update(e.baseDir)
		if err := cache.Save(); err != nil {
			return errorMessage("failed to save cache: %v", err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Project summary updated: %d added, %d updated, %d removed\n\n",
			added, updated, removed))
		appendProjectStats(&sb, cache)
		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project summary (cached: %s)\n\n", cache.Data.Updated))
	appendProjectStats(&sb, cache)
	sb.WriteString("\nUse refresh=true to rescan")
	return sb.String()
}

func appendProjectStats(sb *strings.Builder, cache *ProjectMapCache) {
	files, totalLines, languages := cache.GetStats()
	sb.WriteString(fmt.Sprintf("Files: %d\n", files))
	sb.WriteString(fmt.Sprintf("Total lines: %d\n", totalLines))

	if len(languages) > 0 {
		sb.WriteString("Languages:\n")
		// Sort by count descending
		type langCount struct {
			lang  string
			count int
		}
		var lc []langCount
		for l, c := range languages {
			lc = append(lc, langCount{l, c})
		}
		sort.Slice(lc, func(i, j int) bool { return lc[i].count > lc[j].count })
		for _, l := range lc {
			sb.WriteString(fmt.Sprintf("  %s: %d files\n", l.lang, l.count))
		}
	}
}
