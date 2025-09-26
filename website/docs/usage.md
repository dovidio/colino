# Usage

Colino now has two primary modes:

- MCP server to expose your local content cache to an LLM.
- Daemon for periodic ingestion of RSS feeds (and YouTube transcripts via RSS items).

## Quick setup

Run the interactive wizard to create your config, bootstrap the DB, and (on macOS) install the daemon:
```bash
./colino setup
```

## Ingesting content

Run a single ingestion cycle:
```bash
./colino ingest
```

To run on a schedule, use launchd/systemd/cron to invoke `./colino ingest`.

Install as a macOS launchd schedule:
```bash
./colino ingest schedule \
  # interval is configured in launchd \
  # sources flag removed; daemon ingests all sources \
  --log-file "$HOME/Library/Logs/Colino/daemon.launchd.log"

# Uninstall
./colino ingest unschedule
```

## MCP server

Start the server on stdio (for clients like Codex):
```bash
./colino server
```

Then configure your client to launch `./colino server` as an MCP server. Tools available:
- `list_cache(hours=24, source?, limit=50, include_content=false)`
- `get_content(ids[] | url | hours, source?, limit?, include_content=true)`

Example Codex snippet:
```toml
[mcp_servers.colino]
command = "/absolute/path/to/colino"
args = ["server"]
```

## Help
```bash
./colino --help
```
