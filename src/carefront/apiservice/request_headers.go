package apiservice

import (
	"carefront/common"
	"carefront/libs/golog"
	"net/http"
	"strconv"
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
	AppType        string // (Patient,Doctor,etc)
	AppEnvironment string // (Feature,Dev,Demo,Beta,etc)
	AppVersion     string
	AppBuild       string
	common.Platform
	PlatformVersion  string
	Device           string
	DeviceModel      string
	DeviceID         string
	ScreenWidth      int64
	ScreenHeight     int64
	DeviceResolution string
}

func ExtractSpruceHeaders(r *http.Request) *SpruceHeaders {
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
		sHeaders.AppBuild = sVersionDataComponents[3]
	}

	// S-OS
	sOSDataComponents := strings.Split(r.Header.Get(spruceOSHeader), ";")
	if len(sOSDataComponents) > 0 {
		var err error
		sHeaders.Platform, err = common.GetPlatform(sOSDataComponents[0])
		if err != nil {
			golog.Warningf("Unable to determine platfrom from request header. Ignoring error for now: %s", err)
			sHeaders.Platform = ("")
		}
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

	var err error
	if len(sDeviceComponents) > 2 {
		sHeaders.ScreenWidth, err = strconv.ParseInt(sDeviceComponents[2], 10, 64)
		if err != nil {
			golog.Warningf("Unable to parse screen width header value %s to integer type", sDeviceComponents[2])
		}
	}
	if len(sDeviceComponents) > 3 {
		sHeaders.ScreenHeight, err = strconv.ParseInt(sDeviceComponents[3], 10, 64)
		if err != nil {
			golog.Warningf("Unable to parse screen height header value %s to integer type", sDeviceComponents[3])
		}
	}
	if len(sDeviceComponents) > 4 {
		sHeaders.DeviceResolution = sDeviceComponents[4]
	}

	// S-Device-ID
	sHeaders.DeviceID = r.Header.Get(spruceDeviceIDHeader)

	return &sHeaders
}
