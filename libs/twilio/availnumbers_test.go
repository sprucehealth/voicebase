package twilio

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestAvailablePhoneNumberService_Validate(t *testing.T) {
	m := AvailablePhoneNumbersParams{}

	if err := m.Validate(); err == nil {
		t.Fatalf("Expected error but got none")
	}

	m.SMSEnabled = true
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAvailablePhoneNumberService_ListLocal(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("AvailablePhoneNumbers", "US", "Local")
	output := `
	{
	    "uri": "\/2010-04-01\/Accounts\/ACde6f1e11047ebd6fe7a55f120be3a900\/AvailablePhoneNumbers\/US\/Local.json?AreaCode=510",
	    "available_phone_numbers": [
	        {
	            "friendly_name": "(510) 564-7903",
	            "phone_number": "+15105647903",
	            "lata": "722",
	            "rate_center": "OKLD TRNID",
	            "latitude": "37.780000",
	            "longitude": "-122.380000",
	            "region": "CA",
	            "postal_code": "94703",
	            "iso_country": "US",
	            "capabilities":{
	                "voice":true,
	                "SMS":true,
	                "MMS":false
	            }
	        },
	        {
	            "friendly_name": "(510) 488-4379",
	            "phone_number": "+15104884379",
	            "lata": "722",
	            "rate_center": "OKLD FRTVL",
	            "latitude": "37.780000",
	            "longitude": "-122.380000",
	            "region": "CA",
	            "postal_code": "94602",
	            "iso_country": "US",
	            "capabilities":{
	                "voice":true,
	                "SMS":true,
	                "MMS":false
	            }
	        }
	    ]
	}
	`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, output)
	})

	params := AvailablePhoneNumbersParams{
		SMSEnabled:   true,
		MMSEnabled:   true,
		VoiceEnabled: true,
		AreaCode:     "415",
	}

	availableNumbers, _, err := client.AvailablePhoneNumbers.ListLocal(params)
	if err != nil {
		t.Fatal(err)
	}

	want := []*AvailablePhoneNumber{
		{
			FriendlyName: "(510) 564-7903",
			PhoneNumber:  "+15105647903",
			LATA:         "722",
			RateCenter:   "OKLD TRNID",
			Latitude:     37.78,
			Longitude:    -122.38,
			Region:       "CA",
			PostalCode:   "94703",
			ISOCountry:   "US",
			Capabilities: map[string]bool{
				"voice": true,
				"SMS":   true,
				"MMS":   false,
			},
		},
		{
			FriendlyName: "(510) 488-4379",
			PhoneNumber:  "+15104884379",
			LATA:         "722",
			RateCenter:   "OKLD FRTVL",
			Latitude:     37.78,
			Longitude:    -122.38,
			Region:       "CA",
			PostalCode:   "94602",
			ISOCountry:   "US",
			Capabilities: map[string]bool{
				"voice": true,
				"SMS":   true,
				"MMS":   false,
			},
		},
	}

	if !reflect.DeepEqual(availableNumbers, want) {
		t.Errorf("AvailablePhoneNumbers.ListLocal() returned %+v, want %+v", availableNumbers, want)
	}

}
