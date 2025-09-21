package colinodb

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    _ "modernc.org/sqlite"
)

// Content mirrors the Python content_cache schema (subset of columns we expose).
type Content struct {
    ID                 string
    Source             string
    AuthorUsername     string
    AuthorDisplayName  sql.NullString
    Content            string
    URL                sql.NullString
    CreatedAt          time.Time
    FetchedAt          sql.NullTime
    Metadata           sql.NullString // JSON
    LikeCount          int64
    ReplyCount         int64
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
    q := `SELECT id, source, author_username, author_display_name, content, url, created_at, fetched_at, metadata, like_count, reply_count
FROM content_cache WHERE datetime(created_at) >= datetime(?)`
    args := []any{since.UTC().Format(time.RFC3339)}
    if source != "" {
        q += " AND source = ?"
        args = append(args, source)
    }
    q += " ORDER BY datetime(created_at) DESC"
    if limit > 0 {
        q += fmt.Sprintf(" LIMIT %d", limit)
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
