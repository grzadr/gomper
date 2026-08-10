.PHONY: all test test-verbose test-coverage check-coverage coverage-html lint build clean run

# Binary name and output directory
BINARY_NAME=gomper
BIN_DIR=bin
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
MIN_COVERAGE=90.0

# Default target
all: build

# Build the binary (depends on clean, lint, test-coverage, and check-coverage passing)
build: clean lint check-coverage
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) main.go
	@echo "Binary created at $(BIN_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

# Check that test coverage meets minimum threshold
check-coverage: test-coverage
	@echo "Verifying test coverage is at least $(MIN_COVERAGE)%..."
	@go tool cover -func=$(COVERAGE_FILE) | awk -v min=$(MIN_COVERAGE) '/total:/ { \
		sub("%", "", $$3); \
		if ($$3 < min) { \
			print "ERROR: Test coverage (" $$3 "%) is below required threshold (" min "%)."; \
			exit 1; \
		} else { \
			print "Test coverage check passed: " $$3 "% >= " min "%"; \
		} \
	}'

# Display coverage report in HTML
coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE)

# Run linter (golangci-lint if available, fallback to go vet)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, running go vet..."; \
		go vet ./...; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR)

# Run application
run:
	go run main.go
