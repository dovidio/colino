package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/spf13/cobra"

    "golino/internal/config"
    "golino/internal/db"
    "golino/internal/digest"
    "golino/internal/ingest"
    "golino/internal/ui"
)

func main() {
    root := &cobra.Command{
        Use:   "golino",
        Short: "Golino is a Go port of the Colino CLI",
        Long:  "Golino provides fast ingestion of RSS/YouTube feeds with a friendly terminal UI.",
    }

    var cfgPath string
    root.PersistentFlags().StringVar(&cfgPath, "config", "", "Path to config.yaml (defaults to ~/.config/colino/config.yaml)")

    // ingest command
    ingestCmd := &cobra.Command{
        Use:   "ingest",
        Short: "Ingest from configured sources",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := signalContext(cmd.Context())
            defer cancel()

            cfg, err := config.LoadOrCreate(cfgPath)
            if err != nil {
                return err
            }

            database, err := db.Open(cfg)
            if err != nil {
                return err
            }
            defer database.Close()

            progress := ui.NewProgress()
            defer progress.Close()

            return ingest.Run(ctx, cfg, database, progress)
        },
    }
    root.AddCommand(ingestCmd)

    // list command
    var hours int
    listCmd := &cobra.Command{
        Use:   "list",
        Short: "List items from DB",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := signalContext(cmd.Context())
            defer cancel()

            cfg, err := config.LoadOrCreate(cfgPath)
            if err != nil {
                return err
            }
            database, err := db.Open(cfg)
            if err != nil {
                return err
            }
            defer database.Close()

            items, err := db.ListRecent(ctx, database, hours)
            if err != nil {
                return err
            }
            for _, it := range items {
                fmt.Printf("[%s] %s\n  %s\n", it.SourceType, it.Title, it.Link)
            }
            return nil
        },
    }
    listCmd.Flags().IntVar(&hours, "hours", 24, "How many past hours to include")
    root.AddCommand(listCmd)

    // digest command
    var rssOnly bool
    digestCmd := &cobra.Command{
        Use:   "digest [optional: feed URL]",
        Short: "Produce a digest for configured sources or a URL",
        Args:  cobra.ArbitraryArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx, cancel := signalContext(cmd.Context())
            defer cancel()

            cfg, err := config.LoadOrCreate(cfgPath)
            if err != nil {
                return err
            }
            database, err := db.Open(cfg)
            if err != nil {
                return err
            }
            defer database.Close()

            return digest.Run(ctx, cfg, database, args, digest.Options{RSSOnly: rssOnly})
        },
    }
    digestCmd.Flags().BoolVar(&rssOnly, "rss", false, "Use RSS summarization (ignore YouTube)")
    root.AddCommand(digestCmd)

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}

func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
    ctx, cancel := context.WithCancel(parent)
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-ch
        cancel()
    }()
    return ctx, cancel
}
