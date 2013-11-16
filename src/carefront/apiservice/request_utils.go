package apiservice

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

var ErrBadAuthToken = errors.New("BadAuthToken")

const (
	genericUserErrorMessage = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
)

func GetAuthTokenFromHeader(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrBadAuthToken
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "token" {
		return "", ErrBadAuthToken
	}
	return parts[1], nil
}

type ErrorResponse struct {
	DeveloperError string `json:"developer_error,omitempty"`
	UserError      string `json:"user_error,omitempty"`
}

func WriteJSONToHTTPResponseWriter(w http.ResponseWriter, httpStatusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		log.Printf("apiservice: failed to json encode: %+v", err)
	}
}

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	developerError := new(ErrorResponse)
	developerError.DeveloperError = errorString
	developerError.UserError = genericUserErrorMessage
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, developerError)
}

func WriteUserError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	userError := new(ErrorResponse)
	userError.UserError = errorString
	WriteJSONToHTTPResponseWriter(w, httpStatusCode, userError)
}
