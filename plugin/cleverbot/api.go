package cleverbot

type (
	Response struct {
		State  string `json:"cs"`
		Output string `json:"output"`
	}
)
