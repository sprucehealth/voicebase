package main

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
