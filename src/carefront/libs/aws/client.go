package aws

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Keys       Keys
	role       string
	cred       *Credentials
	httpClient *http.Client
}

func (c *Client) refreshKeys() {
	if c.cred == nil || c.role == "" {
		return
	}

	if c.cred.Expiration.Before(time.Now()) {
		cred, err := CredentialsForRole(c.role)
		if err != nil {
			log.Printf("aws: failed to refresh credentials for role %s: %s", c.role, err.Error())
		} else {
			c.cred = cred
			c.Keys = cred.Keys()
		}
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.refreshKeys()
	Sign(c.cred.Keys(), req)
	return c.httpClient.Do(req)
}

func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) Head(url string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(req)
}

func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

func ClientForRole(role string, httpClient *http.Client) (*Client, error) {
	cred, err := CredentialsForRole(role)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		role:       role,
		cred:       cred,
		Keys:       cred.Keys(),
		httpClient: httpClient,
	}, nil
}

func ClientWithKeys(keys Keys, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		Keys:       keys,
		httpClient: httpClient,
	}, nil
}
