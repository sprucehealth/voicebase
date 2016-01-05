package twilio

import (
	"errors"

	"github.com/google/go-querystring/query"
)

type AvailablePhoneNumbersService struct {
	client *Client
}

type AvailablephoneNumbersIFace interface {
	ListLocal(params AvailablePhoneNumbersParams) ([]*AvailablePhoneNumber, *Response, error)
}

type AvailablePhoneNumber struct {
	FriendlyName string          `json:"friendly_name"`
	PhoneNumber  string          `json:"phone_number"`
	LATA         string          `json:"lata"`
	RateCenter   string          `json:"rate_center"`
	Latitude     float64         `json:"latitude,string"`
	Longitude    float64         `json:"longitude,string"`
	Region       string          `json:"region"`
	PostalCode   string          `json:"postal_code"`
	ISOCountry   string          `json:"iso_country"`
	Capabilities map[string]bool `json:"capabilities"`
}

type AvailablePhoneNumbersResponse struct {
	URI                   string                  `json:"uri"`
	AvailablePhoneNumbers []*AvailablePhoneNumber `json:"available_phone_numbers"`
}

type AvailablePhoneNumbersParams struct {
	AreaCode                      string `url:"AreaCode,omitempty"`
	SMSEnabled                    bool   `url:"SmsEnabled"`
	MMSEnabled                    bool   `url:"MmsEnabled"`
	VoiceEnabled                  bool
	ExcludeAllAddressRequired     bool
	ExcludeLocalAddressRequired   bool
	ExcludeForeignAddressRequired bool
}

func (a AvailablePhoneNumbersParams) Validate() error {
	if !a.SMSEnabled && !a.MMSEnabled && !a.VoiceEnabled {
		return errors.New("Atleast one of the capabilities should be specified")
	}

	return nil
}

func (a *AvailablePhoneNumbersService) ListLocal(params AvailablePhoneNumbersParams) ([]*AvailablePhoneNumber, *Response, error) {
	if err := params.Validate(); err != nil {
		return nil, nil, err
	}

	u := a.client.EndPoint("AvailablePhoneNumbers", "US", "Local")

	v, err := query.Values(params)
	if err != nil {
		return nil, nil, err
	}

	req, err := a.client.NewRequest("GET", u.String()+"?"+v.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}

	apr := new(AvailablePhoneNumbersResponse)
	resp, err := a.client.Do(req, apr)
	if err != nil {
		return nil, nil, err
	}

	return apr.AvailablePhoneNumbers, resp, nil
}
