package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type staticContentHandler struct {
	dataAPI               api.DataAPI
	contentStorageService api.CloudStorageAPI
	bucketLocation        string
	region                string
}

func NewStaticContentHandler(dataAPI api.DataAPI, contentStorageService api.CloudStorageAPI, bucketLocation, region string) http.Handler {
	return apiservice.SupportedMethods(&staticContentHandler{
		dataAPI:               dataAPI,
		contentStorageService: contentStorageService,
		bucketLocation:        bucketLocation,
		region:                region,
	}, []string{apiservice.HTTP_GET})
}

type StaticContentRequestData struct {
	ContentTag string `schema:"content_tag"`
}

func (s *staticContentHandler) NonAuthenticated() bool {
	return true
}

func (s *staticContentHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
}

func (s *staticContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestData := new(StaticContentRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	rawData, responseHeader, err := s.contentStorageService.GetObjectAtLocation(s.bucketLocation, requestData.ContentTag, s.region)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	w.Header().Set("Content-Type", responseHeader["Content-Type"][0])
	if responseHeader["Content-Encoding"] != nil && len(responseHeader["Content-Encoding"]) > 0 {
		w.Header().Set("Content-Encoding", responseHeader["Content-Encoding"][0])
	}
	w.Write(rawData)
	w.WriteHeader(http.StatusOK)
}
