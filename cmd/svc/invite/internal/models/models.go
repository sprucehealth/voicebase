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
)

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
}
