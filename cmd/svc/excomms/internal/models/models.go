package models

import (
	"time"

	"github.com/sprucehealth/backend/libs/phone"
)

// ProvisionedPhoneNumber represents a provisioned phone number
// for a specific purpose.
type ProvisionedPhoneNumber struct {
	PhoneNumber    phone.Number
	ProvisionedFor string
	Provisioned    time.Time
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
}

// Event represents an entry pertaining to an external
// communication along with its corresponding data.
type Event struct {
	Source      string
	Destination string
	Data        interface{}
	Type        string
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
	PhoneNumber         phone.Number
	DestinationEntityID string
	OwnerEntityID       string
	OrganizationID      string
	Expires             time.Time
}
