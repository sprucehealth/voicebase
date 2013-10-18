package main

import (
	"bytes"
	"carefront/api"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PhotoUploadHandler struct {
	PhotoApi api.Photo
	DataApi  *api.DataService
}

type PhotoUploadResponse struct {
	PhotoUrl string `json:"photoUrl"`
}

type PhotoUploadErrorResponse struct {
	PhotoUploadErrorString string `json:"error"`
}

func (h *PhotoUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	file, _, err := r.FormFile("photo")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	caseId := r.FormValue("caseId")
	if caseId == "" {
		w.WriteHeader(http.StatusBadRequest)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{"missing caseId!"})
		return
	}

	photoType := r.FormValue("photoType")
	if photoType == "" {
		w.WriteHeader(http.StatusBadRequest)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{"missing photoType!"})
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	// create a caseImage and mark it as ready for upload
	caseIdInt, err := strconv.ParseInt(caseId, 0, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{"incorrect format for caseId!"})
		return
	}
	photoId, err := h.DataApi.CreatePhotoForCase(caseIdInt, photoType)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString(caseId)
	buffer.WriteString("/")
	buffer.WriteString(strconv.FormatInt(photoId, 10))

	// synchronously upload the image and return a response back to the user when the
	// upload is complete
	signedUrl, err := h.PhotoApi.Upload(data, buffer.String(), os.Getenv("CASE_BUCKET"), time.Now().Add(10*time.Minute))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	// mark the photo upload as complete
	err = h.DataApi.MarkPhotoUploadComplete(caseIdInt, photoId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(PhotoUploadResponse{signedUrl})
}
