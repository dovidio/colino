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

### Daily Workflow

**Morning Review** (5 minutes):
```bash
./colino list                # See what's new
```

**Content Update** (runs automatically):
```bash
./colino daemon              # Manual refresh if needed
```

**AI Analysis** (when you want to dive deeper):
```bash
./colino server              # Start MCP server
```

Then use your AI assistant to ask questions like:
- "What are the main themes in yesterday's tech news?"
- "Summarize articles about AI developments"
- "Find content about renewable energy"

## Choosing Your Sources

### Quality Over Quantity
Start with a curated list of sources that provide real value. Here are some categories to consider:

**Technology & Research:**
- Hacker News RSS (https://hnrss.org/frontpage)
- Ars Technica (https://feeds.arstechnica.com/arstechnica/index)
- MIT Technology Review (https://www.technologyreview.com/feed/)

**Business & Finance:**
- Stratechery (https://stratechery.com/feed/)
- The Atlantic Business (https://www.theatlantic.com/feed/business/)

**Science & Learning:**
- Nature News (https://www.nature.com/news/rss)
- ScienceDaily (https://www.sciencedaily.com/rss/all.xml)

### Testing New Sources
Before adding a source to your permanent list, test it:

```bash
./colino digest "https://example.com/article-url"
```

This shows you exactly what content Colino will extract from that source.

## Working with Your Content

### Quick Browser Checks
Use the list command to see what's new without opening your AI client:

```bash
./colino list                    # Last 24 hours
./colino list --hours 168       # Last week
```

### Deep Analysis with AI
When you want to understand trends or dive deep into topics:

1. Start the MCP server: `./colino server`
2. Use your AI client with these tools:
   - `list_cache()` - Discover recent content
   - `get_content()` - Fetch full articles for analysis

**Example AI Prompts:**
- "What are the emerging trends in my RSS feeds from the past week?"
- "Summarize all articles about climate change published in the last 3 days"
- "Find connections between today's tech news and business developments"

### Managing Content Overload
If you're getting too much content:

1. **Review Your Sources**: Remove feeds that consistently provide low-value content
2. **Adjust Time Windows**: Use shorter time periods when querying your AI
3. **Filter by Source**: Focus on specific feeds when doing research

## Advanced Workflows

### Research Projects
When researching a specific topic:

1. Add specialized RSS feeds related to your research area
2. Let Colino collect content for a few days
3. Use AI to analyze patterns and extract key insights

### Industry Monitoring
For professional development:

1. Include industry publications and company blogs
2. Set up daily or weekly check-ins with your AI assistant
3. Ask for summaries of important developments and trends

### Learning Paths
For skill development:

1. Add educational resources and tutorial sites
2. Use AI to identify content matching your learning goals
3. Create personalized summaries and action items

## Automation Tips

### macOS Users
Take advantage of automatic background fetching:

```bash
./colino daemon schedule       # Set and forget
```

Your content will always be fresh and ready for analysis.

### Manual Scheduling
If you prefer more control:

```bash
# Morning routine
./colino daemon && ./colino list

# Weekly deep dive
./colino daemon && ./colino server
```

## Troubleshooting Common Issues

**No new content appearing?**
- Check your RSS feeds are still active
- Verify your internet connection
- Try running `./colino daemon` manually

**AI can't find content?**
- Ensure you've run ingestion at least once
- Try a longer time window (72+ hours)
- Check the MCP server is running correctly

**Too much content?**
- Reduce the number of RSS feeds
- Use more specific time windows
- Ask your AI to filter by keywords or sources
