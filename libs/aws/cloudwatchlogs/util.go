package cloudwatchlogs

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Type       string `json:"__type"`
	Message    string `json:"message"`
}

func (er *ErrorResponse) Error() string {
	return fmt.Sprintf("aws.cloudwatchlogs: %d %s %s", er.StatusCode, er.Type, er.Message)
}

func (er *ErrorResponse) String() string {
	return fmt.Sprintf("%d %s %s", er.StatusCode, er.Type, er.Message)
}

func parseErrorResponse(res *http.Response) error {
	er := &ErrorResponse{
		StatusCode: res.StatusCode,
	}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(er); err != nil {
		return err
	}
	return er
}

func intPtrIfNonZero(i int) *int {
	if i > 0 {
		return &i
	}
	return nil
}
