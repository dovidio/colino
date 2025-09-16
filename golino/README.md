Golino
======

Golino is a Go port of the Colino CLI. It focuses on performance (concurrency via goroutines) and a pleasant terminal experience (progress indicators) while mirroring Colino’s features: ingesting RSS/YouTube sources, storing entries in SQLite, listing recent items, and generating a simple digest.

Features
- Parallel ingest of RSS and YouTube feeds using a worker pool.
- SQLite storage with upsert semantics on GUID.
- Simple TUI progress bar during ingestion.
- Config compatibility: will auto-create `~/.config/colino/config.yaml` with defaults.
- Commands: `ingest`, `list`, `digest` (supports OpenAI summaries).

Structure
- `cmd/golino/main.go`: Cobra CLI entrypoint.
- `internal/config`: YAML config load/save with sensible defaults.
- `internal/db`: SQLite schema and queries.
- `internal/sources/rss`: Minimal RSS parser using `encoding/xml`.
- `internal/sources/youtube`: Converts channel URLs to RSS feeds and reuses RSS parser.
- `internal/ingest`: Orchestrates concurrent fetching + DB upserts with progress.
- `internal/digest`: Simple digest output (LLM stub-ready).
- `internal/ui`: Progress bar wrapper.

Getting Started
1) Ensure Go 1.21+ is installed.
2) From repo root: `cd golino`
3) Build: `go build ./cmd/golino`
4) Run examples:
   - `./golino ingest`
   - `./golino list --hours 24`
   - `./golino digest` or `./golino digest --rss`
   - With OpenAI summary: `export OPENAI_API_KEY=... && ./golino digest`

Configuration
- Default location: `~/.config/colino/config.yaml`.
- Example fields:

  ```yaml
  database_path: /path/to/colino.db
  sources:
    rss:
      - https://example.com/feed
    youtube:
      - https://www.youtube.com/channel/UC_x5XG1OV2P6uZZ5FSM9Ttw
  ingest:
    concurrency: 8
    timeout_seconds: 20
    user_agent: golino/0.1
  openai:
    model: gpt-4o-mini
    temperature: 0.3
    system_prompt: |
      You write crisp, factual tech/news digests.
    user_prompt: |
      Summarize the following items into 3-6 short sections by theme.
      Use markdown with headers and bullet points; include links inline.
      Focus on substance; avoid fluff; keep it under ~250 words.

      Items:
      {{items}}
  ```

Notes
- YouTube support relies on channel RSS feeds; handles are best converted to channel IDs.
- For LLM-based digests, set `OPENAI_API_KEY` in your environment. Digest uses the configured prompts in `openai.system_prompt` and `openai.user_prompt` (with `{{items}}` placeholder). If missing or on error, it falls back to the plain digest.
