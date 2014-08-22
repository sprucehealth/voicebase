package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

const (
	maxThumbnailMemory = 1024 * 1024
)

type doctorThumbnailAPIHandler struct {
	dataAPI        api.DataAPI
	thumbnailStore storage.Store
}

func NewDoctorThumbnailAPIHandler(dataAPI api.DataAPI, thumbnailStore storage.Store) http.Handler {
	return httputil.SupportedMethods(&doctorThumbnailAPIHandler{
		dataAPI:        dataAPI,
		thumbnailStore: thumbnailStore,
	}, []string{"GET", "POST"})
}

func (h *doctorThumbnailAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	thumbSize := mux.Vars(r)["size"]
	if thumbSize != "small" && thumbSize != "large" {
		www.APINotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromId(doctorID)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == "POST" {
		audit.LogAction(account.ID, "AdminAPI", "UpdateDoctorThumbnail", map[string]interface{}{"doctor_id": doctorID, "size": thumbSize})

		if err := r.ParseMultipartForm(maxThumbnailMemory); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		file, head, err := r.FormFile("thumbnail")
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		defer file.Close()

		size, err := common.SeekerSize(file)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		headers := http.Header{
			"Content-Type":             []string{head.Header.Get("Content-Type")},
			"X-Amz-Meta-Original-Name": []string{head.Filename},
		}
		storeID, err := h.thumbnailStore.PutReader(fmt.Sprintf("doctor_%d_%s", doctorID, thumbSize), file, size, headers)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		update := &api.DoctorUpdate{}
		switch thumbSize {
		case "small":
			update.SmallThumbnailID = &storeID
		case "large":
			update.LargeThumbnailID = &storeID
		}
		if err := h.dataAPI.UpdateDoctor(doctorID, update); err != nil {
			www.APIInternalError(w, r, err)
		}

		www.JSONResponse(w, r, http.StatusOK, nil)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetDoctorThumbnail", map[string]interface{}{"doctor_id": doctorID, "size": thumbSize})

	var storeID string
	switch thumbSize {
	case "small":
		storeID = doctor.SmallThumbnailID
	case "large":
		storeID = doctor.LargeThumbnailID
	}
	if storeID == "" {
		www.APINotFound(w, r)
		return
	}
	url, err := h.thumbnailStore.GetSignedURL(storeID, time.Now().Add(time.Hour*24))
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}
