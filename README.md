
# Colino 📰

Your personal, hackable news cache with an MCP server for LLMs.

Colino now ships as a Go binary that:
- Runs a background daemon to ingest RSS feeds (and YouTube transcripts discovered via RSS entries) into a local SQLite DB.
- Exposes the content via a Model Context Protocol (MCP) server for LLM clients.

Quick start
```bash
go build -o colino ./cmd/colino
./colino daemon --once          # ingest once
./colino server                 # run MCP server on stdio
```

Docs: [https://colino.pages.dev](https://colino.pages.dev)

Note: macOS is the primary target (launchd integration is available).
