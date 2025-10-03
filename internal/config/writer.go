package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"gopkg.in/yaml.v3"
)

// UserConfig represents the user configuration used during setup
type UserConfig struct {
	RSSFeeds     []string         `yaml:"rss_feeds"`
	IntervalMin  int             `yaml:"interval_min"`
	WebshareUser string          `yaml:"webshare_user,omitempty"`
	WebsharePass string          `yaml:"webshare_pass,omitempty"`
	YTNameByURL  map[string]string `yaml:"yt_name_by_url,omitempty"`
	DatabasePath string          `yaml:"database,omitempty"`
	AI           *AIConfig       `yaml:"ai,omitempty"`
}


// WriteConfig writes the user configuration to the config file
func WriteConfig(uc UserConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".config", "colino")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")

	// Preserve existing database_path if present (avoid clobber). If no file exists, no-op.
	prevDB := ""
	if prev, err := loadExistingConfig(path); err == nil {
		if v, ok := prev["database_path"].(string); ok && strings.TrimSpace(v) != "" {
			prevDB = v
		}
	}

	// Manually render YAML so we can attach comments to YouTube feeds
	var sb strings.Builder
	sb.WriteString("# Colino configuration\n")

	// Database path (use existing if preserved, otherwise use default)
	dbPath := uc.DatabasePath
	if strings.TrimSpace(prevDB) != "" {
		dbPath = prevDB
	}
	if strings.TrimSpace(dbPath) != "" {
		sb.WriteString("database:\n")
		sb.WriteString(fmt.Sprintf("  path: %q\n", dbPath))
	}

	// RSS feeds
	if len(uc.RSSFeeds) > 0 {
		sb.WriteString("rss:\n")
		sb.WriteString("  feeds:\n")
		for _, u := range uc.RSSFeeds {
			line := fmt.Sprintf("    - %s", strings.TrimSpace(u))
			if uc.YTNameByURL != nil {
				if name, ok := uc.YTNameByURL[u]; ok && strings.TrimSpace(name) != "" {
					line += fmt.Sprintf("  # YouTube: %s", name)
				}
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	// YouTube proxy configuration
	if strings.TrimSpace(uc.WebshareUser) != "" && strings.TrimSpace(uc.WebsharePass) != "" {
		sb.WriteString("youtube:\n")
		sb.WriteString("  proxy:\n")
		sb.WriteString("    enabled: true\n")
		sb.WriteString("    webshare:\n")
		sb.WriteString(fmt.Sprintf("      username: %q\n", uc.WebshareUser))
		sb.WriteString(fmt.Sprintf("      password: %q\n", uc.WebsharePass))
	}

	// AI configuration
	if uc.AI != nil {
		sb.WriteString("ai:\n")
		if strings.TrimSpace(uc.AI.Model) != "" {
			sb.WriteString(fmt.Sprintf("  model: %q\n", uc.AI.Model))
		}
		if strings.TrimSpace(uc.AI.BaseUrl) != "" {
			sb.WriteString(fmt.Sprintf("  base_url: %q\n", uc.AI.BaseUrl))
		}
		if strings.TrimSpace(uc.AI.ArticlePrompt) != "" {
			sb.WriteString("  article_prompt: |\n")
			for _, line := range strings.Split(uc.AI.ArticlePrompt, "\n") {
				sb.WriteString("    " + line + "\n")
			}
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// loadExistingConfig loads existing configuration from a file
func loadExistingConfig(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// BackupFile creates a backup of the specified file with a timestamp
func BackupFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ts := time.Now().Format("20060102-150405")
	bak := path + ".bak-" + ts
	return os.WriteFile(bak, b, 0o644)
}