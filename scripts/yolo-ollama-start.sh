#!/bin/bash
# yolo-ollama-start.sh - Start Ollama server with optional logging
#
# Usage: ./yolo-ollama-start.sh [--log]
#   --log    Redirect ollama output to logs/ollama.log

if [ "$1" = "--log" ]; then
    echo "Starting ollama serve with output logged to logs/ollama.log"
    mkdir -p logs
    nohup ollama serve >> logs/ollama.log 2>&1 &
    echo "Ollama PID: $!"
    echo "To view logs in real-time: tail -f logs/ollama.log"
    echo "To stop ollama: pkill -f 'ollama serve'"
else
    echo "Starting ollama serve (output will appear on terminal if OLLAMA_DEBUG is set)"
    nohup ollama serve > /dev/null 2>&1 &
    echo "Ollama PID: $!"
fi
