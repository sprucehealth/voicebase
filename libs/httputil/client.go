package httputil

import "net/http"

type Client interface {
	Head(url string) (*http.Response, error)
}

type DefaultClient struct{}

func (c *DefaultClient) Head(url string) (*http.Response, error) {
	return http.Head(url)
}
