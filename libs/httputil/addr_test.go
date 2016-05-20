package httputil

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestRemoteAddrFromRequest(t *testing.T) {
	cases := map[string]struct {
		Request     *http.Request
		BehindProxy bool
		Expected    string
	}{
		"NoProxy": {
			Request: &http.Request{
				RemoteAddr: "remoteAddr",
			},
			Expected: "remoteAddr",
		},
		"ValidProxy": {
			Request: &http.Request{
				RemoteAddr: "notTemoteAddr",
				Header: http.Header{
					"X-Forwarded-For": []string{"remoteAddr"},
				},
			},
			BehindProxy: true,
			Expected:    "remoteAddr",
		},
		"UnknownRemote": {
			Request: &http.Request{
				RemoteAddr: "",
			},
			Expected: UnknownRemoteAddr,
		},
		"UnknownProxy": {
			Request: &http.Request{
				RemoteAddr: "",
				Header: http.Header{
					"X-Forwarded-For": []string{""},
				},
			},
			Expected: UnknownRemoteAddr,
		},
	}

	for cn, c := range cases {
		test.EqualsCase(t, cn, c.Expected, RemoteAddrFromRequest(c.Request, c.BehindProxy))
	}
}
