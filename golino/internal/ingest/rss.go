package ingest

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
	"github.com/mmcdole/gofeed"

	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/youtube"
)

// RSSIngestor fetches RSS/Atom feeds and persists full content into the Colino DB.
type RSSIngestor struct {
	AppCfg      config.AppConfig
	Client      *http.Client
	Logger      *log.Logger
	parser      *gofeed.Parser
	minInterval time.Duration
}

// NewRSSIngestor constructs an RSS ingestor with sensible defaults.
func NewRSSIngestor(appCfg config.AppConfig, timeoutSec int, logger *log.Logger) *RSSIngestor {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	cli := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	p := gofeed.NewParser()
	p.Client = cli
	minInt := 1500 * time.Millisecond
	if appCfg.ScraperMaxWorkers > 8 { // be a bit more gentle when highly parallel
		minInt = 2 * time.Second
	}
	return &RSSIngestor{AppCfg: appCfg, Client: cli, Logger: logger, parser: p, minInterval: minInt}
}

type rssTask struct {
	FeedTitle string
	FeedURL   string
	Entry     *gofeed.Item
	Host      string
}

func (ri *RSSIngestor) debugf(format string, args ...any) {
	if ri.Logger != nil {
		ri.Logger.Printf(format, args...)
	}
}

// Ingest fetches all provided feed URLs and stores new items into DB.
func (ri *RSSIngestor) Ingest(ctx context.Context, db *sql.DB, feeds []string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("nil db")
	}
	// Ensure schema exists to avoid failures on fresh DBs
	if err := colinodb.InitSchema(db); err != nil {
		return 0, err
	}

	// Preload existing URLs to avoid obvious duplicates upfront
	existingURLSet, _ := colinodb.GetURLsBySource(ctx, db, "rss")

	// Fetch all feeds concurrently
	type feedResult struct {
		url  string
		host string
		feed *gofeed.Feed
		err  error
	}
	var wgFeeds sync.WaitGroup
	resCh := make(chan feedResult, len(feeds))
	for _, raw := range feeds {
		feedURL := strings.TrimSpace(raw)
		if feedURL == "" {
			continue
		}
		host := func() string {
			if u, err := neturl.Parse(feedURL); err == nil {
				return u.Host
			}
			return ""
		}()
		wgFeeds.Add(1)
		go func(feedURL, host string) {
			defer wgFeeds.Done()
			f, err := ri.parser.ParseURLWithContext(feedURL, ctx)
			// Be polite between subsequent feed requests from same goroutine
			select {
			case <-ctx.Done():
				return
			case <-time.After(ri.minInterval):
			}
			resCh <- feedResult{url: feedURL, host: host, feed: f, err: err}
		}(feedURL, host)
	}

	go func() { wgFeeds.Wait(); close(resCh) }()

	// Build tasks from feeds
	tasks := make([]rssTask, 0, 128)
	for r := range resCh {
		if r.err != nil || r.feed == nil {
			ri.debugf("rss feed fetch failed: host=%s url=%s err=%v", r.host, r.url, r.err)
			continue
		}
		skippedExisting := 0
		count := 0
		for _, it := range r.feed.Items {
			if ri.AppCfg.RSSMaxPostsPerFeed > 0 && count >= ri.AppCfg.RSSMaxPostsPerFeed {
				break
			}
			if it == nil {
				continue
			}
			// Skip if URL already present (cheap pre-filter)
			if u := strings.TrimSpace(it.Link); u != "" {
				if _, ok := existingURLSet[u]; ok {
					skippedExisting++
					continue
				}
			}
			host := r.host
			if u, err := neturl.Parse(it.Link); err == nil && u.Host != "" {
				host = u.Host
			}
			tasks = append(tasks, rssTask{FeedTitle: r.feed.Title, FeedURL: r.url, Entry: it, Host: host})
			count++
		}
		if ri.Logger != nil {
			ri.Logger.Printf("rss feed parsed: host=%s url=%s items=%d queued=%d skipped_existing=%d", r.host, r.url, len(r.feed.Items), count, skippedExisting)
		}
	}

	// Scrape per host: one worker per domain
	tasksByHost := map[string][]rssTask{}
	for _, t := range tasks {
		h := strings.TrimSpace(t.Host)
		if h == "" {
			h = "__unknown__"
		}
		tasksByHost[h] = append(tasksByHost[h], t)
	}
	if ri.Logger != nil {
		ri.Logger.Printf("rss ingest: scraping hosts=%d tasks=%d", len(tasksByHost), len(tasks))
	}
	var wgScrape sync.WaitGroup
	saved := 0
	processed := 0
	mu := sync.Mutex{}
	for _, list := range tasksByHost {
		items := list
		wgScrape.Add(1)
		go func() {
			defer wgScrape.Done()
			for _, t := range items {
				did, _ := ri.processOne(ctx, db, t, &saved, &mu)
				mu.Lock()
				processed++
				mu.Unlock()
				if ctx.Err() != nil {
					return
				}
				if did { // pace only when we actually fetched from the site
					select {
					case <-ctx.Done():
						return
					case <-time.After(ri.minInterval):
					}
				}
			}
		}()
	}
	wgScrape.Wait()
	if ri.Logger != nil {
		ri.Logger.Printf("rss ingest: done tasks=%d saved=%d processed=%d", len(tasks), saved, processed)
	}
	return saved, nil
}

func (ri *RSSIngestor) processOne(ctx context.Context, db *sql.DB, t rssTask, saved *int, mu *sync.Mutex) (bool, error) {
	it := t.Entry
	if it == nil {
		ri.debugf("skip process (nil entry): feed_url=%s", t.FeedURL)
		return false, nil
	}
	id := firstNonEmpty(it.GUID, it.Link)
	if id == "" {
		ri.debugf("skip process (no id): url=%s title=%q", it.Link, it.Title)
		return false, nil
	}
	url := it.Link
	// Skip if already cached by ID or URL (N+1/2 queries; acceptable for local SQLite)
	if existing, err := colinodb.GetByID(ctx, db, id); err == nil && existing != nil {
		ri.debugf("skip process (db exists id): id=%s url=%s", id, url)
		return false, nil
	}
	if url != "" {
		if byURL, err := colinodb.GetByURL(ctx, db, url); err == nil && byURL != nil {
			ri.debugf("skip process (db exists url): url=%s", url)
			return false, nil
		}
	}
	content := firstNonEmpty(it.Content, it.Description)
	title := it.Title
	createdAt := time.Now().UTC()
	if it.PublishedParsed != nil {
		createdAt = it.PublishedParsed.UTC()
	} else if it.UpdatedParsed != nil {
		createdAt = it.UpdatedParsed.UTC()
	}

    // If YouTube video, fetch transcript instead of readability extraction (Webshare proxy optional via config)
    didFetch := false
    if isYouTubeURL(url) {
        var ws *youtube.WebshareProxyConfig
        if ri.AppCfg.YouTubeProxyEnabled && strings.TrimSpace(ri.AppCfg.WebshareUsername) != "" && strings.TrimSpace(ri.AppCfg.WebsharePassword) != "" {
            ws = &youtube.WebshareProxyConfig{
                Username: ri.AppCfg.WebshareUsername,
                Password: ri.AppCfg.WebsharePassword,
            }
        }
        if vid := extractYouTubeID(url); vid != "" {
            if snippets, err := youtube.FetchDefaultTranscript(ctx, nil, vid, ws); err == nil && len(snippets) > 0 {
                didFetch = true
                var sb strings.Builder
                for _, sn := range snippets {
                    line := strings.TrimSpace(sn.Text)
                    if line == "" {
                        continue
                    }
                    if sb.Len() > 0 {
                        sb.WriteString("\n")
                    }
                    sb.WriteString(line)
                }
                tr := strings.TrimSpace(sb.String())
                if tr != "" {
                    content = "YouTube Transcript:\n" + tr
                }
            } else {
                ri.debugf("yt transcript unavailable: url=%s err=%v", url, err)
            }
        }
    }
	// scrape full text and enhance (fallback / non-YouTube)
	if !isYouTubeURL(url) || strings.TrimSpace(content) == "" || !strings.Contains(content, "YouTube Transcript:") {
		full := ri.extractMainText(ctx, url)
		if full != "" {
			didFetch = true
		}
		if len(full) != len(content) || full != content {
			content = content + "\nFull Content:\n" + full
		}
		if full == "" {
			ri.debugf("content fetch empty: url=%s", url)
		}
	}

    meta := fmt.Sprintf(`{"feed_url":%q,"feed_title":%q,"entry_title":%q}`, t.FeedURL, t.FeedTitle, title)
    src := "article"
    if isYouTubeURL(url) {
        src = "youtube"
    }
    rec := colinodb.ContentInsert{
        ID:                id,
        Source:            src,
        AuthorUsername:    t.FeedTitle,
        AuthorDisplayName: t.FeedTitle,
        Content:           content,
        URL:               url,
        CreatedAt:         createdAt,
		MetadataJSON:      meta,
		LikeCount:         0,
		ReplyCount:        0,
	}
	if err := colinodb.UpsertContent(ctx, db, rec); err != nil {
		ri.debugf("upsert failed: id=%s url=%s err=%v", id, url, err)
		return didFetch, err
	}
	ri.debugf("upsert ok: id=%s url=%s", id, url)
	mu.Lock()
	*saved++
	mu.Unlock()
	return didFetch, nil
}

func (ri *RSSIngestor) extractMainText(ctx context.Context, url string) string {
	if strings.TrimSpace(url) == "" {
		return ""
	}
	// fetch
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "Colino/Go-Ingestor")
	resp, err := ri.Client.Do(req)
	if err != nil || resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	// read once, reuse for readability and goquery to avoid double fetching
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(bodyBytes) == 0 {
		return ""
	}
	// readability
	base, _ := neturl.Parse(url)
	art, err := readability.FromReader(bytes.NewReader(bodyBytes), base)
	if err == nil && len(strings.TrimSpace(art.TextContent)) > 100 {
		return strings.TrimSpace(art.TextContent)
	}
	// fallback: goquery
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
	if err != nil {
		return ""
	}
	selectors := []string{"article", "main", "#content", ".post", ".article"}
	for _, sel := range selectors {
		if s := strings.TrimSpace(doc.Find(sel).Text()); len(s) > 100 {
			return s
		}
	}
	// last resort: entire text
	body := strings.TrimSpace(doc.Text())
	if len(body) > 200 {
		return body
	}
	return ""
}

// Filtering removed: we ingest everything; LLM will filter downstream.
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var ytHostRe = regexp.MustCompile(`(?i)(^|\.)youtube\.com$`)

func isYouTubeURL(u string) bool {
	if strings.TrimSpace(u) == "" {
		return false
	}
	parsed, err := neturl.Parse(u)
	if err != nil {
		return false
	}
	h := strings.ToLower(parsed.Host)
	if h == "youtu.be" {
		return true
	}
	return ytHostRe.MatchString(h)
}

func extractYouTubeID(u string) string {
	parsed, err := neturl.Parse(u)
	if err != nil {
		return ""
	}
	h := strings.ToLower(parsed.Host)
	if h == "youtu.be" {
		return strings.Trim(parsed.Path, "/")
	}
	if ytHostRe.MatchString(h) {
		if strings.HasPrefix(parsed.Path, "/watch") {
			q := parsed.Query()
			return strings.TrimSpace(q.Get("v"))
		}
		if strings.HasPrefix(parsed.Path, "/shorts/") {
			return strings.Trim(strings.TrimPrefix(parsed.Path, "/shorts/"), "/")
		}
	}
	return ""
}
