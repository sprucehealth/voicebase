package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/www"
)

type providerProfileImageAPIHandler struct {
	dataAPI    api.DataAPI
	imageStore storage.Store
}

func NewProviderProfileImageAPIHandler(dataAPI api.DataAPI, imageStore storage.Store) http.Handler {
	return httputil.SupportedMethods(&providerProfileImageAPIHandler{
		dataAPI:    dataAPI,
		imageStore: imageStore,
	}, httputil.Get, httputil.Put)
}

func (h *providerProfileImageAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	var imageSuffix string
	profileImageType := mux.Vars(r)["type"]
	switch profileImageType {
	case "thumbnail":
		// Note: for legacy reasons (when we used to have small and large thumbnails), continuing to upload
		// the thumbnail image with the large suffix
		imageSuffix = "large"
	case "hero":
		imageSuffix = "hero"
	default:
		www.APINotFound(w, r)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromID(doctorID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)

	if r.Method == httputil.Put {
		audit.LogAction(account.ID, "AdminAPI", "UpdateProviderThumbnail", map[string]interface{}{"doctor_id": doctorID, "type": profileImageType})

		if err := r.ParseMultipartForm(maxMemory); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		file, head, err := r.FormFile("profile_image")
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

		meta := map[string]string{
			"X-Amz-Meta-Original-Name": head.Filename,
		}
		storeID, err := h.imageStore.PutReader(fmt.Sprintf("doctor_%d_%s", doctorID, imageSuffix), file, size, "", meta)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		update := &api.DoctorUpdate{}
		switch profileImageType {
		case "thumbnail":
			update.LargeThumbnailID = &storeID
		case "hero":
			update.HeroImageID = &storeID
		}
		if err := h.dataAPI.UpdateDoctor(doctorID, update); err != nil {
			www.APIInternalError(w, r, err)
		}

		httputil.JSONResponse(w, http.StatusOK, nil)
		return
	}

	audit.LogAction(account.ID, "AdminAPI", "GetProviderThumbnail", map[string]interface{}{"doctor_id": doctorID, "type": profileImageType})

	var storeID string
	switch profileImageType {
	case "thumbnail":
		storeID = doctor.LargeThumbnailID
	case "hero":
		storeID = doctor.HeroImageID
	}
	if storeID == "" {
		www.APINotFound(w, r)
		return
	}
	url, err := h.imageStore.SignedURL(storeID, time.Hour)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}
