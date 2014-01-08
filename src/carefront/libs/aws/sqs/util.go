package sqs

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	StatusCode int
	Type       string `xml:"Error>Type"`
	Message    string `xml:"Error>Message"`
	RequestID  string
}

func (er *ErrorResponse) Error() string {
	return fmt.Sprintf("[%d] %s: %s", er.StatusCode, er.Type, er.Message)
}

func ParseErrorResponse(res *http.Response) error {
	dec := xml.NewDecoder(res.Body)
	er := ErrorResponse{
		StatusCode: res.StatusCode,
	}
	if err := dec.Decode(&er); err != nil {
		return err
	}
	return &er
}
