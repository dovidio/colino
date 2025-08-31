"""
Ingest Manager - Handles all ingestion operations for Colino
"""

from datetime import datetime, timedelta, timezone
from typing import List, Dict, Any, Optional
import logging

from .config import Config
from .db import Database
from .sources.rss import RSSSource
from .sources.youtube import YouTubeSource

logger = logging.getLogger(__name__)


class IngestManager:
    """Manages all ingestion operations for different sources"""
    
    def __init__(self, db: Database = None):
        self.db = db or Database()
    
    def ingest(self, sources: List[str] = None, since_hours: int = None) -> List[Dict[str, Any]]:
        """
        Ingest content from specified sources
        
        Args:
            sources: List of source names ('rss', 'youtube'). Defaults to all sources.
            since_hours: Hours to look back. Uses config default if None.
            
        Returns:
            List of all ingested posts
        """
        sources = sources or ["rss", "youtube"]  # Default to all sources
        all_posts = []
        
        for source in sources:
            if source == "rss":
                print("📰 RSS: Fetching posts from RSS feeds")
                posts = self.ingest_rss(since_hours)
                all_posts.extend(posts)
                print(f"✅ Fetched {len(posts)} new posts from RSS feeds")
            elif source == "youtube":
                print("📺 YouTube: Fetching posts from YouTube subscriptions")
                posts = self.ingest_youtube(since_hours)
                all_posts.extend(posts)
                print(f"✅ Fetched {len(posts)} posts from YouTube")
            else:
                logger.warning(f"Unknown source: {source}")
                print(f"⚠️  Unknown source: {source}")
        
        return all_posts
    
    def ingest_rss(self, since_hours: int = None) -> List[Dict[str, Any]]:
        """
        Ingest posts from RSS feeds
        
        Args:
            since_hours: Hours to look back. Uses config default if None.
            
        Returns:
            List of ingested RSS posts
        """
        feed_urls = Config.RSS_FEEDS
        since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
        since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
        
        if not feed_urls:
            logger.warning("No RSS feeds configured")
            print("⚠️  No RSS feeds configured. Add feeds to your config.yaml file.")
            return []
        
        logger.info(f"Fetching RSS posts from {len(feed_urls)} feeds since {since_time}")
        
        rss_source = RSSSource(db=self.db)
        posts = rss_source.get_recent_posts(feed_urls, since_time)
        
        # Apply content filtering if configured
        if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
            posts = self._apply_content_filter(posts)

        # Save posts to database
        saved_count = 0
        for post in posts:
            if self.db.save_content(post):
                saved_count += 1
        
        logger.info(f"Successfully saved {saved_count}/{len(posts)} RSS posts")
        return posts

    def ingest_youtube(self, since_hours: int = None) -> List[Dict[str, Any]]:
        """
        Ingest posts from YouTube subscriptions
        
        Args:
            since_hours: Hours to look back. Uses config default if None.
            
        Returns:
            List of ingested YouTube posts
        """
        since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
        since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
        
        logger.info(f"Fetching YouTube posts since {since_time}")
        
        try:
            youtube_source = YouTubeSource()
            youtube_source.authenticate()

            if not youtube_source.is_authenticated:
                logger.error("YouTube authentication required")
                print("❌ YouTube authentication required. Run: colino authenticate --source youtube")
                return []
            
            channels = youtube_source.get_subscriptions()
            youtube_source.sync_subscriptions_to_db(channels, self.db)
            
            if not channels:
                logger.warning("No YouTube channels found")
                print("⚠️  No YouTube channels found. Make sure you're subscribed to some channels.")
                return []

            rss_source = RSSSource()
            youtube_posts = []
            
            for channel in channels:
                posts = rss_source.get_recent_posts([channel['rss_url']], since_time)
                for post in posts:
                    post['source'] = 'youtube'
                    post['metadata']['video_id'] = youtube_source.extract_video_id(post['url'])
                    post['metadata']['channel_id'] = channel['channel_id']
                    
                    # Check if we already have this content in the database
                    existing_content = self.db.get_content_by(post['url'])
                    if existing_content:
                        logger.debug(f"YouTube video already exists in cache: {post['url']}, skipping")
                        continue
                    
                    # Enhance with transcript only for new posts
                    post = youtube_source.enhance_youtube_post(post)
                    youtube_posts.append(post)

            if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
                youtube_posts = self._apply_content_filter(youtube_posts)

            saved_count = 0
            for post in youtube_posts:
                if self.db.save_content(post):
                    saved_count += 1

            logger.info(f"Successfully saved {saved_count}/{len(youtube_posts)} YouTube posts")
            return youtube_posts

        except Exception as e:
            logger.error(f"Error fetching YouTube posts: {e}")
            print(f"❌ Error fetching YouTube posts: {e}")
            return []

    def _apply_content_filter(self, posts: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Apply keyword filtering to posts
        
        Args:
            posts: List of posts to filter
            
        Returns:
            Filtered list of posts
        """
        filtered_posts = []
        
        for post in posts:
            content_text = f"{post['content']} {post.get('metadata', {}).get('entry_title', '')}".lower()
            
            # If filter keywords are set, only include posts that contain them
            if Config.FILTER_KEYWORDS:
                if not any(keyword.lower() in content_text for keyword in Config.FILTER_KEYWORDS if keyword.strip()):
                    continue
            
            # Exclude posts with exclude keywords
            if Config.EXCLUDE_KEYWORDS:
                if any(keyword.lower() in content_text for keyword in Config.EXCLUDE_KEYWORDS if keyword.strip()):
                    continue
            
            filtered_posts.append(post)
        
        if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
            logger.info(f"Content filtering: {len(filtered_posts)}/{len(posts)} posts kept")
        
        return filtered_posts
