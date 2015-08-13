package apiservice

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/libs/httputil"
)

// Authorized interface helps ensure that caller of every handler is authorized
// to process the call it is intended for.
type Authorized interface {
	IsAuthorized(ctx context.Context, r *http.Request) (bool, error)
	httputil.ContextHandler
}

type isAuthorizedHandler struct {
	h Authorized
}

type DELETEAuthorizer interface {
	IsDELETEAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type GETAuthorizer interface {
	IsGETAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type PATCHAuthorizer interface {
	IsPATCHAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type POSTAuthorizer interface {
	IsPOSTAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type PUTAuthorizer interface {
	IsPUTAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type methodGranularAuthorizedHandler struct {
	h httputil.ContextHandler
}

func (h *methodGranularAuthorizedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(ctx, w, r, h.h, authorization, VerifyAuthCode) {
		return
	}

	var authorized bool
	var err error
	switch r.Method {
	case httputil.Delete:
		auther, ok := h.h.(DELETEAuthorizer)
		if ok {
			authorized, err = auther.IsDELETEAuthorized(ctx, r)
		}
	case httputil.Get:
		auther, ok := h.h.(GETAuthorizer)
		if ok {
			authorized, err = auther.IsGETAuthorized(ctx, r)
		}
	case httputil.Patch:
		auther, ok := h.h.(PATCHAuthorizer)
		if ok {
			authorized, err = auther.IsPATCHAuthorized(ctx, r)
		}
	case httputil.Post:
		auther, ok := h.h.(POSTAuthorizer)
		if ok {
			authorized, err = auther.IsPOSTAuthorized(ctx, r)
		}
	case httputil.Put:
		auther, ok := h.h.(PUTAuthorizer)
		if ok {
			authorized, err = auther.IsPUTAuthorized(ctx, r)
		}
	default:
		WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	if IsBadRequestError(err) {
		WriteBadRequestError(ctx, err, w, r)
		return
	} else if err != nil {
		WriteError(ctx, err, w, r)
		return
	}

	if !authorized {
		WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	h.h.ServeHTTP(ctx, w, r)
}

func MethodGranularAuthorizationRequired(h httputil.ContextHandler) httputil.ContextHandler {
	return &methodGranularAuthorizedHandler{
		h: h,
	}
}

func NoAuthorizationRequired(h httputil.ContextHandler) httputil.ContextHandler {
	return httputil.ContextHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if verifyAuthSetupInTest(ctx, w, r, h, authorization, VerifyAuthCode) {
			return
		}

		h.ServeHTTP(ctx, w, r)
	})
}

func AuthorizationRequired(h Authorized) httputil.ContextHandler {
	return &isAuthorizedHandler{
		h: h,
	}
}

func (i *isAuthorizedHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if verifyAuthSetupInTest(ctx, w, r, i.h, authorization, VerifyAuthCode) {
		return
	}

	// handler has to be authorized
	if authorized, err := i.h.IsAuthorized(ctx, r); err != nil {
		WriteError(ctx, err, w, r)
		return
	} else if !authorized {
		WriteAccessNotAllowedError(ctx, w, r)
		return
	}

	i.h.ServeHTTP(ctx, w, r)
}
