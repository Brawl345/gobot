package gemini

// Models: https://ai.google.dev/gemini-api/docs/models/gemini

const (
	ApiUrlGemini     = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
	ApiUrlFileUpload = "https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s"
	RoleModel        = "model"
	RoleUser         = "user"
)

type (
	Content struct {
		Role  string `json:"role"`
		Parts []Part `json:"parts"`
	}

	FileData struct {
		MimeType string `json:"mimeType,omitempty"`
		FileUri  string `json:"fileUri,omitempty"`
	}

	Part struct {
		Text     string    `json:"text,omitempty"`
		FileData *FileData `json:"fileData,omitempty"`
	}

	SafetySetting struct {
		Category  string `json:"category"`  // https://ai.google.dev/api/generate-content#v1beta.HarmCategory
		Threshold string `json:"threshold"` // https://ai.google.dev/api/generate-content#HarmBlockThreshold
	}

	GenerationConfig struct {
		Temperature     float64 `json:"temperature"`
		TopK            int     `json:"topK"`
		TopP            int     `json:"topP"`
		MaxOutputTokens int     `json:"maxOutputTokens"`
	}

	SystemInstruction struct {
		Parts []Part `json:"parts"`
	}

	// GenerateContentRequest - https://ai.google.dev/api/generate-content#request-body
	GenerateContentRequest struct {
		Contents          []Content         `json:"contents"`
		SafetySettings    []SafetySetting   `json:"safetySettings"`
		GenerationConfig  GenerationConfig  `json:"generationConfig"`
		SystemInstruction SystemInstruction `json:"system_instruction"`
	}

	// GenerateContentResponse - https://ai.google.dev/api/generate-content#generatecontentresponse
	GenerateContentResponse struct {
		Candidates []struct {
			Content       Content `json:"content"`
			FinishReason  string  `json:"finishReason"`
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		} `json:"candidates"`
	}

	// FileUploadResponse - https://ai.google.dev/api/files#response-body
	FileUploadResponse struct {
		File struct {
			MimeType string `json:"mimeType"`
			Uri      string `json:"uri"`
		} `json:"file"`
	}
)

func (c *Content) Chars() int {
	chars := 0
	for _, part := range c.Parts {
		chars += len(part.Text)
	}
	return chars
}
