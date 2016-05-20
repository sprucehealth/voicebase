package handlers

import (
	"net/http"

	"golang.org/x/net/context"
)

type thumbnailHandler struct{}

func (h *thumbnailHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
}
