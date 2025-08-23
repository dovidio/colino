import sqlite3
import json
from datetime import datetime, timezone
from typing import List, Dict, Any
from config import Config

class Database:
    def __init__(self, db_path: str = None):
        self.db_path = db_path or Config.DATABASE_PATH
        self.init_database()
    
    def init_database(self):
        """Initialize the database with required tables"""
        with sqlite3.connect(self.db_path) as conn:
            conn.execute('''
                CREATE TABLE IF NOT EXISTS posts (
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
                CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at);
            ''')
            
            conn.execute('''
                CREATE INDEX IF NOT EXISTS idx_posts_source_author ON posts(source, author_username);
            ''')
    
    def save_post(self, post_data: Dict[str, Any]) -> bool:
        """Save a post to the database"""
        try:
            with sqlite3.connect(self.db_path) as conn:
                conn.execute('''
                    INSERT OR REPLACE INTO posts 
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
    
    def get_posts_since(self, since: datetime, source: str = None) -> List[Dict[str, Any]]:
        """Get posts since a specific timestamp"""
        query = "SELECT * FROM posts WHERE datetime(created_at) >= datetime(?)"
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