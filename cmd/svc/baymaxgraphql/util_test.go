package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/test"
)

func TestInitialsForEntity(t *testing.T) {
	test.Equals(t, "", initialsForEntity(&models.Entity{FirstName: "", LastName: ""}))
	test.Equals(t, "A", initialsForEntity(&models.Entity{FirstName: "Aphex", LastName: ""}))
	test.Equals(t, "Z", initialsForEntity(&models.Entity{FirstName: "", LastName: "Zappa"}))
	test.Equals(t, "AZ", initialsForEntity(&models.Entity{FirstName: "Aphex", LastName: "Zappa"}))
	test.Equals(t, "👀Ž", initialsForEntity(&models.Entity{FirstName: "👀phex", LastName: "Žappa"}))
}

func TestRemoteAddrFromRequest(t *testing.T) {
	tcs := []struct {
		h string
		e string
	}{
		{"", ""},
		{"blah", "blah"},
		{"one,two", "one"},
		{"one,two,three", "one"},
	}
	for _, tc := range tcs {
		req := &http.Request{}
		req.RemoteAddr = tc.h
		req.Header = map[string][]string{"X-Forwarded-For": {tc.h}}
		test.EqualsCase(t, tc.h, tc.e, remoteAddrFromRequest(req, true))
		test.EqualsCase(t, tc.h, tc.h, remoteAddrFromRequest(req, false))
	}
}

func TestDedupeStrings(t *testing.T) {
	tcs := []struct {
		s []string
		e []string
	}{
		{nil, nil},
		{[]string{}, []string{}},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "a"}, []string{"a"}},
		{[]string{"a", "b"}, []string{"a", "b"}},
		{[]string{"a", "a", "b"}, []string{"a", "b"}},
		{[]string{"a", "b", "b"}, []string{"a", "b"}},
		{[]string{"a", "b", "b", "c"}, []string{"a", "b", "c"}},
	}
	for _, tc := range tcs {
		test.EqualsCase(t, fmt.Sprintf("%v", tc.s), tc.e, dedupeStrings(tc.s))
	}
}
