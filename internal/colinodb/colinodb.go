package colinodb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Content struct {
	ID                string
	Source            string
	AuthorUsername    string
	AuthorDisplayName sql.NullString
	Content           string
	URL               sql.NullString
	CreatedAt         time.Time
	FetchedAt         sql.NullTime
	Metadata          sql.NullString // JSON
	LikeCount         int64
	ReplyCount        int64
}

func Open(dbPath string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func GetSince(ctx context.Context, db *sql.DB, since time.Time, source string, limit int) ([]Content, error) {
	// created_at is stored as a Go time string like "YYYY-MM-DD HH:MM:SS +0000 UTC"
	// or potentially RFC3339 in future. Normalize by comparing only the first
	// 19 chars (YYYY-MM-DD[ T]HH:MM:SS) which SQLite's datetime() can parse.
	q := `SELECT id, source, author_username, author_display_name, content, url, created_at, fetched_at, metadata, like_count, reply_count
FROM content_cache WHERE datetime(substr(created_at,1,19)) >= datetime(?)`
	// Use a format SQLite understands without timezone suffix.
	sinceStr := since.UTC().Format("2006-01-02 15:04:05")
	args := []any{sinceStr}
	if source != "" {
		q += " AND source = ?"
		args = append(args, source)
	}
	q += " ORDER BY datetime(substr(created_at,1,19)) DESC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Content
	for rows.Next() {
		var c Content
		if err := rows.Scan(&c.ID, &c.Source, &c.AuthorUsername, &c.AuthorDisplayName, &c.Content, &c.URL, &c.CreatedAt, &c.FetchedAt, &c.Metadata, &c.LikeCount, &c.ReplyCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func GetByID(ctx context.Context, db *sql.DB, id string) (*Content, error) {
	row := db.QueryRowContext(ctx, `SELECT id, source, author_username, author_display_name, content, url, created_at, fetched_at, metadata, like_count, reply_count FROM content_cache WHERE id = ?`, id)
	var c Content
	if err := row.Scan(&c.ID, &c.Source, &c.AuthorUsername, &c.AuthorDisplayName, &c.Content, &c.URL, &c.CreatedAt, &c.FetchedAt, &c.Metadata, &c.LikeCount, &c.ReplyCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func GetByURL(ctx context.Context, db *sql.DB, url string) (*Content, error) {
	row := db.QueryRowContext(ctx, `SELECT id, source, author_username, author_display_name, content, url, created_at, fetched_at, metadata, like_count, reply_count FROM content_cache WHERE url = ? ORDER BY datetime(created_at) DESC LIMIT 1`, url)
	var c Content
	if err := row.Scan(&c.ID, &c.Source, &c.AuthorUsername, &c.AuthorDisplayName, &c.Content, &c.URL, &c.CreatedAt, &c.FetchedAt, &c.Metadata, &c.LikeCount, &c.ReplyCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// GetURLsBySource returns a set of URLs for rows with the given source.
func GetURLsBySource(ctx context.Context, db *sql.DB, source string) (map[string]struct{}, error) {
	urls := make(map[string]struct{})
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(source) == "" {
		rows, err = db.QueryContext(ctx, `SELECT url FROM content_cache WHERE url IS NOT NULL`)
	} else {
		rows, err = db.QueryContext(ctx, `SELECT url FROM content_cache WHERE source = ? AND url IS NOT NULL`, source)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u sql.NullString
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		if u.Valid && strings.TrimSpace(u.String) != "" {
			urls[u.String] = struct{}{}
		}
	}
	return urls, rows.Err()
}

// IsURLSkipped returns true if the URL is in the skip cache and not expired.
func IsURLSkipped(ctx context.Context, db *sql.DB, url string) (bool, error) {
	if strings.TrimSpace(url) == "" {
		return false, nil
	}
	var exists int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM ingest_skip WHERE url = ? AND (expires_at IS NULL OR datetime(expires_at) > CURRENT_TIMESTAMP)`, url).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

// SkipURL inserts/updates a skip entry for the URL with a TTL.
func SkipURL(ctx context.Context, db *sql.DB, url, reason string, ttl time.Duration) error {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	expires := time.Now().Add(ttl).UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `INSERT INTO ingest_skip (url, reason, expires_at) VALUES (?, ?, ?)
        ON CONFLICT(url) DO UPDATE SET reason=excluded.reason, expires_at=excluded.expires_at`, url, reason, expires)
	return err
}

// ContentInsert captures data for upserting into content_cache.
type ContentInsert struct {
	ID                string
	Source            string
	AuthorUsername    string
	AuthorDisplayName string
	Content           string
	URL               string
	CreatedAt         time.Time
	MetadataJSON      string
	LikeCount         int64
	ReplyCount        int64
}

func UpsertContent(ctx context.Context, db *sql.DB, c ContentInsert) error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Source) == "" {
		return errors.New("missing id or source")
	}
	_, err := db.ExecContext(ctx, `INSERT INTO content_cache
        (id, source, author_username, author_display_name, content, url, created_at, metadata, like_count, reply_count)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
           source=excluded.source,
           author_username=excluded.author_username,
           author_display_name=excluded.author_display_name,
           content=excluded.content,
           url=excluded.url,
           created_at=excluded.created_at,
           metadata=excluded.metadata,
           like_count=excluded.like_count,
           reply_count=excluded.reply_count
        `,
		c.ID, c.Source, c.AuthorUsername, c.AuthorDisplayName, c.Content, nullIfEmpty(c.URL), c.CreatedAt, nullIfEmpty(c.MetadataJSON), c.LikeCount, c.ReplyCount,
	)
	return err
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
