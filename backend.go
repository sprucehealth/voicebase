package voicebase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	prodAPIURL = "https://apis.voicebase.com/v3"
)

func (c *Client) callMultipart(ctx context.Context, method, path, boundary string, body io.Reader, v interface{}) error {
	contentType := "multipart/form-data; boundary=" + boundary

	req, err := c.newRequest(ctx, method, path, contentType, body)
	if err != nil {
		return err
	}

	return c.do(req, v)
}

func (c *Client) call(ctx context.Context, method, path string, v interface{}) error {
	req, err := c.newRequest(ctx, method, path, "", nil)
	if err != nil {
		return err
	}

	return c.do(req, v)
}

// NewRequest is used by Call to generate an http.Request.
func (c *Client) newRequest(ctx context.Context, method, path, contentType string, body io.Reader) (*http.Request, error) {
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

	req.Header.Add("Authorization", "Bearer "+c.bearerToken)

	return req, nil
}

// Do is used by Call to execute an API request and parse the response. It uses
// the backend's HTTP client to execute the request and unmarshals the response
// into v. It also handles unmarshaling errors returned by the API.
func (c *Client) do(req *http.Request, v interface{}) error {
	res, err := c.httpClient.Do(req)
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
