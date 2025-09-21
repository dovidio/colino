Colino MCP + Daemon (Go)
========================

This is a Go binary that:
- Runs a Model Context Protocol (MCP) server to expose your Colino DB to LLM clients.
- Runs a background daemon that ingests RSS content directly in Go.

Tools
- list_cache(hours=24, source?, limit=50, include_content=false)
  - Reads recent items from the `content_cache` table, returning metadata and either a content preview or full content.
- get_content(ids[] | url | hours, source?, limit?, include_content=true)
  - Fetches specific items by ID/URL or a time-window slice.

Build
```bash
go build -o golino ./cmd/golino
```

Run MCP server (stdio)
```bash
./golino
# or
./golino server
```

Run daemon (periodic ingestion)
```bash
./golino daemon \
  --interval-minutes 30 \

# Single run (useful for testing):
./golino daemon --once
```

Install as a launchd agent (macOS)
```bash
# Install and load (runs every 30 minutes; each invocation does a single ingest)
./golino daemon install \
  --label com.colino.daemon \
  --interval-minutes 30 \
  --sources rss \
  --log-file "$HOME/Library/Logs/Colino/daemon.launchd.log"

# Uninstall (unloads and removes the plist)
./golino daemon uninstall --label com.colino.daemon
```

Daemon config (optional) in `~/.config/colino/config.yaml`:
```yaml
daemon:
  enabled: true            # not required by the binary, useful for future installers
  interval_minutes: 30
  sources: [rss]
  # Optional log file
  log_file: "~/Library/Logs/Colino/daemon.log"
```

Codex integration
```toml
[mcp_servers.colino]
command = "/Users/umbertodovidio/hack/colino/golino/golino"
args = ["server"]
env = {}
```
- Add the snippet in `.codex/config.toml`
- Build locally first, then point `command` to the built binary.
- Codex connects over stdio; tools will be auto-discovered (`list_cache`, `get_content`, `ingest_recent`).

Notes
- The server discovers the SQLite DB path from `~/.config/colino/config.yaml` (Python config: `database.path`; Golino config: `database_path`). It falls back to the default platform path.
 - If the database is missing, the daemon will initialize the schema on first run.
 - RSS ingestion is implemented in Go using `gofeed` (parsing) and `go-readability`/`goquery` (content extraction). No server-side filters are applied; ingest everything and filter in your LLM or client.
 - YouTube entries are treated like RSS items. When a feed item links to YouTube, the daemon attempts to fetch the default transcript and stores it as content. To improve reliability, you can enable a Webshare proxy in your config:

   yaml
   youtube:
     proxy:
       enabled: true
       webshare:
         username: "YOUR_WS_USER"
         password: "YOUR_WS_PASS"
 - The `daemon install` subcommand generates and loads a `launchd` agent that runs `golino daemon --once` on a schedule via `StartInterval`.
