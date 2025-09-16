package ingest

import (
    "context"
    "database/sql"
    "net/http"
    "runtime"
    "sync"
    "time"

    "golino/internal/config"
    "golino/internal/db"
    "golino/internal/models"
    "golino/internal/sources/rss"
    "golino/internal/sources/youtube"
    "golino/internal/ui"
)

type task struct {
    url        string
    sourceType string
}

func Run(ctx context.Context, cfg config.Config, database *sql.DB, progress *ui.Progress) error {
    tasks := make([]task, 0, len(cfg.Sources.RSS)+len(cfg.Sources.YouTube))
    for _, u := range cfg.Sources.RSS {
        tasks = append(tasks, task{url: u, sourceType: "rss"})
    }
    for _, u := range cfg.Sources.YouTube {
        tasks = append(tasks, task{url: u, sourceType: "youtube"})
    }

    if len(tasks) == 0 {
        return nil
    }

    conc := cfg.Ingest.Concurrency
    if conc <= 0 {
        conc = runtime.NumCPU()
        if conc < 4 {
            conc = 4
        }
    }

    progress.Start(len(tasks), "Fetching sources...")
    defer progress.Stop()

    client := &http.Client{Timeout: time.Duration(cfg.Ingest.TimeoutSecond) * time.Second}

    chTasks := make(chan task)
    var wg sync.WaitGroup
    wg.Add(conc)

    for i := 0; i < conc; i++ {
        go func() {
            defer wg.Done()
            for t := range chTasks {
                var items []models.Item
                var err error
                switch t.sourceType {
                case "rss":
                    items, err = rss.Fetch(ctx, client, t.url, "rss", cfg.Ingest.UserAgent)
                case "youtube":
                    items, err = youtube.Fetch(ctx, client, t.url, cfg.Ingest.UserAgent)
                }
                // Insert items
                if err == nil {
                    for _, it := range items {
                        _, _ = db.UpsertItem(ctx, database, it)
                    }
                }
                progress.Increment(t.sourceType + ": " + t.url)
            }
        }()
    }

    go func() {
        defer close(chTasks)
        for _, t := range tasks {
            select {
            case <-ctx.Done():
                return
            case chTasks <- t:
            }
        }
    }()

    wg.Wait()
    return nil
}
