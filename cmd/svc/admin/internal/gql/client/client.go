package client

import (
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
)

const (
	// DomainsKey is where in the root object the domains structure is stored
	DomainsKey = "domains"
	// DirectoryClientParamKey is where in the root object the directory client is stored
	DirectoryClientParamKey = "directory_client"

	// SettingsClientParamKey is where in the root object the settings client is stored
	SettingsClientParamKey = "settings_client"

	// PaymentsClientParamKey is where in the root object the payments client is stored
	PaymentsClientParamKey = "payments_client"

	// InviteClientParamKey is where in the root object the invite client is stored
	InviteClientParamKey = "invite_client"

	// AuthClientParamKey is where in the root object the auth client is stored
	AuthClientParamKey = "auth_client"
)

// Domains returns the domain sturcture mapped into the request params
func Domains(p graphql.ResolveParams) *Domain {
	return p.Info.RootValue.(map[string]interface{})[DomainsKey].(*Domain)
}

// Directory returns the directory client mapped into the request params
func Directory(p graphql.ResolveParams) directory.DirectoryClient {
	return p.Info.RootValue.(map[string]interface{})[DirectoryClientParamKey].(directory.DirectoryClient)
}

// Settings returns the settings client mapped into the request params
func Settings(p graphql.ResolveParams) settings.SettingsClient {
	return p.Info.RootValue.(map[string]interface{})[SettingsClientParamKey].(settings.SettingsClient)
}

// Payments returns the payments client mapped into the request params
func Payments(p graphql.ResolveParams) payments.PaymentsClient {
	return p.Info.RootValue.(map[string]interface{})[PaymentsClientParamKey].(payments.PaymentsClient)
}

// Invite returns the invite client mapped into the request params
func Invite(p graphql.ResolveParams) invite.InviteClient {
	return p.Info.RootValue.(map[string]interface{})[InviteClientParamKey].(invite.InviteClient)
}

// Auth returns the invite client mapped into the request params
func Auth(p graphql.ResolveParams) auth.AuthClient {
	return p.Info.RootValue.(map[string]interface{})[AuthClientParamKey].(auth.AuthClient)
}

// Domain collects the domains used for url generation
type Domain struct {
	AdminAPI  string
	InviteAPI string
	Web       string
}

// InitRoot attaches the various clients into the request structure
func InitRoot(p map[string]interface{},
	domain *Domain,
	directoryClient directory.DirectoryClient,
	settingsClient settings.SettingsClient,
	paymentsClient payments.PaymentsClient,
	inviteClient invite.InviteClient,
	authClient auth.AuthClient) map[string]interface{} {
	p[DirectoryClientParamKey] = directoryClient
	p[SettingsClientParamKey] = settingsClient
	p[PaymentsClientParamKey] = paymentsClient
	p[InviteClientParamKey] = inviteClient
	p[AuthClientParamKey] = authClient
	p[DomainsKey] = domain
	return p
}
