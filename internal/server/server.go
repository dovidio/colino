package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"golino/internal/colinodb"
	"golino/internal/config"
)

type ListCacheParams struct {
	Hours          int     `json:"hours"`
	Source         *string `json:"source,omitempty"`
	Limit          *int    `json:"limit,omitempty"`
	IncludeContent bool    `json:"include_content"`
}

type GetContentParams struct {
	IDs            []string `json:"ids,omitempty"`
	URL            *string  `json:"url,omitempty"`
	Hours          *int     `json:"hours,omitempty"`
	Source         *string  `json:"source,omitempty"`
	Limit          *int     `json:"limit,omitempty"`
	IncludeContent bool     `json:"include_content"`
}

func Run(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "colino", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "list_cache", Description: "List cached posts from the Colino DB"}, handleListCache)
	mcp.AddTool(server, &mcp.Tool{Name: "get_content", Description: "Get content by IDs, URL, or time window"}, handleGetContent)

	return server.Run(ctx, &mcp.StdioTransport{})
}

// Returns a list with all the elements present in the cache, respecting filtering parameters
func handleListCache(ctx context.Context, req *mcp.CallToolRequest, p ListCacheParams) (*mcp.CallToolResult, any, error) {
	if p.Hours <= 0 {
		p.Hours = 24
	}
	lim := 50
	if p.Limit != nil && *p.Limit > 0 {
		lim = *p.Limit
	}

	dbPath, err := config.LoadDBPath()
	if err != nil {
		return nil, nil, err
	}
	if !fileExists(dbPath) {
		return nil, map[string]any{
			"ok":      false,
			"message": fmt.Sprintf("Colino database not found at %s", dbPath),
			"hint":    "Run './colino ingest' to create/populate the DB, or set database_path in ~/.config/colino/config.yaml.",
			"db_path": dbPath,
		}, nil
	}
	db, err := colinodb.Open(dbPath)
	if err != nil {
		return nil, map[string]any{
			"ok":      false,
			"message": "Failed opening the Colino database",
			"error":   err.Error(),
			"db_path": dbPath,
		}, nil
	}
	defer db.Close()

	since := time.Now().Add(-time.Duration(p.Hours) * time.Hour)
	src := ""
	if p.Source != nil {
		s := strings.ToLower(strings.TrimSpace(*p.Source))
		if s == "article" || s == "youtube" {
			src = s
		}
	}
	rows, err := colinodb.GetSince(ctx, db, since, src, lim)
	if err != nil {
		// Friendly message if schema is missing (e.g., empty DB file)
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil, map[string]any{
				"ok":      false,
				"message": "Colino database is present but not initialized (missing tables)",
				"hint":    "Run './colino ingest' once to initialize the schema.",
				"db_path": dbPath,
			}, nil
		}
		return nil, map[string]any{
			"ok":      false,
			"message": "Query failed while reading from the Colino database",
			"error":   err.Error(),
			"db_path": dbPath,
		}, nil
	}
	type item struct {
		ID        string          `json:"id"`
		Source    string          `json:"source"`
		Author    string          `json:"author_username"`
		Title     string          `json:"title"`
		URL       string          `json:"url"`
		CreatedAt time.Time       `json:"created_at"`
		Metadata  json.RawMessage `json:"metadata,omitempty"`
		Content   string          `json:"content,omitempty"`
		Preview   string          `json:"content_preview,omitempty"`
	}
	var items []item
	for _, r := range rows {
		it := item{
			ID:        r.ID,
			Source:    r.Source,
			Author:    r.AuthorUsername,
			URL:       nullSQLString(r.URL),
			CreatedAt: r.CreatedAt,
		}
		// Try to derive a title from metadata.entry_title if present
		if r.Metadata.Valid {
			var meta map[string]any
			if json.Unmarshal([]byte(r.Metadata.String), &meta) == nil {
				if t, ok := meta["entry_title"].(string); ok {
					it.Title = t
				}
			}
			it.Metadata = json.RawMessage(r.Metadata.String)
		}
		if p.IncludeContent {
			it.Content = r.Content
		} else {
			if len(r.Content) > 400 {
				it.Preview = r.Content[:400] + "..."
			} else {
				it.Preview = r.Content
			}
		}
		items = append(items, it)
	}
	resp := map[string]any{"count": len(items), "items": items}
	return nil, resp, nil

}

// Handle a get content requests, returning content either by id or url. TODO: we should also allow to fetch content on the fly if it doesn't exist
func handleGetContent(ctx context.Context, req *mcp.CallToolRequest, p GetContentParams) (*mcp.CallToolResult, any, error) {
	dbPath, err := config.LoadDBPath()
	if err != nil {
		return nil, nil, err
	}
	if !fileExists(dbPath) {
		return nil, map[string]any{
			"ok":      false,
			"message": fmt.Sprintf("Colino database not found at %s", dbPath),
			"hint":    "Run './colino ingest' to create/populate the DB, or set database_path in ~/.config/colino/config.yaml.",
			"db_path": dbPath,
		}, nil
	}
	db, err := colinodb.Open(dbPath)
	if err != nil {
		return nil, map[string]any{
			"ok":      false,
			"message": "Failed opening the Colino database",
			"error":   err.Error(),
			"db_path": dbPath,
		}, nil
	}
	defer db.Close()

	var out []map[string]any

	// by IDs
	if len(p.IDs) > 0 {
		for _, id := range p.IDs {
			c, err := colinodb.GetByID(ctx, db, id)
			if err != nil {
				return nil, nil, err
			}
			if c != nil {
				out = append(out, serialize(*c, p.IncludeContent))
			}
		}
		return nil, map[string]any{"count": len(out), "items": out}, nil
	}

	// by URL
	if p.URL != nil && *p.URL != "" {
		c, err := colinodb.GetByURL(ctx, db, *p.URL)
		if err != nil {
			return nil, nil, err
		}
		if c != nil {
			out = append(out, serialize(*c, p.IncludeContent))
		}
		return nil, map[string]any{"count": len(out), "items": out}, nil
	}

	// by window
	hrs := 24
	if p.Hours != nil && *p.Hours > 0 {
		hrs = *p.Hours
	}
	lim := 0
	if p.Limit != nil && *p.Limit > 0 {
		lim = *p.Limit
	}
	src := ""
	if p.Source != nil {
		s := strings.ToLower(strings.TrimSpace(*p.Source))
		if s == "article" || s == "youtube" {
			src = s
		}
	}
	rows, err := colinodb.GetSince(ctx, db, time.Now().Add(-time.Duration(hrs)*time.Hour), src, lim)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil, map[string]any{
				"ok":      false,
				"message": "Colino database is present but not initialized (missing tables)",
				"hint":    "Run './colino ingest' once to initialize the schema.",
				"db_path": dbPath,
			}, nil
		}
		return nil, map[string]any{
			"ok":      false,
			"message": "Query failed while reading from the Colino database",
			"error":   err.Error(),
			"db_path": dbPath,
		}, nil
	}
	for _, r := range rows {
		out = append(out, serialize(r, p.IncludeContent))
	}
	return nil, map[string]any{"count": len(out), "items": out}, nil

}

// Serialize colino db content in a dto
// content is restricted to 400 characters
func serialize(c colinodb.Content, includeContent bool) map[string]any {
	m := map[string]any{
		"id":                  c.ID,
		"source":              c.Source,
		"author_username":     c.AuthorUsername,
		"author_display_name": nullSQLString(c.AuthorDisplayName),
		"url":                 nullSQLString(c.URL),
		"created_at":          c.CreatedAt,
		"like_count":          c.LikeCount,
		"reply_count":         c.ReplyCount,
	}
	if c.Metadata.Valid {
		m["metadata"] = json.RawMessage(c.Metadata.String)
		// try to expose title
		var meta map[string]any
		if json.Unmarshal([]byte(c.Metadata.String), &meta) == nil {
			if t, ok := meta["entry_title"].(string); ok {
				m["title"] = t
			}
		}
	}
	if includeContent {
		m["content"] = c.Content
	} else {
		if len(c.Content) > 400 {
			m["content_preview"] = c.Content[:400] + "..."
		} else {
			m["content_preview"] = c.Content
		}
	}
	return m
}

// Return the string only if valid, otherwise return an empty string
func nullSQLString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Check if a file exists, validating the p search path
func fileExists(p string) bool {
	if p == "" {
		return false
	}
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}
