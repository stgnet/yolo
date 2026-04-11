#!/bin/bash
# Manual test script for Victron BLE functionality
# Run this to test scanning, connecting, and reading values from real devices

set -e

echo "=== Victron BLE Test Suite ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}✓ PASS:${NC} $1"
}

fail() {
    echo -e "${RED}✗ FAIL:${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}⚠ WARN:${NC} $1"
}

# Test 1: Check dependencies
echo "Test 1: Checking dependencies..."
if command -v go &> /dev/null; then
    pass "Go is installed ($(go version))"
else
    fail "Go is not installed"
fi

# Test 2: Run unit tests
echo ""
echo "Test 2: Running unit tests..."
cd "$(dirname "$0")/src/yolo"
if go test ./victron/... -v > /tmp/victron_test.log 2>&1; then
    pass "All unit tests passed"
else
    fail "Unit tests failed (see /tmp/victron_test.log)"
fi

# Test 3: Check coverage
echo ""
echo "Test 3: Checking test coverage..."
COVERAGE=$(go test ./victron -cover 2>&1 | grep "coverage:" | awk '{print $2}')
if [ -n "$COVERAGE" ]; then
    echo "  Core package coverage: $COVERAGE"
    if [ "$(echo "$COVERAGE >= 80%" | bc)" -eq 1 ] 2>/dev/null; then
        pass "Coverage is above 80% target ($COVERAGE)"
    else
        warn "Coverage is below 80% target ($COVERAGE)"
    fi
else
    warn "Could not determine coverage"
fi

# Test 4: Build examples
echo ""
echo "Test 4: Building example programs..."
if go build -o /tmp/victron-scan ./victron/examples/basic.go 2>/dev/null; then
    pass "Example program builds successfully"
else
    fail "Example program failed to build"
fi

# Test 5: BLE scan (optional, requires nearby devices)
echo ""
echo "Test 5: BLE Device Scan (optional)"
echo "  This test requires a Victron device nearby with BLE enabled."
echo "  Skipping automatic execution - run manually if needed:"
echo "    go run ./victron/examples/basic.go"

# Summary
echo ""
echo "=== Test Suite Complete ==="
echo ""
echo "For manual testing with actual devices:"
echo "  1. Ensure Victron device has BLE enabled"
echo "  2. Run: go run ./victron/examples/basic.go"
echo "  3. Or use the CLI tool: go run ./victron/cmd/victron-read/main.go scan"
echo ""
echo "Test logs available at: /tmp/victron_test.log"
