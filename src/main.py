#!/usr/bin/env python3
"""
Colino - RSS-Focused Social Digest
Your own hackable RSS feed aggregator and filter.
"""

import argparse
import logging
from datetime import datetime, timedelta, timezone
from typing import List, Optional
import re
import xml.sax.saxutils

from config import Config
from db import Database
from sources.rss import RSSSource
from summarize import DigestGenerator

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def setup_database():
    """Initialize the database"""
    logger.info("Setting up database...")
    db = Database()
    return db

def fetch_rss_posts(feed_urls: List[str] = None, since_hours: int = None):
    """Fetch posts from RSS feeds"""
    feed_urls = feed_urls or Config.RSS_FEEDS
    since_hours = since_hours or Config.DEFAULT_LOOKBACK_HOURS
    since_time = datetime.now(timezone.utc) - timedelta(hours=since_hours)
    
    if not feed_urls:
        logger.warning("No RSS feeds configured. Add feeds to your .env file or provide URLs.")
        return []
    
    logger.info(f"Fetching RSS posts from {len(feed_urls)} feeds since {since_time}")
    
    db = setup_database()
    rss_source = RSSSource()
    
    posts = rss_source.get_recent_posts(feed_urls, since_time)
    
    # Apply content filtering if configured
    if Config.FILTER_KEYWORDS or Config.EXCLUDE_KEYWORDS:
        posts = apply_content_filter(posts)
    
    # Save posts to database
    saved_count = 0
    for post in posts:
        if db.save_post(post):
            saved_count += 1
    
    logger.info(f"Successfully saved {saved_count}/{len(posts)} RSS posts")
    return posts

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
        logger.info(f"Content filtering: {len(filtered_posts)}/{len(posts)} posts kept")
    
    return filtered_posts

def discover_rss_feeds(website_url: str):
    """Discover RSS feeds from a website"""
    logger.info(f"Discovering RSS feeds for {website_url}")
    
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
        print(f"   Or test individually: python src/main.py fetch --urls {feed_urls[0]}")

def test_feed(feed_url: str):
    """Test a single RSS feed"""
    logger.info(f"Testing RSS feed: {feed_url}")
    
    rss_source = RSSSource()
    feed_data = rss_source.parse_feed(feed_url)
    
    if not feed_data:
        print(f"❌ Failed to parse feed: {feed_url}")
        return
    
    print(f"✅ Feed parsed successfully!")
    print(f"   Title: {feed_data['title']}")
    print(f"   Description: {feed_data['description'][:100]}{'...' if len(feed_data['description']) > 100 else ''}")
    print(f"   Entries: {len(feed_data['entries'])}")
    print(f"   Last updated: {feed_data.get('updated', 'Unknown')}")
    
    if feed_data['entries']:
        print(f"\n📝 Recent entries:")
        for i, entry in enumerate(feed_data['entries'][:5], 1):
            title = entry.get('title', 'No title')
            pub_date = getattr(entry, 'published', 'Unknown date')
            print(f"   {i}. {title} ({pub_date})")

def list_recent_posts(hours: int = 24, limit: int = None):
    """List recent posts from the database"""
    since_time = datetime.now(timezone.utc) - timedelta(hours=hours)
    
    db = setup_database()
    posts = db.get_posts_since(since_time, source='rss')
    
    if limit:
        posts = posts[:limit]
    
    print(f"\n📰 Recent RSS posts from the last {hours} hours ({len(posts)} posts):\n")
    
    if not posts:
        print("  No posts found. Try:")
        print("  - Increasing the time range with --hours")
        print("  - Running 'python src/main.py fetch' to get new posts")
        print("  - Adding more RSS feeds to your configuration")
        return
    
    for i, post in enumerate(posts, 1):
        created_at = datetime.fromisoformat(post['created_at']).strftime('%Y-%m-%d %H:%M')
        
        print(f"📰 {post['author_display_name']} ({created_at})")
        
        # Show entry title if available
        title = post.get('metadata', {}).get('entry_title', '')
        if title:
            print(f"   📌 {title}")
        
        print(f"   {post['content'][:200]}{'...' if len(post['content']) > 200 else ''}")
        print(f"   🔗 {post['url']}")
        
        # Show tags if available
        tags = post.get('metadata', {}).get('entry_tags', [])
        if tags:
            print(f"   🏷️  {', '.join(tags[:5])}")
        
        print()

def generate_digest(hours: int = None, output_file: str = None):
    """Generate an AI-powered digest of recent articles"""
    hours = hours or Config.DEFAULT_LOOKBACK_HOURS
    since_time = datetime.now(timezone.utc) - timedelta(hours=hours)
    
    logger.info(f"Generating digest for articles from last {hours} hours")
    
    # Get recent posts from database
    db = setup_database()
    posts = db.get_posts_since(since_time, source='rss')
    
    if not posts:
        print(f"❌ No posts found from the last {hours} hours")
        print("   Try running 'python src/main.py fetch' first or increase --hours")
        return
    
    print(f"🤖 Generating AI digest for {len(posts)} recent articles...")
    print(f"   Using model: {Config.LLM_MODEL}")
    
    try:
        # Generate digest
        digest_generator = DigestGenerator()
        digest_content = digest_generator.summarize_articles(posts)
        
        # Auto-save if enabled or output_file specified
        if output_file or Config.AI_AUTO_SAVE:
            if not output_file:
                # Auto-generate filename
                import os
                os.makedirs(Config.AI_SAVE_DIRECTORY, exist_ok=True)
                timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
                output_file = f"{Config.AI_SAVE_DIRECTORY}/digest_{timestamp}.md"
            
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
        logger.error(f"Error generating digest: {e}")
        print(f"❌ Error generating digest: {e}")

def export_opml(output_file: str = None):
    """Export RSS feeds as OPML file for backup/sharing"""
    output_file = output_file or f"colino_feeds_{datetime.now().strftime('%Y%m%d')}.opml"
    
    if not Config.RSS_FEEDS:
        print("❌ No RSS feeds configured to export")
        return
    
    # Basic OPML structure
    opml_content = '''<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
<head>
<title>Colino RSS Feeds</title>
<dateCreated>{date}</dateCreated>
</head>
<body>
'''.format(date=datetime.now(timezone.utc).strftime('%a, %d %b %Y %H:%M:%S %z'))
    
    rss_source = RSSSource()
    
    for feed_url in Config.RSS_FEEDS:
        if not feed_url.strip():
            continue
            
        # Try to get feed title
        feed_data = rss_source.parse_feed(feed_url.strip())
        title = feed_data['title'] if feed_data else feed_url
        
        # Escape XML characters for proper OPML format
        escaped_title = xml.sax.saxutils.escape(title)
        escaped_url = xml.sax.saxutils.escape(feed_url.strip())
        
        opml_content += f'<outline type="rss" text="{escaped_title}" xmlUrl="{escaped_url}" />\n'
    
    opml_content += '''</body>
</opml>'''
    
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(opml_content)
    
    print(f"✅ Exported {len(Config.RSS_FEEDS)} feeds to {output_file}")

def import_opml(opml_file: str):
    """Import RSS feeds from OPML file and update .env"""
    try:
        import xml.etree.ElementTree as ET
        import shutil
        import os
        
        tree = ET.parse(opml_file)
        root = tree.getroot()
        
        feeds = []
        for outline in root.findall('.//outline[@type="rss"]'):
            xml_url = outline.get('xmlUrl')
            if xml_url:
                feeds.append(xml_url)
        
        if not feeds:
            print("❌ No RSS feeds found in OPML file")
            return
        
        print(f"📥 Found {len(feeds)} feeds in OPML file:")
        for i, feed_url in enumerate(feeds, 1):
            print(f"   {i}. {feed_url}")
        
        # Backup existing .env file if it exists
        env_file = '.env'
        if os.path.exists(env_file):
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            backup_file = f'.env.backup.{timestamp}'
            shutil.copy2(env_file, backup_file)
            print(f"\n💾 Backed up existing .env to {backup_file}")
        
        # Read existing .env content
        env_lines = []
        rss_feeds_updated = False
        
        if os.path.exists(env_file):
            with open(env_file, 'r') as f:
                env_lines = f.readlines()
        
        # Update or add RSS_FEEDS line
        new_rss_line = f"RSS_FEEDS={','.join(feeds)}\n"
        
        for i, line in enumerate(env_lines):
            if line.strip().startswith('RSS_FEEDS='):
                env_lines[i] = new_rss_line
                rss_feeds_updated = True
                break
        
        # If RSS_FEEDS line not found, add it
        if not rss_feeds_updated:
            if env_lines and not env_lines[-1].endswith('\n'):
                env_lines.append('\n')
            env_lines.append(new_rss_line)
        
        # Write updated .env file
        with open(env_file, 'w') as f:
            f.writelines(env_lines)
        
        print(f"✅ Updated {env_file} with {len(feeds)} RSS feeds")
        print(f"📄 You can now run: python src/main.py fetch")
        
    except Exception as e:
        print(f"❌ Error importing OPML file: {e}")

def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description='Colino - Your hackable RSS feed aggregator')
    
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # Fetch command
    fetch_parser = subparsers.add_parser('fetch', help='Fetch from RSS feeds')
    fetch_parser.add_argument('--urls', nargs='+', help='Specific RSS feed URLs to fetch from')
    fetch_parser.add_argument('--hours', type=int, help='Hours to look back (default: 24)')
    
    # Discover command
    discover_parser = subparsers.add_parser('discover', help='Discover RSS feeds from a website')
    discover_parser.add_argument('url', help='Website URL to scan for RSS feeds')
    
    # Test command
    test_parser = subparsers.add_parser('test', help='Test a single RSS feed')
    test_parser.add_argument('url', help='RSS feed URL to test')
    
    # List command
    list_parser = subparsers.add_parser('list', help='List recent posts from database')
    list_parser.add_argument('--hours', type=int, default=24, help='Hours to look back (default: 24)')
    list_parser.add_argument('--limit', type=int, help='Maximum number of posts to show')
    
    # Digest command
    digest_parser = subparsers.add_parser('digest', help='Generate AI-powered summary of recent articles')
    digest_parser.add_argument('--hours', type=int, help='Hours to look back (default: 24)')
    digest_parser.add_argument('--output', help='Save digest to file instead of displaying')
    
    # Export/Import commands
    export_parser = subparsers.add_parser('export', help='Export feeds as OPML')
    export_parser.add_argument('--output', help='Output OPML file name')
    
    import_parser = subparsers.add_parser('import', help='Import feeds from OPML')
    import_parser.add_argument('file', help='OPML file to import')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        print(f"\n💡 Quick start:")
        print(f"   1. Add RSS feeds to your .env file")
        print(f"   2. Run: python src/main.py fetch")
        print(f"   3. View: python src/main.py list")
        return
    
    try:
        if args.command == 'fetch':
            fetch_rss_posts(args.urls, args.hours)
        
        elif args.command == 'discover':
            discover_rss_feeds(args.url)
        
        elif args.command == 'test':
            test_feed(args.url)
        
        elif args.command == 'list':
            list_recent_posts(args.hours, args.limit)
        
        elif args.command == 'digest':
            generate_digest(args.hours, args.output)
        
        elif args.command == 'export':
            export_opml(args.output)
        
        elif args.command == 'import':
            import_opml(args.file)
        
    except KeyboardInterrupt:
        logger.info("Operation cancelled by user")
    except Exception as e:
        logger.error(f"Error: {e}")
        raise

if __name__ == '__main__':
    main() 