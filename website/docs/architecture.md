# Architecture

Colino is a local-first content cache with two main runtime components: a background ingestion daemon and a Model Context Protocol (MCP) server. Both are shipped in a single CLI binary.

## High-level Overview

- **Ingestion**: Ingests sources (RSS feeds, article extraction via Trafilatura, YouTube transcripts) into a local SQLite database. Stored content is plain text, using Trafilatura output or a stripped-text fallback when extraction fails.

- **MCP Server**: Exposes your local cache to LLM clients over stdio via a small set of tools for discovery and retrieval.

- **Config + Paths**: User config lives under `~/.config/colino/config.yaml`; the SQLite database defaults to `~/Library/Application Support/Colino/colino.db` on macOS, or `colino.db` elsewhere.
