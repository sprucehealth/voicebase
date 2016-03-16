package twilio

import (
	"errors"
	"strings"

	"github.com/google/go-querystring/query"
)

type IncomingPhoneNumberService struct {
	client *Client
}

type IncomingPhoneNumberIFace interface {
	PurchaseLocal(params PurchasePhoneNumberParams) (*IncomingPhoneNumber, *Response, error)
	List(params ListPurchasedPhoneNumberParams) (*ListPurchasedPhoneNumbersResponse, *Response, error)
	Delete(sid string) (*Response, error)
}

type IncomingPhoneNumber struct {
	SID                  string          `json:"sid"`
	AccountSID           string          `json:"account_sid"`
	FriendlyName         string          `json:"friendly_name"`
	PhoneNumber          string          `json:"phone_number"`
	VoiceURL             string          `json:"voice_url"`
	VoiceMethod          string          `json:"voice_method"`
	VoiceFallbackURL     string          `json:"voice_fallback_url"`
	VoiceFallbackMethod  string          `json:"voice_fallback_method"`
	VoiceCallerIDLookup  bool            `json:"voice_caller_id_lookup"`
	StatusCallback       string          `json:"status_callback"`
	StatusCallbackMethod string          `json:"status_callback_method"`
	VoiceApplicationSID  string          `json:"voice_application_sid"`
	DateCreated          Timestamp       `json:"date_created"`
	DateUpdated          Timestamp       `json:"date_updated"`
	SMSURL               string          `json:"sms_url"`
	SMSMethod            string          `json:"sms_method"`
	SMSFallbackURL       string          `json:"sms_fallback_url"`
	SMSFallbackMethod    string          `json:"sms_fallback_method"`
	SMSApplicationSID    string          `json:"sms_application_sid"`
	Capabilities         map[string]bool `json:"capabilities"`
	APIVersion           string          `json:"api_version"`
	URI                  string          `json:"uri"`
}

type ListPurchasedPhoneNumbersResponse struct {
	Page                 int                    `json:"page"`
	PageSize             int                    `json:"page_size"`
	URI                  string                 `json:"uri"`
	FirstPageURI         string                 `json:"first_page_uri"`
	PreviousPageURI      string                 `json:"previous_page_uri"`
	IncomingPhoneNumbers []*IncomingPhoneNumber `json:"incoming_phone_numbers"`
}

type PurchasePhoneNumberParams struct {
	AreaCode            string `url:"AreaCode,omitempty"`
	PhoneNumber         string `url:"PhoneNumber,omitempty"`
	VoiceApplicationSID string `url:"VoiceApplicationSid,omitempty"`
	SMSApplicationSID   string `url:"SmsApplicationSid,omitempty"`
}

type ListPurchasedPhoneNumberParams struct {
	PhoneNumber string `url:"PhoneNumber,omitempty"`
}

func (p PurchasePhoneNumberParams) Validate() error {
	if p.AreaCode == "" && p.PhoneNumber == "" {
		return errors.New("Either area code or phone number is required.")
	}
	return nil
}

func (i *IncomingPhoneNumberService) PurchaseLocal(params PurchasePhoneNumberParams) (*IncomingPhoneNumber, *Response, error) {
	if err := params.Validate(); err != nil {
		return nil, nil, err
	}

	u := i.client.EndPoint("IncomingPhoneNumbers")

	v, err := query.Values(params)
	if err != nil {
		return nil, nil, err
	}

	req, err := i.client.NewRequest("POST", u.String(), strings.NewReader(v.Encode()))
	if err != nil {
		return nil, nil, err
	}

	ip := new(IncomingPhoneNumber)
	resp, err := i.client.Do(req, ip)
	if err != nil {
		return nil, nil, err
	}

	return ip, resp, nil
}

func (i *IncomingPhoneNumberService) List(params ListPurchasedPhoneNumberParams) (*ListPurchasedPhoneNumbersResponse, *Response, error) {
	if params.PhoneNumber == "" {
		return nil, nil, errors.New("phone number required.")
	}

	u := i.client.EndPoint("IncomingPhoneNumbers")

	v, err := query.Values(params)
	if err != nil {
		return nil, nil, err
	}

	req, err := i.client.NewRequest("GET", u.String()+"?"+v.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	l := new(ListPurchasedPhoneNumbersResponse)
	resp, err := i.client.Do(req, l)
	if err != nil {
		return nil, nil, err
	}

	return l, resp, nil
}

func (i *IncomingPhoneNumberService) Delete(sid string) (*Response, error) {
	if sid == "" {
		return nil, errors.New("phone number sid is required")
	}

	u := i.client.EndPoint("IncomingPhoneNumbers", sid)

	req, err := i.client.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return i.client.Do(req, nil)
}
