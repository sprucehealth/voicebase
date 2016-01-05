package twilio

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

type CallService struct {
	client *Client
}

type CallIFace interface {
	Create(v url.Values) (*Call, *Response, error)
	Make(params CallParams) (*Call, *Response, error)
	Modify(sid string, params CallModificationParams) (*Call, *Response, error)
	Get(sid string) (*Call, *Response, error)
}

type Call struct {
	SID            string    `json:"sid"`
	ParentCallSID  string    `json:"parent_call_sid"`
	DateCreated    Timestamp `json:"date_created"`
	DateUpdated    Timestamp `json:"date_updated,omitempty"`
	AccountSid     string    `json:"account_sid"`
	To             string    `json:"to"`
	From           string    `json:"from"`
	PhoneNumberSID string    `json:"phone_number_sid"`
	Status         string    `json:"status"`
	StartTime      Timestamp `json:"start_time"`
	EndTime        Timestamp `json:"end_time,omitempty"`
	Duration       string    `json:"duration"`
	Price          Price     `json:"price,omitempty"`
	PriceUnit      string    `json:"price_unit"`
	Direction      string    `json:"direction"`
	AnsweredBy     string    `json:"answered_by"`
	ForwardedFrom  string    `json:"forwarded_from"`
	CallerName     string    `json:"caller_name"`
	Uri            string    `json:"uri"`
}

type CallParams struct {
	From           string
	To             string
	URL            string `url:"Url,omitempty"`
	ApplicationSID string `url:"ApplicationSid,omitempty"`
}

func (c CallParams) Validate() error {
	if c.URL == "" {
		return errors.New("URL is required.")
	}

	return nil
}

func (c *CallService) Create(v url.Values) (*Call, *Response, error) {
	u := c.client.EndPoint("Calls")

	req, err := c.client.NewRequest("POST", u.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, nil, err
	}

	call := new(Call)
	resp, err := c.client.Do(req, call)
	if err != nil {
		return nil, resp, err
	}

	return call, resp, err
}

func (c *CallService) Make(params CallParams) (*Call, *Response, error) {
	if err := params.Validate(); err != nil {
		return nil, nil, err
	}

	v, err := query.Values(params)
	if err != nil {
		return nil, nil, err
	}
	return c.Create(v)
}

type CallModificationParams struct {
	URL                  string `url:"Url,omitempty"`
	Method               string `url:"Method,omitempty"`
	Status               string `url:"Status,omitempty"`
	FallbackURL          string `url:"FallbackUrl,omitempty"`
	FallbackMethod       string `url:"FallbackMethod,omitempty"`
	StatusCallback       string `url:"StatusCallback,omitempty"`
	StatusCallbackMethod string `url:"StatusCallbackMethod,omitempty"`
}

func (c CallModificationParams) Validate() error {
	if c.URL == "" && c.Status == "" {
		return errors.New("Either status or url is required to modify the call.")
	}

	if c.URL != "" {
		if _, err := url.Parse(c.URL); err != nil {
			return errors.New("Invalid url.")
		}
	}

	switch c.Status {
	case "", "canceled", "completed":
	default:
		return fmt.Errorf("Invalid call status for modification: %s. Can only be canceled or completed.", c.Status)
	}

	return nil
}

func (c *CallService) Modify(sid string, params CallModificationParams) (*Call, *Response, error) {
	if err := params.Validate(); err != nil {
		return nil, nil, err
	}

	v, err := query.Values(params)
	if err != nil {
		return nil, nil, err
	}

	u := c.client.EndPoint("Calls", sid)

	req, err := c.client.NewRequest("POST", u.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, nil, err
	}

	call := new(Call)
	resp, err := c.client.Do(req, call)
	if err != nil {
		return nil, resp, err
	}

	return call, resp, err
}

func (c *CallService) Get(sid string) (*Call, *Response, error) {
	u := c.client.EndPoint("Calls", sid)

	req, err := c.client.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	call := new(Call)
	resp, err := c.client.Do(req, call)
	if err != nil {
		return nil, nil, err
	}

	return call, resp, err
}
