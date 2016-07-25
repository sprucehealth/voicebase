package client

import (
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
)

const (
	// DirectoryClientParamKey is where in the root object the directory client is stored
	DirectoryClientParamKey = "directory_client"

	// SettingsClientParamKey is where in the root object the settings client is stored
	SettingsClientParamKey = "settings_client"
)

// Directory returns the directory client mapped into the request params
func Directory(p graphql.ResolveParams) directory.DirectoryClient {
	return p.Info.RootValue.(map[string]interface{})[DirectoryClientParamKey].(directory.DirectoryClient)
}

// Settings returns the settings client mapped into the request params
func Settings(p graphql.ResolveParams) settings.SettingsClient {
	return p.Info.RootValue.(map[string]interface{})[SettingsClientParamKey].(settings.SettingsClient)
}

// InitRoot attaches the various clients into the request structure
func InitRoot(p map[string]interface{},
	directoryClient directory.DirectoryClient,
	settingsClient settings.SettingsClient) map[string]interface{} {
	p[DirectoryClientParamKey] = directoryClient
	p[SettingsClientParamKey] = settingsClient
	return p
}
