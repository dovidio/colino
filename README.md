# Colino 📰

Your own hackable RSS feed aggregator

Colino is a simple, powerful, and completely free RSS feed aggregator that lets you create your own personalized news digest from any RSS-enabled website. No API keys, no rate limits, no corporate algorithms - just pure, unfiltered content from the sources you choose.

## Why RSS?
# RSS Feeds - Add your favorite RSS/Atom feed URLs separated by commas
RSS_FEEDS=https://hnrss.org/frontpage,https://rss.cnn.com/rss/edition.rss,https://feeds.bbci.co.uk/news/rss.xml,https://blog.python.org/feeds/posts/default.rss

### RSS Settings (add to .env file)
RSS_USER_AGENT=Colino RSS Reader 1.0.0
RSS_TIMEOUT=30

# Database Configuration
DATABASE_PATH=colino.db

# General Settings
MAX_POSTS_PER_FEED=100
DEFAULT_LOOKBACK_HOURS=24

# Content Filtering (Optional)
# Only show posts containing these keywords (comma-separated, leave empty to disable)
FILTER_KEYWORDS=

# Hide posts containing these keywords (comma-separated)
EXCLUDE_KEYWORDS=ads,sponsored,advertisement