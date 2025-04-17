package gemini

import (
	"fmt"
	"strings"
)

// Models: https://ai.google.dev/gemini-api/docs/models/gemini

const (
	ApiUrlGemini     = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
	ApiUrlFileUpload = "https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s"
	RoleModel        = "model"
	RoleUser         = "user"
	MaxSourceLinks   = 5
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
		Temperature     float64         `json:"temperature"`
		TopK            int             `json:"topK"`
		TopP            int             `json:"topP"`
		MaxOutputTokens int             `json:"maxOutputTokens"`
		ThinkingConfig  *ThinkingConfig `json:"thinkingConfig,omitempty"`
	}

	// ThinkingConfig - https://ai.google.dev/api/generate-content#ThinkingConfig
	ThinkingConfig struct {
		IncludeThoughts bool `json:"includeThoughts,omitempty"`
		ThinkingBudget  int  `json:"thinkingBudget,omitempty"`
	}

	SystemInstruction struct {
		Parts []Part `json:"parts"`
	}

	// Tool - https://ai.google.dev/api/caching#Tool
	Tool struct {
		// GoogleSearch - https://ai.google.dev/api/caching#GoogleSearch (has no fields)
		GoogleSearch struct {
		} `json:"google_search"`
	}

	// GenerateContentRequest - https://ai.google.dev/api/generate-content#request-body
	GenerateContentRequest struct {
		Contents          []Content         `json:"contents"`
		SafetySettings    []SafetySetting   `json:"safetySettings"`
		GenerationConfig  GenerationConfig  `json:"generationConfig"`
		SystemInstruction SystemInstruction `json:"system_instruction"`
		Tools             []Tool            `json:"tools"`
	}

	// GroundingMetadata - https://ai.google.dev/api/generate-content#GroundingMetadata
	GroundingMetadata struct {
		// https://ai.google.dev/api/generate-content#GroundingChunk
		GroundingChunks []struct {
			// https://ai.google.dev/api/generate-content#Web
			Web struct {
				Uri   string `json:"uri"`
				Title string `json:"title"`
			} `json:"web"`
		} `json:"groundingChunks"`
		WebSearchQueries []string `json:"webSearchQueries"`
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
			GroundingMetadata GroundingMetadata `json:"groundingMetadata"`
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

func (g *GroundingMetadata) Links() string {
	if len(g.GroundingChunks) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("(")
	for i, chunk := range g.GroundingChunks {
		sb.WriteString(
			fmt.Sprintf(
				"<a href=\"%s\">%s</a>",
				chunk.Web.Uri,
				chunk.Web.Title,
			),
		)
		if i < len(g.GroundingChunks)-1 {
			sb.WriteString(", ")
		}
		if i == MaxSourceLinks-1 && len(g.GroundingChunks) > MaxSourceLinks {
			sb.WriteString("...")
			break
		}
	}
	sb.WriteString(")")

	return sb.String()
}

func (c *Content) Text() string {
	var sb strings.Builder
	for _, part := range c.Parts {
		sb.WriteString(part.Text)
	}
	return sb.String()
}
