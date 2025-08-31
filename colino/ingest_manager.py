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
from .sources.base import BaseSource

logger = logging.getLogger(__name__)


class IngestManager:
    """Manages all ingestion operations for different sources"""
    
    def __init__(self, db: Database = None):
        self.db = db or Database()
        
        self._sources = {
            'rss': RSSSource(db=self.db),
            'youtube': YouTubeSource(db=self.db)
        }
    
    def get_source(self, source_name: str) -> BaseSource:
        """Get a source instance by name"""
        return self._sources.get(source_name)
    
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
        since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
        since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
        
        all_posts = []
        
        for source_name in sources:
            source = self.get_source(source_name)
            if not source:
                logger.warning(f"Unknown source: {source_name}")
                print(f"⚠️  Unknown source: {source_name}")
                continue
            
            print(f"{source_name.upper()}: Fetching posts from {source_name}")
            
            try:
                recent_content = source.get_recent_content(since_time)
                
                # Apply content filtering if configured
                if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
                    recent_content = self._apply_content_filter(recent_content)
                
                saved_count = 0
                for content in recent_content:
                    if self.db.save_content(content):
                        saved_count += 1
                
                all_posts.extend(recent_content)
                print(f"✅ Fetched {len(recent_content)} posts from {source_name}")
                logger.info(f"Successfully saved {saved_count}/{len(recent_content)} {source_name} posts")
                
            except Exception as e:
                logger.error(f"Error fetching {source_name} posts: {e}")
                print(f"❌ Error fetching {source_name} posts: {e}")
        
        return all_posts
    
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
