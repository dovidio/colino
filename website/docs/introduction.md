# Introduction

## What is Colino?
Colino is a local content cache and MCP server that helps you stay informed by ingesting content from multiple sources (RSS feeds, articles, and YouTube links discovered via RSS) into a SQLite database on your machine. It then exposes this content to your LLM client via the Model Context Protocol (MCP).

Colino is privacy-first: data stays on your device, and there’s no user tracking. It’s open source, so you can inspect, modify, and contribute to the code. You choose how your LLM uses the data (summarization, search, analysis) through your client’s prompts and workflows.

## Why?
Social media and news sites rely heavily on images, videos, and distractions. This keeps you engaged longer, but often wastes your time and can have negative psychological effects [[1]](https://www.sciencedirect.com/science/article/pii/S2590291125005212).

The solution is to be intentional about the information you consume and use tools that help you filter out noise. Colino is one of those tools.

- Build your own curated list of high-signal, low-noise RSS sources
- Preview YouTube video content via transcripts to decide if it’s worth your time
- Use your LLM client to summarize, filter, and query your local cache via MCP
- Keep everything local and hackable
