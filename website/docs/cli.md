# CLI Reference

Colino ships as a single binary with subcommands for ingestion and the MCP server. Below is a concise reference for common tasks.

## Global
```bash
./colino --help
./colino --version
```

## Setup
Interactive wizard that writes `~/.config/colino/config.yaml`, bootstraps the DB, and optionally installs a launchd schedule on macOS.
```bash
./colino setup
```

## Daemon
Run ingestion once. For scheduling, use launchd/systemd/cron.

One-shot ingest:
```bash
./colino ingest
```

Run in the foreground every N minutes:
```bash
# Interval flag removed; configure periodicity via your scheduler
```

Install as a macOS `launchd` agent:
```bash
./colino ingest schedule \
  # interval flag removed; scheduler controls cadence \
  # sources flag removed; daemon ingests all sources
  --log-file "$HOME/Library/Logs/Colino/daemon.launchd.log"
```

Uninstall the agent:
```bash
./colino ingest unschedule
```

Notes
- Sources are currently consolidated via RSS: `article` performs full-text extraction with Trafilatura; `youtube` attempts transcript retrieval for YouTube links. The stored `content` field is plain text, not HTML.
- Logs default to `~/Library/Logs/Colino/colino.log` unless overridden.

## MCP Server
Expose local content to your LLM client via stdio.
```bash
./colino server
```

Configure your client to start the MCP server. Example for a TOML-based client config:
```toml
[mcp_servers.colino]
command = "/absolute/path/to/colino"
args = ["server"]
```

### Tools
- `list_cache(hours=24, source?, limit=50, include_content=false)`
  - Returns recent entries; use `hours` to scope. Set `include_content=true` to include bodies (otherwise metadata only).
- `get_content(ids[] | url | hours, source?, limit?, include_content=true)`
  - Fetch by a list of entry IDs, a URL, or a time window.

## Build from Source
```bash
go version   # requires 1.23+
go build -o colino ./cmd/colino
```

## Install as a User Tool
- Move the binary into a directory on your PATH (e.g., `~/bin`), or call it via an absolute path from your MCP client.
- macOS users can rely on `ingest schedule` for scheduled runs; non-macOS users can use systemd/cron with `./colino ingest` on a schedule.
