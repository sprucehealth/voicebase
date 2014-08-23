package apiservice

import (
	"fmt"
	"net/http"
	"testing"
)

func TestSpruceHeadersFromiOS(t *testing.T) {
	req, err := http.NewRequest("GET", "localhost", nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

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

	sHeaders := ExtractSpruceHeaders(req)

	checkField(appType, sHeaders.AppType, t)
	checkField(appEnvironment, sHeaders.AppEnvironment, t)
	checkField(appVersion, sHeaders.AppVersion.String(), t)
	checkField(appBuild, sHeaders.AppBuild, t)
	checkField(platform, sHeaders.Platform.String(), t)
	checkField(platformVersion, sHeaders.PlatformVersion, t)
	checkField(device, sHeaders.Device, t)
	checkField(deviceModel, sHeaders.DeviceModel, t)
	checkField(resolution, sHeaders.DeviceResolution, t)
	checkField(deviceID, sHeaders.DeviceID, t)

	if sHeaders.ScreenHeight != 1136 {
		t.Fatalf("Expected field value to be %d but was %d instead", 1136, sHeaders.ScreenHeight)
	}

	if sHeaders.ScreenWidth != 640 {
		t.Fatalf("Expected field value to be %d but was %d instead", 640, sHeaders.ScreenWidth)
	}

}

func checkField(expectedFieldValue, currentFieldValue string, t *testing.T) {
	if expectedFieldValue != currentFieldValue {
		t.Fatalf("Expected field value to be %s but was %s instead", expectedFieldValue, currentFieldValue)
	}
}
