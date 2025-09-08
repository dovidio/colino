import logging

import requests
import trafilatura

from .config import config

logger = logging.getLogger(__name__)


class ArticleScraper:
    """Scrapes and extracts full content from web articles"""

    def __init__(self) -> None:
        self.session = requests.Session()
        self.session.headers.update(
            {"User-Agent": "Mozilla/5.0 (compatible; Colino RSS Reader/1.0)"}
        )

    def scrape_article_content(self, url: str) -> str | None:
        """Scrape and extract main content from a web page"""
        try:
            logger.info(f"Scraping content from: {url}")

            response = self.session.get(url, timeout=config.RSS_TIMEOUT)
            response.raise_for_status()

            # Use trafilatura to extract main content
            content: str | None = trafilatura.extract(
                response.text,
                include_comments=False,
                include_tables=True,
                include_formatting=False,
                output_format="txt",
            )

            if content and len(content) > 100:  # Only use if we got substantial content
                # Clean up whitespace
                content = " ".join(content.split())
                logger.info(f"Scraped {len(content)} characters from {url}")
                return content
            else:
                logger.debug(
                    f"Scraped content too short from {url}, keeping RSS content"
                )
                return None

        except requests.exceptions.RequestException as e:
            logger.warning(f"Network error scraping {url}: {e}")
            return None
        except Exception as e:
            logger.warning(f"Could not scrape content from {url}: {e}")
            return None
