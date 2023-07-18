package voicebase

import (
	"net/http"
)

type Client struct {
	bearerToken string
	httpClient  *http.Client
}

type ClientConfig struct {
	BearerToken string
	// HTTPClient is optional and will default to http.DefaultClient
	HTTPClient *http.Client
}

func NewClient(cfg ClientConfig) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &Client{
		bearerToken: cfg.BearerToken,
		httpClient:  cfg.HTTPClient,
	}
}
