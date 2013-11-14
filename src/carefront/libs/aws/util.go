package aws

import (
	"bytes"
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
