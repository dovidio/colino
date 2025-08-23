import feedparser
import requests
from datetime import datetime, timezone
from typing import List, Dict, Any
import logging
from urllib.parse import urljoin, urlparse
from config import Config


logger = logging.getLogger(__name__)

class RSSSource:
    def __init__(self):
        """Initialize RSS feed parser"""
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': Config.RSS_USER_AGENT
        })
    
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
                    
                    # Extract content
                    content = ""
                    if hasattr(entry, 'content') and entry.content:
                        content = entry.content[0].value
                    elif hasattr(entry, 'summary'):
                        content = entry.summary
                    elif hasattr(entry, 'description'):
                        content = entry.description
                    
                    # Clean up content (remove HTML tags for preview)
                    import re
                    content_preview = re.sub('<[^<]+?>', '', content)[:500]
                    
                    post_data = {
                        'id': entry.get('id', entry.get('link', '')),
                        'source': 'rss',
                        'author_username': feed_data['title'],
                        'author_display_name': feed_data['title'],
                        'content': content_preview,
                        'url': entry.get('link', ''),
                        'created_at': pub_date or datetime.now(timezone.utc),
                        'like_count': 0,  # Most RSS feeds don't have engagement metrics
                        'reply_count': 0,  # Could be populated if RSS feed includes comment counts
                        'metadata': {
                            'feed_url': feed_url,
                            'feed_title': feed_data['title'],
                            'entry_title': entry.get('title', ''),
                            'entry_author': entry.get('author', ''),
                            'entry_tags': [tag.term for tag in getattr(entry, 'tags', [])],
                            'full_content': content
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
    
    def discover_feed_url(self, website_url: str) -> List[str]:
        """Try to discover RSS feed URLs from a website"""
        feed_urls = []
        
        try:
            response = self.session.get(website_url, timeout=Config.RSS_TIMEOUT)
            response.raise_for_status()
            
            # Look for feed links in HTML
            import re
            from html.parser import HTMLParser
            
            class FeedLinkParser(HTMLParser):
                def __init__(self):
                    super().__init__()
                    self.feed_urls = []
                
                def handle_starttag(self, tag, attrs):
                    if tag.lower() == 'link':
                        attrs_dict = dict(attrs)
                        rel = attrs_dict.get('rel', '').lower()
                        if rel in ['alternate', 'feed']:
                            type_val = attrs_dict.get('type', '').lower()
                            if 'rss' in type_val or 'atom' in type_val or 'xml' in type_val:
                                href = attrs_dict.get('href')
                                if href:
                                    # Convert relative URLs to absolute
                                    feed_url = urljoin(website_url, href)
                                    self.feed_urls.append(feed_url)
            
            parser = FeedLinkParser()
            parser.feed(response.text)
            feed_urls.extend(parser.feed_urls)
            
            # Try common feed URLs
            base_domain = f"{urlparse(website_url).scheme}://{urlparse(website_url).netloc}"
            common_paths = ['/feed', '/rss', '/atom.xml', '/rss.xml', '/feed.xml', '/feeds/all.atom.xml']
            
            for path in common_paths:
                potential_url = urljoin(base_domain, path)
                try:
                    test_response = self.session.head(potential_url, timeout=10)
                    if test_response.status_code == 200:
                        content_type = test_response.headers.get('content-type', '').lower()
                        if 'xml' in content_type or 'rss' in content_type or 'atom' in content_type:
                            feed_urls.append(potential_url)
                except:
                    pass
            
        except Exception as e:
            logger.error(f"Error discovering feeds for {website_url}: {e}")
        
        # Remove duplicates
        feed_urls = list(set(feed_urls))
        logger.info(f"Discovered {len(feed_urls)} feed URLs for {website_url}")
        
        return feed_urls 
