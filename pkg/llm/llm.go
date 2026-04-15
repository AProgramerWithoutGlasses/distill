package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	inner *openai.Client
	model string
}

func NewClient(apiKey, model, baseURL string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	c := openai.NewClient(opts...)
	return &Client{
		inner: &c,
		model: model,
	}
}

func (c *Client) Chat(ctx context.Context, prompt string) (string, error) {
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
