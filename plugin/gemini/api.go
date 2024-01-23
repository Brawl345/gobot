package gemini

const (
	ApiUrl    = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
	RoleModel = "model"
	RoleUser  = "user"
)

type (
	Content struct {
		Role  string `json:"role"`
		Parts []Part `json:"parts"`
	}

	Part struct {
		Text string `json:"text"`
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

	Request struct {
		Contents         []Content        `json:"contents"`
		SafetySettings   []SafetySetting  `json:"safetySettings"`
		GenerationConfig GenerationConfig `json:"generationConfig"`
	}

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
