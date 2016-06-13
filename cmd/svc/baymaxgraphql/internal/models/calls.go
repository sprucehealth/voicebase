package models

// Call role enum
const (
	CallRoleCaller    = "CALLER"
	CallRoleRecipient = "RECIPIENT"
)

// Call state enum
const (
	CallStatePending   = "PENDING"
	CallStateAccepted  = "ACCEPTED"
	CallStateDeclined  = "DECLINED"
	CallStateConnected = "CONNECTED"
	CallStateFailed    = "FAILED"
	CallStateCompleted = "COMPLETED"
)

// Call channel enum
const (
	CallChannelTypePhone = "PHONE"
	CallChannelTypeVOIP  = "VOIP"
	CallChannelTypeVideo = "VIDEO"
)

// Call represents a video or audio call
type Call struct {
	ID                    string             `json:"id"`
	AccessToken           string             `json:"accessToken"`
	Role                  string             `json:"role"` // CallRoleEnum
	Caller                *CallParticipant   `json:"caller"`
	Recipients            []*CallParticipant `json:"recipients"`
	AllowVideo            bool               `json:"allowVideo"`
	VideoEnabledByDefault bool               `json:"videoEnabledByDefault"`
}

// CallParticipant represents a person participating in a vidoe or audio call
type CallParticipant struct {
	EntityID       string `json:"-"`
	TwilioIdentity string `json:"twilioIdentity"`
	State          string `json:"state"` // CallStateEnum
}

type CallableIdentity struct {
	Name      string          `json:"name"`
	Endpoints []*CallEndpoint `json:"endpoints"`
	Entity    *Entity         `json:"entity"`
}

type CallEndpoint struct {
	Channel      string `json:"channel"` // CallChannelType enum
	DisplayValue string `json:"displayValue"`
	ValueOrID    string `json:"valueOrID"`
}
