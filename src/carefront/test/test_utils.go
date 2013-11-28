package test

import (
	"net/http"
	"strconv"
)

// FakeResponseWriter for testing purposes
type FakeResponseWriter struct {
	Headers http.Header
	body    []byte
}

// Implementing the ResponseWriter interface
func (f *FakeResponseWriter) Header() http.Header {
	return f.Headers
}

func (f *FakeResponseWriter) Write(response_body []byte) (int, error) {
	// writing status ok since if its gotten this far, it means that its going to
	// be a succesful writing of a response
	f.WriteHeader(http.StatusOK)
	f.body = response_body
	return 0, nil
}

func (f *FakeResponseWriter) WriteHeader(statusCode int) {
	f.Headers.Add("Status", strconv.Itoa(statusCode))
}
