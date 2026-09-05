.PHONY: all test test-verbose test-coverage check-coverage coverage-html lint fmt build test-ci clean run

# Binary name and output directory
BINARY_NAME=gomper
BIN_DIR=bin
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
MIN_COVERAGE=95.0

# Default target
all: build

# Build the binary with VCS metadata stamped automatically
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build -json -buildvcs=true -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Binary created at $(BIN_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test -race ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v -race ./...

# Run tests with coverage
test-coverage: test
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

# Structured CI test target with JSON telemetry and race detector
test-ci:
	@echo "Running CI test suite with structured telemetry..."
	@mkdir -p $(COVERAGE_DIR)
	go test -json -race -coverprofile=$(COVERAGE_FILE) ./...

# Check that test coverage meets minimum threshold
check-coverage: test-coverage
	@echo "Verifying test coverage is at least $(MIN_COVERAGE)%..."
	@go tool cover -func=$(COVERAGE_FILE) | awk -v min=$(MIN_COVERAGE) '/total:/ { \
		sub("%", "", $$3); \
		if (($$3 + 0) < (min + 0)) { \
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

# Run linter via go tool
lint:
	go tool golangci-lint run --config golangci.yaml

# Format code and run linter autofixes
fmt:
	go fmt ./...
	go tool golangci-lint run --fix

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR)

# Run application
run: build
	./$(BIN_DIR)/$(BINARY_NAME)

