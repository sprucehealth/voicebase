// Package apiservice contains the PingHandler
//	Description:
//		PingHandler is an HTTP handler for processing a request to a basic healt-check request
//	Request:
//		GET /v1/ping
//	Response:
//		Content-Type: text/plain
//		Content: pong
//		Status: HTTP/1.1 200 OK
package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/apiservice"
)

const (
	pong = "pong"
)

type pingHandler int

func NewPingHandler() http.Handler {
	return pingHandler(0)
}

func (h pingHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (h pingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(pong)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h pingHandler) NonAuthenticated() bool {
	return true
}
