#!/bin/bash
# Generate code coverage report for YOLO
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out
