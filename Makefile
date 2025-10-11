.PHONY: build test demo-build demo-record demo-clean demo-server help

# Default target
all: build

# Build Colino binary
build:
	go build -o colino ./cmd/colino

# Build demo server binary
demo-build:
	go build -o demo-server ./cmd/demo-server

# Run tests
test:
	go test ./...

# Run demo server (for manual testing)
demo-server: demo-build
	./demo-server -port 8080

# Record demo using VHS (requires VHS to be installed)
demo-record:
	@echo "Starting demo server..."
	./demo-server -port 8080 > /dev/null 2>&1 & \
	sleep 2 && \
	echo "Recording demo..." && \
	vhs demo/demo.tape && \
	echo "Demo recording complete!" && \
	pkill -f demo-server || true

# Clean demo artifacts
demo-clean:
	rm -f demo-server demo/demo.gif demo/golden.ascii demo/golden.ascii.tmp

# Full demo setup and recording
demo: demo-build demo-record

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build Colino binary"
	@echo "  test         - Run tests"
	@echo "  demo-build   - Build demo server binary"
	@echo "  demo-server  - Run demo server on port 8080"
	@echo "  demo-record  - Record demo using VHS (requires VHS)"
	@echo "  demo-clean   - Clean demo artifacts"
	@echo "  demo         - Full demo setup and recording"
	@echo "  help         - Show this help message"
