# Morning Show Makefile

.PHONY: build run clean test deps help

# Default target
all: build

# Build the morning-show binary
build:
	@echo "Building morning-show..."
	go build -o morning-show main.go
	@echo "Build complete!"

# Run the morning-show script
run: build
	@echo "Running morning-show..."
	./morning-show

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -f morning-show
	rm -f morning-show.wav
	rm -f morning-show.txt
	@echo "Clean complete!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed!"

# Run tests (if any)
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests complete!"

# Show help
help:
	@echo "Morning Show - Available targets:"
	@echo "  build    - Build the morning-show binary"
	@echo "  run      - Build and run the morning-show script"
	@echo "  clean    - Remove build artifacts and output files"
	@echo "  deps     - Install and update dependencies"
	@echo "  test     - Run tests"
	@echo "  help     - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  GEMINI_API_KEY - Required: Your Google Gemini API key"
	@echo "  MINIFLU_URL    - Optional: Miniflu API endpoint (default: http://localhost:8080/api/messages/unread)"
	@echo "  OUTPUT_FILE    - Optional: Output filename (default: morning-show.wav)"