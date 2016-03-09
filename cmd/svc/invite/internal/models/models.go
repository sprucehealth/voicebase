package models

import (
	"time"
)

type InviteType string

const (
	ColleagueInvite InviteType = "COLLEAGUE"
)

type Invite struct {
	Token                string
	OrganizationEntityID string
	InviterEntityID      string
	Type                 InviteType
	Email                string
	PhoneNumber          string
	URL                  string
	Created              time.Time
	Values               map[string]string
}
