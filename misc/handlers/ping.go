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
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	pong = "pong"
)

type pingHandler int

func NewPingHandler() http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			pingHandler(0)), []string{"GET"})
}

func (h pingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(pong)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
