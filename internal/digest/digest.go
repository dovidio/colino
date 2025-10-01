package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"

	"golino/internal/colinodb"
	"golino/internal/config"
	"golino/internal/ingest"
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

	fmt.Printf("Digesting %s with base url : %s\n", url, appConfig.AIConf.BaseUrl)
	article, err := getArticleFromCache(ctx, url)
	if err != nil {
		fmt.Printf("Article was not found in cache, scraping content...\n")
		article, err = getFreshArticle(ctx, url)
		if err != nil {
			return err
		}
		fmt.Printf("Article content was fetched, digesting...")
	} else {
		fmt.Printf("Article was found in cache %v %v, digesting...\n", article, err)
	}

	template, err := template.New("template").Parse(appConfig.AIConf.ArticlePrompt)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = template.Execute(&buf, article)
	if err != nil {
		return err
	}
	hydratedPrompt := buf.String()

	client := openai.NewClient(
		option.WithBaseURL(appConfig.AIConf.BaseUrl),
	)
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	chatCompletion, err := client.Chat.Completions.New(timeoutCtx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(hydratedPrompt),
		},
		Model: appConfig.AIConf.Model,
	})
	if err != nil {
		panic(err.Error())
	}
	println(chatCompletion.Choices[0].Message.Content)
	return nil
}

func getArticleFromCache(ctx context.Context, url string) (*Article, error) {
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

func getFreshArticle(ctx context.Context, url string) (*Article, error) {
	client := &http.Client{Timeout: time.Duration(10) * time.Second}
	content := ingest.ExtractMainText(ctx, url, client)
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
