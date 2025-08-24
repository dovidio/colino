# Colino 📰

Your own hackable aggregator and AI-powered digest generator.

## What is Colino?

Colino is a simple, powerful, and completely free feed aggregator that lets you create your own personalized news digest from any RSS-enabled website. Additionally, Colino leverages LLM to generate a digest of the latest news
Currently, it supports the following sources:
- RSS feed
- Youtube channel

## Status
Colino is in active development and apis and commands are expected to change quickly before the official release.

## Quick Start

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Configure:**
   ```bash
   # Copy and edit config file  
   cp config.yaml ~/.config/colino/config.yaml
   # Set OpenAI API key (secure)
   export OPENAI_API_KEY="your_openai_api_key_here"
   ```

2.1. ** Configuring Youtube Data source:**

To configure a youtube data source, you'll need access to OAuth 2.0 credentials.
To do so, follow these steps:
- Go to [Google Cloud Console](https://console.cloud.google.com/) and create a new project
- Enable the Youtube Data API v3
- Create OAuth 2.0 credentials
- Download the `credentials.json` file and store it in the config folder 

3. **Fetch RSS content:**
   ```bash
   python src/main.py fetch --source rss
   ```

4. **Generate AI digest:**
   ```bash
   python src/main.py digest --source rss
   ```

## Usage Examples

```bash
# Discover RSS feeds from any website
python src/main.py discover https://example.com

# Test a specific feed
python src/main.py test https://feeds.example.com/rss

# Fetch from specific feeds only
python src/main.py fetch --urls https://hnrss.org/frontpage

# Fetch from youtube
python src/main.py fetch --source youtube

# List recent posts with filtering
python src/main.py list --hours 48 --limit 20

# Generate AI-powered digest
python src/main.py digest --hours 24
python src/main.py digest --output daily_digest.md

# Generate AI-powered article digest
python src/main.py digest --post-id yt:video:zBWTiAss25E

# Export your feeds for backup
python src/main.py export --output my_feeds.opml

# Import feeds from OPML file
python src/main.py import my_feeds.opml
```

### Feedback and contribution

Create a new issue for any feedback, requests and contribution ideas. We can take it from there
