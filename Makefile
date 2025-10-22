.PHONY: build test demo demo-clean demo-server help

# Default target
all: build

# Build Colino binary
build:
	go build -o colino ./cmd/colino

format:
	go fmt ./... 

# Run tests
test:
	go test ./...

# Clean demo artifacts
demo-clean:
	rm -f demo-server tapes/setup.gif tapes/tui.gif tapes/setup.ascii tapes/tui.ascii tapes/setup.ascii.tmp tapes/tui.ascii.tmp

# Record demo (usage: make demo DEMO=setup or make demo DEMO=tui)
demo:
	@if [ -z "$(DEMO)" ]; then \
		echo "Usage: make demo DEMO=setup or make demo DEMO=tui"; \
		exit 1; \
	fi
	@TEMP_HOME=$$(mktemp -d); \
	echo "🏠 Using temporary home: $$TEMP_HOME"; \
	nix-shell nix/shell.nix --run "HOME=$$TEMP_HOME go run ./cmd/vhs-helper $(DEMO)"; \
	rm -rf $$TEMP_HOME

# Force rebuild demo (clears nix cache)
demo-fresh:
	@if [ -z "$(DEMO)" ]; then \
		echo "Usage: make demo-fresh DEMO=setup or make demo-fresh DEMO=tui"; \
		exit 1; \
	fi
	@echo "🧹 Clearing build cache..."
	@rm -f colino demo-server tapes/setup.gif tapes/tui.gif tapes/setup.ascii tapes/tui.ascii
	@echo "🎬 Generating $(DEMO) demo in fresh environment..."
	@TEMP_HOME=$$(mktemp -d); \
	echo "🏠 Using temporary home: $$TEMP_HOME"; \
	nix-shell nix/shell.nix --run "HOME=$$TEMP_HOME make demo DEMO=$(DEMO)"; \
	rm -rf $$TEMP_HOME

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build Colino binary"
	@echo "  test         - Run tests"
	@echo "  demo         - Record demo (usage: make demo DEMO=setup|tui)"
	@echo "  demo-clean   - Clean demo artifacts"
	@echo "  demo-fresh   - Force rebuild demo with pure nix environment (usage: make demo-fresh DEMO=setup|tui)"
	@echo "  help         - Show this help message"
