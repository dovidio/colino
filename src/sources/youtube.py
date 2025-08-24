import json
import os
import webbrowser
from datetime import datetime, timezone
from typing import List, Dict, Any, Optional
import logging
from urllib.parse import urljoin, urlparse, parse_qs

import requests
from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError
from youtube_transcript_api import YouTubeTranscriptApi
from youtube_transcript_api.formatters import TextFormatter

from config import Config

logger = logging.getLogger(__name__)

class YouTubeSource:
    """YouTube source for fetching subscriptions and video transcripts"""
    
    # YouTube API scopes
    SCOPES = ['https://www.googleapis.com/auth/youtube.readonly']
    
    def __init__(self):
        self.credentials = None
        self.youtube_service = None
        self.token_file = Config.YOUTUBE_TOKEN_FILE
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': Config.RSS_USER_AGENT
        })
    
    def authenticate(self) -> bool:
        """Authenticate with YouTube API using OAuth2"""
        
        # Load existing credentials
        if os.path.exists(self.token_file):
            self.credentials = Credentials.from_authorized_user_file(
                self.token_file, self.SCOPES
            )
        
        # If credentials don't exist or are invalid, run OAuth flow
        if not self.credentials or not self.credentials.valid:
            if self.credentials and self.credentials.expired and self.credentials.refresh_token:
                try:
                    self.credentials.refresh(Request())
                    logger.info("YouTube credentials refreshed")
                except Exception as e:
                    logger.warning(f"Failed to refresh YouTube credentials: {e}")
                    self.credentials = None
            
            if not self.credentials:
                if not Config.YOUTUBE_CLIENT_SECRETS_FILE or not os.path.exists(Config.YOUTUBE_CLIENT_SECRETS_FILE):
                    raise ValueError(
                        "YouTube client secrets file not found. "
                        "Download it from Google Cloud Console and set YOUTUBE_CLIENT_SECRETS_FILE in config"
                    )
                
                flow = InstalledAppFlow.from_client_secrets_file(
                    Config.YOUTUBE_CLIENT_SECRETS_FILE, self.SCOPES
                )
                
                logger.info("Opening browser for YouTube authentication...")
                self.credentials = flow.run_local_server(port=8080)
                
                # Save credentials for next run
                with open(self.token_file, 'w') as f:
                    f.write(self.credentials.to_json())
                logger.info(f"YouTube credentials saved to {self.token_file}")
        
        # Build YouTube service
        try:
            self.youtube_service = build('youtube', 'v3', credentials=self.credentials)
            logger.info("YouTube API service initialized")
            return True
        except Exception as e:
            logger.error(f"Failed to initialize YouTube service: {e}")
            return False
    
    def get_subscriptions(self) -> List[Dict[str, Any]]:
        """Get user's YouTube subscriptions"""
        
        if not self.authenticate():
            raise Exception("Failed to authenticate with YouTube API")
        
        subscriptions = []
        next_page_token = None
        
        try:
            while True:
                request = self.youtube_service.subscriptions().list(
                    part='snippet',
                    mine=True,
                    maxResults=50,
                    pageToken=next_page_token
                )
                
                response = request.execute()
                
                for item in response['items']:
                    snippet = item['snippet']
                    channel_id = snippet['resourceId']['channelId']
                    
                    subscription = {
                        'channel_id': channel_id,
                        'channel_title': snippet['title'],
                        'channel_description': snippet['description'],
                        'thumbnail_url': snippet['thumbnails']['default']['url'],
                        'subscribed_at': snippet['publishedAt'],
                        'rss_url': f'https://www.youtube.com/feeds/videos.xml?channel_id={channel_id}'
                    }
                    
                    subscriptions.append(subscription)
                
                next_page_token = response.get('nextPageToken')
                if not next_page_token:
                    break
                    
        except HttpError as e:
            logger.error(f"YouTube API error getting subscriptions: {e}")
            raise
        
        logger.info(f"Retrieved {len(subscriptions)} YouTube subscriptions")
        return subscriptions
    
    def get_rss_feeds(self) -> List[str]:
        """Get RSS feed URLs for all subscriptions"""
        subscriptions = self.get_subscriptions()
        return [sub['rss_url'] for sub in subscriptions]
    
    def extract_video_id(self, url: str) -> Optional[str]:
        """Extract YouTube video ID from URL"""
        
        if not url:
            return None
        
        # Handle various YouTube URL formats
        patterns = [
            'youtube.com/watch?v=',
            'youtu.be/',
            'youtube.com/embed/',
            'youtube.com/v/',
        ]
        
        for pattern in patterns:
            if pattern in url:
                if pattern == 'youtu.be/':
                    # youtu.be/VIDEO_ID
                    return url.split('youtu.be/')[-1].split('?')[0].split('&')[0]
                else:
                    # youtube.com/watch?v=VIDEO_ID
                    parsed = urlparse(url)
                    if parsed.query:
                        query_params = parse_qs(parsed.query)
                        if 'v' in query_params:
                            return query_params['v'][0]
                    
                    # Handle embed/v/ formats
                    if '/embed/' in url:
                        return url.split('/embed/')[-1].split('?')[0]
                    if '/v/' in url:
                        return url.split('/v/')[-1].split('?')[0]
        
        return None
    
    def get_video_transcript(self, video_id: str) -> Optional[str]:
        """Get transcript for a YouTube video"""
        
        if not Config.YOUTUBE_EXTRACT_TRANSCRIPTS:
            return None
        
        try:
            # Try to get transcript in preferred languages
            languages = Config.YOUTUBE_TRANSCRIPT_LANGUAGES
            api = YouTubeTranscriptApi()
            transcript_list = api.fetch(video_id, languages=languages)
            
            if len(transcript_list) == 0:
                logger.debug(f"No transcript available for video {video_id}")
                return None
            
            formatter = TextFormatter()
            transcript_text = formatter.format_transcript(transcript_list)
            
            # Clean up transcript
            transcript_text = transcript_text.replace('\n', ' ')
            transcript_text = ' '.join(transcript_text.split())  # Remove extra whitespace
            
            logger.info(f"Extracted transcript for video {video_id} ({len(transcript_text)} chars)")
            return transcript_text
            
        except Exception as e:
            logger.warning(f"Could not get transcript for video {video_id}: {e}")
            return None
    
    def enhance_youtube_post(self, post_data: Dict[str, Any]) -> Dict[str, Any]:
        """Enhance a YouTube RSS post with transcript if available"""
        
        url = post_data.get('url', '')
        video_id = self.extract_video_id(url)
        
        if not video_id:
            return post_data
        
        # Get transcript
        transcript = self.get_video_transcript(video_id)
        
        if transcript:
            # Add transcript to metadata
            if 'metadata' not in post_data:
                post_data['metadata'] = {}
            
            post_data['metadata']['youtube_video_id'] = video_id
            post_data['metadata']['youtube_transcript'] = transcript
            
            # Enhance content with transcript preview
            transcript_preview = transcript[:300] + "..." if len(transcript) > 300 else transcript
            original_content = post_data.get('content', '')
            post_data['content'] = f"{original_content}\n\nTranscript preview: {transcript_preview}"
            
            logger.info(f"Enhanced YouTube post {video_id} with transcript")
        
        return post_data
    
    def sync_subscriptions_to_db(self, subscriptions, db):
        """Sync YouTube subscriptions to database"""
        saved_count = 0
        for sub in subscriptions:
            if db.save_subscription(sub):
                saved_count += 1
        
        logger.info(f"Synced {saved_count}/{len(subscriptions)} YouTube subscriptions")
        return saved_count
    
    def get_synced_rss_feeds(self, db) -> List[str]:
        """Get RSS feed URLs from synced subscriptions"""
        
        try:
            with db.get_connection() as conn:
                cursor = conn.execute('SELECT rss_url FROM youtube_subscriptions')
                feeds = [row[0] for row in cursor.fetchall()]
            
            logger.info(f"Retrieved {len(feeds)} YouTube RSS feeds from database")
            return feeds
            
        except Exception as e:
            logger.error(f"Error getting synced YouTube feeds: {e}")
            return []

    @property
    def is_authenticated(self) -> bool:
        """Check if the YouTube service is authenticated and ready."""
        return self.youtube_service is not None
