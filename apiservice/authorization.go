package apiservice

import "net/http"

// Authorized interface helps ensure that caller of every handler is authorized
// to process the call it is intended for.
type Authorized interface {
	IsAuthorized(r *http.Request) (bool, error)
	http.Handler
}

type isAuthorizedHandler struct {
	h Authorized
}

func NoAuthorizationRequired(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if verifyAuthSetupInTest(w, r, h, authorization, VerifyAuthCode) {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func AuthorizationRequired(h Authorized) http.Handler {
	return &isAuthorizedHandler{
		h: h,
	}
}

func (i *isAuthorizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(w, r, i.h, authorization, VerifyAuthCode) {
		return
	}

	// handler has to be authorized
	if authorized, err := i.h.IsAuthorized(r); err != nil {
		WriteError(err, w, r)
		return
	} else if !authorized {
		WriteAccessNotAllowedError(w, r)
		return
	}

	i.h.ServeHTTP(w, r)
}
