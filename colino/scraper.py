import logging
from concurrent.futures import ThreadPoolExecutor, as_completed

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

    def scrape_articles_parallel(
        self, urls: list[str], max_workers: int = 5
    ) -> dict[str, str | None]:
        """Scrape multiple articles in parallel

        Args:
            urls: List of URLs to scrape
            max_workers: Maximum number of concurrent scraping threads

        Returns:
            Dictionary mapping URLs to their scraped content (or None if failed)
        """
        if not urls:
            return {}

        results: dict[str, str | None] = {}

        logger.info(
            f"Starting parallel scraping of {len(urls)} articles with {max_workers} workers"
        )

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            # Submit all scraping tasks
            future_to_url = {
                executor.submit(self.scrape_article_content, url): url for url in urls
            }

            # Collect results as they complete
            for future in as_completed(future_to_url):
                url = future_to_url[future]
                try:
                    content = future.result()
                    results[url] = content
                    if content:
                        logger.debug(f"Successfully scraped {url}")
                    else:
                        logger.debug(f"No content extracted from {url}")
                except Exception as e:
                    logger.warning(f"Error in parallel scraping of {url}: {e}")
                    results[url] = None

        successful_scrapes = sum(
            1 for content in results.values() if content is not None
        )
        logger.info(
            f"Parallel scraping completed: {successful_scrapes}/{len(urls)} successful"
        )

        return results
