package home

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/www"
)

const (
	photoTypeSelfie       = "selfie"
	photoTypeGovernmentID = "governmentid"
)

const maxConsentImageRequestMemory = 5 * 1024 * 1024

type parentalConsentImageAPIHandler struct {
	dataAPI    api.DataAPI
	dispatcher dispatch.Publisher
	mediaStore *media.Store
}

type parentalConsentImageAPIGETResponse struct {
	Types map[string]*imageTypeResponse `json:"types"`
}

type imageTypeResponse struct {
	URL string `json:"url"`
}

func newParentalConsentImageAPIHAndler(dataAPI api.DataAPI, dispatcher dispatch.Publisher, mediaStore *media.Store) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(
		www.APIRoleRequiredHandler(&parentalConsentImageAPIHandler{
			dataAPI:    dataAPI,
			dispatcher: dispatcher,
			mediaStore: mediaStore,
		}, api.RolePatient), httputil.Post, httputil.Get)
}

func (h *parentalConsentImageAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)
	parentPatientID, err := h.dataAPI.GetPatientIDFromAccountID(account.ID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	switch r.Method {
	case httputil.Post:
		h.post(ctx, w, r, account, parentPatientID)
	case httputil.Get:
		h.get(ctx, w, r, parentPatientID)
	}
}

func (h *parentalConsentImageAPIHandler) post(ctx context.Context, w http.ResponseWriter, r *http.Request, account *common.Account, parentPatientID int64) {
	if err := r.ParseMultipartForm(maxConsentImageRequestMemory); err != nil {
		www.APIBadRequestError(w, r, "Failed to parse request")
		return
	}
	imageType := r.FormValue("type")

	// Validate the image type
	switch imageType {
	case photoTypeSelfie, photoTypeGovernmentID:
	default:
		www.APIBadRequestError(w, r, "Invalid image type")
		return
	}

	file, fileHandler, err := r.FormFile("file")
	if err != nil {
		if err == http.ErrMissingFile {
			www.APIBadRequestError(w, r, "File is required")
			return
		}
		www.APIInternalError(w, r, err)
		return
	}
	defer file.Close()

	mimeType := fileHandler.Header.Get("Content-Type")

	size, err := common.SeekerSize(file)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	mediaURL, err := h.mediaStore.PutReader(fmt.Sprintf("parental-consent-proof-%d-%s", parentPatientID, imageType), file, size, mimeType, nil)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	mediaID, err := h.dataAPI.AddMedia(account.ID, mediaURL, mimeType)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	proof := &api.ParentalConsentProof{}
	switch imageType {
	case photoTypeSelfie:
		proof.SelfiePhotoID = &mediaID
	case photoTypeGovernmentID:
		proof.GovernmentIDPhotoID = &mediaID
	}
	if _, err := h.dataAPI.UpsertParentConsentProof(parentPatientID, proof); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// Check if all conditions of consent have been met (consent given and all proof uploaded).
	proof, err = h.dataAPI.ParentConsentProof(parentPatientID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	if proof.IsComplete() {
		consent, err := h.dataAPI.AllParentalConsent(parentPatientID)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		for childPatientID, consent := range consent {
			if consent.Consented {
				if err := patient.ParentalConsentCompleted(h.dataAPI, h.dispatcher, parentPatientID, childPatientID); err != nil {
					www.APIInternalError(w, r, err)
					return
				}
			}
		}
	}

	res, err := h.newImageTypeResponse(mediaID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (h *parentalConsentImageAPIHandler) get(ctx context.Context, w http.ResponseWriter, r *http.Request, parentPatientID int64) {
	proof, err := h.dataAPI.ParentConsentProof(parentPatientID)
	if err != nil && !api.IsErrNotFound(err) {
		www.APIInternalError(w, r, err)
		return
	}
	res := &parentalConsentImageAPIGETResponse{
		Types: make(map[string]*imageTypeResponse),
	}
	if proof != nil {
		if pid := proof.SelfiePhotoID; pid != nil {
			res.Types[photoTypeSelfie], err = h.newImageTypeResponse(*pid)
			if err != nil {
				www.APIInternalError(w, r, err)
				return
			}
		}
		if pid := proof.GovernmentIDPhotoID; pid != nil {
			res.Types[photoTypeGovernmentID], err = h.newImageTypeResponse(*pid)
			if err != nil {
				www.APIInternalError(w, r, err)
				return
			}
		}
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (h *parentalConsentImageAPIHandler) newImageTypeResponse(pid int64) (*imageTypeResponse, error) {
	url, err := h.mediaStore.SignedURL(pid, time.Hour*24)
	if err != nil {
		return nil, err
	}
	return &imageTypeResponse{
		URL: url,
	}, nil
}
