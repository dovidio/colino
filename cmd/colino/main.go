package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"golino/internal/config"
	"golino/internal/digest"
	"golino/internal/ingest"
	"golino/internal/launchd"
	"golino/internal/list"
	"golino/internal/server"
	"golino/internal/setup"
)

func main() {
	app := &cli.Command{
		Name:  "colino",
		Usage: "Colino",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Run MCP server on stdio",
				Action: func(ctx context.Context, c *cli.Command) error {
					return server.Run(ctx)
				},
			},
			{
				Name:  "list",
				Usage: "List cached content",
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "hours", Usage: "Time window in hours (default: 24)", Value: 24},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return list.Run(ctx, c.Int("hours"))
				},
			},
			{
				Name:  "setup",
				Usage: "Setup Colino's configuration",
				Action: func(ctx context.Context, c *cli.Command) error {
					return setup.Run(ctx)
				},
			},
			{
				Name:  "digest",
				Usage: "Digest an article or a video",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "content",
						UsageText: "url",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return digest.Run(ctx, c.StringArg("content"))
				},
			},
			{
				Name:  "daemon",
				Usage: "Run ingestion daemon",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "once", Usage: "Run single ingest cycle and exit"},
					&cli.IntFlag{Name: "interval-minutes", Usage: "Override interval minutes (default from config: 30)"},
					&cli.StringFlag{Name: "sources", Usage: "Comma-separated sources (article,youtube)", Value: "article"},
					&cli.StringFlag{Name: "log-file", Usage: "Path to daemon log file"},
				},
				Commands: []*cli.Command{
					{
						Name:  "install",
						Usage: "Install launchd agent (macOS)",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "label", Value: "com.colino.daemon", Usage: "launchd label"},
							&cli.IntFlag{Name: "interval-minutes", Value: 30, Usage: "interval minutes"},
							&cli.StringFlag{Name: "sources", Value: "article", Usage: "sources list"},
							&cli.StringFlag{Name: "log-file", Usage: "daemon log file path"},
							&cli.StringFlag{Name: "plist", Usage: "custom plist path (default ~/Library/LaunchAgents/<label>.plist)"},
						},
						Action: func(ctx context.Context, c *cli.Command) error {
							exe, _ := os.Executable()
							if strings.TrimSpace(exe) == "" {
								return fmt.Errorf("cannot discover program path")
							}
							args := []string{"daemon", "--once"}
							if v := c.String("sources"); strings.TrimSpace(v) != "" {
								args = append(args, "--sources", v)
							}
							if v := c.String("log-file"); strings.TrimSpace(v) != "" {
								args = append(args, "--log-file", v)
							}
							opt := launchd.InstallOptions{
								Label:           c.String("label"),
								IntervalMinutes: c.Int("interval-minutes"),
								ProgramPath:     exe,
								ProgramArgs:     args,
								StdOutPath:      c.String("log-file"),
								StdErrPath:      c.String("log-file"),
								PlistPath:       c.String("plist"),
							}
							path, err := launchd.Install(opt)
							if err != nil {
								return err
							}
							fmt.Printf("launchd agent installed and loaded: %s\n", path)
							return nil
						},
					},
					{
						Name:  "uninstall",
						Usage: "Uninstall launchd agent (macOS)",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "label", Value: "com.colino.daemon", Usage: "launchd label"},
							&cli.StringFlag{Name: "plist", Usage: "path to plist (default ~/Library/LaunchAgents/<label>.plist)"},
						},
						Action: func(ctx context.Context, c *cli.Command) error {
							if err := launchd.Uninstall(c.String("label"), c.String("plist")); err != nil {
								return err
							}
							fmt.Println("launchd agent unloaded and removed")
							return nil
						},
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					opts := ingest.Options{
						LogFile: c.String("log-file"),
					}
					return ingest.Run(ctx, opts, config.AppConfigLoader())
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
