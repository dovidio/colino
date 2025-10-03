# CLI Reference

Colino ships as a single binary with commands for setup, ingestion, content management, and MCP server access. Everything is designed to be simple and intuitive.

## Global Commands
```bash
./colino --help          # Show all available commands
./colino --version       # Show current version
```

## Setup - Your First Steps
The interactive setup wizard guides you through creating your configuration, adding RSS feeds, and optional scheduling.

```bash
./colino setup
```

The setup will help you:
- Create your configuration file
- Add your first RSS feeds
- Set up automatic ingestion (macOS only for now)
- Run an initial content fetch

## Content Ingestion
Fetch new content from your RSS feeds and add it to your local knowledge base.

### Manual Ingestion
```bash
./colino daemon              # Fetch all content from your configured feeds
```

### Automatic Scheduling (macOS)
```bash
./colino daemon schedule     # Install automatic background fetching
./colino daemon unschedule   # Remove automatic fetching
```

## Explore Your Content
See what content you've collected without needing an AI assistant.

### List Recent Content
```bash
./colino list                # Show content from the last 24 hours
./colino list --hours 72     # Show content from the last 3 days
```

Perfect for quickly checking what's new in your feeds without opening your AI client.

## Digest Individual Content
Process specific articles or videos directly from your terminal.

```bash
./colino digest "https://example.com/article"     # Process an article
./colino digest "https://youtube.com/watch?v=..."  # Process a YouTube video
```

This is useful for:
- Quick content analysis without full ingestion
- Testing new sources before adding them to your feeds
- Processing individual interesting links you discover

## MCP Server - Connect with AI
Expose your knowledge base to your preferred AI assistant through Model Context Protocol.

```bash
./colino server
```

Configure your AI client to connect to Colino as an MCP server. Example configuration:

```toml
[mcp_servers.colino]
command = "/path/to/colino"
args = ["server"]
```

**Available AI Tools:**
- `list_cache(hours=24, source?, limit=50, include_content=false)` - Discover recent content
- `get_content(ids[] | url | hours, source?, limit?, include_content=true)` - Fetch full content for analysis

## Common Workflows

### Daily Content Review
```bash
./colino list                # See what's new
./colino daemon              # Fetch latest content
./colino server              # Start AI for deeper analysis
```

### Add New Source
```bash
./colino digest "https://new-source.com/feed"  # Test the source
# Add to ~/.config/colino/config.yaml if good
./colino daemon                          # Ingest from new source
```

## Technical Details
- **Requirements**: Go 1.23+ for building from source
- **Storage**: Local SQLite database (default: `~/Library/Application Support/Colino/colino.db` on macOS)
- **Platform**: Primary support for macOS (with launchd integration), works on other platforms with manual scheduling
- **Logging**: macOS logs to `~/Library/Logs/Colino/colino.log` by default

## Installation Tips
- Add the built binary to your PATH for easier access
- Use absolute paths when configuring MCP clients
- macOS users get automatic scheduling, other platforms can use systemd/cron
