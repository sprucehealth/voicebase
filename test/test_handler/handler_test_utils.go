package test_handler

import "net/http"

type MockHandler struct {
	H     http.Handler
	Setup func()
}

func (h *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Setup != nil {
		h.Setup()
	}
	h.H.ServeHTTP(w, r)
}
