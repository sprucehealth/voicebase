package main

import (
	"net/http"
	"strings"
	"errors"
)

var ErrBadAuthToken = errors.New("BadAuthToken")

func GetAuthTokenFromHeader(r *http.Request) (string, error) {

	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrBadAuthToken
	}

	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "token" {
		return	"", ErrBadAuthToken 
	}
	return parts[1], nil
}
