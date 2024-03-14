package summarize

// https://docs.anthropic.com/claude/reference/messages_post

const (
	AnthropicApiUrl  = "https://api.anthropic.com/v1/messages"
	AnthropicVersion = "2023-06-01"
	Model            = "claude-3-haiku-20240307"
	User             = "user"
)

type (
	Request struct {
		Model       string       `json:"model"`
		System      string       `json:"system"`
		Messages    []ApiMessage `json:"messages"`
		MaxTokens   int          `json:"max_tokens"`
		Temperature float32      `json:"temperature"`
	}

	ApiMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	Response struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
)
