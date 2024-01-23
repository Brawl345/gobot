package speech_to_text

type (
	ApiResponse struct {
		Text string `json:"text"`
	}

	ApiErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
)
