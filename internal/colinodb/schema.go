package colinodb

import "database/sql"

// InitSchema ensures the DB has the tables needed for content ingestion.
func InitSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS content_cache (
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
        )`,
		`CREATE INDEX IF NOT EXISTS idx_content_cache_created_at ON content_cache(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_content_cache_source_author ON content_cache(source, author_username)`,
		`CREATE TABLE IF NOT EXISTS ingest_skip (
            url TEXT PRIMARY KEY,
            reason TEXT,
            expires_at TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_ingest_skip_expires_at ON ingest_skip(expires_at)`,
		`CREATE TABLE IF NOT EXISTS feed_cache (
            url TEXT PRIMARY KEY,
            etag TEXT,
            last_modified TEXT,
            checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_feed_cache_checked_at ON feed_cache(checked_at)`,
		`CREATE TABLE IF NOT EXISTS feed_bodyhash (
            url TEXT PRIMARY KEY,
            hash TEXT,
            checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_feed_bodyhash_checked_at ON feed_bodyhash(checked_at)`,
		`CREATE TABLE IF NOT EXISTS feed_backoff (
            url TEXT PRIMARY KEY,
            next_check_at TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_feed_backoff_next ON feed_backoff(next_check_at)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
