package youtube

import (
    "context"
    "net/http"
    "regexp"
    "strings"

    "golino/internal/models"
    "golino/internal/sources/rss"
)

var channelRe = regexp.MustCompile(`(?i)youtube\.com/(?:@[^/]+|channel/([A-Za-z0-9_-]+))`)

// BuildFeedURL attempts to convert a YouTube channel URL or handle into a feed URL.
func BuildFeedURL(input string) string {
    in := strings.TrimSpace(input)
    if strings.Contains(in, "feeds/videos.xml") {
        return in
    }
    // If it's a channel ID
    if strings.Contains(in, "/channel/") {
        parts := strings.Split(in, "/channel/")
        id := parts[len(parts)-1]
        id = strings.Trim(id, "/")
        if id != "" {
            return "https://www.youtube.com/feeds/videos.xml?channel_id=" + id
        }
    }
    // If it's a handle, YouTube doesn't offer a direct feed; users should prefer channel IDs.
    // We'll just return input; rss.Fetch may fail if invalid.
    return in
}

func Fetch(ctx context.Context, client *http.Client, inputURL string, userAgent string) ([]models.Item, error) {
    url := BuildFeedURL(inputURL)
    return rss.Fetch(ctx, client, url, "youtube", userAgent)
}
