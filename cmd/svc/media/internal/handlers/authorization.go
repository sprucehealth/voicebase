package handlers

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mediactx"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/service"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mux"
)

type authorizationHandler struct {
	idParamName string
	svc         service.Service
	h           http.Handler
}

func authorizationRequired(h http.Handler, svc service.Service) http.Handler {
	return &authorizationHandler{
		svc: svc,
		h:   h,
	}
}

func (h *authorizationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	h.h.ServeHTTP(w, r)
}
