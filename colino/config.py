import os
from pathlib import Path
from typing import Any, cast

import yaml


class Config:
    def __init__(self) -> None:
        self._config = self._load_config()

    def _load_config(self) -> dict[str, Any]:
        """Load configuration from YAML files, or create a default one if missing."""
        config_paths = [
            Path.home() / ".config" / "colino" / "config.yaml",
            Path("config.yaml"),
        ]

        for config_path in config_paths:
            if config_path.exists():
                with open(config_path) as f:
                    return cast(dict[str, Any], yaml.safe_load(f))

        # If no config found, create default at ~/.config/colino/config.yaml
        # Set default DB path for macOS
        if os.name == "posix" and "darwin" in os.uname().sysname.lower():
            db_default_path = str(
                Path.home() / "Library" / "Application Support" / "Colino" / "colino.db"
            )
        else:
            db_default_path = "colino.db"

        default_config = {
            "rss": {
                "feeds": ["https://hnrss.org/frontpage"],
                "user_agent": "Colino RSS Reader 1.0.0",
                "timeout": 30,
                "max_posts_per_feed": 100,
                "scraper_max_workers": 5,
            },
            "daemon": {
                "enabled": False,
                "interval_minutes": 30,
                "sources": ["rss", "youtube"],
                "since_hours": 24,
                # Prefer a stable CLI if installed via pipx/brew
                # Set to "colino" to use the installed entrypoint; leave empty to auto-detect.
                "ingest_command": "",
                "log_file": str(
                    Path.home() / "Library" / "Logs" / "Colino" / "daemon.log"
                ),
            },
            "filters": {
                "include_keywords": [],
                "exclude_keywords": ["ads", "sponsored", "advertisement"],
            },
            "youtube": {
                "transcript_languages": ["en", "it"],
                "proxy": {"enabled": False},
            },
            "ai": {
                "model": "gpt-5-mini",
                "stream": False,
                "auto_save": True,
                "save_directory": "digests",
                "prompt": (
                    "You are an expert news curator and summarizer. Create concise, insightful summaries of news articles and blog posts. Focus on:\n"
                    "1. Key insights and takeaways\n"
                    "2. Important facts and developments\n"
                    "3. Implications and context\n"
                    "4. Clear, engaging writing\n\n"
                    "Format your response in clean markdown with headers and bullet points.\n\n"
                    "Please create a comprehensive digest summary of these {{ article_count }} recent articles/posts:\n\n"
                    "{% for article in articles %}\n"
                    "## Article {{ loop.index }}: {{ article.title }}\n"
                    "**Source:** {{ article.source }} | **Published:** {{ article.published }}\n"
                    "**URL:** {{ article.url }}\n\n"
                    "**Content:**\n"
                    "{{ article.content[:1500] }}{% if article.content|length > 1500 %}...{% endif %}\n\n"
                    "---\n"
                    "{% endfor %}\n\n"
                    "If any of the previous articles don't have any meat, and they feel very clickbaity, make a note. We'll share a list later.\n\n"
                    "Please provide:\n"
                    "1. **Executive Summary** - 2-3 sentences covering the main themes across all {{ article_count }} articles\n"
                    "2. **Key Highlights** - Bullet points of the most important developments (include most articles)\n"
                    "3. **Notable Insights** - Interesting patterns, trends, or implications you see\n"
                    "4. **Article Breakdown** - Brief one-line summary for each of the {{ article_count }} article together with the link of the article and the source.\n"
                    "5. **Top Recommendations** - Which 3-4 articles deserve the deepest attention and why. Add the link to the article.\n"
                    "6. **Purge candidates** - List the articles that are not very novel and that you suggest to remove, and the reason why.\n\n"
                    "Keep it concise but comprehensive. Use clear markdown formatting. Do not offer any follow up help."
                ),
                "article_prompt": (
                    "You are an expert news curator and summarizer.\n"
                    "Create an insightful summary of the article content below.\n"
                    "The content can come from news articles, youtube videos transcripts or blog posts.\n"
                    "Format your response in clean markdown with headers and bullet points if required.\n\n"
                    "## Article {{ article.title }}\n"
                    "**Source:** {{ article.source }} | **Published:** {{ article.published }}\n"
                    "**URL:** {{ article.url }}\n\n"
                    "**Content:**\n"
                    "{{ article.content[:10000] }}{% if article.content|length > 10000 %}...{% endif %}"
                ),
                "youtube_prompt": (
                    "You are an expert video summarizer.\n"
                    "I'm going to send you the transcript of a video and you'll summarize it for me.\n"
                    "Format your response in clean markdown with headers and bullet points if required.\n\n"
                    "Video transcript: {{ transcript }}\n\n"
                    "If possible, find links to existing theories, philosophic positions, trends and suggest possible follow ups. If not, don't mention any of that.\n"
                    "Do not offer any follow up help."
                ),
            },
            "database": {
                "path": db_default_path,
            },
            "general": {
                "default_lookback_hours": 24,
            },
        }

        config_dir = Path.home() / ".config" / "colino"
        config_dir.mkdir(parents=True, exist_ok=True)
        config_file = config_dir / "config.yaml"
        with open(config_file, "w") as f:
            yaml.safe_dump(default_config, f, sort_keys=False)
        print(f"No config file found. Created default config at {config_file}")
        return default_config

    # RSS Properties
    @property
    def RSS_FEEDS(self) -> list[str]:
        return cast(list[str], self._config.get("rss", {}).get("feeds", []))

    @property
    def RSS_USER_AGENT(self) -> str:
        return cast(
            str,
            self._config.get("rss", {}).get("user_agent", "Colino RSS Reader 1.0.0"),
        )

    @property
    def RSS_TIMEOUT(self) -> int:
        return cast(int, self._config.get("rss", {}).get("timeout", 30))

    @property
    def RSS_SCRAPER_MAX_WORKERS(self) -> int:
        """Maximum number of parallel workers for content scraping"""
        return cast(int, self._config.get("rss", {}).get("scraper_max_workers", 5))

    @property
    def MAX_POSTS_PER_FEED(self) -> int:
        return cast(int, self._config.get("rss", {}).get("max_posts_per_feed", 100))

    # YouTube Properties
    @property
    def YOUTUBE_OAUTH_PROXY_URL(self) -> str:
        return "https://colino.umberto.xyz"

    @property
    def YOUTUBE_TRANSCRIPT_LANGUAGES(self) -> list[str]:
        return cast(
            list[str],
            self._config.get("youtube", {}).get("transcript_languages", ["en", "auto"]),
        )

    @property
    def YOUTUBE_PROXY_ENABLED(self) -> bool:
        return cast(
            bool, self._config.get("youtube", {}).get("proxy", {}).get("enabled", False)
        )

    @property
    def YOUTUBE_PROXY_WEBSHARE_USERNAME(self) -> str | None:
        return cast(
            str | None,
            self._config.get("youtube", {})
            .get("proxy", {})
            .get("webshare", {})
            .get("username"),
        )

    @property
    def YOUTUBE_PROXY_WEBSHARE_PASSWORD(self) -> str | None:
        return cast(
            str | None,
            self._config.get("youtube", {})
            .get("proxy", {})
            .get("webshare", {})
            .get("password"),
        )

    @property
    def FILTER_KEYWORDS(self) -> list[str]:
        return cast(
            list[str], self._config.get("filters", {}).get("include_keywords", [])
        )

    @property
    def EXCLUDE_KEYWORDS(self) -> list[str]:
        return cast(
            list[str], self._config.get("filters", {}).get("exclude_keywords", [])
        )

    # AI Properties
    @property
    def OPENAI_API_KEY(self) -> str:
        # Always prioritize environment variable for security
        return cast(
            str,
            os.getenv("OPENAI_API_KEY")
            or self._config.get("ai", {}).get("openai_api_key"),
        )

    @property
    def LLM_MODEL(self) -> str:
        return cast(str, self._config.get("ai", {}).get("model", "gpt-5-mini"))

    @property
    def AI_STREAM(self) -> bool:
        return cast(bool, self._config.get("ai", {}).get("stream", False))

    @property
    def AI_AUTO_SAVE(self) -> bool:
        return cast(bool, self._config.get("ai", {}).get("auto_save", True))

    @property
    def AI_SAVE_DIRECTORY(self) -> str:
        return cast(str, self._config.get("ai", {}).get("save_directory", "digests"))

    @property
    def AI_PROMPT_TEMPLATE(self) -> str:
        return cast(str, self._config.get("ai", {}).get("prompt", ""))

    @property
    def AI_ARTICLE_PROMPT_TEMPLATE(self) -> str:
        return cast(str, self._config.get("ai", {}).get("article_prompt", ""))

    @property
    def AI_PROMPT_YOUTUBE(self) -> str:
        return cast(str, self._config.get("ai", {}).get("youtube_prompt", ""))

    # Database Properties
    @property
    def DATABASE_PATH(self) -> str:
        return cast(str, self._config.get("database", {}).get("path", "colino.db"))

    # General Properties
    @property
    def DEFAULT_LOOKBACK_HOURS(self) -> int:
        return cast(
            int, self._config.get("general", {}).get("default_lookback_hours", 24)
        )

    def validate_openai_config(self) -> bool:
        """Validate OpenAI API credentials"""
        if not self.OPENAI_API_KEY:
            raise ValueError(
                "Missing OPENAI_API_KEY environment variable. Get one from https://platform.openai.com/api-keys"
            )
        return True


# Create global config instance
config = Config()
