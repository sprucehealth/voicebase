package apiservice

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var ErrBadAuthToken = errors.New("BadAuthToken")

const (
	GENERIC_USER_ERROR = "Something went wrong on our end. Apologies for the inconvenience and please try again later!"
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

func WriteJSONToHTTPResponseWriter(w http.ResponseWriter, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}

func WriteDeveloperError(w http.ResponseWriter, httpStatusCode int, errorString string) error {
	w.WriteHeader(httpStatusCode)
	developerError := new(ErrorResponse)
	developerError.DeveloperError = errorString
	developerError.UserError = GENERIC_USER_ERROR
	enc := json.NewEncoder(w)
	return enc.Encode(developerError)
}

func WriteUserError(w http.ResponseWriter, httpStatusCode int, errorString string) error {
	w.WriteHeader(httpStatusCode)
	userError := new(ErrorResponse)
	userError.UserError = errorString
	enc := json.NewEncoder(w)
	return enc.Encode(userError)
}
