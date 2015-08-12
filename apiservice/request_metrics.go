package apiservice

import (
	"os"

	"github.com/sprucehealth/backend/libs/golog"
)

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
		golog.Errorf("Failed to get hostname: %s", err.Error())
	}
}

type AuthEvent string

const (
	AuthEventNoSuchLogin     AuthEvent = "NoSuchLogin"
	AuthEventInvalidPassword AuthEvent = "InvalidPassword"
	AuthEventInvalidToken    AuthEvent = "InvalidToken"
)
