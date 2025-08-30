# Colino 📰

Your own hackable aggregator and AI-powered digest generator.

## What is Colino?

Colino is a simple, powerful, and completely free feed aggregator that lets you create your own personalized news digest from any RSS-enabled website. Additionally, Colino leverages LLM to generate a digest of the latest news
Currently, it supports the following sources:
- RSS feed
- Youtube channel

## Status
Colino is in active development and apis and commands are expected to change quickly before the official release.

## Setup project

The project uses pyenv and poetry to manage python version and dependencies

1. **Use correct python version**
   ```bash
   pyenv local
   ```

2. **Install dependencies:**
   ```bash
   poetry install
   ```


3. **Fetch RSS content:**
   ```bash
   poetry run colino fetch --source rss
   ```

4. **Generate AI digest:**
   ```bash
   poetry run colino digest --source rss
   ```

## Configuring the Youtube Data source

Currently the Youtube data source relies on the Youtube Api to fetch users subscriptions.
To use the Youtube Api, you'll need access to OAuth 2.0 credentials.
To do so, follow these steps:
- Go to [Google Cloud Console](https://console.cloud.google.com/) and create a new project
- Enable the Youtube Data API v3
- Create OAuth 2.0 credentials
- Download the `credentials.json` file and store it in the config folder 

## Install colino as command line tool 

```bash
poetry build
pipx install dist/colino-0.1.0-py3-none-any.whl
```

## Usage Examples

```bash
# Discover RSS feeds from any website
colino discover https://example.com

# Test a specific feed
colino test https://feeds.example.com/rss

# Fetch from specific feeds only
colino fetch --urls https://hnrss.org/frontpage

# Fetch from youtube
colino fetch --source youtube

# List recent posts with filtering
colino list --hours 48 --limit 20

# Generate AI-powered digest
colino digest --hours 24
colino digest --output daily_digest.md

# Generate AI-powered article digest
colino digest --post-id yt:video:zBWTiAss25E

# Generate AI-powered video digest from a youtube video
colino digest video --youtube-video-url https://www.youtube.com/watch?v=vQJKtTXkpCI

# Export your feeds for backup
colino export --output my_feeds.opml

# Import feeds from OPML file
colino import my_feeds.opml
```

### Feedback and contribution

Create a new issue for any feedback, requests and contribution ideas. We can take it from there
