package tui

import (
	"database/sql"
	"encoding/json"

	"golino/internal/colinodb"
)

func contentToArticleDetail(item *colinodb.Content) *articleDetail {
	return &articleDetail{
		content:        item.Content,
		url:            item.URL.String,
		metadata:       item.Metadata.String,
		authorUsername: item.AuthorUsername,
		createdAt:      item.CreatedAt,
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func extractTitleM(metadata sql.NullString) string {
	if !metadata.Valid {
		return ""
	}

	return extractField(metadata.String, "entry_title")
}

func extractContentTypeM(metadata sql.NullString) string {
	if !metadata.Valid {
		return ""
	}

	return extractField(metadata.String, "content_type")
}

func extractTitle(metadata string) string {
	return extractField(metadata, "entry_title")
}

func extractContentType(metadata string) string {
	return extractField(metadata, "content_type")
}

func extractField(metadata string, field string) string {
	if metadata == "" {
		return ""
	}

	var meta map[string]any
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
		return ""
	}

	if fieldValue, ok := meta[field].(string); ok {
		return fieldValue
	}

	return ""

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
