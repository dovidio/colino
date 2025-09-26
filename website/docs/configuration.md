# Configuration

Colino reads `config.yaml` from:
- `~/.config/colino/config.yaml`
- `./config.yaml`

Only a subset of settings are used by the new Go daemon and MCP server.

## Database
```yaml
database_path: "~/Library/Application Support/Colino/colino.db"
```
- If not set, Colino uses the default macOS path above; otherwise `colino.db` in the current directory on other platforms.
- For backwards compatibility, `database.path` (from the old Python config) is still honored.

## RSS
```yaml
rss:
  feeds:
    - https://hnrss.org/frontpage
  timeout: 30               # seconds (default 30)
  max_posts_per_feed: 100   # default 100
  scraper_max_workers: 5    # default 5
```

## YouTube Transcript (optional)
```yaml
youtube:
  proxy:
    enabled: false
    webshare:
      username: "your_username"
      password: "your_password"
```
- When an RSS entry links to YouTube, the daemon attempts to fetch the default transcript and store it as content.
- If many transcripts are fetched and you hit rate limits, enable a Webshare rotating proxy.

## Daemon (optional)
```yaml
daemon:
  enabled: true             # informational; CLI flags control runtime
  # daemon settings are not read; scheduling is handled by launchd/systemd/cron
  sources: [article]        # placeholders for future source controls
  log_file: "~/Library/Logs/Colino/colino.log"
```

Notes
- Filtering and AI summarization settings from the previous Python CLI are no longer used here.
- The MCP server exposes data via tools; any summarization happens in your LLM client.
