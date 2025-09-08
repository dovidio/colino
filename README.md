# Colino 📰

Your own hackable aggregator and AI-powered digest generator.

## What is Colino?

Colino is a simple, powerful, and completely free feed aggregator that lets you create your own personalized news digest from any RSS-enabled website. Additionally, Colino leverages LLM to generate a digest of the latest news
Currently, it supports the following sources:
- RSS feed
- Your youtube subscriptions

## Status
Colino is in active development and apis and commands are expected to change before the official release.

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

3. **Digest content using AI:**
By default the digest command will use the last 24 hours of content from all configured sources.
   ```bash
   poetry run colino digest
   ```
You can also pass a single url to digest a specific article or video:
   ```bash
   poetry run colino digest <a url>
   ```
Or you can choose to digest only content from a specific source:
   ```bash
   poetry run colino digest --youtube
   ```

## Install colino as command line tool

```bash
poetry build
pipx install dist/colino-*.whl

```

### Feedback and contribution

Create a new issue for any feedback, requests and contribution ideas. We can take it from there
