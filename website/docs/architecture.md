# Architecture

Colino 0.2.0 (alpha) is a local-first content cache with two main runtime components: a background ingestion daemon and a Model Context Protocol (MCP) server. Both are shipped in a single CLI binary.

## High-level Overview
- Daemon: periodically ingests sources (RSS feeds, article extraction via Trafilatura, YouTube transcripts discovered via RSS) into a local SQLite DB. Stored `content` is plain text (no HTML), using Trafilatura output or a stripped-text fallback from RSS description when extraction fails.
- MCP server: exposes your local cache to LLM clients over stdio via a small set of tools for discovery and retrieval.
- Config + Paths: user config lives under `~/.config/colino/config.yaml`; the SQLite DB defaults to `~/Library/Application Support/Colino/colino.db` on macOS, or `colino.db` elsewhere.

## Module Layout (Go)
- `cmd/colino`: CLI entrypoint; subcommands `server`, `daemon`, `setup`.
- `internal/server`: MCP server implementation, tools:
  - `list_cache(hours, source?, limit, include_content?)`
  - `get_content(ids[] | url | hours, source?, limit?, include_content)`
- `internal/daemon`: scheduling/ingestion loop; supports `--once` and interval-based runs; macOS launchd installer lives under `internal/launchd`.
- `internal/ingest/rss.go`: RSS ingestion, Trafilatura extraction, YouTube transcript fetching (when an RSS item links to YouTube).
- `internal/colinodb`: SQLite schema and queries; content normalization and upsert logic.
- `internal/config`: config parsing, default paths, and expansion of `~`.

## Data Flow
1. Sources: RSS feeds are defined in config (`rss.feeds`). For items that link to articles, Colino fetches the page and extracts readable content. For YouTube links, it attempts to fetch the default transcript.
2. Storage: entries are normalized into the SQLite DB (`colino.db`). Duplicate handling is performed by URL or a stable key derived from the feed + link.
3. Access: your LLM client connects to the MCP server (`./colino server`), and invokes tools to list or fetch content by time window, URL, or IDs.

## Runtime Behavior
- Daemon
  - One-shot: `./colino daemon --once`
  - Scheduled: `./colino daemon --interval-minutes 30`
  - macOS service install: `./colino daemon install --interval-minutes 30 --sources article,youtube`
- Server
  - Stdio: `./colino server`
  - Tools return metadata and optionally the full content body based on `include_content`.

## Privacy & Locality
- All ingestion and storage happens locally; Colino does not send your content to remote services.
- Proxies for YouTube transcripts are optional and opt-in via config.

## Logs & Diagnostics
- macOS daemon logs default to `~/Library/Logs/Colino/colino.log` (or the path you pass during install).
- If the MCP server can’t find a DB, run one ingestion cycle first: `./colino daemon --once`.

## Versioning
- This architecture reflects the 0.2.0 (alpha) rewrite of Colino, consolidating the daemon and MCP server into a single Go binary for simplicity and performance.
