import sqlite3
import json
from datetime import datetime, timezone
from typing import List, Dict, Any
from .config import Config
import logging

logger = logging.getLogger(__name__)

class Database:
    def __init__(self, db_path: str = None):
        self.db_path = db_path or Config.DATABASE_PATH
        self.init_database()
    
    def init_database(self):
        """Initialize the database with required tables"""
        with sqlite3.connect(self.db_path) as conn:
            conn.execute('''
                CREATE TABLE IF NOT EXISTS content_cache (
                    id TEXT PRIMARY KEY,
                    source TEXT NOT NULL,
                    author_username TEXT NOT NULL,
                    author_display_name TEXT,
                    content TEXT NOT NULL,
                    url TEXT,
                    created_at TIMESTAMP NOT NULL,
                    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    metadata TEXT,
                    like_count INTEGER DEFAULT 0,
                    reply_count INTEGER DEFAULT 0
                )
            ''')

            conn.execute('''
                CREATE TABLE IF NOT EXISTS youtube_subscriptions (
                    channel_id TEXT PRIMARY KEY,
                    channel_title TEXT NOT NULL,
                    channel_description TEXT,
                    thumbnail_url TEXT,
                    rss_url TEXT NOT NULL,
                    subscribed_at TIMESTAMP,
                    last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            ''')
            
            conn.execute('''
                CREATE TABLE IF NOT EXISTS oauth_tokens (
                    service TEXT PRIMARY KEY,
                    access_token TEXT NOT NULL,
                    refresh_token TEXT,
                    expires_at REAL,
                    token_type TEXT DEFAULT 'Bearer',
                    scope TEXT,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            ''')
            
            conn.execute('''
                CREATE INDEX IF NOT EXISTS idx_content_cache_created_at ON content_cache(created_at);
            ''')
            
            conn.execute('''
                CREATE INDEX IF NOT EXISTS idx_content_cache_source_author ON content_cache(source, author_username);
            ''')
    
    def save_post(self, post_data: Dict[str, Any]) -> bool:
        """Save a post to the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.execute('''
                    INSERT OR REPLACE INTO content_cache 
                    (id, source, author_username, author_display_name, content, url, 
                     created_at, metadata, like_count, reply_count)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ''', (
                    post_data['id'],
                    post_data['source'],
                    post_data['author_username'],
                    post_data.get('author_display_name'),
                    post_data['content'],
                    post_data.get('url'),
                    post_data['created_at'],
                    json.dumps(post_data.get('metadata', {})),
                    post_data.get('like_count', 0),
                    post_data.get('reply_count', 0)
                ))
            return True
        except Exception as e:
            print(f"Error saving post {post_data.get('id')}: {e}")
            return False
    
    def get_post_by(self, id: str) -> Dict[str, Any]:
        """Get post by id""" 
        query = "SELECT * FROM content_cache WHERE id = ?" 
        params = [id]
        with sqlite3.connect(self.db_path) as conn:
            conn.row_factory = sqlite3.Row
            cursor = conn.execute(query, params)
            
            posts = []
            rows = cursor.fetchall()
            if len(rows) != 1:
                logger.info(f"Couldn't find post with id: {id}")
                return None

            post = dict(rows[0])
            if post['metadata']:
                post['metadata'] = json.loads(post['metadata'])
            
            return post

    def get_posts_since(self, since: datetime, source: str = None) -> List[Dict[str, Any]]:
        """Get posts since a specific timestamp"""
        query = "SELECT * FROM content_cache WHERE datetime(created_at) >= datetime(?)"
        params = [since.isoformat()]
        
        if source:
            query += " AND source = ?"
            params.append(source)
        
        query += " ORDER BY created_at DESC"
        
        with sqlite3.connect(self.db_path) as conn:
            conn.row_factory = sqlite3.Row
            cursor = conn.execute(query, params)
            
            posts = []
            for row in cursor.fetchall():
                post = dict(row)
                if post['metadata']:
                    post['metadata'] = json.loads(post['metadata'])
                posts.append(post)
            
            return posts

    def save_subscription(self, sub: Dict[str, Any]) -> bool:
        """Save a subscription to the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.execute('''
                    INSERT OR REPLACE INTO youtube_subscriptions
                    (channel_id, channel_title, channel_description, thumbnail_url, 
                        rss_url, subscribed_at, last_synced)
                    VALUES (?, ?, ?, ?, ?, ?, ?)
                ''', (
                    sub['channel_id'],
                    sub['channel_title'],
                    sub['channel_description'],
                    sub['thumbnail_url'],
                    sub['rss_url'],
                    sub['subscribed_at'],
                    datetime.now(timezone.utc).isoformat()
                ))
            return True
        except Exception as e:
            print(f"Error saving youtube subscription {sub.get('channel_id')}: {e}")
            return False

    def save_oauth_tokens(self, service: str, access_token: str, refresh_token: str = None, 
                         expires_at: float = None, token_type: str = 'Bearer', scope: str = None) -> bool:
        """Save OAuth tokens to the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.execute('''
                    INSERT OR REPLACE INTO oauth_tokens
                    (service, access_token, refresh_token, expires_at, token_type, scope, updated_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?)
                ''', (
                    service,
                    access_token,
                    refresh_token,
                    expires_at,
                    token_type,
                    scope,
                    datetime.now(timezone.utc).isoformat()
                ))
            logger.info(f"OAuth tokens saved for service: {service}")
            return True
        except Exception as e:
            logger.error(f"Error saving OAuth tokens for {service}: {e}")
            return False

    def get_oauth_tokens(self, service: str) -> Dict[str, Any]:
        """Get OAuth tokens from the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.row_factory = sqlite3.Row
                cursor = conn.execute('''
                    SELECT * FROM oauth_tokens WHERE service = ?
                ''', (service,))
                
                row = cursor.fetchone()
                if row:
                    tokens = dict(row)
                    logger.info(f"OAuth tokens loaded for service: {service}")
                    return tokens
                else:
                    logger.info(f"No OAuth tokens found for service: {service}")
                    return {}
        except Exception as e:
            logger.error(f"Error loading OAuth tokens for {service}: {e}")
            return {}

    def delete_oauth_tokens(self, service: str) -> bool:
        """Delete OAuth tokens from the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.execute('DELETE FROM oauth_tokens WHERE service = ?', (service,))
            logger.info(f"OAuth tokens deleted for service: {service}")
            return True
        except Exception as e:
            logger.error(f"Error deleting OAuth tokens for {service}: {e}")
            return False

    def get_connection(self):
        """Get a database connection (for backward compatibility)"""
        return sqlite3.connect(self.db_path)
