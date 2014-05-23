package apiservice

import (
	"net/http"
	"strings"
)

const (
	spruceVersionHeader  = "S-Version"
	spruceOSHeader       = "S-OS"
	spruceDeviceHeader   = "S-Device"
	spruceDeviceIDHeader = "S-Device-ID"
)

// See here for header definitions:
// https://github.com/SpruceHealth/backend/issues/148
type SpruceHeaders struct {
	AppType          string
	AppEnvironment   string
	AppVersion       string
	Build            string
	Platform         string
	PlatformVersion  string
	Device           string
	DeviceModel      string
	DeviceID         string
	ScreenWidth      string
	ScreenHeight     string
	DeviceResolution string
}

func extractPushConfigDataFromRequest(r *http.Request) *SpruceHeaders {
	sHeaders := SpruceHeaders{}

	// S-Version
	sVersionDataComponents := strings.Split(r.Header.Get(spruceVersionHeader), ";")
	if len(sVersionDataComponents) > 0 {
		sHeaders.AppType = sVersionDataComponents[0]
	}
	if len(sVersionDataComponents) > 1 {
		sHeaders.AppEnvironment = sVersionDataComponents[1]
	}
	if len(sVersionDataComponents) > 2 {
		sHeaders.AppVersion = sVersionDataComponents[2]
	}
	if len(sVersionDataComponents) > 3 {
		sHeaders.Build = sVersionDataComponents[3]
	}

	// S-OS
	sOSDataComponents := strings.Split(r.Header.Get(spruceOSHeader), ";")
	if len(sOSDataComponents) > 0 {
		sHeaders.Platform = sOSDataComponents[0]
	}
	if len(sOSDataComponents) > 1 {
		sHeaders.PlatformVersion = sOSDataComponents[1]
	}

	// S-Device
	sDeviceComponents := strings.Split(r.Header.Get(spruceDeviceHeader), ";")
	if len(sDeviceComponents) > 0 {
		sHeaders.Device = sDeviceComponents[0]
	}
	if len(sDeviceComponents) > 1 {
		sHeaders.DeviceModel = sDeviceComponents[1]
	}
	if len(sDeviceComponents) > 2 {
		sHeaders.ScreenWidth = sDeviceComponents[2]
	}
	if len(sDeviceComponents) > 3 {
		sHeaders.ScreenHeight = sDeviceComponents[3]
	}
	if len(sDeviceComponents) > 4 {
		sHeaders.DeviceResolution = sDeviceComponents[4]
	}

	// S-Device-ID
	sHeaders.DeviceID = r.Header.Get(spruceDeviceIDHeader)

	return &sHeaders
}
