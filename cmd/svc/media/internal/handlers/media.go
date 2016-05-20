package handlers

import (
	"net/http"

	"golang.org/x/net/context"
)

type mediaHandler struct{}

func (h *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
}
