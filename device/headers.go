package device

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	spruceVersionHeader  = "S-Version"
	spruceOSHeader       = "S-OS"
	spruceDeviceHeader   = "S-Device"
	spruceDeviceIDHeader = "S-Device-ID"

	deviceIDCookie = "did"
)

// SpruceHeaders are parsed from the HTTP request headers set
// by the client. See here for header definitions:
// https://github.com/sprucehealth/backend/issues/148
type SpruceHeaders struct {
	AppType          string // (Patient,Doctor,etc)
	AppEnvironment   string // (Feature,Dev,Demo,Beta,etc)
	AppVersion       *encoding.Version
	AppBuild         string
	Platform         Platform
	PlatformVersion  string
	Device           string
	DeviceModel      string
	DeviceID         string
	ScreenWidth      int
	ScreenHeight     int
	DeviceResolution string
}

// ExtractSpruceHeaders parses device headers from the request which are set by the apps. For web it
// tries to do its best to store a device ID in the cookies and use whatever information is available.
func ExtractSpruceHeaders(w http.ResponseWriter, r *http.Request) *SpruceHeaders {
	sHeaders := SpruceHeaders{}

	// S-Version
	hdr := r.Header.Get(spruceVersionHeader)
	if hdr == "" {
		hdr = r.Header.Get(strings.ToLower(spruceVersionHeader))
	}
	if hdr != "" {
		parts := strings.Split(hdr, ";")
		if len(parts) > 0 {
			sHeaders.AppType = parts[0]
		}
		if len(parts) > 1 {
			sHeaders.AppEnvironment = parts[1]
		}
		if len(parts) > 2 {
			var err error
			sHeaders.AppVersion, err = encoding.ParseVersion(parts[2])
			if err != nil {
				golog.ContextLogger(r.Context()).Warningf("Unable to parse app version %s: %s", parts[2], err)
			}
		}
		if len(parts) > 3 {
			sHeaders.AppBuild = parts[3]
		}
	}

	// S-OS
	if hdr := r.Header.Get(spruceOSHeader); hdr != "" {
		parts := strings.Split(hdr, ";")
		if len(parts) > 0 {
			var err error
			sHeaders.Platform, err = ParsePlatform(parts[0])
			if err != nil {
				golog.ContextLogger(r.Context()).Warningf("Unable to determine platfrom from request header %s. Ignoring error for now: %s", parts[0], err)
				sHeaders.Platform = ("")
			}
		}
		if len(parts) > 1 {
			sHeaders.PlatformVersion = parts[1]
		}
	}

	// S-Device
	if hdr := r.Header.Get(spruceDeviceHeader); hdr != "" {
		parts := strings.Split(hdr, ";")
		if len(parts) > 0 {
			sHeaders.Device = parts[0]
		}
		if len(parts) > 1 {
			sHeaders.DeviceModel = parts[1]
		}

		var err error
		if len(parts) > 2 {
			sHeaders.ScreenWidth, err = strconv.Atoi(parts[2])
			if err != nil {
				golog.ContextLogger(r.Context()).Warningf("Unable to parse screen width header value %s to integer type", parts[2])
			}
		}
		if len(parts) > 3 {
			sHeaders.ScreenHeight, err = strconv.Atoi(parts[3])
			if err != nil {
				golog.ContextLogger(r.Context()).Warningf("Unable to parse screen height header value %s to integer type", parts[3])
			}
		}
		if len(parts) > 4 {
			sHeaders.DeviceResolution = parts[4]
		}
	}

	// S-Device-ID
	sHeaders.DeviceID = r.Header.Get(spruceDeviceIDHeader)

	if w != nil && sHeaders.DeviceID == "" {
		c, err := r.Cookie(deviceIDCookie)
		if err == http.ErrNoCookie || len(c.Value) < deviceIDLength {
			sHeaders.DeviceID, err = generateDeviceID()
			if err != nil {
				golog.ContextLogger(r.Context()).Errorf("Failed to generate device ID: %s", err)
			} else {
				domain := r.Host
				if i := strings.IndexByte(domain, ':'); i > 0 {
					domain = domain[:i]
				}
				http.SetCookie(w, &http.Cookie{
					Name:     deviceIDCookie,
					Domain:   domain,
					Value:    sHeaders.DeviceID,
					Path:     "/",
					Secure:   !environment.IsDev(),
					HttpOnly: true,
				})
			}
		} else {
			sHeaders.DeviceID = c.Value
		}
	}

	return &sHeaders
}

const deviceIDLength = 20

func generateDeviceID() (string, error) {
	// REMINDER: Update idLength if this function changes
	tokBytes := make([]byte, 16)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", err
	}
	tok := strings.TrimRight(base64.URLEncoding.EncodeToString(tokBytes), "=")
	return tok, nil
}
