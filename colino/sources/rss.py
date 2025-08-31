import feedparser
import requests
from datetime import datetime, timezone
from typing import List, Dict, Any
import logging
from ..config import Config
from ..scraper import ArticleScraper
from ..db import Database


logger = logging.getLogger(__name__)

class RSSSource:
    def __init__(self, db: Database = None):
        """Initialize RSS feed parser"""
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': Config.RSS_USER_AGENT
        })
        self.scraper = ArticleScraper()
        self.db = db
    
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
    
    def get_recent_posts(self, feed_urls: List[str], since_time: datetime = None) -> List[Dict[str, Any]]:
        """Get recent posts from multiple RSS feeds"""
        all_posts = []
        
        for feed_url in feed_urls:
            try:
                feed_data = self.parse_feed(feed_url)
                if not feed_data:
                    continue
                
                for entry in feed_data['entries']:
                    # Parse publication date
                    pub_date = None
                    if hasattr(entry, 'published_parsed') and entry.published_parsed:
                        pub_date = datetime(*entry.published_parsed[:6], tzinfo=timezone.utc)
                    elif hasattr(entry, 'updated_parsed') and entry.updated_parsed:
                        pub_date = datetime(*entry.updated_parsed[:6], tzinfo=timezone.utc)
                    
                    # Skip if too old
                    if since_time and pub_date and pub_date < since_time:
                        continue
                    
                    # Create unique ID for the post
                    post_id = entry.get('id', entry.get('link', ''))
                    if not post_id:
                        continue
                    
                    # Check if we already have this content in cache
                    if self.db:
                        existing_content = self.db.get_content_by(post_id)
                        if existing_content:
                            logger.debug(f"Content already exists for {post_id}, using cached version")
                            continue
                    
                    # Extract initial content from RSS
                    rss_content = ""
                    if hasattr(entry, 'content') and entry.content:
                        rss_content = entry.content[0].value
                    elif hasattr(entry, 'summary'):
                        rss_content = entry.summary
                    elif hasattr(entry, 'description'):
                        rss_content = entry.description
                    
                    # Try to scrape full article content if URL available and scraping enabled
                    full_content = rss_content
                    article_url = entry.get('link', '') 
                    
                    scraped_content = self.scraper.scrape_article_content(article_url)
                    if scraped_content and len(scraped_content) > len(rss_content):
                        full_content = full_content + "\nFull Content:\n" + scraped_content
                        logger.info(f"Using scraped content for: {entry.get('title', 'Unknown')}")
                    
                    # Clean up content for preview (remove HTML tags)
                    import re
                    content_preview = re.sub('<[^<]+?>', '', rss_content)[:500]
                    
                    post_data = {
                        'id': post_id,
                        'source': 'rss',
                        'author_username': feed_data['title'],
                        'author_display_name': feed_data['title'],
                        'content': full_content,  # Store full scraped content
                        'url': article_url,
                        'created_at': pub_date or datetime.now(timezone.utc),
                        'like_count': 0,
                        'reply_count': 0,
                        'metadata': {
                            'feed_url': feed_url,
                            'feed_title': feed_data['title'],
                            'entry_title': entry.get('title', ''),
                            'entry_author': entry.get('author', ''),
                            'entry_tags': [tag.term for tag in getattr(entry, 'tags', [])],
                            'rss_content': rss_content,  # Keep original RSS content as backup
                            'content_preview': content_preview
                        }
                    }
                    
                    all_posts.append(post_data)
                    
            except Exception as e:
                logger.error(f"Error processing feed {feed_url}: {e}")
                continue
        
        # Sort by publication date (newest first)
        all_posts.sort(key=lambda x: x['created_at'], reverse=True)
        
        logger.info(f"Retrieved {len(all_posts)} RSS posts from {len(feed_urls)} feeds")
        return all_posts 
