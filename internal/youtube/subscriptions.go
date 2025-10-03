package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golino/internal/httpclient"
)

// SubscriptionsService handles YouTube subscription operations
type SubscriptionsService struct {
	client *httpclient.Client
}

// NewSubscriptionsService creates a new subscriptions service
func NewSubscriptionsService() *SubscriptionsService {
	client := httpclient.New(20 * time.Second)
	return &SubscriptionsService{client: client}
}

// FetchSubscriptions retrieves the user's YouTube subscriptions
func (s *SubscriptionsService) FetchSubscriptions(ctx context.Context, accessToken string) ([]Channel, error) {
	var channels []Channel
	baseURL := "https://www.googleapis.com/youtube/v3/subscriptions?mine=true&part=snippet&maxResults=50"
	pageToken := ""

	for {
		requestURL := baseURL
		if pageToken != "" {
			requestURL += "&pageToken=" + url.QueryEscape(pageToken)
		}

		resp, err := s.client.Get(ctx, requestURL, map[string]string{
			"Authorization": "Bearer " + accessToken,
		})
		if err != nil {
			return channels, fmt.Errorf("failed to fetch subscriptions: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return channels, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		var response struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Snippet struct {
					Title      string `json:"title"`
					ResourceID struct {
						ChannelID string `json:"channelId"`
					} `json:"resourceId"`
				} `json:"snippet"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return channels, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, item := range response.Items {
			id := strings.TrimSpace(item.Snippet.ResourceID.ChannelID)
			title := strings.TrimSpace(item.Snippet.Title)
			if id != "" && title != "" {
				channels = append(channels, Channel{
					ID:    id,
					Title: title,
				})
			}
		}

		if strings.TrimSpace(response.NextPageToken) == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	return channels, nil
}

// ChannelsToRSSFeeds converts YouTube channels to RSS feed URLs
func ChannelsToRSSFeeds(channels []Channel) []string {
	var feeds []string
	for _, channel := range channels {
		feedURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channel.ID)
		feeds = append(feeds, feedURL)
	}
	return feeds
}

// ChannelsToNameMap creates a mapping from RSS feed URLs to channel names
func ChannelsToNameMap(channels []Channel) map[string]string {
	nameMap := make(map[string]string)
	for _, channel := range channels {
		feedURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channel.ID)
		nameMap[feedURL] = channel.Title
	}
	return nameMap
}
