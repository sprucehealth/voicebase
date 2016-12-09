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
	Token                   string
	OrganizationEntityID    string
	InviterEntityID         string
	Type                    InviteType
	Email                   string
	PhoneNumber             string
	URL                     string
	ParkedEntityID          string
	Created                 time.Time
	Values                  map[string]string
	Tags                    []string
	VerificationRequirement VerificationRequirement
}

// InviteUpdate represents the mutable aspects of an invite
type InviteUpdate struct {
	Tags []string
}

type VerificationRequirement string

const (
	// PhoneMatchRequired indicates that the phone number associated with the invite
	// should match the phone number entered by the user when attempting to create their
	// account.
	PhoneMatchRequired VerificationRequirement = "PHONE_MATCH"

	// PhoneVerificationRequired indicates that phone number should be verified
	// when creating the account.
	PhoneVerificationRequired VerificationRequirement = "PHONE_VERIFICATION"

	// EmailVerificationRequired indicates that the email address of the patient should be
	// verified when creating the account.
	EmailVerificationRequired VerificationRequirement = "EMAIL_VERIFICATION"
)
