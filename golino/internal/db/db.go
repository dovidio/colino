package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "time"

    _ "modernc.org/sqlite"

    "golino/internal/config"
    "golino/internal/models"
)

func Open(cfg config.Config) (*sql.DB, error) {
    if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
        // ignore if path has no directory component
        _ = err
    }
    dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", cfg.DatabasePath)
    database, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }
    database.SetConnMaxLifetime(0)
    database.SetMaxIdleConns(2)
    database.SetMaxOpenConns(1)
    if err := migrate(database); err != nil {
        _ = database.Close()
        return nil, err
    }
    return database, nil
}

func migrate(db *sql.DB) error {
    schema := `
CREATE TABLE IF NOT EXISTS items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  guid TEXT UNIQUE,
  source_url TEXT,
  source_type TEXT,
  title TEXT,
  link TEXT,
  published_at TIMESTAMP,
  summary TEXT,
  content TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_items_published ON items(published_at);
`
    _, err := db.Exec(schema)
    return err
}

func UpsertItem(ctx context.Context, db *sql.DB, it models.Item) (int64, error) {
    // Try insert; on conflict(guid) update selected columns
    res, err := db.ExecContext(ctx, `
INSERT INTO items (guid, source_url, source_type, title, link, published_at, summary, content)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(guid) DO UPDATE SET
  title=excluded.title,
  link=excluded.link,
  published_at=excluded.published_at,
  summary=excluded.summary,
  content=excluded.content
`, it.GUID, it.SourceURL, it.SourceType, it.Title, it.Link, it.Published.UTC(), it.Summary, it.Content)
    if err != nil {
        return 0, err
    }
    id, _ := res.LastInsertId()
    return id, nil
}

func ListRecent(ctx context.Context, db *sql.DB, hours int) ([]models.Item, error) {
    since := time.Now().Add(-time.Duration(hours) * time.Hour).UTC()
    rows, err := db.QueryContext(ctx, `
SELECT id, guid, source_url, source_type, title, link, published_at, summary, content, created_at
FROM items
WHERE published_at >= ?
ORDER BY published_at DESC
`, since)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []models.Item
    for rows.Next() {
        var it models.Item
        var pub, created sql.NullTime
        if err := rows.Scan(&it.ID, &it.GUID, &it.SourceURL, &it.SourceType, &it.Title, &it.Link, &pub, &it.Summary, &it.Content, &created); err != nil {
            return nil, err
        }
        if pub.Valid {
            it.Published = pub.Time
        }
        if created.Valid {
            it.CreatedAt = created.Time
        }
        out = append(out, it)
    }
    return out, rows.Err()
}

func CountAll(ctx context.Context, db *sql.DB) (int64, error) {
    row := db.QueryRowContext(ctx, `SELECT count(1) FROM items`)
    var n int64
    if err := row.Scan(&n); err != nil {
        return 0, err
    }
    return n, nil
}

var ErrNotFound = errors.New("not found")
