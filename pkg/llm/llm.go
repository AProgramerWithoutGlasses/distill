package llm

import (
	"context"
	"goweb_staging/pkg/settings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var LLMclient LLMClient

type LLMClient struct {
	inner *openai.Client
	model string
}

func NewLLMClient(app *settings.AppConfig) {
	opts := []option.RequestOption{option.WithAPIKey(app.LLMConfig.APIKey)}
	if app.LLMConfig.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(app.LLMConfig.BaseURL))
	}
	c := openai.NewClient(opts...)

	LLMclient = LLMClient{
		inner: &c,
		model: app.LLMConfig.Model,
	}
}

func (c *LLMClient) Chat(ctx context.Context, prompt string) (string, error) {
	completion, err := c.inner.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: c.model,
	})
	if err != nil {
		return "", err
	}
	return completion.Choices[0].Message.Content, nil
}
