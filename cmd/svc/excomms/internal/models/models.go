package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/phone"
)

// EndpointType represents a unique identifier to communicate
// with a user over a channel.
type EndpointType string

const (
	EndpointTypePhone = EndpointType("phone")
	EndpointTypeEmail = EndpointType("email")
)

func GetEndpointType(str string) (EndpointType, error) {
	switch str {
	case "phone":
		return EndpointTypePhone, nil
	case "email":
		return EndpointTypeEmail, nil
	}
	return EndpointType(""), fmt.Errorf("Unknown endpoint type: %s", str)
}

func (e *EndpointType) Scan(src interface{}) error {
	var err error
	var et EndpointType
	switch v := src.(type) {
	case string:
		et, err = GetEndpointType(v)
	case []byte:
		et, err = GetEndpointType(string(v))
	}
	*e = et
	return err
}

func (e EndpointType) Value() (driver.Value, error) {
	return string(e), nil
}

// ProvisionedEndpoint represents a provisioned endpoint for a specific purpose.
type ProvisionedEndpoint struct {
	Endpoint               string
	EndpointType           EndpointType
	ProvisionedFor         string
	Provisioned            time.Time
	Deprovisioned          bool
	DeprovisionedTimestamp *time.Time
	DeprovisionedReason    *string
}

// CallRequest represents a request to make a call from the source to the destination
// before the expiration time.
type CallRequest struct {
	Source         phone.Number
	Destination    phone.Number
	Proxy          phone.Number
	Requested      time.Time
	OrganizationID string
	CallSID        string
	CallerEntityID string
	CalleeEntityID string
}

// IncomingCall represents a call from an external party to an organization.
type IncomingCall struct {
	CallSID        string
	Source         phone.Number
	Destination    phone.Number
	OrganizationID string
	Urgent         bool
	AfterHours     bool
}

// CallEvent represents an entry pertaining to a call event
// along with its corresponding data.
type CallEvent struct {
	Source      string
	Destination string
	Data        interface{}
	Type        string
}

type ByExpiresProxyPhoneNumbers []*ProxyPhoneNumber

func (a ByExpiresProxyPhoneNumbers) Len() int      { return len(a) }
func (a ByExpiresProxyPhoneNumbers) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByExpiresProxyPhoneNumbers) Less(i, j int) bool {
	if a[i].Expires == nil && a[j].Expires == nil {
		return false
	} else if a[i].Expires == nil {
		return true
	} else if a[j].Expires == nil {
		return false
	}

	return a[i].Expires.Before(*a[j].Expires)
}

// ProxyPhoneNumber represents a phone number that dials out to a specific
// phone number when the proxy phone number is dialed.
type ProxyPhoneNumber struct {
	PhoneNumber phone.Number
	Expires     *time.Time
}

// ProxyPhoneNumberReservation represents a particular reservation to dial a specific
// number.
type ProxyPhoneNumberReservation struct {
	ProxyPhoneNumber       phone.Number
	OriginatingPhoneNumber phone.Number
	DestinationPhoneNumber phone.Number
	DestinationEntityID    string
	OwnerEntityID          string
	OrganizationID         string
	Created                time.Time
	Expires                time.Time
}

// Media represents an object uploaded to cloud storage.
type Media struct {
	ID       string
	Type     string
	Location string
	Name     *string
	Duration time.Duration
}
