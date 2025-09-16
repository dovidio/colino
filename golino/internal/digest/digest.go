package digest

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "sort"
    "strings"
    "time"

    openai "github.com/sashabaranov/go-openai"

    "golino/internal/config"
    "golino/internal/db"
    "golino/internal/models"
)

type Options struct {
    RSSOnly bool
}

// Run prints a digest. If OPENAI_API_KEY is set, generate a summary via OpenAI.
func Run(ctx context.Context, cfg config.Config, database *sql.DB, args []string, opts Options) error {
    items, err := db.ListRecent(ctx, database, 24)
    if err != nil {
        return err
    }
    if opts.RSSOnly {
        filtered := items[:0]
        for _, it := range items {
            if it.SourceType == "rss" {
                filtered = append(filtered, it)
            }
        }
        items = filtered
    }
    sort.Slice(items, func(i, j int) bool { return items[i].Published.After(items[j].Published) })

    if key, ok := os.LookupEnv("OPENAI_API_KEY"); ok && key != "" {
        out, err := summarizeWithOpenAI(ctx, key, items, cfg.OpenAI)
        if err == nil {
            _, _ = os.Stdout.WriteString(out)
            return nil
        }
        // fall back to plain digest if summarization fails
    }

    // Plain digest fallback
    groups := map[string][]string{}
    for _, it := range items {
        ts := it.Published.Local().Format(time.RFC822)
        line := fmt.Sprintf("- %s (%s)\n  %s", it.Title, ts, it.Link)
        groups[it.SourceType] = append(groups[it.SourceType], line)
    }

    b := &strings.Builder{}
    fmt.Fprintln(b, "# Daily Digest")
    for k, lines := range groups {
        fmt.Fprintf(b, "\n## %s\n", strings.ToUpper(k))
        for _, l := range lines {
            fmt.Fprintln(b, l)
        }
    }
    _, _ = os.Stdout.WriteString(b.String())
    return nil
}

func summarizeWithOpenAI(ctx context.Context, apiKey string, items []models.Item, aiCfg config.OpenAI) (string, error) {
    if len(items) == 0 {
        return "No items to summarize.", nil
    }
    // Limit items and content to keep prompt size reasonable
    const maxItems = 40
    const maxText = 400
    if len(items) > maxItems {
        items = items[:maxItems]
    }

    // Build items text
    itemsText := &strings.Builder{}
    for i, it := range items {
        ts := it.Published.Local().Format(time.RFC822)
        text := it.Summary
        if text == "" {
            text = it.Content
        }
        if len(text) > maxText {
            text = text[:maxText] + "…"
        }
        fmt.Fprintf(itemsText, "%d. [%s] %s (%s)\n%s\n\n", i+1, strings.ToUpper(it.SourceType), it.Title, ts, strings.TrimSpace(text))
        fmt.Fprintf(itemsText, "Link: %s\n\n", it.Link)
    }

    // Apply user-configured prompts with fallback defaults
    system := aiCfg.SystemPrompt
    if strings.TrimSpace(system) == "" {
        system = "You write crisp, factual tech/news digests."
    }
    user := aiCfg.UserPrompt
    if strings.TrimSpace(user) == "" {
        user = "Summarize the following items into 3-6 short sections by theme.\nUse markdown with headers and bullet points; include links inline.\nFocus on substance; avoid fluff; keep it under ~250 words.\n\nItems:\n{{items}}"
    }
    if strings.Contains(user, "{{items}}") {
        user = strings.ReplaceAll(user, "{{items}}", itemsText.String())
    } else {
        // If the template doesn't include placeholder, append items for safety
        user = user + "\n\nItems:\n" + itemsText.String()
    }

    client := openai.NewClient(apiKey)
    req := openai.ChatCompletionRequest{
        Model:       firstNonEmpty(aiCfg.Model, "gpt-4o-mini"),
        Temperature: aiCfg.Temperature,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleSystem, Content: system},
            {Role: openai.ChatMessageRoleUser, Content: user},
        },
    }
    resp, err := client.CreateChatCompletion(ctx, req)
    if err != nil {
        return "", err
    }
    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no choices returned")
    }
    return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func firstNonEmpty(values ...string) string {
    for _, v := range values {
        if strings.TrimSpace(v) != "" {
            return v
        }
    }
    return ""
}
