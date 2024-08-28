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
	errStr := fmt.Sprintf("voicebase: status=%d %s: ", e.Status, e.Reference)
	for i, err := range e.Errors {
		if i > 0 {
			errStr += ", "
		}
		errStr += err.Error
	}
	for _, warn := range e.Warnings {
		errStr += fmt.Sprintf(", warning (code %d @ %s): %s", warn.Code, warn.Path, warn.Message)
	}
	return errStr
}
