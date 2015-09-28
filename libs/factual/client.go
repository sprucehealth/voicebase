package factual

import (
	"encoding/json"
	"fmt"

	"github.com/mrjones/oauth"
)

const apiBaseURL = "http://api.v3.factual.com"

const (
	statusOK    = "ok"
	statuserror = "error"
)

// Client for the Factual API
type Client struct {
	consumer *oauth.Consumer
	token    *oauth.AccessToken
}

type response struct {
	Version  int    `json:"version"`
	Status   string `json:"status"`
	Response struct {
		Data         interface{} `json:"data"`
		IncludedRows int         `json:"included_rows"`
	} `json:"response,omitempty"`
	// For errors
	ErrorType string `json:"error_type,omitempty"`
	Message   string `json:"message,omitempty"`
	Data      string `json:"data,omitempty"`
}

// APIError is an API response error
type APIError struct {
	ErrorType string `json:"error_type,omitempty"`
	Message   string `json:"message,omitempty"`
	Data      string `json:"data,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("factual: [%s] %s", e.ErrorType, e.Message)
}

// New returns a new client with the provided oauth key and secret
func New(key, secret string) *Client {
	return &Client{
		consumer: oauth.NewConsumer(key, secret, oauth.ServiceProvider{}),
		token:    &oauth.AccessToken{},
	}
}

func (c *Client) get(path string, params map[string]string, res interface{}) (int, error) {
	hres, err := c.consumer.Get(apiBaseURL+path, params, c.token)
	if err != nil {
		return 0, fmt.Errorf("factual: failed to GET %s: %s", path, err)
	}
	defer hres.Body.Close()
	var r response
	r.Response.Data = res
	if err := json.NewDecoder(hres.Body).Decode(&r); err != nil {
		return 0, fmt.Errorf("factual: failed to decode JSON response: %s", err)
	}
	if r.Status != statusOK {
		return 0, &APIError{ErrorType: r.ErrorType, Message: r.Message, Data: r.Data}
	}
	return r.Response.IncludedRows, nil
}
