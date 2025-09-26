
# Colino 📰

Your personal, hackable news cache with an MCP server for LLMs.

Colino now ships as a Go binary that:
- Runs a background daemon to ingest RSS feeds (and YouTube transcripts discovered via RSS entries) into a local SQLite DB.
- Exposes the content via a Model Context Protocol (MCP) server for LLM clients.

Quick start
```bash
go build -o colino ./cmd/colino
./colino ingest                 # ingest once (scheduling handled by launchd/systemd)
./colino server                 # run MCP server on stdio
```

Docs: [https://colino.pages.dev](https://colino.pages.dev)

Note: macOS is the primary target (launchd integration is available).

## Git hooks (auto-format + vet)

This repo includes a versioned pre-commit hook that formats staged Go files with `gofmt` and runs `go vet`.

Enable it once per repo:

```bash
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

On commit, it will:
- Format any staged `*.go` files and re-stage them
- Run `go vet ./...` and block the commit if issues are found
