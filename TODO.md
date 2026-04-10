# YOLO Agent - TODO & Future Work

## 🎯 High Priority

### Test Coverage Improvements

- [ ] **Main package (yolo)** - Currently at 41.2% coverage
  - Add tests for `handleToolOutput` function
  - Mock Ollama API calls in agent tests
  - Test tool registration and execution flow
  
- [ ] **Victron/macos** - Platform-specific BLE implementation
  - Add mock BLE device tests (already have backend, need more tests)
  - Test Bluetooth scanning with various device types

### Documentation

- [x] Add architecture diagram showing YOLO agent components (ARCHITECTURE.md)
- [ ] Document Ollama model requirements and prompts
- [ ] Create troubleshooting guide for common issues

## 📋 Medium Priority

### Code Quality

- [ ] Refactor error handling consistency across packages
- [ ] Add more context cancellation support in long-running operations
- [ ] Implement structured logging throughout codebase

### Feature Enhancements

- [ ] Add configuration file support for YOLO agent settings
- [ ] Support multiple Ollama models with model selection
- [ ] Add rate limiting for external API calls

## 🔧 Low Priority / Nice to Have

### Testing

- [ ] Add integration tests for email sending (with test SMTP server)
- [ ] Benchmark performance of Victron device polling
- [ ] Add fuzzing tests for VE.Direct parser

### Documentation

- [ ] Create developer onboarding guide
- [ ] Document all MCP tool integrations
- [ ] Write blog post about YOLO agent architecture

## 📊 Coverage Status

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| yolo/victron | 87.1% | 90% | ✅ Near target |
| yolo/email | 73.4% | 80% | ⚠️ Good progress |
| yolo/main | 41.2% | 60% | 🔴 Needs work |
| yolo/victron/macos | 10.6% | 50% | 🔴 Platform-specific |

## 🏗️ Technical Debt

- Email package's `sendViaSendmail` is hard to test (executes OS commands)
  - Solution: Create mock sendmail executable for testing
  
- Victron/macos uses platform-specific code
  - Consider abstracting BLE interface for cross-platform support
  
- Main agent complexity makes testing difficult
  - Consider splitting into smaller, more testable components

## 🚀 Future Features (Stretch Goals)

- [ ] Web UI for monitoring YOLO agent activity
- [ ] Slack/Discord integration for notifications
- [ ] Support for other BLE energy monitoring devices (not just Victron)
- [ ] Machine learning model training pipeline integration
- [ ] Multi-device support (control multiple Victron devices simultaneously)

---

*Last updated: 2025*
