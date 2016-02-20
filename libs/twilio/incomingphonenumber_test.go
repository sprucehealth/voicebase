package twilio

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestIncomingPhoneNumberService_BadAreaCode(t *testing.T) {
	sid := os.Getenv("TEST_TWILIO_SID")
	token := os.Getenv("TEST_TWILIO_TOKEN")
	if sid == "" || token == "" {
		t.Skip("TEST_TWILIO_SID and/or TEST_TWILIO_TOKEN not set")
	}
	c := NewClient(sid, token, nil)
	_, _, err := c.IncomingPhoneNumber.PurchaseLocal(PurchasePhoneNumberParams{
		AreaCode: "555",
	})
	e := err.(*Exception)
	if e.Code != ErrorCodeInvalidAreaCode {
		t.Fatalf("Expected Code %d got %d", ErrorCodeInvalidAreaCode, e.Code)
	}
}

func TestIncomingPhoneNumberService_Validate(t *testing.T) {
	m := PurchasePhoneNumberParams{}
	if err := m.Validate(); err == nil {
		t.Fatalf("Expected error but got none: %s", err.Error())
	}

	m.AreaCode = "415"
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestIncomingPhoneNumberService_PurchaseLocal(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("IncomingPhoneNumbers")

	output := `
	{
    "sid": "PN2a0747eba6abf96b7e3c3ff0b4530f6e",
    "account_sid": "ACdc5f1e11047ebd6fe7a55f120be3a900",
    "friendly_name": "My Company Line",
    "phone_number": "+15105647903",
    "voice_url": "http://demo.twilio.com/docs/voice.xml",
    "voice_method": "POST",
    "voice_fallback_url": null,
    "voice_fallback_method": "POST",
    "status_callback": null,
    "status_callback_method": null,
    "voice_caller_id_lookup": null,
    "voice_application_sid": null,
    "date_created": "Mon, 16 Aug 2010 23:00:23 +0000",
    "date_updated": "Mon, 16 Aug 2010 23:00:23 +0000",
    "sms_url": null,
    "sms_method": "POST",
    "sms_fallback_url": null,
    "sms_fallback_method": "GET",
    "sms_application_sid": "AP9b2e38d8c592488c397fc871a82a74ec",
    "capabilities": {
        "voice": true,
        "sms": true,
        "mms": false
    },
    "api_version": "2010-04-01",
    "uri": "\/2010-04-01\/Accounts\/ACdc5f1e11047ebd6fe7a55f120be3a900\/IncomingPhoneNumbers\/PN2a0747eba6abf96b7e3c3ff0b4530f6e.json"
}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprintf(w, output)
	})

	params := PurchasePhoneNumberParams{
		AreaCode: "415",
	}

	incomingPhoneNumber, _, err := client.IncomingPhoneNumber.PurchaseLocal(params)
	if err != nil {
		t.Fatal(err)
	}

	tm := parseTimestamp("Mon, 16 Aug 2010 23:00:23 +0000")

	want := &IncomingPhoneNumber{
		SID:                  "PN2a0747eba6abf96b7e3c3ff0b4530f6e",
		AccountSID:           "ACdc5f1e11047ebd6fe7a55f120be3a900",
		FriendlyName:         "My Company Line",
		PhoneNumber:          "+15105647903",
		VoiceURL:             "http://demo.twilio.com/docs/voice.xml",
		VoiceMethod:          "POST",
		VoiceFallbackURL:     "",
		VoiceFallbackMethod:  "POST",
		VoiceCallerIDLookup:  false,
		StatusCallback:       "",
		StatusCallbackMethod: "",
		VoiceApplicationSID:  "",
		DateCreated:          tm,
		DateUpdated:          tm,
		SMSURL:               "",
		SMSMethod:            "POST",
		SMSFallbackURL:       "",
		SMSFallbackMethod:    "GET",
		SMSApplicationSID:    "AP9b2e38d8c592488c397fc871a82a74ec",
		Capabilities: map[string]bool{
			"voice": true,
			"sms":   true,
			"mms":   false,
		},
		APIVersion: "2010-04-01",
		URI:        "/2010-04-01/Accounts/ACdc5f1e11047ebd6fe7a55f120be3a900/IncomingPhoneNumbers/PN2a0747eba6abf96b7e3c3ff0b4530f6e.json",
	}

	if !reflect.DeepEqual(incomingPhoneNumber, want) {
		t.Errorf("IncomingPhoneNumber.PurchaseLocal returned %+v, want %+v", incomingPhoneNumber, want)
	}
}
