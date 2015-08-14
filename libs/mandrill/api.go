package mandrill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/samuel/go-metrics/metrics"
)

const defaultBaseURL = "https://mandrillapp.com/api/1.0"

type Client struct {
	key               string
	baseURL           string
	ipPool            string
	statSendSucceeded *metrics.Counter
	statSendFailed    *metrics.Counter
}

func NewClient(key, ipPool string, metricsRegistry metrics.Registry) *Client {
	c := &Client{
		key:               key,
		baseURL:           defaultBaseURL,
		ipPool:            ipPool,
		statSendSucceeded: metrics.NewCounter(),
		statSendFailed:    metrics.NewCounter(),
	}
	if metricsRegistry != nil {
		metricsRegistry.Add("send/succeeded", c.statSendSucceeded)
		metricsRegistry.Add("send/failed", c.statSendFailed)
	}
	return c
}

func (c *Client) SendMessageTemplate(name string, content []Var, msg *Message, async bool) ([]*SendMessageResponse, error) {
	req := &struct {
		Key             string   `json:"key"`
		TemplateName    string   `json:"template_name"`
		TemplateContent []Var    `json:"template_content"`
		Message         *Message `json:"message"`
		Async           bool     `json:"async"`
		IPPool          string   `json:"ip_pool,omitempty"`
		// SendAt          time.Time   `json:"send_at,omitempty"` // TODO: should be formatted as "YYYY-MM-DD HH:MM:SS"
	}{
		Key:             c.key,
		TemplateName:    name,
		TemplateContent: content,
		Message:         msg,
		Async:           async,
		IPPool:          c.ipPool,
	}
	var res []*SendMessageResponse
	if err := c.post("/messages/send-template", req, &res); err != nil {
		c.statSendFailed.Inc(1)
		return nil, err
	}
	c.statSendSucceeded.Inc(1)
	return res, nil
}

func (c *Client) post(path string, req, res interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := http.Post(c.baseURL+path+".json", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var e Error
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return fmt.Errorf("mandrill: failed to decode error response JSON: %s", err)
		}
		return &e
	}
	return json.NewDecoder(resp.Body).Decode(res)
}

func Bool(b bool) *bool {
	return &b
}
