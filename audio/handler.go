package audio

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
)

type Handler struct {
	dataAPI api.DataAPI
	store   storage.Store
}

func NewHandler(dataAPI api.DataAPI, store storage.Store) *Handler {
	return &Handler{
		dataAPI: dataAPI,
		store:   store,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		h.get(w, r)
	case apiservice.HTTP_POST:
		h.upload(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
