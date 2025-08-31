import feedparser
import requests
from datetime import datetime, timezone
from typing import List, Dict, Any
import logging
import re
from ..config import Config
from ..scraper import ArticleScraper
from ..db import Database
from .base import BaseSource


logger = logging.getLogger(__name__)

class RSSSource(BaseSource):
    """RSS feed source for fetching articles from RSS/Atom feeds"""
    
    def __init__(self, db: Database = None):
        """Initialize RSS feed parser"""
        super().__init__(db)
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': Config.RSS_USER_AGENT
        })
        self.scraper = ArticleScraper()
    
    @property
    def source_name(self) -> str:
        return 'rss'
    
    def parse_feed(self, feed_url: str) -> Dict[str, Any]:
        """Parse a single RSS feed"""
        try:
            logger.info(f"Parsing RSS feed: {feed_url}")
            
            # Download feed content
            response = self.session.get(feed_url, timeout=Config.RSS_TIMEOUT)
            response.raise_for_status()
            
            # Parse feed
            feed = feedparser.parse(response.content)
            
            if feed.bozo:
                logger.warning(f"Feed {feed_url} has parsing issues: {feed.bozo_exception}")
            
            return {
                'url': feed_url,
                'title': feed.feed.get('title', 'Unknown Feed'),
                'description': feed.feed.get('description', ''),
                'link': feed.feed.get('link', ''),
                'entries': feed.entries,
                'updated': feed.feed.get('updated_parsed'),
                'author': feed.feed.get('author', '')
            }
            
        except Exception as e:
            logger.error(f"Error parsing RSS feed {feed_url}: {e}")
            return None
    
    def get_recent_content(self, since_time: datetime = None) -> List[Dict[str, Any]]:
        """Get recent posts from RSS feeds - implementation of BaseSource abstract method"""
        feed_urls = Config.RSS_FEEDS
        return self.get_posts_from_feeds(feed_urls, since_time)
    
    def get_posts_from_feeds(self, feed_urls: List[str], since_time: datetime = None) -> List[Dict[str, Any]]:
        """Get recent posts from multiple RSS feeds"""
        all_posts = []
        
        for feed_url in feed_urls:
            try:
                feed_data = self.parse_feed(feed_url)
                if not feed_data:
                    continue
                
                for entry in feed_data['entries']:
                    post_data = self._process_rss_entry(entry, feed_data, feed_url, since_time)
                    if post_data:
                        all_posts.append(post_data)
                    
            except Exception as e:
                logger.error(f"Error processing feed {feed_url}: {e}")
                continue
        
        # Sort by publication date (newest first)
        all_posts.sort(key=lambda x: x['created_at'] or datetime.min.replace(tzinfo=timezone.utc), reverse=True)
        
        logger.info(f"Retrieved {len(all_posts)} RSS posts from {len(feed_urls)} feeds")
        return all_posts
    
    def _process_rss_entry(self, entry, feed_data: Dict[str, Any], feed_url: str, since_time: datetime = None) -> Dict[str, Any]:
        """Process a single RSS entry into a standardized post format"""
        # Parse publication date
        pub_date = self._parse_entry_date(entry)
        
        # Create unique ID for the post
        post_id = entry.get('id', entry.get('link', ''))
        if not post_id:
            return None
        
        article_url = entry.get('link', '')
        
        # Skip if should be filtered out
        if self._should_skip_post(article_url, since_time, pub_date):
            return None
        
        # Extract and enhance content
        rss_content = self._extract_rss_content(entry)
        full_content = self._enhance_content(rss_content, article_url, entry)
        
        # Clean up content for preview (remove HTML tags)
        content_preview = re.sub('<[^<]+?>', '', rss_content)[:500]
        
        # Create standardized post data
        return self._create_post_data(
            id=post_id,
            source='rss',
            author_username=feed_data['title'],
            author_display_name=feed_data['title'],
            content=full_content,
            url=article_url,
            created_at=pub_date or datetime.now(timezone.utc),
            metadata={
                'feed_url': feed_url,
                'feed_title': feed_data['title'],
                'entry_title': entry.get('title', ''),
                'entry_author': entry.get('author', ''),
                'entry_tags': [tag.term for tag in getattr(entry, 'tags', [])],
                'rss_content': rss_content,
                'content_preview': content_preview
            }
        )

    def _create_post_data(self, **kwargs) -> Dict[str, Any]:
        """
        Create a standardized post data structure
        
        This ensures all sources return posts in the same format
        """
        required_fields = {
            'id': kwargs.get('id', ''),
            'source': kwargs.get('source', self.source_name),
            'author_username': kwargs.get('author_username', ''),
            'author_display_name': kwargs.get('author_display_name', ''),
            'content': kwargs.get('content', ''),
            'url': kwargs.get('url', ''),
            'created_at': kwargs.get('created_at', datetime.now()),
            'like_count': kwargs.get('like_count', 0),
            'reply_count': kwargs.get('reply_count', 0),
            'metadata': kwargs.get('metadata', {})
        }
        
        return required_fields

    def _should_skip_post(self, url: str, since_time: datetime = None, pub_date: datetime = None) -> bool:
        """
        Common logic to determine if a post should be skipped
        
        Args:
            url: Post URL
            since_time: Time threshold for filtering
            pub_date: Publication date of the post
            
        Returns:
            True if the post should be skipped
        """
        # Skip if too old
        if since_time and pub_date and pub_date < since_time:
            logger.debug(f"Skipping old post: {url}")
            return True
        
        # Check if already in cache
        if self.db and self.db.get_content_by(url):
            logger.debug(f"Content already exists for {url}, skipping")
            return True
        
        if url and '/shorts/' in url:
            logger.debug(f"Skipping YouTube Shorts URL: {url}")
            return True
        
        return False
    
    def _extract_rss_content(self, entry) -> str:
        """Extract content from RSS entry in order of preference"""
        if hasattr(entry, 'content') and entry.content:
            return entry.content[0].value
        elif hasattr(entry, 'summary'):
            return entry.summary
        elif hasattr(entry, 'description'):
            return entry.description
        return ""
    
    def _parse_entry_date(self, entry) -> datetime:
        """Parse publication date from RSS entry"""
        if hasattr(entry, 'published_parsed') and entry.published_parsed:
            return datetime(*entry.published_parsed[:6], tzinfo=timezone.utc)
        elif hasattr(entry, 'updated_parsed') and entry.updated_parsed:
            return datetime(*entry.updated_parsed[:6], tzinfo=timezone.utc)
        return None
    
    def _enhance_content(self, rss_content: str, article_url: str, entry) -> str:
        """Enhance RSS content with scraped full article content if available"""
        full_content = rss_content
        
        scraped_content = self.scraper.scrape_article_content(article_url)
        if scraped_content and len(scraped_content) > len(rss_content):
            full_content = full_content + "\nFull Content:\n" + scraped_content
            logger.info(f"Using scraped content for: {entry.get('title', 'Unknown')}")
        
        return full_content
