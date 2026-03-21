package openai

import (
	openailib "github.com/sashabaranov/go-openai"
)

// Client wraps the sashabaranov/go-openai client and exposes only what this
// application requires. Additional capabilities (streaming, TTS, STT) are
// accessed directly through the embedded client field.
type Client struct {
	client *openailib.Client
}

// NewClient creates a new OpenAI Client using the provided API key.
func NewClient(apiKey string) *Client {
	return &Client{
		client: openailib.NewClient(apiKey),
	}
}

// Underlying returns the raw go-openai client for callers that need direct
// access (e.g., streaming or audio endpoints).
func (c *Client) Underlying() *openailib.Client {
	return c.client
}
