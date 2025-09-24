# Release Notes: 0.2.0-alpha

Colino 0.2.0-alpha is a complete rewrite as a Go binary that bundles a background ingestion daemon and an MCP server into one CLI. This release focuses on performance, reliability, and a clean local-first architecture.

## Highlights
- Single Go binary providing both the ingestion daemon and MCP server.
- Local SQLite database by default: `~/Library/Application Support/Colino/colino.db` on macOS, `./colino.db` elsewhere.
- RSS ingestion with Trafilatura-based extraction for articles.
- YouTube transcripts when RSS items link to YouTube (optional proxy support).
- MCP tools for LLM clients: `list_cache` and `get_content` over stdio.

## Breaking Changes
- Python CLI is replaced. pip/pipx installs are no longer supported.
- Config keys are simplified and paths standardized; old Python-only options are ignored.
- Summarization and filtering now live in your LLM client; Colino stores and serves content.

## Migration Guide
1. Build the new binary:
   ```bash
   go build -o colino ./cmd/colino
   ```
2. Run the setup to create `~/.config/colino/config.yaml` and bootstrap the DB:
   ```bash
   ./colino setup
   ```
3. Optionally install the macOS daemon:
   ```bash
   ./colino daemon install --interval-minutes 30 --sources article,youtube
   ```
4. Update your MCP client to launch `./colino server` instead of the Python CLI.

## CLI Overview
- `./colino setup` — interactive config + initial ingest, optional launchd install.
- `./colino daemon --once` — one-shot ingest.
- `./colino daemon --interval-minutes 30` — run continuously on an interval.
- `./colino daemon install|uninstall` — manage macOS launchd agent.
- `./colino server` — start the MCP server on stdio.

## MCP Tools
- `list_cache(hours=24, source?, limit=50, include_content=false)` — recent items.
- `get_content(ids[] | url | hours, source?, limit?, include_content=true)` — fetch by IDs/URL/window.

## Known Issues (alpha)
- Some YouTube videos do not provide transcripts; content may be empty.
- If the DB is missing, run one ingest cycle first (`./colino daemon --once`).

## Contributors
Thanks for trying the alpha and filing issues. Feedback on configuration, ingestion speed, and MCP client compatibility is especially helpful.
