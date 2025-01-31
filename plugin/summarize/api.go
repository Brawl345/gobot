package summarize

const (
	User   = "user"
	System = "system"
)

type (
	Request struct {
		Model           string       `json:"model"`
		Messages        []ApiMessage `json:"messages"`
		PresencePenalty float32      `json:"presence_penalty"`
		MaxTokens       int          `json:"max_tokens"`
		Temperature     float32      `json:"temperature"`
	}

	ApiMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	Response struct {
		Choices []struct {
			Message ApiMessage `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
)
