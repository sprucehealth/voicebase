package hint

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	prodAPIURL         = "https://api.hint.com/api"
	stagingAPIURL      = "https://api.staging.hint.com/api"
	prodProviderURL    = "https://provider.hint.com"
	stagingProviderURL = "https://provider.staging.hint.com"
)

// apiURL returns the production or staging url based on the
// Testing bool
func apiURL() string {
	if Testing {
		return stagingAPIURL
	}
	return prodAPIURL
}

// ProviderURL returns the production or staging url for where the provider
// logs in to use Hint.
func ProviderURL() string {
	if Testing {
		return stagingProviderURL
	}
	return prodProviderURL
}

// Backend is an interface for making calls against a Hint service.
// This interface exists to enable mocking during testing if needed.
type Backend interface {
	Call(method, path, key string, params Params, v interface{}) (http.Header, error)
}

// BackendConfiguration is the internal implementation for making HTTP calls to Hint.
type BackendConfiguration struct {
	HTTPClient *http.Client
}

// GetBackend returns the currently used backend in the binding.
func GetBackend() Backend {
	return BackendConfiguration{
		HTTPClient: httpClient,
	}
}

//Key is the Hint Partner API key used globally in the binding.
var Key string

// Testing indicates whether to use the staging or production URL
var Testing bool

var httpClient = &http.Client{Timeout: 30 * time.Second}

// SetHTTPClient overrides the default HTTP client.
func SetHTTPClient(client *http.Client) {
	httpClient = client
}

func (s BackendConfiguration) Call(method, path, key string, params Params, v interface{}) (http.Header, error) {
	var data io.Reader
	if params != nil {
		if err := params.Validate(); err != nil {
			return nil, err
		}

		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		data = bytes.NewReader(jsonData)
	}

	req, err := s.NewRequest(method, path, key, data)
	if err != nil {
		return nil, err
	}

	responseHeaders, err := s.Do(req, v)
	if err != nil {
		return nil, err
	}

	return responseHeaders, nil
}

// NewRequest is used by Call to generate an http.Request.
func (s BackendConfiguration) NewRequest(method, path, key string, body io.Reader) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = apiURL() + path

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(key, "")

	switch method {
	case "POST", "PATCH", "PUT":
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// Do is used by Call to execute an API request and parse the response. It uses
// the backend's HTTP client to execute the request and unmarshals the response
// into v. It also handles unmarshaling errors returned by the API.
func (s BackendConfiguration) Do(req *http.Request, v interface{}) (http.Header, error) {

	res, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		var hintError Error
		if err := json.Unmarshal(resBody, &hintError); err != nil {
			return nil, err
		}
		return nil, &hintError
	}

	if v != nil {
		if err := json.Unmarshal(resBody, v); err != nil {
			return nil, err
		}
		return res.Header, nil
	}

	return res.Header, nil
}
