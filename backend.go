package voicebase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	prodAPIURL = "https://apis.voicebase.com/v3"
)

// Backend is an interface for making calls against the Voicebase service.
// This interface exists to enable mocking during testing if needed.
type Backend interface {
	Call(ctx context.Context, method, path, key string, v interface{}) error
	CallMultipart(ctx context.Context, method, path, key, boundary string, body io.Reader, v interface{}) error
}

// BackendConfiguration is the internal implementation for making HTTP calls to Voicebase.
type BackendConfiguration struct {
	HTTPClient *http.Client
}

// GetBackend returns the currently used backend in the binding.
func GetBackend() Backend {
	return BackendConfiguration{
		HTTPClient: httpClient,
	}
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// SetHTTPClient overrides the default HTTP client.
func SetHTTPClient(client *http.Client) {
	httpClient = client
}

func (s BackendConfiguration) CallMultipart(ctx context.Context, method, path, key, boundary string, body io.Reader, v interface{}) error {
	contentType := "multipart/form-data; boundary=" + boundary

	req, err := s.NewRequest(ctx, method, path, key, contentType, body)
	if err != nil {
		return err
	}

	return s.Do(req, v)
}

func (s BackendConfiguration) Call(ctx context.Context, method, path, key string, v interface{}) error {
	req, err := s.NewRequest(ctx, method, path, key, "", nil)
	if err != nil {
		return err
	}

	return s.Do(req, v)
}

// NewRequest is used by Call to generate an http.Request.
func (s BackendConfiguration) NewRequest(ctx context.Context, method, path, key, contentType string, body io.Reader) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = prodAPIURL + path

	req, err := http.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	req.Header.Add("Authorization", "Bearer "+key)

	return req, nil
}

// Do is used by Call to execute an API request and parse the response. It uses
// the backend's HTTP client to execute the request and unmarshals the response
// into v. It also handles unmarshaling errors returned by the API.
func (s BackendConfiguration) Do(req *http.Request, v interface{}) error {
	res, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		var vErr Error
		if err := json.NewDecoder(res.Body).Decode(&vErr); err != nil {
			vErr.Status = res.StatusCode
		}
		return &vErr
	}

	if v != nil {
		return json.NewDecoder(res.Body).Decode(&v)
	}

	return nil
}
