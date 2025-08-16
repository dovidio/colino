# Colino 📰

Your own hackable RSS feed aggregator and AI-powered digest generator.

## What is Colino?

Colino is a simple, powerful, and completely free RSS feed aggregator that lets you create your own personalized news digest from any RSS-enabled website. Additionally, Colino leverages LLM to generate a digest of the latest news

## Quick Start

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Create your .env file:**
   ```bash
   # RSS feeds
   RSS_FEEDS=https://hnrss.org/frontpage,https://rss.cnn.com/rss/edition.rss
   
   # For AI digest (optional)
   OPENAI_API_KEY=your_openai_api_key_here
   ```

3. **Fetch content:**
   ```bash
   python src/main.py fetch
   ```

4. **Generate AI digest:**
   ```bash
   python src/main.py digest
   ```

## Usage Examples

```bash
# Discover RSS feeds from any website
python src/main.py discover https://example.com

# Test a specific feed
python src/main.py test https://feeds.example.com/rss

# Fetch from specific feeds only
python src/main.py fetch --urls https://hnrss.org/frontpage

# List recent posts with filtering
python src/main.py list --hours 48 --limit 20

# Generate AI-powered digest
python src/main.py digest --hours 24
python src/main.py digest --output daily_digest.md

# Export your feeds for backup
python src/main.py export --output my_feeds.opml

# Import feeds from OPML file
python src/main.py import my_feeds.opml
```