# YOLO Feature Wishlist — OpenClaw Parity & Improvements

Comparison baseline: [OpenClaw](https://github.com/openclaw/openclaw) (v2026, 247k+ GitHub stars).

> **Design constraints preserved**: YOLO intentionally uses only local LLMs via Ollama (no paid API keys), starts with zero pre-configuration beyond model selection, and supports autonomous mode. This wishlist respects those constraints — nothing here should add startup friction.

---

## 1. Skills / Plugin System

**Status: Missing entirely**

OpenClaw's most impactful feature is its skills system — modular `SKILL.md` directories that inject context, constraints, and tool-usage instructions into the system prompt. 5,400+ community skills exist on ClawHub.

### Recommended for YOLO
- [ ] **Skill loading from `skills/` directories** — each skill is a folder with a `SKILL.md` (YAML frontmatter + instructions). Load from `~/.yolo/skills/` (global) and `<project>/skills/` (workspace), with workspace taking precedence.
- [ ] **Hot-reload skill files** — watch `SKILL.md` for changes, reload without restart.
- [ ] **Skill injection into system prompt** — eligible skills get compacted and appended to the system prompt automatically.
- [ ] **`/skills` command** — list active/available skills from the CLI.
- [ ] **Community skill registry** — lower priority, but a simple git-based registry (like a curated list repo) would let users share YOLO skills.

> This is the single highest-leverage missing feature. It turns YOLO from a fixed-tool agent into an extensible platform.

---

## 2. Memory & Knowledge Persistence

**Status: Partially implemented (history.json, knowledge.md) — needs significant improvement**

OpenClaw has a tiered memory architecture: always-loaded MEMORY.md (~100 lines), daily context files, and deep knowledge with vector search.

### Recommended for YOLO
- [ ] **Structured MEMORY.md** — a curated, always-loaded memory file (distinct from `knowledge.md`) that the agent actively maintains. Cap at ~100 lines of durable facts: user preferences, project conventions, coding style, key decisions.
- [ ] **Daily context logs** — auto-create `memory/YYYY-MM-DD.md` files with raw observations. Load today + yesterday automatically.
- [ ] **Memory promotion** — periodically (or on command) distill daily logs into structured knowledge and prune stale entries.
- [ ] **`/memory` command** — view, edit, and manage memory tiers from the CLI.
- [ ] **Semantic search over memory** (stretch) — use Ollama embeddings to enable vector search over the deep knowledge tier, so relevant context surfaces automatically without loading everything.

> Current `knowledge.md` is a good start but is passive — the agent doesn't actively curate it.

---

## 3. MCP (Model Context Protocol) Support

**Status: Missing**

OpenClaw integrates heavily with MCP servers, giving it access to thousands of tool providers (databases, APIs, cloud services) via a standard protocol.

### Recommended for YOLO
- [ ] **MCP client implementation** — support connecting to MCP servers (stdio and HTTP transport). This unlocks the entire MCP ecosystem without building individual integrations.
- [ ] **MCP config in `.yolo/mcp.json`** — simple JSON config to declare MCP servers, similar to how Claude Code and OpenClaw handle it.
- [ ] **Dynamic tool registration from MCP** — tools from MCP servers appear alongside native tools in the `ollamaTools` slice.

> MCP is becoming the standard protocol for AI tool access. Supporting it would massively expand YOLO's capabilities without building each integration by hand.

---

## 4. Tool Access Control & Permissions

**Status: Missing — all tools are always available**

OpenClaw has `tools.allow` / `tools.deny` config with deny-wins semantics, plus per-agent tool profiles.

### Recommended for YOLO
- [ ] **Tool allowlist/denylist in config** — let users restrict which tools the agent can call (e.g., disable `send_email` or `run_command` for safety).
- [ ] **Confirmation prompts for destructive actions** — before file deletion, email sending, or command execution, optionally require user confirmation (configurable per-tool).
- [ ] **Tool profiles** — predefined sets like "safe" (no commands, no email), "developer" (all code tools), "full" (everything).

> This is important for trust and safety, especially in autonomous mode where the agent operates without supervision.

---

## 5. Context Window Management

**Status: Basic (40 message limit, no summarization)**

OpenClaw supports session pruning, compaction (`/compact` command), and intelligent context management.

### Recommended for YOLO
- [ ] **Conversation compaction / summarization** — when approaching context limits, summarize older messages rather than just dropping them. Use the LLM itself to create a summary.
- [ ] **`/compact` command** — manually trigger conversation compaction.
- [ ] **Sliding window with summary prefix** — keep a running summary of the conversation so far, appended before the recent message window.
- [ ] **Per-model context awareness** — dynamically adjust MaxContextMessages based on the detected context window size rather than a fixed 40.

> The current hard 40-message cutoff means long sessions lose important context abruptly.

---

## 6. Multi-Model / Model Failover

**Status: Single model at a time, manual switching only**

OpenClaw supports multiple model providers with auth rotation and automatic failover.

### Recommended for YOLO
- [ ] **Model failover chain** — configure a primary + fallback model list. If the primary model fails or is unavailable, automatically try the next.
- [ ] **Per-task model routing** — use a smaller/faster model for simple tasks (file listing, search) and a larger model for complex reasoning. Could be configured in `.yolo/config.json`.
- [ ] **Model performance tracking** — log response times and success rates per model to inform routing decisions.

> Since YOLO depends on local Ollama models which may be resource-constrained, failover is especially valuable.

---

## 7. Scheduling & Cron

**Status: Missing**

OpenClaw has built-in cron jobs, wakeup schedules, and webhook triggers with exponential retry backoff.

### Recommended for YOLO
- [ ] **Simple cron/scheduler** — let the agent schedule future actions (e.g., "check build status every 10 minutes", "run tests at 6 AM").
- [ ] **Persistent schedules in `.yolo/cron.json`** — survive restarts.
- [ ] **Heartbeat mode** — in autonomous mode, wake up at configurable intervals even when idle to check for pending work (inbox, todos, scheduled tasks).

> This pairs naturally with YOLO's existing autonomous mode — scheduled tasks give autonomous mode more to do.

---

## 8. Improved Edit Tool

**Status: Implemented but limited (first-match only, no multi-edit)**

YOLO's `edit_file` does simple first-match string replacement. OpenClaw's coding agent skills leverage more sophisticated editing.

### Recommended for YOLO
- [ ] **Multi-match editing** — support replacing all occurrences, or the Nth occurrence.
- [ ] **Line-range editing** — edit by line numbers (replace lines 10-20 with new content).
- [ ] **Diff-based editing** — accept unified diff format for complex multi-hunk edits.
- [ ] **Edit preview / dry-run** — show what would change before applying (especially useful in autonomous mode).
- [ ] **Undo/rollback** — maintain a simple edit history per file, allowing the agent to revert recent changes.

> The current first-match-only approach causes failures when the search string appears multiple times or when complex edits are needed.

---

## 9. Git Integration

**Status: Only via `run_command` — no native git tools**

OpenClaw's coding agent skill and many community skills have first-class git support.

### Recommended for YOLO
- [ ] **Native git tools** — `git_status`, `git_diff`, `git_commit`, `git_log`, `git_branch`, `git_checkout` as first-class tools instead of requiring `run_command("git ...")`.
- [ ] **Auto-commit on significant changes** — optionally create checkpoint commits after successful multi-file edits (configurable).
- [ ] **Diff awareness in context** — automatically include `git diff` output in the system prompt so the agent always knows what's changed.

> Native git tools give the LLM better-structured output and enable smarter behaviors like pre-commit validation.

---

## 10. Project Understanding / Code Intelligence

**Status: Basic (list_files, search_files, read_file)**

OpenClaw ecosystem has tools like AgentLens that provide hierarchical codebase views and dependency analysis.

### Recommended for YOLO
- [ ] **Project map / tree tool** — generate a structural overview of the codebase (file tree with brief descriptions) that fits in a small context window.
- [ ] **Dependency graph** — for Go projects, parse imports to understand package relationships.
- [ ] **Symbol search** — search for function/type/variable definitions, not just text patterns.
- [ ] **File summary cache** — maintain a `.yolo/project-map.json` with per-file summaries, updated incrementally as files change.

> Helps the agent make better decisions about which files to read and edit, especially in larger codebases.

---

## 11. Session Management

**Status: Basic (single session, history.json)**

OpenClaw has named sessions, session isolation, cross-session messaging, and session persistence.

### Recommended for YOLO
- [ ] **Named sessions** — support multiple named conversation threads (e.g., "feature-x", "bugfix-y") with separate histories.
- [ ] **Session switching** — `/session <name>` to switch between contexts.
- [ ] **Session summary on resume** — when resuming a session, show a brief AI-generated summary of where things left off.

> Currently YOLO has a single global session which makes it hard to context-switch between tasks.

---

## 12. Safety & Sandboxing

**Status: Path sandboxing only — commands are unrestricted**

OpenClaw runs execution in Docker containers and has per-session elevated bash toggles.

### Recommended for YOLO
- [ ] **Command allowlist/denylist** — restrict which commands can be run (e.g., block `rm -rf /`, `sudo`, etc.).
- [ ] **Dry-run mode** — show proposed commands without executing, requiring explicit approval.
- [ ] **Execution sandbox option** — optionally run commands in a container or restricted environment.
- [ ] **Audit log** — persist all tool executions (especially commands and file mutations) to `.yolo/audit.log` for review.

> Especially important for autonomous mode where no human is watching.

---

## 13. Voice & TTS Improvements

**Status: Basic TTS implemented — no voice input**

OpenClaw has wake-word detection, continuous voice mode, and multiple TTS backends (ElevenLabs, OpenAI TTS, Edge TTS).

### Recommended for YOLO
- [ ] **Voice input via local STT** — use Whisper (via Ollama or standalone) for speech-to-text input.
- [ ] **Wake word detection** — optional always-listening mode that activates on a keyword.
- [ ] **Streaming TTS** — start speaking before the full response is generated.
- [ ] **Multiple TTS backends** — add piper-tts (local, fast, high-quality) as an option alongside espeak.

> Voice input would make YOLO usable hands-free, which pairs well with autonomous mode.

---

## 14. Web & Browser Improvements

**Status: Basic (DuckDuckGo search, HTML fetch, Playwright)**

OpenClaw has managed Chrome with snapshots, profiles, and action replay.

### Recommended for YOLO
- [ ] **Search engine options** — add Brave Search API (or SearXNG for self-hosted) alongside DuckDuckGo for better results.
- [ ] **Screenshot tool** — capture and describe screenshots (useful for debugging UI work).
- [ ] **Browser session persistence** — keep browser state between tool calls instead of starting fresh each time.
- [ ] **Improved HTML extraction** — use readability-style algorithms to extract article content more cleanly.

---

## 15. Output & UI Improvements

**Status: Two modes (buffer + terminal split) — functional but basic**

OpenClaw has a web UI (WebChat), Control dashboard, companion apps, and rich media handling.

### Recommended for YOLO
- [ ] **Markdown rendering in terminal** — render bold, italic, headers, code blocks with proper ANSI formatting instead of raw markdown.
- [ ] **Progress indicators** — show a spinner or progress bar during long operations (LLM generation, file searches).
- [ ] **Tool call visualization** — clearly show which tool is being called and its arguments before execution, not just results.
- [ ] **Collapsible output** — for long tool results, show a summary with option to expand.
- [ ] **Color-coded tool results** — different colors for different tool types (file ops, commands, search, etc.).

---

## 16. Usage Tracking & Metrics

**Status: Missing**

OpenClaw tracks per-response usage metrics with optional cost reporting.

### Recommended for YOLO
- [ ] **Token usage tracking** — track tokens sent/received per conversation and per session.
- [ ] **Performance metrics** — response times, tool execution times, model throughput.
- [ ] **`/usage` command** — display session statistics.
- [ ] **Resource monitoring** — show Ollama GPU/memory utilization.

> Helps users understand model performance and optimize their setup.

---

## 17. Webhook / External Trigger Support

**Status: Missing**

OpenClaw supports webhooks for external event-driven agent activation.

### Recommended for YOLO
- [ ] **Simple HTTP API** — expose a local endpoint that accepts messages, triggering agent action (e.g., CI/CD webhooks, file system events).
- [ ] **File watch triggers** — monitor specific files/directories and trigger agent action on changes.

---

## 18. Existing Features Needing Improvement

These features exist in YOLO but could be better:

### Sub-agents
- **Current**: 20-round limit, restricted tool subset, JSON-based results.
- **Needed**: Sub-agent progress streaming (see what sub-agents are doing in real-time), sub-agent cancellation, configurable round limits, sub-agent-to-sub-agent communication.

### Search
- **Current**: DuckDuckGo Instant Answers only (limited results quality).
- **Needed**: Full web search (not just instant answers), search result ranking, snippet extraction from actual pages.

### History
- **Current**: 200-message limit, simple prune-oldest.
- **Needed**: Importance-weighted pruning (keep messages with tool calls, errors, and decisions longer), searchable history, export capability.

### Email
- **Current**: Functional but tightly coupled to specific domain (b-haven.org).
- **Needed**: Configurable SMTP settings, support for arbitrary email providers, OAuth support for Gmail/Outlook.

### Configuration
- **Current**: Basic JSON config with a few toggles.
- **Needed**: Hierarchical config (global `~/.yolo/config.json` + project-local `.yolo/config.json`), config validation, `yolo config set/get` CLI, environment-specific overrides.

### Error Recovery
- **Current**: Fail-fast on file mutations, basic error messages.
- **Needed**: Retry logic with backoff for transient failures (Ollama connection drops), graceful degradation when tools fail, error classification (transient vs permanent).

---

## Priority Ranking

| Priority | Feature | Impact | Effort |
|----------|---------|--------|--------|
| **P0** | Skills/plugin system | Transforms YOLO from fixed to extensible | Medium |
| **P0** | Context window management (compaction) | Prevents context loss in long sessions | Medium |
| **P0** | Tool access control | Critical for autonomous mode safety | Low |
| **P1** | Memory system improvements | Better long-term agent intelligence | Medium |
| **P1** | MCP support | Unlocks ecosystem of integrations | High |
| **P1** | Improved edit tool | Core workflow quality | Low |
| **P1** | Git integration (native tools) | Common workflow, better UX | Low |
| **P1** | Project understanding tools | Better code navigation | Medium |
| **P2** | Safety & sandboxing | Trust & safety | Medium |
| **P2** | Session management | Multi-task workflows | Medium |
| **P2** | Scheduling / cron | Autonomous mode enhancement | Medium |
| **P2** | Output & UI improvements | Developer experience | Low-Med |
| **P2** | Model failover | Resilience | Low |
| **P3** | Voice input (STT) | Nice-to-have | Medium |
| **P3** | Usage tracking | Observability | Low |
| **P3** | Webhooks / triggers | External integration | Medium |
| **P3** | Web/browser improvements | Enhanced research | Medium |

---

## Features Intentionally NOT Recommended

These OpenClaw features are excluded because they conflict with YOLO's design philosophy:

- **Cloud API key management / paid model support** — YOLO is local-LLM-only by design.
- **Messaging platform integrations** (WhatsApp, Telegram, Discord, etc.) — YOLO is a terminal-first developer tool, not a chatbot platform.
- **Companion mobile/desktop apps** — out of scope for a CLI tool.
- **Web UI / dashboard** — YOLO is intentionally terminal-native.
- **Multi-user / team features** — YOLO is a personal tool.
- **OAuth / cloud auth flows** — adds startup friction, conflicts with local-first philosophy.
- **Enterprise features** (audit trails, compliance, RBAC) — not the target audience.

---

*Generated by comparing YOLO (commit HEAD) against OpenClaw (March 2026). See [OpenClaw GitHub](https://github.com/openclaw/openclaw) and [OpenClaw Docs](https://docs.openclaw.ai) for reference.*
