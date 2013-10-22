// Package apiservice contains the GetSignedUrlHandler
//	Description:
//		Return signed urls for all photos belonging to a particular case.
//		The photographs will authorized for download for 10 minutes.
//
//	Request:
//		GET /v1/imagesforcase/<case_id>
//
//	Response:
//		Content-Type: application/json
//		Content:
//			{
//				"signedUrls" : [ <signed_url_1>, <signed_url_2>, ... ]
//			}
package apiservice

import (
	"carefront/api"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type GetSignedUrlsHandler struct {
	PhotoApi           api.Photo
	CaseBucketLocation string
}

type GetSignedUrlsResponse struct {
	SignedUrls []string `json:"signedUrls"`
}

type GetSignedUrlsErrorResponse struct {
	GetSignedUrlErrorString string `json:"error"`
}

func (h *GetSignedUrlsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caseId := r.URL
	pathPieces := strings.Split(caseId.String(), "/")
	signedUrls, err := h.PhotoApi.GenerateSignedUrlsForKeysInBucket(h.CaseBucketLocation, pathPieces[3], time.Now().Add(10*time.Minute))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc := json.NewEncoder(w)
		enc.Encode(GetSignedUrlsErrorResponse{err.Error()})
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(GetSignedUrlsResponse{signedUrls})
}
