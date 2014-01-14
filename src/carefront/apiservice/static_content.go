package apiservice

import (
	"carefront/api"
	"github.com/gorilla/schema"
	"net/http"
)

type StaticContentHandler struct {
	DataApi               api.DataAPI
	ContentStorageService api.CloudStorageAPI
	BucketLocation        string
	Region                string
}

type StaticContentRequestData struct {
	ContentTag string `schema:"content_tag"`
}

func (s *StaticContentHandler) NonAuthenticated() bool {
	return true
}

func (s *StaticContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getContent(w, r)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusNotFound, nil)
	}
}

func (s *StaticContentHandler) getContent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(StaticContentRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input to check elligibility: "+err.Error())
		return
	}

	rawData, responseHeader, err := s.ContentStorageService.GetObjectAtLocation(s.BucketLocation, requestData.ContentTag, s.Region)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get static content: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", responseHeader["Content-Type"][0])
	w.Write(rawData)
	w.WriteHeader(http.StatusOK)
}
