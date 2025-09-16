package models

import "time"

type Item struct {
    ID         int64
    GUID       string
    SourceURL  string
    SourceType string // rss|youtube
    Title      string
    Link       string
    Published  time.Time
    Summary    string
    Content    string
    CreatedAt  time.Time
}
