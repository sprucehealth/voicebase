package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sprucehealth/backend/apiservice"
)

func do(baseURL, authToken, hostHeader, method, path string, params url.Values, req, res interface{}, headers http.Header) error {
	var body io.Reader
	if req != nil {
		if r, ok := req.(io.Reader); ok {
			body = r
		} else if b, ok := req.([]byte); ok {
			body = bytes.NewReader(b)
		} else {
			if headers == nil {
				headers = http.Header{}
			}
			headers.Set("Content-Type", "application/json")
			b := &bytes.Buffer{}
			if err := json.NewEncoder(b).Encode(req); err != nil {
				return err
			}
			body = b
		}
	}

	u := baseURL + path
	if len(params) != 0 {
		u += "?" + params.Encode()
	}
	httpReq, err := http.NewRequest(method, u, body)
	if err != nil {
		return err
	}
	for k, v := range headers {
		httpReq.Header[k] = v
	}
	if authToken != "" {
		httpReq.Header.Set("Authorization", "token "+authToken)
	}
	if hostHeader != "" {
		httpReq.Host = hostHeader
	}
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpRes.Body.Close()

	switch httpRes.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("apiclient: API endpoint '%s%s' not found", baseURL, path)
	case http.StatusMethodNotAllowed:
		return fmt.Errorf("apiclient: method %s not allowed on endpoint '%s'", method, path)
	case http.StatusOK:
		if res != nil {
			return json.NewDecoder(httpRes.Body).Decode(res)
		}
		return nil
	}

	var e apiservice.SpruceError
	if err := json.NewDecoder(httpRes.Body).Decode(&e); err != nil {
		return fmt.Errorf("apiclient: failed to decode error on %d status code: %s", httpRes.StatusCode, err.Error())
	}
	e.HTTPStatusCode = httpRes.StatusCode
	return &e
}
