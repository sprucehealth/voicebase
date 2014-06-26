package mturk

import (
	"github.com/sprucehealth/backend/third_party/launchpad.net/goamz/aws"
)

func Sign(auth aws.Auth, service, method, timestamp string, params map[string]string) {
	sign(auth, service, method, timestamp, params)
}
