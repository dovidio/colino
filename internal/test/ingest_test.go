package test

import (
	"context"
	"fmt"
	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/ingest"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIngest(t *testing.T) {
	// Set universe
	server := httptest.NewServer(NewDemoHandler())
	defer server.Close()

	databasePath := fmt.Sprintf("/tmp/ingest_test_%d.sqlite", time.Now().UnixNano())
	appConfig := config.AppConfig{
		RSSTimeoutSec:       30,
		ScraperMaxWorkers:   5,
		YouTubeProxyEnabled: false,
		WebshareUsername:    "",
		WebsharePassword:    "",
		DatabasePath:        databasePath,
	}
	appConfig.RSSFeeds = append(appConfig.RSSFeeds, fmt.Sprintf("%s/rss", server.URL))

	// Run the ingestion
	db, err := colinodb.Open(appConfig.DatabasePath)
	if err != nil {
		t.Fatalf("Could not open db: %v", err)
	}
	defer db.Close()

	logger := log.New(os.Stdout, "[colino-daemon] ", log.LstdFlags)
	ingestor := ingest.NewRSSIngestor(appConfig, 2000, 0, logger)
	ingestor.Ingest(t.Context(), db)

	// Assert that content was ingested
	err = assertDatabaseContent(t, t.Context(), databasePath)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
}

func assertDatabaseContent(t *testing.T, ctx context.Context, databasePath string) error {
	db, err := colinodb.Open(databasePath)
	if err != nil {
		return err
	}

	content, err := colinodb.GetSince(ctx, db, time.Date(2025, time.August, 1, 0, 0, 0, 0, time.Local), "article", 100)
	if err != nil {
		return err
	}

	if len(content) != 4 {
		t.Fatalf("Expected 4 articles saved, found %d", len(content))
	}

	for index, article := range content {
		if article.Source != "article" {
			t.Fatalf("Expected article %d source to be article, found %s instead", index, article.Source)
		}
		if strings.TrimSpace(article.Content) == "" {
			t.Fatalf("Expected article %d to contain content", index)
		}
		// Check for content that exists in our demo server articles
		if !strings.Contains(article.Content, "Part") && !strings.Contains(article.Content, "seasons") {
			t.Fatalf("Expected article %d to contain expected content markers. Actual content: %s", index, article.Content)
		}
	}

	return nil
}
