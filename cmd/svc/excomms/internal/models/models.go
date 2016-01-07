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

type ByLastReservedProxyPhoneNumbers []*ProxyPhoneNumber

func (a ByLastReservedProxyPhoneNumbers) Len() int      { return len(a) }
func (a ByLastReservedProxyPhoneNumbers) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLastReservedProxyPhoneNumbers) Less(i, j int) bool {
	if a[i].LastReserved == nil && a[j].LastReserved == nil {
		return false
	} else if a[i].LastReserved == nil {
		return true
	} else if a[j].LastReserved == nil {
		return false
	}

	return a[i].LastReserved.Before(*a[j].LastReserved)
}

// ProxyPhoneNumber represents a phone number that dials out to a specific
// phone number when the proxy phone number is dialed.
type ProxyPhoneNumber struct {
	PhoneNumber  phone.Number
	Expires      *time.Time
	LastReserved *time.Time
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
