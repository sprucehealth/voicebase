package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/httputil"
)

// Authorized interface helps ensure that caller of every handler is authorized
// to process the call it is intended for.
type Authorized interface {
	IsAuthorized(r *http.Request) (bool, error)
	http.Handler
}

type isAuthorizedHandler struct {
	h Authorized
}

type DELETEAuthorizer interface {
	IsDELETEAuthorized(r *http.Request) (bool, error)
}

type GETAuthorizer interface {
	IsGETAuthorized(r *http.Request) (bool, error)
}

type PATCHAuthorizer interface {
	IsPATCHAuthorized(r *http.Request) (bool, error)
}

type POSTAuthorizer interface {
	IsPOSTAuthorized(r *http.Request) (bool, error)
}

type PUTAuthorizer interface {
	IsPUTAuthorized(r *http.Request) (bool, error)
}

type methodGranularAuthorizedHandler struct {
	h http.Handler
}

func (h *methodGranularAuthorizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(w, r, h.h, authorization, VerifyAuthCode) {
		return
	}

	var authorized bool
	var err error
	switch r.Method {
	case httputil.Delete:
		auther, ok := h.h.(DELETEAuthorizer)
		if ok {
			authorized, err = auther.IsDELETEAuthorized(r)
		}
	case httputil.Get:
		auther, ok := h.h.(GETAuthorizer)
		if ok {
			authorized, err = auther.IsGETAuthorized(r)
		}
	case httputil.Patch:
		auther, ok := h.h.(PATCHAuthorizer)
		if ok {
			authorized, err = auther.IsPATCHAuthorized(r)
		}
	case httputil.Post:
		auther, ok := h.h.(POSTAuthorizer)
		if ok {
			authorized, err = auther.IsPOSTAuthorized(r)
		}
	case httputil.Put:
		auther, ok := h.h.(PUTAuthorizer)
		if ok {
			authorized, err = auther.IsPUTAuthorized(r)
		}
	default:
		WriteAccessNotAllowedError(w, r)
		return
	}

	if IsBadRequestError(err) {
		WriteBadRequestError(err, w, r)
		return
	} else if err != nil {
		WriteError(err, w, r)
		return
	}

	if !authorized {
		WriteAccessNotAllowedError(w, r)
		return
	}

	h.h.ServeHTTP(w, r)
}

func MethodGranularAuthorizationRequired(h http.Handler) http.Handler {
	return &methodGranularAuthorizedHandler{
		h: h,
	}
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
