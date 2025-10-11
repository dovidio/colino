package list

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golino/internal/colinodb"
	"golino/internal/config"
)

func Run(ctx context.Context, hours int) error {
	if hours <= 0 {
		hours = 24
	}

	dbPath, err := config.LoadDBPath()
	if err != nil {
		return err
	}
	if !fileExists(dbPath) {
		fmt.Printf("Colino database not found at %s\n", dbPath)
		fmt.Println("Hint: Run './colino daemon' to create/populate the DB, or set database_path in ~/.config/colino/config.yaml.")
		return nil
	}

	db, err := colinodb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed opening the Colino database: %w", err)
	}
	defer db.Close()

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	rows, err := colinodb.GetSince(ctx, db, since, "", 0) // no source filter, no limit
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			fmt.Println("Colino database is present but not initialized (missing tables)")
			fmt.Println("Hint: Run './colino ingest' once to initialize the schema.")
			return nil
		}
		return fmt.Errorf("query failed while reading from the Colino database: %w", err)
	}

	if len(rows) == 0 {
		fmt.Printf("No content found in the last %d hours.\n", hours)
		return nil
	}

	fmt.Printf("Found %d items from the last %d hours:\n\n", len(rows), hours)

	for _, r := range rows {
		title := extractTitle(r.Metadata)
		if title == "" {
			title = "No title"
		}

		author := r.AuthorUsername
		if author == "" {
			author = "Unknown author"
		}

		preview := r.Content
		if len(preview) > 400 {
			preview = preview[:400] + "..."
		}

		fmt.Printf("ID: %s\n", r.ID)
		fmt.Printf("Title: %s\n", title)
		fmt.Printf("Author: %s\n", author)
		fmt.Printf("Source: %s\n", r.Source)
		fmt.Printf("Date: %s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Preview: %s\n", preview)
		fmt.Println(strings.Repeat("-", 80))
	}

	return nil
}

func extractTitle(metadata sql.NullString) string {
	if !metadata.Valid {
		return ""
	}

	// Try to parse as JSON and extract entry_title
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(metadata.String), &meta); err != nil {
		return ""
	}

	if title, ok := meta["entry_title"].(string); ok {
		return title
	}

	return ""
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
