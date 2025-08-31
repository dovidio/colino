#!/usr/bin/env python3
"""
Colino - RSS-Focused Social Digest
Your own hackable RSS feed aggregator and filter.
"""

import argparse
import logging
from logging.handlers import RotatingFileHandler
import os
from pathlib import Path
from datetime import datetime, timedelta, timezone
from typing import List, Optional
import re

from .config import Config
from .db import Database
from .sources.rss import RSSSource
from .sources.youtube import YouTubeSource
from .summarize import DigestGenerator

def setup_database():
    """Initialize the database"""
    get_logger().info("Setting up database...")
    db = Database()
    return db

def ingest(sources: List[str] = None, since_hours: int = None):
    """Ingest content from specified sources"""
    sources = sources or ["rss", "youtube"]  # Default to all sources
    all_posts = []
    
    for source in sources:
        if source == "rss":
            print("📰 RSS: Fetching posts from RSS feeds")
            posts = ingest_rss(since_hours)
            all_posts.extend(posts)
            print(f"✅ Fetched {len(posts)} new posts from RSS feeds")
        elif source == "youtube":
            print("📺 YouTube: Fetching posts from YouTube subscriptions")
            posts = fetch_youtube_posts(since_hours)
            all_posts.extend(posts)
            print(f"✅ Fetched {len(posts)} posts from YouTube")
        else:
            raise ValueError(f"Unknown source: {source}")
    
    return all_posts

def ingest_rss(since_hours: int = None):
    """Ingest posts from RSS feeds"""
    feed_urls = Config.RSS_FEEDS
    since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
    since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
    if not feed_urls:
        get_logger().warning("No RSS feeds configured. Add feeds to your .env file or provide URLs.")
        return []
    get_logger().info(f"Fetching RSS posts from {len(feed_urls)} feeds since {since_time}")
    db = setup_database()
    rss_source = RSSSource(db=db)  # Pass database instance
    posts = rss_source.get_recent_posts(feed_urls, since_time)
    # Apply content filtering if configured
    if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
        posts = apply_content_filter(posts)

    # Save posts to database
    saved_count = 0
    for post in posts:
        if db.save_content(post):
            saved_count += 1
    get_logger().info(f"Successfully saved {saved_count}/{len(posts)} RSS posts")
    return posts

def fetch_youtube_posts(since_hours: int = None):
    """Fetch posts from YouTube subscriptions"""
    since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
    since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
    
    get_logger().info(f"Fetching YouTube posts since {since_time}")
    
    try:
        youtube_source = YouTubeSource()
        youtube_source.authenticate()

        if not youtube_source.is_authenticated:
            get_logger().error("YouTube authentication required. Run: python src/main.py authenticate --source youtube")
            return []
        db = setup_database() 
        channels = youtube_source.get_subscriptions()
        youtube_source.sync_subscriptions_to_db(channels, db)
        if not channels:
            get_logger().warning("No YouTube channels found. Make sure you're subscribed to some channels.")
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
                existing_content = db.get_content_by(post['url'])
                if existing_content:
                    get_logger().debug(f"YouTube video already exists in cache: {post['url']}, skipping")
                    continue
                
                # Enhance with transcript only for new posts
                post = youtube_source.enhance_youtube_post(post)
                youtube_posts.append(post)

        if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
            youtube_posts = apply_content_filter(youtube_posts)

        saved_count = 0
        for post in youtube_posts:
            if db.save_content(post):
                saved_count += 1

        get_logger().info(f"Successfully saved {saved_count}/{len(youtube_posts)} YouTube posts")
        return youtube_posts

    except Exception as e:
        get_logger().error(f"Error fetching YouTube posts: {e}")
        return []

def apply_content_filter(posts: List[dict]) -> List[dict]:
    """Apply keyword filtering to posts"""
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
        get_logger().info(f"Content filtering: {len(filtered_posts)}/{len(posts)} posts kept")
    
    return filtered_posts

def discover_rss_feeds(website_url: str):
    """Discover RSS feeds from a website"""
    get_logger().info(f"Discovering RSS feeds for {website_url}")
    
    rss_source = RSSSource()
    feed_urls = rss_source.discover_feed_url(website_url)
    
    print(f"\n🔍 Discovered RSS feeds for {website_url}:")
    if not feed_urls:
        print("  No RSS feeds found.")
        print("  Try checking the website manually for RSS/Atom links.")
    else:
        for i, feed_url in enumerate(feed_urls, 1):
            print(f"  {i}. {feed_url}")
        
        print(f"\n💡 To use these feeds:")
        print(f"   Add to .env: RSS_FEEDS={','.join(feed_urls)}")

def list_recent_posts(hours: int = 24, limit: int = None, source: str = None):
    """List recent posts from the database"""
    since_time = datetime.now(timezone.utc) - timedelta(hours=hours)
    
    db = setup_database()
    posts = db.get_content_since(since_time, source=source)
    
    if limit:
        posts = posts[:limit]
    
    source_filter = f" from {source}" if source else ""
    print(f"\n📰 Recent posts{source_filter} from the last {hours} hours ({len(posts)} posts):\n")
    
    if not posts:
        print("  No posts found. Try:")
        print("  - Increasing the time range with --hours")
        print("  - Running 'python src/main.py ingest' to get new posts")
        if not source:
            print("  - Adding more RSS feeds to your configuration")
            print("  - Setting up YouTube with: python src/main.py authenticate --source youtube")
        return
    
    for i, post in enumerate(posts, 1):
        created_at = datetime.fromisoformat(post['created_at']).strftime('%Y-%m-%d %H:%M')
        
        # Add source emoji
        source_emoji = "📺" if post['source'] == 'youtube' else "📰"
        
        print(f"{source_emoji} {post['author_display_name']} ({created_at})")
        
        # Show entry title if available
        title = post.get('metadata', {}).get('entry_title', '')
        if title:
            print(f"   📌 {title} - {post.get('id')}")
        
        print(f"   {post['content'][:200]}{'...' if len(post['content']) > 200 else ''}")
        print(f"   🔗 {post['url']}")
        
        # Show tags if available
        tags = post.get('metadata', {}).get('entry_tags', [])
        if tags:
            print(f"   🏷️  {', '.join(tags[:5])}")
        
        # Show YouTube-specific info
        if post['source'] == 'youtube':
            video_id = post.get('metadata', {}).get('video_id')
            if video_id:
                print(f"   📺 Video ID: {video_id}")
        
        print()

def generate_digest(hours: int = None, output_file: str = None, source: str = None):
    """Generate an AI-powered digest of recent articles"""
    hours = hours or Config.DEFAULT_LOOKBACK_HOURS
    since_time = datetime.now(timezone.utc) - timedelta(hours=hours)
    
    source_filter = f" from {source}" if source else ""
    get_logger().info(f"Generating digest for articles{source_filter} from last {hours} hours")
    
    # Get recent posts from database
    db = setup_database()
    posts = db.get_content_since(since_time, source=source)
    
    if not posts:
        print(f"❌ No posts found{source_filter} from the last {hours} hours")
        print("   Try running 'python src/main.py ingest' first or increase --hours")
        return
    
    print(f"🤖 Generating AI digest for {len(posts)} recent articles{source_filter}...")
    print(f"   Using model: {Config.LLM_MODEL}")
    
    try:
        # Generate digest
        digest_generator = DigestGenerator()
        digest_content = digest_generator.summarize_articles(posts)
        
        # Auto-save if enabled or output_file specified
        if Config.AI_AUTO_SAVE:
            if not output_file:
                # Auto-generate filename
                import os
                os.makedirs(Config.AI_SAVE_DIRECTORY, exist_ok=True)
                timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
                source_suffix = f"_{source}" if source else ""
                output_file = f"{Config.AI_SAVE_DIRECTORY}/digest{source_suffix}_{timestamp}.md"
            
            with open(output_file, 'w', encoding='utf-8') as f:
                f.write(digest_content)
            print(f"✅ Digest saved to {output_file}")
        
        # Always show digest in console unless explicitly saving to file
        if not output_file or Config.AI_AUTO_SAVE:
            print("\n" + "="*60)
            print(digest_content)
            print("="*60)
            
    except ValueError as e:
        if "openai_api_key" in str(e) or "Incorrect API key" in str(e):
            print("❌ OpenAI API key not configured or invalid")
            print("   Set environment variable: export OPENAI_API_KEY='your_key_here'")
            print("   Get one from: https://platform.openai.com/api-keys")
        else:
            print(f"❌ Configuration error: {e}")
    except Exception as e:
        get_logger().error(f"Error generating digest: {e}")
        print(f"❌ Error generating digest: {e}")

def generate_youtube_digest(youtube_video_url: str, output_file: str = None):
    print(f"Generating youtube digest for video {youtube_video_url}")
    youtube_source = YouTubeSource()
    video_id = youtube_source.extract_video_id(youtube_video_url)
    transcript = youtube_source.get_video_transcript(video_id)

    if not transcript:
        print("Sorry, the video doesn't have transcript 😭")
        return

    try:
        digest_generator = DigestGenerator()
        digest_content = digest_generator.summarize_video(transcript)
        
        # Save or display digest
        _save_or_display_digest(digest_content, output_file, "youtube_video")
            
    except ValueError as e:
        if "openai_api_key" in str(e) or "Incorrect API key" in str(e):
            print("❌ OpenAI API key not configured or invalid")
            print("   Set environment variable: export OPENAI_API_KEY='your_key_here'")
            print("   Get one from: https://platform.openai.com/api-keys")
        else:
            print(f"❌ Configuration error: {e}")
    except Exception as e:
        get_logger().error(f"Error generating digest: {e}")
        print(f"❌ Error generating digest: {e}")

def generate_post_digest(post_id: str, output_file: str = None):
    db = setup_database()
    post = db.get_content_by(id=post_id)

    if not post:
        print(f"❌ Post not found with ID: {post_id}")
        return

    try:
        # Generate digest
        digest_generator = DigestGenerator()
        digest_content = digest_generator.summarize_article(post)
        
        # Save or display digest
        _save_or_display_digest(digest_content, output_file, post['source'])
            
    except ValueError as e:
        if "openai_api_key" in str(e) or "Incorrect API key" in str(e):
            print("❌ OpenAI API key not configured or invalid")
            print("   Set environment variable: export OPENAI_API_KEY='your_key_here'")
            print("   Get one from: https://platform.openai.com/api-keys")
        else:
            print(f"❌ Configuration error: {e}")
    except Exception as e:
        get_logger().error(f"Error generating digest: {e}")
        print(f"❌ Error generating digest: {e}")

def digest_url(url: str, output_file: str = None):
    """Digest content from a specific URL"""
    get_logger().info(f"Processing digest for URL: {url}")
    
    try:
        db = setup_database()
        
        # First, try to find content by ID (the URL itself might be the ID)
        existing_content = db.get_content_by(url)
        
        # If not found by ID, try to find by URL
        if not existing_content:
            existing_content = db.get_content_by_url(url)
        
        if existing_content:
            print(f"✅ Found cached content for {url}")
            print(f"   Source: {existing_content['source']}")
            print(f"   Cached at: {existing_content['fetched_at']}")
            
            # Generate digest from cached content
            digest_generator = DigestGenerator()
            digest_content = digest_generator.summarize_article(existing_content)
            
            # Save or display digest
            _save_or_display_digest(digest_content, output_file, f"cached_{existing_content['source']}")
        
        else:
            print(f"🔍 Content not found in cache for {url}")
            
            # Check if it's a YouTube URL
            youtube_source = YouTubeSource()
            video_id = youtube_source.extract_video_id(url)
            
            if video_id:
                print(f"📺 Detected YouTube video: {video_id}")
                print("🎬 Fetching transcript...")
                
                # Fetch transcript directly
                transcript = youtube_source.get_video_transcript(video_id)
                
                if not transcript:
                    print("❌ No transcript available for this video")
                    return
                
                print(f"✅ Extracted transcript ({len(transcript)} characters)")
                
                # Generate digest from transcript
                digest_generator = DigestGenerator()
                digest_content = digest_generator.summarize_video(transcript)
                
                # Save or display digest
                _save_or_display_digest(digest_content, output_file, "youtube_video")
            
            else:
                print(f"🌐 Detected website URL")
                print("📄 Fetching and processing content...")
                
                # For regular websites, use RSS scraper to get content
                rss_source = RSSSource()
                scraped_content = rss_source.scraper.scrape_article_content(url)
                
                if not scraped_content:
                    print("❌ Could not extract content from the website")
                    return
                
                print(f"✅ Extracted content ({len(scraped_content)} characters)")
                
                # Create article data structure for digest
                article_data = {
                    'title': 'Scraped Article',
                    'feed_title': '',
                    'content': scraped_content,
                    'url': url,
                    'source': 'website',
                    'published': datetime.now().isoformat()
                }
                
                # Generate digest
                digest_generator = DigestGenerator()
                digest_content = digest_generator.generate_llm_article_digest(article_data)
                
                # Save or display digest
                _save_or_display_digest(digest_content, output_file, "website")
                
    except ValueError as e:
        if "openai_api_key" in str(e) or "Incorrect API key" in str(e):
            print("❌ OpenAI API key not configured or invalid")
            print("   Set environment variable: export OPENAI_API_KEY='your_key_here'")
            print("   Get one from: https://platform.openai.com/api-keys")
        else:
            print(f"❌ Configuration error: {e}")
    except Exception as e:
        get_logger().error(f"Error processing URL digest: {e}")
        print(f"❌ Error processing URL: {e}")

def _save_or_display_digest(digest_content: str, output_file: str = None, source_type: str = ""):
    """Helper function to save or display digest content"""
    # Auto-save if enabled or output_file specified
    if output_file or Config.AI_AUTO_SAVE:
        if not output_file:
            # Auto-generate filename
            import os
            os.makedirs(Config.AI_SAVE_DIRECTORY, exist_ok=True)
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            source_suffix = f"_{source_type}" if source_type else ""
            output_file = f"{Config.AI_SAVE_DIRECTORY}/digest{source_suffix}_{timestamp}.md"
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(digest_content)
        print(f"✅ Digest saved to {output_file}")
    
    # Always show digest in console unless explicitly saving to file
    if not output_file or Config.AI_AUTO_SAVE:
        print("\n" + "="*60)
        print(digest_content)
        print("="*60)
        
def initialize_logging():
    log_dir = Path.home() / "Library" / "Logs" / "Colino"
    log_dir.mkdir(parents=True, exist_ok=True)
    
    log_file = log_dir / "colino.log"
    
    # Configure rotating file handler
    file_handler = RotatingFileHandler(
        str(log_file),
        maxBytes=10*1024*1024,  # 10MB per file
        backupCount=5,          # Keep 5 backup files
        encoding='utf-8'
    )
    
    # Configure logging with rotation
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            file_handler,
        ]
    )

def get_logger():
    """Get or create logger"""
    return logging.getLogger(__name__)

def main():
    """Main entry point"""
    initialize_logging()
    parser = argparse.ArgumentParser(description='Colino - Your hackable RSS feed aggregator')
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Ingest command
    ingest_parser = subparsers.add_parser('ingest', help='Ingest from RSS feeds or other sources')
    ingest_parser.add_argument('--rss', action='store_true', help='Ingest from RSS feeds')
    ingest_parser.add_argument('--youtube', action='store_true', help='Ingest from YouTube subscriptions')
    ingest_parser.add_argument('--all', action='store_true', help='Ingest from all configured sources (default)')
    ingest_parser.add_argument('--hours', type=int, help='Hours to look back (default: 24)')
    
    # Discover command
    discover_parser = subparsers.add_parser('discover', help='Discover RSS feeds from a website')
    discover_parser.add_argument('url', help='Website URL to scan for RSS feeds')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List recent posts from database')
    list_parser.add_argument('--hours', type=int, default=24, help='Hours to look back (default: 24)')
    list_parser.add_argument('--limit', type=int, help='Maximum number of posts to show')
    list_parser.add_argument('--source', choices=['rss', 'youtube'], help='Filter by source')
    
    # Digest command
    digest_parser = subparsers.add_parser('digest', help='Generate AI-powered summary of recent articles or specific URLs')
    digest_parser.add_argument('url', nargs='?', help='URL to digest (YouTube video or website)')
    digest_parser.add_argument('--hours', type=int, help='Hours to look back (default: 24) - for recent articles mode')
    digest_parser.add_argument('--output', help='Save digest to file instead of displaying')
    digest_parser.add_argument('--source', choices=['rss', 'youtube'], help='Generate digest for specific source - for recent articles mode')
    digest_parser.add_argument('--post-id', type=str, help='Generate digest for a specific post ID')
    digest_parser.add_argument('--youtube-video-url', type=str, help='Generate digest for a specific YouTube video URL')

    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        print(f"\n💡 Quick start:")
        print(f"   RSS: Add feeds to config.yaml, then run: python src/main.py ingest --rss")
        print(f"   YouTube: Run: python src/main.py authenticate --source youtube")
        print(f"   Then: python src/main.py ingest --youtube")
        print(f"   All sources: python src/main.py ingest --all (or just: python src/main.py ingest)")
        print(f"   View: python src/main.py list")
        print(f"   Digest URL: python src/main.py digest https://example.com")
        print(f"   Digest YouTube: python src/main.py digest https://www.youtube.com/watch?v=VIDEO_ID")
        print(f"   Digest post: python src/main.py digest --post-id POST_ID")
        print(f"   Digest recent: python src/main.py digest --source rss")
        return
    
    try:
        if args.command == 'ingest':
            # Determine which sources to ingest from
            sources = []
            
            # If no flags are specified or --all is specified, ingest from all sources
            if args.all or (not args.rss and not args.youtube):
                sources = ['rss', 'youtube']
            else:
                if args.rss:
                    sources.append('rss')
                if args.youtube:
                    sources.append('youtube')
            
            ingest(sources, args.hours)
        
        elif args.command == 'discover':
            discover_rss_feeds(args.url)
        
        elif args.command == 'list':
            list_recent_posts(args.hours, args.limit, args.source)
        
        elif args.command == 'digest':
            if args.post_id:
                # Digest specific post by ID
                generate_post_digest(args.post_id, args.output)
            elif args.youtube_video_url:
                # Digest specific YouTube video
                generate_youtube_digest(args.youtube_video_url, args.output)
            elif args.url:
                # Digest specific URL
                digest_url(args.url, args.output)
            else:
                # Digest recent articles
                generate_digest(args.hours, args.output, args.source)
        
    except KeyboardInterrupt:
        get_logger().info("Operation cancelled by user")
    except Exception as e:
        get_logger().error(f"Error: {e}")
        raise

if __name__ == '__main__':
    main()
