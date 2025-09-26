package ingest

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golino/internal/colinodb"
	"golino/internal/config"
)

// Options allow overriding config values from CLI flags.
type Options struct {
	LogFile string
}

// Run executes a single ingestion run. Scheduling is delegated to launchd/systemd/cron.
func Run(ctx context.Context, opts Options) error {
	// Always ingest all sources (article, youtube)
	sources := []string{"article", "youtube"}
	logFile := strings.TrimSpace(opts.LogFile)
	if logFile != "" {
		logFile = config.ExpandPath(logFile)
	}

	logger := log.New(os.Stdout, "[colino-daemon] ", log.LstdFlags)
	var closeLog func() error = func() error { return nil }
	// Default log file if not provided
	if logFile == "" {
		if runtime.GOOS == "darwin" {
			if home, err := os.UserHomeDir(); err == nil {
				logFile = filepath.Join(home, "Library", "Logs", "Colino", "colino.log")
			}
		} else {
			logFile = "colino-daemon.log"
		}
	}
	if err := os.MkdirAll(dirOf(logFile), 0o755); err == nil {
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			logger.SetOutput(f)
			closeLog = f.Close
		}
	}
	defer closeLog()

	// single run and exit; scheduling handled externally (e.g., launchd)
	return runGoIngest(ctx, logger, sources)
}

func runGoIngest(ctx context.Context, logger *log.Logger, sources []string) error {
	// Load app config for feeds
	appCfg, _ := config.LoadAppConfig()
	dbPath, err := config.LoadDBPath()
	if err != nil {
		return err
	}
	dbPath = config.ExpandPath(dbPath)
	db, err := colinodb.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Always run ingestion of RSS feeds; entries are saved as source="article" or "youtube" based on URL.
	ri := NewRSSIngestor(appCfg, appCfg.RSSTimeoutSec, logger)
	n, err := ri.Ingest(ctx, db, appCfg.RSSFeeds)
	if err != nil {
		logger.Printf("ingest error: %v", err)
	} else {
		logger.Printf("ingest saved: %d", n)
	}
	logger.Printf("ingest completed")
	return nil
}

func dirOf(p string) string {
	i := strings.LastIndex(p, string(os.PathSeparator))
	if i <= 0 {
		return "."
	}
	return p[:i]
}
