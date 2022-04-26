package dcrypt

import "regexp"

var textRegex = regexp.MustCompile("(?s)<textarea>(.+)</textarea>")

type Response struct {
	FormErrors struct {
		Dlcfile []string `json:"dlcfile"`
	} `json:"form_errors"`
	Success struct {
		Message string   `json:"message"`
		Links   []string `json:"links"`
	} `json:"success"`
}
