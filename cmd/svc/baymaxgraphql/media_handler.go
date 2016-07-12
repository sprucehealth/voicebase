package main

import (
	"context"
	"net/http"
	"path"

	"github.com/sprucehealth/backend/libs/httputil"
)

type mediaHandler struct {
	mediaAPIDomain string
}

// NewMediaHandler returns an initialized instance of mediaHandler
func NewMediaHandler(mediaAPIDomain string) httputil.ContextHandler {
	return &mediaHandler{
		mediaAPIDomain: mediaAPIDomain,
	}
}

func (m *mediaHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Utilize code 307 to preserve the data and method
	http.Redirect(w, r, m.constructRedirect(r), http.StatusTemporaryRedirect)
}

func (m *mediaHandler) constructRedirect(r *http.Request) string {
	id := r.FormValue("id")
	p := path.Join("media", id)
	if !isOriginal(r) {
		p = path.Join(p, "thumbnail")
	}
	rURL := m.mediaAPIDomain + "/" + p
	if r.URL.RawQuery != "" {
		rURL = rURL + "?" + r.URL.RawQuery
	}
	return rURL
}

func isOriginal(r *http.Request) bool {
	width, err := httputil.ParseFormInt(r, "width")
	if err != nil {
		return false
	}
	height, err := httputil.ParseFormInt(r, "height")
	if err != nil {
		return false
	}
	return width == 0 && height == 0
}
