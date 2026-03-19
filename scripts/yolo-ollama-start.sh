#!/bin/bash
# yolo-ollama-start.sh - Start Ollama server with optional logging
#
# Usage: ./yolo-ollama-start.sh [--log]
#   --log    Redirect ollama output to logs/ollama.log for debugging

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$BASE_DIR/logs"
LOG_FILE="$LOG_DIR/ollama.log"

# Function to check if ollama is running
check_ollama_running() {
    pgrep -f "ollama serve" > /dev/null 2>&1
}

# Function to get the PID of running ollama
get_ollama_pid() {
    pgrep -f "ollama serve" || echo ""
}

if [ "$1" = "--log" ]; then
    echo "============================================"
    echo "Starting Ollama server with debug logging"
    echo "============================================"
    echo ""
    
    # Stop any running ollama instances
    if check_ollama_running; then
        OLD_PID=$(get_ollama_pid)
        echo "Stopping existing ollama instance (PID: $OLD_PID)..."
        pkill -f "ollama serve" 2>/dev/null || true
        sleep 1
    fi
    
    # Create logs directory if it doesn't exist
    mkdir -p "$LOG_DIR"
    
    # Remove stale lock file if any
    rm -f /tmp/ollama.pid
    
    # Start ollama with logging
    echo "Starting ollama serve..."
    nohup ollama serve >> "$LOG_FILE" 2>&1 &
    OLLAMA_PID=$!
    
    # Store PID for easy access
    echo $OLLAMA_PID > /tmp/ollama.pid
    
    # Wait briefly and check if it started successfully
    sleep 2
    if ps -p $OLLAMA_PID > /dev/null; then
        echo "✓ Ollama server started with PID: $OLLAMA_PID"
        echo ""
        echo "Debug output is being logged to: $LOG_FILE"
        echo ""
        echo "To view logs in real-time:"
        echo "  tail -f $LOG_FILE"
        echo ""
        echo "To stop ollama:"
        echo "  kill $OLLAMA_PID"
        echo "  or: pkill -f 'ollama serve'"
        echo ""
        echo "YOLO can read the log file at: $LOG_FILE"
    else
        echo "✗ Failed to start ollama server"
        echo "Check the error output above for details"
        exit 1
    fi
else
    echo "============================================"
    echo "Starting Ollama server (quiet mode)"
    echo "============================================"
    echo ""
    
    # Stop any running ollama instances
    if check_ollama_running; then
        OLD_PID=$(get_ollama_pid)
        echo "Stopping existing ollama instance (PID: $OLD_PID)..."
        pkill -f "ollama serve" 2>/dev/null || true
        sleep 1
    fi
    
    # Start ollama with output suppressed
    echo "Starting ollama serve..."
    nohup ollama serve > /dev/null 2>&1 &
    OLLAMA_PID=$!
    
    # Wait briefly and check if it started successfully
    sleep 2
    if ps -p $OLLAMA_PID > /dev/null; then
        echo "✓ Ollama server started with PID: $OLLAMA_PID"
        echo ""
        echo "Output is suppressed. To enable logging:"
        echo "  ./yolo-ollama-start.sh --log"
        echo ""
        echo "To stop ollama:"
        echo "  kill $OLLAMA_PID"
        echo "  or: pkill -f 'ollama serve'"
    else
        echo "✗ Failed to start ollama server"
        exit 1
    fi
fi
