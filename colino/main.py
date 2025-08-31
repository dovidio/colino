#!/usr/bin/env python3
"""
Colino - RSS-Focused Social Digest
Your own hackable RSS feed aggregator and filter.
"""

import argparse
import logging
from datetime import UTC, datetime, timedelta
from logging.handlers import RotatingFileHandler
from pathlib import Path
from typing import Any

from .db import Database
from .digest_manager import DigestManager
from .ingest_manager import IngestManager


def setup_database() -> Database:
    """Initialize the database"""
    get_logger().info("Setting up database...")
    db = Database()
    return db


def ingest(
    sources: list[str] | None = None, since_hours: int | None = None
) -> list[dict[str, Any]]:
    """Ingest content from specified sources"""
    db = setup_database()
    ingest_manager = IngestManager(db)
    return ingest_manager.ingest(sources, since_hours)


def list_recent_posts(
    hours: int = 24, limit: int | None = None, source: str | None = None
) -> None:
    """List recent posts from the database"""
    since_time = datetime.now(UTC) - timedelta(hours=hours)

    db = setup_database()
    posts = db.get_content_since(since_time, source=source)

    if limit:
        posts = posts[:limit]

    source_filter = f" from {source}" if source else ""
    print(
        f"\n📰 Recent posts{source_filter} from the last {hours} hours ({len(posts)} posts):\n"
    )

    if not posts:
        print("  No posts found. Try:")
        print("  - Increasing the time range with --hours")
        print("  - Running 'python src/main.py ingest' to get new posts")

    for _i, post in enumerate(posts, 1):
        created_at = datetime.fromisoformat(post["created_at"]).strftime(
            "%Y-%m-%d %H:%M"
        )

        # Add source emoji
        source_emoji = "📺" if post["source"] == "youtube" else "📰"

        print(f"{source_emoji} {post['author_display_name']} ({created_at})")

        # Show entry title if available
        title = post.get("metadata", {}).get("entry_title", "")
        if title:
            print(f"   📌 {title} - {post.get('id')}")

        print(
            f"   {post['content'][:200]}{'...' if len(post['content']) > 200 else ''}"
        )
        print(f"   🔗 {post['url']}")

        # Show tags if available
        tags = post.get("metadata", {}).get("entry_tags", [])
        if tags:
            print(f"   🏷️  {', '.join(tags[:5])}")

        # Show YouTube-specific info
        if post["source"] == "youtube":
            video_id = post.get("metadata", {}).get("video_id")
            if video_id:
                print(f"   📺 Video ID: {video_id}")

        print()


def generate_digest(
    hours: int | None = None,
    output_file: str | None = None,
    source: str | None = None,
    auto_ingest: bool = True,
) -> bool:
    """Generate an AI-powered digest of recent articles"""
    digest_manager = DigestManager()
    return digest_manager.digest_recent_articles(
        hours, output_file, source, auto_ingest
    )


def digest_url(url: str, output_file: str | None = None) -> bool:
    """Digest content from a specific URL"""
    digest_manager = DigestManager()
    return digest_manager.digest_url(url, output_file)


def initialize_logging() -> None:
    log_dir = Path.home() / "Library" / "Logs" / "Colino"
    log_dir.mkdir(parents=True, exist_ok=True)

    log_file = log_dir / "colino.log"

    # Configure rotating file handler
    file_handler = RotatingFileHandler(
        str(log_file),
        maxBytes=10 * 1024 * 1024,  # 10MB per file
        backupCount=5,  # Keep 5 backup files
        encoding="utf-8",
    )

    # Configure logging with rotation
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=[
            file_handler,
        ],
    )


def get_logger() -> logging.Logger:
    """Get or create logger"""
    return logging.getLogger(__name__)


def main() -> None:
    """Main entry point"""
    initialize_logging()
    parser = argparse.ArgumentParser(
        description="Colino - News and digests from the terminal"
    )

    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Ingest command
    ingest_parser = subparsers.add_parser(
        "ingest", help="Ingest from RSS feeds or other sources"
    )
    ingest_parser.add_argument(
        "--rss", action="store_true", help="Ingest from RSS feeds"
    )
    ingest_parser.add_argument(
        "--youtube", action="store_true", help="Ingest from YouTube subscriptions"
    )
    ingest_parser.add_argument(
        "--all",
        action="store_true",
        help="Ingest from all configured sources (default)",
    )
    ingest_parser.add_argument(
        "--hours", type=int, help="Hours to look back (default: 24)"
    )

    # List command
    list_parser = subparsers.add_parser("list", help="List recent posts from database")
    list_parser.add_argument(
        "--hours", type=int, default=24, help="Hours to look back (default: 24)"
    )
    list_parser.add_argument(
        "--limit", type=int, help="Maximum number of posts to show"
    )
    list_parser.add_argument(
        "--source", choices=["rss", "youtube"], help="Filter by source"
    )

    # Digest command
    digest_parser = subparsers.add_parser(
        "digest", help="Generate AI-powered summary of recent articles or specific URLs"
    )
    digest_parser.add_argument(
        "url", nargs="?", help="URL to digest (YouTube video or website)"
    )
    digest_parser.add_argument(
        "--rss", action="store_true", help="Digest recent RSS articles"
    )
    digest_parser.add_argument(
        "--youtube", action="store_true", help="Digest recent YouTube videos"
    )
    digest_parser.add_argument(
        "--hours", type=int, help="Hours to look back (default: 24)"
    )
    digest_parser.add_argument(
        "--output", help="Save digest to file instead of displaying"
    )
    digest_parser.add_argument(
        "--skip-ingest",
        action="store_true",
        help="Skip automatic ingestion of recent sources before digesting",
    )

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        print("\n💡 Quick start:")
        print(
            "   RSS: Add feeds to config.yaml, then run: python src/main.py ingest --rss"
        )
        print("   YouTube: Run: python src/main.py ingest --youtube")
        print(
            "   All sources: python src/main.py ingest --all (or just: python src/main.py ingest)"
        )
        print("   View: python src/main.py list")
        print("   Digest URL: python src/main.py digest https://example.com")
        print(
            "   Digest YouTube: python src/main.py digest https://www.youtube.com/watch?v=VIDEO_ID"
        )
        print("   Digest RSS: python src/main.py digest --rss")
        print("   Digest YouTube: python src/main.py digest --youtube")
        print("   Digest all: python src/main.py digest")
        return

    try:
        if args.command == "ingest":
            # Determine which sources to ingest from
            sources = []

            # If no flags are specified or --all is specified, ingest from all sources
            if args.all or (not args.rss and not args.youtube):
                sources = ["rss", "youtube"]
            else:
                if args.rss:
                    sources.append("rss")
                if args.youtube:
                    sources.append("youtube")

            ingest(sources, args.hours)

        elif args.command == "list":
            list_recent_posts(args.hours, args.limit, args.source)

        elif args.command == "digest":
            if args.url:
                # Digest specific URL
                digest_url(args.url, args.output)
            elif args.rss:
                # Digest recent RSS articles
                generate_digest(args.hours, args.output, "rss", not args.skip_ingest)
            elif args.youtube:
                # Digest recent YouTube videos
                generate_digest(
                    args.hours, args.output, "youtube", not args.skip_ingest
                )
            else:
                # Digest recent articles from all sources
                generate_digest(args.hours, args.output, None, not args.skip_ingest)

    except KeyboardInterrupt:
        get_logger().info("Operation cancelled by user")
    except Exception as e:
        get_logger().error(f"Error: {e}")
        raise


if __name__ == "__main__":
    main()
