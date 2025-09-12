import logging
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from typing import Any, ClassVar
from urllib.parse import parse_qs, urlparse

import requests
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError
from youtube_transcript_api import YouTubeTranscriptApi
from youtube_transcript_api.formatters import TextFormatter
from youtube_transcript_api.proxies import WebshareProxyConfig

from ..config import config
from ..db import Database
from ..oauth_proxy import OAuthProxyClient, TokenManager
from .base import BaseSource

logger = logging.getLogger(__name__)


class YouTubeSource(BaseSource):
    """YouTube source for fetching subscriptions and video transcripts"""

    # YouTube API scopes
    SCOPES: ClassVar[list[str]] = ["https://www.googleapis.com/auth/youtube.readonly"]

    def __init__(self, db: Database | None = None) -> None:
        super().__init__(db)
        self.credentials: Credentials | None = None
        self.youtube_service: Any | None = None
        self.session = requests.Session()
        self.session.headers.update({"User-Agent": config.RSS_USER_AGENT})

        # Initialize OAuth proxy client (always used)
        self.oauth_client = OAuthProxyClient(config.YOUTUBE_OAUTH_PROXY_URL)
        self.token_manager = TokenManager(self.oauth_client, self.db, "youtube")

    def _get_optimal_worker_count(self) -> int:
        """Get optimal number of worker threads based on CPU cores"""
        cpu_count = os.cpu_count() or 4  # Default to 4 if unable to detect
        # Use min(cpu_count, 8) to balance performance vs resource usage
        # Too many concurrent requests might still hit rate limits
        optimal_count = min(cpu_count, 8)
        logger.info(
            f"Using {optimal_count} worker threads for parallel transcript fetching (CPU cores: {cpu_count})"
        )
        return optimal_count

    @property
    def source_name(self) -> str:
        return "youtube"

    def get_recent_content(
        self, since_time: datetime | None = None
    ) -> list[dict[str, Any]]:
        """Get recent content from YouTube subscriptions"""
        if not self.authenticate():
            logger.error("YouTube authentication required")
            return []

        try:
            # Get subscriptions and their RSS feeds
            subscriptions = self.get_subscriptions()
            if self.db is not None:
                self.sync_subscriptions_to_db(subscriptions, self.db)

            if not subscriptions:
                logger.warning("No YouTube channels found")
                return []

            # Import here to avoid circular dependency
            from .rss import RSSSource

            rss_source = RSSSource(db=self.db)

            # Get RSS feeds for all subscriptions
            rss_feeds = [sub["rss_url"] for sub in subscriptions]

            # Fetch posts from RSS feeds
            youtube_posts = rss_source.get_posts_from_feeds(rss_feeds, since_time)

            # Process each post to add YouTube-specific metadata
            processed_posts = []
            for post in youtube_posts:
                # Set source to youtube instead of rss
                post["source"] = "youtube"

                # Add YouTube-specific metadata
                video_id = self.extract_video_id(post["url"])
                if video_id:
                    post["metadata"]["video_id"] = video_id

                    # Find the channel info for this post
                    for sub in subscriptions:
                        if sub["rss_url"] == post["metadata"].get("feed_url"):
                            post["metadata"]["channel_id"] = sub["channel_id"]
                            break

                processed_posts.append(post)

            # Enhance all posts with transcripts in parallel
            processed_posts = self.enhance_youtube_posts_parallel(processed_posts)

            logger.info(f"Successfully processed {len(processed_posts)} YouTube posts")
            return processed_posts

        except Exception as e:
            logger.error(f"Error fetching YouTube posts: {e}")
            return []

    def authenticate(self) -> bool:
        """Authenticate with YouTube API using OAuth proxy"""
        try:
            access_token = self.token_manager.get_access_token()

            # Create credentials object from access token
            self.credentials = Credentials(token=access_token)

            # Build YouTube service
            self.youtube_service = build("youtube", "v3", credentials=self.credentials)
            logger.info("YouTube API service initialized via OAuth proxy")
            return True

        except Exception as e:
            logger.error(f"Failed to authenticate via OAuth proxy: {e}")

            # If authentication fails, try to force re-authentication
            try:
                logger.info("Attempting forced re-authentication...")
                access_token = self.token_manager.authenticate()
                self.credentials = Credentials(token=access_token)
                self.youtube_service = build(
                    "youtube", "v3", credentials=self.credentials
                )
                logger.info("YouTube API service initialized after re-authentication")
                return True
            except Exception as e2:
                logger.error(f"Re-authentication also failed: {e2}")
                return False

    def get_subscriptions(self) -> list[dict[str, Any]]:
        """Get user's YouTube subscriptions"""

        if not self.authenticate():
            raise Exception("Failed to authenticate with YouTube API")

        if self.youtube_service is None:
            raise Exception("YouTube service not initialized")

        subscriptions = []
        next_page_token = None

        try:
            while True:
                request = self.youtube_service.subscriptions().list(
                    part="snippet", mine=True, maxResults=50, pageToken=next_page_token
                )

                response = request.execute()

                for item in response["items"]:
                    snippet = item["snippet"]
                    channel_id = snippet["resourceId"]["channelId"]

                    subscription = {
                        "channel_id": channel_id,
                        "channel_title": snippet["title"],
                        "channel_description": snippet["description"],
                        "thumbnail_url": snippet["thumbnails"]["default"]["url"],
                        "subscribed_at": snippet["publishedAt"],
                        "rss_url": f"https://www.youtube.com/feeds/videos.xml?channel_id={channel_id}",
                    }

                    subscriptions.append(subscription)

                next_page_token = response.get("nextPageToken")
                if not next_page_token:
                    break

        except HttpError as e:
            logger.error(f"YouTube API error getting subscriptions: {e}")
            raise

        logger.info(f"Retrieved {len(subscriptions)} YouTube subscriptions")
        return subscriptions

    def extract_video_id(self, url: str) -> str | None:
        """Extract YouTube video ID from URL"""

        if not url:
            return None

        # Handle various YouTube URL formats
        patterns = [
            "youtube.com/watch?v=",
            "youtu.be/",
            "youtube.com/embed/",
            "youtube.com/v/",
        ]

        for pattern in patterns:
            if pattern in url:
                if pattern == "youtu.be/":
                    # youtu.be/VIDEO_ID
                    return url.split("youtu.be/")[-1].split("?")[0].split("&")[0]
                else:
                    # youtube.com/watch?v=VIDEO_ID
                    parsed = urlparse(url)
                    if parsed.query:
                        query_params = parse_qs(parsed.query)
                        if "v" in query_params:
                            return query_params["v"][0]

                    # Handle embed/v/ formats
                    if "/embed/" in url:
                        return url.split("/embed/")[-1].split("?")[0]
                    if "/v/" in url:
                        return url.split("/v/")[-1].split("?")[0]

        return None

    def get_video_transcript(self, video_id: str) -> Any:
        """Get transcript for a YouTube video"""

        try:
            # Try to get transcript in preferred languages
            languages = config.YOUTUBE_TRANSCRIPT_LANGUAGES

            proxy_config = self._get_proxy_config()

            if proxy_config:
                logger.info(f"Using rotating proxy for transcript fetching: {video_id}")
                api = YouTubeTranscriptApi(proxy_config=proxy_config)
            else:
                api = YouTubeTranscriptApi()

            transcript_list = api.fetch(video_id, languages=languages)

            if len(transcript_list) == 0:
                logger.debug(f"No transcript available for video {video_id}")
                return None

            formatter = TextFormatter()
            transcript_text = formatter.format_transcript(transcript_list)

            # Clean up transcript
            transcript_text = transcript_text.replace("\n", " ")
            transcript_text = " ".join(
                transcript_text.split()
            )  # Remove extra whitespace

            logger.info(
                f"Extracted transcript for video {video_id} ({len(transcript_text)} chars)"
            )
            return transcript_text

        except Exception as e:
            logger.warning(f"Could not get transcript for video {video_id}: {e}")
            return None

    def _get_proxy_config(self) -> WebshareProxyConfig | None:
        """Get proxy configuration for YouTube transcript API if enabled"""
        if not config.YOUTUBE_PROXY_ENABLED:
            return None

        # Check if Webshare credentials are configured
        username = config.YOUTUBE_PROXY_WEBSHARE_USERNAME
        password = config.YOUTUBE_PROXY_WEBSHARE_PASSWORD

        if not username or not password:
            logger.warning(
                "Proxy is enabled but Webshare credentials not configured. "
                "Please set youtube.proxy.webshare.username and youtube.proxy.webshare.password in config.yaml"
            )
            return None

        return WebshareProxyConfig(
            proxy_username=username,
            proxy_password=password,
        )

    def enhance_youtube_post(self, post_data: dict[str, Any]) -> dict[str, Any]:
        """Enhance a YouTube RSS post with transcript if available"""

        url = post_data.get("url", "")
        video_id = self.extract_video_id(url)

        if not video_id:
            return post_data

        transcript = self.get_video_transcript(video_id)

        if transcript:
            # Add video ID to metadata for reference
            if "metadata" not in post_data:
                post_data["metadata"] = {}

            post_data["metadata"]["youtube_video_id"] = video_id

            # Append full transcript to content
            original_content = post_data.get("content", "")
            post_data["content"] = (
                f"{original_content}\n\n--- YouTube Transcript ---\n{transcript}"
            )

            logger.info(
                f"Enhanced YouTube post {video_id} with full transcript ({len(transcript)} chars)"
            )

        return post_data

    def enhance_youtube_posts_parallel(
        self, posts: list[dict[str, Any]]
    ) -> list[dict[str, Any]]:
        """Enhance multiple YouTube posts with transcripts in parallel"""
        if not posts:
            return posts

        # Filter posts that have video IDs
        posts_with_video_ids = []
        posts_without_video_ids = []

        for post in posts:
            video_id = self.extract_video_id(post.get("url", ""))
            if video_id:
                post["metadata"]["video_id"] = video_id
                posts_with_video_ids.append(post)
            else:
                posts_without_video_ids.append(post)

        if not posts_with_video_ids:
            logger.info(
                "No YouTube posts with valid video IDs found for transcript enhancement"
            )
            return posts

        # Get optimal worker count
        max_workers = self._get_optimal_worker_count()

        logger.info(
            f"Enhancing {len(posts_with_video_ids)} YouTube posts with transcripts using {max_workers} workers"
        )

        # Create a mapping to preserve order
        post_to_future = {}
        enhanced_posts = {}

        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            # Submit all transcript fetching jobs
            for post in posts_with_video_ids:
                video_id = post["metadata"]["video_id"]
                # video_id is guaranteed to be a string here since we filtered out None values
                assert isinstance(video_id, str)
                future = executor.submit(
                    self._fetch_transcript_for_post, post, video_id
                )
                post_to_future[future] = post

            # Collect results as they complete
            for future in as_completed(post_to_future):
                original_post = post_to_future[future]
                try:
                    enhanced_post = future.result()
                    enhanced_posts[id(original_post)] = enhanced_post
                except Exception as e:
                    logger.error(
                        f"Failed to enhance post {original_post.get('url', 'unknown')}: {e}"
                    )
                    # Keep original post if enhancement fails
                    enhanced_posts[id(original_post)] = original_post

        # Reconstruct the original order
        result_posts = []
        for post in posts:
            if id(post) in enhanced_posts:
                result_posts.append(enhanced_posts[id(post)])
            else:
                result_posts.append(post)

        successful_enhancements = sum(
            1
            for post in result_posts
            if post.get("metadata", {}).get("youtube_video_id")
        )
        logger.info(
            f"Successfully enhanced {successful_enhancements}/{len(posts_with_video_ids)} YouTube posts with transcripts"
        )

        return result_posts

    def _fetch_transcript_for_post(
        self, post_data: dict[str, Any], video_id: str
    ) -> dict[str, Any]:
        """Helper method to fetch transcript for a single post (used in parallel processing)"""
        try:
            transcript = self.get_video_transcript(video_id)

            if transcript:
                # Create a copy to avoid modifying the original in place
                enhanced_post = post_data.copy()
                enhanced_post["metadata"] = post_data.get("metadata", {}).copy()

                # Add video ID to metadata for reference
                enhanced_post["metadata"]["youtube_video_id"] = video_id

                # Append full transcript to content
                original_content = enhanced_post.get("content", "")
                enhanced_post["content"] = (
                    f"{original_content}\n\n--- YouTube Transcript ---\n{transcript}"
                )

                logger.debug(
                    f"Enhanced post {video_id} with transcript ({len(transcript)} chars)"
                )
                return enhanced_post
            else:
                logger.debug(f"No transcript available for video {video_id}")
                return post_data

        except Exception as e:
            logger.warning(f"Failed to get transcript for video {video_id}: {e}")
            return post_data

    def sync_subscriptions_to_db(
        self, subscriptions: list[dict[str, Any]], db: Database
    ) -> int:
        """Sync YouTube subscriptions to database"""
        saved_count = 0
        for sub in subscriptions:
            if db.save_subscription(sub):
                saved_count += 1

        logger.info(f"Synced {saved_count}/{len(subscriptions)} YouTube subscriptions")
        return saved_count

    def get_synced_rss_feeds(self, db: Database) -> list[str]:
        """Get RSS feed URLs from synced subscriptions"""

        try:
            with db.get_connection() as conn:
                cursor = conn.execute("SELECT rss_url FROM youtube_subscriptions")
                feeds = [row[0] for row in cursor.fetchall()]

            logger.info(f"Retrieved {len(feeds)} YouTube RSS feeds from database")
            return feeds

        except Exception as e:
            logger.error(f"Error getting synced YouTube feeds: {e}")
            return []
