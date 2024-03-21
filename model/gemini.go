package model

import "database/sql"

type (
	GeminiData struct {
		History   sql.NullString `db:"gemini_history"`
		ExpiresOn sql.NullTime   `db:"gemini_history_expires_on"`
	}

	GeminiContent struct {
		Role  string       `json:"role"`
		Parts []GeminiPart `json:"parts"`
	}

	GeminiPart struct {
		Text       string            `json:"text,omitempty"`
		InlineData *GeminiInlineData `json:"inlineData,omitempty"`
	}

	GeminiInlineData struct {
		MimeType string `json:"mimeType,omitempty"`
		Data     string `json:"data,omitempty"`
	}
)

func (c *GeminiContent) Chars() int {
	chars := 0
	for _, part := range c.Parts {
		chars += len(part.Text)
	}
	return chars
}
