package voicebase

import (
	"encoding/json"
)

type ErrorItem struct {
	Error string `json:"error"`
}

// Error is a structured response indicating a non-2xx HTTP response.
type Error struct {
	Errors    ErrorItem `json:"errors"`
	Reference string    `json:"reference"`
	Status    int       `json:"status"`
}

func (e *Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}
