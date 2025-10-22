package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"
	"golino/internal/test"
)

func main() {
	cmd := &cli.Command{
		Name:  "vhs-helper",
		Usage: "VHS demo recording helper for Colino",
		Commands: []*cli.Command{
			{
				Name:  "setup",
				Usage: "Record setup demo (config, daemon, list, digest)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runDemo("setup")
				},
			},
			{
				Name:  "tui",
				Usage: "Record TUI demo (search, navigation, reading)",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runDemo("tui")
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runDemo(demoName string) error {
	fmt.Printf("🎬 Starting %s demo generation...\n", demoName)

	// Build colino binary
	if err := buildColino(); err != nil {
		return fmt.Errorf("failed to build colino: %w", err)
	}

	// Start demo server
	demoSrv := test.NewDemoServer("localhost", 8080)
	demoSrv.Start()

	// Wait for server to be ready with health check
	fmt.Println("⏳ Waiting for server to be ready...")
	if err := waitForServer("http://localhost:8080", 30*time.Second); err != nil {
		return fmt.Errorf("demo server failed to start: %w", err)
	}

	// Ensure server is stopped when we're done
	defer func() {
		if err := demoSrv.Stop(); err != nil {
			log.Printf("Error stopping demo server: %v", err)
		}
	}()

	// For TUI demo, create config file and ingest articles
	if demoName == "tui" {
		fmt.Println("📝 Creating config for TUI demo...")
		if err := createDemoConfig(); err != nil {
			return fmt.Errorf("failed to create demo config: %w", err)
		}

		fmt.Println("📥 Ingesting articles for TUI demo...")
		if err := runCommand("./colino", "daemon"); err != nil {
			return fmt.Errorf("failed to ingest articles: %w", err)
		}
	}

	// Copy colino binary to tapes directory
	if err := copyColinoToTapes(); err != nil {
		return fmt.Errorf("failed to copy colino binary: %w", err)
	}

	// Run VHS recording
	tapePath := fmt.Sprintf("tapes/%s.vhs", demoName)
	if err := runVHS(tapePath); err != nil {
		return fmt.Errorf("failed to run VHS: %w", err)
	}

	// Validate generated files
	if err := validateGeneratedFiles(demoName); err != nil {
		return err
	}

	fmt.Printf("✅ %s demo completed successfully!\n", demoName)
	return nil
}

func waitForServer(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	start := time.Now()

	for time.Since(start) < timeout {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Println("✅ Demo server is ready!")
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("server not ready after %v", timeout)
}

func buildColino() error {
	fmt.Println("🔧 Building colino...")
	return runCommand("go", "build", "-o", "colino", "./cmd/colino")
}

func copyColinoToTapes() error {
	fmt.Println("📁 Copying colino binary to tapes directory...")
	return runCommand("cp", "./colino", "tapes/colino")
}

func runVHS(tapePath string) error {
	fmt.Printf("🎥 Recording VHS demo: %s\n", tapePath)
	return runCommand("vhs", tapePath)
}

func validateGeneratedFiles(demoName string) error {
	fmt.Println("🔍 Validating generated files...")

	gifPath := fmt.Sprintf("tapes/%s.gif", demoName)
	asciiPath := fmt.Sprintf("tapes/%s.ascii", demoName)

	// Check if GIF was generated
	if _, err := os.Stat(gifPath); os.IsNotExist(err) {
		return fmt.Errorf("❌ %s GIF not generated", demoName)
	}

	// Check if ASCII file was generated
	if _, err := os.Stat(asciiPath); os.IsNotExist(err) {
		return fmt.Errorf("❌ %s ASCII file not generated", demoName)
	}

	fmt.Printf("✅ %s files generated successfully:\n", demoName)
	fmt.Printf("   📹 %s\n", gifPath)
	fmt.Printf("   📄 %s\n", asciiPath)

	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createDemoConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "colino")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	configContent := `# Colino configuration
database:
  path: "` + filepath.Join(homeDir, "Library", "Application Support", "Colino", "colino.db") + `"
rss:
  feeds:
    - http://localhost:8080/rss
ai:
  model: "meta-llama/llama-3.3-70b-instruct:free"
  base_url: "https://openrouter.ai/api/v1"
  article_prompt: |
    You are an expert news curator and summarizer.
    Create an insightful summary of the article content below.
    The content can come from news articles, youtube videos transcripts or blog posts.
    Format your response in clean markdown with headers and bullet points if required.

    ## Article {{.Title}}
    **Source:** {{.Source}} | **Published:** {{.Published}}
    **URL:** {{.Url}}

    **Content:**
    {{.Content}}
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ Created config file at: %s\n", configPath)
	return nil
}
