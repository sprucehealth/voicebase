package voicebase

import "fmt"

type ErrorItem struct {
	Error string `json:"error"`
}

type Message struct {
	Code    int    `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Error is a structured response indicating a non-2xx HTTP response.
type Error struct {
	Warnings  []Message `json:"warnings"`
	Errors    []ErrorItem `json:"errors"`
	Reference string    `json:"reference"`
	Status    int       `json:"status"`
}

func (e *Error) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("voicebase: status=%d %s: %s", e.Status, e.Reference, e.Errors[0].Error)
	}
	return ""
}
