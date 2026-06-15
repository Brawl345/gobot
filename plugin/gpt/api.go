package gpt

const (
	ApiURL = "https://api.openai.com/v1/responses"
	Model  = "gpt-5.5"

	TypeInputText          = "input_text"
	TypeInputImage         = "input_image"
	TypeMessage            = "message"
	TypeOutputText         = "output_text"
	TypeFunctionCall       = "function_call"
	TypeFunctionCallOutput = "function_call_output"

	RoleUser = "user"

	StatusIncomplete = "incomplete"
)

type (
	InputText struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	InputImage struct {
		Type     string `json:"type"`
		ImageURL string `json:"image_url"`
	}

	InputMessage struct {
		Role    string `json:"role"`
		Content []any  `json:"content"`
	}

	FunctionCallOutput struct {
		Type   string `json:"type"`
		CallID string `json:"call_id"`
		Output string `json:"output"`
	}

	Property struct {
		Type        string   `json:"type"`
		Description string   `json:"description"`
		Enum        []string `json:"enum,omitempty"`
	}

	FunctionParameters struct {
		Type                 string              `json:"type"`
		Properties           map[string]Property `json:"properties"`
		Required             []string            `json:"required"`
		AdditionalProperties bool                `json:"additionalProperties"`
	}

	FunctionTool struct {
		Type        string             `json:"type"`
		Name        string             `json:"name"`
		Description string             `json:"description"`
		Parameters  FunctionParameters `json:"parameters"`
		Strict      bool               `json:"strict"`
	}

	Reasoning struct {
		Effort string `json:"effort"`
	}

	Request struct {
		Model              string         `json:"model"`
		Input              []any          `json:"input"`
		Instructions       string         `json:"instructions"`
		Store              bool           `json:"store"`
		MaxOutputTokens    int            `json:"max_output_tokens"`
		PreviousResponseID string         `json:"previous_response_id,omitempty"`
		Tools              []FunctionTool `json:"tools,omitempty"`
		Reasoning          Reasoning      `json:"reasoning"`
	}

	OutputContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	// OutputItem handles both "message" and "function_call" output types.
	OutputItem struct {
		Type string `json:"type"`
		// "message" fields
		Role    string          `json:"role,omitempty"`
		Content []OutputContent `json:"content,omitempty"`
		// "function_call" fields
		CallID    string `json:"call_id,omitempty"`
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	}

	Response struct {
		ID     string       `json:"id"`
		Status string       `json:"status"`
		Output []OutputItem `json:"output"`
	}

	APIErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	Tool interface {
		Definition() FunctionTool
		Execute(arguments string) (string, error)
		Emoji() string
	}
)
