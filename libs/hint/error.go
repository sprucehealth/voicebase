package hint

import (
	"encoding/json"
)

// Error is a structured response indicating a non-2xx HTTP response.
type Error struct {
	HTTPStatusCode int                 `json:"status"`
	Message        string              `json:"message"`
	Errors         map[string][]string `json:"errors"`
}

func (e *Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}
