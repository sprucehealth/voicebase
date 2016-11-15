package httputil

import "net/http"

type Client interface {
	Head(url string) (*http.Response, error)
}
