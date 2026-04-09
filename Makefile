# YOLO Developer Makefile
# Streamlined development workflow

.PHONY: help test test-cover lint vet fmt ci clean build run

help: ## Show this help message
	@echo "YOLO Development Workflow"
	@echo "=========================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests with race detection
	@echo "Running tests with race detection..."
	go test -race -v ./...

test-cover: ## Generate coverage report
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	go tool cover -func=coverage.out | tail -1

lint: ## Run go vet (basic linting)
	@echo "Running go vet..."
	go vet ./...

vet: lint

fmt: ## Format code with gofmt
	@echo "Formatting code..."
	go fmt ./...

ci: test-cover lint ## Run full CI pipeline locally
	@echo "CI pipeline complete!"

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning up..."
	rm -f coverage.out coverage.html
	rm -rf dist/
	find . -type f -name "*.exe" -delete

build: ## Build the binary
	@echo "Building yolo..."
	go build -o dist/yolo .

run: build ## Build and run
	./dist/yolo
