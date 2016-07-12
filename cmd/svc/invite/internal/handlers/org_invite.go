package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/libs/mux"

	"context"
)

type orgCodeHandler struct {
	dal dal.DAL
}

func (h *orgCodeHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token, ok := mux.Vars(ctx)["token"]
	if !ok {
		http.Error(w, "Invalid invite token", http.StatusBadRequest)
		return
	}
	inv, err := h.dal.InviteForToken(ctx, token)
	if err == dal.ErrNotFound {
		http.NotFound(w, r)
		return
	} else if err != nil {
		internalError(w, err)
		return
	}
	http.Redirect(w, r, inv.URL, http.StatusSeeOther)
}
