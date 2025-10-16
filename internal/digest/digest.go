package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"

	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/ingest"
	"golino/internal/youtube"
)

type Article struct {
	Title     string
	Source    string
	Published string
	Url       string
	Content   string
}

func Run(ctx context.Context, url string) error {
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		return err
	}

	if appConfig.AIConf.BaseUrl == "" {
		return fmt.Errorf("AI base URL is not configured")
	}
	if appConfig.AIConf.Model == "" {
		return fmt.Errorf("AI model is not configured")
	}
	if appConfig.AIConf.ArticlePrompt == "" {
		return fmt.Errorf("AI article prompt is not configured")
	}

	fmt.Printf("Digesting %s with base url : %s\n", url, appConfig.AIConf.BaseUrl)
	content, err := getContentFromCache(ctx, url)
	if err != nil {
		fmt.Printf("Content was not found in cache, scraping content...\n")
		content, err = getFreshContent(ctx, appConfig, url)
		if err != nil {
			fmt.Printf("❌ Could not extract content from the website")
			return err
		}
		fmt.Printf("✅ Extracted content (%d characters)\n", len(content.Content))
	} else {
		fmt.Printf("Content was found in cache, digesting...\n")
	}

	template, err := template.New("template").Parse(appConfig.AIConf.ArticlePrompt)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = template.Execute(&buf, content)
	if err != nil {
		return err
	}
	hydratedPrompt := buf.String()

	client := openai.NewClient(
		option.WithBaseURL(appConfig.AIConf.BaseUrl),
	)
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if appConfig.AIConf.Stream {
		// Streaming response
		chatCompletion := client.Chat.Completions.NewStreaming(timeoutCtx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(hydratedPrompt),
			},
			Model: appConfig.AIConf.Model,
		})

		// Stream the response using the iterator
		for chatCompletion.Next() {
			chunk := chatCompletion.Current()
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				fmt.Print(chunk.Choices[0].Delta.Content)
			}
		}

		if err := chatCompletion.Err(); err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		fmt.Println() // New line after streaming
		return nil
	} else {
		// Non-streaming response
		chatCompletion, err := client.Chat.Completions.New(timeoutCtx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(hydratedPrompt),
			},
			Model: appConfig.AIConf.Model,
		})
		if err != nil {
			return fmt.Errorf("failed to get AI completion: %w", err)
		}

		if len(chatCompletion.Choices) == 0 {
			return fmt.Errorf("AI returned no choices")
		}

		if strings.TrimSpace(chatCompletion.Choices[0].Message.Content) == "" {
			return fmt.Errorf("AI returned empty content")
		}

		fmt.Println(chatCompletion.Choices[0].Message.Content)
		return nil
	}
}

func getContentFromCache(ctx context.Context, url string) (*Article, error) {
	dbPath, err := config.LoadDBPath()
	if err != nil {
		return nil, err
	}
	db, err := colinodb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	content, err := colinodb.GetByURL(ctx, db, url)
	if err != nil {
		return nil, err
	}
	if content == nil || content.ID == "" {
		return nil, fmt.Errorf("No content found in cache")
	}

	metadata := content.Metadata
	if !metadata.Valid {
		return nil, fmt.Errorf("Did not found any metadata")
	}
	var rssMetadata ingest.RSSMetadata
	dec := json.NewDecoder(strings.NewReader(metadata.String))
	err = dec.Decode(&rssMetadata)
	if err != nil {
		return nil, err
	}
	article := Article{
		Source:    content.Source,
		Title:     rssMetadata.EntryTitle,
		Url:       rssMetadata.FeedUrl,
		Published: content.CreatedAt.String(),
		Content:   content.Content,
	}
	return &article, nil
}

func getFreshContent(ctx context.Context, appConfig config.AppConfig, url string) (*Article, error) {
	ri := ingest.NewRSSIngestor(appConfig, 60, 0, log.Default())
	content := ""
	if youtube.IsYouTubeURL(url) {
		content, _, _ = ri.FetchYoutubeTranscript(ctx, url)
		// now we need to extract youtube video id, build a client and extract the webproxy configuration
	} else {
		content, _, _ = ri.FetchArticle(ctx, url)
	}

	if content == "" {
		return nil, fmt.Errorf("could not extract content")
	}
	article := Article{
		Source:    "scraped",
		Title:     "unknown",
		Url:       url,
		Published: "unknown",
		Content:   content,
	}

	return &article, nil
}
