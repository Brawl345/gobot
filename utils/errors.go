package utils

import "fmt"

type HttpError struct {
	StatusCode int
	Status     string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("unexpected HTTP status: %s", e.Status)
}
