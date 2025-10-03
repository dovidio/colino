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
    - https://hnrss.org/frontpage
    - https://feeds.arstechnica.com/arstechnica/index
    - https://stratechery.com/feed/
  timeout: 30               # seconds to wait for each feed
  max_posts_per_feed: 100   # maximum posts to keep per feed
  scraper_max_workers: 5    # parallel downloads for article content
```

#### Choosing Quality Feeds

Start with sources that provide consistent value:

**Technology & Research:**
- `https://hnrss.org/frontpage` - Hacker News
- `https://feeds.arstechnica.com/arstechnica/index` - Ars Technica
- `https://www.technologyreview.com/feed/` - MIT Technology Review

**Business & Analysis:**
- `https://stratechery.com/feed/` - Stratechery
- `https://www.aei.org/feed/` - American Enterprise Institute
- `https://www.theatlantic.com/feed/business/` - The Atlantic Business

**Science & Learning:**
- `https://www.nature.com/news/rss` - Nature News
- `https://www.sciencedaily.com/rss/all.xml` - ScienceDaily

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

### Daemon Settings (Informational)

```yaml
daemon:
  enabled: true
  log_file: "~/Library/Logs/Colino/colino.log"
```

These settings are mainly for reference:
- **enabled**: Informational only - controlled by CLI commands
- **log_file**: Where to write log files (macOS default shown above)

## Advanced Configuration Examples

### Research-Oriented Setup

```yaml
database_path: "~/research/colino.db"

rss:
  feeds:
    # Academic journals
    - https://www.nature.com/nature/articles?type=news&format=rss
    - https://science.sciencemag.org/rss/news_current.xml
    # Research blogs
    - https://blog.acolyer.org/feed/
    - https://www.marginalrevolution.com/feed/
  timeout: 60
  max_posts_per_feed: 200
  scraper_max_workers: 3

youtube:
  proxy:
    enabled: false
```

### Business Intelligence Setup

```yaml
rss:
  feeds:
    # Industry news
    - https://feeds.bloomberg.com/markets/news.rss
    - https://feeds.feedburner.com/venturebeat/SZYF
    - https://techcrunch.com/feed/
    # Company blogs
    - https://blog.google/rss/
    - https://openai.com/blog/rss.xml
  timeout: 30
  max_posts_per_feed: 50
  scraper_max_workers: 8
```

### Minimalist Setup

```yaml
rss:
  feeds:
    - https://hnrss.org/frontpage
    - https://stratechery.com/feed/
  timeout: 20
  max_posts_per_feed: 30
  scraper_max_workers: 2
```

## Configuration Tips

### Starting Out
1. **Begin with 3-5 feeds** you know and trust
2. **Use defaults** for timeout and worker settings
3. **Monitor for a week** before adding more feeds

### Performance Tuning
- **Reduce `scraper_max_workers`** if you have connection issues
- **Increase `timeout`** for slow websites or poor connections
- **Lower `max_posts_per_feed`** if your database grows too quickly

### Feed Quality Management
- **Remove feeds** that consistently provide low-value content
- **Add specialized feeds** for specific research projects
- **Test new feeds** with `colino digest` before adding permanently

## Troubleshooting Configuration

### Feed Not Working?
1. **Test the URL**: Open it in your browser or feed reader
2. **Check timeout**: Increase from 30 to 60 seconds
3. **Verify format**: Should be standard RSS/Atom

### Too Much Content?
1. **Reduce `max_posts_per_feed`** to 20-50 per feed
2. **Remove less valuable feeds**
3. **Use time-based queries** with your AI assistant

### Connection Issues?
1. **Reduce `scraper_max_workers`** to 2-3
2. **Increase `timeout`** to 60+ seconds
3. **Check your internet connection**

## Changing Configuration

1. **Edit the file** at `~/.config/colino/config.yaml`
2. **Restart any running daemon** processes
3. **Run manual ingestion** to test changes:
   ```bash
   colino daemon
   ```

## Getting Help

If you're having trouble with configuration:

1. **Check the syntax** - YAML is picky about indentation
2. **Validate URLs** - Make sure feeds are accessible
3. **Start simple** - Use the setup wizard to create a working config
4. **Ask for help** - Open an issue on GitHub with your config (remove sensitive info)

Remember: The goal is a configuration that serves your information needs without becoming overwhelming. Start simple and evolve based on your actual usage patterns.
