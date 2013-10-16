package main

import (
	"encoding/json"
	"log"
	"net/http"
	"carefront/api"
	"bytes"
	"io/ioutil"
	"os"
)

type PhotoUploadHandler struct {
	PhotoApi api.Photo
}

type PhotoUploadResponse struct {
	PhotoUrl string `json:"photoUrl"`
}

type PhotoUploadErrorResponse struct {
	PhotoUploadErrorString string `json:"error"`
}

func (h *PhotoUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	file,_,err := r.FormFile("photo")
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
 	
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString(caseId)
	buffer.WriteString("/")
	buffer.WriteString("photo")

	// synchronously upload the image and return a response back to the user when the
	// upload is complete
	_, err = h.PhotoApi.Upload(data, buffer.String(), os.Getenv("CASE_BUCKET"))
	
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.Encode(PhotoUploadErrorResponse{err.Error()})
		return
	}	 

	enc := json.NewEncoder(w)
	enc.Encode(PhotoUploadResponse{"photo"})	
}

func (h *PhotoUploadHandler) NonAuthenticated() bool {
	return true
}

