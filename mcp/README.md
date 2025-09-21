Colino MCP Server (Go)
======================

This is a Go Model Context Protocol (MCP) server that exposes your Colino content database to LLM clients.

Tools
- list_cache(hours=24, source?, limit=50, include_content=false)
  - Reads recent items from the `content_cache` table, returning metadata and either a content preview or full content.
- get_content(ids[] | url | hours, source?, limit?, include_content=true)
  - Fetches specific items by ID/URL or a time-window slice.

Build
```bash
cd mcp
go build -o colino-mcp ./cmd/colino-mcp
```

Run (stdio transport)
```bash
./colino-mcp
```

Codex integration
```toml
[mcp_servers.colino]
command = "/Users/umbertodovidio/hack/colino/mcp/colino-mcp"
args = []
env = {}
```
- Add the snippet in `.codex/config.toml`
- Build locally first, then point `command` to the built binary.
- Codex connects over stdio; tools will be auto-discovered (`list_cache`, `get_content`, `ingest_recent`).

Notes
- The server discovers the SQLite DB path from `~/.config/colino/config.yaml` (Python config: `database.path`; Golino config: `database_path`). It falls back to the default platform path.
 - If the database is missing or not initialized, tools return a friendly hint explaining how to create it with the Python CLI.
