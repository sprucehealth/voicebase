package twilio

import (
	"fmt"
	"time"
)

// IPMessagingGrant grants access to Twilio IP Messaging
type IPMessagingGrant struct {
	ServiceSID        string `json:"service_sid,omitempty"`
	EndpointID        string `json:"endpoint_id,omitempty"`
	DeploymentRoleSID string `json:"deployment_role_sid,omitempty"`
	PushCredentialSID string `json:"push_credential_sid,omitempty"`
}

// Key implements the Grant interface
func (IPMessagingGrant) Key() string {
	return "ip_messaging"
}

// ConversationsGrant grants access to Twilio Conversations
type ConversationsGrant struct {
	ConfigurationProfileSID string `json:"configuration_profile_sid,omitempty"`
}

// Key implements the Grant interface
func (ConversationsGrant) Key() string {
	return "rtc"
}

type Grant interface {
	Key() string
}

type accessTokenPayload struct {
	JTI    string                 `json:"jti"`    // application chosen unique identifier for the token
	ISS    string                 `json:"iss"`    // issuer - the API Key whose secret signs the token.
	Sub    string                 `json:"sub"`    // Sid of the account to which access is scoped.
	Exp    int64                  `json:"exp"`    // timestamp on which the token will expire. Tokens have a maximum age of 24 hours.
	Grants map[string]interface{} `json:"grants"` // granted permissions the token has.
}

type AccessToken struct {
	Identity string
	Grants   []Grant
	TTL      int
}

func (at *AccessToken) ToJWT(accountSID, signingKeySID, secret string) (string, error) {
	headers := map[string]interface{}{
		"typ": "JWT",
		"cty": "twilio-fpa;v=1",
	}

	grants := make(map[string]interface{}, len(at.Grants)+1)
	if at.Identity != "" {
		grants["identity"] = at.Identity
	}
	for _, g := range at.Grants {
		grants[g.Key()] = g
	}

	now := time.Now().Unix()
	ttl := at.TTL
	if ttl <= 0 {
		ttl = 3600
	}
	payload := &accessTokenPayload{
		JTI:    fmt.Sprintf("%s-%d", signingKeySID, now),
		ISS:    signingKeySID,
		Sub:    accountSID,
		Exp:    now + int64(ttl),
		Grants: grants,
	}
	tok, err := jwtEncode(payload, []byte(secret), hs256, headers)
	return string(tok), err
}
