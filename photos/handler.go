package photos

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws"
	"github.com/sprucehealth/backend/libs/golog"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"

	goamz "github.com/sprucehealth/backend/third_party/launchpad.net/goamz/aws"
	"github.com/sprucehealth/backend/third_party/launchpad.net/goamz/s3"
)

type Handler struct {
	dataAPI api.DataAPI
	awsAuth aws.Auth
	bucket  string
	region  goamz.Region
}

type getRequest struct {
	PhotoId     int64  `schema:"photo_id,required"`
	ClaimerType string `schema:"claimer_type,required"`
	ClaimerId   int64  `schema:"claimer_id,required"`
}

type uploadResponse struct {
	PhotoId int64 `json:"photo_id,string"`
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
	region, ok := goamz.Regions[u.Host]
	if !ok {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Invalid region: "+u.Host)
		return
	}
	pathParts := strings.SplitN(u.Path, "/", 3)
	bucketName := pathParts[1]
	path := pathParts[2]

	s3conn := s3.New(common.AWSAuthAdapter(h.awsAuth), region)
	bucket := s3conn.Bucket(bucketName)
	rc, header, err := bucket.GetReader(path)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get photo: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", header.Get("Content-Type"))
	w.Header().Set("Content-Length", header.Get("Content-Length"))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		golog.Errorf("Failed to send photo image: %s", err.Error())
	}
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

	data, err := ioutil.ReadAll(file)
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Failed to read photo data: "+err.Error())
		return
	}

	uidBytes := make([]byte, 16)
	if _, err := rand.Read(uidBytes); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to generate key: "+err.Error())
		return
	}
	uid := hex.EncodeToString(uidBytes)
	path := fmt.Sprintf("photo-%s", uid)
	url := fmt.Sprintf("s3://%s/%s/%s", h.region.Name, h.bucket, path)

	contentType := handler.Header.Get("Content-Type")

	s3conn := s3.New(common.AWSAuthAdapter(h.awsAuth), h.region)
	bucket := s3conn.Bucket(h.bucket)
	err = bucket.Put(path, data, contentType, s3.BucketOwnerFull, map[string][]string{"x-amz-server-side-encryption": {"AES256"}})
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to upload photo: "+err.Error())
		return
	}

	id, err := h.dataAPI.AddPhoto(personId, url, contentType)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to add photo: "+err.Error())
		return
	}

	res := &uploadResponse{
		PhotoId: id,
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
