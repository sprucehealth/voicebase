package test_handler

import "net/http"

type MockHandler struct {
	H     http.Handler
	Setup func()
}

func (h *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Setup()
	h.H.ServeHTTP(w, r)
}
