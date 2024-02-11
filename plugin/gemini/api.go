package gemini

const (
	ApiUrlGemini       = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
	ApiUrlGeminiVision = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro-vision:generateContent"
	RoleModel          = "model"
	RoleUser           = "user"
)

type (
	Content struct {
		Role  string `json:"role"`
		Parts []Part `json:"parts"`
	}

	InlineData struct {
		MimeType string `json:"mimeType,omitempty"`
		Data     string `json:"data,omitempty"`
	}

	Part struct {
		Text       string      `json:"text,omitempty"`
		InlineData *InlineData `json:"inlineData,omitempty"`
	}

	SafetySetting struct {
		Category  string `json:"category"`
		Threshold string `json:"threshold"`
	}

	GenerationConfig struct {
		Temperature     float64 `json:"temperature"`
		TopK            int     `json:"topK"`
		TopP            int     `json:"topP"`
		MaxOutputTokens int     `json:"maxOutputTokens"`
	}

	// Request - https://ai.google.dev/api/rest/v1beta/models/generateContent#request-body
	Request struct {
		Contents         []Content        `json:"contents"`
		SafetySettings   []SafetySetting  `json:"safetySettings"`
		GenerationConfig GenerationConfig `json:"generationConfig"`
	}

	// Response - https://ai.google.dev/api/rest/v1beta/GenerateContentResponse
	Response struct {
		Candidates []struct {
			Content       Content `json:"content"`
			FinishReason  string  `json:"finishReason"`
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		} `json:"candidates"`
	}
)

func (c *Content) Chars() int {
	chars := 0
	for _, part := range c.Parts {
		chars += len(part.Text)
	}
	return chars
}
