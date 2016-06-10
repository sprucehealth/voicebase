package twilio

import (
	"reflect"
	"testing"
)

func TestIPMessagingGrant(t *testing.T) {
	at := &AccessToken{
		Grants: []Grant{
			IPMessagingGrant{
				ServiceSID:        "IS123",
				PushCredentialSID: "CR123",
			},
		},
	}
	token, err := at.ToJWT("AC123", "SK123", "secret")
	if err != nil {
		t.Fatal(err)
	}

	payload := new(accessTokenPayload)
	if err := jwtDecode([]byte(token), []byte("secret"), payload); err != nil {
		t.Fatal(err)
	}
	exp := map[string]interface{}{
		"ip_messaging": map[string]interface{}{
			"service_sid":         "IS123",
			"push_credential_sid": "CR123",
		},
	}
	if !reflect.DeepEqual(exp, payload.Grants) {
		t.Fatalf("Expected %v got %v", exp, payload.Grants)
	}
}
