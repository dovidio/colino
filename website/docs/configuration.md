# Configuration Guide

Colino uses a simple YAML configuration file that controls how it fetches and stores content. Most users won't need to change anything after the initial setup, but this guide shows you what's possible.

## Configuration File Location

Colino looks for your configuration in this order:
1. `~/.config/colino/config.yaml` (recommended)
2. `./config.yaml` (in the current directory)

## Quick Configuration with Setup

For most users, the interactive setup is all you need:

```bash
colino setup
```

This will help you:
- Add your first RSS feeds
- Configure YouTube transcript access
- Set up automatic content fetching (macOS)
- Create the initial configuration file

## Configuration Options

### Database Settings

```yaml
database_path: "~/Library/Application Support/Colino/colino.db"
```

- **macOS default**: `~/Library/Application Support/Colino/colino.db`
- **Other platforms**: `./colino.db` in the current directory
- **Custom location**: Any path you prefer

**Example for Linux/Windows:**
```yaml
database_path: "~/.local/share/colino/colino.db"
```

### RSS Feeds

The heart of your knowledge garden:

```yaml
rss:
  feeds:
    - https://example-feed.com/rss
    - https://another-feed.com/rss
  timeout: 30               # seconds to wait for each feed
  max_posts_per_feed: 100   # maximum posts to keep per feed
  scraper_max_workers: 5    # parallel downloads for article content
```

#### RSS Settings Explained

- **timeout**: How long to wait for feeds to respond (default: 30 seconds)
- **max_posts_per_feed**: Prevents any single feed from overwhelming your database (default: 100)
- **scraper_max_workers**: How many articles to download simultaneously (default: 5, reduce if you have connection issues)

### YouTube Transcripts (Optional)

When your RSS feeds include YouTube links, Colino can automatically fetch transcripts:

```yaml
youtube:
  proxy:
    enabled: false
    webshare:
      username: "your_username"
      password: "your_password"
```

#### When to Use a Proxy

Most users won't need a proxy. Consider one only if:
- You're fetching many YouTube transcripts daily
- You encounter rate limiting errors
- You want to ensure consistent access

**Note**: This is entirely optional and not needed for normal usage.
