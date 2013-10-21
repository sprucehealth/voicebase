// Package apiservice contains the PingHandler
//	Description:
//		PingHandler is an HTTP handler for processing a request to a basic healt-check request
//	Request:
//		GET /v1/ping
//	Response:
//		Content-Type: text/plain
//		Content: pong
//		Status: HTTP/1.1 200 OK
package apiservice

import (
	"net/http"
)

const (
	Pong = "pong"
)

type PingHandler int

func (h PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte(Pong)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h PingHandler) NonAuthenticated() bool {
	return true
}
