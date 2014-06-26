package services

import (
	"github.com/sprucehealth/backend/libs/maps"
	"testing"
)

func TestGoogleMapsForConvertingZipcodeToCityState(t *testing.T) {
	googleMapsService := maps.NewGoogleMapsService(nil)

	// SHOULD RETURN San Francisco, CA
	cityStateInfo, err := googleMapsService.ConvertZipcodeToCityState("94115")
	if err != nil {
		t.Fatal("Querying google maps unexpected failed: " + err.Error())
	}

	if cityStateInfo.LongCityName != "San Francisco" {
		t.Fatal("Expected city to be San Francicso but returned " + cityStateInfo.LongCityName)
	}

	if cityStateInfo.LongStateName != "California" {
		t.Fatal("Expected State name to be California but returned " + cityStateInfo.LongStateName)
	}

	if cityStateInfo.ShortStateName != "CA" {
		t.Fatal("Expected short state name to be CA but returend " + cityStateInfo.ShortStateName)
	}
}
