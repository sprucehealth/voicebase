package ec2

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type ErrorResponse struct {
	StatusCode int
	Errors     []Error `xml:"Errors>Error"`
	RequestID  string
}

func (er *ErrorResponse) Error() string {
	if len(er.Errors) == 1 {
		return fmt.Sprintf("[%d] %s: %s", er.StatusCode, er.Errors[0].Code, er.Errors[0].Message)
	}
	errors := make([]string, len(er.Errors))
	for i, e := range er.Errors {
		errors[i] = fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[%d] ", er.StatusCode) + strings.Join(errors, ", ")
}

func ParseErrorResponse(res *http.Response) error {
	er := ErrorResponse{
		StatusCode: res.StatusCode,
	}
	if err := xml.NewDecoder(res.Body).Decode(&er); err != nil {
		return err
	}
	return &er
}
