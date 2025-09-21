package daemon

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/ingest"
)

// Options allow overriding config values from CLI flags.
type Options struct {
    Once          bool
    IntervalMin   int
    SourcesCSV    string
    LogFile       string
}

// Run starts the ingestion daemon. It respects ~/.config/colino/config.yaml
// daemon section, with CLI flags able to override.
func Run(ctx context.Context, opts Options) error {
	dc, _ := config.LoadDaemonConfig()

	if opts.IntervalMin > 0 {
		dc.IntervalMin = opts.IntervalMin
	}
	if strings.TrimSpace(opts.SourcesCSV) != "" {
		var ss []string
        for _, s := range strings.Split(opts.SourcesCSV, ",") {
            s = strings.ToLower(strings.TrimSpace(s))
            if s == "" {
                continue
            }
            if s == "article" || s == "youtube" {
                ss = append(ss, s)
            }
        }
		if len(ss) > 0 {
			dc.Sources = ss
		}
	}
	if strings.TrimSpace(opts.LogFile) != "" {
		dc.LogFile = config.ExpandPath(opts.LogFile)
	}

	logger := log.New(os.Stdout, "[colino-daemon] ", log.LstdFlags)
	var closeLog func() error = func() error { return nil }
	// Default log file if not provided
	if strings.TrimSpace(dc.LogFile) == "" {
		if runtime.GOOS == "darwin" {
			if home, err := os.UserHomeDir(); err == nil {
				dc.LogFile = filepath.Join(home, "Library", "Logs", "Colino", "colino.log")
			}
		} else {
			dc.LogFile = "colino-daemon.log"
		}
	}
	if err := os.MkdirAll(dirOf(dc.LogFile), 0o755); err == nil {
		if f, err := os.OpenFile(dc.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			logger.SetOutput(f)
			closeLog = f.Close
		}
	}
	defer closeLog()

    // one run and exit
    if opts.Once {
        return runGoIngest(ctx, logger, dc.Sources)
    }

	// periodic loop
	logger.Printf("daemon starting (interval=%d min, sources=%v)\n", dc.IntervalMin, dc.Sources)
	ticker := time.NewTicker(time.Duration(dc.IntervalMin) * time.Minute)
	defer ticker.Stop()

	// initial run
	if err := runGoIngest(ctx, logger, dc.Sources); err != nil {
		logger.Printf("initial ingest error: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("daemon stopping: context cancelled")
			return nil
		case <-ticker.C:
			if err := runGoIngest(ctx, logger, dc.Sources); err != nil {
				logger.Printf("ingest error: %v\n", err)
			}
		}
	}
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
    ri := ingest.NewRSSIngestor(appCfg, appCfg.RSSTimeoutSec, logger)
    n, err := ri.Ingest(ctx, db, appCfg.RSSFeeds)
    if err != nil {
        logger.Printf("ingest error: %v", err)
    } else {
        logger.Printf("ingest saved: %d", n)
    }
    logger.Printf("ingest completed: sources=%v", sources)
    return nil
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if strings.EqualFold(strings.TrimSpace(v), s) {
			return true
		}
	}
	return false
}

func dirOf(p string) string {
	i := strings.LastIndex(p, string(os.PathSeparator))
	if i <= 0 {
		return "."
	}
	return p[:i]
}
