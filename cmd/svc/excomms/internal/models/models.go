package models

import "time"

// ProvisionedPhoneNumber represents a provisioned phone number
// for a specific purpose.
type ProvisionedPhoneNumber struct {
	PhoneNumber    string
	ProvisionedFor string
	Provisioned    time.Time
}

// CallRequest represents a request to make a call from the source to the destination
// before the expiration time.
type CallRequest struct {
	Source         string
	Destination    string
	Proxy          string
	OrganizationID string
	Requested      time.Time
	Expires        time.Time
	CallSID        *string
}

// Event represents an entry pertaining to an external
// communication along with its corresponding data.
type Event struct {
	Source      string
	Destination string
	Data        interface{}
	Type        string
}
