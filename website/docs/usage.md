# Usage Guide

Colino helps you build a personal knowledge garden from high-quality sources. This guide shows you how to use Colino effectively in your daily workflow.

## Getting Started

### First Time Setup
The interactive setup makes it easy to get started:

```bash
./colino setup
```

You'll be guided through:
1. **Adding RSS feeds** - Start with 3-5 feeds you trust
2. **Configuring YouTube** - Optional transcript fetching
3. **Setting up automation** - Background content fetching (macOS)
4. **Initial ingestion** - Your first content fetch

## Working with Your Content

### Check New Content
Use the list command to see what's new without opening your AI client:

```bash
./colino list                    # Last 24 hours
./colino list --hours 168       # Last week
```

### Summarize Specific Articles
Get an AI summary of a specific article or video:

```bash
./colino digest "https://example.com/article-url"
```

This is useful when you want to quickly understand a specific piece of content without adding it to your permanent collection.

### MCP Usage
Connect with your AI assistant through the MCP (Model Context Protocol) server:

```bash
./colino server
```

Once your AI client is configured to use the Colino MCP server, you can ask questions about your recent content and have follow-up conversations with the AI. The AI can access your collected articles and provide insights, summaries, and analysis based on your personal knowledge garden.
