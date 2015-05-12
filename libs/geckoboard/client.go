package geckoboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	apiBaseURL      = "https://push.geckoboard.com"
	jsonContentType = "application/json"
)

type Error struct {
	StatusCode int
	Message    string
}

func (e Error) Error() string {
	return fmt.Sprintf("geckoboard: %d: %s", e.StatusCode, e.Message)
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type pushRequest struct {
	APIKey string      `json:"api_key"`
	Data   interface{} `json:"data"`
}

type Client struct {
	apiKey string
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

func (c *Client) Push(widgetKey string, data interface{}) error {
	return c.post("/v1/send/"+widgetKey, pushRequest{APIKey: c.apiKey, Data: data}, nil)
}

func (c *Client) post(path string, req, res interface{}) error {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return err
	}
	hreq, err := http.NewRequest("POST", apiBaseURL+path, body)
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", jsonContentType)
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var e errorResponse
		if err := json.Unmarshal(rbody, &e); err != nil {
			return &Error{StatusCode: resp.StatusCode, Message: string(rbody)}
		}
		er := &Error{StatusCode: resp.StatusCode}
		if e.Message != "" {
			er.Message = e.Message
		} else if e.Error != "" {
			er.Message = e.Error
		} else {
			er.Message = string(rbody)
		}
		return er
	}
	if res != nil {
		if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
			return fmt.Errorf("geckoboard: failed to decode response: %s", err)
		}
	}
	return nil
}
