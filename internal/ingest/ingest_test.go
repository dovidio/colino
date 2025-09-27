package ingest

import (
	"context"
	"fmt"
	"golino/internal/colinodb"
	"golino/internal/config"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIngest(t *testing.T) {
	// Set universe
	server := httptest.NewServer(createHandler())
	defer server.Close()

	options := Options{
		LogFile: "",
	}
	databasePath := fmt.Sprintf("/tmp/ingest_test_%d.sqlite", time.Now().UnixNano())
	appConfig := config.AppConfig{
		RSSTimeoutSec:       30,
		RSSMaxPostsPerFeed:  100,
		ScraperMaxWorkers:   5,
		YouTubeProxyEnabled: false,
		WebshareUsername:    "",
		WebsharePassword:    "",
		DatabasePath:        databasePath,
	}
	appConfig.RSSFeeds = append(appConfig.RSSFeeds, fmt.Sprintf("%s/rss", server.URL))

	loader := func() (config.AppConfig, error) {
		databasePath := fmt.Sprintf("/tmp/ingest_test_%d.sqlite", time.Now().UnixNano())
		c := config.AppConfig{
			RSSTimeoutSec:       30,
			RSSMaxPostsPerFeed:  100,
			ScraperMaxWorkers:   5,
			YouTubeProxyEnabled: false,
			WebshareUsername:    "",
			WebsharePassword:    "",
			DatabasePath:        databasePath,
		}
		c.RSSFeeds = append(c.RSSFeeds, fmt.Sprintf("%s/rss", server.URL))

		return c, nil
	}

	// Run the ingestion
	Run(t.Context(), options, loader)

	// Assert that content was ingested
	assertDatabaseContent(t, t.Context(), databasePath)
}

// Runs a server that serves mock rss information
// Ideally I would be able to run multiple of them, with different domains so that the scraping can be done in parallel
func createHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rss", rssHandler)
	mux.HandleFunc("/articles/", articlesHandler)
	return mux
}

// Return a list in rss format of all scrapable routes
func rssHandler(w http.ResponseWriter, r *http.Request) {
	pageContent := `
	<?xml version="1.0" encoding="utf-8" standalone="yes"?>
	<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
		<channel>
			<title>Awesome blog</title>
			<link>/</link>
			<description>Recent content on the awesome blog</description>
			<lastBuildDate>Mon, 25 Aug 2025 07:42:16 +0100</lastBuildDate>
			<atom:link href="/index.xml" rel="self" type="application/rss+xml"/>
			<item>
				<title>Breaking News, we have no autumn or spring anymore! Part 1</title>
				<link>/articles/1</link>
				<pubDate>Mon, 25 Aug 2025 07:42:16 +0100</pubDate>
				<guid>/post/2025-08-25>
				<description>Content 1</description>
			</item>
			<item>
				<title>Breaking News, we have no autumn or spring anymore! Part 2</title>
				<link>/articles/2</link>
				<pubDate>Mon, 26 Aug 2025 07:42:16 +0100</pubDate>
				<guid>/post/2025-08-26>
				<description>Content 1</description>
			</item>
			<item>
				<title>Breaking News, we have no autumn or spring anymore! Part 3</title>
				<link>/articles/3</link>
				<pubDate>Mon, 27 Aug 2025 07:42:16 +0100</pubDate>
				<guid>/post/2025-08-27>
				<description>Content 1</description>
			</item>
			<item>
				<title>Breaking News, we have no autumn or spring anymore! Part 4</title>
				<link>/articles/4</link>
				<pubDate>Mon, 28 Aug 2025 07:42:16 +0100</pubDate>
				<guid>/post/2025-08-28>
				<description>Content 1</description>
			</item>
		</channel>
	</rss>
	`

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(pageContent))
}

func articlesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/articles/")
	if path == "" {
		http.Error(w, "Articles ID required", http.StatusBadRequest)
		return
	}

	articleID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Articles ID could not be converted", http.StatusBadRequest)
		return
	}

	// let's build our "page"
	pageContent := `
		<html>
			<head>
			</head>
			<body>
				<h1>Breaking News, we have no autumn or spring anymore! Part %d</h1>
				<p>It occurred to me that we only have two seasons, summer and winter, and our weather abruptly switch between those.</p>
			</body>
		</html>
	`

	pageContent = fmt.Sprintf(pageContent, articleID)

	w.Header().Set("Content-Type", "text/html")

	w.Write([]byte(pageContent))
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
		t.Fatal(fmt.Sprintf("Expected 4 articles saved, found %d", len(content)))
	}
	return nil
}
