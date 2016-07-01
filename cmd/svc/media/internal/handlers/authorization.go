package handlers

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mediactx"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"golang.org/x/net/context"
)

type authorizationHandler struct {
	idParamName string
	svc         service.Service
	h           httputil.ContextHandler
}

func authorizationRequired(h httputil.ContextHandler, svc service.Service) httputil.ContextHandler {
	return &authorizationHandler{
		svc: svc,
		h:   h,
	}
}

func (h *authorizationHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if mediactx.RequiresAuthorization(ctx) {
		acc, err := mediactx.Account(ctx)
		if err != nil {
			internalError(w, err)
			return
		}
		mediaID, err := dal.ParseMediaID(mux.Vars(ctx)[idParamName])
		if err != nil {
			badRequest(w, errors.New("Cannot parse media id"), http.StatusBadRequest)
			return
		}
		if err := h.svc.CanAccess(ctx, mediaID, acc.ID); err == service.ErrAccessDenied {
			forbidden(w, err, golog.WARN)
			return
		} else if err != nil {
			forbidden(w, err, golog.ERR)
			return
		}
	}

	h.h.ServeHTTP(ctx, w, r)
}
