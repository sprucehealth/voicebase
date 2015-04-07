package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	errInvalidClientClock = errors.New("invalid client clock")
)

type clientClock struct {
	sessionID      string
	sessionCounter uint
}

func (c *clientClock) Scan(src interface{}) error {
	var err error
	if src == nil {
		c.sessionID = ""
		c.sessionCounter = 0
		return nil
	}

	switch t := src.(type) {
	case []byte:
		c.sessionID, c.sessionCounter, err = splitClientClock(string(t))
	case string:
		c.sessionID, c.sessionCounter, err = splitClientClock(t)
	default:
		return fmt.Errorf("Cannot scan %v into type clientClock", src)
	}

	return err
}

func (c *clientClock) String() string {
	if c.sessionID == "" && c.sessionCounter == 0 {
		return ""
	}

	return c.sessionID + ":" + strconv.Itoa(int(c.sessionCounter))
}

// lessThan returns true if:
// 1. If the incoming client clock value is empty
// 2. If the incoming client sessionID is different from the existing sessionID
// 3. If the incoming and existing sessionIDs match, and the incoming sessionCounter is higher than
// 	  the existing sessionCounter
func (c *clientClock) lessThan(incoming *clientClock) bool {

	if incoming == nil {
		return true
	}

	// if the client does not specify sessionID and counter
	// then we fallback to the last-write-wins model where we accept
	// all values that the client sends us
	if incoming.sessionID == "" && incoming.sessionCounter == 0 {
		return true
	}

	// accept any new sessionID regardless of the counter value
	if c.sessionID != incoming.sessionID {
		return true
	}

	return c.sessionCounter < incoming.sessionCounter
}

// splitClientClock splits the merged clock value into the sessionID
// and the sessionCounter
func splitClientClock(clientClock string) (string, uint, error) {
	if clientClock == "" {
		return "", 0, nil
	}

	index := strings.IndexRune(clientClock, ':')
	if index == -1 {
		return "", 0, errInvalidClientClock
	}

	sessionID := clientClock[:index]
	if (index + 1) >= len(clientClock) {
		return "", 0, errInvalidClientClock
	}

	sessionCounter, err := strconv.Atoi(clientClock[index+1:])
	if err != nil {
		return "", 0, errInvalidClientClock
	}

	return sessionID, uint(sessionCounter), nil
}
