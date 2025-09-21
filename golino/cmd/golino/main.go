package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"colino-mcp/internal/daemon"
	"colino-mcp/internal/launchd"
	"colino-mcp/internal/server"
)

func main() {
	if len(os.Args) == 1 {
		// default to server (stdio MCP)
		if err := server.Run(context.Background()); err != nil {
			log.Fatalf("mcp server: %v", err)
		}
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "server":
		if err := server.Run(context.Background()); err != nil {
			log.Fatalf("mcp server: %v", err)
		}
	case "daemon":
		if len(os.Args) >= 3 && (os.Args[2] == "install" || os.Args[2] == "uninstall") {
			sub := os.Args[2]
			switch sub {
			case "install":
				fs := flag.NewFlagSet("daemon install", flag.ExitOnError)
				label := fs.String("label", "com.colino.daemon", "launchd label")
				interval := fs.Int("interval-minutes", 30, "interval minutes")
				sources := fs.String("sources", "rss,youtube", "sources list")
				logFile := fs.String("log-file", "", "daemon log file path")
				plistOut := fs.String("plist", "", "custom plist path (defaults to ~/Library/LaunchAgents/<label>.plist)")
				_ = fs.Parse(os.Args[3:])

				// discover program path
				exe, _ := os.Executable()
				if exe == "" {
					log.Fatalf("cannot discover program path")
				}
				// build ProgramArguments: binary daemon --once ...
				args := []string{"daemon", "--once"}
				if v := *sources; strings.TrimSpace(v) != "" {
					args = append(args, "--sources", v)
				}
				if v := *logFile; strings.TrimSpace(v) != "" {
					args = append(args, "--log-file", v)
				}
				opt := launchd.InstallOptions{
					Label:           *label,
					IntervalMinutes: *interval,
					ProgramPath:     exe,
					ProgramArgs:     args,
					StdOutPath:      *logFile,
					StdErrPath:      *logFile,
					PlistPath:       *plistOut,
				}
				path, err := launchd.Install(opt)
				if err != nil {
					log.Fatalf("install failed: %v", err)
				}
				fmt.Printf("launchd agent installed and loaded: %s\n", path)
			case "uninstall":
				fs := flag.NewFlagSet("daemon uninstall", flag.ExitOnError)
				label := fs.String("label", "com.colino.daemon", "launchd label")
				plistPath := fs.String("plist", "", "path to plist (defaults to ~/Library/LaunchAgents/<label>.plist)")
				_ = fs.Parse(os.Args[3:])
				if err := launchd.Uninstall(*label, *plistPath); err != nil {
					log.Fatalf("uninstall failed: %v", err)
				}
				fmt.Println("launchd agent unloaded and removed")
			}
		} else {
			fs := flag.NewFlagSet("daemon", flag.ExitOnError)
			once := fs.Bool("once", false, "run a single ingest cycle and exit")
			interval := fs.Int("interval-minutes", 0, "override interval minutes (default from config: 30)")
			sources := fs.String("sources", "", "comma-separated sources (rss,youtube)")
			logFile := fs.String("log-file", "", "path to daemon log file")
			_ = fs.Parse(os.Args[2:])

			opts := daemon.Options{
				Once:        *once,
				IntervalMin: *interval,
				SourcesCSV:  *sources,
				LogFile:     *logFile,
			}
			if err := daemon.Run(context.Background(), opts); err != nil {
				log.Fatalf("daemon: %v", err)
			}
		}
	case "help", "-h", "--help":
		fmt.Println("Usage:")
		fmt.Println("  colino-mcp server                      # run MCP server on stdio")
		fmt.Println("  colino-mcp daemon [flags]              # run ingestion daemon")
		fmt.Println("  colino-mcp daemon install [flags]      # install launchd agent (macOS)")
		fmt.Println("  colino-mcp daemon uninstall [flags]    # uninstall launchd agent (macOS)")
		fmt.Println()
		fmt.Println("Daemon flags:")
		fmt.Println("  --once                 Run a single ingest cycle and exit")
		fmt.Println("  --interval-minutes N   Override interval minutes (default 30)")
		fmt.Println("  --sources LIST         Comma-separated sources (rss,youtube)")
		// no external command: daemon performs ingestion in Go
		fmt.Println("  --log-file PATH        Path to daemon log file")
		fmt.Println()
		fmt.Println("Daemon install flags (macOS launchd):")
		fmt.Println("  --label NAME           Launch agent label (default com.colino.daemon)")
		fmt.Println("  --interval-minutes N   Run every N minutes")
		fmt.Println("  --sources LIST         Sources for each run")
		// ingest command not needed; daemon uses internal ingestion
		fmt.Println("  --log-file PATH        Redirect stdout/err to this file")
		fmt.Println("  --plist PATH           Custom plist path (defaults to ~/Library/LaunchAgents/<label>.plist)")
	default:
		if strings.TrimSpace(cmd) == "" {
			cmd = "(empty)"
		}
		log.Fatalf("unknown command: %s", cmd)
	}
}
