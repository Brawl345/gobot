package httpUtils

import (
	"fmt"
	"net/http"
)

type HttpError struct {
	StatusCode int
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("unexpected HTTP status: %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

func (e *HttpError) StatusText() string {
	return http.StatusText(e.StatusCode)
}
