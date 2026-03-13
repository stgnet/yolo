# YOLO Autonomous Operations Guide

## Overview

YOLO (Your Own Living Operator) is a self-evolving AI agent designed to work autonomously without human intervention. In autonomous mode, YOLO continuously identifies and implements improvements to its own codebase, processes emails, runs tests, and maintains system health - all without requiring user input.

### How Autonomous Mode Works

When no new user input is provided, YOLO enters autonomous mode and follows a continuous improvement cycle:

1. **Self-Assessment**: Analyzes current state (tests, coverage, code quality)
2. **Prioritization**: Identifies the most impactful next task
3. **Execution**: Takes concrete action using available tools
4. **Verification**: Ensures changes work correctly
5. **Iteration**: Immediately moves to the next improvement

This cycle repeats indefinitely, allowing YOLO to continuously evolve and improve itself.

---

## Self-Improvement Cycle

### Identifying Improvements

YOLO autonomously identifies improvement opportunities through:

- **Test Analysis**: Running `go test -v ./...` to find failing tests
- **Coverage Reports**: Using `go tool cover` to identify uncovered code paths
- **Code Quality Checks**: Running `gofmt -l .` to find formatting issues
- **Build Verification**: Ensuring `go build` succeeds
- **External Research**: Using `learn` tool to discover new features and best practices from the internet

### Code Change Workflow

When YOLO identifies an improvement, it follows a strict workflow:

```
1. Spawn Subagent (spawn_subagent)
   ↓
2. Implement Changes
   ↓
3. Monitor Progress (list_subagents)
   ↓
4. Retrieve Results (read_subagent_result)
   ↓
5. Test Changes (go test -v ./...)
   ↓
6. Format Code (gofmt -w .)
   ↓
7. Commit to Git (git add, git commit)
   ↓
8. Restart YOLO (restart tool)
```

### Safety Mechanisms

YOLO includes several safety mechanisms:

- **Non-Destructive Operations**: Only modifies its own source code and test files
- **Test Validation**: All changes must pass existing tests before commitment
- **Atomic Commits**: Changes are committed in small, reversible units
- **Error Recovery**: Failed subagents are detected and can be retried with different approaches
- **No External Side Effects**: Does not modify user files outside the YOLO repository

---

## Email Processing Automation

YOLO automatically processes incoming emails through a multi-step workflow:

### Tools Available

1. **check_inbox**: Reads emails from `/var/mail/b-haven.org/yolo/new/`
2. **process_inbox_with_response**: Full automation (read → respond → delete)
3. **send_email**: Sends custom emails via sendmail
4. **send_report**: Sends progress reports to scott@stg.net

### Email Response Process

When processing emails, YOLO:

1. Extracts email metadata (from, subject, date)
2. Analyzes content and intent
3. Composes appropriate responses using LLM
4. Sends responses via Postfix (with DKIM signing)
5. Deletes processed messages

### Email Response Prompt Structure

YOLO uses enhanced prompts for email responses that include:
- Sender information
- Message subject and body
- Context about YOLO's current activities
- Guidelines for appropriate tone and content
- Specific constraints (no action items, don't make promises)

---

## Monitoring and Status Checking

### Key Commands

```bash
# Check test status
go test -v ./...

# View test coverage
go tool cover -func=cover.out | grep -E "(total|^[[:space:]]*[^[:space:]]+[[:space:]]*\([[:space:]]*[^%]+$)"

# Check code formatting
gofmt -l .

# Build verification
go build

# List active subagents
# (Use list_subagents tool)

# Check git status
git status
```

### Status Indicators

**Healthy System**:
- ✅ All tests passing
- ✅ Coverage above 50% (many UI/runtime functions hard to test)
- ✅ No formatting issues
- ✅ Clean git working directory
- ✅ Subagents completing successfully

**Needs Attention**:
- ❌ Failing tests
- ❌ Low coverage in core functions
- ❌ Formatting errors
- ❌ Staged/uncommitted changes after failures
- ❌ Repeated subagent errors

---

## Configuration Options

### Autonomous Behavior Settings

YOLO's autonomous behavior is configured through its source code:

**tools.go**: Defines available tools and their capabilities
**agent.go**: Controls decision-making logic and tool selection
**main.go**: Entry point and runtime configuration

### Model Selection

YOLO can switch between different Ollama models using the `switch_model` tool. Available models are listed via `list_models`.

Current default: `qwen3.5:27b`

### Email Configuration

- **Incoming**: `/var/mail/b-haven.org/yolo/new/`
- **Outgoing**: yolo@b-haven.org (via Postfix with DKIM)
- **Default Recipient**: scott@stg.net

---

## Troubleshooting Common Issues

### Subagent Failures

**Symptom**: Subagents repeatedly failing with same error

**Solution**: 
1. Check subagent logs via `read_subagent_result`
2. Analyze the error pattern
3. Retry with modified approach or different strategy

### Test Timeouts

**Symptom**: Tests timing out (especially LLM-dependent tests)

**Solution**:
- Use mocked implementations for LLM calls in unit tests
- Increase timeout values for integration tests
- Skip slow tests with build tags if needed

### Build Failures After Changes

**Symptom**: `go build` fails after code modifications

**Solution**:
1. Review recent changes with `git diff`
2. Check for syntax errors or missing imports
3. Verify all tool registrations are consistent
4. Revert problematic commits if necessary

### Coverage Plateau

**Symptom**: Coverage stuck at certain level

**Solution**:
- Identify untested functions via coverage report
- Add targeted unit tests with proper mocking
- Accept that some UI/runtime functions may remain untestable

### Email Processing Issues

**Symptom**: Emails not being processed or responses failing

**Solution**:
1. Verify Maildir structure at `/var/mail/b-haven.org/yolo/new/`
2. Check Postfix service status
3. Review email response prompt for completeness
4. Test send_email tool manually

---

## Best Practices

### Running YOLO Autonomously

1. **Initial Setup**: Ensure all tests pass and coverage is acceptable
2. **Background Operation**: Let YOLO run without interruption
3. **Periodic Checks**: Monitor git commits to see improvements made
4. **Backup Strategy**: Regularly backup the repository before major changes
5. **Resource Management**: Monitor system resources (CPU, memory) for long runs

### Contributing to YOLO's Self-Improvement

- Write clear, testable code that YOLO can understand
- Include comprehensive documentation
- Add meaningful comments explaining complex logic
- Keep functions focused and single-purpose
- Use consistent naming conventions

### Email Interaction Guidelines

- Be clear about what you need from YOLO
- Include relevant context in email subjects
- Understand YOLO has limited capabilities (software development only)
- Don't expect immediate responses (depends on processing cycle)
- Review sent reports for progress updates

### Maintenance Recommendations

1. **Weekly**: Review git history to see improvements made
2. **Monthly**: Run comprehensive test suite and coverage analysis
3. **As Needed**: Check inbox for important communications
4. **Quarterly**: Consider running `learn` tool for new feature discovery

---

## Examples

### Typical Autonomous Session

```
[SYSTEM] No new user input. Autonomous mode activated.
→ YOLO checks tests (all passing)
→ YOLO checks coverage (51.2%)
→ YOLO identifies low-coverage area: email processing
→ YOLO spawns subagent to add email response prompt enhancement
→ Subagent completes successfully
→ YOLO adds unit tests for new functionality
→ YOLO runs gofmt and commits changes
→ YOLO continues to next improvement task
```

### Email Processing Flow

```
Incoming email from scott@stg.net: "Run the learn tool"
→ YOLO detects new email in inbox
→ YOLO reads email content
→ YOLO executes learn() tool
→ Learn performs web search and Reddit analysis
→ YOLO documents findings internally
→ YOLO marks email as processed
→ YOLO continues autonomous operations
```

---

## Conclusion

YOLO's autonomous mode represents a novel approach to self-improving AI agents. By continuously identifying and implementing improvements, processing communications, and maintaining its own health, YOLO demonstrates how AI can operate independently while remaining safe and controllable.

For questions or issues, contact: scott@stg.net
