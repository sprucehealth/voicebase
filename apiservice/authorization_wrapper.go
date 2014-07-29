package apiservice

import "net/http"

type authorizedHandler struct {
	handler http.Handler
}

func (a *authorizedHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (a *authorizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func AuthorizeHandler(h http.Handler) http.Handler {
	return &authorizedHandler{
		handler: h,
	}
}
