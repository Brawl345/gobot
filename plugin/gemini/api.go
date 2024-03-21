package gemini

import (
	"github.com/Brawl345/gobot/model"
)

const (
	ApiUrlGemini       = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"
	ApiUrlGeminiVision = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro-vision:generateContent"
	RoleModel          = "model"
	RoleUser           = "user"
)

type (
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
		Contents         []model.GeminiContent `json:"contents"`
		SafetySettings   []SafetySetting       `json:"safetySettings"`
		GenerationConfig GenerationConfig      `json:"generationConfig"`
	}

	// Response - https://ai.google.dev/api/rest/v1beta/GenerateContentResponse
	Response struct {
		Candidates []struct {
			Content       model.GeminiContent `json:"content"`
			FinishReason  string              `json:"finishReason"`
			SafetyRatings []struct {
				Category    string `json:"category"`
				Probability string `json:"probability"`
			} `json:"safetyRatings"`
		} `json:"candidates"`
	}
)
