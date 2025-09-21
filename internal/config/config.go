package config

import (
    "errors"
    "os"
    "path/filepath"
    "runtime"
    "strings"

    "gopkg.in/yaml.v3"
)

// LoadDBPath returns the SQLite DB path used by Colino.
// It understands both Python Colino config (database.path) and
// Golino config (database_path). Falls back to platform default.
func LoadDBPath() (string, error) {
    // Prefer user config at ~/.config/colino/config.yaml
    cfgPath, err := defaultConfigPath()
    if err == nil {
        if p, err := readDBPathFrom(cfgPath); err == nil && p != "" {
            return ExpandPath(p), nil
        }
        if !errors.Is(err, os.ErrNotExist) && err != nil {
            // if parsing failed for other reasons, we still fall back
            _ = err
        }
    }
    // Fallback to platform default
    if runtime.GOOS == "darwin" {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, "Library", "Application Support", "Colino", "colino.db"), nil
    }
    return "colino.db", nil
}

func defaultConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".config", "colino", "config.yaml"), nil
}

func readDBPathFrom(path string) (string, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    var raw map[string]any
    if err := yaml.Unmarshal(b, &raw); err != nil {
        return "", err
    }
    // Python config: database.path
    if db, ok := raw["database"].(map[string]any); ok {
        if p, ok := db["path"].(string); ok && p != "" {
            return p, nil
        }
    }
    // Golino config: database_path
    if p, ok := raw["database_path"].(string); ok && p != "" {
        return p, nil
    }
    return "", nil
}

// ExpandPath expands leading ~ and environment variables in a filesystem path.
func ExpandPath(p string) string {
    if p == "" {
        return p
    }
    // Expand environment variables like $HOME
    p = os.ExpandEnv(p)
    // Expand leading ~
    if strings.HasPrefix(p, "~") {
        if home, err := os.UserHomeDir(); err == nil {
            if p == "~" {
                p = home
            } else if strings.HasPrefix(p, "~/") {
                p = filepath.Join(home, p[2:])
            }
        }
    }
    return p
}

// DaemonConfig captures scheduling/ingestion settings for the Go daemon.
type DaemonConfig struct {
    Enabled        bool
    IntervalMin    int
    Sources        []string
    LogFile        string
}

// LoadDaemonConfig reads daemon configuration from the user YAML config if present.
// It is tolerant to missing config and returns sensible defaults.
func LoadDaemonConfig() (DaemonConfig, error) {
    // defaults
    dc := DaemonConfig{
        Enabled:       false,
        IntervalMin:   30,
        Sources:       []string{"article", "youtube"},
        LogFile:       "",
    }

    cfgPath, err := defaultConfigPath()
    if err != nil {
        return dc, nil // keep defaults
    }
    b, err := os.ReadFile(cfgPath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return dc, nil
        }
        // unreadable config: return defaults
        return dc, nil
    }
    var raw map[string]any
    if err := yaml.Unmarshal(b, &raw); err != nil {
        return dc, nil
    }
    // daemon section
    if d, ok := raw["daemon"].(map[string]any); ok {
        if v, ok := d["enabled"].(bool); ok {
            dc.Enabled = v
        }
        if v, ok := d["interval_minutes"].(int); ok && v > 0 {
            dc.IntervalMin = v
        } else if vf, ok := d["interval_minutes"].(float64); ok && int(vf) > 0 {
            dc.IntervalMin = int(vf)
        }
        if s, ok := d["sources"].([]any); ok {
            var out []string
            for _, it := range s {
                if str, ok := it.(string); ok {
                    out = append(out, strings.ToLower(strings.TrimSpace(str)))
                }
            }
            if len(out) > 0 {
                dc.Sources = out
            }
        }
        if v, ok := d["log_file"].(string); ok && strings.TrimSpace(v) != "" {
            dc.LogFile = ExpandPath(strings.TrimSpace(v))
        }
    }
    return dc, nil
}

// AppConfig carries ingestion-related settings.
type AppConfig struct {
    RSSFeeds          []string
    RSSTimeoutSec     int
    RSSMaxPostsPerFeed int
    ScraperMaxWorkers int
    // Optional YouTube transcript/proxy settings
    YouTubeProxyEnabled bool
    WebshareUsername    string
    WebsharePassword    string
}

// LoadAppConfig parses relevant ingestion config from ~/.config/colino/config.yaml.
func LoadAppConfig() (AppConfig, error) {
    ac := AppConfig{
        RSSTimeoutSec:        30,
        RSSMaxPostsPerFeed:   100,
        ScraperMaxWorkers:    5,
        YouTubeProxyEnabled:  false,
        WebshareUsername:     "",
        WebsharePassword:     "",
    }
    cfgPath, err := defaultConfigPath()
    if err != nil {
        return ac, nil
    }
    b, err := os.ReadFile(cfgPath)
    if err != nil {
        return ac, nil
    }
    var raw map[string]any
    if err := yaml.Unmarshal(b, &raw); err != nil {
        return ac, nil
    }
    if rss, ok := raw["rss"].(map[string]any); ok {
        if feeds, ok := rss["feeds"].([]any); ok {
            for _, it := range feeds {
                if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
                    ac.RSSFeeds = append(ac.RSSFeeds, s)
                }
            }
        }
        if v, ok := rss["timeout"].(int); ok && v > 0 {
            ac.RSSTimeoutSec = v
        } else if vf, ok := rss["timeout"].(float64); ok && int(vf) > 0 {
            ac.RSSTimeoutSec = int(vf)
        }
        if v, ok := rss["max_posts_per_feed"].(int); ok && v > 0 {
            ac.RSSMaxPostsPerFeed = v
        } else if vf, ok := rss["max_posts_per_feed"].(float64); ok && int(vf) > 0 {
            ac.RSSMaxPostsPerFeed = int(vf)
        }
        if v, ok := rss["scraper_max_workers"].(int); ok && v > 0 {
            ac.ScraperMaxWorkers = v
        } else if vf, ok := rss["scraper_max_workers"].(float64); ok && int(vf) > 0 {
            ac.ScraperMaxWorkers = int(vf)
        }
    }
    // Optional: youtube proxy settings (used when fetching transcripts for YouTube links found in RSS)
    if yt, ok := raw["youtube"].(map[string]any); ok {
        if proxy, ok := yt["proxy"].(map[string]any); ok {
            if v, ok := proxy["enabled"].(bool); ok {
                ac.YouTubeProxyEnabled = v
            }
            if ws, ok := proxy["webshare"].(map[string]any); ok {
                if u, ok := ws["username"].(string); ok {
                    ac.WebshareUsername = strings.TrimSpace(u)
                }
                if p, ok := ws["password"].(string); ok {
                    ac.WebsharePassword = strings.TrimSpace(p)
                }
                // filter_ip_locations and retries_when_blocked are no longer supported
            }
        }
    }
    // filters, ai, and default_lookback are intentionally ignored now.
    return ac, nil
}
