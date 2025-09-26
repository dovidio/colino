# Installation

Colino is now a Go project. You can build from source:

1. Ensure Go 1.23+ is installed: https://go.dev/dl/
2. Clone the repo and build the CLI:

```bash
git clone https://github.com/<you>/colino
cd colino
go build -o colino ./cmd/colino
```

The resulting `./colino` binary provides both the ingestion command and the MCP server.

Optional: add it to your PATH or install to a `bin/` directory of your choice.

3. Run the setup wizard

```bash
./colino setup
```

This guides you through adding RSS feeds, choosing an ingestion interval (for launchd scheduling), and (optionally) configuring a Webshare proxy. It then writes `~/.config/colino/config.yaml`, runs an initial ingest to bootstrap the database, and installs a scheduled launchd job on macOS.
