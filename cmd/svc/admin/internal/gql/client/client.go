package client

import (
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
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

	// ExCommsClientParamKey is where in the root object the excomms client is stored
	ExCommsClientParamKey = "excomms_client"

	// ThreadingClientParamKey is where in the root object the threading client is stored
	ThreadingClientParamKey = "threading_client"

	// PatientSyncClientParamKey is where in the root object the patientSync client is stored
	PatientSyncClientParamKey = "patientsync_client"
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

// PaitentSync returns the patientSync client mapped into the request params
func PatientSync(p graphql.ResolveParams) patientsync.PatientSyncClient {
	return p.Info.RootValue.(map[string]interface{})[PatientSyncClientParamKey].(patientsync.PatientSyncClient)
}

// Invite returns the invite client mapped into the request params
func Invite(p graphql.ResolveParams) invite.InviteClient {
	return p.Info.RootValue.(map[string]interface{})[InviteClientParamKey].(invite.InviteClient)
}

// Auth returns the invite client mapped into the request params
func Auth(p graphql.ResolveParams) auth.AuthClient {
	return p.Info.RootValue.(map[string]interface{})[AuthClientParamKey].(auth.AuthClient)
}

// ExComms returns the excomms client mapped into the request params
func ExComms(p graphql.ResolveParams) excomms.ExCommsClient {
	return p.Info.RootValue.(map[string]interface{})[ExCommsClientParamKey].(excomms.ExCommsClient)
}

// Threading returns the threads client mapped into the request params
func Threading(p graphql.ResolveParams) threading.ThreadsClient {
	return p.Info.RootValue.(map[string]interface{})[ThreadingClientParamKey].(threading.ThreadsClient)
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
	patientSyncClient patientsync.PatientSyncClient,
	inviteClient invite.InviteClient,
	authClient auth.AuthClient,
	threadingClient threading.ThreadsClient) map[string]interface{} {
	p[DirectoryClientParamKey] = directoryClient
	p[SettingsClientParamKey] = settingsClient
	p[PaymentsClientParamKey] = paymentsClient
	p[InviteClientParamKey] = inviteClient
	p[AuthClientParamKey] = authClient
	p[ThreadingClientParamKey] = threadingClient
	p[PatientSyncClientParamKey] = patientSyncClient
	p[DomainsKey] = domain
	return p
}
