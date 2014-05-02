package photos

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/aws"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/schema"

	goamz "launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

type Handler struct {
	dataAPI api.DataAPI
	awsAuth aws.Auth
	bucket  string
	region  goamz.Region
}

type getRequest struct {
	PhotoId     int64  `schema:"photo_id,required"`
	ClaimerType string `schema:"claimer_rype,required"`
	ClaimerId   int64  `schema:"claimer_id,required"`
}

type uploadResponse struct {
	PhotoId int64 `json:"photo_id"`
}

func NewHandler(dataAPI api.DataAPI, awsAuth aws.Auth, bucket, region string) *Handler {
	awsRegion, ok := goamz.Regions[region]
	if !ok {
		awsRegion = goamz.USEast
	}

	return &Handler{
		dataAPI: dataAPI,
		awsAuth: awsAuth,
		bucket:  bucket,
		region:  awsRegion,
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

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var req getRequest
	if err := schema.NewDecoder().Decode(&req, r.Form); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	photo, err := h.dataAPI.GetPhoto(req.PhotoId)
	if err == api.NoRowsError {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get photo: "+err.Error())
		return
	}

	// TODO: need a more robust check for verifying access rights
	if photo.ClaimerType != req.ClaimerType || photo.ClaimerId != req.ClaimerId {
		http.NotFound(w, r)
		return
	}

	u, err := url.Parse(photo.URL)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to parse photo URL: "+err.Error())
		return
	}
	region := u.Host
	bucket := u.Path

	s3conn := s3.New(common.AWSAuthAdapter(h.awsAuth), h.region)
	bucket := s3conn.Bucket(h.bucket)
	bucket.SignedURL(path, expires, additionalHeaders)
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	var personId int64
	doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err == nil {
		personId, err = h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for doctor: "+err.Error())
			return
		}
	} else if patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId); err == nil {
		personId, err = h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for patient: "+err.Error())
			return
		}
	} else {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get patient or doctor: "+err.Error())
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Missing or invalid photo in parameters: "+err.Error())
		return
	}

	uidBytes := make([]byte, 16)
	if _, err := rand.Read(uidBytes); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to generate key: "+err.Error())
		return
	}
	uid := hex.EncodeToString(uidBytes)
	url := fmt.Sprintf("s3://%s/%s/%s", h.region, h.bucket, uid)

	contentType := handler.Header.Get("Content-Type")

	s3conn := s3.New(common.AWSAuthAdapter(h.awsAuth), h.region)
	bucket := s3conn.Bucket(h.bucket)
	err = bucket.PutReader(uid, file, r.ContentLength, contentType, s3.BucketOwnerFull, map[string][]string{"x-amz-server-side-encryption": {"AES256"}})
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to upload photo: "+err.Error())
		return
	}

	id, err := h.dataAPI.AddPhoto(personId, url, contentType)
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Failed to add photo: "+err.Error())
		return
	}

	res := &uploadResponse{
		PhotoId: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
