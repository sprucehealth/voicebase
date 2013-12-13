package kinesis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ErrBadStatusCode int

func (e ErrBadStatusCode) Error() string {
	return fmt.Sprintf("bad status code %d", int(e))
}

func dumpResponse(res *http.Response) {
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, res.Body); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", buf.String())
}

type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Type       string `json:"__type"`
}

func (er *ErrorResponse) Error() string {
	return fmt.Sprintf("aws/kinesis: %d %s", er.StatusCode, er.Type)
}

func ParseErrorResponse(res *http.Response) error {
	er := &ErrorResponse{
		StatusCode: res.StatusCode,
	}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(er); err != nil {
		return err
	}
	return er
}
