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
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
