# Usage

Colino is designed to be simple and efficient. Once installed, you can run it from your terminal with the command:

## Get started

```bash
colino digest
```

This command fetches the latest articles from your configured RSS feeds and YouTube subscriptions, summarizes them using AI, and displays the summaries in your terminal. The first time you run this, you'll need to authenticate with Youtube to allow Colino to access your subscriptions. After that it's gonna be smooth sailing.

## Filtering Options
You can also digest only RSS feeds.
```bash
colino digest --rss
```

Or only videos from your YouTube subscriptions.
```bash
colino digest --youtube
```

By default Colino will look into articles and videos from the last 24 hours. You can change this with the `--hours` flag.
```bash
colino digest --hours 48
```

## Listing content
You can also list all the feeds you have configured.
```bash
colino list
```

## Summarizing single articles or videos

Want to summarize a single article? Just do the following
```bash
colino digest https://arxiv.org/html/1706.03762v7
```

This works also for a single YouTube video
```bash
colino digest https://www.youtube.com/watch?v=eMlx5fFNoYc
```

## Ingesting content
By default the digest command is also scraping/ingesting content, but if you want to just ingest without summarizing, you can do
```bash
colino ingest
```

## Getting help
For more options, run:
```bash
colino --help
```
