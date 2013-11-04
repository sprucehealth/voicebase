// Package apiservice is contains the PhotoUploadHandler
//	Description:
//		Upload photos of a particular type (face, back, chest) for a particular case. The request is synchronous and
//		returns a successful result only if the upload and storage in the cloud succeeded.
//
//	Request:
//		POST /v1/upload
//
//	Request-headers:
//		{
//			"Authorization" : "token <auth_token>"
//		}
//
//	Request-body:
//		Content-Type : multipart/form-data
//		Parameters:
//			photo=<photo_binary_data>
//			case_id=<integer>
//			photo_type=[face_middle, face_right, face_left, back, chest]
//
//	Response:
//		Content-Type : application/json
//		Content:
//			{
//				"photoUrl" : <signed_photo_url>
//			}
package apiservice

import (
	"bytes"
	"carefront/api"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PhotoUploadHandler struct {
	PhotoApi           api.Photo
	CaseBucketLocation string
	DataApi            api.DataAPI
}

type PhotoUploadResponse struct {
	PhotoUrl string `json:"photoUrl"`
}

type PhotoUploadErrorResponse struct {
	PhotoUploadErrorString string `json:"error"`
}

func (h *PhotoUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	file, handler, err := r.FormFile("photo")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	caseId := r.FormValue("case_id")
	if caseId == "" {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{"missing caseId!"})
		return
	}

	photoType := r.FormValue("photo_type")
	if photoType == "" {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{"missing photoType!"})
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	// create a caseImage and mark it as ready for upload
	caseIdInt, err := strconv.ParseInt(caseId, 0, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{"incorrect format for caseId!"})
		return
	}
	photoId, err := h.DataApi.CreatePhotoForCase(caseIdInt, photoType)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString(caseId)
	buffer.WriteString("/")
	buffer.WriteString(strconv.FormatInt(photoId, 10))

	// infer extension from filename if one exists
	parts := strings.Split(handler.Filename, ".")
	if len(parts) > 1 {
		buffer.WriteString(".")
		buffer.WriteString(parts[1])
	}

	// synchronously upload the image and return a response back to the user when the
	// upload is complete
	signedUrl, err := h.PhotoApi.Upload(data, handler.Header.Get("Content-Type"), buffer.String(), h.CaseBucketLocation, time.Now().Add(10*time.Minute))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	// mark the photo upload as complete
	err = h.DataApi.MarkPhotoUploadComplete(caseIdInt, photoId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(&w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	WriteJSONToHTTPResponseWriter(&w, PhotoUploadResponse{signedUrl})
}
