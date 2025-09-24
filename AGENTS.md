# Repository Guidelines

## Project Structure & Module Organization
- Go module at repo root (`go.mod`). CLI entrypoint in `cmd/colino`.
- Internal packages in `internal/`:
  - `server/` (MCP server exposing tools `list_cache`, `get_content`).
  - `daemon/` (scheduler/ingestion loop), `ingest/rss.go` (RSS + Trafilatura extraction + YouTube transcripts).
  - `colinodb/` (SQLite schema and queries), `config/` (YAML config + paths), `launchd/` (macOS agent).
- Runtime config: `~/.config/colino/config.yaml` (created by setup). `config.yaml` at repo root is for local dev.
- SQLite DB: `~/Library/Application Support/Colino/colino.db` on macOS; `colino.db` elsewhere.
- Docs site: `website/` (Docusaurus). Built output goes to `dist/`.

## Build, Test, and Development Commands
- Build CLI: `go build -o colino ./cmd/colino`
- Run MCP server: `./colino server` (stdio MCP)
- One‑shot ingest: `./colino daemon --once`
- Install macOS daemon: `./colino daemon install --interval-minutes 30 --sources article,youtube`
- Lint/format: `gofmt -w .` and `go vet ./...` (keep code idiomatic)
- Website: `cd website && npm ci && npm run start` | build: `npm run build`

## Coding Style & Naming Conventions
- Idiomatic Go: `gofmt`/`goimports` required; keep packages small and cohesive.
- Package names lower-case singular; exported identifiers `PascalCase`, internal `camelCase`.
- Avoid stutter (`package server` → `server.Run`), keep files focused by concern.

## Testing Guidelines
- Place tests alongside code as `_test.go`; run with `go test ./...`.
- Prioritize unit tests for `internal/ingest`, `internal/colinodb`, and config parsing.
- Use table-driven tests; keep DB tests deterministic (temporary DB file).

## Commit & Pull Request Guidelines
- Use Conventional Commits (e.g., `feat:`, `fix:`, `chore:`). Keep subjects concise and imperative; reference issues (e.g., `#123`).
- PRs must describe scope, validation steps (commands/logs), and include screenshots for `website/`.
- Pre-merge: `gofmt`, `go vet`, build `./cmd/colino`, and ensure the app runs (`./colino daemon --once`).

## Security & Configuration Tips
- Do not commit local databases or secrets. User config lives in `~/.config/colino/config.yaml`.
- macOS logs: `~/Library/Logs/Colino/colino.log`. Prefer user-level config over repo `config.yaml`.
