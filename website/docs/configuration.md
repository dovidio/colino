# Configuration

Colino uses a YAML configuration file (`config.yaml`) to control its behavior. Colino will search the following paths for the config file:
- `~/.config/colino/config.yaml`
- `./config.yaml`

## RSS Configuration
The `rss` section controls how Colino fetches and processes RSS feeds.
```yaml
rss:
    feeds:
        - https://hnrss.org/frontpage
    user_agent: "Colino RSS Reader 1.0.0"
    timeout: 30
    max_posts_per_feed: 100
    scraper_max_workers: 5
```
Feeds can be passed as a list of URLs. Other options include
- **user_agent**: Custom user agent string for HTTP requests.
- **timeout**: Timeout (in seconds) for fetching feeds.
- **max_posts_per_feed**: Maximum number of posts to fetch per feed.
- **scraper_max_workers**: Number of parallel threads for scraping article content.

## Filtering
The `filters` section can be used to include or exclude posts based on keywords:
```yaml
filters:
  include_keywords: []  # Only show posts with these words
  exclude_keywords:
    - ads
    - sponsored
    - advertisement
```
- **include_keywords**: Only include posts containing these keywords.
- **exclude_keywords**: Exclude posts containing these keywords.

## YouTube Configuration
The `youtube` section controls how Colino fetches and processes YouTube subscriptions.
```yaml
youtube:
  transcript_languages:
    - en
    - it
  proxy:
    enabled: false
    webshare:
      username: "your_username"
      password: "your_password"
```

### Webshare Proxy Configuration
The `proxy` section is optional and can be used to configure a rotating proxy for fetching YouTube transcripts. This is useful for avoiding rate limits when fetching many transcripts.
Webshare.io is a popular proxy provider that offers rotating residential proxies. Here's a referral link to sign up: [https://www.webshare.io](https://www.webshare.io/?referral_code=vtgc5gn0jdhg)
- **enabled**: Set to `true` to enable proxy usage.
- **webshare**: (Optional) Credentials for webshare.io proxies.
  - **username**: Your webshare.io username.
  - **password**: Your webshare.io password.

## AI Configuration
The `ai` section configures the AI model and summarization behavior.
```yaml
ai:
    model: "gpt-5-mini"
    extract_web_content: true
    auto_save: true
    save_directory: "digests"
    prompt: |
        You are an expert news curator and summarizer. Create concise, insightful summaries of news articles and blog posts. Focus on:
        1. Key insights and takeaways
        2. Important facts and developments
        3. Implications and context
        4. Clear, engaging writing

        Format your response in clean markdown with headers and bullet points.

        Please create a comprehensive digest summary of these {{ article_count }} recent articles/posts:

        {% for article in articles %}
        ## Article {{ loop.index }}: {{ article.title }}
        **Source:** {{ article.source }} | **Published:** {{ article.published }}
        **URL:** {{ article.url }}

        **Content:**
        {{ article.content[:1500] }}{% if article.content|length > 1500 %}...{% endif %}

        ---
        {% endfor %}

        If any of the previous articles don't have any meat, and they feel very clickbaity, make a note. We'll share a list later.

        Please provide:
        1. **Executive Summary** - 2-3 sentences covering the main themes across all {{ article_count }} articles
        2. **Key Highlights** - Bullet points of the most important developments (include most articles)
        3. **Notable Insights** - Interesting patterns, trends, or implications you see
        4. **Article Breakdown** - Brief one-line summary for each of the {{ article_count }} article together with the link of the article and the source.
        5. **Top Recommendations** - Which 3-4 articles deserve the deepest attention and why. Add the link to the article.
        6. **Purge candidates** - List the articles that are not very novel and that you suggest to remove, and the reason why.

        Keep it concise but comprehensive. Use clear markdown formatting. Do not offer any follow up help.

    article_prompt: |
        You are an expert news curator and summarizer.
        Create an insightful summary of the article content below.
        The content can come from news articles, youtube videos transcripts or blog posts.
        Format your response in clean markdown with headers and bullet points if required.

        ## Article {{ article.title }}
        **Source:** {{ article.source }} | **Published:** {{ article.published }}
        **URL:** {{ article.url }}

        **Content:**
        {{ article.content[:10000] }}{% if article.content|length > 10000 %}...{% endif %}

  youtube_prompt: |
    You are an expert video summarizer.
    I'm going to send you the transcript of a video and you'll summarize it for me.
    Format your response in clean markdown with headers and bullet points if required.

    Video transcript: {{ transcript }}

    If possible, find links to existing theories, philosophic positions, trends and suggest possible follow ups. If not, don't mention any of that.
    Do not offer any follow up help.
```

Prompts support Jinja-style templating with the following variables:
- **article_count**: Number of articles being summarized.
- **articles**: List of articles with `title`, `source`, `published`,
    `url`, and `content` fields.
- **transcript**: Transcript text of the YouTube video.
- **model**: The LLM model to use for summarization (e.g., `gpt-5-mini`).
- **extract_web_content**: If `true`, Colino will extract full web content for summarization.
- **auto_save**: If `true`, digests are automatically saved to disk.
- **save_directory**: Directory where digests are saved.
- **prompt**: Custom prompt for digest generation (supports Jinja-style variables).
- **article_prompt**: Custom prompt for single-article summaries.
- **youtube_prompt**: Custom prompt for YouTube transcript summaries.
