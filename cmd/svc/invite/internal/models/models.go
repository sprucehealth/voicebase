package models

import (
	"time"
)

// InviteType represents the purpose/type of the represented invite
type InviteType string

const (
	// ColleagueInvite represents invites for other providers
	ColleagueInvite InviteType = "COLLEAGUE"
	// PatientInvite represents invites for patients
	PatientInvite InviteType = "PATIENT"
	// OrganizationCodeInvite represents invites for organizations
	OrganizationCodeInvite InviteType = "ORGANIZATION_CODE"
)

// Invite represents an invite into the Baymax system
type Invite struct {
	Token                string
	OrganizationEntityID string
	InviterEntityID      string
	Type                 InviteType
	Email                string
	PhoneNumber          string
	URL                  string
	ParkedEntityID       string
	Created              time.Time
	Values               map[string]string
	Tags                 []string
}

// InviteUpdate represents the mutable aspects of an invite
type InviteUpdate struct {
	Tags []string
}
