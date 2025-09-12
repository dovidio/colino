import os
from pathlib import Path
from typing import Any, cast

import yaml


class Config:
    def __init__(self) -> None:
        self._config = self._load_config()

    def _load_config(self) -> dict[str, Any]:
        """Load configuration from YAML files"""
        # Check config locations in order of preference
        config_paths = [
            Path.home() / ".config" / "colino" / "config.yaml",
            Path("config.yaml"),
        ]

        for config_path in config_paths:
            if config_path.exists():
                with open(config_path) as f:
                    return cast(dict[str, Any], yaml.safe_load(f))

        raise ValueError(
            "No config file found. Please create a config.yaml file in the current directory or in ~/.config/colino/"
        )

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
    def YOUTUBE_PROXY_WEBSHARE_ENDPOINT(self) -> str:
        return cast(
            str,
            self._config.get("youtube", {})
            .get("proxy", {})
            .get("webshare", {})
            .get("endpoint", "rotating-residential.webshare.io:9000"),
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
