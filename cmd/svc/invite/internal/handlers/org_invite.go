package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/libs/mux"
)

type orgCodeHandler struct {
	dal dal.DAL
}

func (h *orgCodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
