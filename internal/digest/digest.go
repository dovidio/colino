package digest

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"

	"golino/internal/config"
)

func Run(ctx context.Context, content string) error {

	appConfig, err := config.LoadAppConfig()
	if err != nil {
		return err
	}

	fmt.Printf("Digesting %s with base url : %s\n", content, appConfig.AIConf.BaseUrl)
	client := openai.NewClient(
		option.WithBaseURL(appConfig.AIConf.BaseUrl),
	)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	chatCompletion, err := client.Chat.Completions.New(timeoutCtx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(fmt.Sprintf("Say this is a test and repeat this word %s", content)),
		},
		Model: appConfig.AIConf.Model,
	})
	if err != nil {
		panic(err.Error())
	}
	println(chatCompletion.Choices[0].Message.Content)
	return nil
}
