.PHONY: build test demo demo-clean demo-server help

# Default target
all: build

# Build Colino binary
build:
	go build -o colino ./cmd/colino

format:
	gofmt -l .

# Run tests
test:
	go test ./...

# Clean demo artifacts
demo-clean:
	rm -f demo-server demo/demo.gif demo/golden.ascii demo/golden.ascii.tmp

# Force rebuild demo (clears nix cache)
demo-fresh:
	@echo "🧹 Clearing build cache..."
	@rm -f colino demo-server demo/demo.gif demo/golden.ascii demo/golden.ascii.tmp
	@echo "🎬 Generating demo in fresh environment..."
	@TEMP_HOME=$$(mktemp -d); \
	echo "🏠 Using temporary home: $$TEMP_HOME"; \
	nix-shell nix/shell.nix --run "HOME=$$TEMP_HOME scripts/run-demo.sh"; \
	rm -rf $$TEMP_HOME

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build Colino binary"
	@echo "  test         - Run tests"
	@echo "  demo-build   - Build demo server binary"
	@echo "  demo-server  - Run demo server on port 8080"
	@echo "  demo-record  - Record demo using VHS (requires VHS)"
	@echo "  demo-clean   - Clean demo artifacts"
	@echo "  demo         - Generate demo in clean nix-shell environment"
	@echo "  demo-fresh   - Force rebuild demo with pure nix environment (no cache)"
	@echo "  help         - Show this help message"
