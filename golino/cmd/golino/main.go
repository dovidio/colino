package main

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/urfave/cli/v2"

    "golino/internal/daemon"
    "golino/internal/launchd"
    "golino/internal/server"
)

func main() {
    app := &cli.App{
        Name:  "golino",
        Usage: "Colino MCP server and ingestion daemon",
        // Default action: run server when no subcommand is provided
        Action: func(c *cli.Context) error {
            return server.Run(c.Context)
        },
        Commands: []*cli.Command{
            {
                Name:  "server",
                Usage: "Run MCP server on stdio",
                Action: func(c *cli.Context) error {
                    return server.Run(c.Context)
                },
            },
            {
                Name:  "daemon",
                Usage: "Run ingestion daemon",
                Flags: []cli.Flag{
                    &cli.BoolFlag{Name: "once", Usage: "Run single ingest cycle and exit"},
                    &cli.IntFlag{Name: "interval-minutes", Usage: "Override interval minutes (default from config: 30)"},
                    &cli.StringFlag{Name: "sources", Usage: "Comma-separated sources (rss)", Value: "rss"},
                    &cli.StringFlag{Name: "log-file", Usage: "Path to daemon log file"},
                },
                Subcommands: []*cli.Command{
                    {
                        Name:  "install",
                        Usage: "Install launchd agent (macOS)",
                        Flags: []cli.Flag{
                            &cli.StringFlag{Name: "label", Value: "com.colino.daemon", Usage: "launchd label"},
                            &cli.IntFlag{Name: "interval-minutes", Value: 30, Usage: "interval minutes"},
                            &cli.StringFlag{Name: "sources", Value: "rss", Usage: "sources list"},
                            &cli.StringFlag{Name: "log-file", Usage: "daemon log file path"},
                            &cli.StringFlag{Name: "plist", Usage: "custom plist path (default ~/Library/LaunchAgents/<label>.plist)"},
                        },
                        Action: func(c *cli.Context) error {
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
                        Action: func(c *cli.Context) error {
                            if err := launchd.Uninstall(c.String("label"), c.String("plist")); err != nil {
                                return err
                            }
                            fmt.Println("launchd agent unloaded and removed")
                            return nil
                        },
                    },
                },
                Action: func(c *cli.Context) error {
                    opts := daemon.Options{
                        Once:        c.Bool("once"),
                        IntervalMin: c.Int("interval-minutes"),
                        SourcesCSV:  c.String("sources"),
                        LogFile:     c.String("log-file"),
                    }
                    return daemon.Run(c.Context, opts)
                },
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
