package awsutil

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
)

type awsStatus string

type awsStatusProvider interface {
	Status() (awsStatus, error)
}

func waitForStatus(provider awsStatusProvider, status awsStatus, delay, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		tStatus, err := provider.Status()
		if err != nil {
			return errors.Trace(err)
		}
		if tStatus == status {
			return nil
		}
		time.Sleep(delay)
	}
	return errors.Trace(fmt.Errorf("Status %s was never reached after waiting %v", status, timeout))
}
