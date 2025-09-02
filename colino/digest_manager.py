"""
Digest Manager - Handles all digest operations for Colino
"""

import logging
import os
from datetime import UTC, datetime, timedelta
from typing import Any

from .config import config
from .db import Database
from .ingest_manager import IngestManager
from .sources.rss import RSSSource
from .sources.youtube import YouTubeSource
from .summarize import DigestGenerator

logger = logging.getLogger(__name__)


class DigestManager:
    """Manages all digest operations including URL-based, post-based, and recent articles"""

    def __init__(self, db: Database | None = None):
        self.db = db or Database()
        self.digest_generator = DigestGenerator()

    def digest_url(self, url: str, output_file: str | None = None) -> bool:
        """
        Digest content from a specific URL

        Args:
            url: The URL to digest (YouTube video or website)
            output_file: Optional path to save digest to file

        Returns:
            bool: True if successful, False otherwise
        """
        logger.info(f"Processing digest for URL: {url}")

        try:
            # First, try to find content by ID (the URL itself might be the ID)
            existing_content = self.db.get_content_by(url)

            # If not found by ID, try to find by URL
            if not existing_content:
                existing_content = self.db.get_content_by_url(url)

            if existing_content:
                return self._digest_cached_content(existing_content, output_file)
            else:
                return self._digest_fresh_url(url, output_file)

        except ValueError as e:
            self._handle_api_error(e)
            return False
        except Exception as e:
            logger.error(f"Error processing URL digest: {e}")
            print(f"❌ Error processing URL: {e}")
            return False

    def digest_recent_articles(
        self,
        hours: int | None = None,
        output_file: str | None = None,
        source: str | None = None,
        auto_ingest: bool = True,
        limit: int | None = None,
    ) -> bool:
        """
        Generate digest of recent articles

        Args:
            hours: Hours to look back (uses config default if None)
            output_file: Optional path to save digest to file
            source: Filter by source ('rss' or 'youtube')
            auto_ingest: Whether to automatically ingest recent content before digesting
            limit: Maximum number of articles to include in digest

        Returns:
            bool: True if successful, False otherwise
        """
        hours = hours or config.DEFAULT_LOOKBACK_HOURS
        since_time = datetime.now(UTC) - timedelta(hours=hours)

        # Auto-ingest recent content if enabled
        if auto_ingest:
            self._auto_ingest_recent_content(source, hours)

        source_filter = f" from {source}" if source else ""
        logger.info(
            f"Generating digest for articles{source_filter} from last {hours} hours"
        )

        try:
            # Get recent posts from database
            posts = self.db.get_content_since(since_time, source=source)

            if not posts:
                print(f"❌ No posts found{source_filter} from the last {hours} hours")
                print("   Try running 'colino ingest' first or increase --hours")
                return False

            # Apply source balancing and limit
            posts = self._balance_and_limit_posts(posts, limit, source)

            print(
                f"🤖 Generating AI digest for {len(posts)} recent articles{source_filter}..."
            )
            print(f"   Using model: {config.LLM_MODEL}")

            # Generate digest
            digest_content = self.digest_generator.summarize_articles(posts, limit)

            # Auto-save if enabled or output_file specified
            if config.AI_AUTO_SAVE:
                if not output_file:
                    # Auto-generate filename
                    os.makedirs(config.AI_SAVE_DIRECTORY, exist_ok=True)
                    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                    source_suffix = f"_{source}" if source else ""
                    output_file = f"{config.AI_SAVE_DIRECTORY}/digest{source_suffix}_{timestamp}.md"

                with open(output_file, "w", encoding="utf-8") as f:
                    f.write(digest_content)
                print(f"✅ Digest saved to {output_file}")

            # Always show digest in console unless explicitly saving to file
            if not output_file or config.AI_AUTO_SAVE:
                print("\n" + "=" * 60)
                print(digest_content)
                print("=" * 60)

            return True

        except ValueError as e:
            self._handle_api_error(e)
            return False
        except Exception as e:
            logger.error(f"Error generating digest: {e}")
            print(f"❌ Error generating digest: {e}")
            return False

    def _digest_cached_content(
        self, existing_content: dict[str, Any], output_file: str | None = None
    ) -> bool:
        """Digest content that was found in cache"""
        print(
            f"✅ Found cached content for {existing_content.get('url', 'unknown URL')}"
        )
        print(f"   Source: {existing_content['source']}")
        print(f"   Cached at: {existing_content['fetched_at']}")

        # Generate digest from cached content
        digest_content = self.digest_generator.summarize_article(existing_content)

        # Save or display digest
        self._save_or_display_digest(
            digest_content, output_file, f"cached_{existing_content['source']}"
        )
        return True

    def _digest_fresh_url(self, url: str, output_file: str | None = None) -> bool:
        """Digest content from a fresh URL (not in cache)"""
        print(f"🔍 Content not found in cache for {url}")

        # Check if it's a YouTube URL
        youtube_source = YouTubeSource()
        video_id = youtube_source.extract_video_id(url)

        if video_id:
            return self._digest_fresh_youtube(video_id, output_file)
        else:
            return self._digest_fresh_website(url, output_file)

    def _digest_fresh_youtube(
        self, video_id: str, output_file: str | None = None
    ) -> bool:
        """Digest a fresh YouTube video"""
        print(f"📺 Detected YouTube video: {video_id}")
        print("🎬 Fetching transcript...")

        youtube_source = YouTubeSource()
        transcript = youtube_source.get_video_transcript(video_id)

        if not transcript:
            print("❌ No transcript available for this video")
            return False

        print(f"✅ Extracted transcript ({len(transcript)} characters)")

        # Generate digest from transcript
        digest_content = self.digest_generator.summarize_video(transcript)

        # Save or display digest
        self._save_or_display_digest(digest_content, output_file, "youtube_video")
        return True

    def _digest_fresh_website(self, url: str, output_file: str | None = None) -> bool:
        """Digest a fresh website URL"""
        print("🌐 Detected website URL")
        print("📄 Fetching and processing content...")

        # For regular websites, use RSS scraper to get content
        rss_source = RSSSource()
        scraped_content = rss_source.scraper.scrape_article_content(url)

        if not scraped_content:
            print("❌ Could not extract content from the website")
            return False

        print(f"✅ Extracted content ({len(scraped_content)} characters)")

        # Create article data structure for digest
        article_data = {
            "title": "Scraped Article",
            "feed_title": "",
            "content": scraped_content,
            "url": url,
            "source": "website",
            "published": datetime.now().isoformat(),
        }

        # Generate digest
        digest_content = self.digest_generator.generate_llm_article_digest(article_data)

        # Save or display digest
        self._save_or_display_digest(digest_content, output_file, "website")
        return True

    def _save_or_display_digest(
        self, digest_content: str, output_file: str | None = None, source_type: str = ""
    ) -> None:
        """Helper function to save or display digest content"""
        # Auto-save if enabled or output_file specified
        if output_file or config.AI_AUTO_SAVE:
            if not output_file:
                # Auto-generate filename
                os.makedirs(config.AI_SAVE_DIRECTORY, exist_ok=True)
                timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                source_suffix = f"_{source_type}" if source_type else ""
                output_file = (
                    f"{config.AI_SAVE_DIRECTORY}/digest{source_suffix}_{timestamp}.md"
                )

            with open(output_file, "w", encoding="utf-8") as f:
                f.write(digest_content)
            print(f"✅ Digest saved to {output_file}")

        # Always show digest in console unless explicitly saving to file
        if not output_file or config.AI_AUTO_SAVE:
            print("\n" + "=" * 60)
            print(digest_content)
            print("=" * 60)

    def _handle_api_error(self, e: ValueError) -> None:
        """Handle API-related errors"""
        if "openai_api_key" in str(e) or "Incorrect API key" in str(e):
            print("❌ OpenAI API key not configured or invalid")
            print("   Set environment variable: export OPENAI_API_KEY='your_key_here'")
            print("   Get one from: https://platform.openai.com/api-keys")
        else:
            print(f"❌ Configuration error: {e}")

    def _auto_ingest_recent_content(
        self, source: str | None = None, hours: int | None = None
    ) -> None:
        """
        Automatically ingest recent content before digesting

        Args:
            source: Source to ingest from ('rss', 'youtube', or None for all)
            hours: Hours to look back for ingestion
        """
        print("🔄 Auto-ingesting recent content before generating digest...")

        # Determine which sources to ingest
        if source:
            sources = [source]
        else:
            sources = ["rss", "youtube"]

        # Use IngestManager to ingest recent content
        ingest_manager = IngestManager(db=self.db)
        ingested_posts = ingest_manager.ingest(sources, hours)

        if ingested_posts:
            print(f"✅ Auto-ingested {len(ingested_posts)} recent posts")
        else:
            print("📋 No new posts to ingest")

    def _balance_and_limit_posts(
        self, posts: list[dict[str, Any]], limit: int | None, source: str | None
    ) -> list[dict[str, Any]]:
        """
        Balance posts from different sources and apply limit.

        When no specific source is requested and we have posts from multiple sources,
        try to distribute them evenly. Half from one source, half from the other.

        Args:
            posts: List of posts to balance and limit
            limit: Maximum number of posts to return (None means no limit)
            source: Source filter ('rss', 'youtube', or None for all)

        Returns:
            List of balanced and limited posts
        """
        # If no limit provided, return all posts
        if limit is None:
            return posts

        max_articles = limit

        # If a specific source is requested or we don't need balancing, just limit
        if source or len(posts) <= max_articles:
            return posts[:max_articles]

        # Check if we have multiple sources
        sources_in_posts = {post.get("source") for post in posts}

        # If only one source present, just limit
        if len(sources_in_posts) <= 1:
            return posts[:max_articles]

        # Group posts by source
        posts_by_source: dict[str, list[dict[str, Any]]] = {}
        for post in posts:
            post_source = post.get("source") or "unknown"
            if post_source not in posts_by_source:
                posts_by_source[post_source] = []
            posts_by_source[post_source].append(post)

        # Calculate how many posts to take from each source
        num_sources = len(posts_by_source)
        posts_per_source = max_articles // num_sources
        remainder = max_articles % num_sources

        balanced_posts = []
        source_names = sorted(posts_by_source.keys())  # Sort for consistent ordering

        for i, source_name in enumerate(source_names):
            source_posts = posts_by_source[source_name]
            # Give remainder posts to first sources
            source_limit = posts_per_source + (1 if i < remainder else 0)
            balanced_posts.extend(source_posts[:source_limit])

        logger.info(f"Balanced {len(balanced_posts)} posts across {num_sources} sources (limit: {max_articles})")

        # Sort by creation time to maintain chronological order
        balanced_posts.sort(key=lambda x: x.get("created_at", ""), reverse=True)

        return balanced_posts
