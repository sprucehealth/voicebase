package api

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestClientClock(t *testing.T) {

	// test scanning ability
	var cc clientClock
	err := cc.Scan("session-id-12345:10")
	test.OK(t, err)
	test.Equals(t, "session-id-12345:10", cc.String())
	test.Equals(t, "session-id-12345", cc.sessionID)
	test.Equals(t, uint(10), cc.sessionCounter)

	err = cc.Scan("")
	test.OK(t, err)
	test.Equals(t, "", cc.String())
	test.Equals(t, "", cc.sessionID)
	test.Equals(t, uint(0), cc.sessionCounter)

	cc = clientClock{"session-id-12345", 10}
	test.Equals(t, "session-id-12345:10", cc.String())

	sessionID, sessionCounter, err := splitClientClock(cc.String())
	test.OK(t, err)
	test.Equals(t, "session-id-12345", sessionID)
	test.Equals(t, uint(10), sessionCounter)

	_, _, err = splitClientClock("12415151513513")
	test.Equals(t, true, err != nil)

	// server should accept this write
	incoming := clientClock{"12345", 10}
	accept, err := incoming.lessThan(&clientClock{"12345", 11})
	test.OK(t, err)
	test.Equals(t, true, accept)

	// server should accept an empty incoming clock value
	incoming = clientClock{"12345", 11}
	accept, err = incoming.lessThan(&clientClock{"", 0})
	test.OK(t, err)
	test.Equals(t, true, accept)

	// server should accept a new sessionID
	incoming = clientClock{"12345", 124151}
	accept, err = incoming.lessThan(&clientClock{"135sabab", 1})
	test.OK(t, err)
	test.Equals(t, true, accept)

	// server should not accept the same counter for the same sessionID
	incoming = clientClock{"12345", 12}
	accept, err = incoming.lessThan(&clientClock{"12345", 12})
	test.OK(t, err)
	test.Equals(t, false, accept)

	// server should not accept an lower counter for the same sessionID
	incoming = clientClock{"12345", 12}
	accept, err = incoming.lessThan(&clientClock{"12345", 11})
	test.OK(t, err)
	test.Equals(t, false, accept)
}
