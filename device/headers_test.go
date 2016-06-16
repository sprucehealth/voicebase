package device

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestSpruceHeadersFromiOS(t *testing.T) {
	req, err := http.NewRequest("GET", "localhost", nil)
	test.OK(t, err)

	appType := "Patient"
	appEnvironment := "Feature"
	appVersion := "0.9.0"
	appBuild := "000105"
	platform := "iOS"
	platformVersion := "7.1.1"
	device := "Phone"
	deviceModel := "iPhone6,1"
	screenWidth := "640"
	screenHeight := "1136"
	resolution := "2.0"
	deviceID := "68753A44-4D6F-1226-9C60-0050E4C00067"

	req.Header.Set("S-Version", fmt.Sprintf("%s;%s;%s;%s", appType, appEnvironment, appVersion, appBuild))
	req.Header.Set("S-OS", fmt.Sprintf("%s;%s", platform, platformVersion))
	req.Header.Set("S-Device", fmt.Sprintf("%s;%s;%s;%s;%s", device, deviceModel, screenWidth, screenHeight, resolution))
	req.Header.Set("S-Device-ID", deviceID)

	sHeaders := ExtractSpruceHeaders(nil, req)

	test.Equals(t, appType, sHeaders.AppType)
	test.Equals(t, appEnvironment, sHeaders.AppEnvironment)
	test.Equals(t, appVersion, sHeaders.AppVersion.String())
	test.Equals(t, appBuild, sHeaders.AppBuild)
	test.Equals(t, platform, sHeaders.Platform.String())
	test.Equals(t, platformVersion, sHeaders.PlatformVersion)
	test.Equals(t, device, sHeaders.Device)
	test.Equals(t, deviceModel, sHeaders.DeviceModel)
	test.Equals(t, resolution, sHeaders.DeviceResolution)
	test.Equals(t, deviceID, sHeaders.DeviceID)
	test.Equals(t, 1136, sHeaders.ScreenHeight)
	test.Equals(t, 640, sHeaders.ScreenWidth)
}

func TestHeadersFromWeb(t *testing.T) {
	r, err := http.NewRequest("GET", "localhost", nil)
	test.OK(t, err)

	w := httptest.NewRecorder()
	h := ExtractSpruceHeaders(w, r)
	test.Assert(t, h.DeviceID != "", "DeviceID should not be empty")
	test.Equals(t, fmt.Sprintf("did=%s; Path=/; HttpOnly; Secure", h.DeviceID), w.Header().Get("Set-Cookie"))

	devID := h.DeviceID
	w = httptest.NewRecorder()
	r.Header.Set("Cookie", "did="+devID)
	h = ExtractSpruceHeaders(w, r)
	test.Equals(t, devID, h.DeviceID)
	test.Equals(t, "", w.Header().Get("Set-Cookie"))
}
