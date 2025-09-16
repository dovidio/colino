package rss

import (
    "context"
    "encoding/xml"
    "errors"
    "io"
    "net/http"
    "strings"
    "time"

    "golino/internal/models"
)

type rssFeed struct {
    Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
    Title string   `xml:"title"`
    Link  string   `xml:"link"`
    Desc  string   `xml:"description"`
    Items []rssItem `xml:"item"`
}

type rssItem struct {
    GUID        string `xml:"guid"`
    Title       string `xml:"title"`
    Link        string `xml:"link"`
    Description string `xml:"description"`
    PubDate     string `xml:"pubDate"`
    Content     string `xml:"encoded"`
}

// Fetch retrieves an RSS feed and converts items to models.Item.
func Fetch(ctx context.Context, client *http.Client, sourceURL string, sourceType string, userAgent string) ([]models.Item, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
    if err != nil {
        return nil, err
    }
    if userAgent != "" {
        req.Header.Set("User-Agent", userAgent)
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(&io.LimitedReader{R: resp.Body, N: 1024})
        return nil, errors.New(string(b))
    }
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    // Some feeds may include content:encoded in a different namespace
    // Normalize by removing common namespace prefixes for minimal parsing.
    normalized := strings.ReplaceAll(string(body), "content:encoded", "encoded")
    var feed rssFeed
    if err := xml.Unmarshal([]byte(normalized), &feed); err != nil {
        return nil, err
    }
    var out []models.Item
    for _, it := range feed.Channel.Items {
        t := parseTime(it.PubDate)
        out = append(out, models.Item{
            GUID:       firstNonEmpty(it.GUID, it.Link, it.Title),
            SourceURL:  sourceURL,
            SourceType: sourceType,
            Title:      strings.TrimSpace(it.Title),
            Link:       strings.TrimSpace(it.Link),
            Published:  t,
            Summary:    strings.TrimSpace(it.Description),
            Content:    strings.TrimSpace(it.Content),
        })
    }
    return out, nil
}

func parseTime(s string) time.Time {
    s = strings.TrimSpace(s)
    layouts := []string{
        time.RFC1123Z,
        time.RFC1123,
        time.RFC822Z,
        time.RFC822,
        time.RFC3339,
        "Mon, 02 Jan 2006 15:04:05 MST",
    }
    for _, l := range layouts {
        if t, err := time.Parse(l, s); err == nil {
            return t.UTC()
        }
    }
    return time.Time{}
}

func firstNonEmpty(ss ...string) string {
    for _, s := range ss {
        if strings.TrimSpace(s) != "" {
            return strings.TrimSpace(s)
        }
    }
    return ""
}
